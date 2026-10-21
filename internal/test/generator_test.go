package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"crudding/internal/generator"
	"crudding/internal/manifest"
	"crudding/internal/project"
)

// templates aponta para a raiz do repositório, onde fica a pasta templates/.
var templates = os.DirFS(filepath.Join("..", ".."))

// Arquivos que devem existir, por linguagem e arquitetura. No Node os caminhos
// não têm extensão: o runtime (.js/.ts) é acrescentado na checagem.
var requiredFiles = map[string]map[string][]string{
	"go": {
		"flat":       {"main.go", "routes.go", "user_model.go", "user_types.go", "user_service.go", "user_handler.go", "config/config.go"},
		"layered":    {"cmd/api/main.go", "internal/routes/routes.go", "internal/handlers/user.go", "internal/service/user.go", "internal/types/user.go", "internal/models/user.go", "internal/config/config.go"},
		"modular":    {"cmd/api/main.go", "internal/routes/routes.go", "internal/user/handler.go", "internal/user/service.go", "internal/user/types.go", "internal/user/model.go", "internal/config/config.go"},
		"hexagonal":  {"cmd/api/main.go", "internal/routes/routes.go", "internal/handlers/user.go", "internal/application/user.go", "internal/ports/user_repository.go", "internal/ports/user_service.go", "internal/adapters/postgres/user.go", "internal/domain/user.go", "internal/domain/user_types.go", "internal/config/config.go"},
		"clean_arch": {"cmd/api/main.go", "internal/routes/routes.go", "internal/handlers/user.go", "internal/application/user.go", "internal/ports/user_repository.go", "internal/ports/user_service.go", "internal/infrastructure/persistence/user.go", "internal/domain/user.go", "internal/domain/user_types.go", "internal/config/config.go"},
	},
	"node": {
		"flat":       {"src/main", "src/server", "src/tasks", "src/tasks.test"},
		"layered":    {"src/main", "src/http/server", "src/routes/index", "src/controllers/user-controller", "src/services/order-service", "src/models/user"},
		"modular":    {"src/main", "src/shared/http/server", "src/shared/database", "src/modules/catalog/index", "src/modules/orders/order-service"},
		"hexagonal":  {"src/entrypoints/main", "src/adapters/http/server", "src/adapters/http/product-controller", "src/domain/product-manager", "src/ports/product-repository"},
		"clean_arch": {"src/frameworks/main", "src/frameworks/http/server", "src/entities/order", "src/use-cases/create-order", "src/adapters/controllers/order-controller"},
	},
	"python": {
		"flat":       {"setup.py", "src/resource_controller.py", "src/resource_service.py", "src/resource_model.py", "src/main.py", "src/config/config.py"},
		"layered":    {"src/app/main.py", "src/app/controllers/resource_controller.py", "src/app/services/resource_service.py", "src/app/models/resource.py", "src/app/config/config.py"},
		"modular":    {"src/app/main.py", "src/app/resource/controller.py", "src/app/resource/service.py", "src/app/resource/models.py", "src/app/config/config.py"},
		"hexagonal":  {"src/app/main.py", "src/app/routers/resource_controller.py", "src/app/application/resource_service.py", "src/app/ports/resource_repository.py", "src/app/infrastructure/resource_repository.py", "src/app/domain/resource.py", "src/app/config/config.py"},
		"clean_arch": {"src/app/main.py", "src/app/frameworks_and_drivers/web/resource_controller.py", "src/app/interface_adapters/controllers/resource_controller.py", "src/app/interface_adapters/presenters/resource_presenter.py", "src/app/application/use_cases/create_resource.py", "src/app/application/use_cases/list_resource.py", "src/app/application/port/outbound/resource_repository.py", "src/app/interface_adapters/persistence/resource_repository.py", "src/app/domain/model/resource.py", "src/app/config/config.py"},
	},
}

// Arquivos que NÃO devem existir (camadas removidas propositalmente).
var forbiddenFiles = map[string]map[string][]string{
	"go": {
		"flat":       {"user_repository.go", "tasks.go", "tasks_test.go"},
		"layered":    {"routes", "internal/repository", "internal/router"},
		"modular":    {"routes", "internal/user/repository.go", "internal/router"},
		"hexagonal":  {"routes", "cmd/grpc", "internal/adapters/grpc"},
		"clean_arch": {"routes", "pkg/validator", "migrations"},
	},
	"python": {
		"flat":       {"src/repository.py", "src/resource_repository.py"},
		"layered":    {"src/app/repositories", "src/app/repository", "src/app/services/resource_repository.py", "src/app/router", "src/app/controllers/api.py"},
		"modular":    {"src/app/resource/repository.py", "src/app/shared/api.py"},
		"hexagonal":  {"src/app/routers/api.py"},
		"clean_arch": {"src/app/frameworks_and_drivers/web/api.py"},
	},
}

// Trechos de código que comprovam que o scaffold genérico foi gerado.
var codeChecks = map[string]map[string][2]string{
	"python": {
		"flat":       {"src/resource_model.py", "class Resource"},
		"layered":    {"src/app/services/resource_service.py", "def list_all"},
		"modular":    {"src/app/resource/service.py", "def list_all"},
		"hexagonal":  {"src/app/application/resource_service.py", "class ResourceService"},
		"clean_arch": {"src/app/application/use_cases/create_resource.py", "class CreateResource"},
	},
}

// allConfigs monta a matriz completa a partir do registry de linguagens.
func allConfigs() []project.Config {
	var configs []project.Config
	for _, language := range generator.Languages() {
		runtimes := language.Runtimes
		if len(runtimes) == 0 {
			runtimes = []string{""}
		}
		for _, framework := range language.Frameworks {
			for _, runtime := range runtimes {
				for _, architecture := range language.Architectures {
					configs = append(configs, project.Config{ProjectName: "Sample", Language: language.Name, Runtime: runtime, Framework: framework, Architecture: architecture})
				}
			}
		}
	}
	return configs
}

func assertFilesExist(t *testing.T, root string, files ...string) {
	t.Helper()
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Errorf("arquivo %s ausente: %v", file, err)
		}
	}
}

func assertFilesAbsent(t *testing.T, root string, files ...string) {
	t.Helper()
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(root, file)); !os.IsNotExist(err) {
			t.Errorf("%s não deveria existir", file)
		}
	}
}

func assertFileContains(t *testing.T, root, file, snippet string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, file))
	if err != nil || !strings.Contains(string(content), snippet) {
		t.Errorf("%s não contém %q: %v", file, snippet, err)
	}
}

func TestGenerateAllTemplateCombinations(t *testing.T) {
	for _, config := range allConfigs() {
		t.Run(config.Language+"/"+config.Framework+"/"+config.Runtime+"/"+config.Architecture, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "Sample")
			if err := generator.Generate(destination, config, templates); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			readme, err := os.ReadFile(filepath.Join(destination, "README.md"))
			if err != nil || len(readme) == 0 {
				t.Fatalf("README.md ausente ou vazio: %v", err)
			}

			files := requiredFiles[config.Language][config.Architecture]
			if len(files) == 0 {
				t.Fatalf("sem arquivos obrigatórios cadastrados para %s/%s", config.Language, config.Architecture)
			}
			if config.Runtime != "" {
				withExt := make([]string, len(files))
				for index, file := range files {
					withExt[index] = file + "." + config.Runtime
				}
				files = withExt
			}
			if _, m, err := manifest.Find(destination); err != nil || m.Config() != config {
				t.Errorf("%s = %+v, %v; want %+v", manifest.FileName, m, err, config)
			}
			assertFilesExist(t, destination, files...)
			assertFilesAbsent(t, destination, forbiddenFiles[config.Language][config.Architecture]...)
			if check, ok := codeChecks[config.Language][config.Architecture]; ok {
				assertFileContains(t, destination, check[0], check[1])
			}

			switch config.Language {
			case "go":
				assertFilesExist(t, destination, "go.mod", "go.sum", ".gitignore")
			case "node":
				contents, err := os.ReadFile(filepath.Join(destination, "package.json"))
				if err != nil {
					t.Fatal(err)
				}
				var parsed map[string]any
				if err := json.Unmarshal(contents, &parsed); err != nil {
					t.Fatalf("invalid package.json: %v", err)
				}
				_, err = os.Stat(filepath.Join(destination, "tsconfig.json"))
				if hasTsconfig := err == nil; hasTsconfig != (config.Runtime == "ts") {
					t.Errorf("tsconfig.json presente = %v para runtime %s", hasTsconfig, config.Runtime)
				}
			case "python":
				assertFilesExist(t, destination, "requirements.txt")
			}

			if config.Language != "go" && !strings.Contains(string(readme), "- Arquitetura: "+config.Architecture) {
				t.Errorf("README não identifica a arquitetura %s", config.Architecture)
			}
		})
	}
}

func TestGenerateRefusesExistingDirectory(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "existing")
	if err := os.Mkdir(destination, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	config := project.Config{ProjectName: "existing", Language: "go", Framework: "raw", Architecture: "flat"}
	if err := generator.Generate(destination, config, os.DirFS(filepath.Join("..", ".."))); err == nil {
		t.Fatal("Generate() should reject an existing directory")
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing content changed: %q, %v", content, err)
	}
}

func TestPythonGenericResourceDispatchers(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	templates := os.DirFS(filepath.Join("..", ".."))
	cases := []struct {
		architecture string
		module       string
	}{
		{"flat", "src.main"},
		{"layered", "src.app.main"},
		{"modular", "src.app.main"},
		{"hexagonal", "src.app.main"},
		{"clean_arch", "src.app.main"},
	}

	for _, testCase := range cases {
		t.Run(testCase.architecture, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "Sample")
			config := project.Config{ProjectName: "Sample", Language: "python", Framework: "raw", Architecture: testCase.architecture}
			if err := generator.Generate(destination, config, templates); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			script := `from ` + testCase.module + ` import ROUTES

class MockHandler:
    def __init__(self):
        self.status = None
        self.data = None
    def send_json(self, status, data):
        self.status = status
        self.data = data
    def read_json(self):
        return {}
    def send_response(self, status):
        self.status = status
    def end_headers(self):
        pass

assert "resource" in ROUTES, "resource route missing"
handler = MockHandler()
ROUTES["resource"](handler, "GET", [])
assert handler.status == 501, f"expected 501, got {handler.status}"

handler = MockHandler()
ROUTES["resource"](handler, "GET", ["1"])
assert handler.status == 501, f"expected 501, got {handler.status}"
`
			command := exec.Command(python, "-c", script)
			command.Dir = destination
			command.Env = append(os.Environ(), "PYTHONPATH="+destination)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("generated generic resource failed: %v\n%s", err, output)
			}
		})
	}
}

func TestPythonFrameworkEntrypointsCompile(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	templates := os.DirFS(filepath.Join("..", ".."))
	for _, framework := range []string{"raw", "fastapi", "django", "flask"} {
		t.Run(framework, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "Sample")
			config := project.Config{ProjectName: "Sample", Language: "python", Framework: framework, Architecture: "flat"}
			if err := generator.Generate(destination, config, templates); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			command := exec.Command(python, "-m", "compileall", "-q", destination)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("generated %s entrypoint does not compile: %v\n%s", framework, err, output)
			}
		})
	}
}

func TestGenerateRejectsUnsupportedOptions(t *testing.T) {
	templates := os.DirFS(filepath.Join("..", ".."))
	for _, config := range []project.Config{
		{Language: "go", Framework: "express", Architecture: "flat"},
		{Language: "go", Framework: "raw", Architecture: "monorepo"},
		{Language: "node", Runtime: "ruby", Framework: "raw", Architecture: "layered"},
		{Language: "node", Runtime: "js", Framework: "raw", Architecture: "monorepo"},
		{Language: "python", Framework: "raw", Architecture: "monorepo"},
		{Language: "python", Framework: "raw", Architecture: "unknown"},
	} {
		if err := generator.Generate(filepath.Join(t.TempDir(), "project"), config, templates); err == nil || strings.Contains(err.Error(), "template") {
			t.Errorf("Generate(%+v) should reject unsupported options before rendering, got %v", config, err)
		}
	}
}

func TestGenerateGoVet(t *testing.T) {
	templates := os.DirFS(filepath.Join("..", ".."))
	for _, arch := range []string{"flat", "layered", "modular", "hexagonal", "clean_arch"} {
		dest := filepath.Join(t.TempDir(), arch)
		cfg := project.Config{
			ProjectName:  arch,
			Language:     "go",
			Framework:    "raw",
			Architecture: arch,
		}
		if err := generator.Generate(dest, cfg, templates); err != nil {
			t.Fatalf("Generate() error = %v for %s", err, arch)
		}
	}
}

func TestGeneratedGoRawScaffoldsCompile(t *testing.T) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go is not installed")
	}
	templates := os.DirFS(filepath.Join("..", ".."))
	for _, architecture := range []string{"flat", "layered", "modular", "hexagonal", "clean_arch"} {
		t.Run(architecture, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "Sample")
			config := project.Config{ProjectName: "Sample", Language: "go", Framework: "raw", Architecture: architecture}
			if err := generator.Generate(destination, config, templates); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if smokeTest, ok := goAPISmokeTests[architecture]; ok {
				if err := os.WriteFile(filepath.Join(destination, "api_smoke_test.go"), []byte(smokeTest), 0644); err != nil {
					t.Fatal(err)
				}
			}
			command := exec.Command(goCommand, "test", "./...")
			command.Dir = destination
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("generated Go project does not compile: %v\n%s", err, output)
			}
		})
	}
}

var goAPISmokeTests = map[string]string{
	"flat": `package main
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)
func TestDemoAPI(t *testing.T) {
	router := SetupRouter()
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	created := request(http.MethodPost, "/api/v1/users", ` + "`" + `{"name":"Alice","email":"alice@example.com","password":"secret"}` + "`" + `)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("create = %d %s", created.Code, created.Body.String()) }
	list := request(http.MethodGet, "/api/v1/users", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), ` + "`" + `"alice@example.com"` + "`" + `) { t.Fatalf("list = %d %s", list.Code, list.Body.String()) }
	get := request(http.MethodGet, "/api/v1/users/1", "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("get = %d %s", get.Code, get.Body.String()) }
}`,
	"layered": `package main
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"example.com/sample/internal/routes"
)
func TestDemoAPI(t *testing.T) {
	router := routes.SetupRouter()
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	created := request(http.MethodPost, "/api/v1/users", ` + "`" + `{"name":"Alice","email":"alice@example.com","password":"secret"}` + "`" + `)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("create = %d %s", created.Code, created.Body.String()) }
	list := request(http.MethodGet, "/api/v1/users", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), ` + "`" + `"alice@example.com"` + "`" + `) { t.Fatalf("list = %d %s", list.Code, list.Body.String()) }
	get := request(http.MethodGet, "/api/v1/users/1", "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("get = %d %s", get.Code, get.Body.String()) }
}`,
	"modular": `package main
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"example.com/sample/internal/routes"
)
func TestDemoAPI(t *testing.T) {
	router := routes.SetupRouter()
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	created := request(http.MethodPost, "/api/v1/users", ` + "`" + `{"name":"Alice","email":"alice@example.com","password":"secret"}` + "`" + `)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("create = %d %s", created.Code, created.Body.String()) }
	list := request(http.MethodGet, "/api/v1/users", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), ` + "`" + `"alice@example.com"` + "`" + `) { t.Fatalf("list = %d %s", list.Code, list.Body.String()) }
	get := request(http.MethodGet, "/api/v1/users/1", "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("get = %d %s", get.Code, get.Body.String()) }
}`,
	"hexagonal": `package main
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"example.com/sample/internal/routes"
)
func TestDemoAPI(t *testing.T) {
	router := routes.SetupRouter()
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	created := request(http.MethodPost, "/api/v1/users", ` + "`" + `{"name":"Alice","email":"alice@example.com","password":"secret"}` + "`" + `)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("create = %d %s", created.Code, created.Body.String()) }
	list := request(http.MethodGet, "/api/v1/users", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), ` + "`" + `"alice@example.com"` + "`" + `) { t.Fatalf("list = %d %s", list.Code, list.Body.String()) }
	get := request(http.MethodGet, "/api/v1/users/1", "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("get = %d %s", get.Code, get.Body.String()) }
}`,
	"clean_arch": `package main
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"example.com/sample/internal/routes"
)
func TestDemoAPI(t *testing.T) {
	router := routes.SetupRouter()
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	created := request(http.MethodPost, "/api/v1/users", ` + "`" + `{"name":"Alice","email":"alice@example.com","password":"secret"}` + "`" + `)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("create = %d %s", created.Code, created.Body.String()) }
	list := request(http.MethodGet, "/api/v1/users", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), ` + "`" + `"alice@example.com"` + "`" + `) { t.Fatalf("list = %d %s", list.Code, list.Body.String()) }
	get := request(http.MethodGet, "/api/v1/users/1", "")
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), ` + "`" + `"name":"Alice"` + "`" + `) { t.Fatalf("get = %d %s", get.Code, get.Body.String()) }
}`,
}

func TestPythonSQLAlchemyGeneration(t *testing.T) {
	for _, arch := range []string{"flat", "layered", "modular", "hexagonal", "clean_arch"} {
		t.Run(arch, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "py_orm_app")
			cfg := project.Config{
				ProjectName:  "py-orm-app",
				Language:     "python",
				Framework:    "fastapi",
				Architecture: arch,
				Database:     "postgres",
				ORM:          "sqlalchemy",
			}

			if err := generator.Generate(destination, cfg, templates); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			// 1. requirements.txt deve conter SQLAlchemy e psycopg2
			reqBytes, err := os.ReadFile(filepath.Join(destination, "requirements.txt"))
			if err != nil {
				t.Fatalf("read requirements.txt: %v", err)
			}
			reqText := string(reqBytes)
			if !strings.Contains(reqText, "SQLAlchemy") {
				t.Errorf("requirements.txt does not contain SQLAlchemy:\n%s", reqText)
			}
			if !strings.Contains(reqText, "psycopg2-binary") {
				t.Errorf("requirements.txt does not contain psycopg2-binary:\n%s", reqText)
			}

			// 2. config.py deve conter engine e SessionLocal
			configRel := "src/config/config.py"
			if arch != "flat" {
				configRel = "src/app/config/config.py"
			}
			configBytes, err := os.ReadFile(filepath.Join(destination, configRel))
			if err != nil {
				t.Fatalf("read config.py: %v", err)
			}
			configText := string(configBytes)
			if !strings.Contains(configText, "create_engine") || !strings.Contains(configText, "SessionLocal") || !strings.Contains(configText, "Base = declarative_base()") {
				t.Errorf("config.py does not contain SQLAlchemy setup:\n%s", configText)
			}

			// 3. Checar se o model contém Column e herda de Base
			modelRel := "src/resource_model.py"
			switch arch {
			case "layered":
				modelRel = "src/app/models/resource.py"
			case "modular":
				modelRel = "src/app/resource/models.py"
			case "hexagonal":
				modelRel = "src/app/domain/resource.py"
			case "clean_arch":
				modelRel = "src/app/domain/model/resource.py"
			}
			modelBytes, err := os.ReadFile(filepath.Join(destination, modelRel))
			if err != nil {
				t.Fatalf("read model file %s: %v", modelRel, err)
			}
			modelText := string(modelBytes)
			if !strings.Contains(modelText, "class Resource(Base):") || !strings.Contains(modelText, "Column(String, primary_key=True") {
				t.Errorf("%s does not contain SQLAlchemy model definition:\n%s", modelRel, modelText)
			}
		})
	}
}


