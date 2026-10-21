package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"crudding/internal/generator"
	"crudding/internal/generator/custom"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

// TemplateUsage gera o texto de ajuda do comando "crudding template".
func TemplateUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Gera um projeto novo no diretório atual copiando a estrutura listada no\n")
	b.WriteString("  project.crud de outro projeto (criado com \"crudding init\"). O name do template\n")
	b.WriteString("  é trocado pelo nome do projeto novo nos caminhos e no conteúdo dos arquivos.\n")
	b.WriteString("  - Sem o nome, solicita interativamente.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding template <pasta> [<nome> | name:<nome> | n:<nome>]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding template ../meu-template\n")
	b.WriteString("  crudding template ../meu-template minha-api\n")
	b.WriteString("  crudding template $HOME/templates/go-api n:minha-api\n")
	return b.String()
}

// parseTemplateArgs devolve a pasta do template e o nome do projeto (vazio
// quando deve ser perguntado).
func parseTemplateArgs(args []string) (dir, name string, showHelp bool, err error) {
	if len(args) == 0 {
		return "", "", false, fmt.Errorf("nenhuma pasta de template informada; execute \"crudding template -h\" para ver o uso")
	}
	if slices.ContainsFunc(args, func(token string) bool { return slices.Contains(helpFlags, token) }) {
		return "", "", true, nil
	}
	if len(args) > 2 {
		return "", "", false, fmt.Errorf("argumentos inesperados: %s", strings.Join(args[2:], " "))
	}
	dir = strings.TrimSpace(args[0])
	if dir == "" {
		return "", "", false, fmt.Errorf("nenhuma pasta de template informada")
	}
	if len(args) == 2 {
		name = strings.TrimSpace(args[1])
		if key, value, ok := strings.Cut(name, ":"); ok && (key == "name" || key == "n") {
			name = strings.TrimSpace(value)
		}
		if err := project.ValidateProjectName(name); err != nil {
			return "", "", false, err
		}
	}
	return dir, name, false, nil
}

func runTemplate(args []string, stdout, _ io.Writer) error {
	dir, name, showHelp, err := parseTemplateArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Fprint(stdout, TemplateUsage())
		return nil
	}

	// Lê o template antes de pedir o nome, para falhar cedo.
	templates := os.DirFS(dir)
	m, err := manifest.ReadFS(templates)
	if err != nil {
		return fmt.Errorf("template %s: %w", dir, err)
	}
	fmt.Fprintf(stdout, "Template %s: %s\n", m.Name, m.Stack())
	unlisted, err := custom.Unlisted(templates, m)
	if err != nil {
		return fmt.Errorf("template %s: %w", dir, err)
	}
	if len(unlisted) > 0 {
		fmt.Fprintf(stdout, "Aviso: %d arquivo(s) do template não estão em \"files\" e não serão copiados (rode \"crudding init\" na pasta do template para atualizar):\n", len(unlisted))
		const shown = 20
		for _, file := range unlisted[:min(len(unlisted), shown)] {
			fmt.Fprintf(stdout, "  %s\n", file)
		}
		if len(unlisted) > shown {
			fmt.Fprintf(stdout, "  ... e mais %d\n", len(unlisted)-shown)
		}
	}

	if name == "" {
		if name, err = project.ReadProjectName(os.Stdin, stdout); err != nil {
			return err
		}
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	projectDirectory := filepath.Join(workingDirectory, name)
	if err := generator.GenerateFromTemplate(projectDirectory, name, dir); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "\nProjeto gerado em: %s\n", projectDirectory)
	fmt.Fprintf(stdout, "Template: %s (%d pastas, %d arquivos)\n", m.Name, len(m.Directories), len(m.Files))
	return nil
}
