// Command seed carga datos de demostración: usuarios de cada rol, zonas y
// mesas, catálogo de productos con modificadores. Útil para desarrollo,
// pruebas manuales y para tener un ambiente listo para demostrar el
// sistema. No debe usarse en producción tal cual (las contraseñas son de
// ejemplo).
package main

import (
	"fmt"
	"log"

	"github.com/ronin410/restaurante/backend/internal/authn"
	"github.com/ronin410/restaurante/backend/internal/config"
	"github.com/ronin410/restaurante/backend/internal/dbx"
)

type seedUser struct {
	name, email, password, role, position string
	salary                                float64
}

func main() {
	cfg := config.Load()
	db, err := dbx.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("no se pudo abrir la base de datos: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&count); err != nil {
		log.Fatal(err)
	}
	if count > 0 {
		fmt.Println("Ya existen usuarios; no se vuelve a sembrar la base de datos.")
		return
	}

	users := []seedUser{
		{"Admin Dueño", "admin@taqueria.mx", "admin123", "admin", "Dueño", 0},
		{"Gerardo Gerente", "gerente@taqueria.mx", "gerente123", "gerente", "Gerente", 18000},
		{"Carla Cajera", "cajero@taqueria.mx", "cajero123", "cajero", "Cajera", 8000},
		{"Manolo Mesero", "mesero@taqueria.mx", "mesero123", "mesero", "Mesero", 6000},
		{"Cocinero Coco", "cocina@taqueria.mx", "cocina123", "cocina", "Cocinero", 7500},
		{"Conta Contreras", "contador@taqueria.mx", "contador123", "contador", "Contador externo", 0},
	}
	for _, u := range users {
		hash, err := authn.HashPassword(u.password)
		if err != nil {
			log.Fatal(err)
		}
		mustInsert(db, `INSERT INTO users (name, email, password_hash, role, position, salary) VALUES (?, ?, ?, ?, ?, ?)`,
			u.name, u.email, hash, u.role, u.position, u.salary)
	}

	zoneInterior := mustInsert(db, `INSERT INTO zones (name) VALUES ('Interior')`)
	zoneTerraza := mustInsert(db, `INSERT INTO zones (name) VALUES ('Terraza')`)
	for i := 1; i <= 6; i++ {
		mustInsert(db, `INSERT INTO tables_ (zone_id, number, capacity) VALUES (?, ?, 4)`, zoneInterior, fmt.Sprintf("%d", i))
	}
	for i := 1; i <= 4; i++ {
		mustInsert(db, `INSERT INTO tables_ (zone_id, number, capacity) VALUES (?, ?, 4)`, zoneTerraza, fmt.Sprintf("T%d", i))
	}

	catTacos := mustInsert(db, `INSERT INTO categories (name, sort_order) VALUES ('Tacos', 1)`)
	catOrdenes := mustInsert(db, `INSERT INTO categories (name, sort_order) VALUES ('Órdenes', 2)`)
	catBebidas := mustInsert(db, `INSERT INTO categories (name, sort_order) VALUES ('Bebidas', 3)`)
	catExtras := mustInsert(db, `INSERT INTO categories (name, sort_order) VALUES ('Extras', 4)`)
	catSalsas := mustInsert(db, `INSERT INTO categories (name, sort_order) VALUES ('Salsas', 5)`)

	guisado := mustInsert(db, `INSERT INTO modifier_groups (name, required, multiple) VALUES ('Guisado', 1, 0)`)
	for _, opt := range []string{"Pastor", "Suadero", "Bistec", "Chorizo", "Vegetariano"} {
		mustInsert(db, `INSERT INTO modifier_options (group_id, name, price_delta) VALUES (?, ?, 0)`, guisado, opt)
	}
	extrasMod := mustInsert(db, `INSERT INTO modifier_groups (name, required, multiple) VALUES ('Extras', 0, 1)`)
	mustInsert(db, `INSERT INTO modifier_options (group_id, name, price_delta) VALUES (?, 'Con todo', 0)`, extrasMod)
	mustInsert(db, `INSERT INTO modifier_options (group_id, name, price_delta) VALUES (?, 'Sin cebolla', 0)`, extrasMod)
	mustInsert(db, `INSERT INTO modifier_options (group_id, name, price_delta) VALUES (?, 'Doble queso', 8)`, extrasMod)

	tacoID := mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Taco', 18)`, catTacos)
	mustExec(db, `INSERT INTO product_modifier_groups (product_id, group_id) VALUES (?, ?)`, tacoID, guisado)
	mustExec(db, `INSERT INTO product_modifier_groups (product_id, group_id) VALUES (?, ?)`, tacoID, extrasMod)

	gringaID := mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Gringa', 45)`, catOrdenes)
	mustExec(db, `INSERT INTO product_modifier_groups (product_id, group_id) VALUES (?, ?)`, gringaID, extrasMod)

	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Orden de queso fundido', 65)`, catOrdenes)
	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Refresco', 25)`, catBebidas)
	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Agua de horchata', 22)`, catBebidas)
	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Cerveza', 40)`, catBebidas)
	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Papas fritas', 35)`, catExtras)
	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Salsa verde', 0)`, catSalsas)
	mustInsert(db, `INSERT INTO products (category_id, name, price) VALUES (?, 'Salsa roja', 0)`, catSalsas)

	mustInsert(db, `INSERT INTO inventory_items (name, unit, stock, min_stock) VALUES ('Tortilla de maíz', 'kg', 20, 5)`)
	mustInsert(db, `INSERT INTO inventory_items (name, unit, stock, min_stock) VALUES ('Carne de pastor', 'kg', 15, 5)`)

	fmt.Println("Datos de demostración creados. Usuarios:")
	for _, u := range users {
		fmt.Printf("  %-10s %-25s / %s\n", u.role, u.email, u.password)
	}
}

// mustInsert ejecuta un INSERT de una sola fila en una tabla con columna
// "id" y regresa el id generado. A diferencia de SQLite, el driver de
// Postgres no soporta LastInsertId(); se agrega "RETURNING id" a la
// consulta y se lee con QueryRow en su lugar.
func mustInsert(db *dbx.DB, query string, args ...interface{}) int64 {
	var id int64
	if err := db.QueryRow(query+` RETURNING id`, args...).Scan(&id); err != nil {
		log.Fatalf("seed error: %v (query: %s)", err, query)
	}
	return id
}

// mustExec ejecuta una sentencia sin fila que regresar (p.ej. inserciones en
// tablas con llave compuesta como product_modifier_groups).
func mustExec(db *dbx.DB, query string, args ...interface{}) {
	if _, err := db.Exec(query, args...); err != nil {
		log.Fatalf("seed error: %v (query: %s)", err, query)
	}
}
