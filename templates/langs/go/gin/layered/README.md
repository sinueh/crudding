# go-gin-layered (Layered Architecture)

Projeto Go estruturado utilizando a **Layered Architecture** (Arquitetura em Camadas).

---

## 🏛️ O que é a Layered Architecture?

A **Layered Architecture** organiza o sistema em camadas bem delineadas, com fluxo unidirecional de dependências:

$$\text{Routes (SetupRouter)} \longrightarrow \text{Handlers (Transporte)} \longrightarrow \text{Services (Negócio)} \longrightarrow \text{Models e Types (Dados)}$$

### Princípios Chave:
1. **Entrypoint e Injeção de Dependência (`cmd/api/main.go`)**:
   - Inicializa as dependências (`service` -> `handler` -> `routes`) e sobe o servidor web.
2. **Rotas Centralizadas (`internal/routes/routes.go`)**:
   - Configura o roteador da aplicação recebendo os handlers injetados e mapeando os endpoints HTTP (`/api/v1/...`).
3. **Handlers HTTP (`internal/handlers`)**:
   - Tratam requisições, validam parâmetros/payloads usando os DTOs (`internal/types`) e delegam a regra para os serviços.
4. **Services (`internal/service`)**:
   - Orquestram a lógica de negócio, realizam a conversão entre DTOs e entidades (`models`), e gerenciam o armazenamento em memória.
5. **Models e Types (`internal/models` e `internal/types`)**:
   - `internal/models/`: define as entidades de dados / modelos.
   - `internal/types/`: define DTOs de entrada e saída (Request/Response) da API.

---

## 📁 Estrutura de Diretórios

```text
go-gin-layered/
├── cmd/
│   └── api/
│       └── main.go           # Ponto de entrada e injeção de dependências (gin)
├── internal/
│   ├── handlers/             # Controladores HTTP específicos do framework
│   │   └── user.go
│   ├── routes/               # Configuração e mapeamento de rotas
│   │   └── routes.go
│   ├── service/              # Regras de negócio da aplicação
│   │   └── user.go
│   ├── types/                # DTOs (Request / Response)
│   │   └── user.go
│   └── models/               # Entidades de dados
│       └── user.go
├── go.mod                    # Módulo Go
├── go.sum                    # Checksums de dependências
├── .gitignore                # Arquivos ignorados pelo Git
└── README.md                 # Documentação da arquitetura
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

### 2. Executar a API
```sh
go run ./cmd/api
```

### 3. Executar os testes
```sh
go test -v ./...
```
