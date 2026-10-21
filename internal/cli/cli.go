package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// errCancelled sinaliza que o usuário saiu do menu; não é tratado como falha.
var errCancelled = errors.New("cancelado pelo usuário")

var helpFlags = []string{"-h", "--help", "help"}

// Command descreve um comando disponível na CLI.
type Command struct {
	Args    string
	Summary string
	Run     func(args []string, stdout, stderr io.Writer) error
}

func (c Command) label() string {
	return strings.TrimSpace("crudding " + c.Args)
}

var commands = []Command{
	{
		Args:    "",
		Summary: "gera um novo projeto (menu interativo)",
		Run:     runInteractive,
	},
	{
		Args:    "new",
		Summary: "gera um novo projeto via flags (name/n, lang/l, framework/f, arch/a, runtime/r)",
		Run:     runNew,
	},
	{
		Args:    "init",
		Summary: "grava no diretório atual um project.crud com a estrutura do projeto, para usá-lo como template",
		Run:     runInit,
	},
	{
		Args:    "template",
		Summary: "gera um novo projeto a partir de um projeto-template (crudding template <pasta> [nome])",
		Run:     runTemplate,
	},
	{
		Args:    "stacks",
		Summary: "lista as linguagens, runtimes, frameworks e arquiteturas suportadas",
		Run:     runStacks,
	},
	{
		Args:    "info",
		Summary: "exibe a stack do projeto atual e os endpoints registrados no project.crud",
		Run:     runInfo,
	},
	{
		Args:    "info draw",
		Summary: "desenha em ASCII a arquitetura do projeto atual",
		Run:     runInfoDraw,
	},
	{
		Args:    "add endpoint",
		Summary: "adiciona um endpoint CRUD ao projeto atual (lê o project.crud)",
		Run:     runAddEndpoint,
	},
	{
		Args:    "add db",
		Summary: "adiciona conexão com banco de dados ao projeto atual (lê o project.crud)",
		Run:     runAddDB,
	},
	{
		Args:    "remove endpoint",
		Summary: "remove um endpoint CRUD do projeto atual (lê o project.crud)",
		Run:     runRemoveEndpoint,
	},
}

// Commands devolve a lista de comandos registrados na CLI.
func Commands() []Command {
	return slices.Clone(commands)
}

// Usage gera o texto de ajuda a partir dos comandos registrados.
func Usage() string {
	var b strings.Builder
	b.WriteString("uso:\n")
	for _, cmd := range commands {
		fmt.Fprintf(&b, "  %-21s    %s\n", cmd.label(), cmd.Summary)
	}
	return b.String()
}

// Run despacha os argumentos da linha de comando e devolve o código de saída.
func Run(args []string, stdout, stderr io.Writer) int {
	input := strings.Join(args, " ")
	if slices.Contains(helpFlags, input) {
		fmt.Fprint(stdout, Usage())
		return 0
	}

	cmd, cmdArgs, ok := findCommand(args)
	if !ok {
		fmt.Fprintf(stderr, "comando desconhecido: %s\n\n%s", input, Usage())
		return 2
	}

	return execute(cmd, cmdArgs, stdout, stderr)
}

func findCommand(args []string) (Command, []string, bool) {
	var (
		bestCmd   Command
		bestArgs  []string
		bestParts int
	)
	for _, cmd := range commands {
		if cmd.Args == "" {
			continue
		}
		parts := strings.Fields(cmd.Args)
		if len(args) >= len(parts) && slices.Equal(args[:len(parts)], parts) && len(parts) > bestParts {
			bestCmd = cmd
			bestArgs = args[len(parts):]
			bestParts = len(parts)
		}
	}
	if bestParts > 0 {
		return bestCmd, bestArgs, true
	}
	if len(args) == 0 {
		for _, cmd := range commands {
			if cmd.Args == "" {
				return cmd, nil, true
			}
		}
	}
	return Command{}, nil, false
}

func execute(cmd Command, args []string, stdout, stderr io.Writer) int {
	err := cmd.Run(args, stdout, stderr)
	if err == nil || errors.Is(err, errCancelled) {
		return 0
	}
	fmt.Fprintf(stderr, "crudding: %v\n", err)
	return 1
}
