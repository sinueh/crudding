---
name: crudding
description: Scaffold de APIs (Go, Python, Node) e CRUD tipado com o CLI crudding. Use para criar projeto, adicionar/remover endpoint ou banco.
---

# crudding

## Fluxo
1. `crudding new n:<nome> l:<python|go|node> f:<framework> a:<arch> d:<postgres|mysql|mongo>`
   (opções: `crudding stacks`)
2. `crudding add endpoint product name:str price:decimal image_url:str?`
   tipos: `str` `text` `int` `float` `decimal` `bool` `date` `datetime` `uuid` `ref` (`cart:ref` → FK `cart_id`); `?` = opcional
3. Escreva só a regra de negócio nos arquivos da linha `editar:` do resumo.
4. Suba o banco e aplique `migrations/` conforme o `README.md` gerado.

## Regras (economia de contexto)
- **NÃO** leia os arquivos gerados: o resumo lista o que foi criado. Use `--json` para ler por máquina.
- Estado do projeto: `crudding info --json` (não dê `cat` em `project.crud`).
- Não edite linhas de registro de rotas nem marcadores `crudding:*`; o `add/remove endpoint` e o `add db` dependem deles.
- Sempre passe argumentos (sem argumentos o CLI abre o menu interativo).

## Outros comandos
`crudding remove endpoint <nome>` · `crudding add db <banco>` · `crudding info draw` · `crudding template --help` · `crudding <cmd> --help`

## Referências (ler apenas se necessário)
- [references/fields.md](references/fields.md) — tipos, chaves estrangeiras (`ref`), migrações e restrições de campos tipados.
- [references/templates.md](references/templates.md) — criação, uso, exportação e importação de templates personalizados (`init`, `add template`, `template`).
