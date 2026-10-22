package test

import (
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sinueh-andreata/crudding/internal/generator"
	"github.com/sinueh-andreata/crudding/internal/manifest"
	"github.com/sinueh-andreata/crudding/internal/project"
)

func generateProject(t *testing.T, config project.Config) (string, manifest.Manifest) {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "Sample")
	if err := generator.Generate(destination, config, templates); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	root, m, err := manifest.Find(destination)
	if err != nil {
		t.Fatalf("manifest.Find() error = %v", err)
	}
	return root, m
}

// readTree lê todos os arquivos de root (caminho relativo -> conteúdo).
func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(root, func(target string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, target)
		tree[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// checkPythonSyntax analisa todos os .py sem gravar __pycache__ no projeto.
func checkPythonSyntax(t *testing.T, root string) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		return
	}
	script := `import ast, pathlib, sys
for path in pathlib.Path(sys.argv[1]).rglob("*.py"):
    ast.parse(path.read_text(), str(path))`
	if output, err := exec.Command(python, "-c", script, root).CombinedOutput(); err != nil {
		t.Fatalf("Python inválido: %v\n%s", err, output)
	}
}

// O add endpoint só usa o campo endpoint do project.crud. Em toda combinação
// Python e Go, o projeto gerado já traz o campo apontando para o endpoint de
// exemplo; adicionar e remover endpoints devolve o projeto, byte a byte, ao
// estado inicial.
func TestEndpointRoundTripAllCombinations(t *testing.T) {
	goBin, goErr := exec.LookPath("go")
	for _, config := range allConfigs() {
		if config.Language != "python" && config.Language != "go" {
			continue
		}
		t.Run(config.Language+"/"+config.Framework+"/"+config.Architecture, func(t *testing.T) {
			root, m := generateProject(t, config)
			if !m.Endpoint.Defined() || !slices.Contains(m.Endpoints, m.Endpoint.Name) {
				t.Fatalf("campo endpoint não preenchido na geração: %+v", m.Endpoint)
			}
			assertFilesExist(t, root, append(slices.Clone(m.Endpoint.Files), m.Endpoint.Register...)...)
			initial := readTree(t, root)

			for _, name := range []string{"order_item", "invoice"} {
				result, err := generator.AddEndpoint(root, m, name)
				if err != nil {
					t.Fatalf("AddEndpoint(%s) error = %v", name, err)
				}
				if len(result.Created) != len(m.Endpoint.Files) || !slices.Equal(result.Modified, m.Endpoint.Register) {
					t.Fatalf("AddEndpoint(%s) = %+v", name, result)
				}
				for _, register := range m.Endpoint.Register {
					assertFileContains(t, root, register, name)
				}
			}
			after := assertStructureInSync(t, root)
			if !slices.Equal(after.Endpoints, []string{m.Endpoint.Name, "order_item", "invoice"}) {
				t.Errorf("Endpoints = %v", after.Endpoints)
			}

			switch {
			case config.Language == "python":
				checkPythonSyntax(t, root)
			case config.Framework == "raw" && goErr == nil:
				command := exec.Command(goBin, "test", "./...")
				command.Dir = root
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("projeto Go com endpoints novos não compila: %v\n%s", err, output)
				}
			}

			for _, name := range []string{"invoice", "order_item"} {
				if _, err := generator.RemoveEndpoint(root, m, name); err != nil {
					t.Fatalf("RemoveEndpoint(%s) error = %v", name, err)
				}
			}
			if final := readTree(t, root); !maps.Equal(final, initial) {
				for rel := range maps.Keys(final) {
					if final[rel] != initial[rel] {
						t.Errorf("%s mudou depois de add + remove:\n%s", rel, final[rel])
					}
				}
				for rel := range maps.Keys(initial) {
					if _, ok := final[rel]; !ok {
						t.Errorf("%s sumiu depois de add + remove", rel)
					}
				}
			}
		})
	}
}

// O registro do endpoint novo copia o formato do exemplo de cada framework.
func TestAddEndpointCopiesSampleRegistration(t *testing.T) {
	testCases := []struct {
		config   project.Config
		register string
		want     []string
	}{
		{
			project.Config{ProjectName: "Sample", Language: "python", Framework: "fastapi", Architecture: "modular"},
			"src/app/main.py",
			[]string{
				"from src.app.order_item.controller import router as order_item_router",
				`app.include_router(order_item_router, prefix="/order_item", tags=["OrderItem"])`,
			},
		},
		{
			project.Config{ProjectName: "Sample", Language: "go", Framework: "gin", Architecture: "hexagonal"},
			"internal/routes/routes.go",
			[]string{
				"orderItemRepo := postgres.NewOrderItemRepository()",
				`api.GET("/order_items", orderItemHandler.List)`,
			},
		},
	}
	for _, testCase := range testCases {
		root, m := generateProject(t, testCase.config)
		if _, err := generator.AddEndpoint(root, m, "order_item"); err != nil {
			t.Fatalf("AddEndpoint(%s) error = %v", testCase.config.Language, err)
		}
		for _, line := range testCase.want {
			assertFileContains(t, root, testCase.register, line)
		}
	}
}

func TestAddEndpointRejections(t *testing.T) {
	pythonRoot, pythonManifest := generateProject(t, project.Config{ProjectName: "Sample", Language: "python", Framework: "raw", Architecture: "layered"})
	goRoot, goManifest := generateProject(t, project.Config{ProjectName: "Sample", Language: "go", Framework: "raw", Architecture: "layered"})
	nodeRoot, nodeManifest := generateProject(t, project.Config{ProjectName: "Sample", Language: "node", Runtime: "ts", Framework: "express", Architecture: "modular"})

	for _, testCase := range []struct {
		root string
		m    manifest.Manifest
		name string
		want string
	}{
		{pythonRoot, pythonManifest, "class", "palavra reservada"},
		{goRoot, goManifest, "func", "palavra reservada"},
		{pythonRoot, pythonManifest, "resource", "endpoint de exemplo"},
		{nodeRoot, nodeManifest, "invoice", `campo "endpoint"`},
	} {
		if _, err := generator.AddEndpoint(testCase.root, testCase.m, testCase.name); err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Errorf("AddEndpoint(%s, %s) err = %v, want contendo %q", testCase.m.Language, testCase.name, err, testCase.want)
		}
	}

	if _, err := generator.AddEndpoint(pythonRoot, pythonManifest, "invoice"); err != nil {
		t.Fatal(err)
	}
	before := readTree(t, pythonRoot)
	if _, err := generator.AddEndpoint(pythonRoot, pythonManifest, "invoice"); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Errorf("AddEndpoint repetido: err = %v", err)
	}
	if !maps.Equal(readTree(t, pythonRoot), before) {
		t.Error("AddEndpoint repetido alterou o projeto")
	}

	for _, testCase := range []struct {
		root string
		m    manifest.Manifest
		name string
		want string
	}{
		{pythonRoot, pythonManifest, "nonexistent", "não encontrado"},
		{pythonRoot, pythonManifest, "resource", "endpoint de exemplo"},
		{nodeRoot, nodeManifest, "invoice", `campo "endpoint"`},
	} {
		if _, err := generator.RemoveEndpoint(testCase.root, testCase.m, testCase.name); err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Errorf("RemoveEndpoint(%s, %s) err = %v, want contendo %q", testCase.m.Language, testCase.name, err, testCase.want)
		}
	}
}

// Sem nenhuma linha do exemplo no arquivo de register, nada é gravado.
func TestAddEndpointWithoutSampleRegistrationChangesNothing(t *testing.T) {
	root, m := generateProject(t, project.Config{ProjectName: "Sample", Language: "python", Framework: "raw", Architecture: "modular"})
	mainPath := filepath.Join(root, "src", "app", "main.py")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.Contains(line, "resource") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(mainPath, []byte(strings.Join(kept, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	before := readTree(t, root)

	if _, err := generator.AddEndpoint(root, m, "invoice"); err == nil || !strings.Contains(err.Error(), "nenhuma linha") {
		t.Fatalf("AddEndpoint() err = %v", err)
	}
	if !maps.Equal(readTree(t, root), before) {
		t.Error("projeto alterado apesar do erro")
	}
}
