package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/generator/custom"
	"github.com/sinueh/crudding/internal/generator/fields"
	"github.com/sinueh/crudding/internal/manifest"
)

// Campos tipados ("crudding add endpoint product name:str price:decimal")
//
// O endpoint continua saindo do campo template_endpoint do project.crud: os
// arquivos novos têm os mesmos caminhos das cópias do exemplo e as linhas de
// register são duplicadas do mesmo jeito. Só o conteúdo muda: em vez de copiar
// o exemplo, cada arquivo é renderizado a partir de
// templates/langs/<lang>/fields/<from>, onde <from> é o template que gerou o
// arquivo do exemplo no architectures/<arch>/project.crud.tmpl. O resultado
// passa pela mesma troca de nomes da cópia, então os identificadores batem com
// as linhas de register. O que vem dos campos (nomes, colunas, SQL) é
// protegido dessa troca.
//
// Além disso, templates/langs/<lang>/fields/fields.crud.tmpl lista arquivos
// extras (testes), pacotes e linhas para arquivos de dependências, e um banco
// SQL configurado (postgres/mysql) ganha a migração
// migrations/NNNNNN_create_<nome>.up.sql/.down.sql.

// fieldsDir é a pasta dos templates de campos tipados dentro de templates/langs/<lang>.
const fieldsDir = "fields"

// fieldsRecipe é o fields.crud.tmpl de uma linguagem.
type fieldsRecipe struct {
	// Files são arquivos extras do endpoint (ex.: testes). from é relativo a
	// templates/langs/<lang>; to passa pela troca de nomes.
	Files []manifest.FileMapping `json:"files"`
	// Packages são pastas Python que precisam de __init__.py.
	Packages []string `json:"packages"`
	// Append acrescenta ao fim de into as linhas de from que ele ainda não tem.
	Append []databaseEdit `json:"append"`
}

// typedEndpoint reúne o que a geração tipada precisa saber do projeto.
type typedEndpoint struct {
	root    string
	m       manifest.Manifest
	lang    TypedLanguage
	name    string
	fields  []fields.Field
	opts    EndpointOptions
	langFS  fs.FS
	sources map[string]string // arquivo do exemplo -> template de origem
	data    map[string]any
	rename  func(string) string
	recipe  fieldsRecipe
	notes   []string
}

// ParseEndpointFields valida os campos de "add endpoint" sem olhar o projeto.
func ParseEndpointFields(specs []string) ([]fields.Field, error) {
	return fields.Parse(specs)
}

// AddTypedEndpoint cria o endpoint name com os campos specs ("nome:tipo"),
// gerando model, schemas, acesso a dados, controller, migração e teste. Sem
// campos, é o mesmo que AddEndpoint. Devolve também avisos para o usuário.
func AddTypedEndpoint(root string, m manifest.Manifest, name string, specs []string, templates fs.FS) (common.Result, []string, error) {
	return AddTypedEndpointWithOptions(root, m, EndpointOptions{Name: name, Specs: specs}, templates)
}

// AddTypedEndpointWithOptions cria o endpoint tipado com opções adicionais de proteção e propriedade.
func AddTypedEndpointWithOptions(root string, m manifest.Manifest, opts EndpointOptions, templates fs.FS) (common.Result, []string, error) {
	if (opts.Protected || len(opts.Roles) > 0 || opts.Owner) && !m.HasAuth() {
		return common.Result{}, nil, fmt.Errorf("o projeto não possui autenticação configurada; use 'crudding add auth' antes de usar --protected, --roles ou --owner")
	}
	if opts.Owner {
		hasUserRef := slices.ContainsFunc(opts.Specs, func(s string) bool {
			return strings.HasPrefix(s, "user:") || strings.HasPrefix(s, "user_id:")
		})
		if !hasUserRef {
			opts.Specs = append(opts.Specs, "user:ref")
		}
	}
	if len(opts.Specs) == 0 {
		result, err := AddEndpointWithOptions(root, m, opts)
		return result, nil, err
	}
	m = current(root, m)
	if err := CheckEndpointSupport(m); err != nil {
		return common.Result{}, nil, err
	}
	var reserved []string
	if lang, ok := TypedLanguageFor(m.Language); ok {
		reserved = lang.Reserved()
	}
	list, err := fields.Parse(opts.Specs, reserved...)
	if err != nil {
		return common.Result{}, nil, err
	}
	if m.HasEndpoint(opts.Name) {
		return common.Result{}, nil, fmt.Errorf("endpoint %q já existe no %s", opts.Name, manifest.FileName)
	}
	for _, ref := range fields.Refs(list) {
		if ref != opts.Name && len(m.EndpointFields[ref]) == 0 {
			if m.HasAuth() && (ref == "user" || ref == "users" || (m.Auth != nil && ref == m.Auth.UserEntity)) {
				continue
			}
			return common.Result{}, nil, fmt.Errorf("campo %s:ref: o endpoint %q não tem tabela gerada pelo crudding; crie-o antes com campos (crudding add endpoint %s campo:tipo...)", ref, ref, ref)
		}
	}

	typed, err := newTypedEndpointWithOptions(root, m, opts, list, templates)
	if err != nil {
		return common.Result{}, nil, err
	}
	patch, err := typed.patch()
	if err != nil {
		return common.Result{}, nil, err
	}
	result, err := common.Apply(root, nil, patch)
	if err != nil {
		return common.Result{}, nil, err
	}

	m.AddEndpoint(opts.Name)
	m.SetEndpointFields(opts.Name, fields.Specs(list))
	m.SetEndpointAuth(opts.Name, opts.Protected, opts.Roles, opts.Owner)
	if err := common.RecordCreated(&m, result.Created); err != nil {
		return common.Result{}, nil, err
	}
	if err := manifest.Write(root, m); err != nil {
		return common.Result{}, nil, err
	}
	return result, typed.notes, nil
}

// typedEndpointExtras lista os arquivos extras (ex.: testes) que o
// AddTypedEndpoint criou para name, para o remove endpoint apagá-los também.
// Migrações ficam de fora: não dá para saber se já foram aplicadas.
func typedEndpointExtras(root string, m manifest.Manifest, name string, templates fs.FS) ([]string, error) {
	specs := m.EndpointFields[name]
	if len(specs) == 0 || templates == nil {
		return nil, nil
	}
	var reserved []string
	if lang, ok := TypedLanguageFor(m.Language); ok {
		reserved = lang.Reserved()
	}
	list, err := fields.Parse(specs, reserved...)
	if err != nil {
		return nil, fmt.Errorf("endpoint_fields.%s do %s: %w", name, manifest.FileName, err)
	}
	typed, err := newTypedEndpoint(root, m, name, list, templates)
	if err != nil {
		return nil, err
	}
	var extras []string
	for _, file := range typed.recipe.Files {
		extras = append(extras, typed.rename(file.To))
	}
	return extras, nil
}

func newTypedEndpoint(root string, m manifest.Manifest, name string, list []fields.Field, templates fs.FS) (*typedEndpoint, error) {
	return newTypedEndpointWithOptions(root, m, EndpointOptions{Name: name}, list, templates)
}

func newTypedEndpointWithOptions(root string, m manifest.Manifest, opts EndpointOptions, list []fields.Field, templates fs.FS) (*typedEndpoint, error) {
	stack := m.Language + " + " + m.Framework + " + " + m.Architecture
	unsupported := func(reason string) error {
		return fmt.Errorf("campos tipados não são suportados para %s ainda (%s); rode sem campos para copiar o endpoint de exemplo", stack, reason)
	}
	if m.Architecture == manifest.CustomArchitecture || m.Language == "" {
		return nil, unsupported("projeto com arquitetura custom")
	}
	lang, ok := TypedLanguageFor(m.Language)
	if !ok {
		return nil, unsupported("linguagem sem suporte a campos tipados")
	}
	langFS, err := fs.Sub(templates, path.Join("templates", "langs", m.Language))
	if err != nil {
		return nil, fmt.Errorf("acessar templates da linguagem %s: %w", m.Language, err)
	}
	if _, err := fs.Stat(langFS, path.Join(fieldsDir, "fields.crud.tmpl")); err != nil {
		return nil, unsupported("linguagem sem templates de campos")
	}

	rename, err := custom.EndpointRenamer(m, opts.Name)
	if err != nil {
		return nil, err
	}
	config := m.Config()
	data := common.TemplateData(config)
	archManifest, err := renderArchitecture(langFS, m.Architecture, data)
	if err != nil {
		return nil, unsupported(err.Error())
	}
	if archManifest.TemplateEndpoint.Name != m.TemplateEndpoint.Name {
		return nil, unsupported(fmt.Sprintf("o endpoint de exemplo do %s (%q) não é o da arquitetura (%q)", manifest.FileName, m.TemplateEndpoint.Name, archManifest.TemplateEndpoint.Name))
	}
	sources := map[string]string{}
	for _, mapping := range archManifest.FileMappings {
		sources[mapping.To] = mapping.From
	}
	for key, value := range archManifest.Vars {
		data[key] = value
	}

	typed := &typedEndpoint{root: root, m: m, lang: lang, name: opts.Name, fields: list, opts: opts, langFS: langFS, sources: sources, data: data, rename: rename}
	typed.addFieldData()

	recipeContent, err := typed.render(path.Join(fieldsDir, "fields.crud.tmpl"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(recipeContent, &typed.recipe); err != nil {
		return nil, fmt.Errorf("decodificar %s/fields.crud.tmpl: %w", fieldsDir, err)
	}
	return typed, nil
}

// renderArchitecture renderiza o project.crud.tmpl da arquitetura, que liga
// cada arquivo do endpoint de exemplo ao template que o gerou.
func renderArchitecture(langFS fs.FS, architecture string, data map[string]any) (manifest.Manifest, error) {
	manifestPath := path.Join("architectures", architecture, "project.crud.tmpl")
	content, err := fs.ReadFile(langFS, manifestPath)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("arquitetura %q sem %s", architecture, manifestPath)
	}
	parsed, err := template.New("manifest").Option("missingkey=error").Parse(string(content))
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("interpretar %s: %w", manifestPath, err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, data); err != nil {
		return manifest.Manifest{}, fmt.Errorf("renderizar %s: %w", manifestPath, err)
	}
	return manifest.Parse(rendered.Bytes(), manifestPath)
}

func (t *typedEndpoint) patch() (common.Patch, error) {
	patch, err := custom.EndpointPatchWithOptions(t.root, t.m, t.name, t.opts.Protected || t.opts.Owner, t.opts.Roles)
	if err != nil {
		return common.Patch{}, err
	}
	// EndpointPatch copia os arquivos do exemplo na ordem de template_endpoint.files.
	if len(patch.Raw) != len(t.m.TemplateEndpoint.Files) {
		return common.Patch{}, fmt.Errorf("endpoint.files: esperado %d arquivos, gerados %d", len(t.m.TemplateEndpoint.Files), len(patch.Raw))
	}
	for index, source := range t.m.TemplateEndpoint.Files {
		from, ok := t.sources[source]
		if !ok {
			return common.Patch{}, fmt.Errorf("campos tipados não são suportados para %s: o arquivo de exemplo %s não vem dos templates da arquitetura", t.stack(), source)
		}
		content, err := t.renderTyped(from)
		if err != nil {
			return common.Patch{}, err
		}
		patch.Raw[index].Content = content
	}

	for _, file := range t.recipe.Files {
		content, err := t.renderTyped(file.From)
		if err != nil {
			return common.Patch{}, err
		}
		output := t.rename(file.To)
		if !fs.ValidPath(output) || !filepath.IsLocal(filepath.FromSlash(output)) || output == manifest.FileName {
			return common.Patch{}, fmt.Errorf("%s/fields.crud.tmpl: caminho %q inválido", fieldsDir, output)
		}
		patch.Raw = append(patch.Raw, common.RawFile{OutputPath: output, Content: content})
	}
	patch.Packages = append(patch.Packages, t.recipe.Packages...)
	for _, edit := range t.recipe.Append {
		content, err := t.render(path.Join(fieldsDir, edit.From))
		if err != nil {
			return common.Patch{}, err
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			continue
		}
		lines := strings.Split(text, "\n")
		if err := addRewrite(&patch, t.root, edit.Into, func(content []byte) ([]byte, error) {
			return appendMissingLines(content, lines), nil
		}); err != nil {
			return common.Patch{}, err
		}
	}

	if dialect, ok := fields.DialectFor(t.m.Database); ok {
		prefix := fmt.Sprintf("migrations/%06d_create_%s", nextMigration(t.root), t.name)
		patch.Raw = append(patch.Raw,
			common.RawFile{OutputPath: prefix + ".up.sql", Content: []byte(fields.CreateTable(dialect, t.name, t.fields))},
			common.RawFile{OutputPath: prefix + ".down.sql", Content: []byte(fields.DropTable(dialect, t.name))},
		)
	}
	return patch, nil
}

func (t *typedEndpoint) stack() string {
	return t.m.Language + " + " + t.m.Framework + " + " + t.m.Architecture
}

// renderTyped renderiza fields/<from>, aplica a troca de nomes do exemplo e
// formata o código pela linguagem ativa.
func (t *typedEndpoint) renderTyped(from string) ([]byte, error) {
	typedPath := path.Join(fieldsDir, from)
	if _, err := fs.Stat(t.langFS, typedPath); err != nil {
		return nil, fmt.Errorf("campos tipados não são suportados para %s ainda (falta o template %s); rode sem campos para copiar o endpoint de exemplo", t.stack(), typedPath)
	}
	content, err := t.render(typedPath)
	if err != nil {
		return nil, err
	}
	output := []byte(renameUnprotected(string(content), t.rename))
	return t.lang.Format(from, output), nil
}

// render executa um template de templates/langs/<lang> com os partials de
// fields/partials disponíveis via {{template "nome" .}}.
func (t *typedEndpoint) render(file string) ([]byte, error) {
	partials, err := fs.Glob(t.langFS, path.Join(fieldsDir, "partials", "*.tmpl"))
	if err != nil {
		return nil, err
	}
	files := slices.DeleteFunc(partials, func(partial string) bool { return partial == file })
	var parsed *template.Template
	funcs := template.FuncMap{
		"str": func(text string) string { return protect(t.lang.StringLiteral(text)) },
		// include devolve a saída de um {{define}} como texto, para usar em expressões.
		"include": func(name string, data any) (string, error) {
			var output bytes.Buffer
			err := parsed.ExecuteTemplate(&output, name, data)
			return output.String(), err
		},
	}
	parsed, err = template.New(path.Base(file)).Option("missingkey=error").Funcs(typedFuncs).Funcs(funcs).ParseFS(t.langFS, append(files, file)...)
	if err != nil {
		return nil, fmt.Errorf("ler template %s: %w", file, err)
	}
	var output bytes.Buffer
	if err := parsed.ExecuteTemplate(&output, path.Base(file), t.data); err != nil {
		return nil, fmt.Errorf("renderizar template %s: %w", file, err)
	}
	return output.Bytes(), nil
}

// Os textos que vêm dos campos ficam entre protectStart e protectEnd durante a
// renderização, para a troca de nomes do exemplo não alterá-los (ex.: um
// campo template_id num projeto cujo exemplo é template).
const (
	protectStart = "\x1e"
	protectEnd   = "\x1f"
)

func protect(text string) string { return protectStart + text + protectEnd }

// renameUnprotected aplica rename fora dos trechos protegidos e remove as marcas.
func renameUnprotected(text string, rename func(string) string) string {
	var b strings.Builder
	for {
		start := strings.Index(text, protectStart)
		if start < 0 {
			b.WriteString(rename(text))
			return b.String()
		}
		b.WriteString(rename(text[:start]))
		text = text[start+len(protectStart):]
		end := strings.Index(text, protectEnd)
		if end < 0 {
			b.WriteString(text)
			return b.String()
		}
		b.WriteString(text[:end])
		text = text[end+len(protectEnd):]
	}
}

var typedFuncs = template.FuncMap{
	// protect marca um texto calculado no template para não passar pela troca de nomes.
	"protect": protect,
	// dict monta um mapa para passar vários valores a um {{template}}.
	"dict": func(pairs ...any) (map[string]any, error) {
		if len(pairs)%2 != 0 {
			return nil, fmt.Errorf("dict: número ímpar de argumentos")
		}
		values := map[string]any{}
		for index := 0; index < len(pairs); index += 2 {
			key, ok := pairs[index].(string)
			if !ok {
				return nil, fmt.Errorf("dict: chave %v não é string", pairs[index])
			}
			values[key] = pairs[index+1]
		}
		return values, nil
	},
}
