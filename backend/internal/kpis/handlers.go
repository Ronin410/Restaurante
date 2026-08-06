package kpis

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// KPIs operativos: visibles para admin, gerente y contador.
func (h *Handler) RegisterRoutes(r chi.Router) {
	view := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente, authn.RoleContador)
	r.With(view).Get("/kpis/dashboard", h.dashboard)
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Dashboard(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}
