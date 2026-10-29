package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

func TestWriteAndFindFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	config := project.Config{ProjectName: "demo", Language: "python", Framework: "fastapi", Architecture: "modular"}
	if err := manifest.Write(root, manifest.New(config)); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src", "app")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	found, m, err := manifest.Find(nested)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if found != root {
		t.Errorf("Find() root = %q, want %q", found, root)
	}
	if m.Config() != config {
		t.Errorf("Config() = %+v, want %+v", m.Config(), config)
	}
	if got := m.Stack(); got != "python + fastapi + modular" {
		t.Errorf("Stack() = %q", got)
	}
}

func TestFindErrors(t *testing.T) {
	if _, _, err := manifest.Find(t.TempDir()); err == nil || !strings.Contains(err.Error(), manifest.FileName) {
		t.Errorf("Find() sem manifesto: err = %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, manifest.FileName), []byte(`{"version": 99}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manifest.Find(root); err == nil || !strings.Contains(err.Error(), "versão") {
		t.Errorf("Find() versão inválida: err = %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, manifest.FileName), []byte(`not json`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manifest.Find(root); err == nil {
		t.Error("Find() JSON inválido deveria falhar")
	}
}

func TestManifestEndpoints(t *testing.T) {
	m := manifest.New(project.Config{ProjectName: "app", Language: "go", Framework: "gin", Architecture: "flat"}, "user")
	if !m.HasEndpoint("user") {
		t.Error("HasEndpoint(user) = false, want true")
	}
	if m.HasEndpoint("customer") {
		t.Error("HasEndpoint(customer) = true, want false")
	}

	m.AddEndpoint("customer")
	if !m.HasEndpoint("customer") || len(m.Endpoints) != 2 {
		t.Fatalf("AddEndpoint(customer) falhou: endpoints = %v", m.Endpoints)
	}

	// Não duplica
	m.AddEndpoint("customer")
	if len(m.Endpoints) != 2 {
		t.Fatalf("AddEndpoint duplicado alterou tamanho: %v", m.Endpoints)
	}

	if !m.RemoveEndpoint("customer") {
		t.Error("RemoveEndpoint(customer) = false, want true")
	}
	if m.HasEndpoint("customer") || len(m.Endpoints) != 1 {
		t.Fatalf("RemoveEndpoint falhou em remover: %v", m.Endpoints)
	}

	if m.RemoveEndpoint("nonexistent") {
		t.Error("RemoveEndpoint(nonexistent) = true, want false")
	}
}

func TestManifestWithORM(t *testing.T) {
	cfg := project.Config{
		ProjectName:  "go_orm_app",
		Language:     "go",
		Framework:    "gin",
		Architecture: "hexagonal",
		Database:     "postgres",
		ORM:          "gorm",
	}
	m := manifest.New(cfg, "user")
	if m.ORM != "gorm" {
		t.Errorf("m.ORM = %q, want gorm", m.ORM)
	}
	if m.Config().ORM != "gorm" {
		t.Errorf("m.Config().ORM = %q, want gorm", m.Config().ORM)
	}
	if got := m.Stack(); got != "go + gin + hexagonal + postgres + gorm" {
		t.Errorf("m.Stack() = %q, want %q", got, "go + gin + hexagonal + postgres + gorm")
	}
}

func TestManifestTemplateFields(t *testing.T) {
	root := t.TempDir()
	m := manifest.Manifest{
		Version:      manifest.Version,
		Name:         "minha-api",
		Architecture: manifest.CustomArchitecture,
		Template:     "go-template",
		Ignore:       []string{".git", "*.log"},
		Directories:  []string{"cmd"},
		Files:        []string{"cmd/main.go", "go.mod"},
	}
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}
	read, err := manifest.ReadFS(os.DirFS(root))
	if err != nil {
		t.Fatalf("ReadFS() error = %v", err)
	}
	if read.Template != m.Template || strings.Join(read.Ignore, ",") != ".git,*.log" || strings.Join(read.Directories, ",") != "cmd" || strings.Join(read.Files, ",") != "cmd/main.go,go.mod" {
		t.Errorf("ReadFS() = %+v", read)
	}
	if got := read.Stack(); got != "custom" {
		t.Errorf("Stack() sem linguagem e framework = %q, want %q", got, "custom")
	}
	read.Language = "go"
	if got := read.Stack(); got != "go + custom" {
		t.Errorf("Stack() = %q", got)
	}

	// Todo project.crud segue o mesmo padrão: as listas aparecem mesmo vazias,
	// e "template" só existe em projetos gerados de um template.
	if err := manifest.Write(root, manifest.New(project.Config{ProjectName: "demo", Language: "go", Framework: "raw", Architecture: "flat"})); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, manifest.FileName))
	for _, key := range []string{`"endpoints": []`, `"template_endpoint": {`, `"name": ""`, `"register": []`, `"ignore": []`, `"directories": []`, `"files": []`} {
		if !strings.Contains(string(content), key) {
			t.Errorf("project.crud não contém %s:\n%s", key, content)
		}
	}
	if strings.Contains(string(content), `"template"`) {
		t.Errorf("project.crud embutido contém \"template\":\n%s", content)
	}
}
