package cashcuts

import (
	"database/sql"

	"encoding/json"
	"errors"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

var ErrNotFound = errors.New("turno de caja no encontrado")
var ErrCutNotFound = errors.New("corte no encontrado")
var ErrAlreadyOpen = errors.New("ya existe un turno de caja abierto para este cajero")

type Store struct {
	db *dbx.DB
}

func NewStore(db *dbx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) OpenShiftIDForCashier(cashierID int) (*int, error) {
	var id int
	err := s.db.QueryRow(`SELECT id FROM cash_shifts WHERE cashier_id = ? AND status = 'abierto' ORDER BY id DESC LIMIT 1`, cashierID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *Store) HasOpenShift(cashierID int) (bool, error) {
	id, err := s.OpenShiftIDForCashier(cashierID)
	return id != nil, err
}

func (s *Store) OpenShift(cashierID int, openingFund float64) (Shift, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO cash_shifts (cashier_id, opening_fund) VALUES (?, ?) RETURNING id`, cashierID, openingFund).Scan(&id)
	if err != nil {
		return Shift{}, err
	}
	return s.GetShift(id)
}

const shiftCols = `s.id, s.cashier_id, u.name, s.opening_fund, s.opened_at, s.closed_at, s.counted_cash, s.status`

func scanShift(row interface{ Scan(...interface{}) error }) (Shift, error) {
	var sh Shift
	err := row.Scan(&sh.ID, &sh.CashierID, &sh.CashierName, &sh.OpeningFund, &sh.OpenedAt, &sh.ClosedAt, &sh.CountedCash, &sh.Status)
	return sh, err
}

func (s *Store) GetShift(id int) (Shift, error) {
	row := s.db.QueryRow(`SELECT `+shiftCols+` FROM cash_shifts s JOIN users u ON u.id = s.cashier_id WHERE s.id = ?`, id)
	sh, err := scanShift(row)
	if err == sql.ErrNoRows {
		return sh, ErrNotFound
	}
	return sh, err
}

func (s *Store) ListShifts(status string) ([]Shift, error) {
	query := `SELECT ` + shiftCols + ` FROM cash_shifts s JOIN users u ON u.id = s.cashier_id`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE s.status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY s.opened_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Shift
	for rows.Next() {
		sh, err := scanShift(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *Store) CloseShift(id int, countedCash float64) (Shift, error) {
	_, err := s.db.Exec(`UPDATE cash_shifts SET status = 'cerrado', closed_at = datetime('now'), counted_cash = ? WHERE id = ?`, countedCash, id)
	if err != nil {
		return Shift{}, err
	}
	return s.GetShift(id)
}

// buildBreakdown calcula el desglose de ventas del turno: por forma de
// pago, por mesero, por categoría de producto, propinas y descuentos.
func (s *Store) buildBreakdown(shiftID int) (Breakdown, error) {
	var b Breakdown

	methodRows, err := s.db.Query(`SELECT method, COALESCE(SUM(amount),0) FROM payments WHERE shift_id = ? GROUP BY method`, shiftID)
	if err != nil {
		return b, err
	}
	for methodRows.Next() {
		var m MethodTotal
		if err := methodRows.Scan(&m.Method, &m.Total); err != nil {
			methodRows.Close()
			return b, err
		}
		b.ByMethod = append(b.ByMethod, m)
		b.SalesTotal += m.Total
	}
	methodRows.Close()
	if err := methodRows.Err(); err != nil {
		return b, err
	}

	if err := s.db.QueryRow(`SELECT COALESCE(SUM(tip_amount),0) FROM payments WHERE shift_id = ?`, shiftID).Scan(&b.TipsTotal); err != nil {
		return b, err
	}

	// Cuentas que tuvieron al menos un pago dentro de este turno.
	waiterRows, err := s.db.Query(`
		SELECT o.waiter_id, u.name, COALESCE(SUM(oi.qty * (oi.unit_price + COALESCE((
			SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
		), 0))), 0) AS total
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN users u ON u.id = o.waiter_id
		WHERE oi.status != 'cancelado' AND oi.account_id IN (SELECT DISTINCT account_id FROM payments WHERE shift_id = ?)
		GROUP BY o.waiter_id, u.name`, shiftID)
	if err != nil {
		return b, err
	}
	for waiterRows.Next() {
		var wt WaiterTotal
		if err := waiterRows.Scan(&wt.WaiterID, &wt.WaiterName, &wt.Total); err != nil {
			waiterRows.Close()
			return b, err
		}
		b.ByWaiter = append(b.ByWaiter, wt)
	}
	waiterRows.Close()
	if err := waiterRows.Err(); err != nil {
		return b, err
	}

	categoryRows, err := s.db.Query(`
		SELECT c.name, COALESCE(SUM(oi.qty * (oi.unit_price + COALESCE((
			SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
		), 0))), 0) AS total
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		JOIN categories c ON c.id = p.category_id
		WHERE oi.status != 'cancelado' AND oi.account_id IN (SELECT DISTINCT account_id FROM payments WHERE shift_id = ?)
		GROUP BY c.name`, shiftID)
	if err != nil {
		return b, err
	}
	for categoryRows.Next() {
		var ct CategoryTotal
		if err := categoryRows.Scan(&ct.Category, &ct.Total); err != nil {
			categoryRows.Close()
			return b, err
		}
		b.ByCategory = append(b.ByCategory, ct)
	}
	categoryRows.Close()
	if err := categoryRows.Err(); err != nil {
		return b, err
	}

	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(d.amount),0) FROM discounts d
		WHERE d.account_id IN (SELECT DISTINCT account_id FROM payments WHERE shift_id = ?)`, shiftID).Scan(&b.DiscountsTotal); err != nil {
		return b, err
	}

	return b, nil
}

func (s *Store) expectedCash(shiftID int) (float64, error) {
	var openingFund float64
	if err := s.db.QueryRow(`SELECT opening_fund FROM cash_shifts WHERE id = ?`, shiftID).Scan(&openingFund); err != nil {
		return 0, err
	}
	var cashSales float64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM payments WHERE shift_id = ? AND method = 'efectivo'`, shiftID).Scan(&cashSales); err != nil {
		return 0, err
	}
	return openingFund + cashSales, nil
}

func (s *Store) CreateCut(shiftID int, cutType string, countedCash *float64, createdBy int) (CashCut, error) {
	breakdown, err := s.buildBreakdown(shiftID)
	if err != nil {
		return CashCut{}, err
	}
	expected, err := s.expectedCash(shiftID)
	if err != nil {
		return CashCut{}, err
	}
	var diff *float64
	if countedCash != nil {
		d := *countedCash - expected
		diff = &d
	}
	raw, err := json.Marshal(breakdown)
	if err != nil {
		return CashCut{}, err
	}
	var id int
	err = s.db.QueryRow(
		`INSERT INTO cash_cuts (shift_id, type, totals_json, expected_cash, counted_cash, difference, created_by) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		shiftID, cutType, string(raw), expected, countedCash, diff, createdBy,
	).Scan(&id)
	if err != nil {
		return CashCut{}, err
	}
	return s.GetCut(id)
}

func scanCut(row interface{ Scan(...interface{}) error }) (CashCut, error) {
	var c CashCut
	var raw string
	err := row.Scan(&c.ID, &c.ShiftID, &c.Type, &raw, &c.ExpectedCash, &c.CountedCash, &c.Difference, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal([]byte(raw), &c.Breakdown)
	return c, nil
}

const cutCols = `id, shift_id, type, totals_json, expected_cash, counted_cash, difference, created_by, created_at`

func (s *Store) GetCut(id int) (CashCut, error) {
	row := s.db.QueryRow(`SELECT `+cutCols+` FROM cash_cuts WHERE id = ?`, id)
	c, err := scanCut(row)
	if err == sql.ErrNoRows {
		return c, ErrCutNotFound
	}
	return c, err
}

func (s *Store) ListCutsByShift(shiftID int) ([]CashCut, error) {
	rows, err := s.db.Query(`SELECT `+cutCols+` FROM cash_cuts WHERE shift_id = ? ORDER BY id`, shiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CashCut
	for rows.Next() {
		c, err := scanCut(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ListCuts(from, to, cutType string) ([]CashCut, error) {
	query := `SELECT ` + cutCols + ` FROM cash_cuts WHERE date(created_at) BETWEEN ? AND ?`
	args := []interface{}{from, to}
	if cutType != "" {
		query += ` AND type = ?`
		args = append(args, cutType)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CashCut
	for rows.Next() {
		c, err := scanCut(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
