package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh-andreata/crudding/internal/generator/custom"
	"github.com/sinueh-andreata/crudding/internal/manifest"
)

// InitUsage gera o texto de ajuda do comando "crudding init".
func InitUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Grava no diretório atual um project.crud com a estrutura do projeto (todas as\n")
	b.WriteString("  pastas e arquivos, menos o que estiver em \"ignore\"), para usá-lo como template\n")
	b.WriteString("  com \"crudding template\".\n")
	b.WriteString("  - O ignore começa vazio: adicione padrões no estilo .gitignore (ex.: \"node_modules\",\n")
	b.WriteString("    \".env\", \"*.log\") e rode o init de novo. Pastas .git nunca são listadas.\n")
	b.WriteString("  - Rodar de novo atualiza \"directories\" e \"files\" e mantém o resto do project.crud.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding init\n")
	return b.String()
}

func runInit(args []string, stdout, _ io.Writer) error {
	if len(args) == 1 && slices.Contains(helpFlags, args[0]) {
		fmt.Fprint(stdout, InitUsage())
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("argumentos inesperados: %s", strings.Join(args, " "))
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	m, err := custom.Init(workingDirectory)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s gravado em %s\n", manifest.FileName, filepath.Join(workingDirectory, manifest.FileName))
	fmt.Fprintf(stdout, "Template %s: %d pastas e %d arquivos (o que está em \"ignore\" ficou de fora).\n", m.Name, len(m.Directories), len(m.Files))
	fmt.Fprintf(stdout, "\nPara gerar um projeto a partir dele: crudding template %s <nome>\n", workingDirectory)
	return nil
}
