package custom

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
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

// CheckEndpoint valida o campo template_endpoint do manifesto sem olhar o disco, para a
// CLI falhar antes de pedir o nome do endpoint.
func CheckEndpoint(m manifest.Manifest) error {
	spec := m.TemplateEndpoint
	if err := project.ValidateEndpointName(spec.Name); err != nil {
		return fmt.Errorf("campo template_endpoint do %s: name: %w", manifest.FileName, err)
	}
	if len(spec.Files) == 0 {
		return fmt.Errorf("campo template_endpoint do %s: liste em files os arquivos do endpoint de exemplo %s", manifest.FileName, spec.Name)
	}
	return nil
}

// EndpointPatch monta o Patch de "crudding add endpoint name" a partir do
// campo template_endpoint de m: copia os arquivos do exemplo trocando o nome nos
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
			Edit: func(content []byte) ([]byte, error) { return recipe.registerLines(root, m, target, name, content) },
		})
	}
	return patch, nil
}

// BackupRegister salva uma cópia limpa dos arquivos de register em
// .crudding/<template_endpoint.name>/ antes de remover o endpoint de exemplo da
// estrutura do projeto, para que futuros add/remove endpoint continuem
// funcionando.
func BackupRegister(root string, m *manifest.Manifest) error {
	recipe, err := buildEndpointRecipe(*m, m.TemplateEndpoint.Name, true)
	if err != nil {
		return err
	}
	files, err := recipe.outputs()
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.output == file.source || strings.HasPrefix(file.output, ".crudding/") {
			return fmt.Errorf("%q é o endpoint de exemplo do campo template_endpoint do %s; os arquivos em %s não podem ser removidos", m.TemplateEndpoint.Name, manifest.FileName, file.source)
		}
	}
	var created []string
	for _, target := range recipe.spec.Register {
		if err := checkPaths(target, target); err != nil {
			return fmt.Errorf("endpoint.register: %w", err)
		}
		backupRel := path.Join(".crudding", recipe.spec.Name, target)
		backupPath := filepath.Join(root, filepath.FromSlash(backupRel))
		if _, err := os.Stat(backupPath); err == nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil {
			return fmt.Errorf("ler %s: %w", target, err)
		}
		cleaned := stripOtherEndpoints(*m, recipe, content)
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			return fmt.Errorf("criar diretório para %s: %w", backupRel, err)
		}
		if err := os.WriteFile(backupPath, cleaned, 0o644); err != nil {
			return fmt.Errorf("gravar %s: %w", backupRel, err)
		}
		created = append(created, backupRel)
	}
	if len(created) > 0 {
		return common.RecordCreated(m, created)
	}
	return nil
}

func stripOtherEndpoints(m manifest.Manifest, sampleRecipe endpointRecipe, content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	var sampleLines []string
	for _, line := range lines {
		if sampleRecipe.sampleLine(line) {
			sampleLines = append(sampleLines, line)
		}
	}
	if len(sampleLines) == 0 {
		return content
	}
	otherSet := map[string]bool{}
	for _, other := range m.Endpoints {
		if other == sampleRecipe.spec.Name {
			continue
		}
		otherRecipe, err := buildEndpointRecipe(m, other, true)
		if err != nil {
			continue
		}
		for _, line := range sampleLines {
			if trimmed := strings.TrimSpace(otherRecipe.rename.Replace(line)); trimmed != "" {
				otherSet[trimmed] = true
			}
		}
	}
	if len(otherSet) == 0 {
		return content
	}
	var kept []string
	for _, line := range lines {
		if otherSet[strings.TrimSpace(line)] {
			continue
		}
		kept = append(kept, line)
	}
	return []byte(strings.Join(kept, "\n"))
}

// EndpointRemovalPlan monta o RemovalPlan de "crudding remove endpoint name":
// os arquivos copiados do exemplo, as pastas criadas para eles (removidas se
// ficarem vazias) e as linhas duplicadas nos arquivos de register.
func EndpointRemovalPlan(root string, m manifest.Manifest, name string) (common.RemovalPlan, error) {
	recipe, err := buildEndpointRecipe(m, name, true)
	if err != nil {
		return common.RemovalPlan{}, err
	}
	files, err := recipe.outputs()
	if err != nil {
		return common.RemovalPlan{}, err
	}

	var plan common.RemovalPlan
	for _, file := range files {
		if file.output == file.source || strings.HasPrefix(file.output, ".crudding/") {
			return common.RemovalPlan{}, fmt.Errorf("%q é o endpoint de exemplo do campo template_endpoint do %s; os arquivos em %s não podem ser removidos", name, manifest.FileName, file.source)
		}
		plan.Files = append(plan.Files, file.output)
		// Só as pastas cujo caminho contém o nome do endpoint (ex.: internal/invoice ou internal/template).
		for source, output := path.Dir(file.target), path.Dir(file.output); source != "." && recipe.rename.mentions(source); source, output = path.Dir(source), path.Dir(output) {
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
		if len(lines) == 0 {
			backupPath := filepath.Join(root, filepath.FromSlash(path.Join(".crudding", recipe.spec.Name, target)))
			if refContent, err := os.ReadFile(backupPath); err == nil {
				for _, line := range strings.Split(string(refContent), "\n") {
					if recipe.sampleLine(line) {
						lines = append(lines, recipe.rename.Replace(line))
					}
				}
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

type endpointFile struct{ source, target, output string }

func newEndpointRecipe(m manifest.Manifest, name string) (endpointRecipe, error) {
	return buildEndpointRecipe(m, name, false)
}

func buildEndpointRecipe(m manifest.Manifest, name string, allowSample bool) (endpointRecipe, error) {
	if err := CheckEndpoint(m); err != nil {
		return endpointRecipe{}, err
	}
	if err := project.ValidateEndpointName(name); err != nil {
		return endpointRecipe{}, err
	}
	if slices.Contains(reservedNames[m.Language], name) {
		return endpointRecipe{}, fmt.Errorf("%q é uma palavra reservada em %s", name, m.Language)
	}
	spec := m.TemplateEndpoint
	if !allowSample && name == spec.Name {
		return endpointRecipe{}, fmt.Errorf("%q é o endpoint de exemplo do campo template_endpoint do %s; use outro nome", name, manifest.FileName)
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
	prefix := ".crudding/" + r.spec.Name + "/"
	for index, source := range r.spec.Files {
		var target string
		switch {
		case strings.HasPrefix(source, prefix):
			target = strings.TrimPrefix(source, prefix)
			if !r.rename.mentions(target) {
				target = path.Join(path.Dir(target), r.spec.Name, path.Base(target))
			}
		case strings.HasPrefix(source, ".crudding/"):
			target = strings.TrimPrefix(source, ".crudding/")
		default:
			target = source
		}
		output := r.rename.Replace(target)
		if err := checkPaths(source, output); err != nil {
			return nil, fmt.Errorf("endpoint.files: %w", err)
		}
		if !r.rename.mentions(target) {
			return nil, fmt.Errorf("endpoint.files: %s não contém o nome do exemplo %q, então o arquivo novo teria o mesmo caminho", source, r.spec.Name)
		}
		files[index] = endpointFile{source: source, target: target, output: output}
	}
	return files, nil
}

// sampleLine diz se a linha é do endpoint de exemplo: menciona o exemplo e não
// menciona outro endpoint cujo nome contém o dele.
func (r endpointRecipe) sampleLine(line string) bool {
	return r.rename.mentions(line) && !slices.ContainsFunc(r.others, func(form string) bool { return strings.Contains(line, form) })
}

// registerLines registra o novo endpoint em target: usa as linhas do exemplo
// se ainda estiverem em content; caso o exemplo tenha sido removido da
// estrutura do projeto, usa outro endpoint ativo ou o backup em .crudding/.
func (r endpointRecipe) registerLines(root string, m manifest.Manifest, target, name string, content []byte) ([]byte, error) {
	if slices.ContainsFunc(strings.Split(string(content), "\n"), r.sampleLine) {
		return r.duplicateLines(target, content)
	}
	for i := len(m.Endpoints) - 1; i >= 0; i-- {
		anchor := m.Endpoints[i]
		if anchor == r.spec.Name {
			continue
		}
		anchorManifest := m
		anchorManifest.TemplateEndpoint.Name = anchor
		anchorRecipe, err := buildEndpointRecipe(anchorManifest, name, false)
		if err == nil && slices.ContainsFunc(strings.Split(string(content), "\n"), anchorRecipe.sampleLine) {
			return anchorRecipe.duplicateLines(target, content)
		}
	}
	backupPath := filepath.Join(root, filepath.FromSlash(path.Join(".crudding", r.spec.Name, target)))
	refContent, err := os.ReadFile(backupPath)
	if err != nil {
		return r.duplicateLines(target, content)
	}
	return r.insertFromReference(target, refContent, content)
}

func (r endpointRecipe) insertFromReference(target string, refContent, content []byte) ([]byte, error) {
	refLines := strings.Split(string(refContent), "\n")
	curLines := strings.Split(string(content), "\n")
	var result []string
	curIdx := 0
	found := false
	for refIdx := 0; refIdx < len(refLines); refIdx++ {
		if r.sampleLine(refLines[refIdx]) {
			found = true
			result = append(result, r.rename.Replace(refLines[refIdx]))
			continue
		}
		matchIdx := -1
		for i := curIdx; i < len(curLines); i++ {
			if curLines[i] == refLines[refIdx] {
				matchIdx = i
				break
			}
		}
		if matchIdx >= 0 {
			result = append(result, curLines[curIdx:matchIdx+1]...)
			curIdx = matchIdx + 1
		}
	}
	result = append(result, curLines[curIdx:]...)
	if !found {
		return nil, fmt.Errorf("endpoint.register: %s não tem nenhuma linha que mencione o endpoint de exemplo %q", target, r.spec.Name)
	}
	return []byte(strings.Join(result, "\n")), nil
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

// EndpointRenamer devolve a troca de nomes que o "add endpoint name" aplica
// aos arquivos copiados do exemplo (snake, kebab, Pascal e camelCase), para
// quem gera o conteúdo do endpoint por outro caminho (ex.: campos tipados)
// produzir os mesmos identificadores das linhas de register.
func EndpointRenamer(m manifest.Manifest, name string) (func(string) string, error) {
	recipe, err := newEndpointRecipe(m, name)
	if err != nil {
		return nil, err
	}
	return recipe.rename.Replace, nil
}
