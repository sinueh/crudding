package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
)

// SupportedDatabases lista os bancos de dados suportados pela CLI.
var SupportedDatabases = []string{"postgres", "mysql", "mongo"}

// ValidateDatabase verifica se o banco informado é suportado.
func ValidateDatabase(db string) error {
	if !slices.Contains(SupportedDatabases, db) {
		return fmt.Errorf("banco de dados não suportado: %q (opções: %s)", db, strings.Join(SupportedDatabases, ", "))
	}
	return nil
}

// databaseRecipe é um database.crud.tmpl: descreve como o "add db" altera o
// projeto. Ele é renderizado com os dados do projeto (Database, Runtime,
// ORM...) e os caminhos de from e base são relativos à pasta templates/.
// templates/databases/database.crud.tmpl vale para todo projeto e
// templates/langs/<lang>/database.crud.tmpl, para a linguagem.
type databaseRecipe struct {
	// Files são arquivos novos.
	Files []manifest.FileMapping `json:"files"`
	// Inserts colocam as linhas de from logo antes do marker.
	Inserts []databaseInsert `json:"inserts"`
	// Migrations é a pasta copiada para migrations/ com o próximo número.
	Migrations string `json:"migrations"`
	// Append acrescenta ao fim de into as linhas de from que ele ainda não
	// tem (ex.: go.mod, requirements.txt).
	Append []databaseEdit `json:"append"`
	// Merge mescla o objeto JSON de from no JSON de into (ex.: package.json).
	Merge []databaseEdit `json:"merge"`
}

// databaseInsert vai para into ou, sem into, para o único arquivo do projeto
// que tem a linha marker. Se into não existe e base foi informado, into é
// criado a partir de base. Com unique, as linhas que o arquivo já tem são
// puladas (ex.: o import "fmt" de um segundo banco).
type databaseInsert struct {
	Into   string `json:"into"`
	Base   string `json:"base"`
	Marker string `json:"marker"`
	From   string `json:"from"`
	Unique bool   `json:"unique"`
}

// databaseEdit altera into com o conteúdo de from; into ausente é ignorado.
type databaseEdit struct {
	Into string `json:"into"`
	From string `json:"from"`
}

// AddDatabase adiciona a infraestrutura do banco de dados escolhido ao projeto.
func AddDatabase(root string, m manifest.Manifest, db string, templates fs.FS) (common.Result, error) {
	if err := ValidateDatabase(db); err != nil {
		return common.Result{}, err
	}
	if m.HasDatabase(db) {
		return common.Result{}, fmt.Errorf("o projeto já possui o banco de dados %q configurado", db)
	}

	templatesRoot, err := fs.Sub(templates, "templates")
	if err != nil {
		return common.Result{}, fmt.Errorf("acessar templates: %w", err)
	}
	config := m.Config()
	config.Database = db
	patch, err := buildDatabasePatch(root, m, templatesRoot, common.TemplateData(config))
	if err != nil {
		return common.Result{}, err
	}

	result, err := common.Apply(root, templatesRoot, patch)
	if err != nil {
		return common.Result{}, err
	}

	updated := m
	if onDisk, readErr := manifest.Read(filepath.Join(root, manifest.FileName)); readErr == nil {
		updated = onDisk
	}
	updated.AddDatabase(db)
	if err := common.RecordCreated(&updated, result.Created); err != nil {
		return common.Result{}, err
	}
	if err := manifest.Write(root, updated); err != nil {
		return common.Result{}, err
	}

	return result, nil
}

func buildDatabasePatch(root string, m manifest.Manifest, templates fs.FS, data map[string]any) (common.Patch, error) {
	languageRecipe := path.Join("langs", m.Language, "database.crud.tmpl")
	if _, err := fs.Stat(templates, languageRecipe); m.Language == "" || err != nil {
		return common.Patch{}, fmt.Errorf("add db não é suportado para a linguagem %q do %s", m.Language, manifest.FileName)
	}

	patch := common.Patch{Data: data}
	for _, recipePath := range []string{"databases/database.crud.tmpl", languageRecipe} {
		recipe, err := readDatabaseRecipe(templates, recipePath, data)
		if err != nil {
			return common.Patch{}, err
		}
		if err := recipe.addTo(&patch, root, m.Files, templates, data); err != nil {
			return common.Patch{}, fmt.Errorf("%s: %w", recipePath, err)
		}
	}
	return patch, nil
}

func readDatabaseRecipe(templates fs.FS, recipePath string, data any) (databaseRecipe, error) {
	content, err := common.Render(templates, recipePath, data)
	if err != nil {
		return databaseRecipe{}, err
	}
	var recipe databaseRecipe
	if err := json.Unmarshal(content, &recipe); err != nil {
		return databaseRecipe{}, fmt.Errorf("decodificar %s: %w", recipePath, err)
	}
	return recipe, nil
}

func (r databaseRecipe) addTo(patch *common.Patch, root string, projectFiles []string, templates fs.FS, data map[string]any) error {
	creates := func(rel string) bool {
		return slices.ContainsFunc(patch.Files, func(file common.File) bool { return file.OutputPath == rel })
	}

	for _, file := range r.Files {
		patch.Files = append(patch.Files, common.File{TemplatePath: file.From, OutputPath: file.To})
	}

	for _, insert := range r.Inserts {
		into := insert.Into
		if into == "" {
			var err error
			if into, err = fileWithMarker(root, projectFiles, insert.Marker); err != nil {
				return err
			}
		} else if insert.Base != "" && !exists(root, into) && !creates(into) {
			patch.Files = append(patch.Files, common.File{TemplatePath: insert.Base, OutputPath: into})
		}
		lines, err := renderLines(templates, insert.From, data)
		if err != nil {
			return err
		}
		if insert.Unique {
			lines = withoutExisting(root, into, lines)
		}
		if len(lines) > 0 {
			patch.Inserts = append(patch.Inserts, common.Insert{Path: into, Marker: insert.Marker, Lines: lines})
		}
	}

	if r.Migrations != "" {
		database, _ := data["Database"].(string)
		files, err := migrationFiles(root, templates, r.Migrations, database)
		if err != nil {
			return err
		}
		patch.Files = append(patch.Files, files...)
	}

	for _, edit := range r.Append {
		lines, err := renderLines(templates, edit.From, data)
		if err != nil {
			return err
		}
		if err := addRewrite(patch, root, edit.Into, func(content []byte) ([]byte, error) {
			return appendMissingLines(content, lines), nil
		}); err != nil {
			return err
		}
	}

	for _, edit := range r.Merge {
		content, err := common.Render(templates, edit.From, data)
		if err != nil {
			return err
		}
		var values map[string]any
		if err := json.Unmarshal(content, &values); err != nil {
			return fmt.Errorf("decodificar %s: %w", edit.From, err)
		}
		if err := addRewrite(patch, root, edit.Into, func(content []byte) ([]byte, error) {
			return mergeJSON(content, values)
		}); err != nil {
			return err
		}
	}
	return nil
}

// addRewrite só registra a reescrita se into existe e muda com ela, para o
// arquivo não aparecer como alterado à toa.
func addRewrite(patch *common.Patch, root, into string, edit func([]byte) ([]byte, error)) error {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(into)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ler %s: %w", into, err)
	}
	changed, err := edit(content)
	if err != nil {
		return fmt.Errorf("%s: %w", into, err)
	}
	if !bytes.Equal(changed, content) {
		patch.Rewrites = append(patch.Rewrites, common.Rewrite{Path: into, Edit: edit})
	}
	return nil
}

// renderLines renderiza um snippet e devolve suas linhas, sem a quebra final.
func renderLines(templates fs.FS, snippet string, data any) ([]string, error) {
	content, err := common.Render(templates, snippet, data)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSuffix(string(content), "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

// fileWithMarker devolve o único arquivo de files que tem a linha marker.
func fileWithMarker(root string, files []string, marker string) (string, error) {
	var found []string
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			continue
		}
		if slices.ContainsFunc(strings.Split(string(content), "\n"), func(line string) bool { return strings.TrimSpace(line) == marker }) {
			found = append(found, file)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("nenhum arquivo listado em files do %s tem a linha %q", manifest.FileName, marker)
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("a linha %q aparece em mais de um arquivo: %s", marker, strings.Join(found, ", "))
}

// withoutExisting tira de lines as linhas não vazias que rel já tem.
func withoutExisting(root, rel string, lines []string) []string {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return lines
	}
	existing := strings.Split(string(content), "\n")
	return slices.DeleteFunc(slices.Clone(lines), func(line string) bool {
		return strings.TrimSpace(line) != "" && slices.ContainsFunc(existing, func(current string) bool {
			return strings.TrimSpace(current) == strings.TrimSpace(line)
		})
	})
}

func exists(root, rel string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

// migrationFiles copia os arquivos de dir para migrations/ com o próximo
// número livre. A partir da segunda migração o nome leva o banco (ex.:
// 000002_mongo_init.js), para não confundir com a de outro banco.
func migrationFiles(root string, templates fs.FS, dir, db string) ([]common.File, error) {
	entries, err := fs.ReadDir(templates, dir)
	if err != nil {
		return nil, fmt.Errorf("ler migrations %s: %w", dir, err)
	}
	next := nextMigration(root)
	prefix := fmt.Sprintf("%06d_", next)
	if next > 1 {
		prefix += db + "_"
	}
	var files []common.File
	for _, entry := range entries {
		files = append(files, common.File{
			TemplatePath: path.Join(dir, entry.Name()),
			OutputPath:   path.Join("migrations", prefix+strings.TrimSuffix(entry.Name(), ".tmpl")),
		})
	}
	return files, nil
}

// nextMigration devolve o número seguinte ao da última migração do projeto.
func nextMigration(root string) int {
	entries, _ := os.ReadDir(filepath.Join(root, "migrations"))
	last := 0
	for _, entry := range entries {
		number, _, _ := strings.Cut(entry.Name(), "_")
		if n, err := strconv.Atoi(number); err == nil {
			last = max(last, n)
		}
	}
	return last + 1
}

// appendMissingLines acrescenta ao fim de content as linhas que ele ainda não
// tem.
func appendMissingLines(content []byte, lines []string) []byte {
	text := string(content)
	if len(lines) > 0 && strings.TrimSpace(text) == "" {
		text = ""
	}
	existing := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || slices.ContainsFunc(existing, func(current string) bool { return strings.TrimSpace(current) == strings.TrimSpace(line) }) {
			continue
		}
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += line + "\n"
		existing = append(existing, line)
	}
	return []byte(text)
}

// mergeJSON mescla values no objeto JSON de content; objetos são mesclados
// recursivamente e os demais valores são substituídos.
func mergeJSON(content []byte, values map[string]any) ([]byte, error) {
	var current map[string]any
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decodificar JSON: %w", err)
	}
	mergeValues(current, values)
	merged, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("codificar JSON: %w", err)
	}
	return append(merged, '\n'), nil
}

func mergeValues(target, values map[string]any) {
	for key, value := range values {
		nested, isObject := value.(map[string]any)
		existing, hasObject := target[key].(map[string]any)
		if isObject && hasObject {
			mergeValues(existing, nested)
			continue
		}
		target[key] = value
	}
}
