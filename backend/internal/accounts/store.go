package accounts

import (
	"database/sql"

	"errors"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

var ErrNotFound = errors.New("cuenta no encontrada")
var ErrNoItems = errors.New("no hay productos pendientes de cobro para esta mesa")

type Store struct {
	db *dbx.DB
}

func NewStore(db *dbx.DB) *Store {
	return &Store{db: db}
}

type unassignedItem struct {
	ID        int
	LineTotal float64
}

// unassignedItems regresa las partidas de comandas abiertas de la mesa que
// aún no están asignadas a ninguna cuenta (no facturadas), excluyendo
// canceladas.
func (s *Store) unassignedItems(tableID int) ([]unassignedItem, error) {
	rows, err := s.db.Query(`
		SELECT oi.id, oi.qty * (oi.unit_price + COALESCE((
			SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
		), 0)) AS line_total
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.table_id = ? AND oi.account_id IS NULL AND oi.status != 'cancelado'`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []unassignedItem
	for rows.Next() {
		var it unassignedItem
		if err := rows.Scan(&it.ID, &it.LineTotal); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) itemsBelongToTable(tableID int, itemIDs []int) (bool, error) {
	all, err := s.unassignedItems(tableID)
	if err != nil {
		return false, err
	}
	valid := make(map[int]bool, len(all))
	for _, it := range all {
		valid[it.ID] = true
	}
	for _, id := range itemIDs {
		if !valid[id] {
			return false, nil
		}
	}
	return true, nil
}

func (s *Store) createAccount(tableID *int, splitMode string) (int, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO accounts (table_id, split_mode) VALUES (?, ?) RETURNING id`, tableID, splitMode).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) assignItems(accountID int, itemIDs []int) error {
	for _, id := range itemIDs {
		if _, err := s.db.Exec(`UPDATE order_items SET account_id = ? WHERE id = ?`, accountID, id); err != nil {
			return err
		}
	}
	return s.recalc(accountID)
}

// GenerateAccount crea una cuenta con todas las partidas pendientes de una
// mesa (flujo "completa").
func (s *Store) GenerateAccount(tableID int) (Account, error) {
	items, err := s.unassignedItems(tableID)
	if err != nil {
		return Account{}, err
	}
	if len(items) == 0 {
		return Account{}, ErrNoItems
	}
	accountID, err := s.createAccount(&tableID, "completa")
	if err != nil {
		return Account{}, err
	}
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	if err := s.assignItems(accountID, ids); err != nil {
		return Account{}, err
	}
	return s.GetAccount(accountID)
}

// SplitByProduct crea una cuenta por cada grupo de partidas indicado
// (división de mesa por producto/comensal).
func (s *Store) SplitByProduct(tableID int, groups [][]int) ([]Account, error) {
	accounts := make([]Account, 0, len(groups))
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		ok, err := s.itemsBelongToTable(tableID, group)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("uno o más productos ya fueron asignados a otra cuenta o no pertenecen a la mesa")
		}
		accountID, err := s.createAccount(&tableID, "producto")
		if err != nil {
			return nil, err
		}
		if err := s.assignItems(accountID, group); err != nil {
			return nil, err
		}
		acc, err := s.GetAccount(accountID)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (s *Store) recalc(accountID int) error {
	var subtotal float64
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(oi.qty * (oi.unit_price + COALESCE((
			SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
		), 0))), 0)
		FROM order_items oi WHERE oi.account_id = ? AND oi.status != 'cancelado'`, accountID).Scan(&subtotal)
	if err != nil {
		return err
	}
	var discountTotal float64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM discounts WHERE account_id = ?`, accountID).Scan(&discountTotal); err != nil {
		return err
	}
	var tipTotal float64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(tip_amount), 0) FROM payments WHERE account_id = ?`, accountID).Scan(&tipTotal); err != nil {
		return err
	}
	total := subtotal - discountTotal
	if total < 0 {
		total = 0
	}
	_, err = s.db.Exec(`UPDATE accounts SET subtotal = ?, discount_total = ?, tip_total = ?, total = ? WHERE id = ?`,
		subtotal, discountTotal, tipTotal, total, accountID)
	return err
}

func (s *Store) GetAccount(id int) (Account, error) {
	row := s.db.QueryRow(`
		SELECT a.id, a.table_id, t.number, a.status, a.split_mode, a.subtotal, a.discount_total, a.tip_total, a.total,
		       a.opened_at, a.closed_at, a.closed_by
		FROM accounts a LEFT JOIN tables_ t ON t.id = a.table_id
		WHERE a.id = ?`, id)
	var a Account
	err := row.Scan(&a.ID, &a.TableID, &a.TableNumber, &a.Status, &a.SplitMode, &a.Subtotal, &a.DiscountTotal, &a.TipTotal,
		&a.Total, &a.OpenedAt, &a.ClosedAt, &a.ClosedBy)
	if err == sql.ErrNoRows {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}

	items, err := s.accountItems(id)
	if err != nil {
		return a, err
	}
	a.Items = items

	payments, err := s.listPayments(id)
	if err != nil {
		return a, err
	}
	a.Payments = payments
	for _, p := range payments {
		a.Paid += p.Amount
	}
	a.Balance = a.Total - a.Paid

	discounts, err := s.listDiscounts(id)
	if err != nil {
		return a, err
	}
	a.Discounts = discounts
	return a, nil
}

func (s *Store) accountItems(accountID int) ([]AccountItem, error) {
	rows, err := s.db.Query(`
		SELECT oi.id, oi.product_name_snapshot, oi.qty, oi.unit_price, oi.notes, oi.status,
		       oi.qty * (oi.unit_price + COALESCE((
		         SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
		       ), 0)) AS line_total
		FROM order_items oi WHERE oi.account_id = ? ORDER BY oi.id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountItem
	for rows.Next() {
		var it AccountItem
		if err := rows.Scan(&it.ID, &it.ProductName, &it.Qty, &it.UnitPrice, &it.Notes, &it.Status, &it.LineTotal); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) listPayments(accountID int) ([]Payment, error) {
	rows, err := s.db.Query(`SELECT id, account_id, method, amount, tip_amount, reference, created_by, created_at FROM payments WHERE account_id = ? ORDER BY id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.AccountID, &p.Method, &p.Amount, &p.TipAmount, &p.Reference, &p.CreatedBy, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) listDiscounts(accountID int) ([]Discount, error) {
	rows, err := s.db.Query(`SELECT id, account_id, kind, amount, percent, reason, authorized_by, created_at FROM discounts WHERE account_id = ? ORDER BY id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Discount
	for rows.Next() {
		var d Discount
		if err := rows.Scan(&d.ID, &d.AccountID, &d.Kind, &d.Amount, &d.Percent, &d.Reason, &d.AuthorizedBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) AddDiscount(accountID int, in DiscountInput, authorizedBy int) (Account, error) {
	_, err := s.db.Exec(
		`INSERT INTO discounts (account_id, kind, amount, percent, reason, authorized_by, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		accountID, in.Kind, in.Amount, in.Percent, in.Reason, authorizedBy, authorizedBy,
	)
	if err != nil {
		return Account{}, err
	}
	if err := s.recalc(accountID); err != nil {
		return Account{}, err
	}
	return s.GetAccount(accountID)
}

func (s *Store) AddPayment(accountID int, in PaymentInput, createdBy int, shiftID *int) (Account, error) {
	_, err := s.db.Exec(
		`INSERT INTO payments (account_id, method, amount, tip_amount, reference, created_by, shift_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		accountID, in.Method, in.Amount, in.TipAmount, in.Reference, createdBy, shiftID,
	)
	if err != nil {
		return Account{}, err
	}
	if err := s.recalc(accountID); err != nil {
		return Account{}, err
	}
	return s.GetAccount(accountID)
}

func (s *Store) CloseAccount(id, closedBy int) (Account, error) {
	_, err := s.db.Exec(`UPDATE accounts SET status = 'cerrada', closed_at = datetime('now'), closed_by = ? WHERE id = ?`, closedBy, id)
	if err != nil {
		return Account{}, err
	}
	if err := s.closeSettledOrders(id); err != nil {
		return Account{}, err
	}
	return s.GetAccount(id)
}

func (s *Store) ListByTable(tableID int) ([]Account, error) {
	rows, err := s.db.Query(`SELECT id FROM accounts WHERE table_id = ? ORDER BY id`, tableID)
	if err != nil {
		return nil, err
	}
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	out := make([]Account, 0, len(ids))
	for _, id := range ids {
		a, err := s.GetAccount(id)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// TableHasPendingBilling indica si la mesa todavía tiene algo por cobrar:
// otra cuenta abierta, o partidas vivas que no se han asignado a ninguna
// cuenta (consumo posterior a "Generar cuenta").
func (s *Store) TableHasPendingBilling(tableID int) (bool, error) {
	var open int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE table_id = ? AND status = 'abierta'`, tableID).Scan(&open); err != nil {
		return false, err
	}
	if open > 0 {
		return true, nil
	}
	var loose int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM order_items oi JOIN orders o ON o.id = oi.order_id
		WHERE o.table_id = ? AND oi.account_id IS NULL AND oi.status != 'cancelado'`, tableID).Scan(&loose)
	return loose > 0, err
}

// OpenAccountForTable devuelve el id de la cuenta abierta más reciente de la
// mesa, o 0 si no hay.
func (s *Store) OpenAccountForTable(tableID int) (int, error) {
	var id int
	err := s.db.QueryRow(`SELECT id FROM accounts WHERE table_id = ? AND status = 'abierta' ORDER BY id DESC LIMIT 1`, tableID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// GenerateAccountForOrder crea la cuenta de un pedido sin mesa (para llevar o
// a domicilio) con todas sus partidas pendientes de cobro.
func (s *Store) GenerateAccountForOrder(orderID int) (Account, error) {
	var orderType string
	var tableID sql.NullInt64
	if err := s.db.QueryRow(`SELECT type, table_id FROM orders WHERE id = ?`, orderID).Scan(&orderType, &tableID); err != nil {
		if err == sql.ErrNoRows {
			return Account{}, ErrNoItems
		}
		return Account{}, err
	}
	rows, err := s.db.Query(`SELECT id FROM order_items WHERE order_id = ? AND account_id IS NULL AND status != 'cancelado'`, orderID)
	if err != nil {
		return Account{}, err
	}
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return Account{}, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		return Account{}, ErrNoItems
	}
	var tid *int
	if tableID.Valid {
		v := int(tableID.Int64)
		tid = &v
	}
	accountID, err := s.createAccount(tid, "completa")
	if err != nil {
		return Account{}, err
	}
	if err := s.assignItems(accountID, ids); err != nil {
		return Account{}, err
	}
	return s.GetAccount(accountID)
}

// OpenAccountForOrder devuelve la cuenta abierta ligada a las partidas del pedido, o 0.
func (s *Store) OpenAccountForOrder(orderID int) (int, error) {
	var id int
	err := s.db.QueryRow(`
		SELECT a.id FROM accounts a
		WHERE a.status = 'abierta' AND EXISTS (SELECT 1 FROM order_items oi WHERE oi.account_id = a.id AND oi.order_id = ?)
		ORDER BY a.id DESC LIMIT 1`, orderID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// closeSettledOrders cierra las comandas de la cuenta cuando ya no les queda
// ninguna partida viva sin cobrar. Así una mesa que se vuelve a abrir empieza
// con comanda nueva y los pedidos para llevar salen de la lista de pendientes.
func (s *Store) closeSettledOrders(accountID int) error {
	_, err := s.db.Exec(`
		UPDATE orders SET status = 'cerrada', closed_at = datetime('now')
		WHERE status = 'abierta'
		  AND id IN (SELECT DISTINCT order_id FROM order_items WHERE account_id = ?)
		  AND NOT EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = orders.id AND oi.account_id IS NULL AND oi.status != 'cancelado')`, accountID)
	return err
}
