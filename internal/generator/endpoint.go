package generator

import (
	"fmt"
	"path/filepath"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/generator/custom"
	"github.com/sinueh/crudding/internal/manifest"
)

// CheckEndpointSupport confirma que o project.crud tem o campo template_endpoint
// preenchido e válido, para a CLI falhar antes de pedir o nome ao usuário.
func CheckEndpointSupport(m manifest.Manifest) error {
	if !m.TemplateEndpoint.Defined() {
		return fmt.Errorf("o campo \"template_endpoint\" do %s está vazio: preencha name, files e register com o endpoint de exemplo do projeto para usar add endpoint e remove endpoint", manifest.FileName)
	}
	return custom.CheckEndpoint(m)
}

// AddEndpoint cria o endpoint name no projeto em root copiando o endpoint de
// exemplo do campo template_endpoint do project.crud, e registra o novo endpoint em
// endpoints, files e directories.
func AddEndpoint(root string, m manifest.Manifest, name string) (common.Result, error) {
	m = current(root, m)
	if err := CheckEndpointSupport(m); err != nil {
		return common.Result{}, err
	}
	patch, err := custom.EndpointPatch(root, m, name)
	if err != nil {
		return common.Result{}, err
	}
	result, err := common.Apply(root, nil, patch)
	if err != nil {
		return common.Result{}, err
	}

	m.AddEndpoint(name)
	if err := common.RecordCreated(&m, result.Created); err != nil {
		return common.Result{}, err
	}
	if err := manifest.Write(root, m); err != nil {
		return common.Result{}, err
	}
	return result, nil
}

// RemoveEndpoint desfaz o AddEndpoint de name: remove as cópias do endpoint de
// exemplo e as linhas de registro, e atualiza o project.crud.
func RemoveEndpoint(root string, m manifest.Manifest, name string) (common.RemoveResult, error) {
	m = current(root, m)
	if err := CheckEndpointSupport(m); err != nil {
		return common.RemoveResult{}, err
	}
	if !m.HasEndpoint(name) {
		return common.RemoveResult{}, fmt.Errorf("endpoint %q não encontrado em %s", name, manifest.FileName)
	}
	plan, err := custom.EndpointRemovalPlan(root, m, name)
	if err != nil {
		return common.RemoveResult{}, err
	}
	result, err := common.Remove(root, plan)
	if err != nil {
		return common.RemoveResult{}, err
	}

	m.RemoveEndpoint(name)
	common.RecordRemoved(root, &m, result.Removed)
	if err := manifest.Write(root, m); err != nil {
		return common.RemoveResult{}, err
	}
	return result, nil
}

// current devolve o project.crud em disco, que pode ter mudado desde que m foi
// lido (ex.: por outro add endpoint), mantendo o campo template_endpoint de m.
func current(root string, m manifest.Manifest) manifest.Manifest {
	onDisk, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return m
	}
	onDisk.TemplateEndpoint = m.TemplateEndpoint
	return onDisk
}
