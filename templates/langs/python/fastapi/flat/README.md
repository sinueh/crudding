# python-fastapi-flat

Projeto Python gerado pelo crudding.

- Framework: fastapi
- Arquitetura: flat

O pacote da aplicação fica em `src/`, e os testes ficam em `tests/`.

## API

- `GET /resource` e `GET /resource/{id}`: consulta recursos.
- `POST /resource`, `PATCH /resource/{id}` e `DELETE /resource/{id}`: operações CRUD.

O exemplo deixa a implementação das operações como exercício; até lá, as rotas respondem `501`.

## Executar

Instale as dependências:

```sh
python -m pip install -r requirements.txt
```

Inicie a API:

```sh
uvicorn src.main:app --reload
```
