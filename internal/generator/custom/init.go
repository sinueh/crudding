// Package custom transforma um projeto qualquer em template: "crudding init"
// grava a estrutura do projeto no project.crud e "crudding template" a copia
// para um projeto novo, trocando o nome do template pelo do projeto.
package custom

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"crudding/internal/generator/common"
	"crudding/internal/manifest"
)

// Init grava em dir o project.crud com a estrutura do projeto (directories e
// files, menos o ignore). Um project.crud novo começa com o ignore vazio; se
// ele já existe, só a estrutura é regenerada e o resto (name, stack,
// endpoints, ignore) é mantido.
func Init(dir string) (manifest.Manifest, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("resolver diretório: %w", err)
	}

	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if errors.Is(err, fs.ErrNotExist) {
		m, err = manifest.Manifest{
			Version:      manifest.Version,
			Name:         filepath.Base(root),
			Architecture: manifest.CustomArchitecture,
		}, nil
	}
	if err != nil {
		return manifest.Manifest{}, err
	}
	if m.Directories, m.Files, err = common.ListStructure(os.DirFS(root), m.Ignore); err != nil {
		return manifest.Manifest{}, err
	}
	if err := manifest.Write(root, m); err != nil {
		return manifest.Manifest{}, err
	}
	return m, nil
}
