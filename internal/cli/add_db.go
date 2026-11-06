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
	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
)

// AddDBUsage gera o texto de ajuda do comando "crudding add db".
func AddDBUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Adiciona a infraestrutura e configuração de um banco de dados ao projeto atual.\n")
	b.WriteString("  Gera ou atualiza docker-compose.yml, .env.example, migrations/ e adiciona as\n")
	b.WriteString("  funções de conexão e dependências necessárias (go.mod, requirements.txt ou package.json).\n")
	b.WriteString("  - Sem argumentos (\"crudding add db\"), abre menu para escolher o banco.\n")
	b.WriteString("  - Com argumento, adiciona o banco diretamente.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding add db [<banco> | db:<banco> | d:<banco>] [--json]\n\n")
	b.WriteString("bancos suportados:\n")
	b.WriteString("  postgres, mysql, mongo\n\n")
	b.WriteString("flags:\n")
	b.WriteString(jsonFlagUsage)
	b.WriteString("\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add db\n")
	b.WriteString("  crudding add db postgres\n")
	b.WriteString("  crudding add db d:mysql\n")
	return b.String()
}

// dbSummary monta o resumo do "add db": subir o banco e instalar os drivers.
func dbSummary(root string, m manifest.Manifest, result common.Result) summary {
	return summary{
		Command:  "add db",
		Project:  root,
		Created:  result.Created,
		Modified: appendManifest(result.Modified),
		Next:     appendNonEmpty([]string{"docker compose up -d"}, installCommand(m)),
	}
}

func parseDBArgs(args []string) (string, bool, bool, error) {
	if len(args) == 0 {
		return "", false, false, nil
	}
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			return "", false, true, nil
		}
	}
	fs := flag.NewFlagSet("crudding add db", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var db string
	fs.StringVar(&db, "db", "", "banco de dados (postgres, mysql, mongo)")
	fs.StringVar(&db, "d", "", "banco de dados (postgres, mysql, mongo)")
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
	if db != "" && len(rest) > 0 {
		return "", false, false, fmt.Errorf("argumentos inesperados: %s", strings.Join(rest, " "))
	}
	if len(rest) > 1 {
		return "", false, false, fmt.Errorf("argumentos inesperados: %s", strings.Join(rest[1:], " "))
	}
	if db == "" && len(rest) == 1 {
		token := rest[0]
		if key, val, ok := strings.Cut(token, ":"); ok && (key == "db" || key == "d") {
			db = val
		} else {
			db = token
		}
	}
	db = strings.TrimSpace(strings.ToLower(db))
	if err := generator.ValidateDatabase(db); err != nil {
		return "", false, false, err
	}
	return db, true, false, nil
}

func runAddDB(args []string, stdout, _ io.Writer) error {
	args, asJSON, err := splitJSONFlag(args)
	if err != nil {
		return err
	}
	argDB, hasArg, showHelp, err := parseDBArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Fprint(stdout, AddDBUsage())
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

	db := argDB
	if !hasArg && asJSON {
		return errJSONNeedsArg("o banco de dados")
	}
	if !hasArg {
		selected, err := choose("escolha o banco de dados", generator.SupportedDatabases)
		if err != nil {
			return err
		}
		db = selected
	}

	result, err := generator.AddDatabase(root, m, db, crudding.Templates)
	if err != nil {
		return err
	}
	if !asJSON {
		fmt.Fprintf(stdout, "\nBanco de dados %s adicionado com sucesso.\n", db)
	}
	return dbSummary(root, m, result).write(stdout, asJSON)
}
