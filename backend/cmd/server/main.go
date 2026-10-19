// Command server arranca la API REST del sistema de administración para
// restaurante (POS + back office).
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ronin410/restaurante/backend/internal/app"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/config"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

func main() {
	cfg := config.Load()

	db, err := dbx.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("no se pudo iniciar la base de datos: %v", err)
	}
	defer db.Close()

	bootstrapAdmin(db)
	if cfg.JWTSecret == "dev-secret-cambiar-en-produccion" {
		log.Println("AVISO: JWT_SECRET usa el valor de desarrollo; define uno propio en producción")
	}

	router := app.New(db, cfg)

	addr := ":" + cfg.Port
	log.Printf("servidor escuchando en %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}

// bootstrapAdmin crea el primer administrador al arrancar, solo si la base no
// tiene usuarios y están definidas ADMIN_EMAIL y ADMIN_PASSWORD. Sirve para
// despliegues en la nube (ej. Render) donde no hay una terminal para correr
// el comando de datos de ejemplo. Después de entrar, cambia la contraseña
// desde Personal y puedes quitar ADMIN_PASSWORD de las variables.
func bootstrapAdmin(db *dbx.DB) {
	email, password := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")
	if email == "" || password == "" {
		return
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&count); err != nil || count > 0 {
		return
	}
	if len(password) < 6 {
		log.Println("ADMIN_PASSWORD debe tener al menos 6 caracteres; no se creó el administrador")
		return
	}
	name := os.Getenv("ADMIN_NAME")
	if name == "" {
		name = "Administrador"
	}
	hash, err := authn.HashPassword(password)
	if err != nil {
		log.Printf("no se pudo cifrar la contraseña del administrador: %v", err)
		return
	}
	_, err = db.Exec(
		`INSERT INTO users (name, email, password_hash, role, position, salary, hire_date, active) VALUES (?, ?, ?, 'admin', 'Administrador', 0, date('now'), 1)`,
		name, email, hash,
	)
	if err != nil {
		log.Printf("no se pudo crear el administrador inicial: %v", err)
		return
	}
	log.Printf("administrador inicial creado: %s", email)
}
