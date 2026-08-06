// Package audit registra en bitácora las acciones sensibles del sistema
// (cancelaciones, descuentos, autorizaciones, cortes de caja) para cumplir
// el requerimiento de seguridad de trazabilidad ("quién canceló, quién
// descontó, etc.").
package audit

import "database/sql"

type Logger struct {
	db *sql.DB
}

func New(db *sql.DB) *Logger {
	return &Logger{db: db}
}

func (l *Logger) Log(userID int, action, entity string, entityID int, detail string) {
	// El registro de auditoría no debe tumbar la operación principal si
	// falla; se hace best-effort.
	_, _ = l.db.Exec(
		`INSERT INTO audit_logs (user_id, action, entity, entity_id, detail) VALUES (?, ?, ?, ?, ?)`,
		userID, action, entity, entityID, detail,
	)
}

type Entry struct {
	ID        int     `json:"id"`
	UserID    *int    `json:"user_id"`
	UserName  *string `json:"user_name"`
	Action    string  `json:"action"`
	Entity    string  `json:"entity"`
	EntityID  *int    `json:"entity_id"`
	Detail    string  `json:"detail"`
	CreatedAt string  `json:"created_at"`
}

func (l *Logger) List(limit int, entity string) ([]Entry, error) {
	query := `SELECT a.id, a.user_id, u.name, a.action, a.entity, a.entity_id, a.detail, a.created_at
	          FROM audit_logs a LEFT JOIN users u ON u.id = a.user_id`
	args := []interface{}{}
	if entity != "" {
		query += ` WHERE a.entity = ?`
		args = append(args, entity)
	}
	query += ` ORDER BY a.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.Action, &e.Entity, &e.EntityID, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
