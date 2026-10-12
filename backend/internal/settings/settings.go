// Package settings guarda la configuración general del negocio (nombre e
// ícono que se muestran en la app), para que el sistema sirva a cualquier
// tipo de restaurante sin tocar código.
package settings

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ronin410/restaurante/backend/internal/audit"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/dbx"
	"github.com/ronin410/restaurante/backend/internal/httpx"
)

type Settings struct {
	BusinessName string `json:"business_name"`
	BusinessIcon string `json:"business_icon"`
}

var errValidation = errors.New("datos inválidos")

type Handler struct {
	db    *dbx.DB
	audit *audit.Logger
}

func NewHandler(db *dbx.DB, audit *audit.Logger) *Handler {
	return &Handler{db: db, audit: audit}
}

// RegisterPublicRoutes: el nombre se muestra también en la pantalla de inicio de sesión.
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/settings", h.get)
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.With(authn.RequireRole(authn.RoleAdmin)).Put("/settings", h.update)
}

func (h *Handler) load() (Settings, error) {
	s := Settings{BusinessName: "Mi Restaurante", BusinessIcon: "🍽️"}
	rows, err := h.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return s, err
		}
		switch k {
		case "business_name":
			s.BusinessName = v
		case "business_icon":
			s.BusinessIcon = v
		}
	}
	return s, rows.Err()
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.load()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in Settings
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	in.BusinessName = strings.TrimSpace(in.BusinessName)
	in.BusinessIcon = strings.TrimSpace(in.BusinessIcon)
	if in.BusinessName == "" || len([]rune(in.BusinessName)) > 60 {
		httpx.Error(w, http.StatusBadRequest, errValidation.Error()+": el nombre es obligatorio (máx. 60 caracteres)")
		return
	}
	if in.BusinessIcon == "" {
		in.BusinessIcon = "🍽️"
	}
	if len([]rune(in.BusinessIcon)) > 8 {
		httpx.Error(w, http.StatusBadRequest, errValidation.Error()+": el ícono debe ser un emoji o pocas letras")
		return
	}
	for k, v := range map[string]string{"business_name": in.BusinessName, "business_icon": in.BusinessIcon} {
		if _, err := h.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, k, v); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "update", "settings", 0, in.BusinessName)
	httpx.JSON(w, http.StatusOK, in)
}
