package employees

import (
	"database/sql"

	"errors"
	"github.com/ronin410/restaurante/backend/internal/dbx"

	"github.com/ronin410/restaurante/backend/internal/authn"
)

var ErrNotFound = errors.New("empleado no encontrado")
var ErrHasHistory = errors.New("el empleado ya tiene historial (comandas, pagos, cortes…) y no se puede eliminar; dalo de baja")
var ErrEmailTaken = errors.New("el correo ya está registrado")

type Store struct {
	db *dbx.DB
}

func NewStore(db *dbx.DB) *Store {
	return &Store{db: db}
}

type userRow struct {
	ID           int
	Name         string
	Email        string
	PasswordHash string
	Role         string
	Position     string
	Salary       float64
	HireDate     string
	Active       bool
}

func scanUser(row rowScanner) (userRow, error) {
	var u userRow
	var activeInt int
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.Position, &u.Salary, &u.HireDate, &activeInt)
	u.Active = activeInt == 1
	return u, err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func toPublic(u userRow) User {
	return User{
		ID:       u.ID,
		Name:     u.Name,
		Email:    u.Email,
		Role:     authn.Role(u.Role),
		Position: u.Position,
		Salary:   u.Salary,
		HireDate: u.HireDate,
		Active:   u.Active,
	}
}

const userCols = `id, name, email, password_hash, role, position, salary, hire_date, active`

func (s *Store) GetByEmail(email string) (userRow, error) {
	row := s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE email = ?`, email)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) GetByID(id int) (userRow, error) {
	row := s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) Create(in CreateUserInput, passwordHash string) (User, error) {
	existing, err := s.GetByEmail(in.Email)
	if err == nil && existing.ID != 0 {
		return User{}, ErrEmailTaken
	}
	var id int
	err = s.db.QueryRow(
		`INSERT INTO users (name, email, password_hash, role, position, salary, hire_date, active) VALUES (?, ?, ?, ?, ?, ?, ?, 1) RETURNING id`,
		in.Name, in.Email, passwordHash, string(in.Role), in.Position, in.Salary, in.HireDate,
	).Scan(&id)
	if err != nil {
		return User{}, err
	}
	row, err := s.GetByID(id)
	if err != nil {
		return User{}, err
	}
	return toPublic(row), nil
}

func (s *Store) List(activeOnly bool) ([]User, error) {
	query := `SELECT ` + userCols + ` FROM users`
	if activeOnly {
		query += ` WHERE active = 1`
	}
	query += ` ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, toPublic(u))
	}
	return out, rows.Err()
}

func (s *Store) Update(id int, in UpdateUserInput) (User, error) {
	activeInt := 0
	if in.Active {
		activeInt = 1
	}
	if _, err := s.GetByID(id); err != nil {
		return User{}, err
	}
	_, err := s.db.Exec(
		`UPDATE users SET name = ?, role = ?, position = ?, salary = ?, active = ?, updated_at = datetime('now') WHERE id = ?`,
		in.Name, string(in.Role), in.Position, in.Salary, activeInt, id,
	)
	if err != nil {
		return User{}, err
	}
	if in.Email != "" {
		var other int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ? AND id != ?`, in.Email, id).Scan(&other); err != nil {
			return User{}, err
		}
		if other > 0 {
			return User{}, ErrEmailTaken
		}
		if _, err := s.db.Exec(`UPDATE users SET email = ? WHERE id = ?`, in.Email, id); err != nil {
			return User{}, err
		}
	}
	if in.HireDate != "" {
		if _, err := s.db.Exec(`UPDATE users SET hire_date = ? WHERE id = ?`, in.HireDate, id); err != nil {
			return User{}, err
		}
	}
	if in.Password != "" {
		hash, err := authn.HashPassword(in.Password)
		if err != nil {
			return User{}, err
		}
		if _, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id); err != nil {
			return User{}, err
		}
	}
	row, err := s.GetByID(id)
	if err != nil {
		return User{}, err
	}
	return toPublic(row), nil
}

// ---- Asistencia ----

func (s *Store) ClockIn(userID int) (Attendance, error) {
	var id int
	err := s.db.QueryRow(`INSERT INTO attendance (user_id, clock_in) VALUES (?, datetime('now')) RETURNING id`, userID).Scan(&id)
	if err != nil {
		return Attendance{}, err
	}
	return s.getAttendance(id)
}

func (s *Store) ClockOut(attendanceID int) (Attendance, error) {
	_, err := s.db.Exec(`UPDATE attendance SET clock_out = datetime('now') WHERE id = ? AND clock_out IS NULL`, attendanceID)
	if err != nil {
		return Attendance{}, err
	}
	return s.getAttendance(attendanceID)
}

func (s *Store) OpenAttendanceForUser(userID int) (Attendance, error) {
	row := s.db.QueryRow(`SELECT id, user_id, clock_in, clock_out, source FROM attendance WHERE user_id = ? AND clock_out IS NULL ORDER BY id DESC LIMIT 1`, userID)
	var a Attendance
	err := row.Scan(&a.ID, &a.UserID, &a.ClockIn, &a.ClockOut, &a.Source)
	if err == sql.ErrNoRows {
		return Attendance{}, ErrNotFound
	}
	return a, err
}

func (s *Store) getAttendance(id int) (Attendance, error) {
	row := s.db.QueryRow(`SELECT id, user_id, clock_in, clock_out, source FROM attendance WHERE id = ?`, id)
	var a Attendance
	err := row.Scan(&a.ID, &a.UserID, &a.ClockIn, &a.ClockOut, &a.Source)
	return a, err
}

func (s *Store) ListAttendance(userID int, from, to string) ([]Attendance, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, clock_in, clock_out, source FROM attendance
		 WHERE user_id = ? AND date(clock_in) BETWEEN ? AND ? ORDER BY clock_in DESC`,
		userID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attendance
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(&a.ID, &a.UserID, &a.ClockIn, &a.ClockOut, &a.Source); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---- Incidencias ----

func (s *Store) CreateIncident(userID, createdBy int, in CreateIncidentInput) (Incident, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO incidents (user_id, type, date, note, created_by) VALUES (?, ?, ?, ?, ?) RETURNING id`,
		userID, in.Type, in.Date, in.Note, createdBy,
	).Scan(&id)
	if err != nil {
		return Incident{}, err
	}
	return s.getIncident(id)
}

func (s *Store) getIncident(id int) (Incident, error) {
	row := s.db.QueryRow(`SELECT id, user_id, type, date, note, created_by, created_at FROM incidents WHERE id = ?`, id)
	var i Incident
	err := row.Scan(&i.ID, &i.UserID, &i.Type, &i.Date, &i.Note, &i.CreatedBy, &i.CreatedAt)
	return i, err
}

func (s *Store) ListIncidents(userID int) ([]Incident, error) {
	rows, err := s.db.Query(`SELECT id, user_id, type, date, note, created_by, created_at FROM incidents WHERE user_id = ? ORDER BY date DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Incident
	for rows.Next() {
		var i Incident
		if err := rows.Scan(&i.ID, &i.UserID, &i.Type, &i.Date, &i.Note, &i.CreatedBy, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// Delete elimina al empleado por completo (junto con su checador e
// incidencias) en una transacción. Si ya tiene historial operativo (comandas,
// pagos, cortes, auditoría…) la base lo impide: se devuelve ErrHasHistory y
// no se pierde nada; en ese caso lo correcto es darlo de baja.
func (s *Store) Delete(id int) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	tx, err := s.db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM attendance WHERE user_id = $1`,
		`DELETE FROM incidents WHERE user_id = $1`,
		`DELETE FROM users WHERE id = $1`,
	} {
		if _, err := tx.Exec(q, id); err != nil {
			return ErrHasHistory
		}
	}
	return tx.Commit()
}
