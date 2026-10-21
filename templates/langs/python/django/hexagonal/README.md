# python-django-hexagonal

Projeto Python gerado pelo crudding.

- Framework: django
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

Inicie o servidor de desenvolvimento:

```sh
python -m src.app.main runserver
```
