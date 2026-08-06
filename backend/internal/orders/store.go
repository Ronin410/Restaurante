package orders

import (
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("comanda no encontrada")
var ErrItemNotFound = errors.New("partida no encontrada")
var ErrProductNotFound = errors.New("producto no encontrado")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type productInfo struct {
	Name  string
	Price float64
}

func (s *Store) productInfo(id int) (productInfo, error) {
	var p productInfo
	err := s.db.QueryRow(`SELECT name, price FROM products WHERE id = ? AND active = 1`, id).Scan(&p.Name, &p.Price)
	if err == sql.ErrNoRows {
		return p, ErrProductNotFound
	}
	return p, err
}

type modifierInfo struct {
	Name       string
	PriceDelta float64
}

func (s *Store) modifierInfo(id int) (modifierInfo, error) {
	var m modifierInfo
	err := s.db.QueryRow(`SELECT name, price_delta FROM modifier_options WHERE id = ?`, id).Scan(&m.Name, &m.PriceDelta)
	return m, err
}

func (s *Store) CreateOrder(tableID *int, orderType string, waiterID int, customerName, customerPhone, customerAddress string) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO orders (table_id, type, waiter_id, customer_name, customer_phone, customer_address) VALUES (?, ?, ?, ?, ?, ?)`,
		tableID, orderType, waiterID, customerName, customerPhone, customerAddress,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *Store) MaxTanda(orderID int) (int, error) {
	var max sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(tanda) FROM order_items WHERE order_id = ?`, orderID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64), nil
}

// AddItem inserta una partida resolviendo nombre/precio del producto y de
// cada modificador seleccionado (guardando "snapshot" para que cambios de
// precio futuros no alteren comandas ya capturadas).
func (s *Store) AddItem(orderID, tanda int, in NewItemInput) (OrderItem, error) {
	product, err := s.productInfo(in.ProductID)
	if err != nil {
		return OrderItem{}, err
	}
	qty := in.Qty
	if qty <= 0 {
		qty = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO order_items (order_id, product_id, product_name_snapshot, qty, unit_price, notes, tanda, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'enviado')`,
		orderID, in.ProductID, product.Name, qty, product.Price, in.Notes, tanda,
	)
	if err != nil {
		return OrderItem{}, err
	}
	itemID, _ := res.LastInsertId()

	for _, optID := range in.ModifierOptionIDs {
		mod, err := s.modifierInfo(optID)
		if err != nil {
			return OrderItem{}, err
		}
		if _, err := s.db.Exec(
			`INSERT INTO order_item_modifiers (order_item_id, modifier_option_id, name_snapshot, price_delta_snapshot) VALUES (?, ?, ?, ?)`,
			itemID, optID, mod.Name, mod.PriceDelta,
		); err != nil {
			return OrderItem{}, err
		}
	}
	return s.GetItem(int(itemID))
}

func (s *Store) GetItem(id int) (OrderItem, error) {
	row := s.db.QueryRow(
		`SELECT id, order_id, product_id, product_name_snapshot, qty, unit_price, notes, tanda, status, account_id,
		        sent_at, ready_at, delivered_at, cancelled_reason, cancelled_by
		 FROM order_items WHERE id = ?`, id)
	var it OrderItem
	err := row.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Qty, &it.UnitPrice, &it.Notes, &it.Tanda,
		&it.Status, &it.AccountID, &it.SentAt, &it.ReadyAt, &it.DeliveredAt, &it.CancelledReason, &it.CancelledBy)
	if err == sql.ErrNoRows {
		return it, ErrItemNotFound
	}
	if err != nil {
		return it, err
	}
	mods, err := s.itemModifiers(it.ID)
	if err != nil {
		return it, err
	}
	it.Modifiers = mods
	it.LineTotal = lineTotal(it)
	return it, nil
}

func (s *Store) itemModifiers(itemID int) ([]ItemModifier, error) {
	rows, err := s.db.Query(`SELECT id, name_snapshot, price_delta_snapshot FROM order_item_modifiers WHERE order_item_id = ?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ItemModifier
	for rows.Next() {
		var m ItemModifier
		if err := rows.Scan(&m.ID, &m.Name, &m.PriceDelta); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func lineTotal(it OrderItem) float64 {
	total := it.UnitPrice
	for _, m := range it.Modifiers {
		total += m.PriceDelta
	}
	return total * it.Qty
}

func (s *Store) GetOrder(id int) (Order, error) {
	row := s.db.QueryRow(`
		SELECT o.id, o.table_id, t.number, o.type, o.waiter_id, u.name, o.status,
		       o.customer_name, o.customer_phone, o.customer_address, o.opened_at, o.closed_at
		FROM orders o
		LEFT JOIN tables_ t ON t.id = o.table_id
		JOIN users u ON u.id = o.waiter_id
		WHERE o.id = ?`, id)
	var o Order
	err := row.Scan(&o.ID, &o.TableID, &o.TableNumber, &o.Type, &o.WaiterID, &o.WaiterName, &o.Status,
		&o.CustomerName, &o.CustomerPhone, &o.CustomerAddress, &o.OpenedAt, &o.ClosedAt)
	if err == sql.ErrNoRows {
		return o, ErrNotFound
	}
	if err != nil {
		return o, err
	}
	items, err := s.listOrderItems(id)
	if err != nil {
		return o, err
	}
	o.Items = items
	for _, it := range items {
		if it.Status != "cancelado" {
			o.Total += it.LineTotal
		}
	}
	return o, nil
}

func (s *Store) listOrderItems(orderID int) ([]OrderItem, error) {
	rows, err := s.db.Query(
		`SELECT id, order_id, product_id, product_name_snapshot, qty, unit_price, notes, tanda, status, account_id,
		        sent_at, ready_at, delivered_at, cancelled_reason, cancelled_by
		 FROM order_items WHERE order_id = ? ORDER BY tanda, id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderItem
	// Importante: se recolectan primero todas las partidas y se cierra el
	// cursor antes de disparar las consultas anidadas de modificadores; el
	// pool de conexiones de SQLite es de una sola conexión, así que anidar
	// una consulta dentro de este mismo for (con "rows" aún abierto)
	// provoca un interbloqueo (la conexión única queda ocupada por "rows").
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Qty, &it.UnitPrice, &it.Notes, &it.Tanda,
			&it.Status, &it.AccountID, &it.SentAt, &it.ReadyAt, &it.DeliveredAt, &it.CancelledReason, &it.CancelledBy); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		mods, err := s.itemModifiers(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Modifiers = mods
		out[i].LineTotal = lineTotal(out[i])
	}
	return out, nil
}

func (s *Store) ListOpenOrders(tableID *int) ([]Order, error) {
	query := `SELECT o.id FROM orders o WHERE o.status = 'abierta'`
	args := []interface{}{}
	if tableID != nil {
		query += ` AND o.table_id = ?`
		args = append(args, *tableID)
	}
	query += ` ORDER BY o.opened_at`
	rows, err := s.db.Query(query, args...)
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

	out := make([]Order, 0, len(ids))
	for _, id := range ids {
		o, err := s.GetOrder(id)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

// ListKitchenItems regresa las partidas pendientes (enviado/preparando) para
// el KDS, con contexto de mesa/tipo de pedido.
func (s *Store) ListKitchenItems() ([]OrderItem, error) {
	rows, err := s.db.Query(`
		SELECT oi.id, oi.order_id, oi.product_id, oi.product_name_snapshot, oi.qty, oi.unit_price, oi.notes, oi.tanda,
		       oi.status, oi.account_id, oi.sent_at, oi.ready_at, oi.delivered_at, oi.cancelled_reason, oi.cancelled_by,
		       COALESCE(t.number, ''), o.type
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		LEFT JOIN tables_ t ON t.id = o.table_id
		WHERE oi.status IN ('enviado','preparando')
		ORDER BY oi.sent_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderItem
	// Ver nota en listOrderItems: recolectar antes de anidar consultas.
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Qty, &it.UnitPrice, &it.Notes, &it.Tanda,
			&it.Status, &it.AccountID, &it.SentAt, &it.ReadyAt, &it.DeliveredAt, &it.CancelledReason, &it.CancelledBy,
			&it.TableNumber, &it.OrderType); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		mods, err := s.itemModifiers(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Modifiers = mods
		out[i].LineTotal = lineTotal(out[i])
	}
	return out, nil
}

func (s *Store) UpdateItemStatus(id int, status string) (OrderItem, error) {
	switch status {
	case "listo":
		_, err := s.db.Exec(`UPDATE order_items SET status = ?, ready_at = datetime('now') WHERE id = ?`, status, id)
		if err != nil {
			return OrderItem{}, err
		}
	case "entregado":
		_, err := s.db.Exec(`UPDATE order_items SET status = ?, delivered_at = datetime('now') WHERE id = ?`, status, id)
		if err != nil {
			return OrderItem{}, err
		}
	default:
		_, err := s.db.Exec(`UPDATE order_items SET status = ? WHERE id = ?`, status, id)
		if err != nil {
			return OrderItem{}, err
		}
	}
	return s.GetItem(id)
}

func (s *Store) CancelItem(id int, reason string, cancelledBy int) (OrderItem, error) {
	_, err := s.db.Exec(
		`UPDATE order_items SET status = 'cancelado', cancelled_reason = ?, cancelled_by = ? WHERE id = ?`,
		reason, cancelledBy, id,
	)
	if err != nil {
		return OrderItem{}, err
	}
	return s.GetItem(id)
}

func (s *Store) CloseOrder(id int) error {
	_, err := s.db.Exec(`UPDATE orders SET status = 'cerrada', closed_at = datetime('now') WHERE id = ?`, id)
	return err
}
