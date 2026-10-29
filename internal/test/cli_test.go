package test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sinueh/crudding/internal/cli"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
}

func TestCLIHelpAndUsage(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(%q) exit code = %d, want 0", arg, code)
		}
		if stderr.Len() != 0 {
			t.Fatalf("Run(%q) unexpected stderr = %q", arg, stderr.String())
		}
		if got := stdout.String(); !strings.Contains(got, "crudding add endpoint") || !strings.Contains(got, "uso:") {
			t.Fatalf("Run(%q) stdout = %q, missing usage entries", arg, got)
		}
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"foo", "bar"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run(unknown) exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("Run(unknown) unexpected stdout = %q", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, "comando desconhecido: foo bar") || !strings.Contains(got, cli.Usage()) {
		t.Fatalf("Run(unknown) stderr = %q", got)
	}
}

func TestCLICommandsRegistry(t *testing.T) {
	cmds := cli.Commands()
	if len(cmds) < 2 {
		t.Fatalf("len(Commands()) = %d, want >= 2", len(cmds))
	}
	for _, cmd := range cmds {
		if cmd.Summary == "" {
			t.Errorf("command %q has empty Summary", cmd.Args)
		}
		if cmd.Run == nil {
			t.Errorf("command %q has nil Run", cmd.Args)
		}
	}
}

func TestCLINewHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{"new", arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(new %q) exit code = %d, want 0", arg, code)
		}
		if stderr.Len() != 0 {
			t.Fatalf("Run(new %q) unexpected stderr = %q", arg, stderr.String())
		}
		got := stdout.String()
		for _, section := range []string{
			"como funciona:",
			"como cada flag funciona:",
			"name:<nome> ou n:<nome>",
			"exemplos:",
			"crudding new name:meu-app lang:go framework:gin arch:hexagonal",
			"crudding new n:meu-app l:go f:gin a:hexagonal",
			"lang/l:\n  python\n  go\n  node",
			"runtime/r:\n  js\n  ts",
			"framework/f:\n  raw",
			"arch/a:\n  flat",
		} {
			if !strings.Contains(got, section) {
				t.Fatalf("Run(new %q) stdout missing %q:\n%s", arg, section, got)
			}
		}
	}
}

func TestCLINewListOptionsByKey(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{
			args: []string{"new", "l"},
			want: "lang/l:\n  python\n  go\n  node\n",
		},
		{
			args: []string{"new", "lang"},
			want: "lang/l:\n  python\n  go\n  node\n",
		},
		{
			args: []string{"new", "r"},
			want: "runtime/r:\n  js\n  ts\n",
		},
		{
			args: []string{"new", "f"},
			want: "framework/f:\n  raw\n  fastapi\n",
		},
		{
			args: []string{"new", "l:go", "f"},
			want: "framework/f:\n  raw\n  gin\n  fiber\n  chi\n",
		},
		{
			args: []string{"new", "a"},
			want: "arch/a:\n  flat\n  layered\n  modular\n  hexagonal\n  clean_arch\n",
		},
		{
			args: []string{"new", "o"},
			want: "orm/o:\n  none\n  sqlalchemy\n  gorm\n  prisma\n  typeorm\n",
		},
		{
			args: []string{"new", "orm"},
			want: "orm/o:\n  none\n  sqlalchemy\n  gorm\n  prisma\n  typeorm\n",
		},
		{
			args: []string{"new", "l:go", "o"},
			want: "orm/o:\n  none\n  gorm\n",
		},
		{
			args: []string{"new", "l:python", "o"},
			want: "orm/o:\n  none\n  sqlalchemy\n",
		},
	}

	for _, tc := range cases {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run(%v) exit code = %d, stderr = %q", tc.args, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.want) {
				t.Fatalf("Run(%v) stdout = %q, want substring %q", tc.args, stdout.String(), tc.want)
			}
		})
	}
}

func TestCLINewNonInteractive(t *testing.T) {
	cases := []struct {
		name        string
		projectName string
		args        []string
	}{
		{
			name:        "full keys",
			projectName: "meu-app",
			args:        []string{"new", "name:meu-app", "lang:go", "framework:gin", "arch:hexagonal", "runtime:xpto"},
		},
		{
			name:        "short keys",
			projectName: "meu-app-curto",
			args:        []string{"new", "n:meu-app-curto", "l:go", "f:gin", "a:hexagonal", "r:xpto"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			chdirTemp(t, tmp)

			var stdout, stderr bytes.Buffer
			code := cli.Run(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run(%v) exit code = %d, stderr = %q", tc.args, code, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr = %q", stderr.String())
			}

			projectDir := filepath.Join(tmp, tc.projectName)
			_, m, err := manifest.Find(projectDir)
			if err != nil {
				t.Fatalf("manifest.Find() error = %v", err)
			}
			want := project.Config{
				ProjectName:  tc.projectName,
				Language:     "go",
				Framework:    "gin",
				Architecture: "hexagonal",
			}
			if m.Config() != want {
				t.Fatalf("manifest config = %+v, want %+v", m.Config(), want)
			}
			assertFilesExist(t, projectDir, "cmd/api/main.go", "internal/routes/routes.go")
		})
	}
}

func TestCLINewNonInteractiveNodeWithRuntime(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	args := []string{"new", "n:node-app", "l:node", "r:ts", "f:express", "a:layered"}
	code := cli.Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", args, code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "node-app")
	_, m, err := manifest.Find(projectDir)
	if err != nil {
		t.Fatalf("manifest.Find() error = %v", err)
	}
	want := project.Config{
		ProjectName:  "node-app",
		Language:     "node",
		Runtime:      "ts",
		Framework:    "express",
		Architecture: "layered",
	}
	if m.Config() != want {
		t.Fatalf("manifest config = %+v, want %+v", m.Config(), want)
	}
}

func TestCLINewNonInteractiveValidationErrors(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "new without args requires help",
			args: []string{"new"},
			want: "crudding new -h",
		},
		{
			name: "missing flags",
			args: []string{"new", "name:meu-app", "lang:go"},
			want: "flags obrigatórias ausentes: framework/f, arch/a",
		},
		{
			name: "missing node runtime",
			args: []string{"new", "n:node-app", "l:node", "f:express", "a:flat"},
			want: "flags obrigatórias ausentes: runtime/r",
		},
		{
			name: "name key without value does not list options",
			args: []string{"new", "n"},
			want: "nenhum nome de projeto informado",
		},
		{
			name: "invalid project name",
			args: []string{"new", "name:../bad", "lang:go", "framework:raw", "arch:flat"},
			want: "nome inválido",
		},
		{
			name: "unknown flag",
			args: []string{"new", "n:meu-app", "l:go", "f:raw", "a:flat", "extra"},
			want: "flag desconhecida: extra",
		},
		{
			name: "orm without database",
			args: []string{"new", "n:app-err", "l:go", "f:gin", "a:hexagonal", "orm:gorm"},
			want: "nenhum banco de dados foi selecionado",
		},
		{
			name: "orm with mongo",
			args: []string{"new", "n:app-err", "l:go", "f:gin", "a:hexagonal", "db:mongo", "orm:gorm"},
			want: "não é suportado com banco mongo",
		},
		{
			name: "unsupported orm",
			args: []string{"new", "n:app-err", "l:go", "f:gin", "a:hexagonal", "db:postgres", "orm:hibernate"},
			want: "ORM Go não suportado: hibernate",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run(tc.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("Run(%v) exit code = %d, want 1", tc.args, code)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), tc.want)
			}
		})
	}
}

func TestCLINewNonInteractiveWithORM(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	args := []string{"new", "n:go-gorm-app", "l:go", "f:gin", "a:hexagonal", "db:postgres", "orm:gorm"}
	code := cli.Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", args, code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "go-gorm-app")
	_, m, err := manifest.Find(projectDir)
	if err != nil {
		t.Fatalf("manifest.Find() error = %v", err)
	}
	want := project.Config{
		ProjectName:  "go-gorm-app",
		Language:     "go",
		Framework:    "gin",
		Architecture: "hexagonal",
		Database:     "postgres",
		ORM:          "gorm",
	}
	if m.Config() != want {
		t.Fatalf("manifest config = %+v, want %+v", m.Config(), want)
	}
	if got := m.Stack(); got != "go + gin + hexagonal + postgres + gorm" {
		t.Fatalf("manifest stack = %q, want %q", got, "go + gin + hexagonal + postgres + gorm")
	}
}

func TestCLIAddEndpointNonInteractive(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	newArgs := []string{"new", "n:meu-app", "l:go", "f:gin", "a:hexagonal"}
	if code := cli.Run(newArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new) exit code = %d, stderr = %q", code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "meu-app")
	chdirTemp(t, projectDir)

	stdout.Reset()
	stderr.Reset()
	addArgs := []string{"add", "endpoint", "product"}
	if code := cli.Run(addArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", addArgs, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Endpoint product criado.") {
		t.Fatalf("stdout = %q, missing endpoint summary", stdout.String())
	}
	assertFilesExist(t, projectDir, "internal/handlers/product.go", "internal/application/product.go")

	stdout.Reset()
	stderr.Reset()
	addFlagArgs := []string{"add", "endpoint", "n:order_item"}
	if code := cli.Run(addFlagArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", addFlagArgs, code, stderr.String())
	}
	assertFilesExist(t, projectDir, "internal/handlers/order_item.go", "internal/application/order_item.go")

	stdout.Reset()
	stderr.Reset()
	badArgs := []string{"add", "endpoint", "Invalid-Name"}
	if code := cli.Run(badArgs, &stdout, &stderr); code != 1 {
		t.Fatalf("Run(%v) exit code = %d, want 1", badArgs, code)
	}
	if !strings.Contains(stderr.String(), "nome de endpoint inválido") {
		t.Fatalf("stderr = %q, want invalid endpoint error", stderr.String())
	}

	// Help flags inside project
	for _, h := range []string{"-h", "--help", "help"} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run([]string{"add", "endpoint", h}, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(add endpoint %s) exit code = %d, want 0", h, code)
		}
		if !strings.Contains(stdout.String(), "crudding add endpoint") {
			t.Fatalf("stdout = %q, missing add endpoint usage", stdout.String())
		}
	}
}

func TestCLIAddEndpointHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{"add", "endpoint", arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(add endpoint %q) exit code = %d, want 0", arg, code)
		}
		if stderr.Len() != 0 {
			t.Fatalf("Run(add endpoint %q) unexpected stderr = %q", arg, stderr.String())
		}
		got := stdout.String()
		for _, section := range []string{
			"como funciona:",
			"uso:",
			"crudding add endpoint",
			"exemplos:",
		} {
			if !strings.Contains(got, section) {
				t.Fatalf("Run(add endpoint %q) stdout missing %q:\n%s", arg, section, got)
			}
		}
	}
}

func TestCLIAddDBHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{"add", "db", arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(add db %q) exit code = %d, want 0", arg, code)
		}
		if stderr.Len() != 0 {
			t.Fatalf("Run(add db %q) unexpected stderr = %q", arg, stderr.String())
		}
		got := stdout.String()
		for _, section := range []string{
			"como funciona:",
			"uso:",
			"crudding add db",
			"bancos suportados:",
			"postgres, mysql, mongo",
			"exemplos:",
		} {
			if !strings.Contains(got, section) {
				t.Fatalf("Run(add db %q) stdout missing %q:\n%s", arg, section, got)
			}
		}
	}
}


func TestCLIRemoveEndpoint(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	newArgs := []string{"new", "n:meu-app", "l:go", "f:gin", "a:hexagonal"}
	if code := cli.Run(newArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new) exit code = %d, stderr = %q", code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "meu-app")
	chdirTemp(t, projectDir)

	// Adiciona product e order_item
	cli.Run([]string{"add", "endpoint", "product"}, &stdout, &stderr)
	cli.Run([]string{"add", "endpoint", "order_item"}, &stdout, &stderr)
	assertFilesExist(t, projectDir, "internal/handlers/product.go", "internal/handlers/order_item.go")

	// Remover product com argumento posicional
	stdout.Reset()
	stderr.Reset()
	removeArgs := []string{"remove", "endpoint", "product"}
	if code := cli.Run(removeArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", removeArgs, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Endpoint product removido.") {
		t.Fatalf("stdout = %q, missing remove summary", stdout.String())
	}
	assertFilesAbsent(t, projectDir, "internal/handlers/product.go")
	assertFilesExist(t, projectDir, "internal/handlers/order_item.go")

	// Remover order_item com flag n:
	stdout.Reset()
	stderr.Reset()
	removeFlagArgs := []string{"remove", "endpoint", "n:order_item"}
	if code := cli.Run(removeFlagArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", removeFlagArgs, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Endpoint order_item removido.") {
		t.Fatalf("stdout = %q, missing remove summary", stdout.String())
	}
	assertFilesAbsent(t, projectDir, "internal/handlers/order_item.go")

	// Tentar remover endpoint já removido
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"remove", "endpoint", "product"}, &stdout, &stderr); code != 1 {
		t.Fatalf("remover endpoint inexistente exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "não encontrado") {
		t.Fatalf("stderr = %q, want não encontrado error", stderr.String())
	}

	// Nome inválido
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"remove", "endpoint", "Bad-Name"}, &stdout, &stderr); code != 1 {
		t.Fatalf("remover com nome inválido exit code = %d, want 1", code)
	}

	// Help flags
	for _, h := range []string{"-h", "--help", "help"} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run([]string{"remove", "endpoint", h}, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(remove endpoint %s) exit code = %d, want 0", h, code)
		}
		if !strings.Contains(stdout.String(), "crudding remove endpoint") {
			t.Fatalf("stdout = %q, missing remove endpoint usage", stdout.String())
		}
	}
}


func TestCLIStacks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"stacks"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(stacks) exit code = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run(stacks) unexpected stderr = %q", stderr.String())
	}

	got := stdout.String()
	for _, want := range []string{
		"python (Python)",
		"├── frameworks: raw, fastapi, django, flask",
		"└── architectures: flat, layered, modular, hexagonal, clean_arch",
		"go (Go)",
		"├── frameworks: raw, gin, fiber, chi",
		"node (Node)",
		"├── runtimes: js, ts",
		"├── frameworks: raw, express, nest, fastify",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run(stacks) stdout missing %q:\n%s", want, got)
		}
	}
}

func TestCLIStacksFilterAndHelp(t *testing.T) {
	for _, arg := range []string{"go", "lang:go", "l:go"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{"stacks", arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(stacks %q) exit code = %d, stderr = %q", arg, code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, "go (Go)") || strings.Contains(got, "python (Python)") {
			t.Fatalf("Run(stacks %q) unexpected stdout:\n%s", arg, got)
		}
	}

	for _, arg := range []string{"-h", "--help", "help"} {
		var stdout, stderr bytes.Buffer
		code := cli.Run([]string{"stacks", arg}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("Run(stacks %q) exit code = %d, want 0", arg, code)
		}
		if !strings.Contains(stdout.String(), "crudding stacks") {
			t.Fatalf("Run(stacks %q) stdout = %q, missing usage", arg, stdout.String())
		}
	}

	for _, args := range [][]string{
		{"stacks", "rust"},
		{"stacks", "f:gin"},
		{"stacks", "go", "node"},
	} {
		var stdout, stderr bytes.Buffer
		code := cli.Run(args, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("Run(%v) exit code = %d, want 1", args, code)
		}
		if stderr.Len() == 0 {
			t.Fatalf("Run(%v) expected error message on stderr", args)
		}
	}
}

func TestCLIInfo(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"new", "n:go-api", "l:go", "f:gin", "a:hexagonal"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new go) exit code = %d, stderr = %q", code, stderr.String())
	}

	goDir := filepath.Join(tmp, "go-api")
	chdirTemp(t, goDir)

	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"add", "endpoint", "product"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(add endpoint) exit code = %d, stderr = %q", code, stderr.String())
	}

	_, m, err := manifest.Find(goDir)
	if err != nil {
		t.Fatalf("manifest.Find() error = %v", err)
	}
	if len(m.Endpoints) != 2 || m.Endpoints[0] != "user" || m.Endpoints[1] != "product" {
		t.Fatalf("manifest.Endpoints = %v, want [user product]", m.Endpoints)
	}

	stdout.Reset()
	stderr.Reset()
	code := cli.Run([]string{"info"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(info) exit code = %d, stderr = %q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"Projeto: go-api",
		"Stack: go + gin + hexagonal",
		"Linguagem: go",
		"Framework: gin",
		"Arquitetura: hexagonal",
		"Endpoints (2):",
		"  - user",
		"  - product",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run(info) stdout missing %q:\n%s", want, got)
		}
	}

	// Projeto Node (modular)
	chdirTemp(t, tmp)
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"new", "n:node-api", "l:node", "r:ts", "f:express", "a:modular"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new node) exit code = %d, stderr = %q", code, stderr.String())
	}
	chdirTemp(t, filepath.Join(tmp, "node-api"))

	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"info"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(info node) exit code = %d, stderr = %q", code, stderr.String())
	}
	gotNode := stdout.String()
	for _, want := range []string{
		"Projeto: node-api",
		"Stack: node/ts + express + modular",
		"Linguagem/motor: node/ts",
		"Endpoints (2):",
		"  - catalog",
		"  - orders",
	} {
		if !strings.Contains(gotNode, want) {
			t.Fatalf("Run(info node) stdout missing %q:\n%s", want, gotNode)
		}
	}

	for _, arg := range []string{"-h", "--help", "help"} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run([]string{"info", arg}, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(info %q) exit code = %d, want 0", arg, code)
		}
		if !strings.Contains(stdout.String(), "crudding info") {
			t.Fatalf("Run(info %q) missing usage:\n%s", arg, stdout.String())
		}
	}
}

func TestCLIInfoDraw(t *testing.T) {
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"new", "n:go-hex", "l:go", "f:gin", "a:hexagonal"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(new) exit code = %d, stderr = %q", code, stderr.String())
	}

	projectDir := filepath.Join(tmp, "go-hex")
	chdirTemp(t, projectDir)

	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"info", "draw"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(info draw) exit code = %d, stderr = %q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"Projeto: go-hex (go + gin + hexagonal)",
		"Arquitetura: hexagonal",
		"Driving Adapter",
		"HEXAGON CORE",
		"Driven Adapter",
		"internal/adapters/postgres",
		"Endpoints: user",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run(info draw) stdout missing %q:\n%s", want, got)
		}
	}

	// Test direct architecture argument for all architectures outside a project
	chdirTemp(t, tmp)
	for _, arch := range []string{"flat", "layered", "modular", "hexagonal", "clean_arch"} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run([]string{"info", "draw", arch}, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(info draw %q) exit code = %d, stderr = %q", arch, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Arquitetura: "+arch) || !strings.Contains(stdout.String(), "┌") {
			t.Fatalf("Run(info draw %q) unexpected stdout:\n%s", arch, stdout.String())
		}
	}

	// Test key:value flags
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"info", "draw", "a:clean_arch", "l:python"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(info draw a:clean_arch l:python) exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "frameworks_and_drivers/web") {
		t.Fatalf("Run(info draw a:clean_arch l:python) missing python path:\n%s", stdout.String())
	}

	// Help flags
	for _, arg := range []string{"-h", "--help", "help"} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run([]string{"info", "draw", arg}, &stdout, &stderr); code != 0 {
			t.Fatalf("Run(info draw %q) exit code = %d, want 0", arg, code)
		}
		if !strings.Contains(stdout.String(), "crudding info draw") {
			t.Fatalf("Run(info draw %q) missing usage:\n%s", arg, stdout.String())
		}
	}

	// Validation errors
	for _, args := range [][]string{
		{"info", "draw"},
		{"info", "draw", "invalid_arch"},
		{"info", "draw", "a:hexagonal", "l:rust"},
		{"info", "draw", "foo:bar"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("Run(%v) exit code = %d, want 1", args, code)
		}
	}
}

func TestCLIInit(t *testing.T) {
	root := newTemplateDir(t, "meu-projeto", map[string]string{"main.go": "package main\n", ".env": "SECRET=1"})
	chdirTemp(t, root)

	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(init) exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "project.crud gravado") {
		t.Errorf("stdout = %q", stdout.String())
	}
	_, m, err := manifest.Find(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "meu-projeto" || m.Architecture != manifest.CustomArchitecture || strings.Join(m.Files, ",") != ".env,main.go" || len(m.Ignore) != 0 {
		t.Errorf("manifesto = %+v", m)
	}

	writeFile(t, root, "novo.go", "package main\n")
	if code := cli.Run([]string{"init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(init) de novo: exit code = %d, stderr = %q", code, stderr.String())
	}
	if _, m, _ = manifest.Find(root); strings.Join(m.Files, ",") != ".env,main.go,novo.go" {
		t.Errorf("Files após rodar de novo = %v", m.Files)
	}

	stderr.Reset()
	if code := cli.Run([]string{"init", "extra"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "argumentos inesperados") {
		t.Errorf("Run(init extra) = %d, stderr = %q", code, stderr.String())
	}
}

func TestCLITemplate(t *testing.T) {
	t.Setenv("CRUDDING_TEMPLATES_DIR", t.TempDir())
	templateDir := copyExampleTemplate(t)
	writeFile(t, templateDir, "NOTAS.md", "criado depois do init\n")
	tmp := t.TempDir()
	chdirTemp(t, tmp)

	var stdout, stderr bytes.Buffer
	// Salva o template
	args := []string{"template", templateDir, "n:meu-template"}
	if code := cli.Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", args, code, stderr.String())
	}
	for _, snippet := range []string{"Aviso: 1 arquivo", "NOTAS.md", "Template meu-template salvo com sucesso", "crudding template list"} {
		if !strings.Contains(stdout.String(), snippet) {
			t.Errorf("stdout não contém %q:\n%s", snippet, stdout.String())
		}
	}

	// Lista os templates salvos
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{"template", "list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(template list) exit code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "meu-template") {
		t.Errorf("stdout de template list não contém meu-template:\n%s", stdout.String())
	}

	// Gera projeto a partir do template salvo
	stdout.Reset()
	stderr.Reset()
	argsUse := []string{"template", "use", "meu-template", "n:minha-api"}
	if code := cli.Run(argsUse, &stdout, &stderr); code != 0 {
		t.Fatalf("Run(%v) exit code = %d, stderr = %q", argsUse, code, stderr.String())
	}
	for _, snippet := range []string{"Projeto gerado em", "Template: meu-template"} {
		if !strings.Contains(stdout.String(), snippet) {
			t.Errorf("stdout não contém %q:\n%s", snippet, stdout.String())
		}
	}
	assertFileContains(t, filepath.Join(tmp, "minha-api"), "go.mod", "module minha-api")
	assertFilesAbsent(t, filepath.Join(tmp, "minha-api"), "NOTAS.md")

	for _, testCase := range []struct {
		args []string
		want string
	}{
		{[]string{"template"}, "nenhuma pasta de template"},
		{[]string{"template", filepath.Join(tmp, "nao-existe"), "outro"}, "não encontrado"},
		{[]string{"template", templateDir, "a/b"}, "nome inválido"},
		{[]string{"template", templateDir, "outro", "extra"}, "argumentos inesperados"},
		{[]string{"template", "use"}, "nenhum template informado"},
		{[]string{"template", "use", "nao-existe"}, "não encontrado"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run(testCase.args, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), testCase.want) {
			t.Errorf("Run(%v) = %d, stderr = %q, want contendo %q", testCase.args, code, stderr.String(), testCase.want)
		}
	}
	assertFilesAbsent(t, tmp, "outro")

	stdout.Reset()
	if code := cli.Run([]string{"template", "-h"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "crudding template <pasta>") {
		t.Errorf("Run(template -h) = %d, stdout = %q", code, stdout.String())
	}

	stdout.Reset()
	if code := cli.Run([]string{"template", "list", "-h"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "crudding template list") {
		t.Errorf("Run(template list -h) = %d, stdout = %q", code, stdout.String())
	}
}
