# python-flask-clean-arch

Projeto Python gerado pelo crudding.

- Framework: flask
- Arquitetura: clean_arch

As dependências apontam para as camadas internas; a composição da aplicação fica em `src/app/bootstrap.py`.

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
