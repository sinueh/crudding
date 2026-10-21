package golang

import (
	"fmt"
	"path"

	"crudding/internal/generator/common"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

var Frameworks = []string{"raw", "gin", "fiber", "chi"}

var Architectures = []string{"flat", "layered", "modular", "hexagonal", "clean_arch"}

var ORMs = []string{"gorm"}

type templateData struct {
	common.BaseData
	GoModule string
	Package  string
	Class    string
	VarName  string
}

// sampleName é o endpoint de exemplo gerado em todo projeto Go. O campo
// endpoint do project.crud aponta para ele, e o "crudding add endpoint" copia
// seus arquivos.
const sampleName = "user"

// sampleFile é um arquivo do endpoint de exemplo. O template fica em
// base/common, ou no framework quando framework é true.
type sampleFile struct {
	template  string
	output    string
	framework bool
}

type architecture struct {
	main   string
	routes string // arquivo de rotas, onde o exemplo é registrado
	sample []sampleFile
}

var architectures = map[string]architecture{
	"flat": {main: "main.go", routes: "routes.go", sample: []sampleFile{
		{template: "model.go", output: "user_model.go"},
		{template: "types.go", output: "user_types.go"},
		{template: "service.go", output: "user_service.go"},
		{template: "handler.go", output: "user_handler.go", framework: true},
	}},
	"layered": {main: "cmd/api/main.go", routes: "internal/routes/routes.go", sample: []sampleFile{
		{template: "model.go", output: "internal/models/user.go"},
		{template: "types.go", output: "internal/types/user.go"},
		{template: "service.go", output: "internal/service/user.go"},
		{template: "handler.go", output: "internal/handlers/user.go", framework: true},
	}},
	"modular": {main: "cmd/api/main.go", routes: "internal/routes/routes.go", sample: []sampleFile{
		{template: "model.go", output: "internal/user/model.go"},
		{template: "types.go", output: "internal/user/types.go"},
		{template: "service.go", output: "internal/user/service.go"},
		{template: "handler.go", output: "internal/user/handler.go", framework: true},
	}},
	"hexagonal": {main: "cmd/api/main.go", routes: "internal/routes/routes.go", sample: []sampleFile{
		{template: "model.go", output: "internal/domain/user.go"},
		{template: "types.go", output: "internal/domain/user_types.go"},
		{template: "ports_repository.go", output: "internal/ports/user_repository.go"},
		{template: "ports_service.go", output: "internal/ports/user_service.go"},
		{template: "service.go", output: "internal/application/user.go"},
		{template: "repository.go", output: "internal/adapters/postgres/user.go"},
		{template: "handler.go", output: "internal/handlers/user.go", framework: true},
	}},
	"clean_arch": {main: "cmd/api/main.go", routes: "internal/routes/routes.go", sample: []sampleFile{
		{template: "model.go", output: "internal/domain/user.go"},
		{template: "types.go", output: "internal/domain/user_types.go"},
		{template: "ports_repository.go", output: "internal/ports/user_repository.go"},
		{template: "ports_service.go", output: "internal/ports/user_service.go"},
		{template: "service.go", output: "internal/application/user.go"},
		{template: "repository.go", output: "internal/infrastructure/persistence/user.go"},
		{template: "handler.go", output: "internal/handlers/user.go", framework: true},
	}},
}

// Plan monta o projeto Go. Framework e arquitetura já foram validados pelo
// registry em internal/generator.
func Plan(_ string, config project.Config) (common.Plan, error) {
	arch, ok := architectures[config.Architecture]
	if !ok {
		return common.Plan{}, fmt.Errorf("arquitetura Go não suportada: %s", config.Architecture)
	}
	archDir := path.Join("templates/langs/go/base", config.Architecture)
	fwDir := path.Join("templates/langs/go/frameworks", config.Framework)
	commonDir := "templates/langs/go/base/common"

	configOutput := "internal/config/config.go"
	if config.Architecture == "flat" {
		configOutput = "config/config.go"
	}

	files := []common.File{
		{TemplatePath: path.Join(commonDir, "go.mod.tmpl"), OutputPath: "go.mod"},
		{TemplatePath: path.Join(commonDir, "go.sum.tmpl"), OutputPath: "go.sum"},
		{TemplatePath: path.Join(commonDir, ".gitignore.tmpl"), OutputPath: ".gitignore"},
		{TemplatePath: path.Join(archDir, "README.md.tmpl"), OutputPath: "README.md"},
		{TemplatePath: path.Join(commonDir, "config.go.tmpl"), OutputPath: configOutput},
		{TemplatePath: path.Join(fwDir, "main.go.tmpl"), OutputPath: arch.main},
		{TemplatePath: path.Join(fwDir, "routes.go.tmpl"), OutputPath: arch.routes},
	}
	if config.Database != "" {
		files = append(files, common.DatabaseFiles(config.Database)...)
	}

	endpoint := manifest.EndpointSpec{Name: sampleName, Register: []string{arch.routes}}
	for _, file := range arch.sample {
		templateDir := commonDir
		if file.framework {
			templateDir = fwDir
		}
		files = append(files, common.File{TemplatePath: path.Join(templateDir, file.template+".tmpl"), OutputPath: file.output})
		endpoint.Files = append(endpoint.Files, file.output)
	}

	goModule := "example.com/" + project.Slug(config.ProjectName)
	data := templateData{
		BaseData: common.NewBaseData(config),
		GoModule: goModule,
		Package:  sampleName,
		Class:    "User",
		VarName:  sampleName,
	}

	return common.Plan{Files: files, Data: data, Endpoints: []string{sampleName}, Endpoint: endpoint}, nil
}
