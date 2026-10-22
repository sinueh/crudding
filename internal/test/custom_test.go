package test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sinueh-andreata/crudding/internal/generator/custom"
	"github.com/sinueh-andreata/crudding/internal/manifest"
)

func newTemplateDir(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		writeFile(t, root, rel, content)
	}
	return root
}

func TestCustomInitStartsWithEmptyIgnore(t *testing.T) {
	root := newTemplateDir(t, "meu-template", map[string]string{
		".gitignore":          "node_modules\n",
		"go.mod":              "module meu-template\n",
		"node_modules/x/a.js": "x",
		".env":                "SECRET=1",
		".git/HEAD":           "ref: refs/heads/main\n",
	})
	if err := os.Mkdir(filepath.Join(root, "migrations"), 0755); err != nil {
		t.Fatal(err)
	}

	m, err := custom.Init(root)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if m.Name != "meu-template" || m.Architecture != manifest.CustomArchitecture || m.Language != "" {
		t.Errorf("manifesto = %+v", m)
	}
	// Nada entra no ignore automaticamente, nem o .gitignore; só .git é pulado.
	if len(m.Ignore) != 0 {
		t.Errorf("Ignore = %v, want vazio", m.Ignore)
	}
	if want := []string{".env", ".gitignore", "go.mod", "node_modules/x/a.js"}; !slices.Equal(m.Files, want) {
		t.Errorf("Files = %v, want %v", m.Files, want)
	}
	if want := []string{"migrations", "node_modules", "node_modules/x"}; !slices.Equal(m.Directories, want) {
		t.Errorf("Directories = %v, want %v", m.Directories, want)
	}

	content, _ := os.ReadFile(filepath.Join(root, manifest.FileName))
	for _, snippet := range []string{`"language": ""`, `"framework": ""`, `"architecture": "custom"`, `"endpoints": []`, `"ignore": []`} {
		if !strings.Contains(string(content), snippet) {
			t.Errorf("project.crud não contém %s:\n%s", snippet, content)
		}
	}
}

func TestCustomInitAppliesIgnorePatterns(t *testing.T) {
	root := newTemplateDir(t, "x", map[string]string{
		"cmd/api/main.go":      "package main\n",
		"cmd/bin/tool.go":      "package main\n", // /bin é ancorado na raiz
		"bin":                  "binário",
		"logs/a.txt":           "x",
		"src/tmp/cache":        "x",
		"app.log":              "x",
		".env":                 "SECRET=1",
		"node_modules/x/a.js":  "x",
		"web/__pycache__/a.pc": "x",
	})
	if _, err := custom.Init(root); err != nil {
		t.Fatal(err)
	}
	_, m, err := manifest.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	m.Ignore = []string{"/bin", "logs/", "**/tmp", "*.log", ".env", "node_modules", "__pycache__"}
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}

	m, err = custom.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"cmd/api/main.go", "cmd/bin/tool.go"}; !slices.Equal(m.Files, want) {
		t.Errorf("Files = %v, want %v", m.Files, want)
	}
	if want := []string{"cmd", "cmd/api", "cmd/bin", "src", "web"}; !slices.Equal(m.Directories, want) {
		t.Errorf("Directories = %v, want %v", m.Directories, want)
	}
}

func TestCustomInitRerunKeepsEditedFields(t *testing.T) {
	root := newTemplateDir(t, "base", map[string]string{"a.go": "package a\n", "README.md": "# base\n"})
	m, err := custom.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	m.Name, m.Language, m.Ignore = "minha-base", "go", []string{"*.md"}
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "b.go", "package a\n")

	m, err = custom.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "minha-base" || m.Language != "go" || !slices.Equal(m.Ignore, []string{"*.md"}) {
		t.Errorf("campos editados não foram mantidos: %+v", m)
	}
	if !slices.Equal(m.Files, []string{"a.go", "b.go"}) {
		t.Errorf("Files = %v", m.Files)
	}
}

func TestCustomInitRejectsSymlinksAndBadPatterns(t *testing.T) {
	root := newTemplateDir(t, "x", map[string]string{"a.go": "package a\n"})
	if err := os.Symlink("a.go", filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink indisponível: %v", err)
	}
	if _, err := custom.Init(root); err == nil || !strings.Contains(err.Error(), "ignore") {
		t.Errorf("Init() com symlink: err = %v", err)
	}

	m := manifest.Manifest{Version: manifest.Version, Name: "x", Architecture: manifest.CustomArchitecture, Ignore: []string{"link"}}
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}
	if _, err := custom.Init(root); err != nil {
		t.Errorf("Init() com symlink no ignore: err = %v", err)
	}

	m.Ignore = []string{"["}
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}
	if _, err := custom.Init(root); err == nil || !strings.Contains(err.Error(), "padrão de ignore inválido") {
		t.Errorf("Init() com padrão inválido: err = %v", err)
	}
}

func TestCustomPlanRenamesAndCopies(t *testing.T) {
	templates := fstest.MapFS{
		"README.md":               {Data: []byte("# Sample\nimport sample\n")},
		"src/sample/__init__.py":  {Data: []byte("")},
		"run-sample.sh":           {Data: []byte("echo sample\n"), Mode: 0755},
		"logo.bin":                {Data: []byte("sample\x00sample")},
		"migrations":              {Mode: fs.ModeDir | 0755},
		manifest.FileName:         {Data: []byte("{}")},
		"src/sample/extra.py":     {Data: []byte("x")},
		"src/sample/not-listed.p": {Data: []byte("x")},
	}
	m := manifest.Manifest{
		Version:      manifest.Version,
		Name:         "Sample",
		Language:     "python",
		Architecture: "flat",
		Endpoints:    []string{"invoice"},
		Directories:  []string{"migrations", "src", "src/sample"},
		Files:        []string{"README.md", "src/sample/__init__.py", "run-sample.sh", "logo.bin", "src/sample/extra.py"},
		Endpoint:     manifest.EndpointSpec{Name: "invoice", Files: []string{"src/sample/extra.py"}, Register: []string{"README.md"}},
	}
	plan, err := custom.Plan(templates, m, "nova-api")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	root := t.TempDir()
	if err := plan.Prepare(root); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	// "Sample" (forma exata) vira o nome digitado; "sample" (slug e snake de
	// uma palavra só) vira o snake_case, válido como pacote Python.
	for rel, want := range map[string]string{
		"README.md":                "# nova-api\nimport nova_api\n",
		"src/nova_api/__init__.py": "",
		"run-nova_api.sh":          "echo nova_api\n",
		"logo.bin":                 "sample\x00sample",
	} {
		if got := readFile(t, root, rel); got != want {
			t.Errorf("%s = %q, want %q", rel, got, want)
		}
	}
	if info, err := os.Stat(filepath.Join(root, "run-nova_api.sh")); err != nil || info.Mode().Perm()&0111 == 0 {
		t.Errorf("permissão de execução não preservada: %v %v", info.Mode(), err)
	}
	if info, err := os.Stat(filepath.Join(root, "migrations")); err != nil || !info.IsDir() {
		t.Errorf("pasta vazia não recriada: %v", err)
	}
	assertFilesAbsent(t, root, manifest.FileName, "src/nova_api/not-listed.p", "src/sample")

	generated := plan.Manifest
	if generated == nil || generated.Name != "nova-api" || generated.Template != "Sample" || generated.Architecture != "flat" || !slices.Equal(generated.Endpoints, []string{"invoice"}) {
		t.Fatalf("manifesto gerado = %+v", generated)
	}
	if generated.Endpoint.Name != "invoice" || !slices.Equal(generated.Endpoint.Files, []string{"src/nova_api/extra.py"}) || !slices.Equal(generated.Endpoint.Register, []string{"README.md"}) {
		t.Errorf("campo endpoint do manifesto gerado = %+v", generated.Endpoint)
	}
	if !slices.Contains(generated.Files, "src/nova_api/__init__.py") || !slices.Contains(generated.Directories, "src/nova_api") {
		t.Errorf("caminhos do manifesto gerado não foram renomeados: %+v", generated)
	}
}

func TestCustomPlanRenamesAllForms(t *testing.T) {
	templates := fstest.MapFS{"README.md": {Data: []byte("Meu Template / meu-template / meu_template\n")}}
	m := manifest.Manifest{Version: manifest.Version, Name: "Meu Template", Files: []string{"README.md"}}
	plan, err := custom.Plan(templates, m, "Minha API")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := plan.Prepare(root); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, root, "README.md"); got != "Minha API / minha-api / minha_api\n" {
		t.Errorf("README.md = %q", got)
	}
}

func TestCustomPlanRejectsInvalidTemplates(t *testing.T) {
	templates := fstest.MapFS{
		"a.txt": {Data: []byte("a")},
		"b.txt": {Data: []byte("b")},
		"link":  {Data: []byte("a.txt"), Mode: fs.ModeSymlink},
	}
	testCases := []struct {
		name  string
		files []string
		want  string
	}{
		{"sem files", nil, "crudding init"},
		{"arquivo inexistente", []string{"c.txt"}, "não existe"},
		{"fora do projeto", []string{"../a.txt"}, "inválido"},
		{"absoluto", []string{"/etc/passwd"}, "inválido"},
		{"manifesto", []string{manifest.FileName}, "gravado pelo crudding"},
		{"colisão", []string{"a.txt", "b.txt"}, "mesmo arquivo"},
		{"symlink", []string{"link"}, "não é um arquivo comum"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// name "a" -> "b" faz a.txt colidir com b.txt.
			m := manifest.Manifest{Version: manifest.Version, Name: "a", Files: testCase.files}
			if _, err := custom.Plan(templates, m, "b"); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("Plan() err = %v, want contendo %q", err, testCase.want)
			}
		})
	}
}

func TestCustomUnlisted(t *testing.T) {
	templates := fstest.MapFS{
		"a.go":              {Data: []byte("a")},
		"b.go":              {Data: []byte("b")},
		"node_modules/x.js": {Data: []byte("x")},
		manifest.FileName:   {Data: []byte("{}")},
	}
	m := manifest.Manifest{Files: []string{"a.go"}, Ignore: []string{"node_modules"}}
	unlisted, err := custom.Unlisted(templates, m)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(unlisted, []string{"b.go"}) {
		t.Errorf("Unlisted() = %v", unlisted)
	}
}
