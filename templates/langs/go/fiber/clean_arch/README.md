# go-fiber-clean_arch (Clean Architecture)

Projeto Go estruturado utilizando os princípios da **Clean Architecture** de Robert C. Martin (Uncle Bob).

---

## 🏛️ O que é a Clean Architecture?

A **Clean Architecture** organiza o sistema em camadas concêntricas, onde a Regra de Dependência estabelece que o código-fonte deve apontar sempre para dentro:

$$\text{Infrastructure (Persistence, HTTP)} \longrightarrow \text{Application (Use Cases)} \longrightarrow \text{Domain (Entities)}$$

### Princípios Chave:
1. **Entidades de Domínio (`internal/domain`)**:
   - Encapsulam as regras fundamentais de negócio do sistema. Não conhecem frameworks nem bancos de dados.
2. **Casos de Uso (`internal/application`)**:
   - Contêm as regras da aplicação que orquestram o fluxo de dados entre entidades e portas.
3. **Portas e Interfaces (`internal/ports`)**:
   - Definem as interfaces de repositório e serviços que a infraestrutura implementa (Inversão de Dependência).
4. **Infraestrutura (`internal/infrastructure` e `internal/handlers`)**:
   - Implementações de persistência (`infrastructure/persistence`) e handlers HTTP adaptados ao framework (fiber).
5. **Entrypoint Centralizado (`cmd/api/main.go`)**:
   - Monta as camadas de dentro para fora e sobe o servidor web via `internal/routes/routes.go`.

---

## 📁 Estrutura de Diretórios

```text
go-fiber-clean_arch/
├── cmd/
│   └── api/
│       └── main.go              # Ponto de entrada (fiber)
├── internal/
│   ├── domain/                  # Entidades centrais de negócio
│   │   ├── user.go
│   │   └── user_types.go
│   ├── ports/                   # Contratos e interfaces (Gateways)
│   │   ├── user_repository.go
│   │   └── user_service.go
│   ├── application/             # Casos de uso da aplicação
│   │   └── user.go
│   ├── infrastructure/          # Detalhes de infraestrutura
│   │   └── persistence/
│   │       └── user.go          # Implementação concreta do repositório
│   ├── handlers/                # Adaptadores de transporte HTTP
│   │   └── user.go
│   └── routes/                  # Centraliza todas as rotas da aplicação
│       └── routes.go
├── go.mod                       # Módulo Go
├── go.sum                       # Checksums de dependências
├── .gitignore                   # Arquivos ignorados pelo Git
└── README.md                    # Documentação da arquitetura
```

---

## API REST

- `POST /api/v1/users` - Cadastra um novo usuário
- `GET /api/v1/users` - Lista todos os usuários
- `GET /api/v1/users/{id}` - Obtém um usuário por ID
- `DELETE /api/v1/users/{id}` - Remove um usuário

## 🚀 Como Executar

### 1. Baixar dependências
```sh
go mod tidy
```

### 2. Executar o servidor
```sh
go run ./cmd/api
```

### 3. Executar os testes
```sh
go test -v ./...
```
