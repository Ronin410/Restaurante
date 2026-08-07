// Package config centraliza la configuración de la aplicación, leída de
// variables de entorno con valores por defecto razonables para desarrollo.
//
// Los puertos por defecto se eligieron deliberadamente fuera de los
// "comunes" (5432 de Postgres, 8080 típico de backends, 5173/3000/80 de
// frontends) para que correr todo junto con Podman en una misma máquina no
// choque con otros servicios. Ver README/PODMAN.md.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port          string // puerto HTTP del backend
	DatabaseURL   string // DSN de conexión a PostgreSQL
	JWTSecret     string
	TokenTTL      time.Duration
	AllowedOrigin string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "9090"),
		DatabaseURL:   BuildDSN(),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-cambiar-en-produccion"),
		TokenTTL:      12 * time.Hour,
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
	}
}

// BuildDSN arma el DSN de PostgreSQL. Si DATABASE_URL está definida se usa
// tal cual (útil para proveedores administrados); si no, se construye a
// partir de PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE/PGSSLMODE, con
// valores por defecto pensados para el contenedor Postgres de este proyecto
// (ver podman/README.md), que escucha en el puerto 6543, no en el 5432
// estándar.
func BuildDSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	host := getEnv("PGHOST", "localhost")
	port := getEnv("PGPORT", "6543")
	user := getEnv("PGUSER", "taqueria")
	password := getEnv("PGPASSWORD", "taqueria")
	dbname := getEnv("PGDATABASE", "taqueria")
	sslmode := getEnv("PGSSLMODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
