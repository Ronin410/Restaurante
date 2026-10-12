package employees

import (
	"errors"
	"fmt"
	"time"

	"github.com/ronin410/restaurante/backend/internal/authn"
)

var ErrInvalidCredentials = errors.New("credenciales inválidas")
var ErrInactiveUser = errors.New("usuario inactivo")
var ErrValidation = errors.New("datos inválidos")

var validRoles = map[authn.Role]bool{
	authn.RoleAdmin: true, authn.RoleGerente: true, authn.RoleCajero: true,
	authn.RoleMesero: true, authn.RoleCocina: true, authn.RoleContador: true,
}

var validIncidentTypes = map[string]bool{"falta": true, "retardo": true, "permiso": true}

type Service struct {
	store *Store
	auth  *authn.Manager
}

func NewService(store *Store, auth *authn.Manager) *Service {
	return &Service{store: store, auth: auth}
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

func (s *Service) Login(email, password string) (LoginResult, error) {
	row, err := s.store.GetByEmail(email)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !authn.CheckPassword(row.PasswordHash, password) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !row.Active {
		return LoginResult{}, ErrInactiveUser
	}
	token, expires, err := s.auth.Issue(row.ID, row.Name, authn.Role(row.Role))
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: expires, User: toPublic(row)}, nil
}

func (s *Service) CreateEmployee(in CreateUserInput) (User, error) {
	if in.Name == "" || in.Email == "" || in.Password == "" {
		return User{}, fmt.Errorf("%w: nombre, correo y contraseña son obligatorios", ErrValidation)
	}
	if !validRoles[in.Role] {
		return User{}, fmt.Errorf("%w: rol inválido", ErrValidation)
	}
	if len(in.Password) < 6 {
		return User{}, fmt.Errorf("%w: la contraseña debe tener al menos 6 caracteres", ErrValidation)
	}
	if in.HireDate == "" {
		in.HireDate = time.Now().Format("2006-01-02")
	}
	hash, err := authn.HashPassword(in.Password)
	if err != nil {
		return User{}, err
	}
	return s.store.Create(in, hash)
}

func (s *Service) UpdateEmployee(actorID, id int, in UpdateUserInput) (User, error) {
	if actorID == id && !in.Active {
		return User{}, fmt.Errorf("%w: no puedes darte de baja a ti mismo", ErrValidation)
	}
	if in.Password != "" && len(in.Password) < 6 {
		return User{}, fmt.Errorf("%w: la contraseña debe tener al menos 6 caracteres", ErrValidation)
	}
	if in.Name == "" {
		return User{}, fmt.Errorf("%w: nombre obligatorio", ErrValidation)
	}
	if !validRoles[in.Role] {
		return User{}, fmt.Errorf("%w: rol inválido", ErrValidation)
	}
	return s.store.Update(id, in)
}

func (s *Service) DeleteEmployee(actorID, id int) error {
	if actorID == id {
		return fmt.Errorf("%w: no puedes eliminarte a ti mismo", ErrValidation)
	}
	return s.store.Delete(id)
}

func (s *Service) ListEmployees(activeOnly bool) ([]User, error) {
	return s.store.List(activeOnly)
}

func (s *Service) GetEmployee(id int) (User, error) {
	row, err := s.store.GetByID(id)
	if err != nil {
		return User{}, err
	}
	return toPublic(row), nil
}

func (s *Service) ClockIn(userID int) (Attendance, error) {
	if _, err := s.store.OpenAttendanceForUser(userID); err == nil {
		return Attendance{}, fmt.Errorf("%w: ya existe una entrada sin salida registrada", ErrValidation)
	}
	return s.store.ClockIn(userID)
}

func (s *Service) ClockOut(userID int) (Attendance, error) {
	open, err := s.store.OpenAttendanceForUser(userID)
	if err != nil {
		return Attendance{}, fmt.Errorf("%w: no hay una entrada abierta", ErrValidation)
	}
	return s.store.ClockOut(open.ID)
}

func (s *Service) ListAttendance(userID int, from, to string) ([]Attendance, error) {
	if from == "" {
		from = "0000-01-01"
	}
	if to == "" {
		to = "9999-12-31"
	}
	return s.store.ListAttendance(userID, from, to)
}

func (s *Service) CreateIncident(userID, createdBy int, in CreateIncidentInput) (Incident, error) {
	if !validIncidentTypes[in.Type] {
		return Incident{}, fmt.Errorf("%w: tipo de incidencia inválido (falta, retardo o permiso)", ErrValidation)
	}
	if in.Date == "" {
		in.Date = time.Now().Format("2006-01-02")
	}
	return s.store.CreateIncident(userID, createdBy, in)
}

func (s *Service) ListIncidents(userID int) ([]Incident, error) {
	return s.store.ListIncidents(userID)
}
