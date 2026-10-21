package generator

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"crudding/internal/generator/common"
	"crudding/internal/generator/custom"
	"crudding/internal/generator/langs/golang"
	"crudding/internal/generator/langs/node"
	"crudding/internal/generator/langs/python"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

// Language descreve uma linguagem suportada. É a fonte única das opções
// exibidas no menu, da validação e da matriz de testes.
type Language struct {
	Name          string   // valor de project.Config.Language
	Label         string   // nome exibido em mensagens de erro
	Runtimes      []string // vazio quando a linguagem não pergunta runtime
	Frameworks    []string
	Architectures []string
	ORMs          []string
}

var languages = []Language{
	{Name: "python", Label: "Python", Frameworks: python.Frameworks, Architectures: python.Architectures, ORMs: python.ORMs},
	{Name: "go", Label: "Go", Frameworks: golang.Frameworks, Architectures: golang.Architectures, ORMs: golang.ORMs},
	{Name: "node", Label: "Node", Runtimes: node.Runtimes, Frameworks: node.Frameworks, Architectures: node.Architectures, ORMs: node.ORMs},
}

// Languages devolve as linguagens suportadas, na ordem de exibição do menu.
func Languages() []Language {
	return slices.Clone(languages)
}

func Lookup(name string) (Language, bool) {
	index := slices.IndexFunc(languages, func(language Language) bool { return language.Name == name })
	if index < 0 {
		return Language{}, false
	}
	return languages[index], true
}

// ValidateORM verifica se o ORM informado é compatível com a linguagem e o banco.
func ValidateORM(language Language, db, orm string) error {
	if orm == "" || orm == "none" || orm == "nenhum" {
		return nil
	}
	if db == "" {
		return fmt.Errorf("ORM %q informado, mas nenhum banco de dados foi selecionado", orm)
	}
	if db == "mongo" {
		return fmt.Errorf("ORM %q não é suportado com banco mongo", orm)
	}
	if !slices.Contains(language.ORMs, orm) {
		return fmt.Errorf("ORM %s não suportado: %s (opções: %s)", language.Label, orm, strings.Join(language.ORMs, ", "))
	}
	return nil
}

func (l Language) validate(config project.Config) error {
	if len(l.Runtimes) > 0 && !slices.Contains(l.Runtimes, config.Runtime) {
		return fmt.Errorf("runtime %s não suportado: %s", l.Label, config.Runtime)
	}
	if !slices.Contains(l.Frameworks, config.Framework) {
		return fmt.Errorf("framework %s não suportado: %s", l.Label, config.Framework)
	}
	if !slices.Contains(l.Architectures, config.Architecture) {
		return fmt.Errorf("arquitetura %s não suportada: %s", l.Label, config.Architecture)
	}
	if err := ValidateORM(l, config.Database, config.ORM); err != nil {
		return err
	}
	return nil
}

// TemplatePath devolve o caminho relativo do template correspondente à configuração
// sob o diretório templates/.
func TemplatePath(config project.Config) string {
	return filepath.ToSlash(filepath.Join("templates", "langs", config.Language, "architectures", config.Architecture, "project.crud.tmpl"))
}

type TemplateData struct {
	common.BaseData
	GoModule     string
	Package      string
	Runtime      string
	EntryScript  string
	EntryModule  string
	Name         string
	Class        string
	Route        string
	VarName      string
	Modules      map[string]string
	Dependencies []string
}

func buildTemplateData(config project.Config) TemplateData {
	data := TemplateData{
		BaseData: common.NewBaseData(config),
		Runtime:  config.Runtime,
	}

	switch config.Language {
	case "go":
		data.GoModule = "example.com/" + project.Slug(config.ProjectName)
		data.Package = "user"
		data.VarName = "user"
		data.Class = "User"
	case "node":
		var nodeEntry = map[string]string{
			"flat":       "src/main",
			"layered":    "src/main",
			"modular":    "src/main",
			"hexagonal":  "src/entrypoints/main",
			"clean_arch": "src/frameworks/main",
		}
		data.EntryScript = nodeEntry[config.Architecture] + "." + config.Runtime
	case "python":
		data.Package = strings.ReplaceAll(strings.ToLower(config.ProjectName), "-", "_")
		data.Name = "resource"
		data.Class = "Resource"
		data.Route = "resource"
		pkgDir := "src/app"
		if config.Architecture == "flat" {
			pkgDir = "src"
		}
		data.EntryScript = path.Join(pkgDir, "main.py")
		data.EntryModule = strings.ReplaceAll(strings.TrimSuffix(data.EntryScript, ".py"), "/", ".")
		data.Modules = pythonModules(pkgDir, config.Architecture)
	}

	return data
}

func pythonModules(pkgDir, architecture string) map[string]string {
	mod := func(file string) string {
		return strings.ReplaceAll(strings.TrimSuffix(file, ".py"), "/", ".")
	}
	m := map[string]string{
		"config": mod(path.Join(pkgDir, "config", "config.py")),
	}
	switch architecture {
	case "flat":
		m["controller"] = mod(path.Join(pkgDir, "resource_controller.py"))
		m["service"] = mod(path.Join(pkgDir, "resource_service.py"))
		m["model"] = mod(path.Join(pkgDir, "resource_model.py"))
	case "layered":
		m["controller"] = mod(path.Join(pkgDir, "controllers", "resource_controller.py"))
		m["service"] = mod(path.Join(pkgDir, "services", "resource_service.py"))
		m["model"] = mod(path.Join(pkgDir, "models", "resource.py"))
	case "modular":
		m["controller"] = mod(path.Join(pkgDir, "resource", "controller.py"))
		m["service"] = mod(path.Join(pkgDir, "resource", "service.py"))
		m["model"] = mod(path.Join(pkgDir, "resource", "models.py"))
	case "hexagonal":
		m["controller"] = mod(path.Join(pkgDir, "routers", "resource_controller.py"))
		m["service"] = mod(path.Join(pkgDir, "application", "resource_service.py"))
		m["port"] = mod(path.Join(pkgDir, "ports", "resource_repository.py"))
		m["adapter"] = mod(path.Join(pkgDir, "infrastructure", "resource_repository.py"))
		m["model"] = mod(path.Join(pkgDir, "domain", "resource.py"))
	case "clean_arch":
		m["controller"] = mod(path.Join(pkgDir, "frameworks_and_drivers", "web", "resource_controller.py"))
		m["service"] = mod(path.Join(pkgDir, "interface_adapters", "controllers", "resource_controller.py"))
		m["presenter"] = mod(path.Join(pkgDir, "interface_adapters", "presenters", "resource_presenter.py"))
		m["create"] = mod(path.Join(pkgDir, "application", "use_cases", "create_resource.py"))
		m["delete"] = mod(path.Join(pkgDir, "application", "use_cases", "delete_resource.py"))
		m["get"] = mod(path.Join(pkgDir, "application", "use_cases", "get_resource.py"))
		m["list"] = mod(path.Join(pkgDir, "application", "use_cases", "list_resource.py"))
		m["update"] = mod(path.Join(pkgDir, "application", "use_cases", "update_resource.py"))
		m["port"] = mod(path.Join(pkgDir, "application", "port", "outbound", "resource_repository.py"))
		m["adapter"] = mod(path.Join(pkgDir, "interface_adapters", "persistence", "resource_repository.py"))
		m["model"] = mod(path.Join(pkgDir, "domain", "model", "resource.py"))
	}
	return m
}

func Generate(destination string, config project.Config, templates fs.FS) error {
	language, ok := Lookup(config.Language)
	if !ok {
		return fmt.Errorf("linguagem não suportada: %s", config.Language)
	}
	if err := language.validate(config); err != nil {
		return err
	}

	langFS, err := fs.Sub(templates, path.Join("templates", "langs", config.Language))
	if err != nil {
		return fmt.Errorf("acessar templates da linguagem %s: %w", config.Language, err)
	}

	manifestTmplPath := path.Join("architectures", config.Architecture, "project.crud.tmpl")
	manifestTmplBytes, err := fs.ReadFile(langFS, manifestTmplPath)
	if err != nil {
		return fmt.Errorf("ler template de manifesto %s: %w", manifestTmplPath, err)
	}

	data := buildTemplateData(config)

	parsedTmpl, err := template.New("manifest").Parse(string(manifestTmplBytes))
	if err != nil {
		return fmt.Errorf("interpretar manifesto %s: %w", manifestTmplPath, err)
	}
	var renderedManifest bytes.Buffer
	if err := parsedTmpl.Execute(&renderedManifest, data); err != nil {
		return fmt.Errorf("renderizar manifesto %s: %w", manifestTmplPath, err)
	}

	m, err := manifest.Parse(renderedManifest.Bytes(), manifestTmplPath)
	if err != nil {
		return fmt.Errorf("decodificar manifesto %s: %w", manifestTmplPath, err)
	}
	if len(m.Endpoints) == 0 && m.Endpoint.Name != "" {
		m.Endpoints = []string{m.Endpoint.Name}
	}
	if config.Runtime != "" {
		m.Runtime = config.Runtime
	}

	plan, err := custom.Plan(langFS, m, config.ProjectName, data)
	if err != nil {
		return fmt.Errorf("montar projeto %s: %w", config.ProjectName, err)
	}
	if plan.Manifest != nil {
		plan.Manifest.Template = ""
		plan.Manifest.ORM = config.ORM
		if len(plan.Manifest.Endpoints) == 0 && plan.Manifest.Endpoint.Name != "" {
			plan.Manifest.Endpoints = []string{plan.Manifest.Endpoint.Name}
		}
		if config.Runtime != "" {
			plan.Manifest.Runtime = config.Runtime
		}
	}
	if err := common.Scaffold(destination, config, langFS, plan); err != nil {
		return err
	}
	if config.Database != "" {
		currentManifest, err := manifest.Read(filepath.Join(destination, manifest.FileName))
		if err != nil {
			return err
		}
		currentManifest.ORM = config.ORM
		if _, err := AddDatabase(destination, currentManifest, config.Database, templates); err != nil {
			return fmt.Errorf("adicionar banco de dados %s: %w", config.Database, err)
		}
	}
	return nil
}

// GenerateFromTemplate gera em destination o projeto projectName copiando a
// estrutura listada no project.crud de templateDir (criada por "crudding
// init"), sem passar pelo registry de linguagens.
func GenerateFromTemplate(destination, projectName, templateDir string) error {
	templateRoot, err := filepath.Abs(templateDir)
	if err != nil {
		return fmt.Errorf("resolver diretório do template: %w", err)
	}
	destinationRoot, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolver diretório do projeto: %w", err)
	}
	if rel, err := filepath.Rel(templateRoot, destinationRoot); err == nil && filepath.IsLocal(rel) {
		return fmt.Errorf("o projeto não pode ser gerado dentro da pasta do template (%s)", templateRoot)
	}

	templates := os.DirFS(templateRoot)
	m, err := manifest.ReadFS(templates)
	if err != nil {
		return fmt.Errorf("template %s: %w", templateDir, err)
	}
	plan, err := custom.Plan(templates, m, projectName)
	if err != nil {
		return fmt.Errorf("template %s: %w", templateDir, err)
	}
	config := m.Config()
	config.ProjectName = projectName
	return common.Scaffold(destinationRoot, config, templates, plan)
}
