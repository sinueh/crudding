package custom

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/sinueh-andreata/crudding/internal/generator/common"
	"github.com/sinueh-andreata/crudding/internal/manifest"
	"github.com/sinueh-andreata/crudding/internal/project"
)

// templateFile é um arquivo do template já lido e com o nome trocado.
type templateFile struct {
	output  string
	content []byte
	mode    fs.FileMode
}

// Plan monta o projeto projectName a partir do template em templates, cujo
// project.crud é m: cria as directories e copia os files listados, trocando o
// nome do template pelo do projeto nos caminhos e no conteúdo dos arquivos de
// texto. Se data for fornecido e os arquivos forem templates (.tmpl), eles
// são renderizados com text/template. Tudo é lido e validado aqui, antes de qualquer gravação.
func Plan(templates fs.FS, m manifest.Manifest, projectName string, data ...any) (common.Plan, error) {
	if len(m.Files) == 0 {
		return common.Plan{}, fmt.Errorf("o %s do template não lista files; rode \"crudding init\" na pasta do template", manifest.FileName)
	}
	rename := renamer(m.Name, projectName)

	directories := make([]string, len(m.Directories))
	for index, directory := range m.Directories {
		output := rename.Replace(directory)
		if err := checkPaths(directory, output); err != nil {
			return common.Plan{}, fmt.Errorf("directories: %w", err)
		}
		directories[index] = output
	}

	mappings := m.FileMappings
	if len(mappings) == 0 {
		mappings = make([]manifest.FileMapping, len(m.Files))
		for index, source := range m.Files {
			mappings[index] = manifest.FileMapping{From: source, To: source}
		}
	}

	files := make([]templateFile, len(mappings))
	sources := map[string]string{} // saída -> origem, para achar colisões
	for index, mapping := range mappings {
		source := mapping.From
		output := strings.TrimSuffix(rename.Replace(mapping.To), ".tmpl")
		if err := checkPaths(source, output); err != nil {
			return common.Plan{}, fmt.Errorf("files: %w", err)
		}
		if previous, ok := sources[output]; ok {
			return common.Plan{}, fmt.Errorf("files: %s e %s geram o mesmo arquivo %s", previous, source, output)
		}
		sources[output] = source

		info, err := fs.Lstat(templates, source)
		if errors.Is(err, fs.ErrNotExist) {
			return common.Plan{}, fmt.Errorf("files: %s não existe no template; rode \"crudding init\" na pasta do template para atualizar", source)
		}
		if err != nil {
			return common.Plan{}, fmt.Errorf("files: %w", err)
		}
		if !info.Mode().IsRegular() {
			return common.Plan{}, fmt.Errorf("files: %s não é um arquivo comum", source)
		}
		content, err := fs.ReadFile(templates, source)
		if err != nil {
			return common.Plan{}, fmt.Errorf("files: %w", err)
		}
		if len(data) > 0 && data[0] != nil && strings.HasSuffix(source, ".tmpl") && !isBinary(content) {
			tmpl, err := template.New(source).Parse(string(content))
			if err != nil {
				return common.Plan{}, fmt.Errorf("interpretar template %s: %w", source, err)
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data[0]); err != nil {
				return common.Plan{}, fmt.Errorf("renderizar template %s: %w", source, err)
			}
			content = buf.Bytes()
		}
		if !isBinary(content) {
			content = []byte(rename.Replace(string(content)))
		}
		// Como o git, só o bit de execução é preservado.
		files[index] = templateFile{output: output, content: content, mode: 0644 | info.Mode().Perm()&0111}
	}

	generated := m
	generated.Name = projectName
	generated.Template = m.Name
	generated.Directories = directories
	generated.Files = make([]string, len(files))
	for index, file := range files {
		generated.Files[index] = file.output
	}
	// Caminhos do campo endpoint podem conter o nome do projeto (ex.: src/<pacote>).
	generated.Endpoint.Files = stripTmplAll(renameAll(rename, m.Endpoint.Files))
	generated.Endpoint.Register = stripTmplAll(renameAll(rename, m.Endpoint.Register))

	return common.Plan{
		Manifest: &generated,
		Prepare: func(root string) error {
			for _, directory := range directories {
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0755); err != nil {
					return fmt.Errorf("criar diretório %s: %w", directory, err)
				}
			}
			for _, file := range files {
				target := filepath.Join(root, filepath.FromSlash(file.output))
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return fmt.Errorf("criar diretório para %s: %w", file.output, err)
				}
				if err := os.WriteFile(target, file.content, file.mode); err != nil {
					return fmt.Errorf("gravar arquivo %s: %w", file.output, err)
				}
			}
			return nil
		},
	}, nil
}

func renameAll(rename nameReplacer, paths []string) []string {
	renamed := make([]string, len(paths))
	for index, rel := range paths {
		renamed[index] = rename.Replace(rel)
	}
	return renamed
}

func stripTmplAll(paths []string) []string {
	result := make([]string, len(paths))
	for index, p := range paths {
		result[index] = strings.TrimSuffix(p, ".tmpl")
	}
	return result
}

// Unlisted devolve os arquivos do template que não estão em files nem casam
// com o ignore, provavelmente criados depois do último "crudding init".
func Unlisted(templates fs.FS, m manifest.Manifest) ([]string, error) {
	_, files, err := common.ListStructure(templates, m.Ignore)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(files, func(file string) bool { return slices.Contains(m.Files, file) }), nil
}

// renamer troca o nome do template pelo do projeto. A forma exata (ex.: "Meu
// Template") vira o nome digitado; slug (meu-template) e snake_case
// (meu_template) viram as formas equivalentes do nome novo. Num nome de uma
// palavra só, slug e snake coincidem e ganha o snake_case, que continua válido
// como identificador (pacote Python, módulo Go).
func renamer(from, to string) nameReplacer {
	fromSlug, toSlug := project.Slug(from), project.Slug(to)
	fromSnake, toSnake := strings.ReplaceAll(fromSlug, "-", "_"), strings.ReplaceAll(toSlug, "-", "_")
	return newNameReplacer([][2]string{{fromSnake, toSnake}, {fromSlug, toSlug}, {from, to}})
}

// nameReplacer troca várias formas de um nome de uma vez.
type nameReplacer struct {
	olds     []string
	replacer *strings.Replacer
}

// newNameReplacer monta o replacer a partir de pares {antigo, novo}. Pares
// vazios, sem mudança ou com um antigo repetido são ignorados (vale o
// primeiro), e os mais longos vêm primeiro, porque o strings.Replacer decide
// empates pela ordem.
func newNameReplacer(pairs [][2]string) nameReplacer {
	var kept [][2]string
	for _, pair := range pairs {
		if pair[0] != "" && pair[0] != pair[1] && !slices.ContainsFunc(kept, func(k [2]string) bool { return k[0] == pair[0] }) {
			kept = append(kept, pair)
		}
	}
	slices.SortStableFunc(kept, func(a, b [2]string) int { return cmp.Compare(len(b[0]), len(a[0])) })

	var r nameReplacer
	var args []string
	for _, pair := range kept {
		r.olds = append(r.olds, pair[0])
		args = append(args, pair[0], pair[1])
	}
	r.replacer = strings.NewReplacer(args...)
	return r
}

func (r nameReplacer) Replace(s string) string { return r.replacer.Replace(s) }

// mentions diz se s contém alguma das formas antigas do nome.
func (r nameReplacer) mentions(s string) bool {
	return slices.ContainsFunc(r.olds, func(old string) bool { return strings.Contains(s, old) })
}

// isBinary usa a mesma heurística do git: um byte NUL no começo do arquivo.
func isBinary(content []byte) bool {
	return bytes.IndexByte(content[:min(len(content), 8000)], 0) >= 0
}

// checkPaths valida o caminho no template e o caminho gerado (ambos com "/"):
// precisam ser limpos, relativos, dentro do projeto e diferentes do
// project.crud.
func checkPaths(source, output string) error {
	for _, rel := range []string{source, output} {
		if !fs.ValidPath(rel) || rel == "." || !filepath.IsLocal(filepath.FromSlash(rel)) {
			return fmt.Errorf("caminho %q inválido: use um caminho relativo dentro do projeto, sem \"..\"", rel)
		}
		if rel == manifest.FileName {
			return fmt.Errorf("%s é gravado pelo crudding e não pode ser listado", manifest.FileName)
		}
	}
	return nil
}
