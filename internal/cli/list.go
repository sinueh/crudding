package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/sinueh/crudding/internal/generator"
)

// StacksUsage gera o texto de ajuda do comando "crudding stacks".
func StacksUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Exibe em formato de árvore todas as combinações suportadas de linguagens,\n")
	b.WriteString("  runtimes, frameworks e arquiteturas registradas no crudding.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding stacks [linguagem | lang:<linguagem> | l:<linguagem>]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding stacks\n")
	b.WriteString("  crudding stacks go\n")
	b.WriteString("  crudding stacks l:node\n")
	return b.String()
}

func formatLanguageTree(lang generator.Language) string {
	type branch struct {
		label  string
		values []string
	}
	var branches []branch
	if len(lang.Runtimes) > 0 {
		branches = append(branches, branch{label: "runtimes", values: lang.Runtimes})
	}
	branches = append(branches,
		branch{label: "frameworks", values: lang.Frameworks},
		branch{label: "architectures", values: lang.Architectures},
	)

	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s)\n", lang.Name, lang.Label)
	for i, br := range branches {
		prefix := "├──"
		if i == len(branches)-1 {
			prefix = "└──"
		}
		fmt.Fprintf(&b, "%s %s: %s\n", prefix, br.label, strings.Join(br.values, ", "))
	}
	return b.String()
}

func runStacks(args []string, stdout, _ io.Writer) error {
	fs := flag.NewFlagSet("crudding stacks", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, StacksUsage())
			return nil
		}
		return err
	}

	tokens := fs.Args()
	if len(tokens) == 1 && tokens[0] == "help" {
		fmt.Fprint(stdout, StacksUsage())
		return nil
	}
	if len(tokens) > 1 {
		return fmt.Errorf("argumentos inesperados: %s", strings.Join(tokens[1:], " "))
	}

	languages := generator.Languages()
	if len(tokens) == 1 {
		target := strings.TrimSpace(tokens[0])
		if key, val, hasColon := strings.Cut(target, ":"); hasColon {
			if key != "lang" && key != "l" {
				return fmt.Errorf("flag desconhecida: %s", tokens[0])
			}
			target = strings.TrimSpace(val)
		}
		lang, ok := generator.Lookup(target)
		if !ok {
			return fmt.Errorf("linguagem não suportada: %s", target)
		}
		languages = []generator.Language{lang}
	}

	for i, lang := range languages {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprint(stdout, formatLanguageTree(lang))
	}
	return nil
}

