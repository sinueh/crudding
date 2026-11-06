# Templates Personalizados (`init`, `add template`, `template`)

Qualquer projeto pode ser salvo como template reutilizável em `~/.crudding/templates/` (ou `$CRUDDING_TEMPLATES_DIR`).

```sh
# 1. (Opcional) Inicializar ou atualizar o project.crud no diretório atual
crudding init

# 2. Salvar diretório como template nomeado
crudding add template ./meu-projeto meu-template

# 3. Listar templates salvos
crudding template list

# 4. Criar novo projeto a partir de um template salvo
crudding template use meu-template novo-projeto

# 5. Exportar template salvo para arquivo .zip
crudding template export meu-template [destino.zip]

# 6. Importar template .zip diretamente para ~/.crudding/templates/
crudding template import ./meu-template.zip
```

## Notas
- Projetos inicializados com `crudding init` recebem `architecture: "custom"`.
- `crudding add db` não é suportado em `architecture: "custom"`.
- `crudding add/remove endpoint` funciona em projetos customizados desde que o campo `template_endpoint` (`name`, `files`, `register`) esteja preenchido no `project.crud`.
