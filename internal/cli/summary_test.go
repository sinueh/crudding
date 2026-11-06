package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/manifest"
)

func TestCompressPaths(t *testing.T) {
	cases := []struct {
		name  string
		paths []string
		want  []string
	}{
		{"vazio", nil, nil},
		{"um arquivo", []string{"src/app/main.py"}, []string{"src/app/main.py"}},
		{
			"mesma pasta e extensão",
			[]string{"src/app/product/service.py", "src/app/product/controller.py", "src/app/product/models.py"},
			[]string{"src/app/product/{controller,models,service}.py"},
		},
		{
			"mesmo nome em pastas irmãs",
			[]string{"internal/models/product.go", "internal/types/product.go", "internal/service/product.go", "internal/handlers/product.go"},
			[]string{"internal/{handlers,models,service,types}/product.go"},
		},
		{
			"prefixo e sufixo comuns",
			[]string{"user_handler.go", "user_model.go", "user_service.go"},
			[]string{"user_{handler,model,service}.go"},
		},
		{
			"sufixo comum",
			[]string{"uc/create_invoice.py", "uc/delete_invoice.py"},
			[]string{"uc/{create,delete}_invoice.py"},
		},
		{
			"extensões diferentes na raiz",
			[]string{"README.md", ".gitignore", "go.mod"},
			[]string{"{.gitignore,README.md,go.mod}"},
		},
		{
			"nome contido em outro",
			[]string{"internal/domain/invoice.go", "internal/domain/invoice_types.go"},
			[]string{"internal/domain/{invoice,invoice_types}.go"},
		},
		{
			"pastas mistas",
			[]string{"src/app/main.py", "project.crud", "src/app/order/models.py", "src/app/order/service.py"},
			[]string{"project.crud", "src/app/main.py", "src/app/order/{models,service}.py"},
		},
		{"duplicados", []string{"a.go", "a.go"}, []string{"a.go"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compressPaths(tc.paths); !slices.Equal(got, tc.want) {
				t.Errorf("compressPaths(%q) = %q, want %q", tc.paths, got, tc.want)
			}
		})
	}
}

func TestEditTargets(t *testing.T) {
	cases := []struct {
		name    string
		created []string
		want    []editTarget
	}{
		{
			"python modular",
			[]string{"src/app/product/controller.py", "src/app/product/service.py", "src/app/product/models.py", "src/app/product/__init__.py"},
			[]editTarget{{"src/app/product/service.py", "regras de negócio"}, {"src/app/product/models.py", "campos"}},
		},
		{
			"go hexagonal ignora ports e DTOs",
			[]string{"internal/domain/invoice.go", "internal/domain/invoice_types.go", "internal/ports/invoice_service.go", "internal/application/invoice.go", "internal/handlers/invoice.go"},
			[]editTarget{{"internal/application/invoice.go", "regras de negócio"}, {"internal/domain/invoice.go", "campos"}},
		},
		{
			"casos de uso",
			[]string{"src/app/application/use_cases/create_x.py", "src/app/application/port/outbound/x_repository.py", "src/app/domain/model/x.py"},
			[]editTarget{{"src/app/application/use_cases/create_x.py", "regras de negócio"}, {"src/app/domain/model/x.py", "campos"}},
		},
		{"interno ignorado", []string{".crudding/template/service.py"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := editTargets(tc.created); !slices.Equal(got, tc.want) {
				t.Errorf("editTargets() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEndpointRoutes(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			"gin com grupo",
			"api := r.Group(\"/api/v1\")\n\tapi.POST(\"/order_item\", h.Create)\n\tapi.GET(\"/order_item/:id\", h.Get)\n\tapi.GET(\"/order_item_line\", x)\n",
			[]string{"/api/v1/order_item"},
		},
		{
			"chi Route",
			"r.Route(\"/api/v1\", func(api chi.Router) {\n\t\tapi.Get(\"/order_item/{id}\", h.Get)\n",
			[]string{"/api/v1/order_item"},
		},
		{
			"net/http com método",
			"mux.HandleFunc(\"GET /api/v1/order_item/{id}\", h.Get)\n",
			[]string{"/api/v1/order_item"},
		},
		{
			"fastapi ignora tags e imports",
			"from src.app.order_item.controller import router as order_item_router\napp.include_router(order_item_router, prefix=\"/order_item\", tags=[\"OrderItem\"])\n",
			[]string{"/order_item"},
		},
		{"django", "    path(\"order_item/\", include(order_item_urls)),\n", []string{"/order_item"}},
		{"dicionário", "    \"order_item\": handle_order_item,\n", []string{"/order_item"}},
		{"import Go", "\t\"example.com/app/internal/order_item\"\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "routes"), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := endpointRoutes(root, []string{"routes", "ausente"}, "order_item"); !slices.Equal(got, tc.want) {
				t.Errorf("endpointRoutes() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSplitJSONFlag(t *testing.T) {
	cases := []struct {
		args     []string
		wantRest []string
		wantJSON bool
		wantErr  bool
	}{
		{[]string{"product"}, []string{"product"}, false, false},
		{[]string{"product", "--json"}, []string{"product"}, true, false},
		{[]string{"-json", "n:x"}, []string{"n:x"}, true, false},
		{[]string{"json:true", "n:x"}, []string{"n:x"}, true, false},
		{[]string{"--json", "json:false"}, []string{}, false, false},
		{[]string{"--json=true"}, []string{}, true, false},
		{[]string{"json:talvez"}, nil, false, true},
	}
	for _, tc := range cases {
		rest, asJSON, err := splitJSONFlag(tc.args)
		if (err != nil) != tc.wantErr {
			t.Fatalf("splitJSONFlag(%q) error = %v, wantErr %v", tc.args, err, tc.wantErr)
		}
		if err == nil && (!slices.Equal(rest, tc.wantRest) || asJSON != tc.wantJSON) {
			t.Errorf("splitJSONFlag(%q) = %q, %v; want %q, %v", tc.args, rest, asJSON, tc.wantRest, tc.wantJSON)
		}
	}
}

func TestSummaryText(t *testing.T) {
	s := summary{
		Command:  "add endpoint",
		Created:  []string{"src/app/product/controller.py", "src/app/product/service.py", "src/app/product/models.py"},
		Modified: []string{"src/app/main.py", "project.crud"},
		Edit:     editTargets([]string{"src/app/product/service.py", "src/app/product/models.py"}),
		Routes:   []string{"/product"},
		Next:     []string{"cd app", "pip install -r requirements.txt", "crudding add endpoint <nome>"},
	}
	want := "" +
		"criado:   src/app/product/{controller,models,service}.py\n" +
		"alterado: project.crud, src/app/main.py\n" +
		"editar:   src/app/product/service.py (regras de negócio)\n" +
		"          src/app/product/models.py (campos)\n" +
		"rota:     /product\n" +
		"próximo:  cd app && pip install -r requirements.txt\n" +
		"          crudding add endpoint <nome>\n"
	if got := s.text(); got != want {
		t.Errorf("text() =\n%s\nwant\n%s", got, want)
	}

	internal := summary{Created: []string{".crudding/template/a.go", "main.go"}, Removed: []string{"x.go"}, Routes: []string{"/x"}, routeNote: "removida"}
	for _, line := range []string{"criado:   main.go, .crudding/ (interno)\n", "removido: x.go\n", "rota:     /x (removida)\n"} {
		if !strings.Contains(internal.text(), line) {
			t.Errorf("text() sem %q:\n%s", line, internal.text())
		}
	}
}

func TestSummaryLongLinesWrap(t *testing.T) {
	var created []string
	for _, dir := range []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet"} {
		created = append(created, "internal/"+dir+"/handler_"+dir+".go")
	}
	text := summary{Created: created}.text()
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("esperava quebra de linha:\n%s", text)
	}
	for _, line := range lines[1:] {
		if !strings.HasPrefix(line, strings.Repeat(" ", summaryLabelWidth)) {
			t.Errorf("continuação sem alinhamento: %q", line)
		}
	}
	for _, line := range lines {
		if len([]rune(line)) > summaryLineWidth {
			t.Errorf("linha maior que %d: %q", summaryLineWidth, line)
		}
	}
}

func TestSummaryJSON(t *testing.T) {
	var b strings.Builder
	s := summary{Command: "remove endpoint", Project: "/tmp/app", Removed: []string{"a.go"}, Edit: []editTarget{{"b.go", "campos"}}}
	if err := s.write(&b, true); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(b.String()), &got); err != nil {
		t.Fatalf("JSON inválido %q: %v", b.String(), err)
	}
	for _, key := range []string{"command", "project", "created", "modified", "removed", "edit", "routes", "next"} {
		if _, ok := got[key]; !ok {
			t.Errorf("chave %q ausente em %s", key, b.String())
		}
	}
	if strings.Contains(b.String(), "null") {
		t.Errorf("listas vazias devem ser [], não null: %s", b.String())
	}
	if edit, _ := got["edit"].([]any); len(edit) != 1 || edit[0] != "b.go" {
		t.Errorf("edit = %v, want [b.go]", got["edit"])
	}
	if strings.Count(b.String(), "\n") != 1 {
		t.Errorf("esperava um único objeto em uma linha: %q", b.String())
	}
}

func TestRunCommand(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		m     manifest.Manifest
		want  string
	}{
		{"go cmd/api", []string{"cmd/api/main.go"}, manifest.Manifest{Language: "go"}, "go run ./cmd/api"},
		{"go flat", []string{"main.go"}, manifest.Manifest{Language: "go"}, "go run ."},
		{"fastapi", []string{"src/app/main.py"}, manifest.Manifest{Language: "python", Framework: "fastapi"}, "uvicorn src.app.main:app --reload"},
		{"django flat", []string{"src/main.py"}, manifest.Manifest{Language: "python", Framework: "django"}, "python -m src.main runserver"},
		{"flask", []string{"src/app/main.py"}, manifest.Manifest{Language: "python", Framework: "flask"}, "python -m src.app.main"},
		{"node", nil, manifest.Manifest{Language: "node"}, "npm start"},
		{"custom", nil, manifest.Manifest{Architecture: "custom"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, file := range tc.files {
				target := filepath.Join(root, filepath.FromSlash(file))
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := runCommand(root, tc.m); got != tc.want {
				t.Errorf("runCommand() = %q, want %q", got, tc.want)
			}
		})
	}
}
