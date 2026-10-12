// Package app ensambla el router HTTP completo a partir de todos los
// módulos de dominio. Se separa de cmd/server para que las pruebas de
// integración puedan levantar el mismo servidor contra una base de datos
// de pruebas en memoria.
package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/ronin410/restaurante/backend/internal/accounting"
	"github.com/ronin410/restaurante/backend/internal/accounts"
	"github.com/ronin410/restaurante/backend/internal/audit"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/cashcuts"
	"github.com/ronin410/restaurante/backend/internal/catalog"
	"github.com/ronin410/restaurante/backend/internal/config"
	"github.com/ronin410/restaurante/backend/internal/dbx"
	"github.com/ronin410/restaurante/backend/internal/employees"
	"github.com/ronin410/restaurante/backend/internal/httpx"
	"github.com/ronin410/restaurante/backend/internal/kpis"
	"github.com/ronin410/restaurante/backend/internal/orders"
	"github.com/ronin410/restaurante/backend/internal/settings"
	"github.com/ronin410/restaurante/backend/internal/tables"
)

// New construye el router HTTP completo con todos los módulos registrados.
func New(db *dbx.DB, cfg config.Config) http.Handler {
	authMgr := authn.NewManager(cfg.JWTSecret, cfg.TokenTTL)
	auditLogger := audit.New(db)

	empHandler := employees.NewHandler(employees.NewService(employees.NewStore(db), authMgr), auditLogger)
	catalogHandler := catalog.NewHandler(catalog.NewService(catalog.NewStore(db)), auditLogger)
	tablesSvc := tables.NewService(tables.NewStore(db))
	tablesHandler := tables.NewHandler(tablesSvc, auditLogger)
	ordersHandler := orders.NewHandler(orders.NewService(orders.NewStore(db)), auditLogger)
	cashcutsStore := cashcuts.NewStore(db)
	cashcutsSvc := cashcuts.NewService(cashcutsStore)
	cashcutsHandler := cashcuts.NewHandler(cashcutsSvc, auditLogger)
	accountsHandler := accounts.NewHandler(accounts.NewService(accounts.NewStore(db), cashcutsSvc, tablesSvc), auditLogger)
	accountingHandler := accounting.NewHandler(accounting.NewService(accounting.NewStore(db)), auditLogger)
	kpisHandler := kpis.NewHandler(kpis.NewService(kpis.NewStore(db)))
	auditHandler := audit.NewHandler(auditLogger)
	settingsHandler := settings.NewHandler(db, auditLogger)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		empHandler.RegisterPublicRoutes(r)
		settingsHandler.RegisterPublicRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(authMgr.Middleware)
			empHandler.RegisterRoutes(r)
			catalogHandler.RegisterRoutes(r)
			tablesHandler.RegisterRoutes(r)
			ordersHandler.RegisterRoutes(r)
			accountsHandler.RegisterRoutes(r)
			cashcutsHandler.RegisterRoutes(r)
			accountingHandler.RegisterRoutes(r)
			kpisHandler.RegisterRoutes(r)
			auditHandler.RegisterRoutes(r)
			settingsHandler.RegisterRoutes(r)
		})
	})

	return r
}
