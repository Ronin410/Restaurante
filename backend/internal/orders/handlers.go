package orders

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
	supervisor := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente)
	kitchen := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente, authn.RoleCocina)

	r.Route("/orders", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Post("/{id}/items", h.addItems)
		r.Post("/{id}/close", h.close)
		r.With(kitchen).Post("/items/{itemId}/status", h.updateItemStatus)
		r.With(supervisor).Post("/items/{itemId}/cancel", h.cancelItem)
	})

	r.Get("/kds/items", h.kdsItems)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	var tableID *int
	if v := r.URL.Query().Get("table_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			tableID = &id
		}
	}
	list, err := h.svc.ListOpenOrders(tableID, r.URL.Query().Get("no_table") == "1")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateOrderInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	order, err := h.svc.CreateOrder(claims.UserID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "create", "order", order.ID, order.Type)
	httpx.JSON(w, http.StatusCreated, order)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	order, err := h.svc.GetOrder(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) addItems(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in AddItemsInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	order, err := h.svc.AddItems(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "add-items", "order", id, "")
	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.svc.CloseOrder(id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "cerrada"})
}

func (h *Handler) kdsItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListKitchenItems()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) updateItemStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "itemId")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in UpdateItemStatusInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	item, err := h.svc.UpdateItemStatus(id, in.Status)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (h *Handler) cancelItem(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "itemId")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in CancelItemInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	item, err := h.svc.CancelItem(id, in.Reason, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "cancel", "order_item", id, in.Reason)
	httpx.JSON(w, http.StatusOK, item)
}

func idParam(r *http.Request, name string) (int, error) {
	return strconv.Atoi(chi.URLParam(r, name))
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrItemNotFound), errors.Is(err, ErrProductNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
