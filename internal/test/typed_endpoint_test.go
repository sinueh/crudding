package test

import (
	"bytes"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/cli"
	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/generator/fields"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

var (
	productFields  = []string{"name:str", "price:decimal", "image_url:str?", "released:date?", "active:bool"}
	cartItemFields = []string{"cart:ref", "product:ref", "quantity:int"}
)

func TestParseFields(t *testing.T) {
	list, err := fields.Parse([]string{"name:str", "image_url:str?", "cart:ref", "at:datetime?"})
	if err != nil {
		t.Fatal(err)
	}
	if got := fields.Specs(list); !slices.Equal(got, []string{"name:str", "image_url:str?", "cart:ref", "at:datetime?"}) {
		t.Errorf("Specs() = %v", got)
	}
	if !list[1].Nullable || list[0].Nullable || list[2].Column() != "cart_id" || list[2].Ref() != "cart" {
		t.Errorf("Parse() = %+v", list)
	}
	if refs := fields.Refs(list); !slices.Equal(refs, []string{"cart"}) {
		t.Errorf("Refs() = %v", refs)
	}

	pyLang, _ := generator.TypedLanguageFor("python")
	goLang, _ := generator.TypedLanguageFor("go")
	allReserved := slices.Concat(pyLang.Reserved(), goLang.Reserved())
	for spec, want := range map[string]string{
		"name":           "nome:tipo",
		"name:":          "nome:tipo",
		"name:string":    "tipo \"string\" desconhecido",
		"Name:str":       "nome inválido",
		"first-name:str": "nome inválido",
		"id:int":         "id é gerado",
		"class:str":      "palavra reservada",
		"type:str":       "palavra reservada",
		"model_x:str":    "palavra reservada",
		"validate:str":   "palavra reservada",
	} {
		if _, err := fields.Parse([]string{spec}, allReserved...); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) err = %v, want contendo %q", spec, err, want)
		}
	}
	// Palavras reservadas são separadas por linguagem: class e model_x só são barrados em Python; type e chan, só em Go.
	if _, err := fields.Parse([]string{"class:str", "model_x:str"}, goLang.Reserved()...); err != nil {
		t.Errorf("Go recusou palavra reservada de Python: %v", err)
	}
	if _, err := fields.Parse([]string{"type:str", "chan:str"}, pyLang.Reserved()...); err != nil {
		t.Errorf("Python recusou palavra reservada de Go: %v", err)
	}
	for _, specs := range [][]string{{"name:str", "name:text"}, {"cart:ref", "cart_id:int"}} {
		if _, err := fields.Parse(specs); err == nil || !strings.Contains(err.Error(), "já é usada") {
			t.Errorf("Parse(%v) err = %v, want coluna repetida", specs, err)
		}
	}
}

func TestMigrationSQL(t *testing.T) {
	list, err := fields.Parse(cartItemFields)
	if err != nil {
		t.Fatal(err)
	}
	mysql := fields.CreateTable(fields.MySQL, "cart_item", list)
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS `cart_item` (",
		"`id` BIGINT AUTO_INCREMENT PRIMARY KEY",
		"`cart_id` BIGINT NOT NULL",
		"`quantity` INT NOT NULL",
		"CONSTRAINT `fk_cart_item_cart_id` FOREIGN KEY (`cart_id`) REFERENCES `cart` (`id`)",
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	} {
		if !strings.Contains(mysql, want) {
			t.Errorf("CreateTable(mysql) sem %q:\n%s", want, mysql)
		}
	}

	product, _ := fields.Parse(productFields)
	postgres := fields.CreateTable(fields.Postgres, "product", product)
	for _, want := range []string{
		`"id" BIGSERIAL PRIMARY KEY`,
		`"name" VARCHAR(255) NOT NULL`,
		`"price" NUMERIC(12, 2) NOT NULL`,
		`"image_url" VARCHAR(255),`,
		`"released" DATE,`,
		`"active" BOOLEAN NOT NULL`,
	} {
		if !strings.Contains(postgres, want) {
			t.Errorf("CreateTable(postgres) sem %q:\n%s", want, postgres)
		}
	}
	refs := fields.CreateTable(fields.Postgres, "cart_item", list)
	if !strings.Contains(refs, `CREATE INDEX IF NOT EXISTS "idx_cart_item_cart_id" ON "cart_item" ("cart_id");`) {
		t.Errorf("CreateTable(postgres) sem índice da chave estrangeira:\n%s", refs)
	}
	if got := fields.DropTable(fields.MySQL, "cart_item"); got != "DROP TABLE IF EXISTS `cart_item`;\n" {
		t.Errorf("DropTable() = %q", got)
	}

	goLang, _ := generator.TypedLanguageFor("go")
	goQueries := fields.BuildQueries(fields.Postgres, "product", product, func(index int) string {
		return goLang.Placeholder(fields.Postgres, index)
	}, true)
	if !strings.Contains(goQueries.Insert, "VALUES ($1, $2, $3, $4, $5) RETURNING \"id\"") || !strings.Contains(goQueries.Update, `WHERE "id" = $6`) {
		t.Errorf("BuildQueries(go, postgres) = %+v", goQueries)
	}
	if !strings.Contains(goQueries.Select, `to_char("released", 'YYYY-MM-DD')`) {
		t.Errorf("BuildQueries(go, postgres).Select = %s", goQueries.Select)
	}
	pyLang, _ := generator.TypedLanguageFor("python")
	pyQueries := fields.BuildQueries(fields.MySQL, "product", product, func(index int) string {
		return pyLang.Placeholder(fields.MySQL, index)
	}, false)
	if pyQueries.Returning || !strings.Contains(pyQueries.Insert, "VALUES (%s, %s, %s, %s, %s)") {
		t.Errorf("BuildQueries(python, mysql) = %+v", pyQueries)
	}
}

// Manifestos antigos (sem endpoint_fields) continuam válidos e os campos são
// gravados por endpoint.
func TestManifestEndpointFields(t *testing.T) {
	old := []byte(`{"version": 1, "name": "app", "language": "go", "framework": "gin", "architecture": "layered", "endpoints": ["template", "invoice"], "ignore": [], "directories": [], "files": []}`)
	m, err := manifest.Parse(old, "project.crud")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Endpoints, []string{"template", "invoice"}) || m.EndpointFields != nil {
		t.Fatalf("Parse(antigo) = %+v", m)
	}

	root := t.TempDir()
	m.AddEndpoint("product")
	m.SetEndpointFields("product", []string{"name:str", "price:decimal"})
	if err := manifest.Write(root, m); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(root, manifest.FileName))
	if !strings.Contains(string(content), `"endpoint_fields": {`) {
		t.Errorf("project.crud sem endpoint_fields:\n%s", content)
	}
	read, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(read.EndpointFields["product"], []string{"name:str", "price:decimal"}) {
		t.Errorf("EndpointFields = %v", read.EndpointFields)
	}
	read.RemoveEndpoint("product")
	if read.EndpointFields != nil || read.HasEndpoint("product") {
		t.Errorf("RemoveEndpoint não limpou os campos: %+v", read)
	}
}

// typedExpectations são trechos que o endpoint product (com campos) precisa ter
// no arquivo do model, por linguagem.
var typedExpectations = map[string][]string{
	"python": {"class ProductIn(BaseModel):", "price: Decimal", "image_url: str | None = Field(default=None, max_length=255)", "class ProductPatch(BaseModel):", "class ProductOut(BaseModel):"},
	"go":     {"*json.Number `json:\"price\"`", "func (r CreateProductRequest) Validate() error", "type UpdateProductRequest struct"},
}

// Em toda combinação Python e Go, add endpoint com campos gera um endpoint
// tipado nos mesmos caminhos da cópia do exemplo, registra os campos no
// project.crud e o remove apaga o endpoint (menos as migrações).
func TestTypedEndpointAllCombinations(t *testing.T) {
	goBin, goErr := exec.LookPath("go")
	for _, config := range allConfigs() {
		if config.Language != "python" && config.Language != "go" {
			continue
		}
		config.Database = "postgres"
		if config.Framework == "raw" || config.Framework == "flask" {
			config.Database = "mysql"
		}
		t.Run(config.Language+"/"+config.Framework+"/"+config.Architecture, func(t *testing.T) {
			root, m := generateProject(t, config)
			for _, endpoint := range []struct {
				name  string
				specs []string
			}{{"product", productFields}, {"cart", []string{"label:str?"}}, {"cart_item", cartItemFields}} {
				result, notes, err := generator.AddTypedEndpoint(root, m, endpoint.name, endpoint.specs, templates)
				if err != nil {
					t.Fatalf("AddTypedEndpoint(%s) error = %v", endpoint.name, err)
				}
				if len(notes) != 0 {
					t.Errorf("AddTypedEndpoint(%s) notes = %v", endpoint.name, notes)
				}
				if !slices.ContainsFunc(result.Created, func(file string) bool {
					return strings.HasPrefix(file, "migrations/") && strings.HasSuffix(file, "_create_"+endpoint.name+".up.sql")
				}) {
					t.Errorf("AddTypedEndpoint(%s) sem migração: %v", endpoint.name, result.Created)
				}
			}
			after := assertStructureInSync(t, root)
			if !slices.Equal(after.EndpointFields["cart_item"], cartItemFields) || !slices.Equal(after.EndpointFields["product"], productFields) {
				t.Errorf("EndpointFields = %v", after.EndpointFields)
			}

			tree := readTree(t, root)
			model := ""
			for file, content := range tree {
				if !strings.HasPrefix(file, ".crudding/") && strings.Contains(file, "product") && strings.Contains(content, "Product") &&
					slices.ContainsFunc(typedExpectations[config.Language], func(snippet string) bool { return strings.Contains(content, snippet) }) {
					model += content
				}
			}
			for _, want := range typedExpectations[config.Language] {
				if !strings.Contains(model, want) {
					t.Errorf("endpoint product sem %q", want)
				}
			}
			for file, content := range tree {
				if strings.Contains(content, "NotImplementedError") && strings.Contains(file, "product") {
					t.Errorf("%s ainda tem stub NotImplementedError", file)
				}
			}

			switch {
			case config.Language == "python":
				checkPythonSyntax(t, root)
				assertFileContains(t, root, "migrations/000004_create_cart_item.up.sql", "REFERENCES")
			case config.Framework == "raw" && goErr == nil:
				command := exec.Command(goBin, "vet", "./...")
				command.Dir = root
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("projeto Go com endpoints tipados não passa no go vet: %v\n%s", err, output)
				}
				command = exec.Command(goBin, "test", "./...")
				command.Dir = root
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("testes gerados falharam: %v\n%s", err, output)
				}
			}

			if _, err := generator.RemoveTypedEndpoint(root, m, "product", templates); err == nil || !strings.Contains(err.Error(), "referenciado") {
				t.Errorf("RemoveTypedEndpoint(product) referenciado: err = %v", err)
			}
			before := readTree(t, root)
			for _, name := range []string{"cart_item", "cart", "product"} {
				if _, err := generator.RemoveTypedEndpoint(root, m, name, templates); err != nil {
					t.Fatalf("RemoveTypedEndpoint(%s) error = %v", name, err)
				}
			}
			final := assertStructureInSync(t, root)
			if len(final.EndpointFields) != 0 || !slices.Equal(final.Endpoints, []string{m.TemplateEndpoint.Name}) {
				t.Errorf("manifesto depois do remove = %+v", final)
			}
			for file := range readTree(t, root) {
				if strings.Contains(file, "product") || strings.Contains(file, "cart") {
					if !strings.HasPrefix(file, "migrations/") {
						t.Errorf("%s ficou depois do remove", file)
					}
				}
			}
			for file := range before {
				if strings.HasPrefix(file, "migrations/") {
					assertFilesExist(t, root, file)
				}
			}
		})
	}
}

// Sem banco, o endpoint tipado guarda os registros em memória e não gera migração.
func TestTypedEndpointWithoutDatabase(t *testing.T) {
	for _, config := range []project.Config{
		{ProjectName: "Sample", Language: "python", Framework: "fastapi", Architecture: "hexagonal"},
		{ProjectName: "Sample", Language: "go", Framework: "gin", Architecture: "modular"},
		{ProjectName: "Sample", Language: "python", Framework: "fastapi", Architecture: "modular", Database: "mongo"},
	} {
		root, m := generateProject(t, config)
		result, notes, err := generator.AddTypedEndpoint(root, m, "product", productFields, templates)
		if err != nil {
			t.Fatalf("%+v: %v", config, err)
		}
		if len(notes) != 1 || !strings.Contains(notes[0], "memória") {
			t.Errorf("%+v: notes = %v", config, notes)
		}
		for _, file := range result.Created {
			if strings.HasPrefix(file, "migrations/") {
				t.Errorf("%+v: migração criada sem banco SQL: %s", config, file)
			}
		}
	}
}

func TestTypedEndpointRejections(t *testing.T) {
	pythonRoot, pythonManifest := generateProject(t, project.Config{ProjectName: "Sample", Language: "python", Framework: "fastapi", Architecture: "modular", Database: "postgres"})
	nodeRoot, nodeManifest := generateProject(t, project.Config{ProjectName: "Sample", Language: "node", Runtime: "ts", Framework: "express", Architecture: "modular"})
	before := readTree(t, pythonRoot)
	for _, testCase := range []struct {
		root  string
		m     manifest.Manifest
		name  string
		specs []string
		want  string
	}{
		{pythonRoot, pythonManifest, "cart_item", []string{"cart:ref"}, `o endpoint "cart" não tem tabela`},
		{pythonRoot, pythonManifest, "product", []string{"price:money"}, "desconhecido"},
		{pythonRoot, pythonManifest, "template", []string{"name:str"}, "já existe"},
		{pythonRoot, pythonManifest, "class", []string{"name:str"}, "palavra reservada"},
		{nodeRoot, nodeManifest, "product", []string{"name:str"}, `campo "template_endpoint"`},
	} {
		if _, _, err := generator.AddTypedEndpoint(testCase.root, testCase.m, testCase.name, testCase.specs, templates); err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Errorf("AddTypedEndpoint(%s, %v) err = %v, want contendo %q", testCase.name, testCase.specs, err, testCase.want)
		}
	}
	if !maps.Equal(readTree(t, pythonRoot), before) {
		t.Error("AddTypedEndpoint recusado alterou o projeto")
	}

	// Um template_endpoint diferente do da arquitetura não tem templates tipados.
	custom := pythonManifest
	custom.TemplateEndpoint.Name = "resource"
	custom.TemplateEndpoint.Files = []string{"src/app/template/service.py"}
	if _, _, err := generator.AddTypedEndpoint(pythonRoot, custom, "product", []string{"name:str"}, templates); err == nil || !strings.Contains(err.Error(), "não são suportados") {
		t.Errorf("AddTypedEndpoint com exemplo customizado: err = %v", err)
	}

	// Uma linguagem sem implementação de TypedLanguage falha de forma explícita.
	unknownDir := t.TempDir()
	unknownLang := pythonManifest
	unknownLang.Language = "java"
	if err := manifest.Write(unknownDir, unknownLang); err != nil {
		t.Fatal(err)
	}
	if _, _, err := generator.AddTypedEndpoint(unknownDir, unknownLang, "product", []string{"name:str"}, templates); err == nil || !strings.Contains(err.Error(), "linguagem sem suporte a campos tipados") {
		t.Errorf("AddTypedEndpoint com linguagem sem TypedLanguage: err = %v", err)
	}
}

func TestCLIAddTypedEndpoint(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"new", "n:loja", "l:go", "f:gin", "a:layered", "d:postgres"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	projectDir := filepath.Join(tmp, "loja")
	chdirTemp(t, projectDir)

	stdout.Reset()
	args := []string{"add", "endpoint", "product", "name:str", "price:decimal", "image_url:str?"}
	if code := cli.Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) = %d, stderr = %q", args, code, stderr.String())
	}
	for _, want := range []string{"Endpoint product criado.", "000002_create_product", "internal/routes/product_test.go"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout sem %q:\n%s", want, stdout.String())
		}
	}
	assertFileContains(t, projectDir, "internal/types/product.go", "ImageURL *string")
	assertFileContains(t, projectDir, "internal/routes/routes.go", `api.PATCH("/product/:id", productHandler.Update)`)

	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"add", "endpoint", "n:order", "total:money"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "desconhecido") {
		t.Errorf("tipo inválido: code = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	if code := cli.Run([]string{"remove", "endpoint", "product"}, &stdout, &stderr); code != 0 {
		t.Fatalf("remove: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "internal/routes/product_test.go") || !strings.Contains(stdout.String(), "migrações de product") {
		t.Errorf("stdout do remove:\n%s", stdout.String())
	}
	assertFilesExist(t, projectDir, "migrations/000002_create_product.up.sql")
}
