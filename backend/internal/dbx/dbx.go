// Package dbx maneja la conexión a PostgreSQL y la ejecución de
// migraciones.
//
// El resto de la aplicación (los store.go de cada módulo) escribe sus
// consultas con placeholders al estilo SQLite/MySQL ("?"), heredados de
// cuando este proyecto usaba SQLite embebido. En vez de reescribir las más
// de 200 consultas a la sintaxis "$1, $2, ..." que exige el driver de
// Postgres, DB.rebind() traduce los placeholders de forma transparente, así
// que el resto del código no cambió al migrar de motor.
package dbx

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed all:migrations
var migrationsFS embed.FS

// DB envuelve *sql.DB reescribiendo placeholders "?" a "$1, $2, ...". Los
// demás métodos de *sql.DB (Close, Ping, Begin, etc.) quedan disponibles tal
// cual por estar embebido.
type DB struct {
	*sql.DB
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(rebind(query), args...)
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(rebind(query), args...)
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRow(rebind(query), args...)
}

// rebind convierte cada "?" en "$1", "$2", etc. Ninguna de las consultas de
// este proyecto usa "?" dentro de un literal de texto, así que una
// sustitución posicional simple es suficiente (no hace falta un parser SQL).
func rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Open conecta a Postgres usando dsn (ver config.BuildDSN) y aplica las
// migraciones pendientes.
func Open(dsn string) (*DB, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("abriendo conexión a la base de datos: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("no se pudo conectar a la base de datos: %w", err)
	}

	db := &DB{DB: sqlDB}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *DB) error {
	if _, err := db.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("creando tabla de migraciones: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("leyendo migraciones: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var exists int
		if err := db.DB.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE name = $1`, name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("leyendo migración %s: %w", name, err)
		}
		tx, err := db.DB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("aplicando migración %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
