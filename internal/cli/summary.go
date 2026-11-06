package cli

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
)

// summary é o resumo impresso ao fim dos comandos que geram ou alteram
// arquivos. Com --json vira um único objeto JSON no stdout, para agentes não
// precisarem ler os arquivos gerados.
type summary struct {
	Command  string       `json:"command"`
	Project  string       `json:"project"`
	Created  []string     `json:"created"`
	Modified []string     `json:"modified"`
	Removed  []string     `json:"removed"`
	Edit     []editTarget `json:"edit"`
	Routes   []string     `json:"routes"`
	Next     []string     `json:"next"`
	// routeNote acompanha a linha "rota:" no texto (ex.: "exemplo").
	routeNote string
}

// editTarget é um arquivo que o usuário deve preencher, com o motivo.
type editTarget struct {
	Path   string
	Reason string
}

const (
	summaryLabelWidth = 10
	summaryLineWidth  = 100
	// internalDir guarda as cópias do endpoint de exemplo usadas pelo add
	// endpoint; ninguém precisa editá-las.
	internalDir = ".crudding/"
)

// MarshalJSON grava o editTarget só com o caminho.
func (e editTarget) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Path)
}

// normalized troca listas nil por vazias, para o JSON sempre ter as mesmas chaves.
func (s summary) normalized() summary {
	for _, list := range []*[]string{&s.Created, &s.Modified, &s.Removed, &s.Routes, &s.Next} {
		if *list == nil {
			*list = []string{}
		}
	}
	if s.Edit == nil {
		s.Edit = []editTarget{}
	}
	return s
}

// write imprime o resumo em texto compacto ou, com asJSON, como um objeto JSON.
func (s summary) write(stdout io.Writer, asJSON bool) error {
	if asJSON {
		return writeJSON(stdout, s.normalized())
	}
	fmt.Fprint(stdout, s.text())
	return nil
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func (s summary) text() string {
	var b strings.Builder
	writeEntries(&b, "criado:", humanEntries(s.Created))
	writeEntries(&b, "removido:", humanEntries(s.Removed))
	writeEntries(&b, "alterado:", humanEntries(s.Modified))
	// Uma linha por motivo, com os arquivos agrupados.
	var reasons []string
	byReason := map[string][]string{}
	for _, target := range s.Edit {
		if _, ok := byReason[target.Reason]; !ok {
			reasons = append(reasons, target.Reason)
		}
		byReason[target.Reason] = append(byReason[target.Reason], target.Path)
	}
	for index, reason := range reasons {
		label := ""
		if index == 0 {
			label = "editar:"
		}
		fmt.Fprintf(&b, "%-*s%s (%s)\n", summaryLabelWidth, label, strings.Join(compressPaths(byReason[reason]), ", "), reason)
	}
	if len(s.Routes) > 0 {
		routes := strings.Join(s.Routes, ", ")
		if s.routeNote != "" {
			routes += " (" + s.routeNote + ")"
		}
		fmt.Fprintf(&b, "%-*s%s\n", summaryLabelWidth, "rota:", routes)
	}
	writeNext(&b, s.Next)
	return b.String()
}

// writeEntries escreve label e as entradas separadas por vírgula, quebrando a
// linha antes de summaryLineWidth com as continuações alinhadas.
func writeEntries(b *strings.Builder, label string, entries []string) {
	if len(entries) == 0 {
		return
	}
	line := fmt.Sprintf("%-*s", summaryLabelWidth, label)
	start := len([]rune(line))
	for index, entry := range entries {
		switch {
		case index == 0:
			line += entry
		case len([]rune(line))+2+len([]rune(entry)) > summaryLineWidth:
			b.WriteString(line + ",\n")
			line = strings.Repeat(" ", start) + entry
		default:
			line += ", " + entry
		}
	}
	b.WriteString(line + "\n")
}

// writeNext junta com && os comandos de shell e põe cada comando do crudding
// numa linha própria (eles não devem rodar depois de um servidor que bloqueia).
func writeNext(b *strings.Builder, next []string) {
	var shell, crudding []string
	for _, command := range next {
		if strings.HasPrefix(command, "crudding ") {
			crudding = append(crudding, command)
		} else {
			shell = append(shell, command)
		}
	}
	lines := crudding
	if len(shell) > 0 {
		lines = append([]string{strings.Join(shell, " && ")}, crudding...)
	}
	for index, line := range lines {
		label := ""
		if index == 0 {
			label = "próximo:"
		}
		fmt.Fprintf(b, "%-*s%s\n", summaryLabelWidth, label, line)
	}
}

// humanEntries agrupa os caminhos para o texto e troca tudo o que está em
// .crudding/ por uma única entrada.
func humanEntries(paths []string) []string {
	var visible []string
	internal := false
	for _, file := range paths {
		if strings.HasPrefix(file, internalDir) {
			internal = true
			continue
		}
		visible = append(visible, file)
	}
	entries := compressPaths(visible)
	if internal {
		entries = append(entries, internalDir+" (interno)")
	}
	return entries
}

// compressPaths agrupa caminhos com chaves para o resumo ficar curto:
// arquivos da mesma pasta viram "pasta/{a,b}.ext" e arquivos de mesmo nome em
// pastas irmãs viram "pai/{x,y}/nome".
func compressPaths(paths []string) []string {
	sorted := slices.Clone(paths)
	slices.SortStableFunc(sorted, func(a, b string) int {
		return cmp.Or(cmp.Compare(path.Dir(a), path.Dir(b)), cmp.Compare(path.Base(a), path.Base(b)))
	})
	sorted = slices.Compact(sorted)

	type group struct {
		dir   string
		names []string
	}
	var groups []group
	for _, file := range sorted {
		dir := path.Dir(file)
		if len(groups) > 0 && groups[len(groups)-1].dir == dir {
			groups[len(groups)-1].names = append(groups[len(groups)-1].names, path.Base(file))
			continue
		}
		groups = append(groups, group{dir: dir, names: []string{path.Base(file)}})
	}

	// Pastas com um único arquivo de mesmo nome sob o mesmo pai se juntam.
	type sibling struct{ parent, base string }
	merged := map[sibling][]string{}
	for _, g := range groups {
		if len(g.names) == 1 && g.dir != "." {
			key := sibling{path.Dir(g.dir), g.names[0]}
			merged[key] = append(merged[key], path.Base(g.dir))
		}
	}

	var entries []string
	for _, g := range groups {
		if len(g.names) == 1 && g.dir != "." {
			key := sibling{path.Dir(g.dir), g.names[0]}
			dirs, ok := merged[key]
			if !ok {
				continue // já incluído na entrada do primeiro irmão
			}
			delete(merged, key)
			if len(dirs) > 1 {
				entries = append(entries, joinPath(key.parent, "{"+strings.Join(dirs, ",")+"}/"+key.base))
				continue
			}
		}
		entries = append(entries, joinPath(g.dir, braceNames(g.names)))
	}
	return entries
}

// braceNames junta nomes da mesma pasta com o prefixo e o sufixo comuns fora
// das chaves, cortados em "_", "-" ou ".": "{a,b}.ext", "user_{model,service}.go".
func braceNames(names []string) string {
	if len(names) == 1 {
		return names[0]
	}
	prefix, suffix := names[0], names[0]
	for _, name := range names[1:] {
		for !strings.HasPrefix(name, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		for !strings.HasSuffix(name, suffix) {
			suffix = suffix[1:]
		}
	}
	prefix = prefix[:strings.LastIndexAny(prefix, "_-.")+1]
	if cut := strings.IndexAny(suffix, "_-."); cut >= 0 {
		suffix = suffix[cut:]
	} else {
		suffix = ""
	}
	middles := make([]string, len(names))
	for index, name := range names {
		if len(name) <= len(prefix)+len(suffix) {
			return "{" + strings.Join(names, ",") + "}"
		}
		middles[index] = name[len(prefix) : len(name)-len(suffix)]
	}
	return prefix + "{" + strings.Join(middles, ",") + "}" + suffix
}

func joinPath(dir, name string) string {
	if dir == "." {
		return name
	}
	return dir + "/" + name
}

// editTargets aponta, entre os arquivos criados de um endpoint, os que o
// usuário deve preencher: a lógica de negócio (serviços, casos de uso) e os
// campos (modelos, entidades). Portas e DTOs ficam de fora.
func editTargets(created []string) []editTarget {
	var logic, fields []editTarget
	for _, file := range created {
		if strings.HasPrefix(file, internalDir) || path.Base(file) == "__init__.py" {
			continue
		}
		segments := strings.Split(strings.ToLower(file), "/")
		base := segments[len(segments)-1]
		dirs := segments[:len(segments)-1]
		if slices.ContainsFunc(dirs, func(dir string) bool { return dir == "ports" || dir == "port" }) {
			continue
		}
		has := func(words ...string) bool {
			return slices.ContainsFunc(segments, func(segment string) bool {
				return slices.ContainsFunc(words, func(word string) bool { return strings.Contains(segment, word) })
			})
		}
		switch {
		case has("service", "use_case", "usecase") || slices.Contains(dirs, "application"):
			logic = append(logic, editTarget{Path: file, Reason: "regras de negócio"})
		case has("model", "domain", "entit") && !strings.Contains(base, "types"):
			fields = append(fields, editTarget{Path: file, Reason: "campos"})
		}
	}
	return append(logic, fields...)
}

var (
	quotedPattern = regexp.MustCompile(`"([^"\n]*)"(\s*:)?`)
	groupPattern  = regexp.MustCompile(`\b(?:Group|Route)\(\s*"(/[^"]*)"`)
)

// endpointRoutes procura nos arquivos de register as rotas HTTP do endpoint
// name: literais entre aspas cujo último segmento (sem parâmetros como :id ou
// {id}) é o nome em alguma forma. Um prefixo único de Group/Route no arquivo
// (ex.: "/api/v1") é acrescentado. Devolve nil se não achar nada.
func endpointRoutes(root string, register []string, name string) []string {
	forms := nameForms(name)
	var routes []string
	for _, rel := range register {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		var prefixes []string
		for _, match := range groupPattern.FindAllStringSubmatch(string(content), -1) {
			if prefix := strings.TrimSuffix(match[1], "/"); prefix != "" && !slices.Contains(prefixes, prefix) {
				prefixes = append(prefixes, prefix)
			}
		}
		for _, line := range strings.Split(string(content), "\n") {
			for _, match := range quotedPattern.FindAllStringSubmatch(line, -1) {
				route, ok := routeLiteral(match[1], match[2] != "", forms)
				if !ok {
					continue
				}
				if len(prefixes) == 1 && !strings.HasPrefix(route, prefixes[0]+"/") {
					route = prefixes[0] + route
				}
				if !slices.Contains(routes, route) {
					routes = append(routes, route)
				}
			}
		}
	}
	return routes
}

// routeLiteral diz se literal é uma rota do endpoint: "/x", "GET /api/x/{id}",
// "x/" (Django) ou a chave "x": de um dicionário de rotas.
func routeLiteral(literal string, isKey bool, forms []string) (string, bool) {
	if fields := strings.Fields(literal); len(fields) > 1 {
		literal = fields[len(fields)-1]
	}
	switch {
	case strings.HasPrefix(literal, "/"):
	case strings.HasSuffix(literal, "/") && !strings.Contains(literal, "."):
	case isKey && !strings.Contains(literal, "/"):
	default:
		return "", false
	}
	var segments []string
	for _, segment := range strings.Split(literal, "/") {
		if segment == "" {
			continue
		}
		if strings.ContainsAny(segment[:1], ":{<") {
			break
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 || !slices.Contains(forms, segments[len(segments)-1]) {
		return "", false
	}
	return "/" + strings.Join(segments, "/"), true
}

// nameForms devolve as formas de um nome snake_case usadas nas rotas:
// snake_case, kebab-case, camelCase e PascalCase.
func nameForms(name string) []string {
	var pascal strings.Builder
	for _, part := range strings.Split(name, "_") {
		if part != "" {
			pascal.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	forms := []string{name, strings.ReplaceAll(name, "_", "-")}
	if p := pascal.String(); p != "" {
		forms = append(forms, strings.ToLower(p[:1])+p[1:], p)
	}
	return forms
}

// projectRoutes devolve as rotas de todos os endpoints do projeto.
func projectRoutes(root string, m manifest.Manifest) []string {
	var routes []string
	for _, name := range m.Endpoints {
		for _, route := range endpointRoutes(root, m.TemplateEndpoint.Register, name) {
			if !slices.Contains(routes, route) {
				routes = append(routes, route)
			}
		}
	}
	return routes
}

// runCommand devolve o comando que sobe a aplicação, a partir da linguagem,
// do framework e dos arquivos do projeto. Vazio se não souber.
func runCommand(root string, m manifest.Manifest) string {
	switch m.Language {
	case "go":
		if fileExists(root, "cmd/api/main.go") {
			return "go run ./cmd/api"
		}
		return "go run ."
	case "python":
		module := ""
		for _, candidate := range []string{"src/app/main.py", "src/main.py", "main.py"} {
			if fileExists(root, candidate) {
				module = strings.ReplaceAll(strings.TrimSuffix(candidate, ".py"), "/", ".")
				break
			}
		}
		switch {
		case module == "":
			return ""
		case m.Framework == "fastapi":
			return "uvicorn " + module + ":app --reload"
		case m.Framework == "django":
			return "python -m " + module + " runserver"
		default:
			return "python -m " + module
		}
	case "node":
		return "npm start"
	}
	return ""
}

// installCommand devolve o comando que instala as dependências do projeto.
func installCommand(m manifest.Manifest) string {
	switch m.Language {
	case "go":
		return "go mod tidy"
	case "python":
		return "pip install -r requirements.txt"
	case "node":
		return "npm install"
	}
	return ""
}

// newProjectSummary monta o resumo de um projeto recém-gerado em root: como a
// pasta acabou de ser criada, tudo o que está nela foi criado pelo comando.
func newProjectSummary(command, root string) (summary, error) {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return summary{}, err
	}
	_, files, err := common.ListStructure(os.DirFS(root), nil)
	if err != nil {
		return summary{}, err
	}
	s := summary{
		Command: command,
		Project: root,
		Created: append(files, manifest.FileName),
	}
	if m.TemplateEndpoint.Defined() {
		s.Routes = projectRoutes(root, m)
		s.routeNote = "exemplo"
	}
	name := filepath.Base(root)
	if strings.ContainsAny(name, " '\"") {
		name = fmt.Sprintf("%q", name)
	}
	s.Next = []string{"cd " + name}
	if m.Database != "" || len(m.Databases) > 0 {
		s.Next = append(s.Next, "docker compose up -d")
	}
	s.Next = appendNonEmpty(s.Next, installCommand(m), runCommand(root, m))
	if m.TemplateEndpoint.Defined() {
		s.Next = append(s.Next, "crudding add endpoint <nome>")
	}
	return s, nil
}

// endpointSummary monta o resumo de add/remove endpoint a partir do resultado
// do gerador; o project.crud é sempre regravado.
func endpointSummary(command, root string, m manifest.Manifest, created, removed, modified []string, routes []string) summary {
	s := summary{
		Command:  command,
		Project:  root,
		Created:  created,
		Removed:  removed,
		Modified: appendManifest(modified),
		Edit:     editTargets(created),
		Routes:   routes,
	}
	if len(removed) > 0 {
		s.routeNote = "removida"
	}
	s.Next = appendNonEmpty(nil, runCommand(root, m))
	return s
}

func appendManifest(modified []string) []string {
	if slices.Contains(modified, manifest.FileName) {
		return slices.Clone(modified)
	}
	return append(slices.Clone(modified), manifest.FileName)
}

func appendNonEmpty(list []string, values ...string) []string {
	for _, value := range values {
		if value != "" {
			list = append(list, value)
		}
	}
	return list
}

func fileExists(root, rel string) bool {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && !info.IsDir()
}

// splitJSONFlag tira de args a flag de saída JSON, aceita como --json, -json,
// --json=true|false ou json:true|false (o estilo chave:valor do "new").
func splitJSONFlag(args []string) ([]string, bool, error) {
	rest := make([]string, 0, len(args))
	asJSON := false
	for _, token := range args {
		var value string
		switch {
		case token == "--json" || token == "-json":
			asJSON = true
			continue
		case strings.HasPrefix(token, "--json="), strings.HasPrefix(token, "-json="):
			_, value, _ = strings.Cut(token, "=")
		case strings.HasPrefix(token, "json:"):
			value = strings.TrimPrefix(token, "json:")
		default:
			rest = append(rest, token)
			continue
		}
		switch strings.TrimSpace(value) {
		case "true", "":
			asJSON = true
		case "false":
			asJSON = false
		default:
			return nil, false, fmt.Errorf("valor inválido para json: %q (use true ou false)", value)
		}
	}
	return rest, asJSON, nil
}

// jsonFlagUsage documenta a flag --json nas ajudas dos comandos.
const jsonFlagUsage = "  --json (ou json:true)   imprime o resumo como um único objeto JSON: command, project,\n" +
	"                          created, modified, removed, edit, routes e next (exige os\n" +
	"                          argumentos; não há perguntas interativas)\n"

// errJSONNeedsArg é devolvido quando o modo JSON precisaria de uma pergunta
// interativa, que escreveria no stdout.
func errJSONNeedsArg(what string) error {
	return fmt.Errorf("--json exige %s nos argumentos (não há perguntas interativas no modo JSON)", what)
}
