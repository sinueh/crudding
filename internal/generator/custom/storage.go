package custom

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
	"github.com/sinueh/crudding/internal/project"
)

// TemplatesDir devolve o caminho da pasta onde templates customizados são armazenados.
// Usa CRUDDING_TEMPLATES_DIR (se definido), CRUDDING_DIR/templates ou ~/.crudding/templates.
func TemplatesDir() (string, error) {
	if env := os.Getenv("CRUDDING_TEMPLATES_DIR"); env != "" {
		return filepath.Abs(env)
	}
	if env := os.Getenv("CRUDDING_DIR"); env != "" {
		return filepath.Join(env, "templates"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("obter diretório do usuário: %w", err)
	}
	return filepath.Join(home, ".crudding", "templates"), nil
}

// EnsureTemplatesDir garante que a pasta .crudding/templates/ exista no disco.
func EnsureTemplatesDir() (string, error) {
	dir, err := TemplatesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("criar pasta de templates %s: %w", dir, err)
	}
	return dir, nil
}

// TemplateItem descreve um template salvo em .crudding/templates/.
type TemplateItem struct {
	Name     string
	Path     string
	Manifest manifest.Manifest
}

// Save copia o projeto em sourceDir para a pasta de templates (.crudding/templates/<templateName>),
// renomeando o projeto para templateName e garantindo um project.crud válido.
func Save(sourceDir, templateName string) (string, manifest.Manifest, error) {
	if err := project.ValidateProjectName(templateName); err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("nome de template inválido: %w", err)
	}

	sourceAbs, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("resolver diretório %s: %w", sourceDir, err)
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("diretório %s não encontrado", sourceDir)
	}
	if !info.IsDir() {
		return "", manifest.Manifest{}, fmt.Errorf("%s não é um diretório", sourceDir)
	}

	templatesDir, err := EnsureTemplatesDir()
	if err != nil {
		return "", manifest.Manifest{}, err
	}

	targetDir := filepath.Join(templatesDir, templateName)
	if _, err := os.Stat(targetDir); err == nil {
		return "", manifest.Manifest{}, fmt.Errorf("template %q já existe em %s", templateName, targetDir)
	}

	if rel, err := filepath.Rel(sourceAbs, targetDir); err == nil && filepath.IsLocal(rel) {
		return "", manifest.Manifest{}, fmt.Errorf("o template não pode ser salvo dentro do diretório de origem (%s)", sourceAbs)
	}

	manifestPath := filepath.Join(sourceAbs, manifest.FileName)
	m, err := manifest.Read(manifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		m = manifest.Manifest{
			Version:      manifest.Version,
			Name:         filepath.Base(sourceAbs),
			Architecture: manifest.CustomArchitecture,
		}
		m.Directories, m.Files, err = common.ListStructure(os.DirFS(sourceAbs), nil)
		if err != nil {
			return "", manifest.Manifest{}, fmt.Errorf("listar estrutura de %s: %w", sourceDir, err)
		}
	} else if err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("ler %s: %w", manifestPath, err)
	}

	templatesFS := os.DirFS(sourceAbs)
	plan, err := Plan(templatesFS, m, templateName)
	if err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("preparar template: %w", err)
	}

	config := m.Config()
	config.ProjectName = templateName
	if plan.Manifest != nil {
		plan.Manifest.Template = ""
	}

	if err := common.Scaffold(targetDir, config, templatesFS, plan); err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("gravar template em %s: %w", targetDir, err)
	}

	savedManifest, err := manifest.Read(filepath.Join(targetDir, manifest.FileName))
	if err != nil {
		return "", manifest.Manifest{}, err
	}

	return targetDir, savedManifest, nil
}

// List lista todos os templates existentes em .crudding/templates/.
func List() ([]TemplateItem, error) {
	dir, err := EnsureTemplatesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listar templates em %s: %w", dir, err)
	}
	var list []TemplateItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		templatePath := filepath.Join(dir, name)
		m, err := manifest.Read(filepath.Join(templatePath, manifest.FileName))
		if err != nil {
			m = manifest.Manifest{Name: name, Architecture: "custom"}
		}
		list = append(list, TemplateItem{
			Name:     name,
			Path:     templatePath,
			Manifest: m,
		})
	}
	return list, nil
}

// Get obtém o caminho e o manifesto de um template salvo pelo nome.
func Get(name string) (string, manifest.Manifest, error) {
	dir, err := EnsureTemplatesDir()
	if err != nil {
		return "", manifest.Manifest{}, err
	}
	target := filepath.Join(dir, name)
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return "", manifest.Manifest{}, fmt.Errorf("template %q não encontrado em %s", name, dir)
	}
	m, err := manifest.Read(filepath.Join(target, manifest.FileName))
	if err != nil {
		return "", manifest.Manifest{}, fmt.Errorf("template %q: %w", name, err)
	}
	return target, m, nil
}

// Remove remove um template salvo pelo nome.
func Remove(name string) error {
	target, _, err := Get(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(target)
}

// Export empacota o template salvo name num arquivo .zip em dest, com o
// conteúdo dentro de uma pasta <name>/, pronta para "crudding add template <pasta>".
// Sem dest, grava ./<name>.zip; se dest for uma pasta existente, grava
// <dest>/<name>.zip. Nunca sobrescreve um arquivo existente.
func Export(name, dest string) (string, error) {
	if err := project.ValidateProjectName(name); err != nil {
		return "", fmt.Errorf("nome de template inválido: %w", err)
	}
	source, _, err := Get(name)
	if err != nil {
		return "", err
	}

	if dest == "" {
		dest = "."
	}
	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		dest = filepath.Join(dest, name+".zip")
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("resolver destino: %w", err)
	}
	if rel, err := filepath.Rel(source, dest); err == nil && filepath.IsLocal(rel) {
		return "", fmt.Errorf("o arquivo exportado não pode ficar dentro do template (%s)", source)
	}

	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("%s já existe", dest)
	}
	if err != nil {
		return "", fmt.Errorf("criar %s: %w", dest, err)
	}
	if err := writeZip(file, os.DirFS(source), name); err != nil {
		file.Close()
		os.Remove(dest)
		return "", fmt.Errorf("exportar template %q: %w", name, err)
	}
	if err := file.Close(); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("gravar %s: %w", dest, err)
	}
	return dest, nil
}

// writeZip grava todo o conteúdo de templates em w sob a pasta prefix/.
// Symlinks e arquivos especiais são recusados, como no Plan.
func writeZip(w io.Writer, templates fs.FS, prefix string) error {
	archive := zip.NewWriter(w)
	err := fs.WalkDir(templates, ".", func(rel string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("%s não é um arquivo comum", rel)
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = prefix
		if rel != "." {
			header.Name = prefix + "/" + rel
		}
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := archive.CreateHeader(header)
		if err != nil || info.IsDir() {
			return err
		}
		src, err := templates.Open(rel)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(writer, src)
		return err
	})
	if err != nil {
		return err
	}
	return archive.Close()
}
