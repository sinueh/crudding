package generator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinueh-andreata/crudding/internal/generator/common"
	"github.com/sinueh-andreata/crudding/internal/manifest"
	"github.com/sinueh-andreata/crudding/internal/project"
)

// SupportedDatabases lista os bancos de dados suportados pela CLI.
var SupportedDatabases = []string{"postgres", "mysql", "mongo"}

// ValidateDatabase verifica se o banco informado é suportado.
func ValidateDatabase(db string) error {
	if !slices.Contains(SupportedDatabases, db) {
		return fmt.Errorf("banco de dados não suportado: %q (opções: %s)", db, strings.Join(SupportedDatabases, ", "))
	}
	return nil
}

type dbTemplateData struct {
	common.BaseData
	ProjectSlug string
	Database    string
	Package     string
	Runtime     string
}

// AddDatabase adiciona a infraestrutura do banco de dados escolhido ao projeto.
func AddDatabase(root string, m manifest.Manifest, db string, templates fs.FS) (common.Result, error) {
	if err := ValidateDatabase(db); err != nil {
		return common.Result{}, err
	}
	if m.Architecture == manifest.CustomArchitecture {
		return common.Result{}, fmt.Errorf("add db ainda não é suportado para projetos de arquitetura %s", manifest.CustomArchitecture)
	}

	if m.HasDatabase(db) {
		return common.Result{}, fmt.Errorf("o projeto já possui o banco de dados %q configurado", db)
	}

	patch, err := buildDatabasePatch(root, m, db)
	if err != nil {
		return common.Result{}, err
	}

	result, err := common.Apply(root, templates, patch)
	if err != nil {
		return common.Result{}, err
	}

	if err := updateProjectDependencies(root, m, db, &result); err != nil {
		return common.Result{}, err
	}

	updated := m
	if onDisk, readErr := manifest.Read(filepath.Join(root, manifest.FileName)); readErr == nil {
		updated = onDisk
	}
	updated.AddDatabase(db)
	if err := common.RecordCreated(&updated, result.Created); err != nil {
		return common.Result{}, err
	}
	if err := manifest.Write(root, updated); err != nil {
		return common.Result{}, err
	}

	return result, nil
}

func buildDatabasePatch(root string, m manifest.Manifest, db string) (common.Patch, error) {
	var files []common.File
	var inserts []common.Insert
	var rewrites []common.Rewrite

	// 1. docker-compose.yml
	dockerComposePath := filepath.Join(root, "docker-compose.yml")
	if _, err := os.Stat(dockerComposePath); os.IsNotExist(err) {
		files = append(files, common.File{
			TemplatePath: fmt.Sprintf("templates/databases/%s/docker-compose.yml.tmpl", db),
			OutputPath:   "docker-compose.yml",
		})
	} else {
		serviceLines, volumeLines := composeInserts(db, project.Slug(m.Name))
		inserts = append(inserts,
			common.Insert{
				Path:   "docker-compose.yml",
				Marker: "# crudding:compose-services",
				Lines:  serviceLines,
			},
			common.Insert{
				Path:   "docker-compose.yml",
				Marker: "# crudding:compose-volumes",
				Lines:  volumeLines,
			},
		)
	}

	// 2. .env.example
	envPath := filepath.Join(root, ".env.example")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		files = append(files, common.File{
			TemplatePath: fmt.Sprintf("templates/databases/%s/.env.example.tmpl", db),
			OutputPath:   ".env.example",
		})
	} else {
		envLines := envInserts(db)
		inserts = append(inserts, common.Insert{
			Path:   ".env.example",
			Marker: "# crudding:env",
			Lines:  envLines,
		})
	}

	// 3. migrations/
	switch db {
	case "postgres", "mysql":
		upFile := "migrations/000001_init.up.sql"
		downFile := "migrations/000001_init.down.sql"
		if _, err := os.Stat(filepath.Join(root, upFile)); err == nil {
			upFile = fmt.Sprintf("migrations/000002_%s_init.up.sql", db)
			downFile = fmt.Sprintf("migrations/000002_%s_init.down.sql", db)
		}
		files = append(files,
			common.File{
				TemplatePath: fmt.Sprintf("templates/databases/%s/migrations/000001_init.up.sql.tmpl", db),
				OutputPath:   upFile,
			},
			common.File{
				TemplatePath: fmt.Sprintf("templates/databases/%s/migrations/000001_init.down.sql.tmpl", db),
				OutputPath:   downFile,
			},
		)
	case "mongo":
		jsFile := "migrations/000001_init.js"
		if _, err := os.Stat(filepath.Join(root, jsFile)); err == nil {
			jsFile = "migrations/000002_mongo_init.js"
		}
		files = append(files, common.File{
			TemplatePath: "templates/databases/mongo/migrations/000001_init.js.tmpl",
			OutputPath:   jsFile,
		})
	}

	// 4. Arquivo de conexão / driver específico da linguagem
	pkg := "config"
	switch m.Language {
	case "go":
		configPath := "internal/config/config.go"
		if m.Architecture == "flat" {
			configPath = "config/config.go"
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(configPath))); os.IsNotExist(err) {
			files = append(files, common.File{
				TemplatePath: "templates/langs/go/architectures/common/config.go.tmpl",
				OutputPath:   configPath,
			})
		} else {
			goInserts := goDatabaseInserts(configPath, db)
			inserts = append(inserts, goInserts...)
		}
	case "python":
		configPath := "src/app/config/config.py"
		if m.Architecture == "flat" {
			configPath = "src/config/config.py"
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(configPath))); os.IsNotExist(err) {
			files = append(files, common.File{
				TemplatePath: "templates/langs/python/architectures/common/config.py.tmpl",
				OutputPath:   configPath,
			})
		} else {
			pyInserts := pythonDatabaseInserts(configPath, db, m.ORM)
			inserts = append(inserts, pyInserts...)
		}
		if m.ORM == "sqlalchemy" {
			modelPath := pythonModelPath(m.Architecture)
			if modelPath != "" {
				rewrites = append(rewrites, common.Rewrite{
					Path: modelPath,
					Edit: func([]byte) ([]byte, error) {
						cfgImport := "from app.config.config import Base"
						if m.Architecture == "flat" {
							cfgImport = "from config.config import Base"
						}
						newModel := fmt.Sprintf("from sqlalchemy import Column, String\n%s\n\n\nclass Resource(Base):\n    __tablename__ = \"resources\"\n\n    id = Column(String, primary_key=True, index=True)\n    # TODO: adicione os campos de Resource.\n", cfgImport)
						return []byte(newModel), nil
					},
				})
			}
		}
	case "node":
		ext := ".js"
		if m.Runtime == "ts" {
			ext = ".ts"
		}
		files = append(files, common.File{
			TemplatePath: fmt.Sprintf("templates/databases/%s/node/db%s.tmpl", db, ext),
			OutputPath:   path.Join("src", "database"+ext),
		})
	default:
		return common.Patch{}, fmt.Errorf("linguagem não suportada para banco de dados: %s", m.Language)
	}

	data := dbTemplateData{
		BaseData:    common.NewBaseData(m.Config()),
		ProjectSlug: project.Slug(m.Name),
		Database:    db,
		Package:     pkg,
		Runtime:     m.Runtime,
	}

	return common.Patch{
		Files:    files,
		Inserts:  inserts,
		Rewrites: rewrites,
		Data:     data,
	}, nil
}

func goDatabaseInserts(configPath, db string) []common.Insert {
	var imports []string
	var structFields []string
	var loadFields []string
	var methods []string

	switch db {
	case "postgres":
		imports = []string{
			"\t\"database/sql\"",
			"\t\"fmt\"",
			"",
			"\t_ \"github.com/lib/pq\"",
		}
		structFields = []string{"\tPostgres PostgresConfig"}
		loadFields = []string{"\t\tPostgres: loadPostgresConfig(),"}
		methods = []string{
			"type PostgresConfig struct {",
			"\tURL string",
			"}",
			"",
			"func loadPostgresConfig() PostgresConfig {",
			"\tdsn := os.Getenv(\"DATABASE_URL\")",
			"\tif dsn == \"\" {",
			"\t\tdsn = fmt.Sprintf(",
			"\t\t\t\"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable\",",
			"\t\t\tgetEnv(\"DB_HOST\", \"localhost\"),",
			"\t\t\tgetEnv(\"DB_PORT\", \"5432\"),",
			"\t\t\tgetEnv(\"DB_USER\", \"postgres\"),",
			"\t\t\tgetEnv(\"DB_PASSWORD\", \"postgres\"),",
			"\t\t\tgetEnv(\"DB_NAME\", \"app\"),",
			"\t\t)",
			"\t}",
			"\treturn PostgresConfig{URL: dsn}",
			"}",
			"",
			"// ConnectPostgres estabelece e valida a conexão com o PostgreSQL.",
			"func ConnectPostgres(cfg PostgresConfig) (*sql.DB, error) {",
			"\tdb, err := sql.Open(\"postgres\", cfg.URL)",
			"\tif err != nil {",
			"\t\treturn nil, fmt.Errorf(\"abrir conexão com postgres: %w\", err)",
			"\t}",
			"\tif err := db.Ping(); err != nil {",
			"\t\treturn nil, fmt.Errorf(\"ping no postgres: %w\", err)",
			"\t}",
			"\treturn db, nil",
			"}",
			"",
		}
	case "mysql":
		imports = []string{
			"\t\"database/sql\"",
			"\t\"fmt\"",
			"",
			"\t_ \"github.com/go-sql-driver/mysql\"",
		}
		structFields = []string{"\tMySQL MySQLConfig"}
		loadFields = []string{"\t\tMySQL: loadMySQLConfig(),"}
		methods = []string{
			"type MySQLConfig struct {",
			"\tDSN string",
			"}",
			"",
			"func loadMySQLConfig() MySQLConfig {",
			"\tdsn := os.Getenv(\"DATABASE_URL\")",
			"\tif dsn == \"\" {",
			"\t\tdsn = fmt.Sprintf(",
			"\t\t\t\"%s:%s@tcp(%s:%s)/%s?parseTime=true\",",
			"\t\t\tgetEnv(\"DB_USER\", \"user\"),",
			"\t\t\tgetEnv(\"DB_PASSWORD\", \"password\"),",
			"\t\t\tgetEnv(\"DB_HOST\", \"localhost\"),",
			"\t\t\tgetEnv(\"DB_PORT\", \"3306\"),",
			"\t\t\tgetEnv(\"DB_NAME\", \"app\"),",
			"\t\t)",
			"\t}",
			"\treturn MySQLConfig{DSN: dsn}",
			"}",
			"",
			"// ConnectMySQL estabelece e valida a conexão com o MySQL.",
			"func ConnectMySQL(cfg MySQLConfig) (*sql.DB, error) {",
			"\tdb, err := sql.Open(\"mysql\", cfg.DSN)",
			"\tif err != nil {",
			"\t\treturn nil, fmt.Errorf(\"abrir conexão com mysql: %w\", err)",
			"\t}",
			"\tif err := db.Ping(); err != nil {",
			"\t\treturn nil, fmt.Errorf(\"ping no mysql: %w\", err)",
			"\t}",
			"\treturn db, nil",
			"}",
			"",
		}
	case "mongo":
		imports = []string{
			"\t\"context\"",
			"\t\"fmt\"",
			"\t\"time\"",
			"",
			"\t\"go.mongodb.org/mongo-driver/mongo\"",
			"\t\"go.mongodb.org/mongo-driver/mongo/options\"",
		}
		structFields = []string{"\tMongo MongoConfig"}
		loadFields = []string{"\t\tMongo: loadMongoConfig(),"}
		methods = []string{
			"type MongoConfig struct {",
			"\tURI    string",
			"\tDBName string",
			"}",
			"",
			"func loadMongoConfig() MongoConfig {",
			"\turi := os.Getenv(\"MONGODB_URI\")",
			"\tif uri == \"\" {",
			"\t\turi = fmt.Sprintf(",
			"\t\t\t\"mongodb://%s:%s@%s:%s/%s?authSource=admin\",",
			"\t\t\tgetEnv(\"DB_USER\", \"root\"),",
			"\t\t\tgetEnv(\"DB_PASSWORD\", \"rootpassword\"),",
			"\t\t\tgetEnv(\"DB_HOST\", \"localhost\"),",
			"\t\t\tgetEnv(\"DB_PORT\", \"27017\"),",
			"\t\t\tgetEnv(\"DB_NAME\", \"app\"),",
			"\t\t)",
			"\t}",
			"\treturn MongoConfig{",
			"\t\tURI:    uri,",
			"\t\tDBName: getEnv(\"DB_NAME\", \"app\"),",
			"\t}",
			"}",
			"",
			"// ConnectMongo estabelece e valida a conexão com o MongoDB.",
			"func ConnectMongo(ctx context.Context, cfg MongoConfig) (*mongo.Client, error) {",
			"\tclient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))",
			"\tif err != nil {",
			"\t\treturn nil, fmt.Errorf(\"abrir conexão com mongo: %w\", err)",
			"\t}",
			"\tpingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)",
			"\tdefer cancel()",
			"\tif err := client.Ping(pingCtx, nil); err != nil {",
			"\t\treturn nil, fmt.Errorf(\"ping no mongo: %w\", err)",
			"\t}",
			"\treturn client, nil",
			"}",
			"",
		}
	}

	return []common.Insert{
		{Path: configPath, Marker: "// crudding:db-imports", Lines: imports},
		{Path: configPath, Marker: "// crudding:db-struct", Lines: structFields},
		{Path: configPath, Marker: "// crudding:db-load", Lines: loadFields},
		{Path: configPath, Marker: "// crudding:db-methods", Lines: methods},
	}
}

func pythonModelPath(arch string) string {
	switch arch {
	case "flat":
		return "src/resource_model.py"
	case "layered":
		return "src/app/models/resource.py"
	case "modular":
		return "src/app/resource/models.py"
	case "hexagonal":
		return "src/app/domain/resource.py"
	case "clean_arch":
		return "src/app/domain/model/resource.py"
	}
	return ""
}

func pythonDatabaseInserts(configPath, db, orm string) []common.Insert {
	var imports []string
	var configLines []string
	var methods []string

	if orm == "sqlalchemy" {
		imports = []string{
			"from sqlalchemy import create_engine",
			"from sqlalchemy.orm import declarative_base, sessionmaker",
		}
		switch db {
		case "postgres":
			configLines = []string{
				"    DATABASE_URL: str = os.getenv(",
				"        \"DATABASE_URL\",",
				"        f\"postgresql://{os.getenv('DB_USER', 'postgres')}:{os.getenv('DB_PASSWORD', 'postgres')}@\"",
				"        f\"{os.getenv('DB_HOST', 'localhost')}:{os.getenv('DB_PORT', '5432')}/{os.getenv('DB_NAME', 'app')}\",",
				"    )",
			}
			methods = []string{
				"",
				"Base = declarative_base()",
				"",
				"def get_sqlalchemy_engine():",
				"    \"\"\"Retorna a engine do SQLAlchemy para PostgreSQL.\"\"\"",
				"    database_url = os.getenv(\"DATABASE_URL\")",
				"    if not database_url:",
				"        database_url = (",
				"            f\"postgresql+psycopg2://{os.getenv('DB_USER', 'postgres')}:{os.getenv('DB_PASSWORD', 'postgres')}@\"",
				"            f\"{os.getenv('DB_HOST', 'localhost')}:{os.getenv('DB_PORT', '5432')}/{os.getenv('DB_NAME', 'app')}\"",
				"        )",
				"    return create_engine(database_url)",
				"",
				"engine = get_sqlalchemy_engine()",
				"SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)",
				"",
				"def get_db():",
				"    \"\"\"Generator de sessão do SQLAlchemy.\"\"\"",
				"    db = SessionLocal()",
				"    try:",
				"        yield db",
				"    finally:",
				"        db.close()",
				"",
				"def init_db():",
				"    \"\"\"Cria as tabelas no banco de dados se não existirem.\"\"\"",
				"    Base.metadata.create_all(bind=engine)",
			}
		case "mysql":
			configLines = []string{
				"    MYSQL_HOST: str = os.getenv(\"DB_HOST\", \"localhost\")",
				"    MYSQL_PORT: int = int(os.getenv(\"DB_PORT\", \"3306\"))",
				"    MYSQL_USER: str = os.getenv(\"DB_USER\", \"user\")",
				"    MYSQL_PASSWORD: str = os.getenv(\"DB_PASSWORD\", \"password\")",
				"    MYSQL_DB: str = os.getenv(\"DB_NAME\", \"app\")",
			}
			methods = []string{
				"",
				"Base = declarative_base()",
				"",
				"def get_sqlalchemy_engine():",
				"    \"\"\"Retorna a engine do SQLAlchemy para MySQL.\"\"\"",
				"    database_url = os.getenv(\"DATABASE_URL\")",
				"    if not database_url:",
				"        database_url = (",
				"            f\"mysql+pymysql://{config.MYSQL_USER}:{config.MYSQL_PASSWORD}@\"",
				"            f\"{config.MYSQL_HOST}:{config.MYSQL_PORT}/{config.MYSQL_DB}\"",
				"        )",
				"    return create_engine(database_url)",
				"",
				"engine = get_sqlalchemy_engine()",
				"SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)",
				"",
				"def get_db():",
				"    \"\"\"Generator de sessão do SQLAlchemy.\"\"\"",
				"    db = SessionLocal()",
				"    try:",
				"        yield db",
				"    finally:",
				"        db.close()",
				"",
				"def init_db():",
				"    \"\"\"Cria as tabelas no banco de dados se não existirem.\"\"\"",
				"    Base.metadata.create_all(bind=engine)",
			}
		}

		return []common.Insert{
			{Path: configPath, Marker: "# crudding:db-imports", Lines: imports},
			{Path: configPath, Marker: "# crudding:db-config", Lines: configLines},
			{Path: configPath, Marker: "# crudding:db-methods", Lines: methods},
		}
	}

	switch db {
	case "postgres":
		imports = []string{
			"import psycopg2",
			"from psycopg2.extras import RealDictCursor",
		}
		configLines = []string{
			"    DATABASE_URL: str = os.getenv(",
			"        \"DATABASE_URL\",",
			"        f\"postgresql://{os.getenv('DB_USER', 'postgres')}:{os.getenv('DB_PASSWORD', 'postgres')}@\"",
			"        f\"{os.getenv('DB_HOST', 'localhost')}:{os.getenv('DB_PORT', '5432')}/{os.getenv('DB_NAME', 'app')}\",",
			"    )",
		}
		methods = []string{
			"",
			"def get_postgres_connection():",
			"    \"\"\"Retorna uma conexão ativa com o banco PostgreSQL.\"\"\"",
			"    database_url = os.getenv(\"DATABASE_URL\")",
			"    if database_url:",
			"        return psycopg2.connect(database_url, cursor_factory=RealDictCursor)",
			"",
			"    return psycopg2.connect(",
			"        host=os.getenv(\"DB_HOST\", \"localhost\"),",
			"        port=os.getenv(\"DB_PORT\", \"5432\"),",
			"        user=os.getenv(\"DB_USER\", \"postgres\"),",
			"        password=os.getenv(\"DB_PASSWORD\", \"postgres\"),",
			"        dbname=os.getenv(\"DB_NAME\", \"app\"),",
			"        cursor_factory=RealDictCursor,",
			"    )",
			"",
			"get_connection = get_postgres_connection",
		}
	case "mysql":
		imports = []string{
			"import pymysql",
			"from pymysql.cursors import DictCursor",
		}
		configLines = []string{
			"    MYSQL_HOST: str = os.getenv(\"DB_HOST\", \"localhost\")",
			"    MYSQL_PORT: int = int(os.getenv(\"DB_PORT\", \"3306\"))",
			"    MYSQL_USER: str = os.getenv(\"DB_USER\", \"user\")",
			"    MYSQL_PASSWORD: str = os.getenv(\"DB_PASSWORD\", \"password\")",
			"    MYSQL_DB: str = os.getenv(\"DB_NAME\", \"app\")",
		}
		methods = []string{
			"",
			"def get_mysql_connection():",
			"    \"\"\"Retorna uma conexão ativa com o banco MySQL.\"\"\"",
			"    return pymysql.connect(",
			"        host=config.MYSQL_HOST,",
			"        port=config.MYSQL_PORT,",
			"        user=config.MYSQL_USER,",
			"        password=config.MYSQL_PASSWORD,",
			"        database=config.MYSQL_DB,",
			"        cursorclass=DictCursor,",
			"    )",
			"",
			"get_connection = get_mysql_connection",
		}
	case "mongo":
		imports = []string{
			"from pymongo import MongoClient",
		}
		configLines = []string{
			"    MONGODB_URI: str = os.getenv(",
			"        \"MONGODB_URI\",",
			"        f\"mongodb://{os.getenv('DB_USER', 'root')}:{os.getenv('DB_PASSWORD', 'rootpassword')}@\"",
			"        f\"{os.getenv('DB_HOST', 'localhost')}:{os.getenv('DB_PORT', '27017')}/\"",
			"        f\"{os.getenv('DB_NAME', 'app')}?authSource=admin\",",
			"    )",
			"    MONGODB_DB_NAME: str = os.getenv(\"DB_NAME\", \"app\")",
		}
		methods = []string{
			"",
			"def get_mongo_database():",
			"    \"\"\"Retorna a instância do banco de dados MongoDB configurado.\"\"\"",
			"    uri = os.getenv(",
			"        \"MONGODB_URI\",",
			"        f\"mongodb://{os.getenv('DB_USER', 'root')}:{os.getenv('DB_PASSWORD', 'rootpassword')}@\"",
			"        f\"{os.getenv('DB_HOST', 'localhost')}:{os.getenv('DB_PORT', '27017')}/\"",
			"        f\"{os.getenv('DB_NAME', 'app')}?authSource=admin\",",
			"    )",
			"    client = MongoClient(uri)",
			"    db_name = os.getenv(\"DB_NAME\", \"app\")",
			"    return client[db_name]",
			"",
			"get_database = get_mongo_database",
		}
	}

	return []common.Insert{
		{Path: configPath, Marker: "# crudding:db-imports", Lines: imports},
		{Path: configPath, Marker: "# crudding:db-config", Lines: configLines},
		{Path: configPath, Marker: "# crudding:db-methods", Lines: methods},
	}
}

func composeInserts(db, projectSlug string) (serviceLines, volumeLines []string) {
	switch db {
	case "postgres":
		serviceLines = []string{
			"  postgres:",
			"    image: postgres:16-alpine",
			fmt.Sprintf("    container_name: %s-postgres", projectSlug),
			"    restart: unless-stopped",
			"    environment:",
			"      POSTGRES_USER: ${DB_USER:-postgres}",
			"      POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}",
			"      POSTGRES_DB: ${DB_NAME:-app}",
			"    ports:",
			"      - \"${DB_PORT:-5432}:5432\"",
			"    volumes:",
			"      - postgres_data:/var/lib/postgresql/data",
		}
		volumeLines = []string{
			"  postgres_data:",
		}
	case "mysql":
		serviceLines = []string{
			"  mysql:",
			"    image: mysql:8.4",
			fmt.Sprintf("    container_name: %s-mysql", projectSlug),
			"    restart: unless-stopped",
			"    environment:",
			"      MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD:-rootpassword}",
			"      MYSQL_DATABASE: ${DB_NAME:-app}",
			"      MYSQL_USER: ${DB_USER:-user}",
			"      MYSQL_PASSWORD: ${DB_PASSWORD:-password}",
			"    ports:",
			"      - \"${DB_PORT:-3306}:3306\"",
			"    volumes:",
			"      - mysql_data:/var/lib/mysql",
		}
		volumeLines = []string{
			"  mysql_data:",
		}
	case "mongo":
		serviceLines = []string{
			"  mongo:",
			"    image: mongo:7.0",
			fmt.Sprintf("    container_name: %s-mongo", projectSlug),
			"    restart: unless-stopped",
			"    environment:",
			"      MONGO_INITDB_ROOT_USERNAME: ${DB_USER:-root}",
			"      MONGO_INITDB_ROOT_PASSWORD: ${DB_PASSWORD:-rootpassword}",
			"    ports:",
			"      - \"${DB_PORT:-27017}:27017\"",
			"    volumes:",
			"      - mongo_data:/data/db",
		}
		volumeLines = []string{
			"  mongo_data:",
		}
	}
	return serviceLines, volumeLines
}

func envInserts(db string) []string {
	switch db {
	case "postgres":
		return []string{
			"DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable",
			"DB_HOST=localhost",
			"DB_PORT=5432",
			"DB_USER=postgres",
			"DB_PASSWORD=postgres",
			"DB_NAME=app",
		}
	case "mysql":
		return []string{
			"DATABASE_URL=mysql://user:password@localhost:3306/app",
			"DB_HOST=localhost",
			"DB_PORT=3306",
			"DB_USER=user",
			"DB_PASSWORD=password",
			"DB_NAME=app",
			"DB_ROOT_PASSWORD=rootpassword",
		}
	case "mongo":
		return []string{
			"MONGODB_URI=mongodb://root:rootpassword@localhost:27017/app?authSource=admin",
			"DB_HOST=localhost",
			"DB_PORT=27017",
			"DB_USER=root",
			"DB_PASSWORD=rootpassword",
			"DB_NAME=app",
		}
	}
	return nil
}

func updateProjectDependencies(root string, m manifest.Manifest, db string, result *common.Result) error {
	switch m.Language {
	case "go":
		return updateGoDependencies(root, db, result)
	case "python":
		return updatePythonDependencies(root, m, db, result)
	case "node":
		return updateNodeDependencies(root, m.Runtime, db, result)
	}
	return nil
}

func updateGoDependencies(root string, db string, result *common.Result) error {
	goModPath := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("ler go.mod: %w", err)
	}

	var pkg string
	switch db {
	case "postgres":
		pkg = "github.com/lib/pq v1.10.9"
	case "mysql":
		pkg = "github.com/go-sql-driver/mysql v1.8.1"
	case "mongo":
		pkg = "go.mongodb.org/mongo-driver v1.17.1"
	}

	text := string(content)
	moduleName := strings.Split(pkg, " ")[0]
	if strings.Contains(text, moduleName) {
		return nil
	}

	requireLine := fmt.Sprintf("\nrequire %s\n", pkg)
	text = strings.TrimRight(text, "\n") + "\n" + requireLine

	if err := os.WriteFile(goModPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("atualizar go.mod: %w", err)
	}

	if !slices.Contains(result.Modified, "go.mod") {
		result.Modified = append(result.Modified, "go.mod")
	}
	return nil
}

func updatePythonDependencies(root string, m manifest.Manifest, db string, result *common.Result) error {
	reqPath := filepath.Join(root, "requirements.txt")
	content, err := os.ReadFile(reqPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("ler requirements.txt: %w", err)
	}

	var pkg string
	switch db {
	case "postgres":
		pkg = "psycopg2-binary>=2.9.9"
	case "mysql":
		pkg = "pymysql>=1.1.0"
	case "mongo":
		pkg = "pymongo>=4.6.0"
	}

	text := string(content)
	modified := false
	if !strings.Contains(text, pkg) && !strings.Contains(text, strings.Split(pkg, ">=")[0]) {
		if !strings.HasSuffix(text, "\n") && len(text) > 0 {
			text += "\n"
		}
		text += pkg + "\n"
		modified = true
	}

	if m.ORM == "sqlalchemy" && !strings.Contains(text, "SQLAlchemy") {
		if !strings.HasSuffix(text, "\n") && len(text) > 0 {
			text += "\n"
		}
		text += "SQLAlchemy>=2.0.0\n"
		modified = true
	}

	if !modified {
		return nil
	}

	if err := os.WriteFile(reqPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("atualizar requirements.txt: %w", err)
	}

	if !slices.Contains(result.Modified, "requirements.txt") {
		result.Modified = append(result.Modified, "requirements.txt")
	}
	return nil
}

func updateNodeDependencies(root string, runtime string, db string, result *common.Result) error {
	pkgPath := filepath.Join(root, "package.json")
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("ler package.json: %w", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(content, &parsed); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	deps, _ := parsed["dependencies"].(map[string]any)
	if deps == nil {
		deps = make(map[string]any)
		parsed["dependencies"] = deps
	}

	devDeps, _ := parsed["devDependencies"].(map[string]any)
	if devDeps == nil {
		devDeps = make(map[string]any)
		parsed["devDependencies"] = devDeps
	}

	switch db {
	case "postgres":
		deps["pg"] = "^8.11.3"
		if runtime == "ts" {
			devDeps["@types/pg"] = "^8.11.0"
		}
	case "mysql":
		deps["mysql2"] = "^3.9.2"
	case "mongo":
		deps["mongodb"] = "^6.3.0"
	}

	updatedContent, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return fmt.Errorf("codificar package.json: %w", err)
	}

	if err := os.WriteFile(pkgPath, append(updatedContent, '\n'), 0644); err != nil {
		return fmt.Errorf("gravar package.json: %w", err)
	}

	if !slices.Contains(result.Modified, "package.json") {
		result.Modified = append(result.Modified, "package.json")
	}
	return nil
}
