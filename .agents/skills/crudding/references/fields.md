# Campos Tipados (`crudding add endpoint <nome> [campo:tipo[?] ...]`)

Passe pares `nome:tipo` após o nome do endpoint (`?` no fim torna o campo opcional/`NULL`) para gerar o scaffold CRUD tipado da tabela (suportado em Python e Go; sem campos copia o endpoint de exemplo).

> **Importante:** O scaffold tipado gera apenas operações CRUD básicas por linha (`CREATE`, `READ`, `UPDATE`, `DELETE`). Sempre decomponha relações 1:N / N:N em tabelas de itens com `ref` (ex.: `cart_item`, `order_item`) e implemente as regras de negócio, cálculos, JOINs e transações nos arquivos de `service`/`handler` após rodar o CLI.

```sh
# Encadeie múltiplos endpoints (e remova o exemplo template) em um único comando shell:
crudding remove endpoint template && \
  crudding add endpoint store name:str && \
  crudding add endpoint product store:ref name:str price:decimal stock:int image_url:str? && \
  crudding add endpoint cart customer:str && \
  crudding add endpoint cart_item cart:ref product:ref quantity:int
```

## Tipos suportados
- `str`: texto curto (`VARCHAR(255)`, `max_length=255`).
- `text`: texto longo (`TEXT`).
- `int`: inteiro (`INTEGER`/`INT`).
- `float`: ponto flutuante (`DOUBLE PRECISION`/`DOUBLE`).
- `decimal`: valor monetário/preciso (`NUMERIC(12, 2)`/`DECIMAL(12, 2)`).
- `bool`: booleano (`BOOLEAN`).
- `date`: data `AAAA-MM-DD` (`DATE`).
- `datetime`: data e hora ISO-8601 (`TIMESTAMPTZ`/`DATETIME(6)`).
- `uuid`: identificador UUID (`UUID`/`CHAR(36)`).
- `ref`: chave estrangeira para `<nome>.id`, gravada na coluna `<nome>_id` (`BIGINT` + `FOREIGN KEY`).

## Regras e restrições
- **Ordem de criação para `ref`**: o endpoint referenciado precisa existir com campos tipados antes de ser referenciado (ex.: crie `cart` e `product` antes de `cart_item`).
- **Remoção**: `crudding remove endpoint <nome>` recusa remover um endpoint referenciado por outro `ref` (remova o dependente antes) e preserva os arquivos em `migrations/`. Você pode rodar `crudding remove endpoint template` para limpar o endpoint de exemplo da estrutura do projeto (mantendo o molde em `.crudding/template/*`).
- **Nomes proibidos em `add endpoint`**: `id` (gerado automaticamente), nomes fora de `snake_case`, palavras reservadas da linguagem e o nome do `template_endpoint` (`template`).
- **Armazenamento e migrações**: usa `postgres` ou `mysql` quando configurado no projeto (gerando `migrations/NNNNNN_create_<nome>.up.sql` e `.down.sql`); sem banco SQL configurado, gera armazenamento em memória. As migrações são montadas em `/docker-entrypoint-initdb.d` no `docker-compose.yml` e podem ser aplicadas a qualquer momento pelo código gerado (`MigratePostgres`/`MigrateMySQL` em Go ao subir a API, `python -m src.app.config.config` ou `python -m src.config.config` em Python, e `runMigrations()` em `src/database.{js,ts}` no Node).
