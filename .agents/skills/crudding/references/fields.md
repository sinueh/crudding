# Campos Tipados (`crudding add endpoint <nome> [campo:tipo[?] ...]`)

Passe pares `nome:tipo` após o nome do endpoint (`?` no fim torna o campo opcional/`NULL`) para gerar CRUD completo em vez de stubs (suportado em Python e Go; sem campos copia o endpoint de exemplo).

```sh
crudding add endpoint product name:str price:decimal image_url:str?
crudding add endpoint cart customer:str
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
- **Remoção**: `crudding remove endpoint <nome>` recusa remover um endpoint referenciado por outro `ref` (remova o dependente antes) e preserva os arquivos em `migrations/`.
- **Nomes proibidos**: `id` (gerado automaticamente), nomes fora de `snake_case`, palavras reservadas da linguagem e o nome do `template_endpoint` de exemplo do projeto.
- **Armazenamento**: usa `postgres` ou `mysql` quando configurado no projeto (gerando `migrations/NNNNNN_create_<nome>.up.sql` e `.down.sql`); sem banco SQL configurado, gera armazenamento em memória.
