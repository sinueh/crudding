# python-flask-hexagonal

Projeto Python gerado pelo crudding.

- Framework: flask
- Arquitetura: hexagonal

O domínio e a aplicação dependem de ports; adapters de entrada e saída ficam separados em `src/app/`.

## API

- `GET /resource` e `GET /resource/{id}`: consulta recursos.
- `POST /resource`, `PATCH /resource/{id}` e `DELETE /resource/{id}`: operações CRUD.

O exemplo deixa a implementação das operações como exercício; até lá, as rotas respondem `501`.

## Executar

Instale as dependências:

```sh
python -m pip install -r requirements.txt
```

Inicie a aplicação:

```sh
python -m src.app.main
```
