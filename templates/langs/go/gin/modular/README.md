# go-gin-modular (Modular Architecture)

Projeto Go estruturado utilizando a **Modular Architecture** (*Package by Feature* / Vertical Slice).

---

## 🏛️ O que é a Modular Architecture?

A **Modular Architecture** organiza o código em torno de módulos funcionais de domínio (`internal/user`, `internal/<recurso>`), mantendo alta coesão dentro do pacote e baixo acoplamento entre módulos.

### Princípios Chave:
1. **Entrypoint e Injeção de Dependência (`cmd/api/main.go`)**:
   - Inicializa as dependências dos módulos (`service` -> `handler` -> `routes`) e sobe o servidor web.
2. **Rotas Centralizadas (`internal/routes/routes.go`)**:
   - Registra os endpoints de cada módulo funcional através de seus handlers.
3. **Módulo por Funcionalidade (`internal/<feature>/`)**:
   - Cada pasta encapsula suas entidades (`model.go`), DTOs (`types.go`), serviço (`service.go`) e handler (`handler.go`).
4. **Escalabilidade**:
   - Novos recursos são adicionados de forma independente via `crudding add endpoint <recurso>`, gerando um novo pacote completo isolado sob `internal/<recurso>/`.

---

## 📁 Estrutura de Diretórios

```text
go-gin-modular/
├── cmd/
│   └── api/
│       └── main.go             # Ponto de entrada HTTP REST (gin)
├── internal/
│   ├── routes/                 # Configuração e mapeamento de rotas
│   │   └── routes.go
│   └── user/                   # Módulo de Usuários (Vertical Slice)
│       ├── model.go            # Entidades do banco de dados
│       ├── types.go            # DTOs (Request / Response)
│       ├── service.go          # Regras de negócio e armazenamento em memória
│       └── handler.go          # Handler HTTP do framework
├── go.mod                      # Módulo Go
├── go.sum                      # Checksums de dependências
├── .gitignore                  # Arquivos ignorados pelo Git
└── README.md                   # Documentação da arquitetura
```

---

## API

- `POST /api/v1/users` - Cria um novo usuário (`name`, `email`, `password`)
- `GET /api/v1/users` - Lista todos os usuários
- `GET /api/v1/users/{id}` - Obtém um usuário por ID
- `DELETE /api/v1/users/{id}` - Remove um usuário

## 🚀 Como Executar

### 1. Baixar dependências
```sh
go mod tidy
```

### 2. Executar a API HTTP
```sh
go run ./cmd/api
```

### 3. Executar os testes
```sh
go test -v ./...
```
