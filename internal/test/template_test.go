package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"crudding/internal/generator"
	"crudding/internal/generator/common"
	"crudding/internal/generator/custom"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

// exampleTemplate é o projeto-template documentado no README.
var exampleTemplate = filepath.Join("..", "..", "examples", "go-template")

func copyExampleTemplate(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "go-template")
	if err := os.CopyFS(dir, os.DirFS(exampleTemplate)); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestTemplateFromGoExample(t *testing.T) {
	templateDir := copyExampleTemplate(t)
	destination := filepath.Join(t.TempDir(), "minha-api")
	if err := generator.GenerateFromTemplate(destination, "minha-api", templateDir); err != nil {
		t.Fatalf("GenerateFromTemplate() error = %v", err)
	}

	assertFileContains(t, destination, "go.mod", "module minha-api")
	assertFileContains(t, destination, "cmd/api/main.go", `"minha-api/internal/health"`)
	assertFileContains(t, destination, "README.md", "# minha-api")
	_, m, err := manifest.Find(destination)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "minha-api" || m.Template != "go-template" || m.Architecture != manifest.CustomArchitecture {
		t.Errorf("manifesto = %+v", m)
	}
	for _, content := range []string{readFile(t, destination, "go.mod"), readFile(t, destination, "cmd/api/main.go")} {
		if strings.Contains(content, "go-template") {
			t.Errorf("nome do template não foi trocado:\n%s", content)
		}
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go is not installed")
	}
	command := exec.Command(goBin, "build", "./...")
	command.Dir = destination
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("projeto gerado não compila: %v\n%s", err, output)
	}
}

// O project.crud de um projeto gerado pelo crudding já lista a estrutura,
// igual ao do "crudding init".
func TestGeneratedManifestListsStructure(t *testing.T) {
	root, m := generateProject(t, project.Config{ProjectName: "Sample", Language: "go", Framework: "gin", Architecture: "layered"})
	if len(m.Ignore) != 0 || !slices.Contains(m.Files, "go.mod") || !slices.Contains(m.Directories, "cmd") {
		t.Fatalf("manifesto = %+v", m)
	}
	directories, files, err := common.ListStructure(os.DirFS(root), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Directories, directories) || !slices.Equal(m.Files, files) {
		t.Errorf("estrutura do manifesto difere da gerada:\n%v\n%v\nwant\n%v\n%v", m.Directories, m.Files, directories, files)
	}
}

// Um projeto gerado pelo crudding já serve de template, sem "crudding init": o
// manifesto mantém a stack, então o add endpoint funciona no projeto novo.
func TestTemplateFromBuiltinProject(t *testing.T) {
	base := filepath.Join(t.TempDir(), "Sample")
	if err := generator.Generate(base, project.Config{ProjectName: "Sample", Language: "python", Framework: "fastapi", Architecture: "flat"}, templates); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "nova-api")
	if err := generator.GenerateFromTemplate(destination, "nova-api", base); err != nil {
		t.Fatalf("GenerateFromTemplate() error = %v", err)
	}
	root, generated, err := manifest.Find(destination)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Name != "nova-api" || generated.Template != "Sample" || generated.Framework != "fastapi" {
		t.Errorf("manifesto = %+v", generated)
	}
	result, err := generator.AddEndpoint(root, generated, "invoice")
	if err != nil {
		t.Fatalf("AddEndpoint() no projeto gerado do template: %v", err)
	}
	if len(result.Created) == 0 {
		t.Errorf("AddEndpoint() = %+v", result)
	}
}

func TestTemplateRejectsInvalidTemplates(t *testing.T) {
	templateDir := copyExampleTemplate(t)
	inside := filepath.Join(templateDir, "novo")
	if err := generator.GenerateFromTemplate(inside, "novo", templateDir); err == nil || !strings.Contains(err.Error(), "dentro da pasta do template") {
		t.Errorf("destino dentro do template: err = %v", err)
	}
	assertFilesAbsent(t, templateDir, "novo")

	withoutFiles := newTemplateDir(t, "vazio", map[string]string{manifest.FileName: `{"version": 1, "name": "vazio", "architecture": "custom"}`})
	withoutManifest := newTemplateDir(t, "sem-manifesto", map[string]string{"a.go": "package a\n"})
	for dir, want := range map[string]string{withoutFiles: "crudding init", withoutManifest: manifest.FileName} {
		destination := filepath.Join(t.TempDir(), "app")
		if err := generator.GenerateFromTemplate(destination, "app", dir); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("GenerateFromTemplate(%s) err = %v, want contendo %q", filepath.Base(dir), err, want)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Errorf("pasta do projeto criada apesar do erro: %v", err)
		}
	}
}

// Sem o campo endpoint preenchido não há como criar endpoints; add db não
// depende do campo e recusa a arquitetura custom.
func TestCustomArchitectureCommandsUnsupported(t *testing.T) {
	root := newTemplateDir(t, "app", map[string]string{"main.go": "package main\n"})
	m, err := custom.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := generator.CheckEndpointSupport(m); err == nil || !strings.Contains(err.Error(), `campo "endpoint"`) {
		t.Errorf("CheckEndpointSupport() err = %v", err)
	}
	if _, err := generator.AddEndpoint(root, m, "invoice"); err == nil || !strings.Contains(err.Error(), `campo "endpoint"`) {
		t.Errorf("AddEndpoint() err = %v", err)
	}
	if _, err := generator.RemoveEndpoint(root, m, "invoice"); err == nil || !strings.Contains(err.Error(), `campo "endpoint"`) {
		t.Errorf("RemoveEndpoint() err = %v", err)
	}
	if _, err := generator.AddDatabase(root, m, "postgres", templates); err == nil || !strings.Contains(err.Error(), "arquitetura custom") {
		t.Errorf("AddDatabase() err = %v", err)
	}
}
