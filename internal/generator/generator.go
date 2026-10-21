package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	parts := []string{"templates", "langs", config.Language}
	if lang, ok := Lookup(config.Language); ok && len(lang.Runtimes) > 0 {
		parts = append(parts, config.Runtime)
	}
	parts = append(parts, config.Framework, config.Architecture)
	return filepath.ToSlash(filepath.Join(parts...))
}

func Generate(destination string, config project.Config, templates fs.FS) error {
	language, ok := Lookup(config.Language)
	if !ok {
		return fmt.Errorf("linguagem não suportada: %s", config.Language)
	}
	if err := language.validate(config); err != nil {
		return err
	}

	tmplRel := TemplatePath(config)
	templateFS, err := fs.Sub(templates, tmplRel)
	if err != nil {
		return fmt.Errorf("acessar template %s: %w", tmplRel, err)
	}
	m, err := manifest.ReadFS(templateFS)
	if err != nil {
		return fmt.Errorf("ler template %s: %w", tmplRel, err)
	}

	plan, err := custom.Plan(templateFS, m, config.ProjectName)
	if err != nil {
		return fmt.Errorf("montar projeto %s: %w", config.ProjectName, err)
	}
	if plan.Manifest != nil {
		plan.Manifest.Template = ""
		plan.Manifest.ORM = config.ORM
	}
	if err := common.Scaffold(destination, config, templateFS, plan); err != nil {
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
