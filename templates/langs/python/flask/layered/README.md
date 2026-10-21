# python-flask-layered

Projeto Python gerado pelo crudding.

- Framework: flask
- Arquitetura: layered

Controllers, services e models ficam organizados em `src/app/`. Cada controller trata as operações HTTP do seu recurso e chama a service correspondente.

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
