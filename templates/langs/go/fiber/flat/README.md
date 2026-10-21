# go-fiber-flat (Flat Architecture)

Projeto Go estruturado utilizando uma **Flat Architecture** (Arquitetura Plana).

---

## 🏛️ O que é a Flat Architecture?

A **Flat Architecture** organiza todos os arquivos no mesmo nível do pacote `main`. É o padrão ideal para protótipos rápidos, ferramentas de linha de comando ou microsserviços simples com poucos endpoints, eliminando o overhead de pacotes internos múltiplos.

### Princípios Chave:
1. **Pacote Único (`main`)**:
   - Todo o código reside na raiz e pertence ao pacote `main`.
2. **Separação por Arquivo**:
   - Cada arquivo cuida de uma responsabilidade:
     - `user_model.go`: Entidade de dados.
     - `user_types.go`: DTOs (Request / Response).
     - `user_service.go`: Regras de negócio e armazenamento em memória.
     - `user_handler.go`: Handler HTTP específico do framework (fiber).
     - `routes.go`: Roteamento e mapeamento de endpoints (`SetupRouter`).
     - `main.go`: Ponto de entrada que inicializa a aplicação.

---

## 📁 Estrutura de Diretórios

```text
go-fiber-flat/
├── main.go               # Ponto de entrada da aplicação
├── routes.go             # Configuração de rotas HTTP
├── user_model.go         # Entidade User
├── user_types.go         # DTOs de entrada e saída
├── user_service.go       # Regras de negócio e persistência em memória
├── user_handler.go       # Handler HTTP (fiber)
├── go.mod                # Módulo Go
├── go.sum                # Checksums de dependências
├── .gitignore            # Arquivos ignorados pelo Git
└── README.md             # Documentação da arquitetura
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

### 2. Executar a aplicação
```sh
go run .
```

### 3. Executar os testes
```sh
go test -v ./...
```
