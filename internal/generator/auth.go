package generator

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh/crudding/internal/generator/common"
	"github.com/sinueh/crudding/internal/manifest"
)

// SupportedAuthStrategies lista as estratégias de autenticação suportadas.
var SupportedAuthStrategies = []string{"jwt", "session", "api-key"}

// SupportedHashers lista os algoritmos de hash suportados.
var SupportedHashers = []string{"bcrypt", "argon2id"}

// ValidateAuthStrategy valida a estratégia informada.
func ValidateAuthStrategy(strategy string) error {
	if !slices.Contains(SupportedAuthStrategies, strategy) {
		return fmt.Errorf("estratégia de autenticação não suportada: %q (opções: %s)", strategy, strings.Join(SupportedAuthStrategies, ", "))
	}
	return nil
}

// ValidateHasher valida o algoritmo de hash informado.
func ValidateHasher(hasher string) error {
	if !slices.Contains(SupportedHashers, hasher) {
		return fmt.Errorf("algoritmo de hash não suportado: %q (opções: %s)", hasher, strings.Join(SupportedHashers, ", "))
	}
	return nil
}

// AuthOptions reúne as configurações para "crudding add auth".
type AuthOptions struct {
	Strategy   string
	Roles      []string
	Hasher     string
	HasRefresh bool
}

// AddAuth adiciona o módulo completo de autenticação ao projeto.
func AddAuth(root string, m manifest.Manifest, opts AuthOptions, templates fs.FS) (common.Result, error) {
	if m.HasAuth() {
		return common.Result{}, fmt.Errorf("o projeto já possui autenticação configurada (%s)", m.Auth.Strategy)
	}
	if m.Architecture == manifest.CustomArchitecture {
		return common.Result{}, fmt.Errorf("add auth não é suportado para projetos com arquitetura custom")
	}

	opts.Strategy = strings.ToLower(strings.TrimSpace(opts.Strategy))
	if opts.Strategy == "" {
		opts.Strategy = "jwt"
	}
	if opts.Strategy == "apikey" {
		opts.Strategy = "api-key"
	}
	if err := ValidateAuthStrategy(opts.Strategy); err != nil {
		return common.Result{}, err
	}

	opts.Hasher = strings.ToLower(strings.TrimSpace(opts.Hasher))
	if opts.Hasher == "" {
		opts.Hasher = "bcrypt"
	}
	if opts.Hasher == "argon2" {
		opts.Hasher = "argon2id"
	}
	if err := ValidateHasher(opts.Hasher); err != nil {
		return common.Result{}, err
	}

	if len(opts.Roles) == 0 {
		opts.Roles = []string{"admin", "user"}
	}

	templatesRoot, err := fs.Sub(templates, "templates")
	if err != nil {
		return common.Result{}, fmt.Errorf("acessar templates: %w", err)
	}

	config := m.Config()
	data := common.TemplateData(config)
	data["Strategy"] = opts.Strategy
	data["Roles"] = opts.Roles
	data["Hasher"] = opts.Hasher
	data["HasRefresh"] = opts.HasRefresh
	data["UserEntity"] = "users"

	switch m.Language {
	case "go":
		data["MiddlewareIdentifier"] = "RequireAuth"
		if m.Architecture == "flat" {
			data["AuthDir"] = "auth"
		} else {
			data["AuthDir"] = "internal/auth"
		}
		if mod := goModule(root); mod != "" {
			data["GoModule"] = mod
		}
	case "python":
		data["MiddlewareIdentifier"] = "get_current_user"
		if m.Architecture == "flat" || m.Architecture == "clean_arch" {
			data["AuthDir"] = "src/auth"
		} else {
			data["AuthDir"] = "src/app/auth"
		}
		data["AuthModule"] = strings.ReplaceAll(data["AuthDir"].(string), "/", ".")
	case "node":
		data["MiddlewareIdentifier"] = "requireAuth"
		data["AuthDir"] = "src/auth"
	default:
		return common.Result{}, fmt.Errorf("add auth não é suportado para a linguagem %q", m.Language)
	}

	patch, err := buildAuthPatch(root, m, templatesRoot, data, opts)
	if err != nil {
		return common.Result{}, err
	}

	result, err := common.Apply(root, templatesRoot, patch)
	if err != nil {
		return common.Result{}, err
	}

	updated := m
	if onDisk, readErr := manifest.Read(filepath.Join(root, manifest.FileName)); readErr == nil {
		updated = onDisk
	}

	updated.SetAuth(manifest.AuthSpec{
		Strategy:             opts.Strategy,
		UserEntity:           "users",
		Roles:                opts.Roles,
		HasRefresh:           opts.HasRefresh,
		Hasher:               opts.Hasher,
		MiddlewareIdentifier: data["MiddlewareIdentifier"].(string),
	})

	if err := common.RecordCreated(&updated, result.Created); err != nil {
		return common.Result{}, err
	}
	if err := manifest.Write(root, updated); err != nil {
		return common.Result{}, err
	}

	return result, nil
}

func buildAuthPatch(root string, m manifest.Manifest, templates fs.FS, data map[string]any, opts AuthOptions) (common.Patch, error) {
	languageRecipe := path.Join("langs", m.Language, "auth.crud.tmpl")
	if _, err := fs.Stat(templates, languageRecipe); err != nil {
		return common.Patch{}, fmt.Errorf("add auth não é suportado para a linguagem %q", m.Language)
	}

	patch := common.Patch{Data: data}
	for _, recipePath := range []string{"auth/auth.crud.tmpl", languageRecipe} {
		recipe, err := readDatabaseRecipe(templates, recipePath, data)
		if err != nil {
			return common.Patch{}, err
		}
		if err := recipe.addTo(&patch, root, m.Files, templates, data); err != nil {
			return common.Patch{}, fmt.Errorf("%s: %w", recipePath, err)
		}
	}

	// Migrações
	migrationDir := "auth/migrations/postgres"
	switch m.Database {
	case "mysql":
		migrationDir = "auth/migrations/mysql"
	case "mongo":
		migrationDir = "auth/migrations/mongo"
	}
	if _, err := fs.Stat(templates, migrationDir); err == nil {
		migFiles, err := migrationFiles(root, templates, migrationDir, "auth")
		if err != nil {
			return common.Patch{}, fmt.Errorf("migrationFiles %s: %w", migrationDir, err)
		}
		patch.Files = append(patch.Files, migFiles...)
	} else {
		return common.Patch{}, fmt.Errorf("fs.Stat %s: %w", migrationDir, err)
	}

	// Registro de rotas /auth nos arquivos do projeto
	if err := injectAuthRoutes(&patch, root, m, data); err != nil {
		return common.Patch{}, err
	}

	return patch, nil
}

func injectAuthRoutes(patch *common.Patch, root string, m manifest.Manifest, data map[string]any) error {
	authDir, _ := data["AuthDir"].(string)
	goMod, _ := data["GoModule"].(string)

	switch m.Language {
	case "go":
		routesFile := "internal/routes/routes.go"
		if m.Architecture == "flat" {
			routesFile = "routes.go"
		}
		if !exists(root, routesFile) {
			return nil
		}
		importLine := fmt.Sprintf(`	"%s/%s/handlers"`+"\n"+`	"%s/%s/middlewares"`+"\n"+`	"%s/%s/security"`, goMod, authDir, goMod, authDir, goMod, authDir)
		if m.Architecture == "flat" {
			importLine = ""
		}
		authSetupLines := []string{
			"\ttokenService := security.NewTokenService()",
			"\tauthMiddleware := middlewares.NewAuthMiddleware(tokenService)",
			"\tauthHandler := handlers.NewAuthHandler(tokenService, authMiddleware, dbs...)",
			"\tauthHandler.RegisterRoutes(api)",
		}
		if m.Framework == "chi" {
			authSetupLines[3] = "\tauthHandler.RegisterRoutes(r)"
		} else if m.Framework == "raw" {
			authSetupLines[3] = "\tauthHandler.RegisterRoutes(mux)"
		}

		return addRewrite(patch, root, routesFile, func(content []byte) ([]byte, error) {
			text := string(content)
			if strings.Contains(text, "authHandler") {
				return content, nil
			}
			if importLine != "" && !strings.Contains(text, goMod+"/"+authDir+"/handlers") {
				if idx := strings.Index(text, "import ("); idx != -1 {
					insertPos := idx + len("import (\n")
					text = text[:insertPos] + importLine + "\n" + text[insertPos:]
				}
			}
			retPos := strings.LastIndex(text, "return ")
			if retPos != -1 {
				insertion := strings.Join(authSetupLines, "\n") + "\n\n\t"
				text = text[:retPos] + insertion + text[retPos:]
			}
			return []byte(text), nil
		})

	case "python":
		mainFile := "src/app/main.py"
		if m.Architecture == "flat" {
			mainFile = "src/main.py"
		} else if m.Architecture == "clean_arch" {
			mainFile = "src/main.py"
		}
		if !exists(root, mainFile) {
			return nil
		}
		authModule := strings.ReplaceAll(authDir, "/", ".") + ".handlers.auth"
		importLine := fmt.Sprintf("from %s import router as auth_router", authModule)
		mountLine := "app.include_router(auth_router)"

		return addRewrite(patch, root, mainFile, func(content []byte) ([]byte, error) {
			text := string(content)
			if strings.Contains(text, "auth_router") {
				return content, nil
			}
			text = importLine + "\n" + text
			if idx := strings.Index(text, `if __name__ == "__main__":`); idx != -1 {
				text = text[:idx] + mountLine + "\n\n" + text[idx:]
			} else {
				text += "\n" + mountLine + "\n"
			}
			return []byte(text), nil
		})

	case "node":
		targetFile := "src/routes/index." + m.Runtime
		if !exists(root, targetFile) {
			targetFile = "src/main." + m.Runtime
		}
		if !exists(root, targetFile) {
			return nil
		}
		importLine := `import { authRouter } from "../auth/handlers/auth.js";`
		mountLine := `router.use("/auth", authRouter);`
		if strings.Contains(targetFile, "main") {
			importLine = `import { authRouter } from "./auth/handlers/auth.js";`
			mountLine = `app.use("/auth", authRouter);`
		}

		return addRewrite(patch, root, targetFile, func(content []byte) ([]byte, error) {
			text := string(content)
			if strings.Contains(text, "authRouter") {
				return content, nil
			}
			text = importLine + "\n" + text
			if retPos := strings.LastIndex(text, "export"); retPos != -1 {
				text = text[:retPos] + mountLine + "\n\n" + text[retPos:]
			} else {
				text += "\n" + mountLine + "\n"
			}
			return []byte(text), nil
		})
	}

	return nil
}
