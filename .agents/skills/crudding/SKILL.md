---
name: crudding
description: Scaffold de APIs (Go, Python, Node) e CRUD tipado com o CLI crudding. Use para criar projeto, adicionar/remover endpoint ou banco.
---

# crudding

> **ATENÇÃO (Anti-Viés de Scaffold):** O `crudding` gera **apenas o esqueleto CRUD básico de tabelas isoladas**. Rodar a CLI **NÃO conclui a tarefa** — é apenas o passo 1 (fundação). Você **DEVE** modelar todas as tabelas relacionais/itens e, depois do scaffold, implementar todas as regras de negócio, cálculos, JOINs, fluxos transacionais e autenticação exigidos pelo usuário.

## Fluxo Obrigatório
1. **Planeje a modelagem relacional completa antes de rodar a CLI**:
   - Não crie apenas tabelas rasas para os substantivos do prompt. Decomponha relações **1:N e N:N** em entidades e tabelas de itens usando `<entidade>:ref` (ex.: `user`, `store`, `product store:ref`, `cart user:ref`, `cart_item cart:ref product:ref`, `order user:ref`, `order_item order:ref product:ref`).
2. **Crie o projeto base**:
   `crudding new n:<nome> l:<python|go|node> f:<framework> a:<arch> d:<postgres|mysql|mongo>` (opções: `crudding stacks`)
3. **Gere todos os endpoints em lote** (respeitando a ordem das FKs `ref` e removendo o `template` inicial):
   - Tipos: `str` `text` `int` `float` `decimal` `bool` `date` `datetime` `uuid` `ref` (`cart:ref` → FK `cart_id`); `?` = opcional.
   - Exemplo em comando único:
     `crudding remove endpoint template && crudding add endpoint store name:str && crudding add endpoint product store:ref name:str price:decimal stock:int && crudding add endpoint cart user:ref && crudding add endpoint cart_item cart:ref product:ref quantity:int`
4. **Implemente as regras de negócio e requisitos do domínio (OBRIGATÓRIO)**:
   - O código gerado pela CLI faz apenas `INSERT/SELECT/UPDATE/DELETE` simples por linha.
   - Leia e edite os arquivos indicados em `editar:` (especialmente `service` e `handler`/`controller`) para implementar o que o prompt pediu: validações de domínio, controle de estoque, cálculo de totais/subtotais, queries com `JOIN`, transações (ex.: checkout), rotas de ação customizadas e autenticação/segurança.
5. **Banco de dados e migrações**:
   - Se adicionar novas tabelas ou colunas manualmente além do CLI, crie ou atualize a migração correspondente em `migrations/`.
   - Suba o banco (`docker compose up -d` aplica `migrations/` ao criar o container) ou aplique migrações com o runner nativo:
     - **Go**: aplicadas automaticamente ao iniciar a API (`MigratePostgres` / `MigrateMySQL` em `config.go`).
     - **Python**: `python -m src.app.config.config` (ou `python -m src.config.config` em `flat`), que executa `run_migrations()` sobre `migrations/*.up.sql` (os testes gerados também chamam `run_migrations()` automaticamente).
     - **Node**: `runMigrations()` exportado em `src/database.{js,ts}`.

## Regras (economia de contexto sem pular implementação)
- **Economize leituras no boilerplate, mas LEIA e EDITE onde há regra de negócio**: NÃO leia arquivos de infraestrutura intocados (`project.crud`, `.crudding/*`, `config`, `migrations/` geradas, schemas básicos que não vão mudar). Porém, **LEIA e EDITE** os arquivos de `service` e `handler`/`controller` (listados em `editar:`) para programar as regras de negócio do prompt.
- Estado do projeto: `crudding info --json` (não dê `cat` em `project.crud`).
- Para remover o endpoint de exemplo residual do projeto, use `crudding remove endpoint template` (nunca apague a pasta `.crudding/template/`, que guarda o molde interno).
- Não remova marcadores `crudding:*`; o `add/remove endpoint` e o `add db` dependem deles.
- Sempre passe argumentos (sem argumentos o CLI abre o menu interativo).

## Outros comandos
`crudding remove endpoint <nome>` · `crudding add db <banco>` · `crudding add skill` · `crudding info draw` · `crudding template --help` · `crudding <cmd> --help`

## Referências (ler apenas se necessário)
- [references/fields.md](references/fields.md) — tipos, chaves estrangeiras (`ref`), migrações e restrições de campos tipados.
- [references/templates.md](references/templates.md) — criação, uso, exportação e importação de templates personalizados (`init`, `add template`, `template`).
