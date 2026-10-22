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

	"github.com/sinueh-andreata/crudding/internal/manifest"
)

// ListStructure lista as pastas e os arquivos de fsys que não casam com
// ignore, no formato dos campos directories e files do project.crud. Uma pasta
// ignorada pula a subárvore inteira; o project.crud e as pastas .git nunca
// entram.
func ListStructure(fsys fs.FS, ignore []string) (directories, files []string, err error) {
	matcher, err := newIgnoreMatcher(ignore)
	if err != nil {
		return nil, nil, err
	}
	directories, files = []string{}, []string{}
	err = fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || name == "." {
			return err
		}
		if excluded(name) || matcher.match(name) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		switch {
		case entry.IsDir():
			directories = append(directories, name)
		case entry.Type().IsRegular():
			files = append(files, name)
		default:
			return fmt.Errorf("%s: apenas arquivos e pastas comuns são aceitos; adicione-o ao ignore", name)
		}
		return nil
	})
	return directories, files, err
}

// RecordCreated acrescenta em m.Files os arquivos criados (relativos, com
// "/"), e em m.Directories as pastas deles, como o ListStructure listaria:
// fora o que o ignore exclui e na mesma ordem.
func RecordCreated(m *manifest.Manifest, created []string) error {
	matcher, err := newIgnoreMatcher(m.Ignore)
	if err != nil {
		return err
	}
	files, directories := slices.Clone(m.Files), slices.Clone(m.Directories)
	for _, file := range created {
		if matcher.matchAny(file) {
			continue
		}
		if !slices.Contains(files, file) {
			files = append(files, file)
		}
		for directory := path.Dir(file); directory != "."; directory = path.Dir(directory) {
			if !slices.Contains(directories, directory) {
				directories = append(directories, directory)
			}
		}
	}
	slices.SortFunc(files, comparePaths)
	slices.SortFunc(directories, comparePaths)
	m.Files, m.Directories = files, directories
	return nil
}

// RecordRemoved tira de m.Files os arquivos removidos e de m.Directories as
// pastas que não existem mais em root.
func RecordRemoved(root string, m *manifest.Manifest, removed []string) {
	m.Files = slices.DeleteFunc(slices.Clone(m.Files), func(file string) bool { return slices.Contains(removed, file) })
	m.Directories = slices.DeleteFunc(slices.Clone(m.Directories), func(directory string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(directory)))
		return errors.Is(err, fs.ErrNotExist)
	})
}

// comparePaths ordena caminhos segmento a segmento, a ordem em que o
// fs.WalkDir os visita.
func comparePaths(a, b string) int {
	return slices.Compare(strings.Split(a, "/"), strings.Split(b, "/"))
}

// excluded diz se rel nunca entra na estrutura, com ou sem ignore.
func excluded(rel string) bool {
	return rel == manifest.FileName || path.Base(rel) == ".git"
}

// ignoreMatcher implementa o subconjunto do .gitignore aceito no campo ignore:
// "/" final é removida, "**/" inicial vale qualquer nível, padrão com "/" casa
// com o caminho relativo inteiro e padrão sem "/" casa com o nome de qualquer
// item.
type ignoreMatcher struct {
	paths []string
	names []string
}

func newIgnoreMatcher(patterns []string) (ignoreMatcher, error) {
	var m ignoreMatcher
	for _, pattern := range patterns {
		clean := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(pattern), "/"), "**/")
		if clean == "" {
			continue
		}
		if _, err := path.Match(clean, ""); err != nil {
			return ignoreMatcher{}, fmt.Errorf("padrão de ignore inválido %q: %w", pattern, err)
		}
		if strings.Contains(clean, "/") {
			m.paths = append(m.paths, strings.TrimPrefix(clean, "/"))
		} else {
			m.names = append(m.names, clean)
		}
	}
	return m, nil
}

// matchAny diz se rel ou alguma pasta acima dele fica de fora da estrutura.
func (m ignoreMatcher) matchAny(rel string) bool {
	for current := rel; current != "."; current = path.Dir(current) {
		if excluded(current) || m.match(current) {
			return true
		}
	}
	return false
}

func (m ignoreMatcher) match(rel string) bool {
	for _, pattern := range m.paths {
		if ok, _ := path.Match(pattern, rel); ok {
			return true
		}
	}
	base := path.Base(rel)
	for _, pattern := range m.names {
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
	}
	return false
}
