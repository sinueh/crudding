package fields

import (
	"fmt"
	"strings"
)

// Dialect é o banco SQL de destino das migrações e das queries geradas.
type Dialect string

const (
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
)

// DialectFor devolve o dialeto SQL do banco do project.crud; ok é false para
// bancos sem SQL (mongo) ou projeto sem banco.
func DialectFor(database string) (Dialect, bool) {
	switch database {
	case "postgres":
		return Postgres, true
	case "mysql":
		return MySQL, true
	}
	return "", false
}

// Quote cerca o identificador com aspas do dialeto, para nomes como "order"
// ou "user" não colidirem com palavras reservadas do SQL.
func (d Dialect) Quote(identifier string) string {
	if d == MySQL {
		return "`" + identifier + "`"
	}
	return `"` + identifier + `"`
}

// ColumnType é o tipo da coluna de f no dialeto.
func (d Dialect) ColumnType(f Field) string {
	postgres := map[string]string{
		"str": "VARCHAR(255)", "text": "TEXT", "int": "INTEGER", "float": "DOUBLE PRECISION",
		"decimal": "NUMERIC(12, 2)", "bool": "BOOLEAN", "date": "DATE", "datetime": "TIMESTAMPTZ",
		"uuid": "UUID", "ref": "BIGINT",
	}
	mysql := map[string]string{
		"str": "VARCHAR(255)", "text": "TEXT", "int": "INT", "float": "DOUBLE",
		"decimal": "DECIMAL(12, 2)", "bool": "BOOLEAN", "date": "DATE", "datetime": "DATETIME(6)",
		"uuid": "CHAR(36)", "ref": "BIGINT",
	}
	if d == MySQL {
		return mysql[f.Type]
	}
	return postgres[f.Type]
}

// CreateTable é a migração .up.sql da tabela: id autoincremento, uma coluna
// por campo (NOT NULL quando obrigatório) e uma FOREIGN KEY por ref.
func CreateTable(d Dialect, table string, list []Field) string {
	q := d.Quote
	idType := "BIGSERIAL PRIMARY KEY"
	if d == MySQL {
		idType = "BIGINT AUTO_INCREMENT PRIMARY KEY"
	}
	lines := []string{fmt.Sprintf("%s %s", q("id"), idType)}
	var constraints, indexes []string
	for _, field := range list {
		line := fmt.Sprintf("%s %s", q(field.Column()), d.ColumnType(field))
		if !field.Nullable {
			line += " NOT NULL"
		}
		lines = append(lines, line)
		if ref := field.Ref(); ref != "" {
			name := "fk_" + table + "_" + field.Column()
			constraints = append(constraints, fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", q(name), q(field.Column()), q(ref), q("id")))
			if d == Postgres {
				// O MySQL indexa as chaves estrangeiras sozinho; o Postgres não.
				indexes = append(indexes, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);", q("idx_"+table+"_"+field.Column()), q(table), q(field.Column())))
			}
		}
	}
	lines = append(lines, constraints...)
	var b strings.Builder
	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n    %s\n)", q(table), strings.Join(lines, ",\n    "))
	if d == MySQL {
		b.WriteString(" ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	}
	b.WriteString(";\n")
	for _, index := range indexes {
		b.WriteString(index + "\n")
	}
	return b.String()
}

// DropTable é a migração .down.sql da tabela.
func DropTable(d Dialect, table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", d.Quote(table))
}

// Queries são as instruções SQL do CRUD de uma tabela, com os placeholders
// do driver: "%s" (pymysql e psycopg2), "$n" (lib/pq) ou "?" (go-sql-driver).
type Queries struct {
	Select    string // todas as linhas, ordenadas por id
	SelectOne string // uma linha por id
	Insert    string // com RETURNING id no postgres
	Update    string // todos os campos, com o id por último
	Delete    string
	Returning bool // Insert devolve o id (postgres); senão use lastrowid
}

// PythonQueries monta as queries com os placeholders "%s" de pymysql e psycopg2.
func PythonQueries(d Dialect, table string, list []Field) Queries {
	return buildQueries(d, table, list, func(int) string { return "%s" }, func(field Field) string { return d.Quote(field.Column()) })
}

// GoQueries monta as queries para database/sql: "$n" no lib/pq e "?" no
// go-sql-driver. Colunas date voltam como texto AAAA-MM-DD, para o Scan em
// string.
func GoQueries(d Dialect, table string, list []Field) Queries {
	placeholder := func(n int) string { return fmt.Sprintf("$%d", n) }
	if d == MySQL {
		placeholder = func(int) string { return "?" }
	}
	selectColumn := func(field Field) string {
		column := d.Quote(field.Column())
		if field.Type != "date" {
			return column
		}
		if d == MySQL {
			return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-%%d')", column)
		}
		return fmt.Sprintf("to_char(%s, 'YYYY-MM-DD')", column)
	}
	return buildQueries(d, table, list, placeholder, selectColumn)
}

func buildQueries(d Dialect, table string, list []Field, placeholder func(int) string, selectColumn func(Field) string) Queries {
	q := d.Quote
	selected := []string{q("id")}
	var columns, values, sets []string
	for index, field := range list {
		selected = append(selected, selectColumn(field))
		columns = append(columns, q(field.Column()))
		values = append(values, placeholder(index+1))
		sets = append(sets, fmt.Sprintf("%s = %s", q(field.Column()), placeholder(index+1)))
	}
	queries := Queries{
		Select:    fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", strings.Join(selected, ", "), q(table), q("id")),
		SelectOne: fmt.Sprintf("SELECT %s FROM %s WHERE %s = %s", strings.Join(selected, ", "), q(table), q("id"), placeholder(1)),
		Insert:    fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q(table), strings.Join(columns, ", "), strings.Join(values, ", ")),
		Update:    fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s", q(table), strings.Join(sets, ", "), q("id"), placeholder(len(list)+1)),
		Delete:    fmt.Sprintf("DELETE FROM %s WHERE %s = %s", q(table), q("id"), placeholder(1)),
	}
	if d == Postgres {
		queries.Insert += " RETURNING " + q("id")
		queries.Returning = true
	}
	return queries
}
