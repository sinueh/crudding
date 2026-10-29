package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

// assertStructureInSync confere que directories e files do project.crud são
// exatamente o que um "crudding init" listaria agora.
func assertStructureInSync(t *testing.T, root string) manifest.Manifest {
	t.Helper()
	_, m, err := manifest.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	directories, files, err := common.ListStructure(os.DirFS(root), m.Ignore)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Directories, directories) || !slices.Equal(m.Files, files) {
		t.Errorf("estrutura do project.crud fora de sincronia:\n  directories %v\n  want        %v\n  files %v\n  want  %v", m.Directories, directories, m.Files, files)
	}
	return m
}

func generateFromExample(t *testing.T, name string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), name)
	if err := generator.GenerateFromTemplate(destination, name, copyExampleTemplate(t)); err != nil {
		t.Fatal(err)
	}
	return destination
}

func TestCustomEndpointAddAndRemove(t *testing.T) {
	root := generateFromExample(t, "loja")
	original := readFile(t, root, "cmd/api/main.go")

	for _, name := range []string{"order_item", "invoice"} {
		_, m, err := manifest.Find(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := generator.CheckEndpointSupport(m); err != nil {
			t.Fatalf("CheckEndpointSupport() error = %v", err)
		}
		result, err := generator.AddEndpoint(root, m, name)
		if err != nil {
			t.Fatalf("AddEndpoint(%s) error = %v", name, err)
		}
		if !slices.Equal(result.Created, []string{"internal/" + name + "/handler.go"}) || !slices.Equal(result.Modified, []string{"cmd/api/main.go"}) {
			t.Fatalf("AddEndpoint(%s) = %+v", name, result)
		}
	}

	handler := readFile(t, root, "internal/order_item/handler.go")
	for _, snippet := range []string{"package order_item", "type OrderItem struct", "order_items []OrderItem", `"GET /order_items"`} {
		if !strings.Contains(handler, snippet) {
			t.Errorf("handler.go não contém %q:\n%s", snippet, handler)
		}
	}
	// invoice copia só as linhas do exemplo item, não as de order_item, que
	// também contém "item".
	main := readFile(t, root, "cmd/api/main.go")
	for _, snippet := range []string{"\"loja/internal/item\"\n\t\"loja/internal/invoice\"\n\t\"loja/internal/order_item\"\n", "item.Register(mux)\n\tinvoice.Register(mux)\n\torder_item.Register(mux)\n"} {
		if !strings.Contains(main, snippet) {
			t.Errorf("main.go não contém %q:\n%s", snippet, main)
		}
	}
	if strings.Count(main, "invoice.Register") != 1 {
		t.Errorf("invoice registrado mais de uma vez:\n%s", main)
	}
	m := assertStructureInSync(t, root)
	if !slices.Equal(m.Endpoints, []string{"item", "order_item", "invoice"}) {
		t.Errorf("Endpoints = %v", m.Endpoints)
	}

	if goBin, err := exec.LookPath("go"); err == nil {
		command := exec.Command(goBin, "build", "./...")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("projeto com endpoints novos não compila: %v\n%s", err, output)
		}
	}

	if _, err := generator.AddEndpoint(root, m, "invoice"); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Errorf("AddEndpoint repetido: err = %v", err)
	}

	for _, name := range []string{"invoice", "order_item"} {
		_, m, err := manifest.Find(root)
		if err != nil {
			t.Fatal(err)
		}
		result, err := generator.RemoveEndpoint(root, m, name)
		if err != nil {
			t.Fatalf("RemoveEndpoint(%s) error = %v", name, err)
		}
		if !slices.Equal(result.Removed, []string{"internal/" + name + "/handler.go"}) {
			t.Errorf("RemoveEndpoint(%s) = %+v", name, result)
		}
	}
	if got := readFile(t, root, "cmd/api/main.go"); got != original {
		t.Errorf("main.go não voltou ao original:\n%s", got)
	}
	assertFilesAbsent(t, root, "internal/invoice", "internal/order_item")
	if m := assertStructureInSync(t, root); !slices.Equal(m.Endpoints, []string{"item"}) {
		t.Errorf("Endpoints = %v", m.Endpoints)
	}
}

func TestCustomEndpointRejectsInvalidSpecs(t *testing.T) {
	root := generateFromExample(t, "loja")
	_, base, err := manifest.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	mainBefore := readFile(t, root, "cmd/api/main.go")

	testCases := []struct {
		name string
		spec manifest.EndpointSpec
		add  string
		want string
	}{
		{"name inválido", manifest.EndpointSpec{Name: "Item", Files: []string{"internal/item/handler.go"}}, "invoice", "name"},
		{"sem files", manifest.EndpointSpec{Name: "item", Register: []string{"cmd/api/main.go"}}, "invoice", "files"},
		{"o próprio exemplo", base.TemplateEndpoint, "item", "endpoint de exemplo"},
		{"arquivo sem o nome", manifest.EndpointSpec{Name: "item", Files: []string{"go.mod"}}, "invoice", "mesmo caminho"},
		{"arquivo inexistente", manifest.EndpointSpec{Name: "item", Files: []string{"internal/item/nada.go"}}, "invoice", "nada.go"},
		{"fora do projeto", manifest.EndpointSpec{Name: "item", Files: []string{"../item.go"}}, "invoice", "inválido"},
		{"register sem o exemplo", manifest.EndpointSpec{Name: "item", Files: []string{"internal/item/handler.go"}, Register: []string{"go.mod"}}, "invoice", "nenhuma linha"},
		{"nome novo inválido", base.TemplateEndpoint, "../x", "inválido"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			m := base
			m.TemplateEndpoint = testCase.spec
			if _, err := generator.AddEndpoint(root, m, testCase.add); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("AddEndpoint() err = %v, want contendo %q", err, testCase.want)
			}
			assertFilesAbsent(t, root, "internal/invoice")
			if got := readFile(t, root, "cmd/api/main.go"); got != mainBefore {
				t.Errorf("main.go alterado apesar do erro:\n%s", got)
			}
		})
	}

	if _, err := generator.RemoveEndpoint(root, base, "item"); err == nil || !strings.Contains(err.Error(), "endpoint de exemplo") {
		t.Errorf("RemoveEndpoint(exemplo) err = %v", err)
	}
	assertFilesExist(t, root, "internal/item/handler.go")
}

// O campo template_endpoint preenchido vale também em projetos de arquitetura embutida,
// no lugar do layout da arquitetura.
func TestCustomEndpointOverridesBuiltinLayout(t *testing.T) {
	root, m := generateProject(t, project.Config{ProjectName: "Sample", Language: "go", Framework: "gin", Architecture: "layered"})
	m.TemplateEndpoint = manifest.EndpointSpec{
		Name:     "user",
		Files:    []string{"internal/handlers/user.go", "internal/service/user.go"},
		Register: []string{"internal/routes/routes.go"},
	}
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}
	result, err := generator.AddEndpoint(root, m, "invoice")
	if err != nil {
		t.Fatalf("AddEndpoint() error = %v", err)
	}
	if !slices.Equal(result.Created, []string{"internal/handlers/invoice.go", "internal/service/invoice.go"}) {
		t.Errorf("AddEndpoint() = %+v", result)
	}
	assertFileContains(t, root, "internal/routes/routes.go", "invoice")
	assertStructureInSync(t, root)
}

func TestBuiltinCommandsKeepStructureInSync(t *testing.T) {
	root, m := generateProject(t, project.Config{ProjectName: "Sample", Language: "python", Framework: "fastapi", Architecture: "modular"})
	if _, err := generator.AddEndpoint(root, m, "invoice"); err != nil {
		t.Fatal(err)
	}
	m = assertStructureInSync(t, root)
	if !slices.Contains(m.Directories, "src/app/invoice") {
		t.Errorf("Directories = %v", m.Directories)
	}
	if _, err := generator.RemoveEndpoint(root, m, "invoice"); err != nil {
		t.Fatal(err)
	}
	assertStructureInSync(t, root)

	goRoot, goManifest := generateProject(t, project.Config{ProjectName: "Sample", Language: "go", Framework: "gin", Architecture: "layered"})
	if _, err := generator.AddDatabase(goRoot, goManifest, "postgres", templates); err != nil {
		t.Fatal(err)
	}
	assertStructureInSync(t, goRoot)
}
