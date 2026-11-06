package generator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/fields"
)

// TypedLanguage descreve como uma linguagem traduz campos tipados em tipos,
// identificadores, declarações, literais, imports, placeholders SQL e
// formatação de código.
type TypedLanguage interface {
	Name() string
	Type(f fields.Field) string
	FieldName(f fields.Field) string
	Reserved() []string
	Placeholder(d fields.Dialect, index int) string
	StringLiteral(s string) string
	Format(path string, src []byte) []byte
	Dependencies(db string) []string

	completeField(f fields.Field, view *fieldView)
	completeEndpoint(root string, list []fields.Field, view *endpointView, data map[string]any)
}

var typedLanguages = map[string]TypedLanguage{
	"python": pythonLang{},
	"go":     goLang{},
}

// TypedLanguageFor devolve a implementação de TypedLanguage da linguagem informada.
func TypedLanguageFor(name string) (TypedLanguage, bool) {
	lang, ok := typedLanguages[name]
	return lang, ok
}

// fieldView é um campo pronto para os templates. Os textos que viram código já
// estão protegidos da troca de nomes do exemplo.
type fieldView struct {
	Name       string // identificador na linguagem (image_url, ImageURL)
	Column     string // coluna e chave JSON (<nome>_id em ref)
	Kind       string // tipo no crudding (str, decimal, ref...)
	Type       string // tipo na linguagem (str, Decimal, string, json.Number...)
	ModelType  string // tipo na entidade/model (ponteiro ou opcional quando nullable)
	Tag        string // anotação/tag de serialização (ex.: `json:"image_url"`)
	Decl       string // declaração na entidade/model
	InDecl     string // declaração no schema/struct de criação
	PatchDecl  string // declaração no schema/struct de atualização parcial
	OutDecl    string // declaração no schema/struct de resposta
	Sample     string // valor de exemplo para criar (literal da linguagem)
	Update     string // valor de exemplo para alterar (literal da linguagem)
	SampleJSON string // valor de exemplo para criar (JSON)
	UpdateJSON string // valor de exemplo para alterar (JSON)
	Nullable   bool
	Required   bool
	IsRef      bool
	Ref        string // tabela referenciada
	MaxLen     int    // tamanho máximo de str (0 = sem limite)
	IsDate     bool
	IsUUID     bool
	IsTime     bool
}

// endpointView agrega os campos e o que depende do conjunto (imports, SQL).
type endpointView struct {
	Fields          []fieldView
	Table           string // nome da tabela (protegido)
	Store           string // "sql" ou "memory"
	Dialect         string // "postgres", "mysql" ou ""
	HasRequired     bool
	HasRequiredRef  bool
	UpdateField     *fieldView // campo alterado no teste de PATCH
	SQL             fields.Queries
	ModelImports    []string // imports do módulo/arquivo de model
	TypesImports    []string // imports do arquivo de types/schemas
	Dependencies    []string // dependências do banco na linguagem
	RequiredColumns string   // colunas obrigatórias entre aspas ("a", "b")
	SamplePayload   string   // corpo de criação no teste (literal da linguagem)
	UpdatePayload   string   // corpo de alteração no teste (literal da linguagem)
	OwnerAware      bool     // se filtra por proprietário (user_id)
	OwnerColumn     string   // coluna do proprietário (ex.: "user_id")
}

// samples são valores de exemplo em JSON: o primeiro para criar, o segundo para alterar.
var samples = map[string][2]string{
	"str": {`"sample %s"`, `"updated %s"`}, "text": {`"sample %s"`, `"updated %s"`},
	"int": {"1", "2"}, "float": {"1.5", "2.5"}, "decimal": {"9.99", "19.99"}, "bool": {"true", "false"},
	"date": {`"2024-01-02"`, `"2024-02-03"`}, "datetime": {`"2024-01-02T03:04:05Z"`, `"2024-02-03T04:05:06Z"`},
	"uuid": {`"123e4567-e89b-12d3-a456-426614174000"`, `"223e4567-e89b-12d3-a456-426614174000"`},
	"ref":  {"1", "1"},
}

func newFieldView(lang TypedLanguage, field fields.Field) fieldView {
	sample := samples[field.Type]
	create, update := sample[0], sample[1]
	if field.Type == "ref" && (field.Ref() == "user" || field.Ref() == "users") {
		create, update = `"123e4567-e89b-12d3-a456-426614174000"`, `"123e4567-e89b-12d3-a456-426614174000"`
	} else if strings.Contains(create, "%s") {
		create, update = fmt.Sprintf(create, field.Name), fmt.Sprintf(update, field.Name)
	}
	view := fieldView{
		Name:       protect(lang.FieldName(field)),
		Column:     protect(field.Column()),
		Kind:       field.Type,
		Type:       protect(lang.Type(field)),
		Nullable:   field.Nullable,
		Required:   !field.Nullable,
		IsRef:      field.Type == "ref",
		Ref:        protect(field.Ref()),
		IsDate:     field.Type == "date",
		IsUUID:     field.Type == "uuid",
		IsTime:     field.Type == "datetime",
		SampleJSON: create,
		UpdateJSON: update,
	}
	if field.Type == "str" {
		view.MaxLen = 255
	}
	lang.completeField(field, &view)
	return view
}

// addFieldData acrescenta aos dados dos templates os campos, o SQL e os
// avisos de armazenamento.
func (t *typedEndpoint) addFieldData() {
	view := endpointView{
		Table:        protect(t.name),
		Store:        "memory",
		Dependencies: t.lang.Dependencies(t.m.Database),
	}
	var required []string
	for _, field := range t.fields {
		fv := newFieldView(t.lang, field)
		view.Fields = append(view.Fields, fv)
		if fv.Required {
			required = append(required, fmt.Sprintf("%q", field.Column()))
			view.HasRequired = true
			if fv.IsRef {
				view.HasRequiredRef = true
			}
		}
	}
	for index := range view.Fields {
		if !view.Fields[index].IsRef {
			view.UpdateField = &view.Fields[index]
			break
		}
	}
	view.RequiredColumns = protect(strings.Join(required, ", "))
	t.lang.completeEndpoint(t.root, t.fields, &view, t.data)

	if dialect, ok := fields.DialectFor(t.m.Database); ok {
		view.Store = "sql"
		view.Dialect = string(dialect)
		dateAsText := t.lang.Type(fields.Field{Type: "date"}) == t.lang.Type(fields.Field{Type: "str"})
		ownerCol := ""
		if t.opts.Owner || t.m.IsOwnerEndpoint(t.name) {
			ownerCol = "user_id"
			view.OwnerAware = true
			view.OwnerColumn = "user_id"
		}
		queries := fields.BuildQueriesWithOptions(dialect, t.name, t.fields, func(index int) string {
			return t.lang.Placeholder(dialect, index)
		}, dateAsText, ownerCol)
		quote := func(sql string) string { return protect(t.lang.StringLiteral(sql)) }
		view.SQL = fields.Queries{
			Select:    quote(queries.Select),
			SelectOne: quote(queries.SelectOne),
			Insert:    quote(queries.Insert),
			Update:    quote(queries.Update),
			Delete:    quote(queries.Delete),
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
