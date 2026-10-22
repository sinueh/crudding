package common

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"
)

type File struct {
	TemplatePath string
	OutputPath   string
}

func CreateProjectDirectory(destination string) (string, error) {
	root, err := filepath.Abs(destination)
	if err != nil {
		return "", fmt.Errorf("resolver diretório do projeto: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(root), 0755); err != nil {
		return "", fmt.Errorf("criar diretório pai: %w", err)
	}
	if err := os.Mkdir(root, 0755); err != nil {
		return "", fmt.Errorf("criar projeto em %s: %w", root, err)
	}
	return root, nil
}

func WriteFiles(root string, templates fs.FS, data any, files []File) error {
	for _, file := range files {
		outputPath := filepath.Join(root, filepath.FromSlash(file.OutputPath))
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("criar diretório para %s: %w", file.OutputPath, err)
		}
		content, err := render(templates, file.TemplatePath, data)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			return fmt.Errorf("gravar arquivo %s: %w", file.OutputPath, err)
		}
	}
	return nil
}

func render(templates fs.FS, templatePath string, data any) ([]byte, error) {
	parsed, err := template.ParseFS(templates, templatePath)
	if err != nil {
		return nil, fmt.Errorf("ler template %s: %w", templatePath, err)
	}
	var output bytes.Buffer
	if err := parsed.ExecuteTemplate(&output, path.Base(templatePath), data); err != nil {
		return nil, fmt.Errorf("renderizar template %s: %w", templatePath, err)
	}
	return output.Bytes(), nil
}
