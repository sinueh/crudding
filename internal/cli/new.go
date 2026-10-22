package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh/crudding"
	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/project"
	"github.com/sinueh/crudding/internal/tui"
)

func choose(title string, options []string) (string, error) {
	choice, confirmed, err := tui.ChooseOption(title, options)
	if err != nil {
		return "", err
	}
	if !confirmed {
		return "", errCancelled
	}
	return choice, nil
}

func chooseLanguage() (generator.Language, error) {
	languages := generator.Languages()
	names := make([]string, len(languages))
	for index, language := range languages {
		names[index] = language.Name
	}
	name, err := choose("escolha a linguagem", names)
	if err != nil {
		return generator.Language{}, err
	}
	language, _ := generator.Lookup(name)
	return language, nil
}

func chooseRuntime(language generator.Language) (string, error) {
	if len(language.Runtimes) == 0 {
		return "", nil
	}
	return choose("escolha o runtime", language.Runtimes)
}

func promptConfig(stdout io.Writer) (project.Config, error) {
	language, err := chooseLanguage()
	if err != nil {
		return project.Config{}, err
	}
	runtime, err := chooseRuntime(language)
	if err != nil {
		return project.Config{}, err
	}
	framework, err := choose("escolha o framework", language.Frameworks)
	if err != nil {
		return project.Config{}, err
	}
	architecture, err := choose("escolha a arquitetura", language.Architectures)
	if err != nil {
		return project.Config{}, err
	}
	database, err := chooseDatabase(language)
	if err != nil {
		return project.Config{}, err
	}
	orm, err := chooseORM(language, database)
	if err != nil {
		return project.Config{}, err
	}
	projectName, err := project.ReadProjectName(os.Stdin, stdout)
	if err != nil {
		return project.Config{}, err
	}
	return project.Config{
		ProjectName:  projectName,
		Language:     language.Name,
		Runtime:      runtime,
		Framework:    framework,
		Architecture: architecture,
		Database:     database,
		ORM:          orm,
	}, nil
}

func chooseDatabase(language generator.Language) (string, error) {
	if language.Name != "go" && language.Name != "python" {
		return "", nil
	}
	db, err := choose("escolha o banco de dados", []string{"nenhum", "postgres", "mysql", "mongo"})
	if err != nil {
		return "", err
	}
	if db == "nenhum" {
		return "", nil
	}
	return db, nil
}

func chooseORM(language generator.Language, database string) (string, error) {
	if database == "" || database == "mongo" || len(language.ORMs) == 0 {
		return "", nil
	}
	options := append([]string{"nenhum"}, language.ORMs...)
	choice, err := choose("escolha o ORM", options)
	if err != nil {
		return "", err
	}
	if choice == "nenhum" {
		return "", nil
	}
	return choice, nil
}

func uniqueValues(extract func(generator.Language) []string) []string {
	var result []string
	for _, lang := range generator.Languages() {
		for _, v := range extract(lang) {
			if !slices.Contains(result, v) {
				result = append(result, v)
			}
		}
	}
	return result
}

func optionsForKey(key, selectedLang string) (string, []string, bool) {
	switch key {
	case "lang", "l":
		return "lang/l", uniqueValues(func(l generator.Language) []string { return []string{l.Name} }), true
	case "runtime", "r":
		if lang, ok := generator.Lookup(selectedLang); ok && len(lang.Runtimes) > 0 {
			return "runtime/r", lang.Runtimes, true
		}
		return "runtime/r", uniqueValues(func(l generator.Language) []string { return l.Runtimes }), true
	case "framework", "f":
		if lang, ok := generator.Lookup(selectedLang); ok {
			return "framework/f", lang.Frameworks, true
		}
		return "framework/f", uniqueValues(func(l generator.Language) []string { return l.Frameworks }), true
	case "arch", "a":
		if lang, ok := generator.Lookup(selectedLang); ok {
			return "arch/a", lang.Architectures, true
		}
		return "arch/a", uniqueValues(func(l generator.Language) []string { return l.Architectures }), true
	case "db", "d", "database":
		return "db/d", []string{"none", "postgres", "mysql", "mongo"}, true
	case "orm", "o":
		if lang, ok := generator.Lookup(selectedLang); ok {
			return "orm/o", append([]string{"none"}, lang.ORMs...), true
		}
		return "orm/o", append([]string{"none"}, uniqueValues(func(l generator.Language) []string { return l.ORMs })...), true
	default:
		return "", nil, false
	}
}

func formatOptionBlock(label string, values []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", label)
	for _, v := range values {
		fmt.Fprintf(&b, "  %s\n", v)
	}
	return b.String()
}

// NewUsage gera o texto de ajuda detalhado do comando "crudding new".
func NewUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Gera a estrutura de um novo projeto CRUD no diretório atual.\n")
	b.WriteString("  - Para o menu interativo, execute apenas \"crudding\".\n")
	b.WriteString("  - Com argumentos no formato chave:valor (ou abreviação c:valor), cria o projeto\n")
	b.WriteString("    diretamente em modo não-interativo.\n")
	b.WriteString("  - Passando apenas o nome ou a primeira letra da flag (ex.: \"crudding new l\" ou\n")
	b.WriteString("    \"crudding new f\"), lista as opções disponíveis para aquela flag (exceto name/n).\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding new name:<nome> lang:<linguagem> framework:<framework> arch:<arquitetura> [runtime:<runtime>] [db:<banco>] [orm:<orm>]\n")
	b.WriteString("  crudding new n:<nome> l:<linguagem> f:<framework> a:<arquitetura> [r:<runtime>] [d:<banco>] [o:<orm>]\n\n")
	b.WriteString("como cada flag funciona:\n")
	b.WriteString("  name:<nome> ou n:<nome>               define o nome do projeto (pasta que será criada)\n")
	b.WriteString("  lang:<linguagem> ou l:<linguagem>     define a linguagem do projeto\n")
	b.WriteString("  framework:<fw> ou f:<fw>              define o framework HTTP utilizado\n")
	b.WriteString("  arch:<arquitetura> ou a:<arquitetura> define o padrão arquitetural de pastas\n")
	b.WriteString("  runtime:<runtime> ou r:<runtime>      define o runtime (obrigatório para node)\n")
	b.WriteString("  db:<banco> ou d:<banco>               define o banco de dados (opcional: postgres, mysql, mongo, none)\n")
	b.WriteString("  orm:<orm> ou o:<orm>                  define o ORM utilizado (opcional: none, gorm, sqlalchemy, etc.)\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding new name:meu-app lang:go framework:gin arch:hexagonal\n")
	b.WriteString("  crudding new n:meu-app l:go f:gin a:hexagonal\n")
	b.WriteString("  crudding new n:api-node l:node r:ts f:express a:layered\n")
	b.WriteString("  crudding new l\n")
	b.WriteString("  crudding new l:go f\n\n")
	b.WriteString("opções disponíveis:\n\n")
	for _, key := range []string{"lang", "runtime", "framework", "arch", "db", "orm"} {
		label, values, _ := optionsForKey(key, "")
		b.WriteString(formatOptionBlock(label, values))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func parseNewArgs(args []string) (project.Config, string, error) {
	fs := flag.NewFlagSet("crudding new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return project.Config{}, NewUsage(), nil
		}
		return project.Config{}, "", err
	}

	tokens := fs.Args()
	if len(tokens) == 1 && tokens[0] == "help" {
		return project.Config{}, NewUsage(), nil
	}

	var (
		config      project.Config
		inspectKeys []string
	)
	for _, token := range tokens {
		key, value, hasColon := strings.Cut(token, ":")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "name", "n":
			if !hasColon || value == "" {
				return project.Config{}, "", fmt.Errorf("nenhum nome de projeto informado")
			}
			config.ProjectName = value
		case "lang", "l":
			if !hasColon || value == "" {
				inspectKeys = append(inspectKeys, "lang")
				continue
			}
			config.Language = value
		case "runtime", "r":
			if !hasColon || value == "" {
				inspectKeys = append(inspectKeys, "runtime")
				continue
			}
			config.Runtime = value
		case "framework", "f":
			if !hasColon || value == "" {
				inspectKeys = append(inspectKeys, "framework")
				continue
			}
			config.Framework = value
		case "arch", "a":
			if !hasColon || value == "" {
				inspectKeys = append(inspectKeys, "arch")
				continue
			}
			config.Architecture = value
		case "db", "d", "database":
			if !hasColon || value == "" {
				inspectKeys = append(inspectKeys, "db")
				continue
			}
			if value == "none" || value == "nenhum" {
				config.Database = ""
			} else {
				if err := generator.ValidateDatabase(value); err != nil {
					return project.Config{}, "", err
				}
				config.Database = value
			}
		case "orm", "o":
			if !hasColon || value == "" {
				inspectKeys = append(inspectKeys, "orm")
				continue
			}
			if value == "none" || value == "nenhum" {
				config.ORM = ""
			} else {
				config.ORM = value
			}
		default:
			return project.Config{}, "", fmt.Errorf("flag desconhecida: %s", token)
		}
	}

	if len(inspectKeys) > 0 {
		var b strings.Builder
		for i, key := range inspectKeys {
			if i > 0 {
				b.WriteString("\n")
			}
			label, values, _ := optionsForKey(key, config.Language)
			b.WriteString(formatOptionBlock(label, values))
		}
		return project.Config{}, b.String(), nil
	}

	var missing []string
	if config.ProjectName == "" {
		missing = append(missing, "name/n")
	}
	if config.Language == "" {
		missing = append(missing, "lang/l")
	}
	if config.Framework == "" {
		missing = append(missing, "framework/f")
	}
	if config.Architecture == "" {
		missing = append(missing, "arch/a")
	}
	if lang, ok := generator.Lookup(config.Language); ok {
		if len(lang.Runtimes) > 0 && config.Runtime == "" {
			missing = append(missing, "runtime/r")
		} else if len(lang.Runtimes) == 0 {
			config.Runtime = ""
		}
	}
	if len(missing) > 0 {
		return project.Config{}, "", fmt.Errorf("flags obrigatórias ausentes: %s", strings.Join(missing, ", "))
	}

	if err := project.ValidateProjectName(config.ProjectName); err != nil {
		return project.Config{}, "", err
	}
	return config, "", nil
}

func formatLanguageLine(config project.Config) string {
	if config.Runtime == "" {
		return fmt.Sprintf("Linguagem: %s", config.Language)
	}
	return fmt.Sprintf("Linguagem/motor: %s/%s", config.Language, config.Runtime)
}

func printNewSummary(stdout io.Writer, projectDirectory string, config project.Config) {
	fmt.Fprintf(stdout, "\nProjeto gerado em: %s\n", projectDirectory)
	fmt.Fprintln(stdout, formatLanguageLine(config))
	fmt.Fprintf(stdout, "Framework: %s\nArquitetura: %s\n", config.Framework, config.Architecture)
	if config.Database != "" {
		fmt.Fprintf(stdout, "Banco de dados: %s\n", config.Database)
	}
	if config.ORM != "" {
		fmt.Fprintf(stdout, "ORM: %s\n", config.ORM)
	}
}

func generateProject(config project.Config, stdout io.Writer) error {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	projectDirectory := filepath.Join(workingDirectory, config.ProjectName)
	if err := generator.Generate(projectDirectory, config, crudding.Templates); err != nil {
		return err
	}
	printNewSummary(stdout, projectDirectory, config)
	return nil
}

func runInteractive(_ []string, stdout, _ io.Writer) error {
	config, err := promptConfig(stdout)
	if err != nil {
		return err
	}
	return generateProject(config, stdout)
}

func runNew(args []string, stdout, _ io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("nenhum argumento informado; execute \"crudding new -h\" para ver o uso")
	}
	config, info, err := parseNewArgs(args)
	if err != nil {
		return err
	}
	if info != "" {
		fmt.Fprint(stdout, info)
		return nil
	}
	return generateProject(config, stdout)
}

