# AGENTS.md

Guia de contexto, arquitetura e instruções para agentes de IA atuando no repositório **crudding**.

---

## 1. Visão Geral do Projeto

**crudding** é uma ferramenta de linha de comando (CLI) interativa desenvolvida em **Go 1.27** usando **Bubble Tea** (`github.com/charmbracelet/bubbletea`). Seu objetivo é fazer o scaffolding de novos projetos em diferentes linguagens, frameworks e padrões arquiteturais a partir de templates pré-definidos embutidos diretamente no binário (`embed.FS`).

### Capacidades Suportadas
- **Linguagens**:
  - `python`: frameworks `raw`, `fastapi`, `django`, `flask`.
  - `go`: frameworks `raw`, `gin`, `fiber`, `chi`.
  - `node`: runtimes `js` e `ts`; frameworks `raw`, `express`, `nest`, `fastify`.
- **Arquiteturas (Padrões de Pastas)**:
  - **Para Golang**:
    1. `flat`: tudo na raiz no pacote `main` (`user_model.go`, `user_types.go`, `user_service.go`, `user_handler.go`, `routes.go`, `main.go`, `README.md`).
    2. `layered`: separação clássica em camadas sob `internal/` (`models`, `types`, `service`, `handlers`, `routes`), roteamento em `internal/routes/routes.go` e injeção em `cmd/api/main.go`.
    3. `modular`: *package-by-feature* com módulos (`internal/<feature>/` com `model.go`, `types.go`, `service.go`, `handler.go`), roteamento em `internal/routes/routes.go` e injeção em `cmd/api/main.go`.
    4. `hexagonal`: *Ports & Adapters* com `internal/domain` (entidades e tipos), `internal/ports` (interfaces repository/service), `internal/application` (serviços de aplicação), `internal/adapters/postgres` (adaptador secundário), `internal/handlers` (adaptador primário HTTP), roteamento em `internal/routes/routes.go` e injeção em `cmd/api/main.go`.
    5. `clean arch`: Clean Architecture concêntrica com `internal/domain` (entidades e tipos), `internal/ports` (fronteiras repository/service), `internal/application` (casos de uso), `internal/infrastructure/persistence` (implementação de persistência), `internal/handlers` (controladores HTTP), roteamento em `internal/routes/routes.go` e injeção em `cmd/api/main.go`.
   - **Para Python**:
      - `flat`, `layered`, `modular`, `hexagonal` e `clean_arch`, com estruturas próprias sob `src/`.
   - **Para Node**:
      - `monorepo`
      - `camadas` (Layered / Onion básico)
      - `clean arch` (Clean Architecture / Uncle Bob)
      - `hexagonal` (Ports & Adapters)
      - `4horse` (Routes, Controllers, Services, Repositories)

Cada arquitetura em Go possui seu próprio `README.md` detalhado explicando o funcionamento, princípios e comandos de execução.

- **Para Node**: runtimes `js` e `ts`; frameworks `raw`, `express`, `fastify`, `nest`; arquiteturas `flat`, `layered`, `modular`, `hexagonal` e `clean_arch`.

---

## 2. Estrutura do Repositório

```text
crudding/
├── cmd/
│   └── crudding/
│       └── main.go              # Ponto de entrada mínimo (delega para internal/cli)
│   └── genall/
│       └── main.go              # Gera todas as combinações suportadas para validação
├── internal/
│   ├── cli/
│   │   ├── cli.go               # Registro de comandos, roteador Run e geração automática de --help
│   │   ├── new.go               # Comando padrão de criação de projeto (runNew)
│   │   ├── add_endpoint.go      # Comando "crudding add endpoint" (runAddEndpoint)
│   │   ├── add_db.go            # Comando "crudding add db" (runAddDB)
│   │   ├── init.go              # Comando "crudding init" (runInit)
│   │   ├── template.go          # Comando "crudding template" (runTemplate)
│   │   └── remove_endpoint.go   # Comando "crudding remove endpoint" (runRemoveEndpoint)
│   ├── generator/
│   │   ├── common/
│   │   │   ├── files.go         # Utilitários de escrita de arquivos e template parsing
│   │   │   ├── scaffold.go      # Criação do projeto (Plan) e gravação do project.crud
│   │   │   ├── structure.go     # ListStructure (walk com ignore) e RecordCreated/RecordRemoved para directories/files
│   │   │   ├── patch.go         # Alterações em projeto existente (Patch): arquivos novos, pacotes Python e inserções em markers, com rollback
│   │   │   └── remove.go        # Remoção de endpoint (RemovalPlan): exclusão de arquivos, limpeza de linhas e diretórios vazios
│   │   ├── custom/
│   │   │   ├── init.go          # Init: grava/atualiza directories e files no project.crud
│   │   │   ├── endpoint.go      # add/remove endpoint a partir do campo "endpoint" (endpoint de exemplo)
│   │   │   └── template.go      # Plan de "crudding template": cópia com troca de nome; Unlisted
│   │   ├── langs/
│   │   │   ├── golang/
│   │   │   │   └── generate.go  # Gerador Go; declara o endpoint de exemplo (user) por arquitetura
│   │   │   ├── node/generate.go   # Gerador específico para projetos Node (JS/TS)
│   │   │   └── python/
│   │   │       ├── generate.go   # Gerador específico para projetos Python
│   │   │       └── sample.go     # Endpoint de exemplo (resource) por arquitetura: arquivos e campo endpoint
│   │   ├── database.go          # Dispatcher de "crudding add db"
│   │   ├── generator.go         # Dispatcher central do gerador
│   │   ├── endpoint.go          # Dispatcher de "crudding add endpoint" e "crudding remove endpoint"
│   ├── manifest/
│   │   └── manifest.go          # Leitura/escrita do project.crud (JSON) e busca nos diretórios pais
│   ├── project/
│   │   ├── config.go            # Configuração compartilhada do projeto
│   │   ├── name.go              # Leitura e sanitização de nome do projeto
│   │   └── project.go           # Utilitário de slugify (ProjectSlug)
│   ├── test/                    # Testes de integração e unidade de internal/
│   └── tui/
│       └── menu.go              # Menu Bubble Tea para seleção interativa
├── templates/                   # Arquivos de template (.tmpl) para cada stack
│   ├── databases/               # Templates de banco (postgres, mysql, mongo): compose, env, migrations, conexão
│   └── langs/                   # Templates de linguagens (go, node, python)
├── examples/
│   └── go-template/             # Projeto-template de exemplo (documentação e fixture de teste; não é embutido)
├── assets.go                    # Declaração do embed.FS para a pasta templates/
├── go.mod                       # Definição de módulo e dependências
├── go.sum
├── LICENSE
└── README.md
```

---

## 3. Fluxo de Execução

1. **Seleção Interativa (`internal/cli/new.go` & `internal/tui/menu.go`)**:
   - Pergunta a **Linguagem** (`python`, `go`, `node`).
   - Se `node`, pergunta o **Runtime** (`js`, `ts`).
   - Pergunta o **Framework** compatível com a linguagem selecionada.
   - Pergunta a **Arquitetura** (com opções dinâmicas dependendo da linguagem selecionada via `architectureOptionsFor`).
   - Se `go` ou `python`, pergunta o **Banco de Dados** (`nenhum`, `postgres`, `mysql`, `mongo`).
   - Lê e valida o **Nome do Projeto** via prompt de texto.
2. **Validação de Segurança (`internal/project/name.go`)**:
   - Rejeita caracteres inválidos e tentativas de path traversal (`..`, barras, etc.).
   - Garante que diretórios existentes nunca sejam sobrescritos (`internal/generator/common/files.go`).
3. **Geração dos Arquivos (`internal/generator/` e `internal/generator/langs/`)**:
   - Define a estrutura de diretórios baseado no layout da arquitetura escolhida.
   - Em Go e Python, inclui o diretório `config/` com `config.go` ou `config.py` para carregar configurações gerais e clientes de banco.
   - Se um banco de dados for selecionado na criação, gera a infraestrutura (`docker-compose.yml`, `.env.example`, `migrations/`), ativa os blocos correspondentes no `config.*` via condicionais de template e adiciona as dependências do driver (`go.mod` ou `requirements.txt`).
   - Processa os templates embutidos utilizando `text/template` e grava no diretório de destino.
   - Em caso de falha no meio do processo, o diretório gerado é limpo automaticamente (`defer os.RemoveAll`).

4. **Manifesto**: `common.Scaffold` grava `project.crud` (JSON, `internal/manifest`) na raiz de todo projeto gerado, por último, com `ignore` vazio e `directories`/`files` listando a estrutura gerada (`common.ListStructure`). Todo `project.crud` tem as mesmas chaves do gerado por `crudding init`: listas vazias são gravadas como `[]` (`Manifest.normalize`).

### `crudding add endpoint`

1. `manifest.Find` procura o `project.crud` a partir do diretório atual, subindo pelos pais.
2. `generator.CheckEndpointSupport` confirma que o campo `endpoint` do manifesto está preenchido e válido. Não há código por linguagem ou arquitetura para criar endpoints: tudo sai desse campo (ver "Campo `endpoint`" abaixo).
3. `project.ReadEndpointName` lê o nome (snake_case).
4. `custom.EndpointPatch` monta um `common.Patch` copiando o endpoint de exemplo (`Raw`) e duplicando as linhas dele nos arquivos de `register` (`Rewrites`).
5. `common.Apply` valida tudo antes de gravar (nada pode ser sobrescrito, todo arquivo de `register` menciona o exemplo) e desfaz as alterações se a gravação falhar.

### `crudding remove endpoint`

1. `manifest.Find` localiza a raiz do projeto e lê o `project.crud`.
2. `generator.CheckEndpointSupport` confirma que o campo `endpoint` está preenchido.
3. `project.ReadEndpointName` lê o nome (ou via argumentos `name:<nome>`, `n:<nome>` ou posicional).
4. Verifica se o endpoint está registrado no `project.crud`.
5. `custom.EndpointRemovalPlan` monta um `common.RemovalPlan`: as cópias a deletar, as pastas que devem ser excluídas se ficarem vazias e as linhas duplicadas nos arquivos de `register`.
6. `common.Remove` remove os arquivos, limpa as linhas das rotas, limpa pastas vazias e atualiza o manifesto `project.crud` com rollback caso haja erro na gravação.

### `crudding add db <postgres|mysql|mongo>`

1. `manifest.Find` localiza a raiz do projeto e lê o `project.crud`.
2. Valida o banco de dados informado (`postgres`, `mysql`, `mongo`).
3. `generator.AddDatabase` monta um `common.Patch`:
   - `docker-compose.yml` (se não existir cria, se já existir insere o novo serviço/volume);
   - `.env.example` com as variáveis de conexão;
   - pasta `migrations/` com a migração inicial (`.up.sql`/`.down.sql` para SQL e `.js` para Mongo);
   - em Go e Python, insere as configurações e funções de conexão dentro do `config/config.go` ou `config/config.py` existente nos markers `crudding:db-*`, suportando múltiplos bancos no mesmo arquivo; em Node grava o arquivo de conexão sob `src/database.{js,ts}`.
4. `common.Apply` valida a ausência de conflitos e grava os arquivos.
5. Atualiza dependências (`go.mod`, `requirements.txt` ou `package.json`) com os drivers do banco e atualiza o manifesto `project.crud` com o banco adicionado.

**Para uma linguagem ou arquitetura ter `add endpoint`**: o `Plan` do gerador deve gerar um endpoint de exemplo e preencher `common.Plan.Endpoint` (`name`, `files`, `register`), como fazem `langs/golang/generate.go` (`user`) e `langs/python/sample.go` (`resource`). O exemplo precisa de três coisas:
- todo arquivo dele tem o nome do exemplo no caminho;
- o registro (rotas, imports, injeção de dependência) fica em linhas únicas que mencionam o exemplo, nos arquivos de `register`;
- nenhuma declaração de várias linhas do exemplo fica num arquivo compartilhado (por isso, no Go hexagonal/clean_arch, os DTOs e ports ficam em `user_types.go`, `user_repository.go` e `user_service.go`).

O `TestEndpointRoundTripAllCombinations` confere isso em todas as combinações. Node ainda não gera o campo. Os markers `crudding:db-*`, `crudding:compose-*` e `crudding:env` são do `add db` e não podem ser removidos dos templates.

### `crudding init` e `crudding template <pasta> [nome]` (projetos-template)

Caminho paralelo ao registry: qualquer projeto vira template, e a estrutura fica no próprio `project.crud`. O formato está documentado no `README.md` e exemplificado em `examples/go-template/`.

1. `custom.Init` grava (ou atualiza) o `project.crud` do diretório atual. Num manifesto novo: `name` = nome da pasta, `architecture: "custom"` e `ignore` vazio (nada é preenchido automaticamente, nem a partir do `.gitignore`). Em seguida chama `common.ListStructure`, que aplica o `ignore` e sempre pula `project.crud` e `.git`, e grava `directories` e `files`. Rodar de novo só regenera essas duas listas.
2. `crudding template` lê o manifesto com `manifest.ReadFS`, avisa sobre arquivos fora de `files` (`custom.Unlisted`) e chama `generator.GenerateFromTemplate`.
3. `custom.Plan` lê e valida tudo antes de gravar e troca o `name` do template pelo nome novo (exata, slug e snake_case) nos caminhos e no conteúdo de texto. O `Prepare` grava as pastas e os arquivos (só o bit de execução é preservado). `Plan.Manifest` faz o `common.Scaffold` gravar o manifesto herdado do template, com `name`, `template` e os caminhos trocados.
4. `add db` recusa `architecture: "custom"` (`AddDatabase`). `add/remove endpoint` dependem só do campo `endpoint`, como em qualquer projeto.

### Campo `endpoint` (endpoints personalizados)

É o único mecanismo de `add/remove endpoint` (`generator.AddEndpoint`/`RemoveEndpoint` chamam `custom.EndpointPatch`/`custom.EndpointRemovalPlan`). Os projetos gerados já trazem o campo apontando para o endpoint de exemplo.

- O `add` copia `files` do endpoint de exemplo (`name`) trocando o nome por snake/kebab/Pascal/camel (`common.Patch.Raw`). Num exemplo de uma palavra só, o nome seguido de maiúscula (`userRepo`) conta como camelCase. Palavras reservadas da linguagem são recusadas (`reservedNames`). Em cada arquivo de `register`, repete logo abaixo cada bloco de linhas consecutivas que menciona o exemplo (`common.Patch.Rewrites`).
- O `remove` apaga as cópias, as pastas cujo caminho mudou com o nome e as linhas repetidas (`common.RemovalPlan`).
- Linhas que mencionam outro endpoint cujo nome contém o do exemplo (`order_item` com exemplo `item`) não contam como do exemplo (`endpointRecipe.others`).
- O próprio exemplo não pode ser adicionado nem removido.
- `AddEndpoint`, `RemoveEndpoint` e `AddDatabase` mantêm `files`/`directories` em dia com `common.RecordCreated`/`RecordRemoved`, em qualquer projeto.

Como o template vem de fora do binário, todo caminho passa por `checkPaths` (`fs.ValidPath`, `filepath.IsLocal`, `project.crud` reservado), symlinks são rejeitados e o destino não pode ficar dentro da pasta do template. Não relaxe essas verificações.

---

## 4. Comandos e Ambiente de Desenvolvimento

### Comandos Go Padrão
```sh
# Executar a CLI em desenvolvimento
go run ./cmd/crudding

# Transformar o projeto atual em template e gerar um projeto a partir dele
go run /caminho/do/crudding/cmd/crudding init
go run /caminho/do/crudding/cmd/crudding template ./examples/go-template meu-app

# Adicionar um endpoint a um projeto gerado (rodar dentro do projeto)
go run /caminho/do/crudding/cmd/crudding add endpoint

# Executar testes unitários
go test ./...

# Instalar o binário localmente
go install ./cmd/crudding
```

### Nota sobre Ambientes em Containers / Flatpak (OSTree)
Se você estiver executando em uma sandbox ou terminal de IDE flatpak (ex: VS Code Flatpak) onde o binário `go` está apenas no host:
```sh
flatpak-spawn --host sh -c 'cd /caminho/do/projeto && go test ./...'
```

---

## 5. Diretrizes para Agentes de Código

Ao implementar novas features, correções ou refatorações neste repositório:

1. **Zero dependências extras desnecessárias**:
   - O projeto utiliza a biblioteca padrão do Go sempre que possível (`text/template`, `io/fs`, `embed`, `path/filepath`). A única dependência externa principal é `bubbletea`.
2. **Preservação de Templates e Embed**:
   - Ao adicionar novos templates dentro de `templates/`, certifique-se de que os nomes e caminhos correspondam exatamente às referências nos geradores de cada linguagem.
   - Lembre-se de que `assets.go` utiliza `//go:embed all:templates`, portanto os caminhos dentro do `embed.FS` contêm o prefixo `templates/...`.
3. **Tratamento de Erros e Limpeza**:
   - Sempre use `fmt.Errorf("...: %w", err)` para empacotamento semântico de erros.
   - Mantenha a proteção contra sobrescrita acidental de diretórios.
4. **Adicionando Novos Frameworks ou Arquiteturas**:
   - Atualize `frameworkOptionsFor` e `architectureOptionsFor` em `cmd/crudding/main.go`.
   - Adicione no mapa de arquiteturas/layouts no respectivo pacote (`internal/generator/<lang>/generate.go`).
   - Crie os templates correspondentes sob `templates/<lang>/`.
   - Adicione os casos de teste em `internal/test/`.
