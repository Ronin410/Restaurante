package accounting

import (
	"errors"
	"fmt"
	"time"
)

var ErrValidation = errors.New("datos inválidos")

var validMovementTypes = map[string]bool{"entrada": true, "salida": true, "merma": true}

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func normalizeRange(from, to string) (string, string) {
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	return from, to
}

func (s *Service) ListExpenses(from, to string) ([]Expense, error) {
	from, to = normalizeRange(from, to)
	return s.store.ListExpenses(from, to)
}

func (s *Service) CreateExpense(in ExpenseInput, createdBy int) (Expense, error) {
	if in.Description == "" || in.Amount <= 0 {
		return Expense{}, fmt.Errorf("%w: descripción y monto (mayor a cero) son obligatorios", ErrValidation)
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}
	return s.store.CreateExpense(in, createdBy)
}

func (s *Service) UpdateExpense(id int, in ExpenseInput) (Expense, error) {
	if in.Description == "" || in.Amount <= 0 {
		return Expense{}, fmt.Errorf("%w: descripción y monto (mayor a cero) son obligatorios", ErrValidation)
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}
	return s.store.UpdateExpense(id, in)
}

func (s *Service) DeleteExpense(id int) error { return s.store.DeleteExpense(id) }

func (s *Service) UpdatePurchase(id int, in PurchaseInput) (Purchase, error) {
	if in.Supplier == "" || in.Amount <= 0 {
		return Purchase{}, fmt.Errorf("%w: proveedor y monto (mayor a cero) son obligatorios", ErrValidation)
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}
	return s.store.UpdatePurchase(id, in)
}

func (s *Service) DeletePurchase(id int) error { return s.store.DeletePurchase(id) }

func (s *Service) UpdateInventoryItem(id int, in InventoryItemInput) (InventoryItem, error) {
	if in.Name == "" {
		return InventoryItem{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	if in.Unit == "" {
		in.Unit = "pza"
	}
	return s.store.UpdateInventoryItem(id, in)
}

func (s *Service) DeleteInventoryItem(id int) error { return s.store.DeleteInventoryItem(id) }

func (s *Service) DeleteMovement(itemID, movementID int) (InventoryItem, error) {
	return s.store.DeleteMovement(itemID, movementID)
}

func (s *Service) CancelInvoice(id int) (Invoice, error) { return s.store.CancelInvoice(id) }

func (s *Service) ListPurchases(from, to string) ([]Purchase, error) {
	from, to = normalizeRange(from, to)
	return s.store.ListPurchases(from, to)
}

func (s *Service) CreatePurchase(in PurchaseInput, createdBy int) (Purchase, error) {
	if in.Supplier == "" || in.Amount <= 0 {
		return Purchase{}, fmt.Errorf("%w: proveedor y monto (mayor a cero) son obligatorios", ErrValidation)
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}
	return s.store.CreatePurchase(in, createdBy)
}

func (s *Service) ListInventory() ([]InventoryItem, error) { return s.store.ListInventory() }

func (s *Service) CreateInventoryItem(in InventoryItemInput) (InventoryItem, error) {
	if in.Name == "" {
		return InventoryItem{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	if in.Unit == "" {
		in.Unit = "pza"
	}
	return s.store.CreateInventoryItem(in)
}

func (s *Service) AddMovement(itemID int, in InventoryMovementInput, createdBy int) (InventoryItem, error) {
	if !validMovementTypes[in.Type] {
		return InventoryItem{}, fmt.Errorf("%w: tipo de movimiento inválido (entrada, salida o merma)", ErrValidation)
	}
	if in.Qty <= 0 {
		return InventoryItem{}, fmt.Errorf("%w: la cantidad debe ser mayor a cero", ErrValidation)
	}
	return s.store.AddMovement(itemID, in, createdBy)
}

func (s *Service) ListMovements(itemID int) ([]InventoryMovement, error) {
	return s.store.ListMovements(itemID)
}

// CreateInvoice simula el timbrado CFDI 4.0. Ver nota en model.go: falta
// integrar un PAC real autorizado por el SAT para timbrado fiscal válido.
func (s *Service) CreateInvoice(in InvoiceInput, createdBy int) (Invoice, error) {
	if in.AccountID == 0 || in.RFC == "" || in.RazonSocial == "" {
		return Invoice{}, fmt.Errorf("%w: cuenta, RFC y razón social son obligatorios", ErrValidation)
	}
	if in.UsoCFDI == "" {
		in.UsoCFDI = "G03"
	}
	return s.store.CreateInvoice(in, createdBy)
}

func (s *Service) ListInvoices() ([]Invoice, error) { return s.store.ListInvoices() }

func (s *Service) GetInvoice(id int) (Invoice, error) { return s.store.GetInvoice(id) }

func (s *Service) SalesReport(from, to string) (SalesReport, error) {
	from, to = normalizeRange(from, to)
	return s.store.SalesReport(from, to)
}

func (s *Service) TaxReport(from, to string) (TaxReport, error) {
	from, to = normalizeRange(from, to)
	sales, err := s.store.SalesReport(from, to)
	if err != nil {
		return TaxReport{}, err
	}
	// Se asume que los precios del menú ya incluyen IVA (práctica común en
	// restaurantes en México), por lo que el IVA trasladado se extrae del
	// total: IVA = total * 16/116.
	iva := sales.Total * 16 / 116
	return TaxReport{
		From:          from,
		To:            to,
		VentaTotal:    sales.Total,
		IVATrasladado: iva,
		ISRRetenido:   0,
		Note:          "Cálculo simplificado: asume precios de menú con IVA incluido (16%) y régimen sin retención de ISR propia. Verificar con el contador antes de declarar.",
	}, nil
}

func (s *Service) ProfitAndLoss(from, to string) (ProfitAndLoss, error) {
	from, to = normalizeRange(from, to)
	sales, err := s.store.SalesReport(from, to)
	if err != nil {
		return ProfitAndLoss{}, err
	}
	expenses, err := s.store.sumExpenses(from, to)
	if err != nil {
		return ProfitAndLoss{}, err
	}
	purchases, err := s.store.sumPurchases(from, to)
	if err != nil {
		return ProfitAndLoss{}, err
	}
	nomina, err := s.store.sumActiveSalaries()
	if err != nil {
		return ProfitAndLoss{}, err
	}
	return ProfitAndLoss{
		From:     from,
		To:       to,
		Ingresos: sales.Total,
		Gastos:   expenses,
		Compras:  purchases,
		Nomina:   nomina,
		Utilidad: sales.Total - expenses - purchases - nomina,
		Note:     "Nómina estimada con los sueldos vigentes de empleados activos, sin prorratear por el rango de fechas seleccionado.",
	}, nil
}
