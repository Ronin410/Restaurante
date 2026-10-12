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

// RegisterRoutes: Contaduría la administran "admin" y "contador" (alta,
// edición y baja de gastos, compras, inventario y facturas); el rol
// "gerente" no tiene acceso a este módulo. Toda alta/edición/baja queda en
// la bitácora de auditoría.
func (h *Handler) RegisterRoutes(r chi.Router) {
	manage := authn.RequireRole(authn.RoleAdmin, authn.RoleContador)
	view := authn.RequireRole(authn.RoleAdmin, authn.RoleContador)

	r.Route("/expenses", func(r chi.Router) {
		r.With(view).Get("/", h.listExpenses)
		r.With(view).Get("/export", h.exportExpenses)
		r.With(manage).Post("/", h.createExpense)
		r.With(manage).Put("/{id}", h.updateExpense)
		r.With(manage).Delete("/{id}", h.deleteExpense)
	})

	r.Route("/purchases", func(r chi.Router) {
		r.With(view).Get("/", h.listPurchases)
		r.With(manage).Post("/", h.createPurchase)
		r.With(manage).Put("/{id}", h.updatePurchase)
		r.With(manage).Delete("/{id}", h.deletePurchase)
	})

	r.Route("/inventory", func(r chi.Router) {
		r.With(view).Get("/", h.listInventory)
		r.With(manage).Post("/", h.createInventoryItem)
		r.With(manage).Put("/{id}", h.updateInventoryItem)
		r.With(manage).Delete("/{id}", h.deleteInventoryItem)
		r.With(manage).Post("/{id}/movements", h.addMovement)
		r.With(manage).Delete("/{id}/movements/{movementId}", h.deleteMovement)
		r.With(view).Get("/{id}/movements", h.listMovements)
	})

	r.Route("/invoices", func(r chi.Router) {
		r.With(view).Get("/", h.listInvoices)
		r.With(view).Get("/{id}", h.getInvoice)
		r.With(manage).Post("/", h.createInvoice)
		r.With(manage).Post("/{id}/cancel", h.cancelInvoice)
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
	h.audit.Log(claims.UserID, "create", "expense", e.ID, e.Description)
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handler) updateExpense(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in ExpenseInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	e, err := h.svc.UpdateExpense(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "update", "expense", id, e.Description)
	httpx.JSON(w, http.StatusOK, e)
}

func (h *Handler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	h.deleteGeneric(w, r, "expense", h.svc.DeleteExpense)
}

func (h *Handler) updatePurchase(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in PurchaseInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.svc.UpdatePurchase(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "update", "purchase", id, p.Supplier)
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) deletePurchase(w http.ResponseWriter, r *http.Request) {
	h.deleteGeneric(w, r, "purchase", h.svc.DeletePurchase)
}

func (h *Handler) updateInventoryItem(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in InventoryItemInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	item, err := h.svc.UpdateInventoryItem(id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "update", "inventory_item", id, item.Name)
	httpx.JSON(w, http.StatusOK, item)
}

func (h *Handler) deleteInventoryItem(w http.ResponseWriter, r *http.Request) {
	h.deleteGeneric(w, r, "inventory_item", h.svc.DeleteInventoryItem)
}

func (h *Handler) deleteMovement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	mid, err2 := strconv.Atoi(chi.URLParam(r, "movementId"))
	if err != nil || err2 != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	item, err := h.svc.DeleteMovement(id, mid)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "delete_movement", "inventory_item", id, strconv.Itoa(mid))
	httpx.JSON(w, http.StatusOK, item)
}

func (h *Handler) cancelInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	inv, err := h.svc.CancelInvoice(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "cancel", "invoice", id, "")
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) deleteGeneric(w http.ResponseWriter, r *http.Request, entity string, del func(int) error) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := del(id); err != nil {
		writeErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "delete", entity, id, "")
	w.WriteHeader(http.StatusNoContent)
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
	h.audit.Log(claims.UserID, "create", "purchase", p.ID, p.Supplier)
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
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "create", "inventory_item", item.ID, item.Name)
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
