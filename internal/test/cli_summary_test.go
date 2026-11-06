package test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/cli"
)

type cliSummary struct {
	Command  string   `json:"command"`
	Project  string   `json:"project"`
	Created  []string `json:"created"`
	Modified []string `json:"modified"`
	Removed  []string `json:"removed"`
	Edit     []string `json:"edit"`
	Routes   []string `json:"routes"`
	Next     []string `json:"next"`
}

// runJSON roda o comando e decodifica o stdout, que deve ter só o objeto JSON.
func runJSON(t *testing.T, target any, args ...string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := cli.Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", args, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run(%v) stderr = %q, want vazio", args, stderr.String())
	}
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("Run(%v) stdout não é JSON: %v\n%s", args, err, stdout.String())
	}
	if rest := strings.TrimSpace(stdout.String()); rest != "" {
		t.Fatalf("Run(%v) stdout tem algo além do JSON: %q", args, rest)
	}
}

func TestCLISummaryJSON(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var created cliSummary
	runJSON(t, &created, "new", "n:shop", "l:python", "f:fastapi", "a:modular", "--json")
	projectDir := filepath.Join(tmp, "shop")
	if created.Command != "new" || created.Project != projectDir {
		t.Errorf("new: command/project = %q/%q", created.Command, created.Project)
	}
	for _, file := range []string{"project.crud", "src/app/main.py", "src/app/template/service.py", "requirements.txt"} {
		if !slices.Contains(created.Created, file) {
			t.Errorf("new: created sem %s: %v", file, created.Created)
		}
	}
	assertFilesExist(t, projectDir, created.Created...)
	if !slices.Equal(created.Routes, []string{"/template"}) {
		t.Errorf("new: routes = %v", created.Routes)
	}
	if len(created.Next) == 0 || created.Next[0] != "cd shop" || !slices.Contains(created.Next, "uvicorn src.app.main:app --reload") {
		t.Errorf("new: next = %v", created.Next)
	}

	chdirTemp(t, projectDir)
	var added cliSummary
	runJSON(t, &added, "add", "endpoint", "product", "json:true")
	wantCreated := []string{"src/app/product/controller.py", "src/app/product/service.py", "src/app/product/models.py", "src/app/product/__init__.py"}
	if added.Command != "add endpoint" || !slices.Equal(added.Created, wantCreated) {
		t.Errorf("add endpoint: %+v", added)
	}
	if !slices.Equal(added.Modified, []string{"src/app/main.py", "project.crud"}) {
		t.Errorf("add endpoint: modified = %v", added.Modified)
	}
	if !slices.Equal(added.Edit, []string{"src/app/product/service.py", "src/app/product/models.py"}) {
		t.Errorf("add endpoint: edit = %v", added.Edit)
	}
	if !slices.Equal(added.Routes, []string{"/product"}) || len(added.Removed) != 0 {
		t.Errorf("add endpoint: routes = %v, removed = %v", added.Routes, added.Removed)
	}

	var info struct {
		Name             string   `json:"name"`
		Project          string   `json:"project"`
		Language         string   `json:"language"`
		Framework        string   `json:"framework"`
		Architecture     string   `json:"architecture"`
		Databases        []string `json:"databases"`
		Endpoints        []string `json:"endpoints"`
		Routes           []string `json:"routes"`
		TemplateEndpoint struct {
			Name     string   `json:"name"`
			Files    []string `json:"files"`
			Register []string `json:"register"`
		} `json:"template_endpoint"`
	}
	runJSON(t, &info, "info", "--json")
	if info.Name != "shop" || info.Language != "python" || info.Framework != "fastapi" || info.Architecture != "modular" {
		t.Errorf("info: %+v", info)
	}
	if !slices.Equal(info.Endpoints, []string{"template", "product"}) || !slices.Equal(info.Routes, []string{"/template", "/product"}) {
		t.Errorf("info: endpoints = %v, routes = %v", info.Endpoints, info.Routes)
	}
	if info.TemplateEndpoint.Name != "template" || !slices.Equal(info.TemplateEndpoint.Register, []string{"src/app/main.py"}) || info.Databases == nil {
		t.Errorf("info: template_endpoint = %+v, databases = %v", info.TemplateEndpoint, info.Databases)
	}

	var db cliSummary
	runJSON(t, &db, "add", "db", "postgres", "--json")
	if db.Command != "add db" || !slices.Contains(db.Created, "docker-compose.yml") || !slices.Contains(db.Modified, "project.crud") {
		t.Errorf("add db: %+v", db)
	}
	if !slices.Equal(db.Next, []string{"docker compose up -d", "pip install -r requirements.txt"}) {
		t.Errorf("add db: next = %v", db.Next)
	}

	var removed cliSummary
	runJSON(t, &removed, "remove", "endpoint", "--json", "product")
	if removed.Command != "remove endpoint" || len(removed.Created) != 0 || len(removed.Edit) != 0 {
		t.Errorf("remove endpoint: %+v", removed)
	}
	slices.Sort(removed.Removed)
	if want := []string{"src/app/product/__init__.py", "src/app/product/controller.py", "src/app/product/models.py", "src/app/product/service.py"}; !slices.Equal(removed.Removed, want) {
		t.Errorf("remove endpoint: removed = %v, want %v", removed.Removed, want)
	}
	if !slices.Equal(removed.Routes, []string{"/product"}) {
		t.Errorf("remove endpoint: routes = %v", removed.Routes)
	}
	assertFilesAbsent(t, projectDir, "src/app/product/service.py")
}

func TestCLISummaryJSONErrors(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"new", "n:api", "l:go", "f:gin", "a:layered"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new) exit code = %d, stderr = %q", code, stderr.String())
	}
	chdirTemp(t, filepath.Join(tmp, "api"))

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"add", "endpoint", "--json"}, "--json exige o nome do endpoint"},
		{[]string{"remove", "endpoint", "--json"}, "--json exige o nome do endpoint"},
		{[]string{"add", "db", "--json"}, "--json exige o banco de dados"},
		{[]string{"add", "endpoint", "x", "json:talvez"}, "valor inválido para json"},
		{[]string{"add", "endpoint", "template", "--json"}, "endpoint de exemplo"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run(tc.args, &stdout, &stderr); code != 1 {
			t.Errorf("Run(%v) exit code = %d, want 1", tc.args, code)
		}
		if stdout.Len() != 0 {
			t.Errorf("Run(%v) stdout = %q, want vazio no erro", tc.args, stdout.String())
		}
		if !strings.Contains(stderr.String(), tc.want) {
			t.Errorf("Run(%v) stderr = %q, want contendo %q", tc.args, stderr.String(), tc.want)
		}
	}
}

func TestCLISummaryText(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"new", "n:api", "l:go", "f:gin", "a:layered"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new) exit code = %d, stderr = %q", code, stderr.String())
	}
	for _, want := range []string{
		"Projeto gerado em: " + filepath.Join(tmp, "api") + " (go + gin + layered)\n",
		"criado:   ",
		"internal/{handlers,models,service,types}/template.go",
		".crudding/ (interno)",
		"rota:     /api/v1/template (exemplo)\n",
		"próximo:  cd api && go mod tidy && go run ./cmd/api\n",
		"          crudding add endpoint <nome>\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("new stdout sem %q:\n%s", want, stdout.String())
		}
	}

	chdirTemp(t, filepath.Join(tmp, "api"))
	stdout.Reset()
	if code := cli.Run([]string{"add", "endpoint", "order_item"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(add endpoint) exit code = %d, stderr = %q", code, stderr.String())
	}
	want := "\nEndpoint order_item criado.\n" +
		"criado:   internal/{handlers,models,service,types}/order_item.go\n" +
		"alterado: project.crud, internal/routes/routes.go\n" +
		"editar:   internal/service/order_item.go (regras de negócio)\n" +
		"          internal/models/order_item.go (campos)\n" +
		"rota:     /api/v1/order_item\n" +
		"próximo:  go run ./cmd/api\n"
	if got := stdout.String(); !strings.HasSuffix(got, want) {
		t.Errorf("add endpoint stdout =\n%s\nwant sufixo\n%s", got, want)
	}

	stdout.Reset()
	if code := cli.Run([]string{"remove", "endpoint", "order_item"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(remove endpoint) exit code = %d, stderr = %q", code, stderr.String())
	}
	for _, want := range []string{
		"removido: internal/{handlers,models,service,types}/order_item.go\n",
		"rota:     /api/v1/order_item (removida)\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("remove endpoint stdout sem %q:\n%s", want, stdout.String())
		}
	}
}

func TestCLITemplateUseJSON(t *testing.T) {
	t.Setenv("CRUDDING_TEMPLATES_DIR", t.TempDir())
	templateDir := copyExampleTemplate(t)
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"add", "template", templateDir, "n:meu-template"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(add template) exit code = %d, stderr = %q", code, stderr.String())
	}

	var used cliSummary
	runJSON(t, &used, "template", "use", "meu-template", "n:minha-api", "--json")
	if used.Command != "template use" || used.Project != filepath.Join(tmp, "minha-api") {
		t.Errorf("template use: %+v", used)
	}
	if !slices.Contains(used.Created, "go.mod") || !slices.Contains(used.Created, "project.crud") {
		t.Errorf("template use: created = %v", used.Created)
	}
	assertFilesExist(t, used.Project, used.Created...)

	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"template", "use", "meu-template", "--json"}, &stdout, &stderr); code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "--json exige o nome do projeto") {
		t.Errorf("template use sem nome: code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}
