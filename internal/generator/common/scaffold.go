package common

import (
	"io/fs"
	"os"

	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

// Plan descreve tudo o que um gerador de linguagem quer produzir. Os geradores
// apenas montam o Plan; criar o diretório, renderizar e limpar em caso de
// falha fica a cargo de Scaffold.
type Plan struct {
	Files     []File
	Data      any
	Inserts   []Insert
	Endpoints []string
	// Prepare roda depois da criação do diretório raiz e antes da escrita dos
	// arquivos (ex.: criar pacotes Python com __init__.py). Opcional.
	Prepare func(root string) error
	// Endpoint preenche o campo endpoint do project.crud: o endpoint de
	// exemplo gerado, que o "crudding add endpoint" copia.
	Endpoint manifest.EndpointSpec
	// Manifest, se definido, é gravado no lugar do manifesto montado a partir
	// do config (ex.: projetos gerados de um template herdam o dele).
	Manifest *manifest.Manifest
}

// Scaffold cria o diretório do projeto (nunca sobrescreve um existente),
// executa o Plan, grava o project.crud com a estrutura gerada e remove tudo o
// que foi gerado se algum passo falhar.
func Scaffold(destination string, config project.Config, templates fs.FS, plan Plan) (err error) {
	root, err := CreateProjectDirectory(destination)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(root)
		}
	}()

	if plan.Prepare != nil {
		if err = plan.Prepare(root); err != nil {
			return err
		}
	}
	if err = WriteFiles(root, templates, plan.Data, plan.Files); err != nil {
		return err
	}
	if len(plan.Inserts) > 0 {
		if _, err = Apply(root, templates, Patch{Inserts: plan.Inserts}); err != nil {
			return err
		}
	}

	// O manifesto é gravado por último para listar a estrutura gerada.
	m := manifest.New(config, plan.Endpoints...)
	m.Endpoint = plan.Endpoint
	if plan.Manifest != nil {
		m = *plan.Manifest
	}
	if m.Directories, m.Files, err = ListStructure(os.DirFS(root), m.Ignore); err != nil {
		return err
	}
	return manifest.Write(root, m)
}

// BaseData reúne os campos de template comuns a todas as linguagens. Deve ser
// embutido nos templateData específicos para que {{.ProjectName}} etc.
// continuem acessíveis diretamente.
type BaseData struct {
	ProjectName  string
	ProjectSlug  string
	Framework    string
	Architecture string
	Database     string
	ORM          string
}

func NewBaseData(config project.Config) BaseData {
	return BaseData{
		ProjectName:  config.ProjectName,
		ProjectSlug:  project.Slug(config.ProjectName),
		Framework:    config.Framework,
		Architecture: config.Architecture,
		Database:     config.Database,
		ORM:          config.ORM,
	}
}

func (b BaseData) HasDatabase(name string) bool {
	return b.Database == name
}

func (b BaseData) HasAnyDatabase() bool {
	return b.Database != ""
}

func (b BaseData) HasORM(name ...string) bool {
	if len(name) == 0 {
		return b.ORM != "" && b.ORM != "none" && b.ORM != "nenhum"
	}
	return b.ORM == name[0]
}
