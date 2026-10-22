// Package manifest lê e grava o project.crud, que registra na raiz de cada
// projeto gerado a combinação usada (linguagem, framework, arquitetura...) e
// os endpoints registrados.
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh-andreata/crudding/internal/project"
)

const (
	FileName = "project.crud"
	Version  = 1
	// CustomArchitecture marca projetos que não seguem uma arquitetura do
	// crudding (ex.: templates criados com "crudding init").
	CustomArchitecture = "custom"
)

type Manifest struct {
	Version      int      `json:"version"`
	Name         string   `json:"name"`
	Language     string   `json:"language"`
	Runtime      string   `json:"runtime,omitempty"`
	Framework    string   `json:"framework"`
	Architecture string   `json:"architecture"`
	Endpoints    []string `json:"endpoints"`
	// Endpoint, quando preenchido, define como "crudding add/remove endpoint"
	// cria e remove recursos neste projeto, no lugar do layout da arquitetura.
	Endpoint  EndpointSpec `json:"endpoint"`
	Database  string       `json:"database,omitempty"`
	Databases []string     `json:"databases,omitempty"`
	ORM       string       `json:"orm,omitempty"`
	// Template é o name do projeto-template que gerou este projeto.
	Template string `json:"template,omitempty"`
	// Ignore, Directories e Files descrevem a estrutura do projeto, usada por
	// "crudding template": a geração e o "crudding init" listam as pastas e os
	// arquivos, menos o ignore.
	Ignore      []string `json:"ignore"`
	Directories []string `json:"directories"`
	Files       []string `json:"files"`
	// FileMappings indica a origem de cada arquivo do template se diferente do destino.
	FileMappings []FileMapping `json:"-"`
}

// FileMapping descreve a origem e o destino de um arquivo no template.
type FileMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (m *Manifest) UnmarshalJSON(data []byte) error {
	type alias Manifest
	var aux struct {
		alias
		RawFiles []json.RawMessage `json:"files"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*m = Manifest(aux.alias)
	m.Files = make([]string, len(aux.RawFiles))
	m.FileMappings = make([]FileMapping, len(aux.RawFiles))
	for i, raw := range aux.RawFiles {
		var str string
		if err := json.Unmarshal(raw, &str); err == nil {
			m.Files[i] = str
			m.FileMappings[i] = FileMapping{From: str, To: str}
			continue
		}
		var mapping FileMapping
		if err := json.Unmarshal(raw, &mapping); err == nil && (mapping.From != "" || mapping.To != "") {
			if mapping.To == "" {
				mapping.To = mapping.From
			}
			m.Files[i] = mapping.To
			m.FileMappings[i] = mapping
			continue
		}
		return fmt.Errorf("files[%d]: formato inválido", i)
	}
	m.normalize()
	return nil
}

// EndpointSpec descreve um endpoint de exemplo que já existe no projeto. O
// "add endpoint <nome>" copia Files trocando Name pelo nome novo e, em cada
// arquivo de Register, duplica as linhas que mencionam Name; o "remove
// endpoint" desfaz as duas coisas.
type EndpointSpec struct {
	Name     string   `json:"name"`     // endpoint de exemplo, em snake_case
	Files    []string `json:"files"`    // arquivos do exemplo, copiados com o nome trocado
	Register []string `json:"register"` // arquivos compartilhados (rotas, imports)
}

// Defined diz se o campo endpoint foi preenchido.
func (e EndpointSpec) Defined() bool {
	return e.Name != "" || len(e.Files) > 0 || len(e.Register) > 0
}

func New(config project.Config, endpoints ...string) Manifest {
	eps := slices.Clone(endpoints)
	if eps == nil {
		eps = []string{}
	}
	var dbs []string
	if config.Database != "" {
		dbs = []string{config.Database}
	}
	return Manifest{
		Version:      Version,
		Name:         config.ProjectName,
		Language:     config.Language,
		Runtime:      config.Runtime,
		Framework:    config.Framework,
		Architecture: config.Architecture,
		Endpoints:    eps,
		Database:     config.Database,
		Databases:    dbs,
		ORM:          config.ORM,
	}
}

func (m *Manifest) AddEndpoint(name string) {
	if !slices.Contains(m.Endpoints, name) {
		m.Endpoints = append(m.Endpoints, name)
	}
}

func (m *Manifest) RemoveEndpoint(name string) bool {
	idx := slices.Index(m.Endpoints, name)
	if idx < 0 {
		return false
	}
	m.Endpoints = slices.Delete(m.Endpoints, idx, idx+1)
	return true
}

func (m Manifest) HasEndpoint(name string) bool {
	return slices.Contains(m.Endpoints, name)
}

func (m Manifest) Config() project.Config {
	return project.Config{
		ProjectName:  m.Name,
		Language:     m.Language,
		Runtime:      m.Runtime,
		Framework:    m.Framework,
		Architecture: m.Architecture,
		Database:     m.Database,
		ORM:          m.ORM,
	}
}

func (m Manifest) HasDatabase(name string) bool {
	if m.Database == name {
		return true
	}
	return slices.Contains(m.Databases, name)
}

func (m *Manifest) AddDatabase(name string) {
	if m.Database == "" {
		m.Database = name
	}
	if !slices.Contains(m.Databases, name) {
		m.Databases = append(m.Databases, name)
	}
}

// Stack descreve a combinação de forma legível, ex.: "python + fastapi + modular".
// Partes vazias (comuns em projetos de arquitetura custom) são omitidas.
func (m Manifest) Stack() string {
	language := m.Language
	if m.Runtime != "" {
		language += "/" + m.Runtime
	}
	databases := strings.Join(m.Databases, ", ")
	if databases == "" {
		databases = m.Database
	}
	parts := slices.DeleteFunc([]string{language, m.Framework, m.Architecture, databases, m.ORM}, func(part string) bool { return part == "" })
	return strings.Join(parts, " + ")
}

func Write(root string, m Manifest) error {
	m.normalize()
	content, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("codificar %s: %w", FileName, err)
	}
	if err := os.WriteFile(filepath.Join(root, FileName), append(content, '\n'), 0644); err != nil {
		return fmt.Errorf("gravar %s: %w", FileName, err)
	}
	return nil
}

func Read(path string) (Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("ler %s: %w", path, err)
	}
	return parse(content, path)
}

// ReadFS lê o project.crud na raiz de fsys (ex.: a pasta de um template).
func ReadFS(fsys fs.FS) (Manifest, error) {
	content, err := fs.ReadFile(fsys, FileName)
	if err != nil {
		return Manifest{}, fmt.Errorf("ler %s: %w", FileName, err)
	}
	return parse(content, FileName)
}

func parse(content []byte, label string) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(content, &m); err != nil {
		return Manifest{}, fmt.Errorf("%s inválido: %w", label, err)
	}
	if m.Version != Version {
		return Manifest{}, fmt.Errorf("%s: versão %d não suportada (esperado %d)", label, m.Version, Version)
	}
	m.normalize()
	return m, nil
}

// Parse decodifica os bytes do JSON em um Manifest.
func Parse(content []byte, label string) (Manifest, error) {
	return parse(content, label)
}

// normalize troca listas nil por vazias, para o project.crud sempre ter as
// mesmas chaves ("endpoints": [] em vez de null).
func (m *Manifest) normalize() {
	for _, list := range []*[]string{&m.Endpoints, &m.Endpoint.Files, &m.Endpoint.Register, &m.Ignore, &m.Directories, &m.Files} {
		if *list == nil {
			*list = []string{}
		}
	}
}

// Find procura o project.crud em dir e nos diretórios acima dele, devolvendo a
// raiz do projeto e o manifesto lido.
func Find(dir string) (string, Manifest, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", Manifest{}, fmt.Errorf("resolver diretório: %w", err)
	}
	for {
		path := filepath.Join(dir, FileName)
		_, statErr := os.Stat(path)
		if statErr == nil {
			m, err := Read(path)
			return dir, m, err
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			return "", Manifest{}, fmt.Errorf("verificar %s: %w", path, statErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", Manifest{}, fmt.Errorf("nenhum %s encontrado; rode o comando dentro de um projeto gerado pelo crudding", FileName)
		}
		dir = parent
	}
}
