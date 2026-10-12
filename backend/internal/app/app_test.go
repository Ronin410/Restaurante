package app_test

// Suite de pruebas de integración: levanta el router completo contra una
// base de datos PostgreSQL real y ejercita los flujos de negocio de cada
// módulo a través de la API HTTP real (igual que lo haría el frontend).
//
// A diferencia de cuando el proyecto usaba SQLite, estas pruebas necesitan
// un Postgres alcanzable. Por defecto usan la misma base que arranca
// `podman/run.sh` (ver PODMAN.md); para apuntar a otra instancia (CI, un
// Postgres local en otro puerto, etc.) exporta TEST_DATABASE_URL. Si no hay
// ningún Postgres disponible, las pruebas se omiten (t.Skip) en vez de
// fallar, para que `go test ./...` no truene en un checkout sin base de
// datos configurada.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ronin410/restaurante/backend/internal/app"
	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/config"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

type testEnv struct {
	t       *testing.T
	handler http.Handler
	db      *dbx.DB
	authMgr *authn.Manager
	tokens  map[string]string // role -> token (un usuario de ejemplo por rol)
	userIDs map[string]int
}

func testDSN() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://taqueria:taqueria@localhost:6543/taqueria_test?sslmode=disable"
}

// allTables se usa para vaciar la base entre pruebas (RESTART IDENTITY hace
// que los ids autogenerados vuelvan a empezar en 1, para que las
// aserciones con ids fijos sean estables entre corridas).
var allTables = []string{
	"audit_logs", "invoices", "inventory_movements", "inventory_items", "purchases", "expenses",
	"discounts", "payments", "order_item_modifiers", "order_items", "accounts", "orders",
	"cash_cuts", "cash_shifts", "table_transfers", "table_sessions", "tables_", "zones",
	"product_modifier_groups", "modifier_options", "modifier_groups", "products", "categories",
	"shifts_schedule", "incidents", "attendance", "users",
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db, err := dbx.Open(testDSN())
	if err != nil {
		t.Skipf("sin Postgres de pruebas disponible (%v); levanta uno con podman/run.sh o exporta TEST_DATABASE_URL", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("TRUNCATE " + strings.Join(allTables, ", ") + " RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("limpiando base de datos de prueba: %v", err)
	}

	cfg := config.Config{JWTSecret: "test-secret", TokenTTL: time.Hour, AllowedOrigin: "*"}
	handler := app.New(db, cfg)
	authMgr := authn.NewManager(cfg.JWTSecret, cfg.TokenTTL)

	env := &testEnv{t: t, handler: handler, db: db, authMgr: authMgr, tokens: map[string]string{}, userIDs: map[string]int{}}

	roles := map[string]string{
		"admin":    "Ada Admin",
		"gerente":  "Gilda Gerente",
		"cajero":   "Carlos Cajero",
		"mesero":   "Mario Mesero",
		"cocina":   "Coco Cocinero",
		"contador": "Cindy Contadora",
	}
	for role, name := range roles {
		env.userIDs[role] = env.seedUser(name, role+"@test.mx", "password123", role)
		token, _, err := authMgr.Issue(env.userIDs[role], name, authn.Role(role))
		if err != nil {
			t.Fatalf("emitiendo token: %v", err)
		}
		env.tokens[role] = token
	}

	env.seedCatalog()
	env.seedTable()
	env.fixSequences()
	return env
}

// fixSequences sincroniza las secuencias de identidad de Postgres después
// de que seedCatalog/seedTable insertan filas con id explícito (para tener
// ids predecibles en las aserciones). Postgres no avanza la secuencia en
// un INSERT con id explícito, así que sin esto el primer INSERT normal
// (dejando que la base genere el id) choca con una llave ya usada.
func (e *testEnv) fixSequences() {
	e.t.Helper()
	tablesWithExplicitSeedIDs := []string{
		"zones", "tables_", "categories", "products", "modifier_groups", "modifier_options",
	}
	for _, table := range tablesWithExplicitSeedIDs {
		_, err := e.db.Exec(fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1))`, table, table,
		))
		if err != nil {
			e.t.Fatalf("sincronizando secuencia de %s: %v", table, err)
		}
	}
}

func (e *testEnv) seedUser(name, email, password, role string) int {
	e.t.Helper()
	hash, err := authn.HashPassword(password)
	if err != nil {
		e.t.Fatalf("hash: %v", err)
	}
	var id int
	err = e.db.QueryRow(`INSERT INTO users (name, email, password_hash, role, position, salary) VALUES (?, ?, ?, ?, '', 0) RETURNING id`,
		name, email, hash, role).Scan(&id)
	if err != nil {
		e.t.Fatalf("insertando usuario: %v", err)
	}
	return id
}

func (e *testEnv) seedCatalog() {
	e.t.Helper()
	if _, err := e.db.Exec(`INSERT INTO categories (id, name, sort_order) VALUES (1, 'Tacos', 1), (2, 'Bebidas', 2)`); err != nil {
		e.t.Fatalf("seed categories: %v", err)
	}
	if _, err := e.db.Exec(`INSERT INTO products (id, category_id, name, price, active) VALUES (1, 1, 'Taco', 18, 1), (2, 2, 'Refresco', 25, 1)`); err != nil {
		e.t.Fatalf("seed products: %v", err)
	}
	if _, err := e.db.Exec(`INSERT INTO modifier_groups (id, name, required, multiple) VALUES (1, 'Guisado', 1, 0)`); err != nil {
		e.t.Fatalf("seed modifier groups: %v", err)
	}
	if _, err := e.db.Exec(`INSERT INTO modifier_options (id, group_id, name, price_delta) VALUES (1, 1, 'Pastor', 0), (2, 1, 'Suadero', 3)`); err != nil {
		e.t.Fatalf("seed modifier options: %v", err)
	}
}

func (e *testEnv) seedTable() {
	e.t.Helper()
	if _, err := e.db.Exec(`INSERT INTO zones (id, name) VALUES (1, 'Interior')`); err != nil {
		e.t.Fatalf("seed zone: %v", err)
	}
	if _, err := e.db.Exec(`INSERT INTO tables_ (id, zone_id, number, capacity) VALUES (1, 1, '1', 4)`); err != nil {
		e.t.Fatalf("seed table: %v", err)
	}
}

// do ejecuta una petición HTTP contra el router y decodifica la respuesta
// JSON en `out` (si no es nil). Regresa el código de estado.
func (e *testEnv) do(method, path, token string, body interface{}, out interface{}) int {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if out != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			e.t.Fatalf("decodificando respuesta de %s %s: %v (body=%s)", method, path, err, rec.Body.String())
		}
	}
	return rec.Code
}

func TestHealthz(t *testing.T) {
	env := newTestEnv(t)
	var out map[string]string
	code := env.do(http.MethodGet, "/healthz", "", nil, &out)
	if code != http.StatusOK || out["status"] != "ok" {
		t.Fatalf("healthz inesperado: code=%d out=%v", code, out)
	}
}

func TestAuthLogin(t *testing.T) {
	env := newTestEnv(t)

	t.Run("credenciales válidas", func(t *testing.T) {
		var out map[string]interface{}
		code := env.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "admin@test.mx", "password": "password123",
		}, &out)
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d: %v", code, out)
		}
		if out["token"] == "" || out["token"] == nil {
			t.Fatalf("esperaba token en la respuesta: %v", out)
		}
	})

	t.Run("contraseña incorrecta", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "admin@test.mx", "password": "incorrecta",
		}, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("esperaba 401, obtuve %d", code)
		}
	})

	t.Run("usuario inexistente", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "nadie@test.mx", "password": "x",
		}, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("esperaba 401, obtuve %d", code)
		}
	})

	t.Run("sin token es rechazado en rutas protegidas", func(t *testing.T) {
		code := env.do(http.MethodGet, "/api/v1/auth/me", "", nil, nil)
		if code != http.StatusUnauthorized {
			t.Fatalf("esperaba 401, obtuve %d", code)
		}
	})
}

func TestRoleAuthorization(t *testing.T) {
	env := newTestEnv(t)

	t.Run("mesero no puede crear empleados", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/employees/", env.tokens["mesero"], map[string]interface{}{
			"name": "X", "email": "x@test.mx", "password": "123456", "role": "mesero",
		}, nil)
		if code != http.StatusForbidden {
			t.Fatalf("esperaba 403, obtuve %d", code)
		}
	})

	t.Run("admin sí puede crear empleados", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/employees/", env.tokens["admin"], map[string]interface{}{
			"name": "Nuevo Mesero", "email": "nuevo@test.mx", "password": "123456", "role": "mesero",
		}, nil)
		if code != http.StatusCreated {
			t.Fatalf("esperaba 201, obtuve %d", code)
		}
	})

	t.Run("solo admin/gerente pueden autorizar traspaso de mesa", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/tables/1/transfer", env.tokens["mesero"], map[string]interface{}{
			"to_waiter_id": env.userIDs["mesero"],
		}, nil)
		if code != http.StatusForbidden {
			t.Fatalf("esperaba 403, obtuve %d", code)
		}
	})

	t.Run("mesero y cajero no pueden crear gastos", func(t *testing.T) {
		for _, role := range []string{"mesero", "cajero", "gerente"} {
			code := env.do(http.MethodPost, "/api/v1/expenses/", env.tokens[role], map[string]interface{}{
				"description": "renta", "amount": 100,
			}, nil)
			if code != http.StatusForbidden {
				t.Fatalf("%s: esperaba 403, obtuve %d", role, code)
			}
		}
	})

	t.Run("contador sí puede leer reportes de ventas", func(t *testing.T) {
		code := env.do(http.MethodGet, "/api/v1/reports/sales", env.tokens["contador"], nil, nil)
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d", code)
		}
	})
}

// TestFullRestaurantFlow recorre el ciclo completo: abrir mesa -> comanda
// con modificadores -> cocina -> generar cuenta -> descuento -> pagos
// mixtos -> cierre -> corte de caja Z -> KPIs, verificando los totales en
// cada paso.
func TestFullRestaurantFlow(t *testing.T) {
	env := newTestEnv(t)

	// 1. Abrir mesa 1 como mesero.
	var table map[string]interface{}
	code := env.do(http.MethodPost, "/api/v1/tables/1/open", env.tokens["mesero"], map[string]interface{}{}, &table)
	if code != http.StatusOK || table["status"] != "ocupada" {
		t.Fatalf("abrir mesa falló: code=%d table=%v", code, table)
	}

	// 2. Crear comanda: 3 tacos pastor (modificador sin costo) + 1 refresco.
	var order struct {
		ID    int     `json:"id"`
		Total float64 `json:"total"`
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	code = env.do(http.MethodPost, "/api/v1/orders/", env.tokens["mesero"], map[string]interface{}{
		"table_id": 1,
		"type":     "mesa",
		"items": []map[string]interface{}{
			{"product_id": 1, "qty": 3, "modifier_option_ids": []int{1}},
			{"product_id": 2, "qty": 1},
		},
	}, &order)
	if code != http.StatusCreated {
		t.Fatalf("crear comanda falló: code=%d", code)
	}
	if order.Total != 79 {
		t.Fatalf("total esperado 79 (3*18 + 25), obtuve %.2f", order.Total)
	}
	if len(order.Items) != 2 {
		t.Fatalf("esperaba 2 partidas, obtuve %d", len(order.Items))
	}
	tacoItemID := order.Items[0].ID

	// 3. Agregar una segunda tanda (después del envío original) sin duplicar ticket.
	var afterAdd struct {
		Items []struct {
			Tanda int `json:"tanda"`
		} `json:"items"`
	}
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/items", order.ID), env.tokens["mesero"], map[string]interface{}{
		"items": []map[string]interface{}{{"product_id": 2, "qty": 1}},
	}, &afterAdd)
	if code != http.StatusOK {
		t.Fatalf("agregar partida falló: code=%d", code)
	}
	if len(afterAdd.Items) != 3 {
		t.Fatalf("esperaba 3 partidas tras agregar, obtuve %d", len(afterAdd.Items))
	}
	if afterAdd.Items[2].Tanda != 2 {
		t.Fatalf("la partida agregada después debía quedar en la tanda 2, quedó en %d", afterAdd.Items[2].Tanda)
	}

	// 4. KDS: cocina ve las partidas pendientes y las mueve por el flujo.
	var kds []map[string]interface{}
	code = env.do(http.MethodGet, "/api/v1/kds/items", env.tokens["cocina"], nil, &kds)
	if code != http.StatusOK || len(kds) != 3 {
		t.Fatalf("KDS esperaba 3 partidas pendientes, code=%d len=%d", code, len(kds))
	}
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/orders/items/%d/status", tacoItemID), env.tokens["cocina"], map[string]string{"status": "preparando"}, nil)
	if code != http.StatusOK {
		t.Fatalf("marcar preparando falló: code=%d", code)
	}
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/orders/items/%d/status", tacoItemID), env.tokens["cocina"], map[string]string{"status": "listo"}, nil)
	if code != http.StatusOK {
		t.Fatalf("marcar listo falló: code=%d", code)
	}

	// 5. Abrir turno de caja y generar la cuenta de la mesa.
	code = env.do(http.MethodPost, "/api/v1/shifts/open", env.tokens["cajero"], map[string]interface{}{"opening_fund": 500}, nil)
	if code != http.StatusCreated {
		t.Fatalf("abrir turno falló: code=%d", code)
	}

	var account struct {
		ID            int     `json:"id"`
		Total         float64 `json:"total"`
		Subtotal      float64 `json:"subtotal"`
		DiscountTotal float64 `json:"discount_total"`
		Balance       float64 `json:"balance"`
	}
	code = env.do(http.MethodPost, "/api/v1/accounts/generate", env.tokens["cajero"], map[string]int{"table_id": 1}, &account)
	if code != http.StatusCreated {
		t.Fatalf("generar cuenta falló: code=%d", code)
	}
	if account.Subtotal != 104 { // 3*18 + 2*25
		t.Fatalf("subtotal esperado 104, obtuve %.2f", account.Subtotal)
	}

	// 6. Un cajero no puede autorizar descuentos: requiere supervisor.
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/discounts", account.ID), env.tokens["cajero"],
		map[string]interface{}{"kind": "descuento", "percent": 10, "reason": "prueba"}, nil)
	if code != http.StatusForbidden {
		t.Fatalf("esperaba 403 al aplicar descuento como cajero, obtuve %d", code)
	}

	// 7. Gerente autoriza 10% de descuento.
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/discounts", account.ID), env.tokens["gerente"],
		map[string]interface{}{"kind": "descuento", "percent": 10, "reason": "cliente frecuente"}, &account)
	if code != http.StatusOK {
		t.Fatalf("aplicar descuento falló: code=%d", code)
	}
	if account.Total != 93.6 {
		t.Fatalf("total esperado 93.6 tras 10%% de descuento sobre 104, obtuve %.2f", account.Total)
	}

	// 8. No se puede cerrar con saldo pendiente.
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/close", account.ID), env.tokens["cajero"], nil, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("esperaba 400 al cerrar con saldo pendiente, obtuve %d", code)
	}

	// 9. Pago mixto: efectivo + tarjeta hasta saldar.
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/payments", account.ID), env.tokens["cajero"],
		map[string]interface{}{"method": "efectivo", "amount": 50, "tip_amount": 15}, &account)
	if code != http.StatusCreated {
		t.Fatalf("pago en efectivo falló: code=%d", code)
	}
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/payments", account.ID), env.tokens["cajero"],
		map[string]interface{}{"method": "tarjeta_debito", "amount": account.Balance}, &account)
	if code != http.StatusCreated {
		t.Fatalf("pago con tarjeta falló: code=%d", code)
	}
	if account.Balance > 0.01 {
		t.Fatalf("saldo debería ser ~0, es %.4f", account.Balance)
	}

	// 10. Pago que excede el saldo debe rechazarse.
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/payments", account.ID), env.tokens["cajero"],
		map[string]interface{}{"method": "efectivo", "amount": 100}, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("esperaba 400 por pago excedente, obtuve %d", code)
	}

	// 11. Cerrar la cuenta.
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/close", account.ID), env.tokens["cajero"], nil, nil)
	if code != http.StatusOK {
		t.Fatalf("cerrar cuenta falló: code=%d", code)
	}

	// 11b. Al cerrar la única cuenta, la mesa queda libre sola (sin paso de
	// limpieza) y se puede volver a abrir; "limpieza" ya no es un estatus.
	var freed struct {
		Status string `json:"status"`
	}
	env.do(http.MethodGet, "/api/v1/tables/1", env.tokens["admin"], nil, &freed)
	if freed.Status != "libre" {
		t.Fatalf("la mesa debía quedar libre al cerrar la cuenta, está %q", freed.Status)
	}
	code = env.do(http.MethodPost, "/api/v1/tables/1/status", env.tokens["admin"], map[string]interface{}{"status": "limpieza"}, nil)
	if code != http.StatusBadRequest {
		t.Fatalf("esperaba 400 con estatus limpieza, obtuve %d", code)
	}
	code = env.do(http.MethodPost, "/api/v1/tables/1/open", env.tokens["admin"], map[string]interface{}{}, nil)
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("la mesa liberada debía poder reabrirse, code=%d", code)
	}
	env.do(http.MethodPost, "/api/v1/tables/1/status", env.tokens["admin"], map[string]interface{}{"status": "libre"}, nil)

	// 12. Cerrar turno de caja: debe generar corte Z sin diferencia.
	var shifts []struct {
		ID int `json:"id"`
	}
	env.do(http.MethodGet, "/api/v1/shifts/", env.tokens["cajero"], nil, &shifts)
	if len(shifts) != 1 {
		t.Fatalf("esperaba 1 turno, obtuve %d", len(shifts))
	}
	var closeResp struct {
		Cut struct {
			ExpectedCash float64  `json:"expected_cash"`
			Difference   *float64 `json:"difference"`
			Breakdown    struct {
				SalesTotal float64 `json:"sales_total"`
				TipsTotal  float64 `json:"tips_total"`
			} `json:"breakdown"`
		} `json:"cut"`
	}
	// efectivo esperado = 500 (fondo) + 50 (pago efectivo) = 550
	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/shifts/%d/close", shifts[0].ID), env.tokens["cajero"],
		map[string]interface{}{"counted_cash": 550}, &closeResp)
	if code != http.StatusOK {
		t.Fatalf("cerrar turno falló: code=%d", code)
	}
	if closeResp.Cut.ExpectedCash != 550 {
		t.Fatalf("efectivo esperado debía ser 550, fue %.2f", closeResp.Cut.ExpectedCash)
	}
	if closeResp.Cut.Difference == nil || *closeResp.Cut.Difference != 0 {
		t.Fatalf("diferencia debía ser 0, fue %v", closeResp.Cut.Difference)
	}
	if closeResp.Cut.Breakdown.TipsTotal != 15 {
		t.Fatalf("propinas del corte debían ser 15, fueron %.2f", closeResp.Cut.Breakdown.TipsTotal)
	}

	// 13. KPIs deben reflejar la venta ya cerrada.
	var dash struct {
		VentaTotal     float64 `json:"venta_total"`
		TicketPromedio float64 `json:"ticket_promedio"`
		PropinasTotal  float64 `json:"propinas_total"`
	}
	code = env.do(http.MethodGet, "/api/v1/kpis/dashboard?from=2020-01-01&to=2030-01-01", env.tokens["admin"], nil, &dash)
	if code != http.StatusOK {
		t.Fatalf("kpis dashboard falló: code=%d", code)
	}
	if dash.VentaTotal != 93.6 {
		t.Fatalf("venta total de KPIs esperada 93.6, obtuve %.2f", dash.VentaTotal)
	}
	if dash.PropinasTotal != 15 {
		t.Fatalf("propinas totales de KPIs esperadas 15, obtuve %.2f", dash.PropinasTotal)
	}

	// 14. La bitácora debe tener registro de la cancelación/autorización.
	var auditEntries []map[string]interface{}
	code = env.do(http.MethodGet, "/api/v1/audit-log?entity=account", env.tokens["admin"], nil, &auditEntries)
	if code != http.StatusOK || len(auditEntries) == 0 {
		t.Fatalf("bitácora de cuentas vacía: code=%d len=%d", code, len(auditEntries))
	}
}

func TestOrderItemCancellationRequiresSupervisorAndReason(t *testing.T) {
	env := newTestEnv(t)
	env.do(http.MethodPost, "/api/v1/tables/1/open", env.tokens["mesero"], map[string]interface{}{}, nil)

	var order struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	env.do(http.MethodPost, "/api/v1/orders/", env.tokens["mesero"], map[string]interface{}{
		"table_id": 1, "type": "mesa",
		"items": []map[string]interface{}{{"product_id": 1, "qty": 1}},
	}, &order)
	itemID := order.Items[0].ID

	t.Run("mesero no puede cancelar (requiere supervisor)", func(t *testing.T) {
		code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/orders/items/%d/cancel", itemID), env.tokens["mesero"],
			map[string]string{"reason": "cliente cambió de opinión"}, nil)
		if code != http.StatusForbidden {
			t.Fatalf("esperaba 403, obtuve %d", code)
		}
	})

	t.Run("gerente sin motivo es rechazado", func(t *testing.T) {
		code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/orders/items/%d/cancel", itemID), env.tokens["gerente"],
			map[string]string{"reason": ""}, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("esperaba 400 por motivo vacío, obtuve %d", code)
		}
	})

	t.Run("gerente con motivo cancela la partida", func(t *testing.T) {
		var item map[string]interface{}
		code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/orders/items/%d/cancel", itemID), env.tokens["gerente"],
			map[string]string{"reason": "producto se le cayó al mesero"}, &item)
		if code != http.StatusOK {
			t.Fatalf("esperaba 200, obtuve %d", code)
		}
		if item["status"] != "cancelado" {
			t.Fatalf("esperaba estatus cancelado, obtuve %v", item["status"])
		}
	})
}

func TestInventoryMovementsAdjustStock(t *testing.T) {
	env := newTestEnv(t)
	var item struct {
		ID    int     `json:"id"`
		Stock float64 `json:"stock"`
	}
	code := env.do(http.MethodPost, "/api/v1/inventory/", env.tokens["admin"], map[string]interface{}{
		"name": "Tortillas", "unit": "kg", "min_stock": 5,
	}, &item)
	if code != http.StatusCreated {
		t.Fatalf("crear insumo falló: code=%d", code)
	}

	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/inventory/%d/movements", item.ID), env.tokens["admin"], map[string]interface{}{
		"type": "entrada", "qty": 20, "reason": "compra semanal",
	}, &item)
	if code != http.StatusCreated || item.Stock != 20 {
		t.Fatalf("entrada de inventario falló: code=%d stock=%.2f", code, item.Stock)
	}

	code = env.do(http.MethodPost, fmt.Sprintf("/api/v1/inventory/%d/movements", item.ID), env.tokens["admin"], map[string]interface{}{
		"type": "merma", "qty": 3, "reason": "se echaron a perder",
	}, &item)
	if code != http.StatusCreated || item.Stock != 17 {
		t.Fatalf("merma de inventario falló: code=%d stock=%.2f", code, item.Stock)
	}
}

func TestExpensesAndPnL(t *testing.T) {
	env := newTestEnv(t)
	code := env.do(http.MethodPost, "/api/v1/expenses/", env.tokens["admin"], map[string]interface{}{
		"category": "renta", "description": "renta de local", "amount": 5000, "date": "2026-01-15",
	}, nil)
	if code != http.StatusCreated {
		t.Fatalf("crear gasto falló: code=%d", code)
	}

	var pnl struct {
		Gastos   float64 `json:"gastos"`
		Utilidad float64 `json:"utilidad"`
	}
	code = env.do(http.MethodGet, "/api/v1/reports/pnl?from=2026-01-01&to=2026-01-31", env.tokens["admin"], nil, &pnl)
	if code != http.StatusOK {
		t.Fatalf("reporte P&L falló: code=%d", code)
	}
	if pnl.Gastos != 5000 {
		t.Fatalf("gastos esperados 5000, obtuve %.2f", pnl.Gastos)
	}
}

// TestAccountantCanManageRecords: el contador da de alta, edita y da de baja
// gastos, compras, inventario y facturas; los demás roles operativos no.
func TestAccountantCanManageRecords(t *testing.T) {
	env := newTestEnv(t)
	tok := env.tokens["contador"]

	var exp struct {
		ID     int     `json:"id"`
		Amount float64 `json:"amount"`
	}
	if code := env.do(http.MethodPost, "/api/v1/expenses/", tok, map[string]interface{}{"description": "luz", "amount": 100, "date": "2026-02-10"}, &exp); code != http.StatusCreated {
		t.Fatalf("contador crear gasto: %d", code)
	}
	if code := env.do(http.MethodPut, fmt.Sprintf("/api/v1/expenses/%d", exp.ID), tok, map[string]interface{}{"description": "luz", "amount": 250, "date": "2026-02-10"}, &exp); code != http.StatusOK || exp.Amount != 250 {
		t.Fatalf("contador editar gasto: code=%d amount=%.2f", code, exp.Amount)
	}
	if code := env.do(http.MethodPut, "/api/v1/expenses/9999", tok, map[string]interface{}{"description": "x", "amount": 1}, nil); code != http.StatusNotFound {
		t.Fatalf("editar gasto inexistente: esperaba 404, obtuve %d", code)
	}
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/expenses/%d", exp.ID), env.tokens["cajero"], nil, nil); code != http.StatusForbidden {
		t.Fatalf("cajero borrar gasto: esperaba 403, obtuve %d", code)
	}
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/expenses/%d", exp.ID), tok, nil, nil); code != http.StatusNoContent {
		t.Fatalf("contador borrar gasto: %d", code)
	}
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/expenses/%d", exp.ID), tok, nil, nil); code != http.StatusNotFound {
		t.Fatalf("borrar gasto ya borrado: esperaba 404, obtuve %d", code)
	}

	var pur struct {
		ID int `json:"id"`
	}
	if code := env.do(http.MethodPost, "/api/v1/purchases/", tok, map[string]interface{}{"supplier": "Tortillería", "amount": 300}, &pur); code != http.StatusCreated {
		t.Fatalf("contador crear compra: %d", code)
	}
	if code := env.do(http.MethodPut, fmt.Sprintf("/api/v1/purchases/%d", pur.ID), tok, map[string]interface{}{"supplier": "Tortillería La Esquina", "amount": 320}, nil); code != http.StatusOK {
		t.Fatalf("contador editar compra: %d", code)
	}
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/purchases/%d", pur.ID), tok, nil, nil); code != http.StatusNoContent {
		t.Fatalf("contador borrar compra: %d", code)
	}

	var item struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Stock float64 `json:"stock"`
	}
	env.do(http.MethodPost, "/api/v1/inventory/", tok, map[string]interface{}{"name": "Limones", "unit": "kg", "min_stock": 2}, &item)
	if code := env.do(http.MethodPut, fmt.Sprintf("/api/v1/inventory/%d", item.ID), tok, map[string]interface{}{"name": "Limón", "unit": "kg", "min_stock": 3}, &item); code != http.StatusOK || item.Name != "Limón" {
		t.Fatalf("contador editar insumo: code=%d name=%s", code, item.Name)
	}
	env.do(http.MethodPost, fmt.Sprintf("/api/v1/inventory/%d/movements", item.ID), tok, map[string]interface{}{"type": "entrada", "qty": 10}, &item)
	var mvs []struct {
		ID int `json:"id"`
	}
	env.do(http.MethodGet, fmt.Sprintf("/api/v1/inventory/%d/movements", item.ID), tok, nil, &mvs)
	if len(mvs) != 1 {
		t.Fatalf("esperaba 1 movimiento, obtuve %d", len(mvs))
	}
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/inventory/%d/movements/%d", item.ID, mvs[0].ID), tok, nil, &item); code != http.StatusOK || item.Stock != 0 {
		t.Fatalf("borrar movimiento debía revertir el existente: code=%d stock=%.2f", code, item.Stock)
	}
	env.do(http.MethodPost, fmt.Sprintf("/api/v1/inventory/%d/movements", item.ID), tok, map[string]interface{}{"type": "entrada", "qty": 5}, nil)
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/inventory/%d", item.ID), tok, nil, nil); code != http.StatusNoContent {
		t.Fatalf("contador borrar insumo con historial: %d", code)
	}
}

// TestZonesAndTablesManagement cubre que mesas y zonas se puedan crear,
// renombrar/editar y "eliminar" (desactivar en el caso de mesas, para no
// perder el historial; borrar de verdad en el caso de zonas vacías).
func TestZonesAndTablesManagement(t *testing.T) {
	env := newTestEnv(t)

	t.Run("renombrar zona", func(t *testing.T) {
		var zone map[string]interface{}
		code := env.do(http.MethodPut, "/api/v1/zones/1", env.tokens["admin"], map[string]string{"name": "Interior renombrado"}, &zone)
		if code != http.StatusOK || zone["name"] != "Interior renombrado" {
			t.Fatalf("renombrar zona falló: code=%d zone=%v", code, zone)
		}
	})

	t.Run("mesero no puede crear zonas", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/zones/", env.tokens["mesero"], map[string]string{"name": "Terraza"}, nil)
		if code != http.StatusForbidden {
			t.Fatalf("esperaba 403, obtuve %d", code)
		}
	})

	var terraza struct {
		ID int `json:"id"`
	}
	code := env.do(http.MethodPost, "/api/v1/zones/", env.tokens["admin"], map[string]string{"name": "Terraza"}, &terraza)
	if code != http.StatusCreated {
		t.Fatalf("crear zona falló: code=%d", code)
	}

	t.Run("no se puede eliminar una zona con mesas", func(t *testing.T) {
		code := env.do(http.MethodDelete, "/api/v1/zones/1", env.tokens["admin"], nil, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("esperaba 400 (zona 1 tiene la mesa sembrada), obtuve %d", code)
		}
	})

	t.Run("sí se puede eliminar una zona vacía", func(t *testing.T) {
		code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/zones/%d", terraza.ID), env.tokens["admin"], nil, nil)
		if code != http.StatusNoContent {
			t.Fatalf("esperaba 204, obtuve %d", code)
		}
	})

	var newTable struct {
		ID     int  `json:"id"`
		Active bool `json:"active"`
	}
	code = env.do(http.MethodPost, "/api/v1/tables/", env.tokens["admin"], map[string]interface{}{
		"zone_id": 1, "number": "9", "capacity": 2,
	}, &newTable)
	if code != http.StatusCreated || !newTable.Active {
		t.Fatalf("crear mesa falló: code=%d table=%+v", code, newTable)
	}

	t.Run("no se puede eliminar una mesa ocupada", func(t *testing.T) {
		env.do(http.MethodPost, fmt.Sprintf("/api/v1/tables/%d/open", newTable.ID), env.tokens["mesero"], map[string]interface{}{}, nil)
		code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/tables/%d", newTable.ID), env.tokens["admin"], nil, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("esperaba 400 por mesa ocupada, obtuve %d", code)
		}
		// la liberamos para el resto de la prueba
		env.do(http.MethodPost, fmt.Sprintf("/api/v1/tables/%d/status", newTable.ID), env.tokens["mesero"], map[string]string{"status": "libre"}, nil)
	})

	t.Run("eliminar (desactivar) una mesa libre la saca del listado por defecto", func(t *testing.T) {
		var deactivated map[string]interface{}
		code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/tables/%d", newTable.ID), env.tokens["admin"], nil, &deactivated)
		if code != http.StatusOK || deactivated["active"] != false {
			t.Fatalf("eliminar mesa falló: code=%d table=%v", code, deactivated)
		}

		var list []map[string]interface{}
		env.do(http.MethodGet, "/api/v1/tables/", env.tokens["admin"], nil, &list)
		for _, tbl := range list {
			if int(tbl["id"].(float64)) == newTable.ID {
				t.Fatalf("la mesa eliminada no debería salir en el listado por defecto")
			}
		}

		var listAll []map[string]interface{}
		env.do(http.MethodGet, "/api/v1/tables/?include_inactive=1", env.tokens["admin"], nil, &listAll)
		found := false
		for _, tbl := range listAll {
			if int(tbl["id"].(float64)) == newTable.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("la mesa eliminada sí debería salir con include_inactive=1")
		}
	})

	t.Run("reactivar una mesa eliminada", func(t *testing.T) {
		var reactivated map[string]interface{}
		code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/tables/%d/reactivate", newTable.ID), env.tokens["admin"], nil, &reactivated)
		if code != http.StatusOK || reactivated["active"] != true {
			t.Fatalf("reactivar mesa falló: code=%d table=%v", code, reactivated)
		}
	})
}

// TestAccountRegenerationAndTakeout: generar cuenta marca la mesa, volver a
// generarla retoma la cuenta abierta (en vez de error) y un pedido para llevar
// se cobra sin mesa; al cerrar la cuenta la comanda se cierra sola.
func TestAccountRegenerationAndTakeout(t *testing.T) {
	env := newTestEnv(t)
	env.do(http.MethodPost, "/api/v1/tables/1/open", env.tokens["mesero"], map[string]interface{}{}, nil)
	env.do(http.MethodPost, "/api/v1/orders/", env.tokens["mesero"], map[string]interface{}{
		"table_id": 1, "type": "mesa", "items": []map[string]interface{}{{"product_id": 2, "qty": 2}},
	}, nil)

	var a1, a2 struct {
		ID int `json:"id"`
	}
	if code := env.do(http.MethodPost, "/api/v1/accounts/generate", env.tokens["cajero"], map[string]int{"table_id": 1}, &a1); code != http.StatusCreated {
		t.Fatalf("generar: %d", code)
	}
	var tb struct {
		Status string `json:"status"`
	}
	env.do(http.MethodGet, "/api/v1/tables/1", env.tokens["cajero"], nil, &tb)
	if tb.Status != "cuenta_solicitada" {
		t.Fatalf("la mesa debía quedar en cuenta_solicitada, está %q", tb.Status)
	}
	if code := env.do(http.MethodPost, "/api/v1/accounts/generate", env.tokens["cajero"], map[string]int{"table_id": 1}, &a2); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("regenerar debía retomar la cuenta abierta, code=%d", code)
	}
	if a2.ID != a1.ID {
		t.Fatalf("esperaba la misma cuenta %d, obtuve %d", a1.ID, a2.ID)
	}
	// Cobrar y cerrar: la mesa se libera y su comanda se cierra.
	env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/payments", a1.ID), env.tokens["cajero"], map[string]interface{}{"method": "efectivo", "amount": 50}, nil)
	if code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/close", a1.ID), env.tokens["cajero"], nil, nil); code != http.StatusOK {
		t.Fatalf("cerrar: %d", code)
	}
	var open []struct{}
	env.do(http.MethodGet, "/api/v1/orders/?table_id=1", env.tokens["cajero"], nil, &open)
	if len(open) != 0 {
		t.Fatalf("la comanda cobrada debía cerrarse, hay %d abiertas", len(open))
	}

	// Pedido para llevar: sin mesa.
	var order struct {
		ID int `json:"id"`
	}
	if code := env.do(http.MethodPost, "/api/v1/orders/", env.tokens["cajero"], map[string]interface{}{
		"type": "llevar", "customer_name": "Juan", "items": []map[string]interface{}{{"product_id": 1, "qty": 2, "modifier_option_ids": []int{1}}},
	}, &order); code != http.StatusCreated {
		t.Fatalf("crear pedido llevar: %d", code)
	}
	var pend []struct {
		ID int `json:"id"`
	}
	env.do(http.MethodGet, "/api/v1/orders/?no_table=1", env.tokens["cajero"], nil, &pend)
	if len(pend) != 1 || pend[0].ID != order.ID {
		t.Fatalf("pedidos sin mesa pendientes inesperados: %+v", pend)
	}
	var acc struct {
		ID      int     `json:"id"`
		Balance float64 `json:"balance"`
	}
	if code := env.do(http.MethodPost, "/api/v1/accounts/generate", env.tokens["cajero"], map[string]int{"order_id": order.ID}, &acc); code != http.StatusCreated {
		t.Fatalf("cuenta de pedido: %d", code)
	}
	env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/payments", acc.ID), env.tokens["cajero"], map[string]interface{}{"method": "efectivo", "amount": acc.Balance}, nil)
	if code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%d/close", acc.ID), env.tokens["cajero"], nil, nil); code != http.StatusOK {
		t.Fatalf("cerrar cuenta de pedido: %d", code)
	}
	env.do(http.MethodGet, "/api/v1/orders/?no_table=1", env.tokens["cajero"], nil, &pend)
	if len(pend) != 0 {
		t.Fatalf("el pedido cobrado debía salir de pendientes")
	}
}

// TestEmployeeManagement: editar sueldo/rol/clave, baja, eliminar, y que el
// checador y los datos de otros empleados solo los vea el propio o gerencia.
func TestEmployeeManagement(t *testing.T) {
	env := newTestEnv(t)
	admin, mesero := env.tokens["admin"], env.tokens["mesero"]
	mid := env.userIDs["mesero"]

	var u struct {
		Role   string  `json:"role"`
		Salary float64 `json:"salary"`
		Active bool    `json:"active"`
	}
	code := env.do(http.MethodPut, fmt.Sprintf("/api/v1/employees/%d", mid), admin, map[string]interface{}{
		"name": "Mario Mesero", "role": "cajero", "position": "Caja", "salary": 9000, "active": true, "password": "nueva123",
	}, &u)
	if code != http.StatusOK || u.Role != "cajero" || u.Salary != 9000 {
		t.Fatalf("editar empleado: code=%d %+v", code, u)
	}
	if code := env.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "mesero@test.mx", "password": "nueva123"}, nil); code != http.StatusOK {
		t.Fatalf("login con la clave nueva: %d", code)
	}
	adminID := env.userIDs["admin"]
	if code := env.do(http.MethodPut, fmt.Sprintf("/api/v1/employees/%d", adminID), admin, map[string]interface{}{"name": "A", "role": "admin", "active": false}, nil); code != http.StatusBadRequest {
		t.Fatalf("darse de baja a uno mismo: esperaba 400, obtuve %d", code)
	}

	// Checador: cada quien el suyo.
	if code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/employees/%d/attendance/clock-in", mid), mesero, nil, nil); code != http.StatusCreated {
		t.Fatalf("checar entrada propia: %d", code)
	}
	if code := env.do(http.MethodPost, fmt.Sprintf("/api/v1/employees/%d/attendance/clock-in", adminID), mesero, nil, nil); code != http.StatusForbidden {
		t.Fatalf("checar a otro: esperaba 403, obtuve %d", code)
	}
	if code := env.do(http.MethodGet, fmt.Sprintf("/api/v1/employees/%d", adminID), mesero, nil, nil); code != http.StatusForbidden {
		t.Fatalf("ver ficha ajena: esperaba 403, obtuve %d", code)
	}

	// Eliminar: uno con historial (checador) sí se puede borrar sin ventas; uno con ventas no.
	var nu struct {
		ID int `json:"id"`
	}
	env.do(http.MethodPost, "/api/v1/employees/", admin, map[string]interface{}{"name": "Temp", "email": "t@t.mx", "password": "temporal1", "role": "mesero"}, &nu)
	env.do(http.MethodPost, fmt.Sprintf("/api/v1/employees/%d/attendance/clock-in", nu.ID), admin, nil, nil)
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/employees/%d", nu.ID), admin, nil, nil); code != http.StatusNoContent {
		t.Fatalf("eliminar empleado sin ventas: %d", code)
	}
	env.do(http.MethodPost, "/api/v1/tables/1/open", env.tokens["cocina"], map[string]interface{}{}, nil)
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/employees/%d", env.userIDs["cocina"]), admin, nil, nil); code != http.StatusConflict {
		t.Fatalf("eliminar empleado con historial: esperaba 409, obtuve %d", code)
	}
	if code := env.do(http.MethodDelete, fmt.Sprintf("/api/v1/employees/%d", adminID), admin, nil, nil); code != http.StatusBadRequest {
		t.Fatalf("eliminarse a uno mismo: esperaba 400, obtuve %d", code)
	}
}

// TestCatalogManagement: editar/borrar opciones, grupos y productos.
func TestCatalogManagement(t *testing.T) {
	env := newTestEnv(t)
	admin := env.tokens["admin"]
	if code := env.do(http.MethodPut, "/api/v1/modifier-groups/1/options/2", admin, map[string]interface{}{"name": "Suadero XL", "price_delta": 5}, nil); code != http.StatusOK {
		t.Fatalf("editar opción: %d", code)
	}
	if code := env.do(http.MethodPut, "/api/v1/modifier-groups/1", admin, map[string]interface{}{"name": "Carne", "required": true}, nil); code != http.StatusOK {
		t.Fatalf("editar grupo: %d", code)
	}
	if code := env.do(http.MethodDelete, "/api/v1/modifier-groups/1/options/2", admin, nil, nil); code != http.StatusNoContent {
		t.Fatalf("borrar opción: %d", code)
	}
	if code := env.do(http.MethodDelete, "/api/v1/modifier-groups/1", admin, nil, nil); code != http.StatusNoContent {
		t.Fatalf("borrar grupo: %d", code)
	}
	if code := env.do(http.MethodDelete, "/api/v1/products/2", admin, nil, nil); code != http.StatusNoContent {
		t.Fatalf("borrar producto sin ventas: %d", code)
	}
	if code := env.do(http.MethodDelete, "/api/v1/categories/1", admin, nil, nil); code != http.StatusConflict {
		t.Fatalf("borrar categoría con productos: esperaba 409, obtuve %d", code)
	}
	if code := env.do(http.MethodDelete, "/api/v1/products/1", env.tokens["mesero"], nil, nil); code != http.StatusForbidden {
		t.Fatalf("mesero borrar producto: esperaba 403, obtuve %d", code)
	}
}

// TestBusinessSettings: el nombre del negocio es configurable (el sistema es
// para cualquier restaurante); se lee sin sesión y solo el admin lo cambia.
func TestBusinessSettings(t *testing.T) {
	env := newTestEnv(t)
	var s struct {
		Name string `json:"business_name"`
		Icon string `json:"business_icon"`
	}
	if code := env.do(http.MethodGet, "/api/v1/settings", "", nil, &s); code != http.StatusOK || s.Name == "" {
		t.Fatalf("leer config sin sesión: code=%d %+v", code, s)
	}
	body := map[string]string{"business_name": "Mariscos El Puerto", "business_icon": "🦐"}
	if code := env.do(http.MethodPut, "/api/v1/settings", env.tokens["gerente"], body, nil); code != http.StatusForbidden {
		t.Fatalf("gerente no debe cambiar la config: %d", code)
	}
	if code := env.do(http.MethodPut, "/api/v1/settings", env.tokens["admin"], body, nil); code != http.StatusOK {
		t.Fatalf("admin cambia config: %d", code)
	}
	env.do(http.MethodGet, "/api/v1/settings", "", nil, &s)
	if s.Name != "Mariscos El Puerto" || s.Icon != "🦐" {
		t.Fatalf("config no se guardó: %+v", s)
	}
	if code := env.do(http.MethodPut, "/api/v1/settings", env.tokens["admin"], map[string]string{"business_name": "  "}, nil); code != http.StatusBadRequest {
		t.Fatalf("nombre vacío: esperaba 400, obtuve %d", code)
	}
	env.do(http.MethodPut, "/api/v1/settings", env.tokens["admin"], map[string]string{"business_name": "Mi Restaurante", "business_icon": "🍽️"}, nil)
}
