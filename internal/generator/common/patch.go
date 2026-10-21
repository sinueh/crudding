package common

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Insert acrescenta Lines em um arquivo existente, logo antes da linha que
// contém apenas Marker (ex.: "# crudding:routes"). As linhas herdam a
// indentação do marker, que permanece no arquivo para inserções futuras.
type Insert struct {
	Path   string // relativo à raiz do projeto, com "/"
	Marker string
	Lines  []string
}

// RawFile é um arquivo novo com o conteúdo já pronto (sem template).
type RawFile struct {
	OutputPath string // relativo à raiz do projeto, com "/"
	Content    []byte
}

// Rewrite reescreve um arquivo existente: Edit recebe o conteúdo atual (já com
// as Inserts aplicadas) e devolve o novo.
type Rewrite struct {
	Path string // relativo à raiz do projeto, com "/"
	Edit func(content []byte) ([]byte, error)
}

// Patch descreve alterações em um projeto já gerado: arquivos novos a partir
// de templates ou com conteúdo pronto, pacotes Python a criar, inserções em
// markers e reescritas de arquivos existentes.
type Patch struct {
	Files    []File
	Raw      []RawFile
	Data     any
	Packages []string // diretórios que devem existir com __init__.py
	Inserts  []Insert
	Rewrites []Rewrite
}

// Result lista os caminhos (relativos à raiz) criados e alterados por Apply.
type Result struct {
	Created  []string
	Modified []string
}

// Apply valida o Patch inteiro antes de tocar no disco (nenhum arquivo novo pode
// existir, todo marker aparece uma única vez, todos os templates renderizam e
// todas as reescritas funcionam). Se a gravação falhar no meio, desfaz o que
// já tinha sido feito.
func Apply(root string, templates fs.FS, patch Patch) (result Result, err error) {
	writes := make([]RawFile, 0, len(patch.Files)+len(patch.Raw))
	for _, file := range patch.Files {
		content, err := render(templates, file.TemplatePath, patch.Data)
		if err != nil {
			return Result{}, err
		}
		writes = append(writes, RawFile{OutputPath: file.OutputPath, Content: content})
	}
	writes = append(writes, patch.Raw...)
	for _, file := range writes {
		if err := ensureAbsent(root, file.OutputPath); err != nil {
			return Result{}, err
		}
	}

	originals := map[string][]byte{}
	edited := map[string][]byte{}
	var editOrder []string
	edit := func(rel string, change func([]byte) ([]byte, error)) error {
		current, ok := edited[rel]
		if !ok {
			var readErr error
			if current, readErr = os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); readErr != nil {
				return fmt.Errorf("ler %s: %w", rel, readErr)
			}
			originals[rel] = current
			editOrder = append(editOrder, rel)
		}
		changed, changeErr := change(current)
		if changeErr != nil {
			return changeErr
		}
		edited[rel] = changed
		return nil
	}
	for _, insert := range patch.Inserts {
		if err := edit(insert.Path, func(content []byte) ([]byte, error) { return insertBefore(content, insert) }); err != nil {
			return Result{}, err
		}
	}
	for _, rewrite := range patch.Rewrites {
		if err := edit(rewrite.Path, rewrite.Edit); err != nil {
			return Result{}, err
		}
	}

	var created []string // caminhos absolutos, na ordem de criação
	defer func() {
		if err == nil {
			return
		}
		for _, rel := range editOrder {
			_ = os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), originals[rel], 0644)
		}
		for _, target := range slices.Backward(created) {
			_ = os.Remove(target)
		}
	}()

	if err = CreatePackages(root, patch.Packages, &created); err != nil {
		return Result{}, err
	}
	for _, file := range writes {
		if err = mkdirAll(root, path.Dir(file.OutputPath), &created); err != nil {
			return Result{}, err
		}
		target := filepath.Join(root, filepath.FromSlash(file.OutputPath))
		if err = writeNew(target, file.Content); err != nil {
			return Result{}, fmt.Errorf("gravar arquivo %s: %w", file.OutputPath, err)
		}
		created = append(created, target)
	}
	for _, rel := range editOrder {
		if err = os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), edited[rel], 0644); err != nil {
			return Result{}, fmt.Errorf("gravar arquivo %s: %w", rel, err)
		}
	}

	for _, target := range created {
		if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, target)
			result.Created = append(result.Created, filepath.ToSlash(rel))
		}
	}
	result.Modified = editOrder
	return result, nil
}

func ensureAbsent(root, rel string) error {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	if err == nil {
		return fmt.Errorf("%s já existe", rel)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("verificar %s: %w", rel, err)
	}
	return nil
}

func insertBefore(content []byte, insert Insert) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	position := -1
	for index, line := range lines {
		if strings.TrimSpace(line) != insert.Marker {
			continue
		}
		if position >= 0 {
			return nil, fmt.Errorf("%s: marker %q aparece mais de uma vez", insert.Path, insert.Marker)
		}
		position = index
	}
	if position < 0 {
		return nil, fmt.Errorf("%s: marker %q não encontrado; adicione a linha %q no ponto onde o crudding deve registrar o código", insert.Path, insert.Marker, insert.Marker)
	}

	marker := lines[position]
	indent := marker[:len(marker)-len(strings.TrimLeft(marker, " \t"))]
	added := make([]string, len(insert.Lines))
	for index, line := range insert.Lines {
		if line != "" {
			line = indent + line
		}
		added[index] = line
	}
	lines = slices.Insert(lines, position, added...)
	return []byte(strings.Join(lines, "\n")), nil
}

// mkdirAll cria rel (relativo a root) e seus ancestrais, registrando em created
// apenas os diretórios que não existiam.
func mkdirAll(root, rel string, created *[]string) error {
	if rel == "." || rel == "" {
		return nil
	}
	current := root
	for _, part := range strings.Split(rel, "/") {
		current = filepath.Join(current, part)
		err := os.Mkdir(current, 0755)
		if err == nil {
			if created != nil {
				*created = append(*created, current)
			}
			continue
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("criar diretório %s: %w", current, err)
		}
	}
	return nil
}

// CreatePackages garante que cada diretório (e seus ancestrais) exista com um
// __init__.py. Arquivos já existentes são preservados; o que for criado é
// registrado em created (que pode ser nil).
func CreatePackages(root string, directories []string, created *[]string) error {
	for _, directory := range directories {
		parts := strings.Split(directory, "/")
		for index := range parts {
			rel := strings.Join(parts[:index+1], "/")
			if err := mkdirAll(root, rel, created); err != nil {
				return fmt.Errorf("criar pacote Python %s: %w", rel, err)
			}
			initFile := filepath.Join(root, filepath.FromSlash(rel), "__init__.py")
			err := writeNew(initFile, nil)
			if err == nil {
				if created != nil {
					*created = append(*created, initFile)
				}
				continue
			}
			if !errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("criar pacote Python %s: %w", rel, err)
			}
		}
	}
	return nil
}

// writeNew grava content em um arquivo que ainda não pode existir.
func writeNew(target string, content []byte) error {
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
