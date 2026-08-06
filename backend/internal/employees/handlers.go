package employees

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ronin410/restaurante/backend/internal/audit"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/httpx"
)

type Handler struct {
	svc   *Service
	audit *audit.Logger
}

func NewHandler(svc *Service, audit *audit.Logger) *Handler {
	return &Handler{svc: svc, audit: audit}
}

// RegisterPublicRoutes monta rutas que no requieren token (login).
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.login)
}

// RegisterRoutes monta el resto de rutas; se asume que r ya exige un token
// válido (ver authn.Middleware montado en el router padre).
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/auth/me", h.me)

	staff := authn.RequireRole(authn.RoleAdmin, authn.RoleGerente)

	r.Route("/employees", func(r chi.Router) {
		r.With(staff).Get("/", h.list)
		r.With(staff).Post("/", h.create)
		r.Get("/{id}", h.get) // cualquier usuario autenticado puede ver su propio perfil vía /auth/me; detalle completo restringido en el front
		r.With(staff).Put("/{id}", h.update)

		r.Post("/{id}/attendance/clock-in", h.clockIn)
		r.Post("/{id}/attendance/clock-out", h.clockOut)
		r.Get("/{id}/attendance", h.listAttendance)

		r.With(staff).Post("/{id}/incidents", h.createIncident)
		r.With(staff).Get("/{id}/incidents", h.listIncidents)
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	result, err := h.svc.Login(in.Email, in.Password)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	h.audit.Log(result.User.ID, "login", "user", result.User.ID, "")
	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, _ := authn.FromContext(r.Context())
	user, err := h.svc.GetEmployee(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "usuario no encontrado")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListEmployees(r.URL.Query().Get("active") == "1")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, users)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateUserInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	user, err := h.svc.CreateEmployee(in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "create", "employee", user.ID, user.Name)
	httpx.JSON(w, http.StatusCreated, user)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	user, err := h.svc.GetEmployee(id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "empleado no encontrado")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in UpdateUserInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	user, err := h.svc.UpdateEmployee(id, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	claims, _ := authn.FromContext(r.Context())
	h.audit.Log(claims.UserID, "update", "employee", id, "")
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) clockIn(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	att, err := h.svc.ClockIn(id)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, att)
}

func (h *Handler) clockOut(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	att, err := h.svc.ClockOut(id)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, att)
}

func (h *Handler) listAttendance(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.svc.ListAttendance(id, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) createIncident(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	var in CreateIncidentInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	claims, _ := authn.FromContext(r.Context())
	inc, err := h.svc.CreateIncident(id, claims.UserID, in)
	if err != nil {
		writeServiceErr(w, err)
		return
	}
	h.audit.Log(claims.UserID, "create", "incident", inc.ID, in.Type)
	httpx.JSON(w, http.StatusCreated, inc)
}

func (h *Handler) listIncidents(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.svc.ListIncidents(id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func idParam(r *http.Request) (int, error) {
	return strconv.Atoi(chi.URLParam(r, "id"))
}

func writeServiceErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	}
}
