// Command server arranca la API REST del sistema de administración para
// taquería (POS + back office).
package main

import (
	"log"
	"net/http"

	"github.com/ronin410/restaurante/backend/internal/app"
	"github.com/ronin410/restaurante/backend/internal/config"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

func main() {
	cfg := config.Load()

	db, err := dbx.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("no se pudo iniciar la base de datos: %v", err)
	}
	defer db.Close()

	router := app.New(db, cfg)

	addr := ":" + cfg.Port
	log.Printf("servidor escuchando en %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
