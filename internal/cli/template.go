package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/generator/custom"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

// TemplateUsage gera o texto de ajuda do comando "crudding template".
func TemplateUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Salva um projeto como template em ~/.crudding/templates/<nome>.\n")
	b.WriteString("  Os templates salvos podem ser listados e usados para gerar novos projetos.\n\n")
	b.WriteString("comandos:\n")
	b.WriteString("  crudding template <pasta> [<nome> | name:<nome> | n:<nome>]\n")
	b.WriteString("    Salva o projeto de <pasta> como template. Sem o nome, solicita interativamente.\n")
	b.WriteString("  crudding template list\n")
	b.WriteString("    Lista todos os templates salvos em ~/.crudding/templates.\n")
	b.WriteString("  crudding template use <template> [<nome> | name:<nome> | n:<nome>]\n")
	b.WriteString("    Gera um novo projeto a partir de um template salvo.\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding template ./meu-projeto\n")
	b.WriteString("  crudding template ./meu-projeto meu-template\n")
	b.WriteString("  crudding template list\n")
	b.WriteString("  crudding template use meu-template minha-api\n")
	return b.String()
}

// TemplateListUsage gera o texto de ajuda do comando "crudding template list".
func TemplateListUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Lista todos os templates salvos na pasta ~/.crudding/templates.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding template list\n")
	return b.String()
}

// TemplateUseUsage gera o texto de ajuda do comando "crudding template use".
func TemplateUseUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Gera um projeto novo no diretório atual copiando a estrutura de um template\n")
	b.WriteString("  salvo em ~/.crudding/templates. O nome do template é trocado pelo nome do novo projeto.\n")
	b.WriteString("  - Sem o nome, solicita interativamente.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding template use <template> [<nome> | name:<nome> | n:<nome>]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding template use meu-template\n")
	b.WriteString("  crudding template use meu-template minha-api\n")
	return b.String()
}

func runTemplate(args []string, stdout, stderr io.Writer) error {
	// Cria/garante que a pasta .crudding/templates/ existe logo que o usuário digita "crudding template".
	_, err := custom.EnsureTemplatesDir()
	if err != nil {
		return err
	}

	if slices.ContainsFunc(args, func(token string) bool { return slices.Contains(helpFlags, token) }) {
		fmt.Fprint(stdout, TemplateUsage())
		return nil
	}

	if len(args) > 0 && args[0] == "list" {
		return runTemplateList(args[1:], stdout, stderr)
	}

	if len(args) > 0 && (args[0] == "use" || args[0] == "new") {
		return runTemplateUse(args[1:], stdout, stderr)
	}

	var dir, name string
	if len(args) == 0 {
		dir, err = project.ReadProjectPath(os.Stdin, stdout)
		if err != nil {
			return err
		}
	} else {
		dir = strings.TrimSpace(args[0])
		if len(args) > 2 {
			return fmt.Errorf("argumentos inesperados: %s", strings.Join(args[2:], " "))
		}
		if len(args) == 2 {
			name = strings.TrimSpace(args[1])
			if key, value, ok := strings.Cut(name, ":"); ok && (key == "name" || key == "n") {
				name = strings.TrimSpace(value)
			}
			if err := project.ValidateProjectName(name); err != nil {
				return err
			}
		}
	}

	// Se o caminho informado não existe localmente, mas existe como template salvo,
	// gera o projeto a partir do template (ex.: crudding template meu-template minha-api).
	if _, statErr := os.Stat(dir); statErr != nil {
		if _, _, getErr := custom.Get(dir); getErr == nil {
			return runTemplateUse(args, stdout, stderr)
		}
		return fmt.Errorf("diretório %s não encontrado", dir)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("diretório %s não encontrado", dir)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("%s não é um diretório", dir)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolver diretório %s: %w", dir, err)
	}

	defaultName := filepath.Base(absDir)
	if m, err := manifest.Read(filepath.Join(absDir, manifest.FileName)); err == nil && m.Name != "" {
		defaultName = m.Name
		unlisted, err := custom.Unlisted(os.DirFS(absDir), m)
		if err == nil && len(unlisted) > 0 {
			fmt.Fprintf(stdout, "Aviso: %d arquivo(s) do template não estão em \"files\" e não serão copiados (rode \"crudding init\" na pasta do template para atualizar):\n", len(unlisted))
			const shown = 20
			for _, file := range unlisted[:min(len(unlisted), shown)] {
				fmt.Fprintf(stdout, "  %s\n", file)
			}
			if len(unlisted) > shown {
				fmt.Fprintf(stdout, "  ... e mais %d\n", len(unlisted)-shown)
			}
		}
	}

	if name == "" {
		name, err = project.ReadTemplateName(os.Stdin, stdout, defaultName)
		if err != nil {
			return err
		}
	}

	targetDir, savedManifest, err := custom.Save(absDir, name)
	if err != nil {
		return err
	}

	stack := savedManifest.Stack()
	if stack == "" {
		stack = "personalizado"
	}
	fmt.Fprintf(stdout, "Template %s salvo com sucesso em: %s\n", name, targetDir)
	fmt.Fprintf(stdout, "Estrutura: %d pastas, %d arquivos (%s)\n\n", len(savedManifest.Directories), len(savedManifest.Files), stack)
	fmt.Fprintf(stdout, "Para listar os templates salvos: crudding template list\n")
	fmt.Fprintf(stdout, "Para gerar um projeto a partir deste template: crudding template use %s <nome-do-projeto>\n", name)
	return nil
}

func runTemplateList(args []string, stdout, _ io.Writer) error {
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			fmt.Fprint(stdout, TemplateListUsage())
			return nil
		}
	}
	templatesDir, err := custom.EnsureTemplatesDir()
	if err != nil {
		return err
	}
	list, err := custom.List()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Fprintf(stdout, "Nenhum template salvo em %s.\n\nPara salvar um template:\n  crudding template <pasta> [nome]\n", templatesDir)
		return nil
	}
	fmt.Fprintf(stdout, "Templates salvos em %s (%d):\n\n", templatesDir, len(list))
	for _, item := range list {
		stack := item.Manifest.Stack()
		if stack == "" {
			stack = "personalizado"
		}
		fmt.Fprintf(stdout, "  - %-20s (%s) - %d pastas, %d arquivos\n", item.Name, stack, len(item.Manifest.Directories), len(item.Manifest.Files))
	}
	fmt.Fprint(stdout, "\nPara gerar um projeto a partir de um template:\n  crudding template use <template> <nome-do-projeto>\n")
	return nil
}

func runTemplateUse(args []string, stdout, _ io.Writer) error {
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			fmt.Fprint(stdout, TemplateUseUsage())
			return nil
		}
	}
	if len(args) == 0 {
		return fmt.Errorf("nenhum template informado; execute \"crudding template list\" para ver os templates salvos")
	}
	if len(args) > 2 {
		return fmt.Errorf("argumentos inesperados: %s", strings.Join(args[2:], " "))
	}

	templateName := strings.TrimSpace(args[0])
	templatePath, m, err := custom.Get(templateName)
	if err != nil {
		return err
	}

	var projectName string
	if len(args) == 2 {
		projectName = strings.TrimSpace(args[1])
		if key, value, ok := strings.Cut(projectName, ":"); ok && (key == "name" || key == "n") {
			projectName = strings.TrimSpace(value)
		}
		if err := project.ValidateProjectName(projectName); err != nil {
			return err
		}
	} else {
		projectName, err = project.ReadProjectName(os.Stdin, stdout)
		if err != nil {
			return err
		}
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	projectDirectory := filepath.Join(workingDirectory, projectName)
	if err := generator.GenerateFromTemplate(projectDirectory, projectName, templatePath); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "\nProjeto gerado em: %s\n", projectDirectory)
	fmt.Fprintf(stdout, "Template: %s (%d pastas, %d arquivos)\n", m.Name, len(m.Directories), len(m.Files))
	return nil
}
