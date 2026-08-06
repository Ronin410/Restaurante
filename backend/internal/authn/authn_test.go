package authn

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("miClaveSegura123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "miClaveSegura123") {
		t.Fatal("esperaba que la contraseña correcta fuera válida")
	}
	if CheckPassword(hash, "otraClave") {
		t.Fatal("esperaba que una contraseña incorrecta fuera inválida")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	mgr := NewManager("secreto-de-prueba", time.Hour)
	token, expires, err := mgr.Issue(42, "Ada Admin", RoleAdmin)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if time.Until(expires) <= 0 {
		t.Fatal("la expiración debería estar en el futuro")
	}
	claims, err := mgr.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.UserID != 42 || claims.Role != RoleAdmin {
		t.Fatalf("claims inesperadas: %+v", claims)
	}
}

func TestParseRejectsTokenFromDifferentSecret(t *testing.T) {
	mgr1 := NewManager("secreto-1", time.Hour)
	mgr2 := NewManager("secreto-2", time.Hour)
	token, _, _ := mgr1.Issue(1, "x", RoleCajero)
	if _, err := mgr2.Parse(token); err == nil {
		t.Fatal("esperaba error al validar un token firmado con otro secreto")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	mgr := NewManager("secreto-de-prueba", -time.Hour) // ya expirado al emitirse
	token, _, _ := mgr.Issue(1, "x", RoleMesero)
	if _, err := mgr.Parse(token); err == nil {
		t.Fatal("esperaba error al validar un token expirado")
	}
}

func TestMiddlewareAndRequireRole(t *testing.T) {
	mgr := NewManager("secreto-de-prueba", time.Hour)
	token, _, _ := mgr.Issue(7, "Mario Mesero", RoleMesero)

	var reachedInnerHandler bool
	var claimsSeen *Claims
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reachedInnerHandler = true
		claimsSeen, _ = FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	protected := mgr.Middleware(RequireRole(RoleAdmin, RoleGerente)(inner))

	t.Run("sin encabezado de autorización", func(t *testing.T) {
		reachedInnerHandler = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("esperaba 401, obtuve %d", rec.Code)
		}
		if reachedInnerHandler {
			t.Fatal("no debería haber llegado al handler interno")
		}
	})

	t.Run("rol sin permiso", func(t *testing.T) {
		reachedInnerHandler = false
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("esperaba 403, obtuve %d", rec.Code)
		}
		if reachedInnerHandler {
			t.Fatal("no debería haber llegado al handler interno")
		}
	})

	t.Run("rol autorizado", func(t *testing.T) {
		reachedInnerHandler = false
		claimsSeen = nil
		adminToken, _, _ := mgr.Issue(1, "Ada Admin", RoleAdmin)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d", rec.Code)
		}
		if !reachedInnerHandler {
			t.Fatal("debería haber llegado al handler interno")
		}
		if claimsSeen == nil || claimsSeen.UserID != 1 {
			t.Fatalf("esperaba encontrar las claims del usuario admin en el contexto, obtuve %+v", claimsSeen)
		}
	})
}
