package generator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/fields"
)

type pythonLang struct{}

var (
	pyTypes = map[string]string{
		"str": "str", "text": "str", "int": "int", "float": "float", "decimal": "Decimal", "bool": "bool",
		"date": "date", "datetime": "datetime", "uuid": "UUID", "ref": "int",
	}
	pyReserved = []string{
		// Palavras reservadas do Python
		"and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif", "else",
		"except", "false", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda", "none",
		"nonlocal", "not", "or", "pass", "raise", "return", "true", "try", "while", "with", "yield",
		// Atributos do BaseModel (Pydantic) e namespace reservado model_*
		"copy", "dict", "json", "schema", "schema_json", "construct", "validate", "fields",
		"parse_obj", "parse_raw", "parse_file", "from_orm", "update_forward_refs", "metadata", "registry",
		"model_*",
	}
)

func (pythonLang) Name() string                      { return "python" }
func (pythonLang) Type(f fields.Field) string        { return pyTypes[f.Type] }
func (pythonLang) FieldName(f fields.Field) string   { return f.Column() }
func (pythonLang) Reserved() []string                { return pyReserved }
func (pythonLang) Placeholder(fields.Dialect, int) string { return "%s" }
func (pythonLang) StringLiteral(s string) string     { return pythonString(s) }
func (pythonLang) Format(_ string, src []byte) []byte { return src }

func (pythonLang) Dependencies(db string) []string {
	switch db {
	case "postgres":
		return []string{"psycopg2-binary>=2.9"}
	case "mysql":
		return []string{"pymysql>=1.1", "cryptography>=42.0"}
	case "mongo":
		return []string{"pymongo>=4.6"}
	default:
		return []string{"pydantic>=2.0"}
	}
}

func (p pythonLang) completeField(field fields.Field, view *fieldView) {
	column := field.Column()
	pyType := p.Type(field)
	modelType := pyType
	if field.Nullable {
		modelType += " | None"
	}
	view.ModelType = protect(modelType)

	var constraint []string
	switch field.Type {
	case "str":
		constraint = append(constraint, "max_length=255")
	case "ref":
		constraint = append(constraint, "gt=0")
	}
	declare := func(annotation string, optional bool) string {
		args := slices.Clone(constraint)
		if optional {
			annotation += " | None"
			args = append([]string{"default=None"}, args...)
		}
		switch {
		case len(args) == 0:
			return protect(fmt.Sprintf("%s: %s", column, annotation))
		case len(args) == 1 && optional && len(constraint) == 0:
			return protect(fmt.Sprintf("%s: %s = None", column, annotation))
		}
		return protect(fmt.Sprintf("%s: %s = Field(%s)", column, annotation, strings.Join(args, ", ")))
	}
	view.InDecl = declare(pyType, field.Nullable)
	view.PatchDecl = declare(pyType, true)
	if field.Nullable {
		view.OutDecl = protect(fmt.Sprintf("%s: %s | None = None", column, pyType))
	} else {
		view.OutDecl = protect(fmt.Sprintf("%s: %s", column, pyType))
	}
	view.Decl = view.OutDecl
	view.Sample = protect(pythonLiteral(view.SampleJSON))
	view.Update = protect(pythonLiteral(view.UpdateJSON))
}

func (pythonLang) completeEndpoint(_ string, list []fields.Field, view *endpointView, _ map[string]any) {
	imports := map[string]bool{}
	var sample, update []string
	for index, field := range list {
		fv := view.Fields[index]
		switch field.Type {
		case "decimal":
			imports["from decimal import Decimal"] = true
		case "date":
			imports["from datetime import date"] = true
		case "datetime":
			imports["from datetime import datetime"] = true
		case "uuid":
			imports["from uuid import UUID"] = true
		}
		if fv.Required || !fv.IsRef {
			sample = append(sample, fmt.Sprintf("%q: %s", field.Column(), pythonLiteral(fv.SampleJSON)))
		}
	}
	if view.UpdateField != nil {
		for index, field := range list {
			if !view.Fields[index].IsRef {
				update = append(update, fmt.Sprintf("%q: %s", field.Column(), pythonLiteral(view.Fields[index].UpdateJSON)))
				break
			}
		}
	}
	// date + datetime viram um import só.
	if imports["from datetime import date"] && imports["from datetime import datetime"] {
		delete(imports, "from datetime import date")
		delete(imports, "from datetime import datetime")
		imports["from datetime import date, datetime"] = true
	}
	view.ModelImports = sortedKeys(imports)
	view.SamplePayload = protect("{" + strings.Join(sample, ", ") + "}")
	view.UpdatePayload = protect("{" + strings.Join(update, ", ") + "}")
}

// pythonString devolve text como literal de string Python entre aspas simples.
func pythonString(text string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\n", `\n`).Replace(text) + "'"
}

// pythonLiteral converte um valor JSON simples em literal Python.
func pythonLiteral(value string) string {
	switch value {
	case "true":
		return "True"
	case "false":
		return "False"
	case "null":
		return "None"
	}
	return value
}
