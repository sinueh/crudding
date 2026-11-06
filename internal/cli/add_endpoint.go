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
	b.WriteString("  crudding add endpoint [<nome> | name:<nome> | n:<nome>] [campo:tipo[?] ...] [--protected] [--roles <roles>] [--owner] [--json]\n\n")
	b.WriteString("flags:\n")
	b.WriteString("  --protected      exige autenticação para acessar as rotas do endpoint\n")
	b.WriteString("  --roles          restringe o endpoint a papéis específicos (RBAC, separados por vírgula)\n")
	b.WriteString("  --owner          associa o registro ao usuário autenticado (coluna user_id:ref e filtro automático)\n")
	b.WriteString(jsonFlagUsage)
	b.WriteString("\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add endpoint\n")
	b.WriteString("  crudding add endpoint customer\n")
	b.WriteString("  crudding add endpoint n:invoice\n")
	b.WriteString("  crudding add endpoint product name:str price:decimal image_url:str?\n")
	b.WriteString("  crudding add endpoint cart_item cart:ref product:ref quantity:int\n")
	b.WriteString("  crudding add endpoint order total:decimal status:str --protected\n")
	b.WriteString("  crudding add endpoint audit_log action:str details:text --roles \"admin\"\n")
	b.WriteString("  crudding add endpoint profile bio:text avatar_url:str? --owner\n")
	return b.String()
}

func parseEndpointArgs(args []string) (string, bool, bool, error) {
	opts, hasArg, showHelp, err := parseTypedEndpointOptions(args)
	return opts.Name, hasArg, showHelp, err
}

func parseTypedEndpointArgs(args []string) (string, []string, bool, bool, error) {
	opts, hasArg, showHelp, err := parseTypedEndpointOptions(args)
	return opts.Name, opts.Specs, hasArg, showHelp, err
}

// parseTypedEndpointOptions extrai o nome, campos tipados e flags de autenticação (--protected, --roles, --owner).
func parseTypedEndpointOptions(args []string) (generator.EndpointOptions, bool, bool, error) {
	var opts generator.EndpointOptions
	if len(args) == 0 {
		return opts, false, false, nil
	}
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			return opts, false, true, nil
		}
	}

	var remaining []string
	for i := 0; i < len(args); i++ {
		token := args[i]
		switch {
		case token == "--protected" || token == "-protected" || token == "protected:true":
			opts.Protected = true
		case token == "--owner" || token == "-owner" || token == "owner:true":
			opts.Owner = true
			opts.Protected = true
		case token == "--roles" || token == "-roles":
			if i+1 < len(args) {
				i++
				opts.Roles = parseRolesList(args[i])
				opts.Protected = true
			}
		case strings.HasPrefix(token, "--roles="):
			opts.Roles = parseRolesList(strings.TrimPrefix(token, "--roles="))
			opts.Protected = true
		case strings.HasPrefix(token, "-roles="):
			opts.Roles = parseRolesList(strings.TrimPrefix(token, "-roles="))
			opts.Protected = true
		case strings.HasPrefix(token, "roles:"):
			opts.Roles = parseRolesList(strings.TrimPrefix(token, "roles:"))
			opts.Protected = true
		default:
			remaining = append(remaining, token)
		}
	}

	name, hasArg, showHelp, err := parseEndpointName(remaining)
	if err != nil || showHelp || !hasArg {
		opts.Name = name
		return opts, hasArg, showHelp, err
	}
	opts.Name = name
	specs := endpointFieldArgs(remaining)
	if _, err := generator.ParseEndpointFields(specs); err != nil {
		return opts, false, false, err
	}
	opts.Specs = specs
	return opts, true, false, nil
}

func parseRolesList(val string) []string {
	var res []string
	for _, r := range strings.Split(val, ",") {
		if trimmed := strings.TrimSpace(r); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
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
	args, asJSON, err := splitJSONFlag(args)
	if err != nil {
		return err
	}
	opts, hasArg, showHelp, err := parseTypedEndpointOptions(args)
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
	if !asJSON {
		fmt.Fprintf(stdout, "Projeto %s: %s\n", root, m.Stack())
	}
	if err := generator.CheckEndpointSupport(m); err != nil {
		return err
	}

	if !hasArg && asJSON {
		return errJSONNeedsArg("o nome do endpoint")
	}
	if !hasArg {
		name, err := project.ReadEndpointName(os.Stdin, stdout)
		if err != nil {
			return err
		}
		opts.Name = name
	}

	result, notes, err := generator.AddTypedEndpointWithOptions(root, m, opts, crudding.Templates)
	if err != nil {
		return err
	}
	routes := endpointRoutes(root, m.TemplateEndpoint.Register, opts.Name)
	report := endpointSummary("add endpoint", root, m, result.Created, nil, result.Modified, routes)
	if !asJSON {
		fmt.Fprintf(stdout, "\nEndpoint %s criado.\n", opts.Name)
	}
	if err := report.write(stdout, asJSON); err != nil {
		return err
	}
	if !asJSON {
		for _, note := range notes {
			fmt.Fprintf(stdout, "  aviso      %s\n", note)
		}
	}
	return nil
}
