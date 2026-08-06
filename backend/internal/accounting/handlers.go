package accounting

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

// RegisterRoutes: por especificación, Contaduría es de solo lectura para el
// rol "contador" y de administración completa para "admin" (el rol
// "gerente" no aparece listado con acceso a este módulo en las
// especificaciones funcionales).
func (h *Handler) RegisterRoutes(r chi.Router) {
	manage := authn.RequireRole(authn.RoleAdmin)
	view := authn.RequireRole(authn.RoleAdmin, authn.RoleContador)

	r.Route("/expenses", func(r chi.Router) {
		r.With(view).Get("/", h.listExpenses)
		r.With(view).Get("/export", h.exportExpenses)
		r.With(manage).Post("/", h.createExpense)
	})

	r.Route("/purchases", func(r chi.Router) {
		r.With(view).Get("/", h.listPurchases)
		r.With(manage).Post("/", h.createPurchase)
	})

	r.Route("/inventory", func(r chi.Router) {
		r.With(view).Get("/", h.listInventory)
		r.With(manage).Post("/", h.createInventoryItem)
		r.With(manage).Post("/{id}/movements", h.addMovement)
		r.With(view).Get("/{id}/movements", h.listMovements)
	})

	r.Route("/invoices", func(r chi.Router) {
		r.With(view).Get("/", h.listInvoices)
		r.With(view).Get("/{id}", h.getInvoice)
		r.With(manage).Post("/", h.createInvoice)
	})

	r.Route("/reports", func(r chi.Router) {
		r.With(view).Get("/sales", h.salesReport)
		r.With(view).Get("/sales/export", h.exportSalesReport)
		r.With(view).Get("/taxes", h.taxReport)
		r.With(view).Get("/pnl", h.pnlReport)
	})
}

func (h *Handler) listExpenses(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListExpenses(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) exportExpenses(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListExpenses(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="gastos.csv"`)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{"Fecha", "Categoría", "Descripción", "Proveedor", "Monto"})
	for _, e := range list {
		_ = cw.Write([]string{e.Date, e.Category, e.Description, e.Supplier, fmt.Sprintf("%.2f", e.Amount)})
	}
}

func (h *Handler) createExpense(w http.ResponseWriter, r *http.Request) {
	var in ExpenseInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	e, err := h.svc.CreateExpense(in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handler) listPurchases(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListPurchases(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createPurchase(w http.ResponseWriter, r *http.Request) {
	var in PurchaseInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	p, err := h.svc.CreatePurchase(in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) listInventory(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListInventory()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createInventoryItem(w http.ResponseWriter, r *http.Request) {
	var in InventoryItemInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	item, err := h.svc.CreateInventoryItem(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (h *Handler) addMovement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in InventoryMovementInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	item, err := h.svc.AddMovement(id, in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "movement", "inventory_item", id, in.Type)
	httpx.JSON(w, http.StatusCreated, item)
}

func (h *Handler) listMovements(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.svc.ListMovements(id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createInvoice(w http.ResponseWriter, r *http.Request) {
	var in InvoiceInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	inv, err := h.svc.CreateInvoice(in, claims.UserID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "invoice", "account", in.AccountID, inv.UsoCFDI)
	httpx.JSON(w, http.StatusCreated, inv)
}

func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListInvoices()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	inv, err := h.svc.GetInvoice(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) salesReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.SalesReport(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func (h *Handler) exportSalesReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.SalesReport(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ventas.csv"`)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{"Reporte de ventas", report.From, "a", report.To})
	_ = cw.Write([]string{"Total", fmt.Sprintf("%.2f", report.Total)})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Por día"})
	_ = cw.Write([]string{"Fecha", "Total"})
	for _, d := range report.ByDay {
		_ = cw.Write([]string{d.Date, fmt.Sprintf("%.2f", d.Total)})
	}
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Por producto"})
	_ = cw.Write([]string{"Producto", "Cantidad", "Total"})
	for _, p := range report.ByProduct {
		_ = cw.Write([]string{p.ProductName, fmt.Sprintf("%.2f", p.Qty), fmt.Sprintf("%.2f", p.Total)})
	}
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"Por categoría"})
	_ = cw.Write([]string{"Categoría", "Total"})
	for _, c := range report.ByCategory {
		_ = cw.Write([]string{c.Category, fmt.Sprintf("%.2f", c.Total)})
	}
}

func (h *Handler) taxReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.TaxReport(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func (h *Handler) pnlReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.ProfitAndLoss(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func idParam(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "id"))
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
