---
name: crudding
description: >-
  Use this skill when scaffolding projects, adding or removing CRUD endpoints,
  adding database infrastructure, inspecting architectures, or managing custom
  project templates (.crud / .zip) with the crudding CLI, or when developing and
  extending the crudding codebase itself.
---

# crudding CLI & Project Skill

`crudding` is a Go CLI tool that scaffolds projects across multiple languages, frameworks, and architectural patterns from embedded templates (`embed.FS`) or user-saved templates (`~/.crudding/templates/`), tracking project structure in a root `project.crud` manifest.

> **Important for AI Agents:** Always pass explicit arguments/flags (`crudding new ...`, `crudding add endpoint <name>`, `crudding add db <db>`) instead of running interactive TUI commands without arguments.
>
> **Não leia os arquivos gerados; use o resumo / `crudding info --json`.** Every command that writes files (`new`, `template use`, `add endpoint`, `remove endpoint`, `add db`) ends with a compact summary of what it created, changed and removed, the files to edit (`editar`), the routes (`rota`) and the next commands (`próximo`). Add `--json` (or `json:true`) to get the same data as one JSON object on stdout. Open only the files listed under `edit` when you need to change them.

### Output (`--json`)

```sh
crudding add endpoint product --json
```
```json
{"command":"add endpoint","project":"/abs/path/shop","created":["src/app/product/controller.py","src/app/product/service.py","src/app/product/models.py","src/app/product/__init__.py"],"modified":["src/app/main.py","project.crud"],"removed":[],"edit":["src/app/product/service.py","src/app/product/models.py"],"routes":["/product"],"next":["uvicorn src.app.main:app --reload"]}
```

- Paths are relative to `project` (the project root). `edit` = business logic (services, use cases) first, then models/entities.
- `crudding info --json` → `{name, project, language, runtime?, framework, architecture, databases, orm?, endpoints, routes, template_endpoint}`.
- In JSON mode stdout holds only the JSON object; errors go to stderr with exit code 1, and missing arguments are errors (no prompts).

---

## 1. Scaffolding a New Project (`crudding new`)

Generate a project non-interactively using `key:value` tokens (or `-flag value`):

```sh
crudding new n:<name> l:<lang> [r:<runtime>] f:<framework> a:<arch> [d:<db>] [o:<orm>] [--json]
```

### Supported Stacks (`crudding stacks`)

| Language (`l:`) | Runtimes (`r:`) | Frameworks (`f:`) | Architectures (`a:`) | ORMs (`o:`) | Databases (`d:`) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `go` | *(none)* | `raw`, `gin`, `fiber`, `chi` | `flat`, `layered`, `modular`, `hexagonal`, `clean_arch` | `none`, `gorm` | `none`, `postgres`, `mysql`, `mongo` |
| `python` | *(none)* | `raw`, `fastapi`, `django`, `flask` | `flat`, `layered`, `modular`, `hexagonal`, `clean_arch` | `none`, `sqlalchemy` | `none`, `postgres`, `mysql`, `mongo` |
| `node` | `js`, `ts` | `raw`, `express`, `fastify`, `nest` | `flat`, `layered`, `modular`, `hexagonal`, `clean_arch` | `none`, `prisma`, `typeorm` | `none`, `postgres`, `mysql`, `mongo` |

**Examples:**
```sh
# Go + Gin + Hexagonal + Postgres + GORM
crudding new n:minha-api l:go f:gin a:hexagonal d:postgres o:gorm

# Python + FastAPI + Clean Architecture
crudding new n:core-service l:python f:fastapi a:clean_arch

# Node (TypeScript) + Fastify + Modular
crudding new n:TsApi l:node r:ts f:fastify a:modular
```

---

## 2. Managing CRUD Endpoints (`add endpoint` / `remove endpoint`)

Run inside a project directory (where `project.crud` exists):

```sh
# Add a new CRUD endpoint (snake_case)
crudding add endpoint <name>

# Remove an existing CRUD endpoint
crudding remove endpoint <name>
```

- **How it works:** Copies the sample endpoint defined in `project.crud` under `"template_endpoint"` (`name`, `files`, `register`), replacing snake_case, kebab-case, camelCase, and PascalCase identifiers, and duplicates/removes registration lines (routes, imports, DI wiring) in the `register` files.
- **Constraints:**
  - The sample endpoint itself (e.g., `user` in Go, `resource` in Python) cannot be removed or re-added.
  - Language reserved keywords are rejected.

---

## 3. Adding Database Infrastructure (`add db`)

Run inside a generated project (`go`, `python`, or `node`, non-`custom` architecture):

```sh
crudding add db <postgres|mysql|mongo>
```

- Creates or updates `docker-compose.yml`, `.env.example`, and `migrations/`.
- Injects connection logic into `config/config.go` (Go), `config/config.py` (Python), or `src/database.{js,ts}` (Node), and updates `go.mod`, `requirements.txt`, or `package.json`.

---

## 4. Inspecting a Project (`info` / `info draw`)

```sh
# Show project name, stack, and registered endpoints from project.crud
crudding info
crudding info --json   # machine-readable: stack, endpoints, routes, template_endpoint

# Render an ASCII diagram of the project's architecture (or any supported architecture)
crudding info draw
crudding info draw a:hexagonal l:go
```

---

## 5. Custom Templates Workflow (`init`, `add template`, `template`)

Any directory can be turned into a reusable template stored in `~/.crudding/templates/` (or `$CRUDDING_TEMPLATES_DIR`):

```sh
# 1. (Optional) Initialize or refresh project.crud in an existing project directory
crudding init

# 2. Save a project directory as a named template
crudding add template ./meu-projeto meu-template

# 3. List saved templates
crudding template list

# 4. Generate a new project from a saved template
crudding template use meu-template novo-projeto

# 5. Export a saved template to a .zip archive
crudding template export meu-template [destino]

# 6. Import a template .zip directly into ~/.crudding/templates/ (non-interactive)
crudding template import ./meu-template.zip
```

---

## 6. Developing & Extending `crudding` (Repository Guide)

When modifying the `crudding` source code:

1. **Key Packages:**
   - `cmd/crudding/main.go` & `cmd/crd/main.go`: CLI entrypoints (`crudding` and abbreviated `crd`).
   - `internal/cli/`: Command routing (`cli.go`) and subcommand handlers (`new.go`, `add_endpoint.go`, `remove_endpoint.go`, `add_db.go`, `init.go`, `add_template.go`, `template.go`, `info.go`, `list.go`). `summary.go` builds the post-command summary and `--json` output from the generator results (`common.Result`/`RemoveResult`).
   - `internal/generator/`: Manifest-driven scaffold & patch engine (`common/`, `custom/`, `langs/golang/`, `langs/python/`, `langs/node/`).
   - `internal/manifest/manifest.go`: `project.crud` JSON schema, normalization, and parent-directory lookup.
   - `templates/langs/<lang>/`: Embedded `.tmpl` files (`frameworks/<fw>/`, `architectures/common/`, `architectures/<arch>/project.crud.tmpl`).
2. **Invariants to Preserve:**
   - Keep `crudding:db-*`, `crudding:compose-*`, and `crudding:env` markers intact in templates so `crudding add db` continues to work.
   - In `template_endpoint` sample endpoints, keep every registration (imports, route wiring, DI) on single lines mentioning the sample endpoint name inside `register` files.
   - Never relax path/symlink validations (`fs.ValidPath`, `filepath.IsLocal`, regular file checks) or directory overwrite protections.
3. **Verification Commands:**
   ```sh
   go vet ./...
   go test ./...
   make genall   # Generates all language/framework/architecture combinations into ./generated
   ```
