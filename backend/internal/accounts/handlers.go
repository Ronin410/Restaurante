package accounts

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

	r.Route("/accounts", func(r chi.Router) {
		r.Post("/generate", h.generate)
		r.Post("/split-by-product", h.splitByProduct)
		r.Get("/{id}", h.get)
		r.Get("/{id}/ticket", h.ticket)
		r.With(supervisor).Post("/{id}/discounts", h.addDiscount)
		r.Post("/{id}/payments", h.addPayment)
		r.Post("/{id}/close", h.close)
	})

	r.Get("/tables/{tableId}/accounts", h.listByTable)
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	var in GenerateAccountInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	var acc Account
	var err error
	if in.OrderID != 0 && in.TableID == 0 {
		acc, err = h.svc.GenerateOrderAccount(in.OrderID)
	} else {
		acc, err = h.svc.GenerateAccount(in.TableID)
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "generate", "account", acc.ID, "")
	httpx.JSON(w, http.StatusCreated, acc)
}

func (h *Handler) splitByProduct(w http.ResponseWriter, r *http.Request) {
	var in SplitByProductInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	accs, err := h.svc.SplitByProduct(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, accs)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	acc, err := h.svc.GetAccount(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, acc)
}

func (h *Handler) ticket(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	t, err := h.svc.GetTicket(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) addDiscount(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in DiscountInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	acc, err := h.svc.AddDiscount(id, in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "discount", "account", id, in.Reason)
	httpx.JSON(w, http.StatusOK, acc)
}

func (h *Handler) addPayment(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in PaymentInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	acc, err := h.svc.AddPayment(id, in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "payment", "account", id, in.Method)
	httpx.JSON(w, http.StatusCreated, acc)
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	acc, err := h.svc.CloseAccount(id, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "close", "account", id, "")
	httpx.JSON(w, http.StatusOK, acc)
}

func (h *Handler) listByTable(w http.ResponseWriter, r *http.Request) {
	tableID, err := idParam(r, "tableId")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.svc.ListByTable(tableID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
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
	case errors.Is(err, ErrNoItems):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
