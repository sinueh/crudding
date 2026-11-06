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

// AddAuthUsage gera o texto de ajuda do comando "crudding add auth".
func AddAuthUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Adiciona um módulo completo de autenticação ao projeto atual.\n")
	b.WriteString("  Gera modelos (User, RefreshToken), schemas de validação, segurança (hashing de senha,\n")
	b.WriteString("  geração e validação de tokens), rotas nativas (/auth/register, /auth/login,\n")
	b.WriteString("  /auth/refresh, /auth/me), middlewares de proteção e RBAC, variáveis de ambiente\n")
	b.WriteString("  (.env.example), migrações e dependências no arquivo de pacotes.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding add auth [<estratégia>] [--roles <roles>] [--hasher <hasher>] [--refresh-token] [--json]\n\n")
	b.WriteString("estratégias suportadas:\n")
	b.WriteString("  jwt (padrão), session, api-key\n\n")
	b.WriteString("flags:\n")
	b.WriteString("  --roles          papéis para RBAC separados por vírgula (padrão: admin,user)\n")
	b.WriteString("  --hasher         algoritmo de hash de senhas: bcrypt (padrão) ou argon2id\n")
	b.WriteString("  --refresh-token  habilita suporte e rotas de token de atualização\n")
	b.WriteString(jsonFlagUsage)
	b.WriteString("\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding add auth\n")
	b.WriteString("  crudding add auth jwt\n")
	b.WriteString("  crudding add auth jwt --roles \"admin,user\" --hasher argon2id --refresh-token\n")
	b.WriteString("  crudding add auth session\n")
	b.WriteString("  crudding add auth api-key\n")
	return b.String()
}

// authSummary monta o resumo do "add auth".
func authSummary(root string, m manifest.Manifest, result common.Result) summary {
	return summary{
		Command:  "add auth",
		Project:  root,
		Created:  result.Created,
		Modified: appendManifest(result.Modified),
		Routes:   []string{"/auth/register", "/auth/login", "/auth/refresh", "/auth/me"},
		Next:     appendNonEmpty([]string{"docker compose up -d (se usar banco de dados)"}, installCommand(m)),
	}
}

func parseAuthArgs(args []string) (generator.AuthOptions, bool, bool, error) {
	opts := generator.AuthOptions{
		Strategy: "jwt",
		Hasher:   "bcrypt",
		Roles:    []string{"admin", "user"},
	}
	if len(args) == 0 {
		return opts, false, false, nil
	}
	for _, token := range args {
		if slices.Contains(helpFlags, token) {
			return opts, false, true, nil
		}
	}

	var rolesStr, hasher, strategy string
	var refresh bool

	fs := flag.NewFlagSet("crudding add auth", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&rolesStr, "roles", "", "papéis para RBAC")
	fs.StringVar(&rolesStr, "r", "", "papéis para RBAC")
	fs.StringVar(&hasher, "hasher", "", "algoritmo de hash")
	fs.StringVar(&hasher, "h", "", "algoritmo de hash")
	fs.StringVar(&strategy, "strategy", "", "estratégia de autenticação")
	fs.StringVar(&strategy, "s", "", "estratégia de autenticação")
	fs.BoolVar(&refresh, "refresh-token", false, "habilita refresh token")
	fs.BoolVar(&refresh, "refresh", false, "habilita refresh token")

	// Filtra flags no formato chave:valor
	var filtered []string
	hasExplicitArg := false
	for _, arg := range args {
		if k, v, ok := strings.Cut(arg, ":"); ok {
			switch k {
			case "roles", "r":
				rolesStr = v
				hasExplicitArg = true
			case "hasher", "h":
				hasher = v
				hasExplicitArg = true
			case "strategy", "s":
				strategy = v
				hasExplicitArg = true
			case "refresh", "refresh-token":
				refresh = v == "true" || v == "1"
				hasExplicitArg = true
			default:
				filtered = append(filtered, arg)
			}
		} else {
			filtered = append(filtered, arg)
		}
	}

	var flagTokens []string
	var posTokens []string
	for i := 0; i < len(filtered); i++ {
		token := filtered[i]
		if strings.HasPrefix(token, "-") {
			flagTokens = append(flagTokens, token)
			name := strings.TrimLeft(token, "-")
			if (name == "roles" || name == "r" || name == "hasher" || name == "h" || name == "strategy" || name == "s") && !strings.Contains(token, "=") {
				if i+1 < len(filtered) && !strings.HasPrefix(filtered[i+1], "-") {
					i++
					flagTokens = append(flagTokens, filtered[i])
				}
			}
		} else {
			posTokens = append(posTokens, token)
		}
	}

	if err := fs.Parse(flagTokens); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, false, true, nil
		}
		return opts, false, false, err
	}

	rest := append(fs.Args(), posTokens...)
	if len(rest) == 1 && slices.Contains(helpFlags, rest[0]) {
		return opts, false, true, nil
	}
	if strategy == "" && len(rest) >= 1 {
		strategy = rest[0]
		hasExplicitArg = true
		rest = rest[1:]
	}
	if len(rest) > 0 {
		return opts, false, false, fmt.Errorf("argumentos inesperados: %s", strings.Join(rest, " "))
	}

	if strategy != "" {
		opts.Strategy = strings.ToLower(strings.TrimSpace(strategy))
		if opts.Strategy == "apikey" {
			opts.Strategy = "api-key"
		}
		if err := generator.ValidateAuthStrategy(opts.Strategy); err != nil {
			return opts, false, false, err
		}
	}
	if hasher != "" {
		opts.Hasher = strings.ToLower(strings.TrimSpace(hasher))
		if opts.Hasher == "argon2" {
			opts.Hasher = "argon2id"
		}
		if err := generator.ValidateHasher(opts.Hasher); err != nil {
			return opts, false, false, err
		}
	}
	if rolesStr != "" {
		var parts []string
		for _, p := range strings.Split(rolesStr, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		if len(parts) > 0 {
			opts.Roles = parts
		}
	}
	opts.HasRefresh = refresh

	return opts, hasExplicitArg || strategy != "", false, nil
}

func runAddAuth(args []string, stdout, _ io.Writer) error {
	args, asJSON, err := splitJSONFlag(args)
	if err != nil {
		return err
	}
	opts, hasArg, showHelp, err := parseAuthArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Fprint(stdout, AddAuthUsage())
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
	if m.HasAuth() {
		return fmt.Errorf("o projeto já possui autenticação configurada (%s)", m.Auth.Strategy)
	}

	if !hasArg && !asJSON {
		selected, err := choose("escolha a estratégia de autenticação", generator.SupportedAuthStrategies)
		if err != nil {
			return err
		}
		opts.Strategy = selected
	}

	result, err := generator.AddAuth(root, m, opts, crudding.Templates)
	if err != nil {
		return err
	}

	report := authSummary(root, m, result)
	if !asJSON {
		fmt.Fprintf(stdout, "\nAutenticação configurada com sucesso (%s)!\n", opts.Strategy)
	}
	return report.write(stdout, asJSON)
}
