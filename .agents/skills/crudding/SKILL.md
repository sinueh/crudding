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

---

## 1. Scaffolding a New Project (`crudding new`)

Generate a project non-interactively using `key:value` tokens (or `-flag value`):

```sh
crudding new n:<name> l:<lang> [r:<runtime>] f:<framework> a:<arch> [d:<db>] [o:<orm>]
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
   - `cmd/crudding/main.go`: CLI entrypoint.
   - `internal/cli/`: Command routing (`cli.go`) and subcommand handlers (`new.go`, `add_endpoint.go`, `remove_endpoint.go`, `add_db.go`, `init.go`, `add_template.go`, `template.go`, `info.go`, `list.go`).
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
