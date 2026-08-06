// Package dbx maneja la conexión a la base de datos y la ejecución de
// migraciones. Usa SQLite (driver puro Go, sin cgo) para que el sistema
// corra en cualquier lugar sin dependencias externas; el acceso a datos del
// resto de la aplicación pasa por SQL estándar para facilitar, a futuro,
// migrar a PostgreSQL cuando se requiera escalar a varias sucursales.
package dbx

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed all:migrations
var migrationsFS embed.FS

// Open abre (creando si es necesario) la base de datos en dbPath y aplica
// las migraciones pendientes.
func Open(dbPath string) (*sql.DB, error) {
	if dbPath != ":memory:" {
		if dir := filepath.Dir(dbPath); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("creando directorio de datos: %w", err)
			}
		}
	}

	// WAL permite lecturas concurrentes mientras hay una escritura en curso
	// y busy_timeout hace que SQLite reintente en vez de fallar de inmediato
	// con "database is locked" si dos peticiones coinciden.
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("abriendo base de datos: %w", err)
	}
	// SQLite solo soporta un escritor a la vez. Además de limitar el pool,
	// el código de acceso a datos debe evitar anidar una consulta dentro
	// del bucle de otra (agotar/cerrar el cursor externo antes de abrir uno
	// nuevo), o una sola conexión puede quedar interbloqueada consigo misma.
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
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
		if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE name = ?`, name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("leyendo migración %s: %w", name, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("aplicando migración %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
