package custom

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"crudding/internal/generator/common"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

// reservedNames são palavras que não podem virar nome de endpoint, porque o
// nome vira pacote ou módulo na linguagem.
var reservedNames = map[string][]string{
	"python": {
		"and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif", "else",
		"except", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda",
		"nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield",
	},
	"go": {
		"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for",
		"func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var",
	},
}

// CheckEndpoint valida o campo endpoint do manifesto sem olhar o disco, para a
// CLI falhar antes de pedir o nome do endpoint.
func CheckEndpoint(m manifest.Manifest) error {
	spec := m.Endpoint
	if err := project.ValidateEndpointName(spec.Name); err != nil {
		return fmt.Errorf("campo endpoint do %s: name: %w", manifest.FileName, err)
	}
	if len(spec.Files) == 0 {
		return fmt.Errorf("campo endpoint do %s: liste em files os arquivos do endpoint de exemplo %s", manifest.FileName, spec.Name)
	}
	return nil
}

// EndpointPatch monta o Patch de "crudding add endpoint name" a partir do
// campo endpoint de m: copia os arquivos do exemplo trocando o nome nos
// caminhos e no conteúdo e, em cada arquivo de register, duplica as linhas que
// mencionam o exemplo.
func EndpointPatch(root string, m manifest.Manifest, name string) (common.Patch, error) {
	recipe, err := newEndpointRecipe(m, name)
	if err != nil {
		return common.Patch{}, err
	}
	files, err := recipe.outputs()
	if err != nil {
		return common.Patch{}, err
	}

	projectFS := os.DirFS(root)
	var patch common.Patch
	for _, file := range files {
		content, err := readRegular(projectFS, file.source)
		if err != nil {
			return common.Patch{}, fmt.Errorf("endpoint.files: %w", err)
		}
		if !isBinary(content) {
			content = []byte(recipe.rename.Replace(string(content)))
		}
		patch.Raw = append(patch.Raw, common.RawFile{OutputPath: file.output, Content: content})
	}
	for _, target := range recipe.spec.Register {
		if err := checkPaths(target, target); err != nil {
			return common.Patch{}, fmt.Errorf("endpoint.register: %w", err)
		}
		patch.Rewrites = append(patch.Rewrites, common.Rewrite{
			Path: target,
			Edit: func(content []byte) ([]byte, error) { return recipe.duplicateLines(target, content) },
		})
	}
	return patch, nil
}

// EndpointRemovalPlan monta o RemovalPlan de "crudding remove endpoint name":
// os arquivos copiados do exemplo, as pastas criadas para eles (removidas se
// ficarem vazias) e as linhas duplicadas nos arquivos de register.
func EndpointRemovalPlan(root string, m manifest.Manifest, name string) (common.RemovalPlan, error) {
	recipe, err := newEndpointRecipe(m, name)
	if err != nil {
		return common.RemovalPlan{}, err
	}
	files, err := recipe.outputs()
	if err != nil {
		return common.RemovalPlan{}, err
	}

	var plan common.RemovalPlan
	for _, file := range files {
		plan.Files = append(plan.Files, file.output)
		// Só as pastas cujo caminho mudou com o nome (ex.: internal/invoice).
		for source, output := path.Dir(file.source), path.Dir(file.output); output != source; source, output = path.Dir(source), path.Dir(output) {
			if !slices.Contains(plan.Directories, output) {
				plan.Directories = append(plan.Directories, output)
			}
		}
	}
	for _, target := range recipe.spec.Register {
		if err := checkPaths(target, target); err != nil {
			return common.RemovalPlan{}, fmt.Errorf("endpoint.register: %w", err)
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil {
			return common.RemovalPlan{}, fmt.Errorf("ler %s: %w", target, err)
		}
		var lines []string
		for _, line := range strings.Split(string(content), "\n") {
			if recipe.sampleLine(line) {
				lines = append(lines, recipe.rename.Replace(line))
			}
		}
		if len(lines) > 0 {
			plan.Removals = append(plan.Removals, common.LineRemoval{Path: target, Lines: lines})
		}
	}
	return plan, nil
}

// endpointRecipe é o campo endpoint validado, com a troca exemplo -> nome novo.
type endpointRecipe struct {
	spec   manifest.EndpointSpec
	rename nameReplacer
	// others são formas de outros endpoints que contêm o nome do exemplo (ex.:
	// user_profile quando o exemplo é user): linhas com elas não são do exemplo.
	others []string
}

type endpointFile struct{ source, output string }

func newEndpointRecipe(m manifest.Manifest, name string) (endpointRecipe, error) {
	if err := CheckEndpoint(m); err != nil {
		return endpointRecipe{}, err
	}
	if err := project.ValidateEndpointName(name); err != nil {
		return endpointRecipe{}, err
	}
	if slices.Contains(reservedNames[m.Language], name) {
		return endpointRecipe{}, fmt.Errorf("%q é uma palavra reservada em %s", name, m.Language)
	}
	spec := m.Endpoint
	if name == spec.Name {
		return endpointRecipe{}, fmt.Errorf("%q é o endpoint de exemplo do campo endpoint do %s; use outro nome", name, manifest.FileName)
	}

	from, to := endpointForms(spec.Name), endpointForms(name)
	pairs := make([][2]string, len(from))
	for index := range from {
		pairs[index] = [2]string{from[index], to[index]}
	}
	// Num exemplo de uma palavra só, snake_case e camelCase coincidem (user).
	// Seguido de maiúscula (userRepo, userID), é camelCase: vira orderItemRepo.
	if camel := from[len(from)-1]; camel == from[0] {
		for letter := 'A'; letter <= 'Z'; letter++ {
			pairs = append(pairs, [2]string{camel + string(letter), to[len(to)-1] + string(letter)})
		}
	}
	recipe := endpointRecipe{spec: spec, rename: newNameReplacer(pairs)}
	for _, other := range m.Endpoints {
		if other == spec.Name {
			continue
		}
		for _, form := range endpointForms(other) {
			if recipe.rename.mentions(form) && !slices.Contains(recipe.others, form) {
				recipe.others = append(recipe.others, form)
			}
		}
	}
	return recipe, nil
}

// outputs liga cada arquivo do exemplo ao caminho do arquivo novo.
func (r endpointRecipe) outputs() ([]endpointFile, error) {
	files := make([]endpointFile, len(r.spec.Files))
	for index, source := range r.spec.Files {
		output := r.rename.Replace(source)
		if err := checkPaths(source, output); err != nil {
			return nil, fmt.Errorf("endpoint.files: %w", err)
		}
		if output == source {
			return nil, fmt.Errorf("endpoint.files: %s não contém o nome do exemplo %q, então o arquivo novo teria o mesmo caminho", source, r.spec.Name)
		}
		files[index] = endpointFile{source: source, output: output}
	}
	return files, nil
}

// sampleLine diz se a linha é do endpoint de exemplo: menciona o exemplo e não
// menciona outro endpoint cujo nome contém o dele.
func (r endpointRecipe) sampleLine(line string) bool {
	return r.rename.mentions(line) && !slices.ContainsFunc(r.others, func(form string) bool { return strings.Contains(line, form) })
}

// duplicateLines repete cada bloco de linhas consecutivas do exemplo logo
// depois dele, com o nome novo.
func (r endpointRecipe) duplicateLines(target string, content []byte) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	result := make([]string, 0, len(lines))
	found := false
	for start := 0; start < len(lines); {
		end := start
		for end < len(lines) && r.sampleLine(lines[end]) {
			end++
		}
		if end == start {
			result = append(result, lines[start])
			start++
			continue
		}
		found = true
		result = append(result, lines[start:end]...)
		for _, line := range lines[start:end] {
			result = append(result, r.rename.Replace(line))
		}
		start = end
	}
	if !found {
		return nil, fmt.Errorf("endpoint.register: %s não tem nenhuma linha que mencione o endpoint de exemplo %q", target, r.spec.Name)
	}
	return []byte(strings.Join(result, "\n")), nil
}

// endpointForms devolve as formas de um nome snake_case usadas no código:
// snake_case, kebab-case, PascalCase e camelCase.
func endpointForms(name string) []string {
	var pascal strings.Builder
	for _, part := range strings.Split(name, "_") {
		if part != "" {
			pascal.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	forms := []string{name, strings.ReplaceAll(name, "_", "-"), pascal.String()}
	if p := pascal.String(); p != "" {
		forms = append(forms, strings.ToLower(p[:1])+p[1:])
	}
	return forms
}

func readRegular(fsys fs.FS, name string) ([]byte, error) {
	info, err := fs.Lstat(fsys, name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s não é um arquivo comum", name)
	}
	return fs.ReadFile(fsys, name)
}
