package python

import (
	"fmt"
	"path"
	"strings"

	"crudding/internal/generator/common"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

// sampleName é o endpoint de exemplo gerado em todo projeto Python. O campo
// endpoint do project.crud aponta para ele, e o "crudding add endpoint" copia
// seus arquivos.
const sampleName = "resource"

const nameToken = "{name}"

type sampleFile struct {
	template     string // relativo à pasta templates/python/, ignorado se isController for true
	output       string // relativo ao pacote base, com {name}
	moduleKey    string // chave opcional usada em {{.Modules.<key>}}
	isController bool   // se true, o template vem de templates/python/frameworks/<framework>/controller.py.tmpl
}

type sampleLayout struct {
	controllerOutput string // caminho do controller do framework relativo ao pacote base
	packages         []string
	files            []sampleFile
}

var sampleLayouts = map[string]sampleLayout{
	"flat": {
		controllerOutput: "{name}_controller.py",
		files: []sampleFile{
			{isController: true, output: "{name}_controller.py"},
			{template: "base/common/service.py.tmpl", output: "{name}_service.py", moduleKey: "service"},
			{template: "base/common/model.py.tmpl", output: "{name}_model.py", moduleKey: "model"},
		},
	},
	"layered": {
		controllerOutput: "controllers/{name}_controller.py",
		files: []sampleFile{
			{isController: true, output: "controllers/{name}_controller.py"},
			{template: "base/common/service.py.tmpl", output: "services/{name}_service.py", moduleKey: "service"},
			{template: "base/common/model.py.tmpl", output: "models/{name}.py", moduleKey: "model"},
		},
	},
	"modular": {
		controllerOutput: "{name}/controller.py",
		packages:         []string{"{name}"},
		files: []sampleFile{
			{isController: true, output: "{name}/controller.py"},
			{template: "base/common/service.py.tmpl", output: "{name}/service.py", moduleKey: "service"},
			{template: "base/common/model.py.tmpl", output: "{name}/models.py", moduleKey: "model"},
		},
	},
	"hexagonal": {
		controllerOutput: "routers/{name}_controller.py",
		files: []sampleFile{
			{isController: true, output: "routers/{name}_controller.py"},
			{template: "base/hexagonal/service.py.tmpl", output: "application/{name}_service.py", moduleKey: "service"},
			{template: "base/common/repository_port.py.tmpl", output: "ports/{name}_repository.py", moduleKey: "port"},
			{template: "base/common/repository_adapter.py.tmpl", output: "infrastructure/{name}_repository.py", moduleKey: "adapter"},
			{template: "base/common/model.py.tmpl", output: "domain/{name}.py", moduleKey: "model"},
		},
	},
	"clean_arch": {
		controllerOutput: "frameworks_and_drivers/web/{name}_controller.py",
		files: []sampleFile{
			{isController: true, output: "frameworks_and_drivers/web/{name}_controller.py"},
			{template: "base/clean_arch/controller.py.tmpl", output: "interface_adapters/controllers/{name}_controller.py", moduleKey: "service"},
			{template: "base/clean_arch/presenter.py.tmpl", output: "interface_adapters/presenters/{name}_presenter.py", moduleKey: "presenter"},
			{template: "base/clean_arch/create.py.tmpl", output: "application/use_cases/create_{name}.py", moduleKey: "create"},
			{template: "base/clean_arch/delete.py.tmpl", output: "application/use_cases/delete_{name}.py", moduleKey: "delete"},
			{template: "base/clean_arch/get.py.tmpl", output: "application/use_cases/get_{name}.py", moduleKey: "get"},
			{template: "base/clean_arch/list.py.tmpl", output: "application/use_cases/list_{name}.py", moduleKey: "list"},
			{template: "base/clean_arch/update.py.tmpl", output: "application/use_cases/update_{name}.py", moduleKey: "update"},
			{template: "base/common/repository_port.py.tmpl", output: "application/port/outbound/{name}_repository.py", moduleKey: "port"},
			{template: "base/common/repository_adapter.py.tmpl", output: "interface_adapters/persistence/{name}_repository.py", moduleKey: "adapter"},
			{template: "base/common/model.py.tmpl", output: "domain/model/{name}.py", moduleKey: "model"},
		},
	},
}

// sample é o endpoint de exemplo montado para um projeto: os arquivos, os
// pacotes, os dados dos templates e o campo endpoint do project.crud. O
// registro no main.py fica nos templates de cada framework.
type sample struct {
	files    []common.File
	packages []string
	data     templateData
	endpoint manifest.EndpointSpec
}

func newSample(pkgDir string, config project.Config) (sample, error) {
	layout, ok := sampleLayouts[config.Architecture]
	if !ok {
		return sample{}, fmt.Errorf("arquitetura Python não suportada: %s", config.Architecture)
	}
	expand := func(rel string) string { return strings.ReplaceAll(rel, nameToken, sampleName) }

	modules := map[string]string{
		"config":     module(path.Join(pkgDir, "config", "config.py")),
		"controller": module(path.Join(pkgDir, expand(layout.controllerOutput))),
	}
	for _, file := range layout.files {
		if file.moduleKey != "" {
			modules[file.moduleKey] = module(path.Join(pkgDir, expand(file.output)))
		}
	}

	s := sample{
		data: templateData{
			BaseData:     common.NewBaseData(config),
			Dependencies: FrameworkDependencies[config.Framework],
			Name:         sampleName,
			Class:        className(sampleName),
			Route:        strings.ReplaceAll(sampleName, "_", "-"),
			Modules:      modules,
		},
		endpoint: manifest.EndpointSpec{Name: sampleName},
	}
	for _, file := range layout.files {
		templatePath := path.Join("templates/langs/python", file.template)
		if file.isController {
			templatePath = fmt.Sprintf("templates/langs/python/frameworks/%s/controller.py.tmpl", config.Framework)
		}
		output := path.Join(pkgDir, expand(file.output))
		s.files = append(s.files, common.File{TemplatePath: templatePath, OutputPath: output})
		s.endpoint.Files = append(s.endpoint.Files, output)
	}
	for _, pkg := range layout.packages {
		directory := path.Join(pkgDir, expand(pkg))
		s.packages = append(s.packages, directory)
		s.endpoint.Files = append(s.endpoint.Files, path.Join(directory, "__init__.py"))
	}
	return s, nil
}

func className(name string) string {
	var class strings.Builder
	for _, part := range strings.Split(name, "_") {
		if part != "" {
			class.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return class.String()
}
