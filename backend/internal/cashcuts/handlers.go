package cashcuts

import (
	"encoding/csv"
	"errors"
	"fmt"
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
	operate := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente, authn.RoleCajero)
	view := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente, authn.RoleCajero, authn.RoleContador)

	r.Route("/shifts", func(r chi.Router) {
		r.With(operate).Post("/open", h.openShift)
		r.With(view).Get("/", h.listShifts)
		r.With(view).Get("/{id}", h.getShift)
		r.With(operate).Post("/{id}/close", h.closeShift)
		r.With(operate).Post("/{id}/cuts/x", h.xCut)
		r.With(view).Get("/{id}/cuts", h.listShiftCuts)
	})

	r.Route("/cuts", func(r chi.Router) {
		r.With(view).Get("/", h.listCuts)
		r.With(view).Get("/{id}", h.getCut)
		r.With(view).Get("/{id}/export", h.exportCut)
	})
}

func (h *Handler) openShift(w http.ResponseWriter, r *http.Request) {
	var in OpenShiftInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	shift, err := h.svc.OpenShift(claims.UserID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "open", "cash_shift", shift.ID, "")
	httpx.JSON(w, http.StatusCreated, shift)
}

func (h *Handler) listShifts(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListShifts(r.URL.Query().Get("status"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) getShift(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	shift, err := h.svc.GetShift(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, shift)
}

func (h *Handler) closeShift(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in CloseShiftInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	shift, cut, err := h.svc.CloseShift(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "close", "cash_shift", id, fmt.Sprintf("diferencia=%.2f", derefFloat(cut.Difference)))
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"shift": shift, "cut": cut})
}

func (h *Handler) xCut(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in CutInput
	_ = httpx.Decode(r, &in)
	claims, _ := authn.FromContext(r.Context())
	cut, err := h.svc.GenerateXCut(id, in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, cut)
}

func (h *Handler) listShiftCuts(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.svc.ListCutsByShift(id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) listCuts(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListCuts(r.URL.Query().Get("from"), r.URL.Query().Get("to"), r.URL.Query().Get("type"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) getCut(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	cut, err := h.svc.GetCut(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cut)
}

// exportCut regresa el corte en CSV, compatible con Excel, para contabilidad.
func (h *Handler) exportCut(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	cut, err := h.svc.GetCut(id)
	if err != nil {
		writeErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="corte_%d.csv"`, cut.ID))
	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"Corte", fmt.Sprintf("%d", cut.ID), "Turno", fmt.Sprintf("%d", cut.ShiftID), "Tipo", cut.Type, "Fecha", cut.CreatedAt})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Desglose por forma de pago"})
	_ = cw.Write([]string{"Método", "Total"})
	for _, m := range cut.Breakdown.ByMethod {
		_ = cw.Write([]string{m.Method, fmt.Sprintf("%.2f", m.Total)})
	}
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Desglose por mesero"})
	_ = cw.Write([]string{"Mesero", "Total"})
	for _, wt := range cut.Breakdown.ByWaiter {
		_ = cw.Write([]string{wt.WaiterName, fmt.Sprintf("%.2f", wt.Total)})
	}
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Desglose por categoría"})
	_ = cw.Write([]string{"Categoría", "Total"})
	for _, c := range cut.Breakdown.ByCategory {
		_ = cw.Write([]string{c.Category, fmt.Sprintf("%.2f", c.Total)})
	}
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Propinas", fmt.Sprintf("%.2f", cut.Breakdown.TipsTotal)})
	_ = cw.Write([]string{"Descuentos/cortesías", fmt.Sprintf("%.2f", cut.Breakdown.DiscountsTotal)})
	_ = cw.Write([]string{"Venta total", fmt.Sprintf("%.2f", cut.Breakdown.SalesTotal)})
	_ = cw.Write([]string{"Efectivo esperado", fmt.Sprintf("%.2f", cut.ExpectedCash)})
	if cut.CountedCash != nil {
		_ = cw.Write([]string{"Efectivo contado", fmt.Sprintf("%.2f", *cut.CountedCash)})
	}
	if cut.Difference != nil {
		_ = cw.Write([]string{"Diferencia (faltante/sobrante)", fmt.Sprintf("%.2f", *cut.Difference)})
	}
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func idParam(r *http.Request, name string) (int, error) {
	return strconv.Atoi(chi.URLParam(r, name))
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation), errors.Is(err, ErrAlreadyOpen):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrCutNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
