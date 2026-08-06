package audit

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/httpx"
)

type Handler struct {
	logger *Logger
}

func NewHandler(logger *Logger) *Handler {
	return &Handler{logger: logger}
}

// La bitácora de acciones (quién canceló, quién descontó, etc.) solo es
// visible para administrador y gerente, por control interno.
func (h *Handler) RegisterRoutes(r chi.Router) {
	view := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente)
	r.With(view).Get("/audit-log", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	entries, err := h.logger.List(limit, r.URL.Query().Get("entity"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, entries)
}
