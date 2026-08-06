package accounting

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("registro no encontrado")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ---- Gastos ----

func (s *Store) ListExpenses(from, to string) ([]Expense, error) {
	rows, err := s.db.Query(`SELECT id, category, description, amount, supplier, date, created_by, created_at
		FROM expenses WHERE date BETWEEN ? AND ? ORDER BY date DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Expense
	for rows.Next() {
		var e Expense
		if err := rows.Scan(&e.ID, &e.Category, &e.Description, &e.Amount, &e.Supplier, &e.Date, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateExpense(in ExpenseInput, createdBy int) (Expense, error) {
	res, err := s.db.Exec(`INSERT INTO expenses (category, description, amount, supplier, date, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		in.Category, in.Description, in.Amount, in.Supplier, in.Date, createdBy)
	if err != nil {
		return Expense{}, err
	}
	id, _ := res.LastInsertId()
	row := s.db.QueryRow(`SELECT id, category, description, amount, supplier, date, created_by, created_at FROM expenses WHERE id = ?`, id)
	var e Expense
	err = row.Scan(&e.ID, &e.Category, &e.Description, &e.Amount, &e.Supplier, &e.Date, &e.CreatedBy, &e.CreatedAt)
	return e, err
}

// ---- Compras ----

func (s *Store) ListPurchases(from, to string) ([]Purchase, error) {
	rows, err := s.db.Query(`SELECT id, supplier, description, amount, date, created_by, created_at
		FROM purchases WHERE date BETWEEN ? AND ? ORDER BY date DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Purchase
	for rows.Next() {
		var p Purchase
		if err := rows.Scan(&p.ID, &p.Supplier, &p.Description, &p.Amount, &p.Date, &p.CreatedBy, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreatePurchase(in PurchaseInput, createdBy int) (Purchase, error) {
	res, err := s.db.Exec(`INSERT INTO purchases (supplier, description, amount, date, created_by) VALUES (?, ?, ?, ?, ?)`,
		in.Supplier, in.Description, in.Amount, in.Date, createdBy)
	if err != nil {
		return Purchase{}, err
	}
	id, _ := res.LastInsertId()
	row := s.db.QueryRow(`SELECT id, supplier, description, amount, date, created_by, created_at FROM purchases WHERE id = ?`, id)
	var p Purchase
	err = row.Scan(&p.ID, &p.Supplier, &p.Description, &p.Amount, &p.Date, &p.CreatedBy, &p.CreatedAt)
	return p, err
}

// ---- Inventario básico ----

func (s *Store) ListInventory() ([]InventoryItem, error) {
	rows, err := s.db.Query(`SELECT id, name, unit, stock, min_stock, updated_at FROM inventory_items ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryItem
	for rows.Next() {
		var it InventoryItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Unit, &it.Stock, &it.MinStock, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.LowStock = it.Stock <= it.MinStock
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) CreateInventoryItem(in InventoryItemInput) (InventoryItem, error) {
	res, err := s.db.Exec(`INSERT INTO inventory_items (name, unit, min_stock) VALUES (?, ?, ?)`, in.Name, in.Unit, in.MinStock)
	if err != nil {
		return InventoryItem{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetInventoryItem(int(id))
}

func (s *Store) GetInventoryItem(id int) (InventoryItem, error) {
	row := s.db.QueryRow(`SELECT id, name, unit, stock, min_stock, updated_at FROM inventory_items WHERE id = ?`, id)
	var it InventoryItem
	err := row.Scan(&it.ID, &it.Name, &it.Unit, &it.Stock, &it.MinStock, &it.UpdatedAt)
	if err == sql.ErrNoRows {
		return it, ErrNotFound
	}
	it.LowStock = it.Stock <= it.MinStock
	return it, err
}

func (s *Store) AddMovement(itemID int, in InventoryMovementInput, createdBy int) (InventoryItem, error) {
	delta := in.Qty
	if in.Type == "salida" || in.Type == "merma" {
		delta = -in.Qty
	}
	if _, err := s.db.Exec(`INSERT INTO inventory_movements (inventory_item_id, type, qty, reason, created_by) VALUES (?, ?, ?, ?, ?)`,
		itemID, in.Type, in.Qty, in.Reason, createdBy); err != nil {
		return InventoryItem{}, err
	}
	if _, err := s.db.Exec(`UPDATE inventory_items SET stock = stock + ?, updated_at = datetime('now') WHERE id = ?`, delta, itemID); err != nil {
		return InventoryItem{}, err
	}
	return s.GetInventoryItem(itemID)
}

func (s *Store) ListMovements(itemID int) ([]InventoryMovement, error) {
	rows, err := s.db.Query(`SELECT id, inventory_item_id, type, qty, reason, created_by, created_at FROM inventory_movements WHERE inventory_item_id = ? ORDER BY id DESC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InventoryMovement
	for rows.Next() {
		var m InventoryMovement
		if err := rows.Scan(&m.ID, &m.InventoryItemID, &m.Type, &m.Qty, &m.Reason, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---- Facturación CFDI (stub) ----

func (s *Store) CreateInvoice(in InvoiceInput, createdBy int) (Invoice, error) {
	res, err := s.db.Exec(`INSERT INTO invoices (account_id, rfc, razon_social, uso_cfdi, status, created_by) VALUES (?, ?, ?, ?, 'timbrada_simulada', ?)`,
		in.AccountID, in.RFC, in.RazonSocial, in.UsoCFDI, createdBy)
	if err != nil {
		return Invoice{}, err
	}
	id, _ := res.LastInsertId()
	uuidSim := simulatedUUID(int(id))
	if _, err := s.db.Exec(`UPDATE invoices SET uuid_fiscal = ? WHERE id = ?`, uuidSim, id); err != nil {
		return Invoice{}, err
	}
	return s.GetInvoice(int(id))
}

func (s *Store) GetInvoice(id int) (Invoice, error) {
	row := s.db.QueryRow(`SELECT id, account_id, rfc, razon_social, uso_cfdi, status, uuid_fiscal, created_at FROM invoices WHERE id = ?`, id)
	var inv Invoice
	err := row.Scan(&inv.ID, &inv.AccountID, &inv.RFC, &inv.RazonSocial, &inv.UsoCFDI, &inv.Status, &inv.UUIDFiscal, &inv.CreatedAt)
	if err == sql.ErrNoRows {
		return inv, ErrNotFound
	}
	return inv, err
}

func (s *Store) ListInvoices() ([]Invoice, error) {
	rows, err := s.db.Query(`SELECT id, account_id, rfc, razon_social, uso_cfdi, status, uuid_fiscal, created_at FROM invoices ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.AccountID, &inv.RFC, &inv.RazonSocial, &inv.UsoCFDI, &inv.Status, &inv.UUIDFiscal, &inv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// ---- Reportes ----

const salesJoin = `
	FROM order_items oi
	JOIN accounts a ON a.id = oi.account_id
	JOIN products p ON p.id = oi.product_id
	JOIN categories c ON c.id = p.category_id
	WHERE a.status = 'cerrada' AND oi.status != 'cancelado' AND date(a.closed_at) BETWEEN ? AND ?`

const lineTotalExpr = `oi.qty * (oi.unit_price + COALESCE((
	SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
), 0))`

// SalesReport arma el reporte de ventas. El total general y el desglose por
// día se calculan sobre accounts.total (neto: subtotal ya menos descuentos
// y cortesías), para que cuadren con lo efectivamente cobrado (y con el
// desglose por método de pago). El desglose por producto/categoría, en
// cambio, se calcula a precio de lista (bruto): un descuento se autoriza a
// nivel cuenta, no por partida, así que prorratearlo por producto añadiría
// complejidad sin una atribución exacta; para "producto más vendido" y
// mezcla de menú el precio de lista es la referencia útil.
func (s *Store) SalesReport(from, to string) (SalesReport, error) {
	report := SalesReport{From: from, To: to}

	const accountsFilter = `FROM accounts WHERE status = 'cerrada' AND date(closed_at) BETWEEN ? AND ?`

	if err := s.db.QueryRow(`SELECT COALESCE(SUM(total),0) `+accountsFilter, from, to).Scan(&report.Total); err != nil {
		return report, err
	}

	dayRows, err := s.db.Query(`SELECT date(closed_at), COALESCE(SUM(total),0) `+accountsFilter+` GROUP BY date(closed_at) ORDER BY date(closed_at)`, from, to)
	if err != nil {
		return report, err
	}
	for dayRows.Next() {
		var d DailySales
		if err := dayRows.Scan(&d.Date, &d.Total); err != nil {
			dayRows.Close()
			return report, err
		}
		report.ByDay = append(report.ByDay, d)
	}
	dayRows.Close()
	if err := dayRows.Err(); err != nil {
		return report, err
	}

	prodRows, err := s.db.Query(`SELECT oi.product_name_snapshot, SUM(oi.qty), COALESCE(SUM(`+lineTotalExpr+`),0) `+salesJoin+` GROUP BY oi.product_name_snapshot ORDER BY 3 DESC`, from, to)
	if err != nil {
		return report, err
	}
	for prodRows.Next() {
		var p ProductSales
		if err := prodRows.Scan(&p.ProductName, &p.Qty, &p.Total); err != nil {
			prodRows.Close()
			return report, err
		}
		report.ByProduct = append(report.ByProduct, p)
	}
	prodRows.Close()
	if err := prodRows.Err(); err != nil {
		return report, err
	}

	catRows, err := s.db.Query(`SELECT c.name, COALESCE(SUM(`+lineTotalExpr+`),0) `+salesJoin+` GROUP BY c.name ORDER BY 2 DESC`, from, to)
	if err != nil {
		return report, err
	}
	for catRows.Next() {
		var c CategorySales
		if err := catRows.Scan(&c.Category, &c.Total); err != nil {
			catRows.Close()
			return report, err
		}
		report.ByCategory = append(report.ByCategory, c)
	}
	catRows.Close()
	return report, catRows.Err()
}

func (s *Store) sumExpenses(from, to string) (float64, error) {
	var total float64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM expenses WHERE date BETWEEN ? AND ?`, from, to).Scan(&total)
	return total, err
}

func (s *Store) sumPurchases(from, to string) (float64, error) {
	var total float64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM purchases WHERE date BETWEEN ? AND ?`, from, to).Scan(&total)
	return total, err
}

func (s *Store) sumActiveSalaries() (float64, error) {
	var total float64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(salary),0) FROM users WHERE active = 1`).Scan(&total)
	return total, err
}

// simulatedUUID genera un folio con apariencia de UUID fiscal para dejar
// claro, en cualquier reporte o ticket, que el timbrado NO es real: falta
// integrar un PAC autorizado por el SAT.
func simulatedUUID(id int) string {
	return fmt.Sprintf("SIMULADO-0000-0000-0000-%012d", id)
}
