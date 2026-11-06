package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/sinueh/crudding"
	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/generator/fields"
	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

// AddEndpointUsage gera o texto de ajuda do comando "crudding add endpoint".
func AddEndpointUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Cria um endpoint novo a partir do campo \"endpoint\" do project.crud: copia os\n")
	b.WriteString("  arquivos do endpoint de exemplo trocando o nome, duplica as linhas dele nos\n")
	b.WriteString("  arquivos de \"register\" (rotas, imports) e atualiza o project.crud.\n")
	b.WriteString("  - Projetos gerados pelo crudding já vêm com o campo preenchido.\n")
	b.WriteString("  - Sem argumentos (\"crudding add endpoint\"), solicita o nome interativamente.\n")
	b.WriteString("  - Com argumento, cria o endpoint diretamente.\n")
	b.WriteString("  - Com campos (nome:tipo, \"?\" no fim = opcional), gera model, schemas, acesso\n")
	b.WriteString("    ao banco (postgres/mysql; sem banco, em memória), controller, migração e teste.\n")
	b.WriteString("    Tipos: " + strings.Join(fields.Types, ", ") + " (ref = chave estrangeira para <nome>, coluna <nome>_id).\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding add endpoint [<nome> | name:<nome> | n:<nome>] [campo:tipo[?] ...]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add endpoint\n")
	b.WriteString("  crudding add endpoint customer\n")
	b.WriteString("  crudding add endpoint n:invoice\n")
	b.WriteString("  crudding add endpoint product name:str price:decimal image_url:str?\n")
	b.WriteString("  crudding add endpoint cart_item cart:ref product:ref quantity:int\n")
	return b.String()
}

func printEndpointSummary(stdout io.Writer, name string, result common.Result) {
	fmt.Fprintf(stdout, "\nEndpoint %s criado.\n", name)
	for _, file := range result.Created {
		fmt.Fprintf(stdout, "  criado     %s\n", file)
	}
	for _, file := range result.Modified {
		fmt.Fprintf(stdout, "  alterado   %s\n", file)
	}
}

func parseEndpointArgs(args []string) (string, bool, bool, error) {
	name, _, hasArg, showHelp, err := parseTypedEndpointArgs(args)
	return name, hasArg, showHelp, err
}

// parseTypedEndpointArgs é o parseEndpointArgs que aceita campos depois do
// nome: "product name:str price:decimal".
func parseTypedEndpointArgs(args []string) (string, []string, bool, bool, error) {
	name, hasArg, showHelp, err := parseEndpointName(args)
	if err != nil || showHelp || !hasArg {
		return name, nil, hasArg, showHelp, err
	}
	specs := endpointFieldArgs(args)
	if _, err := generator.ParseEndpointFields(specs); err != nil {
		return "", nil, false, false, err
	}
	return name, specs, true, false, nil
}

// endpointFieldArgs devolve os argumentos depois do nome do endpoint.
func endpointFieldArgs(args []string) []string {
	for index, token := range args {
		if token == "-name" || token == "--name" {
			return args[min(index+2, len(args)):]
		}
		if strings.HasPrefix(token, "-name=") || strings.HasPrefix(token, "--name=") {
			return args[index+1:]
		}
		if !strings.HasPrefix(token, "-") {
			return args[index+1:]
		}
	}
	return nil
}

func parseEndpointName(args []string) (string, bool, bool, error) {
	if len(args) == 0 {
		return "", false, false, nil
	}
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			return "", false, true, nil
		}
	}
	fs := flag.NewFlagSet("crudding add endpoint", flag.ContinueOnError)
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
	if len(rest) == 1 && slices.Contains(helpFlags, rest[0]) {
		return "", false, true, nil
	}
	if name == "" && len(rest) >= 1 {
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

func runAddEndpoint(args []string, stdout, _ io.Writer) error {
	argName, specs, hasArg, showHelp, err := parseTypedEndpointArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Fprint(stdout, AddEndpointUsage())
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

	result, notes, err := generator.AddTypedEndpoint(root, m, name, specs, crudding.Templates)
	if err != nil {
		return err
	}
	printEndpointSummary(stdout, name, result)
	for _, note := range notes {
		fmt.Fprintf(stdout, "  aviso      %s\n", note)
	}
	return nil
}
