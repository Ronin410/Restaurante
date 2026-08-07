package tables

import (
	"database/sql"

	"errors"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

var ErrNotFound = errors.New("mesa no encontrada")

type Store struct {
	db *dbx.DB
}

func NewStore(db *dbx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListZones() ([]Zone, error) {
	rows, err := s.db.Query(`SELECT id, name FROM zones ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Zone
	for rows.Next() {
		var z Zone
		if err := rows.Scan(&z.ID, &z.Name); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func (s *Store) CreateZone(in ZoneInput) (Zone, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO zones (name) VALUES (?) RETURNING id`, in.Name).Scan(&id)
	if err != nil {
		return Zone{}, err
	}
	return Zone{ID: id, Name: in.Name}, nil
}

const tableCols = `t.id, t.zone_id, z.name, t.number, t.capacity, t.status, t.current_waiter_id, u.name, t.opened_at, t.joined_with`

func (s *Store) scanTable(row interface{ Scan(...interface{}) error }) (Table, error) {
	var t Table
	var waiterName *string
	if err := row.Scan(&t.ID, &t.ZoneID, &t.ZoneName, &t.Number, &t.Capacity, &t.Status, &t.CurrentWaiterID, &waiterName, &t.OpenedAt, &t.JoinedWith); err != nil {
		return t, err
	}
	t.CurrentWaiterName = waiterName
	return t, nil
}

func (s *Store) ListTables() ([]Table, error) {
	rows, err := s.db.Query(`SELECT ` + tableCols + ` FROM tables_ t JOIN zones z ON z.id = t.zone_id LEFT JOIN users u ON u.id = t.current_waiter_id ORDER BY z.name, t.number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Table
	for rows.Next() {
		t, err := s.scanTable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTable(id int) (Table, error) {
	row := s.db.QueryRow(`SELECT `+tableCols+` FROM tables_ t JOIN zones z ON z.id = t.zone_id LEFT JOIN users u ON u.id = t.current_waiter_id WHERE t.id = ?`, id)
	t, err := s.scanTable(row)
	if err == sql.ErrNoRows {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) CreateTable(in TableInput) (Table, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO tables_ (zone_id, number, capacity, status) VALUES (?, ?, ?, 'libre') RETURNING id`,
		in.ZoneID, in.Number, in.Capacity).Scan(&id)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}

func (s *Store) UpdateTable(id int, in TableInput) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET zone_id = ?, number = ?, capacity = ? WHERE id = ?`, in.ZoneID, in.Number, in.Capacity, id)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}

func (s *Store) Open(id, waiterID int) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET status = 'ocupada', current_waiter_id = ?, opened_at = datetime('now') WHERE id = ?`, waiterID, id)
	if err != nil {
		return Table{}, err
	}
	if _, err := s.db.Exec(`INSERT INTO table_sessions (table_id, opened_at) VALUES (?, datetime('now'))`, id); err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}

func (s *Store) SetStatus(id int, status string) (Table, error) {
	if status == "libre" {
		_, err := s.db.Exec(`UPDATE tables_ SET status = 'libre', current_waiter_id = NULL, opened_at = NULL, joined_with = NULL WHERE id = ?`, id)
		if err != nil {
			return Table{}, err
		}
		// Cierra la sesión de ocupación abierta más reciente para poder
		// calcular la rotación de mesas (KPIs).
		if _, err := s.db.Exec(
			`UPDATE table_sessions SET closed_at = datetime('now')
			 WHERE table_id = ? AND closed_at IS NULL
			 AND id = (SELECT id FROM table_sessions WHERE table_id = ? AND closed_at IS NULL ORDER BY id DESC LIMIT 1)`,
			id, id,
		); err != nil {
			return Table{}, err
		}
		return s.GetTable(id)
	}
	_, err := s.db.Exec(`UPDATE tables_ SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}

func (s *Store) AssignWaiter(id, waiterID int) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET current_waiter_id = ? WHERE id = ?`, waiterID, id)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}

func (s *Store) Join(tableID, withTableID int) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET joined_with = ? WHERE id = ?`, tableID, withTableID)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(withTableID)
}

func (s *Store) Unjoin(tableID int) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET joined_with = NULL WHERE id = ?`, tableID)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(tableID)
}

func (s *Store) Transfer(tableID int, fromWaiterID *int, toWaiterID, authorizedBy int) (TransferRecord, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO table_transfers (table_id, from_waiter_id, to_waiter_id, authorized_by) VALUES (?, ?, ?, ?) RETURNING id`,
		tableID, fromWaiterID, toWaiterID, authorizedBy,
	).Scan(&id)
	if err != nil {
		return TransferRecord{}, err
	}
	if _, err := s.db.Exec(`UPDATE tables_ SET current_waiter_id = ? WHERE id = ?`, toWaiterID, tableID); err != nil {
		return TransferRecord{}, err
	}
	row := s.db.QueryRow(`SELECT id, table_id, from_waiter_id, to_waiter_id, authorized_by, created_at FROM table_transfers WHERE id = ?`, id)
	var tr TransferRecord
	err = row.Scan(&tr.ID, &tr.TableID, &tr.FromWaiterID, &tr.ToWaiterID, &tr.AuthorizedBy, &tr.CreatedAt)
	return tr, err
}
