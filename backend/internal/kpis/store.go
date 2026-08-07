package kpis

import (
	"database/sql"

	"github.com/ronin410/restaurante/backend/internal/dbx"
)

type Store struct {
	db *dbx.DB
}

func NewStore(db *dbx.DB) *Store {
	return &Store{db: db}
}

const lineTotalExpr = `oi.qty * (oi.unit_price + COALESCE((
	SELECT SUM(oim.price_delta_snapshot) FROM order_item_modifiers oim WHERE oim.order_item_id = oi.id
), 0))`

const closedAccountsFilter = `a.status = 'cerrada' AND date(a.closed_at) BETWEEN ? AND ?`

// VentaTotal usa accounts.total (neto de descuentos/cortesías) para que
// cuadre con VentaPorMetodo, que se basa en pagos realmente cobrados.
func (s *Store) VentaTotal(from, to string) (float64, error) {
	var total float64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(a.total),0) FROM accounts a WHERE `+closedAccountsFilter, from, to).Scan(&total)
	return total, err
}

func (s *Store) VentaPorMetodo(from, to string) ([]MethodTotal, error) {
	rows, err := s.db.Query(`
		SELECT p.method, COALESCE(SUM(p.amount),0)
		FROM payments p JOIN accounts a ON a.id = p.account_id
		WHERE `+closedAccountsFilter+`
		GROUP BY p.method ORDER BY 2 DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MethodTotal
	for rows.Next() {
		var m MethodTotal
		if err := rows.Scan(&m.Method, &m.Total); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) TicketPromedio(from, to string) (float64, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRow(`SELECT AVG(a.total) FROM accounts a WHERE `+closedAccountsFilter, from, to).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

func (s *Store) ProductoTop(from, to string, asc bool) (*ProductRank, error) {
	order := "DESC"
	if asc {
		order = "ASC"
	}
	row := s.db.QueryRow(`
		SELECT oi.product_name_snapshot, SUM(oi.qty) AS q
		FROM order_items oi JOIN accounts a ON a.id = oi.account_id
		WHERE oi.status != 'cancelado' AND `+closedAccountsFilter+`
		GROUP BY oi.product_name_snapshot ORDER BY q `+order+` LIMIT 1`, from, to)
	var p ProductRank
	err := row.Scan(&p.ProductName, &p.Qty)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// RotacionMinutosPromedio calcula el tiempo promedio de ocupación de mesas
// (sesiones cerradas dentro del rango).
func (s *Store) RotacionMinutosPromedio(from, to string) (float64, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT AVG((strftime('%s', closed_at) - strftime('%s', opened_at)) / 60.0)
		FROM table_sessions
		WHERE closed_at IS NOT NULL AND date(opened_at) BETWEEN ? AND ?`, from, to).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

func (s *Store) PropinasTotal(from, to string) (float64, error) {
	var total float64
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(p.tip_amount),0)
		FROM payments p JOIN accounts a ON a.id = p.account_id
		WHERE `+closedAccountsFilter, from, to).Scan(&total)
	return total, err
}

// PropinasPorMesero distribuye la propina de cada cuenta proporcionalmente
// entre los meseros que atendieron las partidas de esa cuenta (lo exacto en
// el caso, mayoritario, de un solo mesero por cuenta).
func (s *Store) PropinasPorMesero(from, to string) ([]WaiterTip, error) {
	rows, err := s.db.Query(`
		WITH acc_tips AS (
			SELECT a.id AS account_id, COALESCE(SUM(p.tip_amount),0) AS tip_total
			FROM accounts a JOIN payments p ON p.account_id = a.id
			WHERE `+closedAccountsFilter+`
			GROUP BY a.id
		),
		waiter_shares AS (
			SELECT oi.account_id, o.waiter_id, u.name AS waiter_name, SUM(`+lineTotalExpr+`) AS waiter_total
			FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			JOIN users u ON u.id = o.waiter_id
			WHERE oi.account_id IN (SELECT account_id FROM acc_tips) AND oi.status != 'cancelado'
			GROUP BY oi.account_id, o.waiter_id, u.name
		),
		account_totals AS (
			SELECT account_id, SUM(waiter_total) AS total FROM waiter_shares GROUP BY account_id
		)
		SELECT ws.waiter_id, ws.waiter_name, SUM(
			CASE WHEN act.total > 0 THEN at2.tip_total * ws.waiter_total / act.total ELSE 0 END
		) AS tip_share
		FROM waiter_shares ws
		JOIN acc_tips at2 ON at2.account_id = ws.account_id
		JOIN account_totals act ON act.account_id = ws.account_id
		GROUP BY ws.waiter_id, ws.waiter_name
		ORDER BY tip_share DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WaiterTip
	for rows.Next() {
		var w WaiterTip
		if err := rows.Scan(&w.WaiterID, &w.WaiterName, &w.Tips); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) VentaPorMesero(from, to string) ([]WaiterTotal, error) {
	rows, err := s.db.Query(`
		SELECT o.waiter_id, u.name, COALESCE(SUM(`+lineTotalExpr+`),0) AS total
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN users u ON u.id = o.waiter_id
		JOIN accounts a ON a.id = oi.account_id
		WHERE oi.status != 'cancelado' AND `+closedAccountsFilter+`
		GROUP BY o.waiter_id, u.name ORDER BY total DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WaiterTotal
	for rows.Next() {
		var w WaiterTotal
		if err := rows.Scan(&w.WaiterID, &w.WaiterName, &w.Total); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) VentaPorTurno(from, to string) ([]ShiftSales, error) {
	rows, err := s.db.Query(`
		SELECT s.id, u.name, s.opened_at, COALESCE(SUM(p.amount),0) AS total
		FROM cash_shifts s
		JOIN users u ON u.id = s.cashier_id
		LEFT JOIN payments p ON p.shift_id = s.id
		WHERE date(s.opened_at) BETWEEN ? AND ?
		GROUP BY s.id, u.name, s.opened_at ORDER BY s.opened_at DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ShiftSales
	for rows.Next() {
		var sh ShiftSales
		if err := rows.Scan(&sh.ShiftID, &sh.CashierName, &sh.OpenedAt, &sh.Total); err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *Store) FaltantesSobrantesAcumulado(from, to string) (float64, error) {
	var total float64
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(difference),0) FROM cash_cuts
		WHERE type = 'z' AND date(created_at) BETWEEN ? AND ?`, from, to).Scan(&total)
	return total, err
}
