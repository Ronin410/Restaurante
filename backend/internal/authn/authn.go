// Package authn implementa autenticación (JWT + bcrypt) y autorización por
// rol para todo el sistema. Cada empleado inicia sesión individualmente
// (requisito de seguridad de las especificaciones) y el token JWT lleva su
// id y rol para que el middleware de roles decida qué endpoints puede usar.
package authn

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleGerente  Role = "gerente"
	RoleCajero   Role = "cajero"
	RoleMesero   Role = "mesero"
	RoleCocina   Role = "cocina"
	RoleContador Role = "contador"
)

type Claims struct {
	UserID int    `json:"uid"`
	Name   string `json:"name"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

type ctxKey int

const claimsKey ctxKey = iota

var ErrInvalidToken = errors.New("token inválido o expirado")

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func (m *Manager) Issue(userID int, name string, role Role) (string, time.Time, error) {
	expires := time.Now().Add(m.ttl)
	claims := Claims{
		UserID: userID,
		Name:   name,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, expires, err
}

func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// Middleware extrae y valida el Bearer token, poniendo los Claims en el
// contexto de la petición.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, `{"error":"falta encabezado de autorización"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := m.Parse(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"token inválido o expirado"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext regresa las Claims del usuario autenticado en la petición.
func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

// RequireRole regresa un middleware que solo permite continuar a los roles
// indicados. Debe usarse después de Middleware.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	allowed := make(map[Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
				return
			}
			if !allowed[claims.Role] {
				http.Error(w, `{"error":"no autorizado para este recurso"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
