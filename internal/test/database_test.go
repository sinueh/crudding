package test

import (
	"bytes"
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

func TestAddDatabaseGoPostgres(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "my_go_app",
		Language:     "go",
		Framework:    "chi",
		Architecture: "layered",
	}

	root, m := generateProject(t, cfg)

	result, err := generator.AddDatabase(root, m, "postgres", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase() error = %v", err)
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	if len(result.Created) != len(expectedFiles) {
		t.Errorf("result.Created = %v, want %v", result.Created, expectedFiles)
	}

	configContent, err := os.ReadFile(filepath.Join(root, "internal/config/config.go"))
	if err != nil {
		t.Fatalf("Read internal/config/config.go error = %v", err)
	}
	if !strings.Contains(string(configContent), "PostgresConfig") {
		t.Errorf("config.go não contém PostgresConfig: %s", string(configContent))
	}
	if !strings.Contains(string(configContent), "ConnectPostgres") {
		t.Errorf("config.go não contém ConnectPostgres: %s", string(configContent))
	}

	updated, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		t.Fatalf("Read updated manifest error = %v", err)
	}
	if updated.Database != "postgres" {
		t.Errorf("updated.Database = %q, want 'postgres'", updated.Database)
	}

	// Tentativa duplicada deve falhar
	if _, err := generator.AddDatabase(root, updated, "postgres", crudding.Templates); err == nil {
		t.Error("esperava erro ao adicionar banco já configurado, mas foi nil")
	}
}

func TestAddDatabaseGoFlatMySQL(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "flat_mysql_app",
		Language:     "go",
		Framework:    "raw",
		Architecture: "flat",
	}

	root, m := generateProject(t, cfg)

	result, err := generator.AddDatabase(root, m, "mysql", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase(mysql) error = %v", err)
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	if len(result.Created) != len(expectedFiles) {
		t.Errorf("result.Created = %v, want %v", result.Created, expectedFiles)
	}

	configContent, err := os.ReadFile(filepath.Join(root, "config/config.go"))
	if err != nil {
		t.Fatalf("Read config/config.go error = %v", err)
	}
	if !strings.Contains(string(configContent), "MySQLConfig") {
		t.Errorf("config.go não contém MySQLConfig: %s", string(configContent))
	}
}

func TestAddDatabasePythonMongo(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "py_mongo_app",
		Language:     "python",
		Framework:    "fastapi",
		Architecture: "modular",
	}

	root, m := generateProject(t, cfg)

	result, err := generator.AddDatabase(root, m, "mongo", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase(mongo) error = %v", err)
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.js",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	configContent, err := os.ReadFile(filepath.Join(root, "src/app/config/config.py"))
	if err != nil {
		t.Fatalf("Read src/app/config/config.py error = %v", err)
	}
	if !strings.Contains(string(configContent), "MONGODB_URI") {
		t.Errorf("config.py não contém MONGODB_URI: %s", string(configContent))
	}

	// requirements.txt deve ter sido atualizado com pymongo
	reqContent, err := os.ReadFile(filepath.Join(root, "requirements.txt"))
	if err != nil {
		t.Fatalf("Read requirements.txt error = %v", err)
	}
	if !strings.Contains(string(reqContent), "pymongo") {
		t.Errorf("requirements.txt não contém pymongo: %s", string(reqContent))
	}

	if !slices.Contains(result.Modified, "requirements.txt") {
		t.Errorf("result.Modified = %v, esperado conter requirements.txt", result.Modified)
	}
}

func TestAddDatabaseNodePostgresTS(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "node_pg_app",
		Language:     "node",
		Runtime:      "ts",
		Framework:    "fastify",
		Architecture: "hexagonal",
	}

	root, m := generateProject(t, cfg)

	result, err := generator.AddDatabase(root, m, "postgres", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase() error = %v", err)
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
		"src/database.ts",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	pkgContent, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		t.Fatalf("Read package.json error = %v", err)
	}
	pkgStr := string(pkgContent)
	if !strings.Contains(pkgStr, `"pg"`) {
		t.Errorf("package.json não contém \"pg\": %s", pkgStr)
	}
	if !strings.Contains(pkgStr, `"@types/pg"`) {
		t.Errorf("package.json não contém \"@types/pg\": %s", pkgStr)
	}

	if !slices.Contains(result.Modified, "package.json") {
		t.Errorf("result.Modified = %v, esperado conter package.json", result.Modified)
	}
}

func TestAddDatabaseCLI(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "cli_db_app",
		Language:     "go",
		Framework:    "gin",
		Architecture: "modular",
	}
	root, _ := generateProject(t, cfg)

	chdirTemp(t, root)

	// Teste com banco inválido
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"add", "db", "redis"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("Run(add db redis) code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "não suportado") {
		t.Errorf("stderr = %q, esperado mensagem de não suportado", stderr.String())
	}

	// Teste com add db postgres
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"add", "db", "postgres"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(add db postgres) code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Banco de dados postgres adicionado com sucesso") {
		t.Errorf("stdout = %q, missing success message", stdout.String())
	}

	// Teste info exibe o banco
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"info"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(info) code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Banco de dados: postgres") {
		t.Errorf("Run(info) stdout = %q, esperado conter 'Banco de dados: postgres'", stdout.String())
	}
}

func TestAddDatabasePythonMySQL(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "py_mysql_app",
		Language:     "python",
		Framework:    "flask",
		Architecture: "layered",
	}

	root, m := generateProject(t, cfg)

	result, err := generator.AddDatabase(root, m, "mysql", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase(mysql) error = %v", err)
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	reqContent, err := os.ReadFile(filepath.Join(root, "requirements.txt"))
	if err != nil {
		t.Fatalf("Read requirements.txt error = %v", err)
	}
	if !strings.Contains(string(reqContent), "pymysql") {
		t.Errorf("requirements.txt não contém pymysql: %s", string(reqContent))
	}

	if !slices.Contains(result.Modified, "requirements.txt") {
		t.Errorf("result.Modified = %v, esperado conter requirements.txt", result.Modified)
	}
}

func TestAddDatabaseNodeMySQL(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "node_mysql_app",
		Language:     "node",
		Runtime:      "js",
		Framework:    "express",
		Architecture: "flat",
	}

	root, m := generateProject(t, cfg)

	result, err := generator.AddDatabase(root, m, "mysql", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase(mysql) error = %v", err)
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
		"src/database.js",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("arquivo esperado %s não foi criado: %v", file, err)
		}
	}

	pkgContent, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		t.Fatalf("Read package.json error = %v", err)
	}
	if !strings.Contains(string(pkgContent), `"mysql2"`) {
		t.Errorf("package.json não contém \"mysql2\": %s", string(pkgContent))
	}

	if !slices.Contains(result.Modified, "package.json") {
		t.Errorf("result.Modified = %v, esperado conter package.json", result.Modified)
	}
}

func TestNewProjectWithDatabaseGo(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	args := []string{"new", "n:go-db-app", "l:go", "f:gin", "a:layered", "d:postgres"}
	code := cli.Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(new with d:postgres) exit code = %d, stderr = %q", code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "go-db-app")
	assertFilesExist(t, projectDir,
		"internal/config/config.go",
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.up.sql",
		"migrations/000001_init.down.sql",
	)

	configContent, err := os.ReadFile(filepath.Join(projectDir, "internal/config/config.go"))
	if err != nil {
		t.Fatalf("read config.go error = %v", err)
	}
	if count := strings.Count(string(configContent), "type PostgresConfig struct"); count != 1 {
		t.Errorf("config.go tem %d PostgresConfig, want 1: %s", count, string(configContent))
	}

	goModContent, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod error = %v", err)
	}
	if !strings.Contains(string(goModContent), "github.com/lib/pq") {
		t.Errorf("go.mod missing github.com/lib/pq: %s", string(goModContent))
	}

	goSumContent, err := os.ReadFile(filepath.Join(projectDir, "go.sum"))
	if err != nil {
		t.Fatalf("read go.sum error = %v", err)
	}
	if !strings.Contains(string(goSumContent), "github.com/lib/pq v1.10.9 h1:") {
		t.Errorf("go.sum missing github.com/lib/pq entry: %s", string(goSumContent))
	}

	mainContent, err := os.ReadFile(filepath.Join(projectDir, "cmd/api/main.go"))
	if err != nil {
		t.Fatalf("read cmd/api/main.go error = %v", err)
	}
	if !strings.Contains(string(mainContent), "config.ConnectPostgres(cfg.Postgres)") {
		t.Errorf("main.go missing ConnectPostgres call: %s", string(mainContent))
	}
	if !strings.Contains(string(mainContent), "routes.SetupRouter(db)") {
		t.Errorf("main.go missing routes.SetupRouter(db): %s", string(mainContent))
	}
}

func TestNewProjectWithDatabasePython(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	args := []string{"new", "n:py-db-app", "l:python", "f:fastapi", "a:modular", "db:mongo"}
	code := cli.Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(new with db:mongo) exit code = %d, stderr = %q", code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "py-db-app")
	assertFilesExist(t, projectDir,
		"src/app/config/config.py",
		"docker-compose.yml",
		".env.example",
		"migrations/000001_init.js",
	)

	configContent, err := os.ReadFile(filepath.Join(projectDir, "src/app/config/config.py"))
	if err != nil {
		t.Fatalf("read config.py error = %v", err)
	}
	if count := strings.Count(string(configContent), "def get_mongo_database"); count != 1 {
		t.Errorf("config.py tem %d get_mongo_database, want 1: %s", count, string(configContent))
	}

	reqContent, err := os.ReadFile(filepath.Join(projectDir, "requirements.txt"))
	if err != nil {
		t.Fatalf("read requirements.txt error = %v", err)
	}
	if !strings.Contains(string(reqContent), "pymongo") {
		t.Errorf("requirements.txt missing pymongo: %s", string(reqContent))
	}
}

func TestAddMultipleDatabases(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "multi_db_app",
		Language:     "go",
		Framework:    "chi",
		Architecture: "modular",
		Database:     "postgres",
	}

	root, m := generateProject(t, cfg)

	// Já tem postgres gerado upfront, agora adiciona mongo via AddDatabase
	result, err := generator.AddDatabase(root, m, "mongo", crudding.Templates)
	if err != nil {
		t.Fatalf("AddDatabase(mongo) on postgres project error = %v", err)
	}

	// docker-compose.yml e .env.example foram modificados (não criados do zero)
	if !slices.Contains(result.Modified, "docker-compose.yml") {
		t.Errorf("result.Modified = %v, want docker-compose.yml", result.Modified)
	}
	if !slices.Contains(result.Modified, ".env.example") {
		t.Errorf("result.Modified = %v, want .env.example", result.Modified)
	}

	configContent, err := os.ReadFile(filepath.Join(root, "internal/config/config.go"))
	if err != nil {
		t.Fatalf("read config.go error = %v", err)
	}
	contentStr := string(configContent)
	if !strings.Contains(contentStr, "PostgresConfig") {
		t.Errorf("config.go missing PostgresConfig")
	}
	if !strings.Contains(contentStr, "MongoConfig") {
		t.Errorf("config.go missing MongoConfig: %s", contentStr)
	}
	if count := strings.Count(contentStr, "\t\"fmt\"\n"); count != 1 {
		t.Errorf("config.go importa fmt %d vezes, want 1: %s", count, contentStr)
	}
}
