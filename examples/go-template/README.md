# go-template

Projeto-template de exemplo do crudding: uma API Go mínima com `GET /health` e o endpoint de exemplo `GET/POST /items`.

## Executar

```sh
go run ./cmd/api
```

## Usar como template

O `project.crud` desta pasta foi criado com `crudding init` e lista toda a estrutura do projeto. Para gerar um projeto novo a partir dele:

```sh
crudding template caminho/para/go-template minha-api
```

O nome `go-template` é trocado por `minha-api` em todos os arquivos (`go.mod`, imports, este README) e nos caminhos. Depois de criar ou remover arquivos aqui, rode `crudding init` de novo para atualizar a lista.

## Adicionar endpoints

O campo `endpoint` do `project.crud` aponta para `internal/item/` como modelo. Dentro de um projeto gerado a partir deste template:

```sh
crudding add endpoint invoice      # cria internal/invoice/handler.go e registra em cmd/api/main.go
crudding remove endpoint invoice   # desfaz
```
