package test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sinueh/crudding"
	"github.com/sinueh/crudding/internal/cli"
	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

func TestAddAuthHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{"add", "auth", arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(add auth %q) exit code = %d, want 0", arg, code)
		}
		if stderr.Len() != 0 {
			t.Fatalf("Run(add auth %q) unexpected stderr = %q", arg, stderr.String())
		}
		got := stdout.String()
		for _, section := range []string{
			"como funciona:",
			"uso:",
			"crudding add auth",
			"estratégias suportadas:",
			"jwt (padrão), session, api-key",
			"--roles",
			"--hasher",
			"--refresh-token",
			"exemplos:",
		} {
			if !strings.Contains(got, section) {
				t.Fatalf("Run(add auth %q) stdout missing %q:\n%s", arg, section, got)
			}
		}
	}
}

func TestAddAuthValidation(t *testing.T) {
	root, m := generateProject(t, project.Config{
		ProjectName:  "test_auth_app",
		Language:     "go",
		Framework:    "gin",
		Architecture: "layered",
	})

	// Estratégia inválida
	_, err := generator.AddAuth(root, m, generator.AuthOptions{Strategy: "oauth2"}, crudding.Templates)
	if err == nil || !strings.Contains(err.Error(), "estratégia de autenticação não suportada") {
		t.Fatalf("AddAuth(invalid strategy) error = %v, want unsupported strategy error", err)
	}

	// Hasher inválido
	_, err = generator.AddAuth(root, m, generator.AuthOptions{Strategy: "jwt", Hasher: "md5"}, crudding.Templates)
	if err == nil || !strings.Contains(err.Error(), "algoritmo de hash não suportado") {
		t.Fatalf("AddAuth(invalid hasher) error = %v, want unsupported hasher error", err)
	}
}

func TestAddAuthGo(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "my_go_auth_app",
		Language:     "go",
		Framework:    "gin",
		Architecture: "layered",
		Database:     "postgres",
	}

	root, _ := generateProject(t, cfg)
	chdirTemp(t, root)

	// Adiciona auth via CLI
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"add", "auth", "jwt", "--roles", "admin,customer", "--hasher", "argon2id", "--refresh-token"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add auth jwt) exit code = %d, stderr = %s", code, stderr.String())
	}

	// Verifica arquivos criados
	expectedFiles := []string{
		"internal/auth/models/user.go",
		"internal/auth/schemas/auth.go",
		"internal/auth/security/security.go",
		"internal/auth/handlers/auth.go",
		"internal/auth/middlewares/auth.go",
		"migrations/000002_auth_create_auth_tables.up.sql",
		"migrations/000002_auth_create_auth_tables.down.sql",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	// Verifica variáveis em .env.example
	envContent, err := os.ReadFile(filepath.Join(root, ".env.example"))
	if err != nil {
		t.Fatalf("Read .env.example error = %v", err)
	}
	for _, key := range []string{"JWT_SECRET", "JWT_EXPIRATION", "REFRESH_TOKEN_EXPIRATION", "BCRYPT_COST", "ARGON2_MEMORY"} {
		if !strings.Contains(string(envContent), key) {
			t.Errorf(".env.example missing %s:\n%s", key, string(envContent))
		}
	}

	// Verifica dependências em go.mod
	goModContent, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("Read go.mod error = %v", err)
	}
	if !strings.Contains(string(goModContent), "github.com/golang-jwt/jwt/v5") {
		t.Errorf("go.mod missing jwt: %s", string(goModContent))
	}

	// Verifica manifesto atualizado
	mUpdated, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		t.Fatalf("manifest.Read error = %v", err)
	}
	if !mUpdated.HasAuth() {
		t.Fatalf("manifest.HasAuth() = false, want true")
	}
	if mUpdated.Auth.Strategy != "jwt" {
		t.Errorf("manifest.Auth.Strategy = %q, want jwt", mUpdated.Auth.Strategy)
	}
	if mUpdated.Auth.UserEntity != "users" {
		t.Errorf("manifest.Auth.UserEntity = %q, want users", mUpdated.Auth.UserEntity)
	}
	if !slices.Equal(mUpdated.Auth.Roles, []string{"admin", "customer"}) {
		t.Errorf("manifest.Auth.Roles = %v, want [admin, customer]", mUpdated.Auth.Roles)
	}
	if !mUpdated.Auth.HasRefresh {
		t.Errorf("manifest.Auth.HasRefresh = false, want true")
	}
	if mUpdated.Auth.Hasher != "argon2id" {
		t.Errorf("manifest.Auth.Hasher = %q, want argon2id", mUpdated.Auth.Hasher)
	}
	if mUpdated.Auth.MiddlewareIdentifier != "RequireAuth" {
		t.Errorf("manifest.Auth.MiddlewareIdentifier = %q, want RequireAuth", mUpdated.Auth.MiddlewareIdentifier)
	}

	// Tentar adicionar auth de novo deve falhar
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"add", "auth", "jwt"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run(add auth) second time should fail, got code = 0")
	}
	if !strings.Contains(stderr.String(), "já possui autenticação") {
		t.Fatalf("Run(add auth) second time stderr = %q", stderr.String())
	}
}

func TestAddAuthPython(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "my_py_auth_app",
		Language:     "python",
		Framework:    "fastapi",
		Architecture: "layered",
	}

	root, _ := generateProject(t, cfg)
	chdirTemp(t, root)

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"add", "auth", "jwt", "--roles", "admin,user", "--refresh-token"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add auth jwt) exit code = %d, stderr = %s", code, stderr.String())
	}

	expectedFiles := []string{
		"src/app/auth/models/user.py",
		"src/app/auth/schemas/auth.py",
		"src/app/auth/security/security.py",
		"src/app/auth/handlers/auth.py",
		"src/app/auth/middlewares/auth.py",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	checkPythonSyntax(t, root)

	reqs, err := os.ReadFile(filepath.Join(root, "requirements.txt"))
	if err != nil {
		t.Fatalf("Read requirements.txt error = %v", err)
	}
	if !strings.Contains(string(reqs), "pyjwt") {
		t.Errorf("requirements.txt missing pyjwt: %s", string(reqs))
	}

	mUpdated, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		t.Fatalf("manifest.Read error = %v", err)
	}
	if mUpdated.Auth.MiddlewareIdentifier != "get_current_user" {
		t.Errorf("manifest.Auth.MiddlewareIdentifier = %q, want get_current_user", mUpdated.Auth.MiddlewareIdentifier)
	}
}

func TestAddAuthNode(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "my_node_auth_app",
		Language:     "node",
		Runtime:      "ts",
		Framework:    "express",
		Architecture: "layered",
	}

	root, _ := generateProject(t, cfg)
	chdirTemp(t, root)

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"add", "auth", "jwt"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add auth jwt) exit code = %d, stderr = %s", code, stderr.String())
	}

	expectedFiles := []string{
		"src/auth/models/user.ts",
		"src/auth/schemas/auth.ts",
		"src/auth/security/security.ts",
		"src/auth/handlers/auth.ts",
		"src/auth/middlewares/auth.ts",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	pkgJSON, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		t.Fatalf("Read package.json error = %v", err)
	}
	if !strings.Contains(string(pkgJSON), "jsonwebtoken") || !strings.Contains(string(pkgJSON), "bcrypt") {
		t.Errorf("package.json missing jsonwebtoken or bcrypt: %s", string(pkgJSON))
	}
}

func TestAddEndpointWithAuthFlags(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "ecommerce_api",
		Language:     "go",
		Framework:    "gin",
		Architecture: "layered",
		Database:     "postgres",
	}

	root, _ := generateProject(t, cfg)
	chdirTemp(t, root)

	// Sem autenticação configurada, flags --protected, --roles, --owner devem falhar
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"add", "endpoint", "order", "total:decimal", "status:str", "--protected"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run(add endpoint --protected) without auth should fail")
	}
	if !strings.Contains(stderr.String(), "não possui autenticação configurada") {
		t.Fatalf("stderr = %q, want missing auth error", stderr.String())
	}

	// Adiciona auth
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"add", "auth", "jwt"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add auth jwt) exit code = %d, stderr = %s", code, stderr.String())
	}

	// 1. Endpoint protegido (--protected)
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"add", "endpoint", "order", "total:decimal", "status:str", "--protected"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add endpoint order --protected) exit code = %d, stderr = %s", code, stderr.String())
	}

	// 2. Endpoint com restrição de papéis (--roles)
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"add", "endpoint", "audit_log", "action:str", "details:text", "--roles", "admin"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add endpoint audit_log --roles admin) exit code = %d, stderr = %s", code, stderr.String())
	}

	// 3. Endpoint owner-aware (--owner)
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"add", "endpoint", "profile", "bio:text", "avatar_url:str?", "--owner"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add endpoint profile --owner) exit code = %d, stderr = %s", code, stderr.String())
	}

	// Verifica manifesto
	mUpdated, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		t.Fatalf("manifest.Read error = %v", err)
	}

	if !mUpdated.IsProtectedEndpoint("order") {
		t.Errorf("order endpoint not marked protected in manifest: %v", mUpdated.ProtectedEndpoints)
	}
	if !mUpdated.IsProtectedEndpoint("audit_log") {
		t.Errorf("audit_log endpoint not marked protected in manifest: %v", mUpdated.ProtectedEndpoints)
	}
	if !slices.Equal(mUpdated.EndpointRoles["audit_log"], []string{"admin"}) {
		t.Errorf("audit_log endpoint roles = %v, want [admin]", mUpdated.EndpointRoles["audit_log"])
	}
	if !mUpdated.IsOwnerEndpoint("profile") {
		t.Errorf("profile endpoint not marked owner in manifest: %v", mUpdated.OwnerEndpoints)
	}
	// profile deve conter user:ref automaticamente
	profileFields := mUpdated.EndpointFields["profile"]
	if !slices.Contains(profileFields, "user:ref") {
		t.Errorf("profile fields = %v, want to contain user:ref", profileFields)
	}

	// Verifica remoção de endpoint limpa metadados de auth
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"remove", "endpoint", "audit_log"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(remove endpoint audit_log) exit code = %d, stderr = %s", code, stderr.String())
	}
	mClean, _ := manifest.Read(filepath.Join(root, manifest.FileName))
	if mClean.IsProtectedEndpoint("audit_log") {
		t.Errorf("audit_log still in ProtectedEndpoints after removal")
	}
	if _, exists := mClean.EndpointRoles["audit_log"]; exists {
		t.Errorf("audit_log still in EndpointRoles after removal")
	}
}

func TestInfoWithAuth(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "info_auth_app",
		Language:     "python",
		Framework:    "fastapi",
		Architecture: "layered",
	}

	root, _ := generateProject(t, cfg)
	chdirTemp(t, root)

	cli.Run([]string{"add", "auth", "jwt", "--roles", "admin,user"}, io.Discard, io.Discard)

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"info"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(info) exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Autenticação: jwt") {
		t.Errorf("info output missing 'Autenticação: jwt':\n%s", stdout.String())
	}

	// Teste com --json
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"info", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(info --json) exit code = %d, stderr = %s", code, stderr.String())
	}
	var data map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		t.Fatalf("Unmarshal info --json error = %v", err)
	}
	authObj, ok := data["auth"].(map[string]any)
	if !ok || authObj["strategy"] != "jwt" {
		t.Errorf("info --json auth = %v, want strategy jwt", data["auth"])
	}
}

func TestAddAuthJSON(t *testing.T) {
	root, _ := generateProject(t, project.Config{
		ProjectName:  "json_auth_app",
		Language:     "go",
		Framework:    "chi",
		Architecture: "layered",
	})
	chdirTemp(t, root)

	var res cliSummary
	runJSON(t, &res, "add", "auth", "jwt", "--json")
	if res.Command != "add auth" {
		t.Errorf("res.Command = %q, want 'add auth'", res.Command)
	}
	if !slices.Contains(res.Created, "internal/auth/models/user.go") {
		t.Errorf("res.Created missing auth model: %v", res.Created)
	}
	if !slices.Contains(res.Routes, "/auth/login") {
		t.Errorf("res.Routes missing /auth/login: %v", res.Routes)
	}
}

