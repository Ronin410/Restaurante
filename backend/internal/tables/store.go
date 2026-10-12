package tables

import (
	"database/sql"

	"errors"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

var ErrNotFound = errors.New("mesa no encontrada")
var ErrZoneNotFound = errors.New("zona no encontrada")
var ErrZoneHasTables = errors.New("la zona tiene mesas asociadas; muévelas o elimínalas primero")

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

func (s *Store) UpdateZone(id int, in ZoneInput) (Zone, error) {
	res, err := s.db.Exec(`UPDATE zones SET name = ? WHERE id = ?`, in.Name, id)
	if err != nil {
		return Zone{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Zone{}, ErrZoneNotFound
	}
	return Zone{ID: id, Name: in.Name}, nil
}

// DeleteZone borra la zona de verdad (a diferencia de las mesas, nada más
// las referencia) siempre que ya no tenga ninguna mesa asignada, activa o
// no: si la dejáramos borrar con mesas todavía apuntando a ella por
// zone_id, quedarían huérfanas.
func (s *Store) DeleteZone(id int) error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM tables_ WHERE zone_id = ?`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrZoneHasTables
	}
	res, err := s.db.Exec(`DELETE FROM zones WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrZoneNotFound
	}
	return nil
}

const tableCols = `t.id, t.zone_id, z.name, t.number, t.capacity, t.status, t.active, t.current_waiter_id, u.name, t.opened_at, t.joined_with`

func (s *Store) scanTable(row interface{ Scan(...interface{}) error }) (Table, error) {
	var t Table
	var waiterName *string
	var activeInt int
	if err := row.Scan(&t.ID, &t.ZoneID, &t.ZoneName, &t.Number, &t.Capacity, &t.Status, &activeInt, &t.CurrentWaiterID, &waiterName, &t.OpenedAt, &t.JoinedWith); err != nil {
		return t, err
	}
	t.Active = activeInt == 1
	t.CurrentWaiterName = waiterName
	return t, nil
}

func (s *Store) ListTables(includeInactive bool) ([]Table, error) {
	query := `SELECT ` + tableCols + ` FROM tables_ t JOIN zones z ON z.id = t.zone_id LEFT JOIN users u ON u.id = t.current_waiter_id`
	if !includeInactive {
		query += ` WHERE t.active = 1`
	}
	query += ` ORDER BY z.name, t.number`
	rows, err := s.db.Query(query)
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

// Deactivate "elimina" una mesa sin borrar su historial (ver nota de la
// migración 0002): deja de aparecer en el mapa de mesas y no se puede abrir
// de nuevo hasta reactivarla.
func (s *Store) Deactivate(id int) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}

func (s *Store) Reactivate(id int) (Table, error) {
	_, err := s.db.Exec(`UPDATE tables_ SET active = 1 WHERE id = ?`, id)
	if err != nil {
		return Table{}, err
	}
	return s.GetTable(id)
}
