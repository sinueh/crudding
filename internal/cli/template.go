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
	"github.com/sinueh/crudding/internal/project"
)

// TemplateUsage gera o texto de ajuda do comando "crudding template".
func TemplateUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Gerencia os templates salvos em ~/.crudding/templates/<nome>.\n")
	b.WriteString("  Os templates salvos podem ser listados, usados para gerar novos projetos e exportados.\n\n")
	b.WriteString("comandos:\n")
	b.WriteString("  crudding add template <pasta> [<nome> | name:<nome> | n:<nome>]\n")
	b.WriteString("    Salva o projeto de <pasta> como template. Sem o nome, solicita interativamente.\n")
	b.WriteString("  crudding template list\n")
	b.WriteString("    Lista todos os templates salvos em ~/.crudding/templates.\n")
	b.WriteString("  crudding template use <template> [<nome> | name:<nome> | n:<nome>]\n")
	b.WriteString("    Gera um novo projeto a partir de um template salvo.\n")
	b.WriteString("  crudding template export <template> [destino]\n")
	b.WriteString("    Empacota um template salvo em um arquivo .zip (padrão: ./<template>.zip).\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add template ./meu-projeto\n")
	b.WriteString("  crudding add template ./meu-projeto meu-template\n")
	b.WriteString("  crudding template list\n")
	b.WriteString("  crudding template use meu-template minha-api\n")
	b.WriteString("  crudding template export meu-template\n")
	return b.String()
}

// TemplateExportUsage gera o texto de ajuda do comando "crudding template export".
func TemplateExportUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Empacota um template salvo em ~/.crudding/templates num arquivo .zip, com o\n")
	b.WriteString("  conteúdo dentro da pasta <template>/. Para usá-lo em outra máquina, extraia o\n")
	b.WriteString("  .zip e rode \"crudding add template ./<template>\".\n")
	b.WriteString("  - Sem o destino, grava ./<template>.zip; se o destino for uma pasta, grava <destino>/<template>.zip.\n")
	b.WriteString("  - Nunca sobrescreve um arquivo existente.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding template export <template> [destino]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding template export meu-template\n")
	b.WriteString("  crudding template export meu-template ~/Downloads\n")
	b.WriteString("  crudding template export meu-template backup.zip\n")
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

	if len(args) == 0 || slices.ContainsFunc(args, func(token string) bool { return slices.Contains(helpFlags, token) }) {
		fmt.Fprint(stdout, TemplateUsage())
		return nil
	}

	switch args[0] {
	case "list":
		return runTemplateList(args[1:], stdout, stderr)
	case "use", "new":
		return runTemplateUse(args[1:], stdout, stderr)
	case "export":
		return runTemplateExport(args[1:], stdout, stderr)
	}

	// Atalho: crudding template meu-template minha-api gera o projeto a partir do template salvo.
	if _, _, err := custom.Get(strings.TrimSpace(args[0])); err == nil {
		return runTemplateUse(args, stdout, stderr)
	}
	return fmt.Errorf("subcomando desconhecido: %s; para salvar um projeto como template use \"crudding add template <pasta> [nome]\"", args[0])
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
		fmt.Fprintf(stdout, "Nenhum template salvo em %s.\n\nPara salvar um template:\n  crudding add template <pasta> [nome]\n", templatesDir)
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

func runTemplateExport(args []string, stdout, _ io.Writer) error {
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			fmt.Fprint(stdout, TemplateExportUsage())
			return nil
		}
	}
	if len(args) == 0 {
		return fmt.Errorf("nenhum template informado; execute \"crudding template list\" para ver os templates salvos")
	}
	if len(args) > 2 {
		return fmt.Errorf("argumentos inesperados: %s", strings.Join(args[2:], " "))
	}

	name := strings.TrimSpace(args[0])
	var dest string
	if len(args) == 2 {
		dest = strings.TrimSpace(args[1])
	}
	archive, err := custom.Export(name, dest)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Template %s exportado para: %s\n\n", name, archive)
	fmt.Fprintf(stdout, "Para importar em outra máquina, extraia o .zip e rode: crudding add template ./%s\n", name)
	return nil
}
