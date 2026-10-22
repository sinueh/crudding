package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/sinueh-andreata/crudding"
	"github.com/sinueh-andreata/crudding/internal/generator"
	"github.com/sinueh-andreata/crudding/internal/generator/common"
	"github.com/sinueh-andreata/crudding/internal/manifest"
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
	b.WriteString("  crudding add db [<banco> | db:<banco> | d:<banco>]\n\n")
	b.WriteString("bancos suportados:\n")
	b.WriteString("  postgres, mysql, mongo\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add db\n")
	b.WriteString("  crudding add db postgres\n")
	b.WriteString("  crudding add db d:mysql\n")
	return b.String()
}

func printDBSummary(stdout io.Writer, db string, lang string, result common.Result) {
	fmt.Fprintf(stdout, "\nBanco de dados %s adicionado com sucesso.\n", db)
	for _, file := range result.Created {
		fmt.Fprintf(stdout, "  criado     %s\n", file)
	}
	for _, file := range result.Modified {
		fmt.Fprintf(stdout, "  alterado   %s\n", file)
	}
	if lang == "go" {
		fmt.Fprintln(stdout, "\nPara atualizar as dependências Go, execute: go mod tidy")
	} else if lang == "python" {
		fmt.Fprintln(stdout, "\nPara instalar as dependências Python, execute: pip install -r requirements.txt")
	} else if lang == "node" {
		fmt.Fprintln(stdout, "\nPara instalar as dependências Node, execute: npm install")
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
	fmt.Fprintf(stdout, "Projeto %s: %s\n", root, m.Stack())

	db := argDB
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
	printDBSummary(stdout, db, m.Language, result)
	return nil
}

