// Package fields lê os campos tipados de "crudding add endpoint <nome>
// campo:tipo..." e descreve como cada tipo vira código (Python e Go) e coluna
// SQL (postgres e mysql).
package fields

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Types são os tipos aceitos em "nome:tipo". ref é uma chave estrangeira para
// o id da tabela <nome>, gravada na coluna <nome>_id.
var Types = []string{"str", "text", "int", "float", "decimal", "bool", "date", "datetime", "uuid", "ref"}

// Field é um campo de endpoint: Name em snake_case, Type um de Types e
// Nullable quando o tipo termina em "?" (campo opcional, coluna NULL).
type Field struct {
	Name     string
	Type     string
	Nullable bool
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// reserved são nomes que quebrariam o código gerado: palavras reservadas de
// Python e Go, atributos do BaseModel do Pydantic e métodos gerados.
var reserved = []string{
	// Python
	"and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif", "else",
	"except", "false", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda", "none",
	"nonlocal", "not", "or", "pass", "raise", "return", "true", "try", "while", "with", "yield",
	// Go
	"case", "chan", "const", "default", "defer", "fallthrough", "func", "go", "goto", "interface",
	"map", "package", "range", "select", "struct", "switch", "type", "var",
	// Pydantic (BaseModel) e métodos gerados
	"copy", "dict", "json", "schema", "schema_json", "construct", "validate", "fields",
	"parse_obj", "parse_raw", "parse_file", "from_orm", "update_forward_refs", "metadata", "registry",
}

// Parse lê os campos na sintaxe "nome:tipo" ou "nome:tipo?". Recusa nomes
// fora de snake_case, reservados, "id" e repetidos (contando a coluna
// <nome>_id de ref).
func Parse(specs []string) ([]Field, error) {
	var result []Field
	columns := map[string]string{}
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		name, kind, ok := strings.Cut(spec, ":")
		if !ok || name == "" || kind == "" {
			return nil, fmt.Errorf("campo %q inválido: use nome:tipo (ex.: price:decimal, image_url:str?, cart:ref)", spec)
		}
		field := Field{Name: name, Type: strings.TrimSuffix(kind, "?"), Nullable: strings.HasSuffix(kind, "?")}
		if !slices.Contains(Types, field.Type) {
			return nil, fmt.Errorf("campo %q: tipo %q desconhecido (tipos: %s)", spec, field.Type, strings.Join(Types, ", "))
		}
		if !namePattern.MatchString(name) {
			return nil, fmt.Errorf("campo %q: nome inválido %q; use snake_case com letras minúsculas, números e sublinhados, começando por letra", spec, name)
		}
		if name == "id" || field.Column() == "id" {
			return nil, fmt.Errorf("campo %q: id é gerado automaticamente e não pode ser declarado", spec)
		}
		if slices.Contains(reserved, name) || strings.HasPrefix(name, "model_") {
			return nil, fmt.Errorf("campo %q: %q é uma palavra reservada", spec, name)
		}
		if previous, ok := columns[field.Column()]; ok {
			return nil, fmt.Errorf("campo %q: a coluna %q já é usada por %q", spec, field.Column(), previous)
		}
		columns[field.Column()] = spec
		result = append(result, field)
	}
	return result, nil
}

// String devolve o campo na sintaxe da linha de comando ("image_url:str?").
func (f Field) String() string {
	if f.Nullable {
		return f.Name + ":" + f.Type + "?"
	}
	return f.Name + ":" + f.Type
}

// Specs devolve os campos na sintaxe da linha de comando, para o project.crud.
func Specs(list []Field) []string {
	specs := make([]string, len(list))
	for index, field := range list {
		specs[index] = field.String()
	}
	return specs
}

// Column é o nome da coluna e da chave JSON: o nome, ou <nome>_id em ref.
func (f Field) Column() string {
	if f.Type == "ref" {
		return f.Name + "_id"
	}
	return f.Name
}

// Ref é a tabela (endpoint) referenciada por um campo ref.
func (f Field) Ref() string {
	if f.Type == "ref" {
		return f.Name
	}
	return ""
}

// Refs lista os endpoints referenciados pelos campos, sem repetição.
func Refs(list []Field) []string {
	var refs []string
	for _, field := range list {
		if ref := field.Ref(); ref != "" && !slices.Contains(refs, ref) {
			refs = append(refs, ref)
		}
	}
	return refs
}
