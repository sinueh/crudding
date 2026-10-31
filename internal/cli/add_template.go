package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/custom"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

// AddTemplateUsage gera o texto de ajuda do comando "crudding add template".
func AddTemplateUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Salva o projeto de <pasta> como template em ~/.crudding/templates/<nome>,\n")
	b.WriteString("  garantindo um project.crud válido e trocando o nome do projeto pelo do template.\n")
	b.WriteString("  - Sem a pasta ou o nome, solicita interativamente.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding add template <pasta> [<nome> | name:<nome> | n:<nome>]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add template ./meu-projeto\n")
	b.WriteString("  crudding add template ./meu-projeto meu-template\n")
	return b.String()
}

func runAddTemplate(args []string, stdout, _ io.Writer) error {
	if slices.ContainsFunc(args, func(token string) bool { return slices.Contains(helpFlags, token) }) {
		fmt.Fprint(stdout, AddTemplateUsage())
		return nil
	}

	var (
		dir, name string
		err       error
	)
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
