package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sinueh-andreata/crudding/internal/generator/common"
)

var patchTemplates = fstest.MapFS{
	"t/new.tmpl": {Data: []byte("hello {{.}}\n")},
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestApplyCreatesFilesPackagesAndInserts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api.py", "import x\n# m:imports\n\ndef f():\n    if a:\n        pass\n    # m:routes\n    return 1\n")

	result, err := common.Apply(root, patchTemplates, common.Patch{
		Files:    []common.File{{TemplatePath: "t/new.tmpl", OutputPath: "pkg/sub/new.py"}},
		Data:     "world",
		Packages: []string{"pkg/sub"},
		Inserts: []common.Insert{
			{Path: "api.py", Marker: "# m:imports", Lines: []string{"import y"}},
			{Path: "api.py", Marker: "# m:routes", Lines: []string{"if b:", "    return 2"}},
		},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	want := "import x\nimport y\n# m:imports\n\ndef f():\n    if a:\n        pass\n    if b:\n        return 2\n    # m:routes\n    return 1\n"
	if got := readFile(t, root, "api.py"); got != want {
		t.Errorf("api.py =\n%s\nwant\n%s", got, want)
	}
	if got := readFile(t, root, "pkg/sub/new.py"); got != "hello world\n" {
		t.Errorf("new.py = %q", got)
	}
	wantCreated := []string{"pkg/__init__.py", "pkg/sub/__init__.py", "pkg/sub/new.py"}
	if strings.Join(result.Created, ",") != strings.Join(wantCreated, ",") || strings.Join(result.Modified, ",") != "api.py" {
		t.Errorf("Apply() = %+v", result)
	}
}

func TestApplyValidatesBeforeWriting(t *testing.T) {
	cases := map[string]string{
		"marker ausente":   "nothing here\n",
		"marker duplicado": "# m\n# m\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "api.py", content)
			_, err := common.Apply(root, patchTemplates, common.Patch{
				Files:    []common.File{{TemplatePath: "t/new.tmpl", OutputPath: "pkg/new.py"}},
				Packages: []string{"pkg"},
				Inserts:  []common.Insert{{Path: "api.py", Marker: "# m", Lines: []string{"x"}}},
			})
			if err == nil {
				t.Fatal("Apply() deveria falhar")
			}
			if _, err := os.Stat(filepath.Join(root, "pkg")); !os.IsNotExist(err) {
				t.Error("pkg/ não deveria ter sido criado")
			}
			if got := readFile(t, root, "api.py"); got != content {
				t.Errorf("api.py alterado: %q", got)
			}
		})
	}
}

func TestApplyRollsBackOnWriteFailure(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api.py", "# m\n")
	writeFile(t, root, "blocker", "") // arquivo onde o segundo template precisaria de um diretório

	_, err := common.Apply(root, patchTemplates, common.Patch{
		Files: []common.File{
			{TemplatePath: "t/new.tmpl", OutputPath: "pkg/first.py"},
			{TemplatePath: "t/new.tmpl", OutputPath: "blocker/second.py"},
		},
		Packages: []string{"pkg"},
		Inserts:  []common.Insert{{Path: "api.py", Marker: "# m", Lines: []string{"x"}}},
	})
	if err == nil {
		t.Fatal("Apply() deveria falhar")
	}
	if _, err := os.Stat(filepath.Join(root, "pkg")); !os.IsNotExist(err) {
		t.Error("rollback não removeu pkg/")
	}
	if got := readFile(t, root, "api.py"); got != "# m\n" {
		t.Errorf("api.py alterado: %q", got)
	}
}

func TestApplyRefusesExistingFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pkg/new.py", "keep")
	_, err := common.Apply(root, patchTemplates, common.Patch{Files: []common.File{{TemplatePath: "t/new.tmpl", OutputPath: "pkg/new.py"}}})
	if err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Fatalf("Apply() err = %v", err)
	}
	if got := readFile(t, root, "pkg/new.py"); got != "keep" {
		t.Errorf("arquivo existente alterado: %q", got)
	}
}

func TestRemoveExecutesPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "api.py", "import x\nimport y\n# m:imports\n\nline1\nline2\n# m:routes\n")
	writeFile(t, root, "pkg/sub/file.go", "package sub")

	res, err := common.Remove(root, common.RemovalPlan{
		Files:       []string{"pkg/sub/file.go"},
		Directories: []string{"pkg/sub", "pkg"},
		Removals: []common.LineRemoval{
			{Path: "api.py", Lines: []string{"import y", "line2"}},
		},
	})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if len(res.Removed) != 1 || res.Removed[0] != "pkg/sub/file.go" {
		t.Errorf("res.Removed = %v", res.Removed)
	}
	if len(res.Modified) != 1 || res.Modified[0] != "api.py" {
		t.Errorf("res.Modified = %v", res.Modified)
	}

	wantAPI := "import x\n# m:imports\n\nline1\n# m:routes\n"
	if got := readFile(t, root, "api.py"); got != wantAPI {
		t.Errorf("api.py =\n%s\nwant\n%s", got, wantAPI)
	}

	if _, err := os.Stat(filepath.Join(root, "pkg")); !os.IsNotExist(err) {
		t.Error("diretórios vazios pkg/sub e pkg deveriam ter sido removidos")
	}
}

