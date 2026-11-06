package generator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/sinueh/crudding/internal/generator/fields"
)

type goLang struct{}

var (
	goTypes = map[string]string{
		"str": "string", "text": "string", "int": "int", "float": "float64", "decimal": "json.Number", "bool": "bool",
		"date": "string", "datetime": "time.Time", "uuid": "string", "ref": "uint",
	}
	goInitialisms = map[string]string{
		"id": "ID", "url": "URL", "uri": "URI", "uuid": "UUID", "api": "API", "http": "HTTP", "json": "JSON",
		"xml": "XML", "sql": "SQL", "ip": "IP", "html": "HTML", "css": "CSS", "sku": "SKU",
	}
	goReserved = []string{
		"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for",
		"func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var",
	}
)

func (goLang) Name() string                    { return "go" }
func (goLang) Type(f fields.Field) string      { return goTypes[f.Type] }
func (goLang) FieldName(f fields.Field) string { return goFieldName(f.Column()) }
func (goLang) Reserved() []string              { return goReserved }
func (goLang) StringLiteral(s string) string   { return goString(s) }

func (goLang) Placeholder(d fields.Dialect, index int) string {
	if d == fields.MySQL {
		return "?"
	}
	return fmt.Sprintf("$%d", index)
}

func (goLang) Format(path string, src []byte) []byte {
	if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".go.tmpl") {
		// Alinha structs e imports como o gofmt; se não formatar, o código
		// fica como saiu do template para o erro aparecer no go build.
		if formatted, err := format.Source(src); err == nil {
			return formatted
		}
	}
	return src
}

func (goLang) Dependencies(db string) []string {
	switch db {
	case "postgres":
		return []string{"github.com/lib/pq v1.10.9"}
	case "mysql":
		return []string{"github.com/go-sql-driver/mysql v1.8.1"}
	case "mongo":
		return []string{"go.mongodb.org/mongo-driver v1.14.0"}
	default:
		return nil
	}
}

func (g goLang) completeField(field fields.Field, view *fieldView) {
	column := field.Column()
	name := g.FieldName(field)
	goType := g.Type(field)
	modelType := goType
	if field.Nullable {
		modelType = "*" + goType
	}
	tag := fmt.Sprintf("`json:\"%s\"`", column)
	view.ModelType = protect(modelType)
	view.Tag = protect(tag)
	view.Decl = protect(fmt.Sprintf("%s %s %s", name, modelType, tag))
	view.InDecl = protect(fmt.Sprintf("%s *%s %s", name, goType, tag))
	view.PatchDecl = view.InDecl
	view.OutDecl = view.Decl
	view.Sample = protect(view.SampleJSON)
	view.Update = protect(view.UpdateJSON)
}

func (goLang) completeEndpoint(root string, list []fields.Field, view *endpointView, data map[string]any) {
	if module := goModule(root); module != "" {
		data["GoModule"] = module
	}
	modelImports, typesImports := map[string]bool{}, map[string]bool{"errors": true}
	sample, update := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	for index, field := range list {
		fv := view.Fields[index]
		switch field.Type {
		case "decimal":
			modelImports["encoding/json"], typesImports["encoding/json"] = true, true
		case "date":
			typesImports["time"] = true
		case "datetime":
			modelImports["time"], typesImports["time"] = true, true
		case "uuid":
			typesImports["strings"] = true
		}
		if fv.Required || !fv.IsRef {
			sample[field.Column()] = json.RawMessage(fv.SampleJSON)
		}
	}
	for index, field := range list {
		if !view.Fields[index].IsRef {
			update[field.Column()] = json.RawMessage(view.Fields[index].UpdateJSON)
			break
		}
	}
	view.ModelImports = sortedKeys(modelImports)
	view.TypesImports = sortedKeys(typesImports)
	sampleBody, _ := json.Marshal(sample)
	updateBody, _ := json.Marshal(update)
	view.SamplePayload = protect(goString(string(sampleBody)))
	view.UpdatePayload = protect(goString(string(updateBody)))
}

func goFieldName(column string) string {
	var b strings.Builder
	for _, part := range strings.Split(column, "_") {
		if initialism, ok := goInitialisms[part]; ok {
			b.WriteString(initialism)
		} else if part != "" {
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}

// goString devolve text como literal de string Go: entre crases quando possível.
func goString(text string) string {
	if !strings.Contains(text, "`") {
		return "`" + text + "`"
	}
	encoded, _ := json.Marshal(text)
	return string(encoded)
}

// goModule lê o nome do módulo no go.mod do projeto, que pode ter mudado
// desde a geração.
func goModule(root string) string {
	file, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if module, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "module "); ok {
			return strings.TrimSpace(module)
		}
	}
	return ""
}
