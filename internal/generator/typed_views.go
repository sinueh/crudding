package generator

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/fields"
)

// fieldView é um campo pronto para os templates. Os textos que vêm do campo
// já estão protegidos da troca de nomes do exemplo; Type e os booleanos não
// precisam, porque não viram texto no código.
type fieldView struct {
	Name     string // nome do campo (em ref, o endpoint referenciado)
	Column   string // coluna, chave JSON e atributo Python (<nome>_id em ref)
	Type     string
	Nullable bool
	Required bool
	IsRef    bool
	Ref      string // tabela referenciada
	MaxLen   int    // tamanho máximo de str (0 = sem limite)

	// Python
	PyType      string // tipo da anotação (str, Decimal, date...)
	InDecl      string // declaração no <X>In
	PatchDecl   string // declaração no <X>Patch
	OutDecl     string // declaração no <X>Out
	EntityDecl  string // declaração no dataclass da entidade
	PySample    string // valor de exemplo (literal Python)
	PyUpdate    string // outro valor de exemplo (literal Python)
	SampleJSON  string // valor de exemplo (JSON)
	UpdateJSON  string // outro valor de exemplo (JSON)
	GoName      string // nome do campo na struct Go (ImageURL, CartID)
	GoType      string // tipo Go do valor (string, json.Number, time.Time...)
	GoModelType string // tipo na entidade (ponteiro quando nullable)
	GoTag       string // tag da struct: `json:"image_url"`
	IsDate      bool
	IsUUID      bool
	IsTime      bool
}

// endpointView agrega os campos e o que depende do conjunto (imports, SQL).
type endpointView struct {
	Fields         []fieldView
	Table          string // nome da tabela (protegido)
	Store          string // "sql" ou "memory"
	Dialect        string // "postgres", "mysql" ou ""
	HasRequired    bool
	HasRequiredRef bool
	UpdateField    *fieldView // campo alterado no teste de PATCH

	// SQL com os placeholders do driver, como literais da linguagem.
	SQL fields.Queries

	// Python
	PyImports       []string // imports do módulo de model
	PySamplePayload string   // dict Python com os campos obrigatórios e os opcionais não-ref
	PyUpdatePayload string
	// PyRequiredColumns são as colunas obrigatórias como argumentos Python ("a", "b").
	PyRequiredColumns string
	// Go
	GoModelImports []string
	GoTypesImports []string
	SampleJSON     string // corpo JSON de criação (literal Go)
	UpdateJSON     string // corpo JSON de alteração (literal Go)
}

var goInitialisms = map[string]string{
	"id": "ID", "url": "URL", "uri": "URI", "uuid": "UUID", "api": "API", "http": "HTTP", "json": "JSON",
	"xml": "XML", "sql": "SQL", "ip": "IP", "html": "HTML", "css": "CSS", "sku": "SKU",
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

var (
	pyTypes = map[string]string{
		"str": "str", "text": "str", "int": "int", "float": "float", "decimal": "Decimal", "bool": "bool",
		"date": "date", "datetime": "datetime", "uuid": "UUID", "ref": "int",
	}
	goTypes = map[string]string{
		"str": "string", "text": "string", "int": "int", "float": "float64", "decimal": "json.Number", "bool": "bool",
		"date": "string", "datetime": "time.Time", "uuid": "string", "ref": "uint",
	}
	// samples são valores de exemplo em JSON: o primeiro para criar, o segundo para alterar.
	samples = map[string][2]string{
		"str": {`"sample %s"`, `"updated %s"`}, "text": {`"sample %s"`, `"updated %s"`},
		"int": {"1", "2"}, "float": {"1.5", "2.5"}, "decimal": {"9.99", "19.99"}, "bool": {"true", "false"},
		"date": {`"2024-01-02"`, `"2024-02-03"`}, "datetime": {`"2024-01-02T03:04:05Z"`, `"2024-02-03T04:05:06Z"`},
		"uuid": {`"123e4567-e89b-12d3-a456-426614174000"`, `"223e4567-e89b-12d3-a456-426614174000"`},
		"ref":  {"1", "1"},
	}
)

func newFieldView(field fields.Field) fieldView {
	view := fieldView{
		Name:     protect(field.Name),
		Column:   protect(field.Column()),
		Type:     field.Type,
		Nullable: field.Nullable,
		Required: !field.Nullable,
		IsRef:    field.Type == "ref",
		Ref:      protect(field.Ref()),
		IsDate:   field.Type == "date",
		IsUUID:   field.Type == "uuid",
		IsTime:   field.Type == "datetime",
		PyType:   pyTypes[field.Type],
		GoType:   goTypes[field.Type],
	}
	if field.Type == "str" {
		view.MaxLen = 255
	}

	column := field.Column()
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
	view.InDecl = declare(view.PyType, field.Nullable)
	view.PatchDecl = declare(view.PyType, true)
	if field.Nullable {
		view.OutDecl = protect(fmt.Sprintf("%s: %s | None = None", column, view.PyType))
		view.EntityDecl = view.OutDecl
	} else {
		view.OutDecl = protect(fmt.Sprintf("%s: %s", column, view.PyType))
		view.EntityDecl = view.OutDecl
	}

	sample := samples[field.Type]
	create, update := sample[0], sample[1]
	if strings.Contains(create, "%s") {
		create, update = fmt.Sprintf(create, field.Name), fmt.Sprintf(update, field.Name)
	}
	view.SampleJSON, view.UpdateJSON = create, update
	view.PySample, view.PyUpdate = protect(pythonLiteral(create)), protect(pythonLiteral(update))

	view.GoName = protect(goFieldName(column))
	view.GoModelType = view.GoType
	if field.Nullable {
		view.GoModelType = "*" + view.GoType
	}
	view.GoTag = protect(fmt.Sprintf("`json:\"%s\"`", column))
	return view
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

// addFieldData acrescenta aos dados dos templates os campos, o SQL e os
// avisos de armazenamento.
func (t *typedEndpoint) addFieldData() {
	view := endpointView{Table: protect(t.name), Store: "memory"}
	pyImports := map[string]bool{}
	goModel, goTypes := map[string]bool{}, map[string]bool{"errors": true}
	sample, update := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	pySample, pyUpdate, pyRequired := []string{}, []string{}, []string{}
	for _, field := range t.fields {
		fv := newFieldView(field)
		view.Fields = append(view.Fields, fv)
		if fv.Required {
			pyRequired = append(pyRequired, fmt.Sprintf("%q", field.Column()))
			view.HasRequired = true
			if fv.IsRef {
				view.HasRequiredRef = true
			}
		}
		switch field.Type {
		case "decimal":
			pyImports["from decimal import Decimal"] = true
			goModel["encoding/json"], goTypes["encoding/json"] = true, true
		case "date":
			pyImports["from datetime import date"] = true
			goTypes["time"] = true
		case "datetime":
			pyImports["from datetime import datetime"] = true
			goModel["time"], goTypes["time"] = true, true
		case "uuid":
			pyImports["from uuid import UUID"] = true
			goTypes["strings"] = true
		}
		if fv.Required || !fv.IsRef {
			sample[field.Column()] = json.RawMessage(fv.SampleJSON)
			pySample = append(pySample, fmt.Sprintf("%q: %s", field.Column(), pythonLiteral(fv.SampleJSON)))
		}
	}
	for index := range view.Fields {
		if !view.Fields[index].IsRef {
			view.UpdateField = &view.Fields[index]
			field := t.fields[index]
			update[field.Column()] = json.RawMessage(view.Fields[index].UpdateJSON)
			pyUpdate = append(pyUpdate, fmt.Sprintf("%q: %s", field.Column(), pythonLiteral(view.Fields[index].UpdateJSON)))
			break
		}
	}
	// date + datetime viram um import só.
	if pyImports["from datetime import date"] && pyImports["from datetime import datetime"] {
		delete(pyImports, "from datetime import date")
		delete(pyImports, "from datetime import datetime")
		pyImports["from datetime import date, datetime"] = true
	}
	view.PyImports = sortedKeys(pyImports)
	view.GoModelImports = sortedKeys(goModel)
	view.GoTypesImports = sortedKeys(goTypes)
	view.PySamplePayload = protect("{" + strings.Join(pySample, ", ") + "}")
	view.PyUpdatePayload = protect("{" + strings.Join(pyUpdate, ", ") + "}")
	view.PyRequiredColumns = protect(strings.Join(pyRequired, ", "))
	sampleBody, _ := json.Marshal(sample)
	updateBody, _ := json.Marshal(update)
	view.SampleJSON = protect(goString(string(sampleBody)))
	view.UpdateJSON = protect(goString(string(updateBody)))

	if dialect, ok := fields.DialectFor(t.m.Database); ok {
		view.Store = "sql"
		view.Dialect = string(dialect)
		var queries fields.Queries
		quote := goString
		if t.m.Language == "python" {
			queries = fields.PythonQueries(dialect, t.name, t.fields)
			quote = pythonString
		} else {
			queries = fields.GoQueries(dialect, t.name, t.fields)
		}
		view.SQL = fields.Queries{
			Select:    protect(quote(queries.Select)),
			SelectOne: protect(quote(queries.SelectOne)),
			Insert:    protect(quote(queries.Insert)),
			Update:    protect(quote(queries.Update)),
			Delete:    protect(quote(queries.Delete)),
			Returning: queries.Returning,
		}
	} else if t.m.Database != "" {
		t.notes = append(t.notes, fmt.Sprintf("o banco %s ainda não tem acesso a dados gerado: o endpoint %s usa armazenamento em memória (veja o TODO no código gerado)", t.m.Database, t.name))
	} else {
		t.notes = append(t.notes, fmt.Sprintf("projeto sem banco de dados: o endpoint %s usa armazenamento em memória e nenhuma migração foi criada (crudding add db <postgres|mysql> e recrie o endpoint para gerar SQL)", t.name))
	}
	t.data["Endpoint"] = view
	t.data["Fields"] = view.Fields
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
