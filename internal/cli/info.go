package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/sinueh/crudding/internal/generator"
	"github.com/sinueh/crudding/internal/manifest"
)

var validArchitectures = []string{"flat", "layered", "modular", "hexagonal", "clean_arch"}

// InfoUsage gera o texto de ajuda do comando "crudding info".
func InfoUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Lê o project.crud do projeto atual e exibe a stack configurada (linguagem,\n")
	b.WriteString("  framework e arquitetura) e os endpoints registrados.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding info\n")
	b.WriteString("  crudding info draw\n")
	return b.String()
}

// InfoDrawUsage gera o texto de ajuda do comando "crudding info draw".
func InfoDrawUsage() string {
	var b strings.Builder
	b.WriteString("como funciona:\n")
	b.WriteString("  Desenha em ASCII o diagrama da arquitetura do sistema.\n")
	b.WriteString("  - Sem argumentos (\"crudding info draw\"), lê o project.crud do projeto atual.\n")
	b.WriteString("  - Com argumento de arquitetura, exibe o desenho correspondente mesmo fora de um projeto.\n\n")
	b.WriteString("uso:\n")
	b.WriteString("  crudding info draw [<arquitetura> | arch:<arquitetura> | a:<arquitetura>] [lang:<linguagem> | l:<linguagem>]\n\n")
	b.WriteString("exemplos:\n")
	b.WriteString("  crudding info draw\n")
	b.WriteString("  crudding info draw hexagonal\n")
	b.WriteString("  crudding info draw a:clean_arch l:go\n")
	return b.String()
}

func printInfoReport(stdout io.Writer, root string, m manifest.Manifest) {
	cfg := m.Config()
	fmt.Fprintf(stdout, "Projeto: %s (%s)\n", m.Name, root)
	fmt.Fprintf(stdout, "Stack: %s\n", m.Stack())
	fmt.Fprintln(stdout, formatLanguageLine(cfg))
	fmt.Fprintf(stdout, "Framework: %s\n", cfg.Framework)
	fmt.Fprintf(stdout, "Arquitetura: %s\n", cfg.Architecture)
	if m.Database != "" {
		fmt.Fprintf(stdout, "Banco de dados: %s\n", m.Database)
	}
	if m.ORM != "" {
		fmt.Fprintf(stdout, "ORM: %s\n", m.ORM)
	}

	fmt.Fprintf(stdout, "\nEndpoints (%d):\n", len(m.Endpoints))
	if len(m.Endpoints) == 0 {
		fmt.Fprintln(stdout, "  (nenhum)")
	} else {
		for _, ep := range m.Endpoints {
			fmt.Fprintf(stdout, "  - %s\n", ep)
		}
	}
}

func runInfo(args []string, stdout, _ io.Writer) error {
	fs := flag.NewFlagSet("crudding info", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, InfoUsage())
			return nil
		}
		return err
	}

	tokens := fs.Args()
	if len(tokens) == 1 && tokens[0] == "help" {
		fmt.Fprint(stdout, InfoUsage())
		return nil
	}
	if len(tokens) > 0 {
		return fmt.Errorf("argumentos inesperados: %s", strings.Join(tokens, " "))
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return err
	}
	root, m, err := manifest.Find(workingDirectory)
	if err != nil {
		return err
	}
	printInfoReport(stdout, root, m)
	return nil
}

func parseInfoDrawArgs(args []string) (arch, lang string, showHelp bool, err error) {
	fs := flag.NewFlagSet("crudding info draw", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", "", true, nil
		}
		return "", "", false, err
	}

	tokens := fs.Args()
	if len(tokens) == 1 && tokens[0] == "help" {
		return "", "", true, nil
	}

	for _, token := range tokens {
		key, val, hasColon := strings.Cut(token, ":")
		if hasColon {
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			switch key {
			case "arch", "a":
				arch = val
			case "lang", "l":
				lang = val
			default:
				return "", "", false, fmt.Errorf("flag desconhecida: %s", token)
			}
			continue
		}
		if arch == "" {
			arch = strings.TrimSpace(token)
		} else {
			return "", "", false, fmt.Errorf("argumentos inesperados: %s", token)
		}
	}

	if lang != "" {
		if _, ok := generator.Lookup(lang); !ok {
			return "", "", false, fmt.Errorf("linguagem não suportada: %s", lang)
		}
	}
	if arch != "" && !slices.Contains(validArchitectures, arch) {
		return "", "", false, fmt.Errorf("arquitetura não suportada: %s", arch)
	}
	return arch, lang, false, nil
}

func runInfoDraw(args []string, stdout, _ io.Writer) error {
	argArch, argLang, showHelp, err := parseInfoDrawArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Fprint(stdout, InfoDrawUsage())
		return nil
	}

	var m manifest.Manifest
	if argArch == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		_, m, err = manifest.Find(workingDirectory)
		if err != nil {
			return err
		}
		if argLang != "" {
			m.Language = argLang
		}
	} else {
		m.Architecture = argArch
		m.Language = argLang
	}

	drawing, err := renderArchitectureASCII(m)
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, drawing)
	return nil
}

func padRight(text string, width int) string {
	n := utf8.RuneCountInString(text)
	if n >= width {
		return text
	}
	return text + strings.Repeat(" ", width-n)
}

func padCenter(text string, width int) string {
	n := utf8.RuneCountInString(text)
	if n >= width {
		return text
	}
	left := (width - n) / 2
	right := width - n - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func asciiBox(indent string, width int, doubleBorder bool, lines ...string) string {
	topLeft, topRight, bottomLeft, bottomRight, horiz, vert := "┌", "┐", "└", "┘", "─", "│"
	if doubleBorder {
		topLeft, topRight, bottomLeft, bottomRight, horiz, vert = "╔", "╗", "╚", "╝", "═", "║"
	}
	var b strings.Builder
	b.WriteString(indent + topLeft + strings.Repeat(horiz, width+2) + topRight + "\n")
	for _, line := range lines {
		b.WriteString(indent + vert + " " + padRight(line, width) + " " + vert + "\n")
	}
	b.WriteString(indent + bottomLeft + strings.Repeat(horiz, width+2) + bottomRight + "\n")
	return b.String()
}

func renderArchitectureASCII(m manifest.Manifest) (string, error) {
	if !slices.Contains(validArchitectures, m.Architecture) {
		return "", fmt.Errorf("arquitetura não suportada: %s", m.Architecture)
	}

	var b strings.Builder
	if m.Name != "" {
		fmt.Fprintf(&b, "Projeto: %s (%s)\n", m.Name, m.Stack())
	} else if m.Language != "" {
		fmt.Fprintf(&b, "Linguagem: %s\n", m.Language)
	}
	fmt.Fprintf(&b, "Arquitetura: %s (%s)\n\n", m.Architecture, architectureTitle(m.Architecture))

	switch m.Architecture {
	case "flat":
		b.WriteString(drawFlatASCII(m))
	case "layered":
		b.WriteString(drawLayeredASCII(m))
	case "modular":
		b.WriteString(drawModularASCII(m))
	case "hexagonal":
		b.WriteString(drawHexagonalASCII(m))
	case "clean_arch":
		b.WriteString(drawCleanArchASCII(m))
	}

	if len(m.Endpoints) > 0 {
		fmt.Fprintf(&b, "\nEndpoints: %s\n", strings.Join(m.Endpoints, ", "))
	}
	return b.String(), nil
}

func architectureTitle(arch string) string {
	switch arch {
	case "flat":
		return "Flat Architecture / Pacote Único"
	case "layered":
		return "Layered Architecture / Arquitetura em Camadas"
	case "modular":
		return "Modular Architecture / Vertical Slice"
	case "hexagonal":
		return "Hexagonal Architecture / Ports & Adapters"
	case "clean_arch":
		return "Clean Architecture / Camadas Concêntricas"
	default:
		return arch
	}
}

func drawFlatASCII(m manifest.Manifest) string {
	entry, routes, handlers, services, models := "main.go", "routes.go", "*_handler.go", "*_service.go", "*_model / *_types"
	switch m.Language {
	case "python":
		entry, routes, handlers, services, models = "src/main.py", "main.py", "*_controller.py", "*_service.py", "*_model.py"
	case "node":
		ext := nodeExt(m.Runtime)
		entry, routes, handlers, services, models = "src/main" + ext, "src/server" + ext, "src/tasks" + ext, "src/tasks" + ext, "src/tasks" + ext
	}

	var b strings.Builder
	b.WriteString("  [ Cliente HTTP ]\n")
	b.WriteString("         │\n")
	b.WriteString("         ▼\n")
	b.WriteString("┌─────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(&b, "│ %s │\n", padRight("Pacote Único ("+entry+")", 63))
	b.WriteString("│                                                                 │\n")
	b.WriteString("│  ┌────────────────┐   ┌──────────────────┐   ┌────────────────┐ │\n")
	b.WriteString("│  │     Routes     │──►│   Controllers    │──►│    Services    │ │\n")
	fmt.Fprintf(&b, "│  │ %s │   │ %s │   │ %s │ │\n",
		padCenter(routes, 14),
		padCenter(handlers, 16),
		padCenter(services, 14),
	)
	b.WriteString("│  └────────────────┘   └──────────────────┘   └───────┬────────┘ │\n")
	b.WriteString("│                                                      │          │\n")
	b.WriteString("│                                                      ▼          │\n")
	b.WriteString("│                                           ┌───────────────────┐ │\n")
	b.WriteString("│                                           │   Models / Data   │ │\n")
	fmt.Fprintf(&b, "│                                           │ %s │ │\n", padCenter(models, 17))
	b.WriteString("│                                           └───────────────────┘ │\n")
	b.WriteString("└─────────────────────────────────────────────────────────────────┘\n")
	return b.String()
}

func drawLayeredASCII(m manifest.Manifest) string {
	entryPath := "cmd/api/main.go -> internal/routes"
	transportPath := "internal/handlers"
	servicePath := "internal/service"
	modelPath := "internal/models & internal/types"

	switch m.Language {
	case "python":
		entryPath = "src/app/main.py"
		transportPath = "src/app/controllers"
		servicePath = "src/app/services"
		modelPath = "src/app/models"
	case "node":
		ext := nodeExt(m.Runtime)
		entryPath = "src/main" + ext + " -> src/http & src/routes"
		transportPath = "src/controllers"
		servicePath = "src/services"
		modelPath = "src/models"
	}

	const width = 49
	var b strings.Builder
	b.WriteString("                  [ Cliente HTTP ]\n")
	b.WriteString("                         │\n")
	b.WriteString("                         ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Entrypoint & Router",
		entryPath,
	))
	b.WriteString("                         │\n")
	b.WriteString("                         ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Presentation / Transport Layer (Controllers)",
		transportPath,
	))
	b.WriteString("                         │\n")
	b.WriteString("                         ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Business Logic Layer (Services)",
		servicePath,
	))
	b.WriteString("                         │\n")
	b.WriteString("                         ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Data & Domain Layer (Models / Types)",
		modelPath,
	))
	return b.String()
}

func drawModularASCII(m manifest.Manifest) string {
	entryPath := "cmd/api/main.go -> internal/routes"
	modPrefix := "internal/"
	sliceItems := "handler.go -> service.go -> model.go / types.go"

	switch m.Language {
	case "python":
		entryPath = "src/app/main.py"
		modPrefix = "src/app/"
		sliceItems = "controller.py -> service.py -> models.py"
	case "node":
		ext := nodeExt(m.Runtime)
		entryPath = "src/main" + ext + " -> src/shared/http"
		modPrefix = "src/modules/"
		sliceItems = "routes -> controller -> service -> model"
	}

	firstMod := "template"
	secondMod := "<feature>"
	if len(m.Endpoints) >= 1 {
		firstMod = m.Endpoints[0]
	}
	if len(m.Endpoints) >= 2 {
		secondMod = m.Endpoints[1]
	}

	const width = 55
	var b strings.Builder
	b.WriteString("                     [ Cliente HTTP ]\n")
	b.WriteString("                            │\n")
	b.WriteString("                            ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Entrypoint & Router",
		entryPath,
	))
	b.WriteString("               ┌────────────┴────────────┐\n")
	b.WriteString("               ▼                         ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		fmt.Sprintf("Módulo: %s (%s%s/)", firstMod, modPrefix, firstMod),
		"  " + sliceItems,
	))
	b.WriteString(asciiBox("  ", width, false,
		fmt.Sprintf("Módulo: %s (%s%s/)", secondMod, modPrefix, secondMod),
		"  " + sliceItems,
	))
	return b.String()
}

func drawHexagonalASCII(m manifest.Manifest) string {
	drivingAdapter := "internal/handlers & internal/routes"
	drivingPort := "internal/ports (Service Interface)"
	core := "internal/application & internal/domain"
	drivenPort := "internal/ports (Repository Interface)"
	drivenAdapter := "internal/adapters/postgres"

	switch m.Language {
	case "python":
		drivingAdapter = "src/app/routers & src/app/main.py"
		drivingPort = "src/app/application (Service)"
		core = "src/app/application & src/app/domain"
		drivenPort = "src/app/ports (Repository Port)"
		drivenAdapter = "src/app/infrastructure"
	case "node":
		drivingAdapter = "src/adapters/http & src/entrypoints"
		drivingPort = "src/ports (Driving Port / Service)"
		core = "src/domain (Entities & Manager)"
		drivenPort = "src/ports (Driven Port / Repository)"
		drivenAdapter = "src/adapters/persistence"
	}

	const width = 51
	var b strings.Builder
	b.WriteString("                   [ Cliente HTTP ]\n")
	b.WriteString("                          │\n")
	b.WriteString("                          ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Driving Adapter (Primary / HTTP)",
		drivingAdapter,
	))
	b.WriteString("                          │\n")
	b.WriteString("                          ▼\n")
	fmt.Fprintf(&b, "        ( Driving Port: %s )\n", drivingPort)
	b.WriteString("                          │\n")
	b.WriteString("                          ▼\n")
	b.WriteString(asciiBox("  ", width, true,
		"HEXAGON CORE (Application & Domain)",
		core,
	))
	b.WriteString("                          │\n")
	b.WriteString("                          ▼\n")
	fmt.Fprintf(&b, "        ( Driven Port: %s )\n", drivenPort)
	b.WriteString("                          │\n")
	b.WriteString("                          ▼\n")
	b.WriteString(asciiBox("  ", width, false,
		"Driven Adapter (Secondary / Persistence)",
		drivenAdapter,
	))
	return b.String()
}

func drawCleanArchASCII(m manifest.Manifest) string {
	frameworks := "cmd/api/main.go, internal/routes, internal/infrastructure"
	adapters := "internal/handlers & internal/ports"
	useCases := "internal/application (Use Cases)"
	entities := "internal/domain (Entities)"

	switch m.Language {
	case "python":
		frameworks = "src/app/main.py, src/app/frameworks_and_drivers/web"
		adapters = "src/app/interface_adapters (controllers, presenters, persistence)"
		useCases = "src/app/application (use_cases & port/outbound)"
		entities = "src/app/domain/model (Entities)"
	case "node":
		frameworks = "src/frameworks (main, http)"
		adapters = "src/adapters (controllers, presenters, repositories)"
		useCases = "src/use-cases (& ports)"
		entities = "src/entities (Enterprise Entities)"
	}

	var b strings.Builder
	b.WriteString("  ┌─────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(&b, "  │ Frameworks & Drivers (Web / Persistence)                            │\n")
	fmt.Fprintf(&b, "  │ %s │\n", padRight(frameworks, 67))
	b.WriteString("  │   │                                                             ▲   │\n")
	b.WriteString("  │   ▼                                                             │   │\n")
	b.WriteString("  │ ┌─────────────────────────────────────────────────────────────────┐ │\n")
	b.WriteString("  │ │ Interface Adapters (Controllers / Presenters / Gateways)        │ │\n")
	fmt.Fprintf(&b, "  │ │ %s │ │\n", padRight(adapters, 63))
	b.WriteString("  │ │   │                                                         ▲   │ │\n")
	b.WriteString("  │ │   ▼                                                         │   │ │\n")
	b.WriteString("  │ │ ┌─────────────────────────────────────────────────────────────┐ │ │\n")
	b.WriteString("  │ │ │ Application Business Rules (Use Cases)                      │ │ │\n")
	fmt.Fprintf(&b, "  │ │ │ %s │ │ │\n", padRight(useCases, 59))
	b.WriteString("  │ │ │   │                                                         │ │ │\n")
	b.WriteString("  │ │ │   ▼                                                         │ │ │\n")
	b.WriteString("  │ │ │ ╔═════════════════════════════════════════════════════════╗ │ │ │\n")
	b.WriteString("  │ │ │ ║ Enterprise Business Rules (Domain / Entities)           ║ │ │ │\n")
	fmt.Fprintf(&b, "  │ │ │ ║ %s ║ │ │ │\n", padRight(entities, 55))
	b.WriteString("  │ │ │ ╚═════════════════════════════════════════════════════════╝ │ │ │\n")
	b.WriteString("  │ │ └─────────────────────────────────────────────────────────────┘ │ │\n")
	b.WriteString("  │ └─────────────────────────────────────────────────────────────────┘ │\n")
	b.WriteString("  └─────────────────────────────────────────────────────────────────────┘\n")
	return b.String()
}

func nodeExt(runtime string) string {
	if runtime == "js" {
		return ".js"
	}
	return ".ts"
}

