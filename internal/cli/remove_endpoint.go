package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/sinueh-andreata/crudding/internal/generator"
	"github.com/sinueh-andreata/crudding/internal/generator/common"
	"github.com/sinueh-andreata/crudding/internal/manifest"
	"github.com/sinueh-andreata/crudding/internal/project"
)

// RemoveEndpointUsage gera o texto de ajuda do comando "crudding remove endpoint".
func RemoveEndpointUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Desfaz o \"crudding add endpoint\": remove as cópias do endpoint de exemplo e as\n")
	b.WriteString("  linhas duplicadas nos arquivos de \"register\", além de atualizar o project.crud.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding remove endpoint [<nome> | name:<nome> | n:<nome>]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding remove endpoint customer\n")
	b.WriteString("  crudding remove endpoint n:invoice\n")
	return b.String()
}

func printRemoveEndpointSummary(stdout io.Writer, name string, result common.RemoveResult) {
	fmt.Fprintf(stdout, "\nEndpoint %s removido.\n", name)
	for _, file := range result.Removed {
		fmt.Fprintf(stdout, "  removido   %s\n", file)
	}
	for _, file := range result.Modified {
		fmt.Fprintf(stdout, "  alterado   %s\n", file)
	}
}

func parseRemoveEndpointArgs(args []string) (string, bool, bool, error) {
	if len(args) == 0 {
		return "", false, false, nil
	}
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			return "", false, true, nil
		}
	}
	fs := flag.NewFlagSet("crudding remove endpoint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var name string
	fs.StringVar(&name, "name", "", "nome do endpoint (snake_case)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", false, true, nil
		}
		return "", false, false, err
	}
	rest := fs.Args()
	if name != "" && len(rest) > 0 {
		return "", false, false, fmt.Errorf("argumentos inesperados: %s", strings.Join(rest, " "))
	}
	if len(rest) > 1 {
		return "", false, false, fmt.Errorf("argumentos inesperados: %s", strings.Join(rest[1:], " "))
	}
	if name == "" && len(rest) == 1 {
		token := rest[0]
		if key, val, ok := strings.Cut(token, ":"); ok && (key == "name" || key == "n") {
			name = val
		} else {
			name = token
		}
	}
	name = strings.TrimSpace(name)
	if err := project.ValidateEndpointName(name); err != nil {
		return "", false, false, err
	}
	return name, true, false, nil
}

func runRemoveEndpoint(args []string, stdout, _ io.Writer) error {
	argName, hasArg, showHelp, err := parseRemoveEndpointArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Fprint(stdout, RemoveEndpointUsage())
		return nil
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	root, m, err := manifest.Find(workingDirectory)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Projeto %s: %s\n", root, m.Stack())
	if err := generator.CheckEndpointSupport(m); err != nil {
		return err
	}

	name := argName
	if !hasArg {
		name, err = project.ReadEndpointName(os.Stdin, stdout)
		if err != nil {
			return err
		}
	}

	result, err := generator.RemoveEndpoint(root, m, name)
	if err != nil {
		return err
	}
	printRemoveEndpointSummary(stdout, name, result)
	return nil
}

