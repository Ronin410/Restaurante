package app_test

// Suite de pruebas de integración: levanta el router completo contra una
// base de datos SQLite en memoria y ejercita los flujos de negocio de cada
// módulo a través de la API HTTP real (igual que lo haría el frontend).

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	db      *sql.DB
	authMgr *authn.Manager
	tokens  map[string]string // role -> token (un usuario de ejemplo por rol)
	userIDs map[string]int
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db, err := dbx.Open(":memory:")
	if err != nil {
		t.Fatalf("abriendo base de datos de prueba: %v", err)
	}
	t.Cleanup(func() { db.Close() })

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
	return env
}

func (e *testEnv) seedUser(name, email, password, role string) int {
	e.t.Helper()
	hash, err := authn.HashPassword(password)
	if err != nil {
		e.t.Fatalf("hash: %v", err)
	}
	res, err := e.db.Exec(`INSERT INTO users (name, email, password_hash, role, position, salary) VALUES (?, ?, ?, ?, '', 0)`,
		name, email, hash, role)
	if err != nil {
		e.t.Fatalf("insertando usuario: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
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

	t.Run("contador no puede crear gastos (solo lectura)", func(t *testing.T) {
		code := env.do(http.MethodPost, "/api/v1/expenses/", env.tokens["contador"], map[string]interface{}{
			"description": "renta", "amount": 100,
		}, nil)
		if code != http.StatusForbidden {
			t.Fatalf("esperaba 403, obtuve %d", code)
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
