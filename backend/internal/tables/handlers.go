package tables

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ronin410/restaurante/backend/internal/audit"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/httpx"
)

type Handler struct {
	svc   *Service
	audit *audit.Logger
}

func NewHandler(svc *Service, audit *audit.Logger) *Handler {
	return &Handler{svc: svc, audit: audit}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	manage := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente)
	supervisor := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente)

	r.Route("/zones", func(r chi.Router) {
		r.Get("/", h.listZones)
		r.With(manage).Post("/", h.createZone)
	})

	r.Route("/tables", func(r chi.Router) {
		r.Get("/", h.listTables)
		r.Get("/{id}", h.getTable)
		r.With(manage).Post("/", h.createTable)
		r.With(manage).Put("/{id}", h.updateTable)
		r.Post("/{id}/open", h.open)
		r.Post("/{id}/status", h.setStatus)
		r.Post("/{id}/assign", h.assignWaiter)
		r.Post("/{id}/join/{withId}", h.join)
		r.Post("/{id}/unjoin", h.unjoin)
		r.With(supervisor).Post("/{id}/transfer", h.transfer)
	})
}

func (h *Handler) listZones(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListZones()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createZone(w http.ResponseWriter, r *http.Request) {
	var in ZoneInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	z, err := h.svc.CreateZone(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, z)
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListTables()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) getTable(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	t, err := h.svc.GetTable(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	var in TableInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	t, err := h.svc.CreateTable(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTable(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in TableInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	t, err := h.svc.UpdateTable(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	var in struct {
		WaiterID int `json:"waiter_id"`
	}
	_ = httpx.Decode(r, &in)
	waiterID := claims.UserID
	if in.WaiterID != 0 {
		waiterID = in.WaiterID
	}
	t, err := h.svc.Open(id, waiterID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	t, err := h.svc.SetStatus(id, in.Status)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) assignWaiter(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in struct {
		WaiterID int `json:"waiter_id"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	t, err := h.svc.AssignWaiter(id, in.WaiterID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	withID, err := idParam(r, "withId")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	t, err := h.svc.Join(id, withID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) unjoin(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	t, err := h.svc.Unjoin(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in TransferInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	tr, err := h.svc.Transfer(id, in.ToWaiterID, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "transfer", "table", id, "")
	httpx.JSON(w, http.StatusCreated, tr)
}

func idParam(r *http.Request, name string) (int, error) {
	return strconv.Atoi(chi.URLParam(r, name))
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
