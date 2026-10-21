package python

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"crudding/internal/generator/common"
	"crudding/internal/project"
)

var Frameworks = []string{"raw", "fastapi", "django", "flask"}

var Architectures = []string{"flat", "layered", "modular", "hexagonal", "clean_arch"}

var ORMs = []string{"sqlalchemy"}

var FrameworkDependencies = map[string][]string{
	"fastapi": {"fastapi>=0.115", "uvicorn[standard]>=0.30"},
	"django":  {"Django>=5.1"},
	"flask":   {"Flask>=3.0"},
	"raw":     {},
}

type templateData struct {
	common.BaseData
	Dependencies []string
	EntryModule  string
	EntryScript  string
	Name         string
	Class        string
	Route        string
	Modules      map[string]string
}

type architecture struct {
	pkgDir string
	main   string
}

var architectures = map[string]architecture{
	"flat":       {pkgDir: "src", main: "main.py"},
	"layered":    {pkgDir: "src/app", main: "main.py"},
	"modular":    {pkgDir: "src/app", main: "main.py"},
	"hexagonal":  {pkgDir: "src/app", main: "main.py"},
	"clean_arch": {pkgDir: "src/app", main: "main.py"},
}

// Plan monta o projeto Python. Framework e arquitetura já foram validados pelo
// registry em internal/generator.
func Plan(destination string, config project.Config) (common.Plan, error) {
	arch, ok := architectures[config.Architecture]
	if !ok {
		return common.Plan{}, fmt.Errorf("arquitetura Python não suportada: %s", config.Architecture)
	}

	pkgDir := arch.pkgDir
	entryScript := path.Join(pkgDir, arch.main)

	archDir := path.Join("templates/langs/python/base", config.Architecture)
	fwDir := path.Join("templates/langs/python/frameworks", config.Framework)
	commonDir := "templates/langs/python/base/common"

	configOutput := path.Join(pkgDir, "config", "config.py")

	files := []common.File{
		{TemplatePath: path.Join(archDir, "README.md.tmpl"), OutputPath: "README.md"},
		{TemplatePath: path.Join(commonDir, "requirements.txt.tmpl"), OutputPath: "requirements.txt"},
		{TemplatePath: path.Join(commonDir, ".gitignore.tmpl"), OutputPath: ".gitignore"},
		{TemplatePath: path.Join(commonDir, "config.py.tmpl"), OutputPath: configOutput},
		{TemplatePath: path.Join(fwDir, "main.py.tmpl"), OutputPath: entryScript},
	}

	if config.Database != "" {
		files = append(files, common.DatabaseFiles(config.Database)...)
	}

	packages := []string{pkgDir, path.Join(pkgDir, "config")}

	switch config.Architecture {
	case "flat":
		files = append(files,
			common.File{TemplatePath: path.Join(archDir, "setup.py.tmpl"), OutputPath: "setup.py"},
		)
		packages = append(packages, "tests")
	case "layered":
		packages = append(packages,
			path.Join(pkgDir, "controllers"),
			path.Join(pkgDir, "services"),
			path.Join(pkgDir, "models"),
		)
	case "modular":
		packages = append(packages,
			path.Join(pkgDir, sampleName),
		)
	case "hexagonal":
		packages = append(packages,
			path.Join(pkgDir, "domain"),
			path.Join(pkgDir, "application"),
			path.Join(pkgDir, "ports"),
			path.Join(pkgDir, "infrastructure"),
			path.Join(pkgDir, "routers"),
		)
	case "clean_arch":
		packages = append(packages,
			path.Join(pkgDir, "domain/model"),
			path.Join(pkgDir, "application/use_cases"),
			path.Join(pkgDir, "application/port/outbound"),
			path.Join(pkgDir, "interface_adapters/controllers"),
			path.Join(pkgDir, "interface_adapters/presenters"),
			path.Join(pkgDir, "interface_adapters/persistence"),
			path.Join(pkgDir, "frameworks_and_drivers/web"),
		)
	}

	sample, err := newSample(pkgDir, config)
	if err != nil {
		return common.Plan{}, err
	}
	files = append(files, sample.files...)
	packages = append(packages, sample.packages...)

	data := sample.data
	deps := slices.Clone(FrameworkDependencies[config.Framework])
	if config.Database != "" {
		switch config.Database {
		case "postgres":
			deps = append(deps, "psycopg2-binary>=2.9.9")
		case "mysql":
			deps = append(deps, "pymysql>=1.1.0")
		case "mongo":
			deps = append(deps, "pymongo>=4.6.0")
		}
	}
	if config.ORM == "sqlalchemy" {
		deps = append(deps, "SQLAlchemy>=2.0.0")
	}
	data.Dependencies = deps
	data.EntryScript = entryScript
	data.EntryModule = module(entryScript)
	data.ProjectName = strings.ReplaceAll(filepath.Base(destination), "_", "-")

	endpoint := sample.endpoint
	endpoint.Register = []string{entryScript}
	return common.Plan{
		Files:     files,
		Data:      data,
		Endpoints: []string{sampleName},
		Endpoint:  endpoint,
		Prepare:   func(root string) error { return common.CreatePackages(root, packages, nil) },
	}, nil
}

func module(file string) string {
	return strings.ReplaceAll(strings.TrimSuffix(file, ".py"), "/", ".")
}
