package employees

import "github.com/ronin410/restaurante/backend/internal/authn"

type User struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Email    string     `json:"email"`
	Role     authn.Role `json:"role"`
	Position string     `json:"position"`
	Salary   float64    `json:"salary"`
	HireDate string     `json:"hire_date"`
	Active   bool       `json:"active"`
}

type CreateUserInput struct {
	Name     string     `json:"name"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Role     authn.Role `json:"role"`
	Position string     `json:"position"`
	Salary   float64    `json:"salary"`
	HireDate string     `json:"hire_date"`
}

type UpdateUserInput struct {
	Name     string     `json:"name"`
	Email    string     `json:"email"`    // opcional: vacío = no cambia
	Password string     `json:"password"` // opcional: vacío = no cambia
	HireDate string     `json:"hire_date"`
	Role     authn.Role `json:"role"`
	Position string     `json:"position"`
	Salary   float64    `json:"salary"`
	Active   bool       `json:"active"`
}

type Attendance struct {
	ID       int     `json:"id"`
	UserID   int     `json:"user_id"`
	ClockIn  string  `json:"clock_in"`
	ClockOut *string `json:"clock_out"`
	Source   string  `json:"source"`
}

type Incident struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Type      string `json:"type"` // falta | retardo | permiso
	Date      string `json:"date"`
	Note      string `json:"note"`
	CreatedBy int    `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type CreateIncidentInput struct {
	Type string `json:"type"`
	Date string `json:"date"`
	Note string `json:"note"`
}
