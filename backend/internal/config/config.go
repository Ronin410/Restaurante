// Package config centraliza la configuración de la aplicación, leída de
// variables de entorno con valores por defecto razonables para desarrollo.
package config

import (
	"os"
	"time"
)

type Config struct {
	Port          string
	DBPath        string
	JWTSecret     string
	TokenTTL      time.Duration
	AllowedOrigin string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		DBPath:        getEnv("DB_PATH", "./data/taqueria.db"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-cambiar-en-produccion"),
		TokenTTL:      12 * time.Hour,
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
