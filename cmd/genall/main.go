// Command genall gera combinações suportadas de linguagem, runtime,
// framework e arquitetura, nomeando cada projeto como
// linguagem[-runtime]-framework-arquitetura (ex: go-gin-layered, node-ts-express-flat).
//
// Uso:
//
//	go run ./cmd/genall [opções] [diretório de saída]
//
// Exemplos:
//
//	go run ./cmd/genall
//	go run ./cmd/genall lang:go
//	go run ./cmd/genall l:go f:gin a:hexagonal
//	go run ./cmd/genall -l go -f gin,fiber -a layered,hexagonal
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh-andreata/crudding"
	"github.com/sinueh-andreata/crudding/internal/generator"
	"github.com/sinueh-andreata/crudding/internal/project"
)

type filter struct {
	langs      []string
	runtimes   []string
	frameworks []string
	archs      []string
	output     string
}

func splitCSV(val string) []string {
	var out []string
	for _, part := range strings.Split(val, ",") {
		part = strings.TrimSpace(part)
		if part != "" && !slices.Contains(out, part) {
			out = append(out, part)
		}
	}
	return out
}

func matchesFilter(allowed []string, value string) bool {
	return len(allowed) == 0 || slices.Contains(allowed, value)
}

func parseArgs(args []string) (filter, error) {
	f := filter{output: "generated"}

	fs := flag.NewFlagSet("genall", flag.ContinueOnError)
	var langFlag, runtimeFlag, fwFlag, archFlag, outFlag string
	fs.StringVar(&langFlag, "lang", "", "filtrar linguagem (ex: go, python, node)")
	fs.StringVar(&langFlag, "l", "", "atalho para -lang")
	fs.StringVar(&runtimeFlag, "runtime", "", "filtrar runtime (ex: js, ts)")
	fs.StringVar(&runtimeFlag, "r", "", "atalho para -runtime")
	fs.StringVar(&fwFlag, "framework", "", "filtrar framework (ex: gin, fastapi, express)")
	fs.StringVar(&fwFlag, "f", "", "atalho para -framework")
	fs.StringVar(&archFlag, "arch", "", "filtrar arquitetura (ex: flat, layered, modular, hexagonal, clean_arch)")
	fs.StringVar(&archFlag, "a", "", "atalho para -arch")
	fs.StringVar(&outFlag, "out", "", "diretório de saída (padrão: generated)")
	fs.StringVar(&outFlag, "o", "", "atalho para -out")

	if err := fs.Parse(args); err != nil {
		return filter{}, err
	}

	f.langs = append(f.langs, splitCSV(langFlag)...)
	f.runtimes = append(f.runtimes, splitCSV(runtimeFlag)...)
	f.frameworks = append(f.frameworks, splitCSV(fwFlag)...)
	f.archs = append(f.archs, splitCSV(archFlag)...)
	if outFlag != "" {
		f.output = outFlag
	}

	var positionalOut string
	for _, token := range fs.Args() {
		key, val, hasColon := strings.Cut(token, ":")
		if !hasColon {
			if positionalOut != "" {
				return filter{}, fmt.Errorf("argumento inesperado: %s", token)
			}
			positionalOut = token
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if val == "" {
			return filter{}, fmt.Errorf("valor vazio para filtro %q", key)
		}
		switch key {
		case "lang", "l":
			f.langs = append(f.langs, splitCSV(val)...)
		case "runtime", "r":
			f.runtimes = append(f.runtimes, splitCSV(val)...)
		case "framework", "f":
			f.frameworks = append(f.frameworks, splitCSV(val)...)
		case "arch", "a":
			f.archs = append(f.archs, splitCSV(val)...)
		case "out", "o":
			f.output = val
		default:
			return filter{}, fmt.Errorf("filtro desconhecido: %s (use lang/l, runtime/r, framework/f, arch/a, out/o)", key)
		}
	}
	if positionalOut != "" {
		f.output = positionalOut
	}
	return f, nil
}

func configs(f filter) []project.Config {
	var all []project.Config
	for _, language := range generator.Languages() {
		if !matchesFilter(f.langs, language.Name) {
			continue
		}
		runtimes := language.Runtimes
		if len(runtimes) == 0 {
			runtimes = []string{""}
		}
		for _, runtime := range runtimes {
			if runtime != "" && !matchesFilter(f.runtimes, runtime) {
				continue
			}
			for _, framework := range language.Frameworks {
				if !matchesFilter(f.frameworks, framework) {
					continue
				}
				for _, architecture := range language.Architectures {
					if !matchesFilter(f.archs, architecture) {
						continue
					}
					parts := slices.DeleteFunc([]string{language.Name, runtime, framework, architecture}, func(part string) bool { return part == "" })
					all = append(all, project.Config{
						ProjectName:  strings.Join(parts, "-"),
						Language:     language.Name,
						Runtime:      runtime,
						Framework:    framework,
						Architecture: architecture,
					})
				}
			}
		}
	}
	return all
}

func run() error {
	f, err := parseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	selected := configs(f)
	if len(selected) == 0 {
		return fmt.Errorf("nenhuma combinação encontrada para os filtros informados")
	}

	if err := os.MkdirAll(f.output, 0o755); err != nil {
		return fmt.Errorf("criando diretório de saída: %w", err)
	}

	var generated, skipped, failed int
	for _, config := range selected {
		destination := filepath.Join(f.output, config.ProjectName)
		if _, err := os.Stat(destination); err == nil {
			fmt.Printf("pulado   %s (já existe)\n", config.ProjectName)
			skipped++
			continue
		}
		if err := generator.Generate(destination, config, crudding.Templates); err != nil {
			fmt.Fprintf(os.Stderr, "falhou   %s: %v\n", config.ProjectName, err)
			failed++
			continue
		}
		fmt.Printf("gerado   %s\n", config.ProjectName)
		generated++
	}

	fmt.Printf("\n%d gerados, %d pulados, %d falharam em %s\n", generated, skipped, failed, f.output)
	if failed > 0 {
		return fmt.Errorf("%d projetos falharam", failed)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "genall: %v\n", err)
		os.Exit(1)
	}
}
