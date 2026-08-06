-- Esquema inicial del sistema POS + back office para taquería.
-- SQLite (pure-Go driver modernc.org/sqlite). Diseñado para ser portable a
-- PostgreSQL en el futuro (ver internal/dbx): tipos simples, sin funciones
-- específicas de motor salvo CHECK constraints y strftime para fechas.

PRAGMA foreign_keys = ON;

-- ==================== Personal / Usuarios ====================
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin','gerente','cajero','mesero','cocina','contador')),
    position TEXT NOT NULL DEFAULT '',
    salary REAL NOT NULL DEFAULT 0,
    hire_date TEXT NOT NULL DEFAULT (date('now')),
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS attendance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    clock_in TEXT NOT NULL,
    clock_out TEXT,
    source TEXT NOT NULL DEFAULT 'manual', -- manual | biometrico (futuro)
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    type TEXT NOT NULL CHECK (type IN ('falta','retardo','permiso')),
    date TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS shifts_schedule (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    weekday INTEGER NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL
);

-- ==================== Catálogo ====================
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES categories(id),
    name TEXT NOT NULL,
    price REAL NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS modifier_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL, -- p.ej. "Guisado", "Término de cocción"
    required INTEGER NOT NULL DEFAULT 0,
    multiple INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS modifier_options (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL REFERENCES modifier_groups(id),
    name TEXT NOT NULL, -- "sin cebolla", "con todo", "3/4"
    price_delta REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS product_modifier_groups (
    product_id INTEGER NOT NULL REFERENCES products(id),
    group_id INTEGER NOT NULL REFERENCES modifier_groups(id),
    PRIMARY KEY (product_id, group_id)
);

-- ==================== Mesas ====================
CREATE TABLE IF NOT EXISTS zones (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL -- interior, terraza, etc.
);

CREATE TABLE IF NOT EXISTS tables_ (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    zone_id INTEGER NOT NULL REFERENCES zones(id),
    number TEXT NOT NULL,
    capacity INTEGER NOT NULL DEFAULT 4,
    status TEXT NOT NULL DEFAULT 'libre' CHECK (status IN ('libre','ocupada','cuenta_solicitada','limpieza')),
    current_waiter_id INTEGER REFERENCES users(id),
    opened_at TEXT,
    joined_with INTEGER REFERENCES tables_(id), -- union de mesas
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Historial de ocupación de mesas, usado para calcular la rotación
-- (tiempo promedio de ocupación) en el módulo de KPIs.
CREATE TABLE IF NOT EXISTS table_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_id INTEGER NOT NULL REFERENCES tables_(id),
    opened_at TEXT NOT NULL DEFAULT (datetime('now')),
    closed_at TEXT
);

CREATE TABLE IF NOT EXISTS table_transfers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_id INTEGER NOT NULL REFERENCES tables_(id),
    from_waiter_id INTEGER REFERENCES users(id),
    to_waiter_id INTEGER NOT NULL REFERENCES users(id),
    authorized_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ==================== Comandas (órdenes) ====================
CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_id INTEGER REFERENCES tables_(id),
    type TEXT NOT NULL CHECK (type IN ('mesa','llevar','domicilio')),
    waiter_id INTEGER NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'abierta' CHECK (status IN ('abierta','cerrada')),
    customer_name TEXT NOT NULL DEFAULT '',
    customer_phone TEXT NOT NULL DEFAULT '',
    customer_address TEXT NOT NULL DEFAULT '',
    opened_at TEXT NOT NULL DEFAULT (datetime('now')),
    closed_at TEXT
);

CREATE TABLE IF NOT EXISTS order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL REFERENCES orders(id),
    product_id INTEGER NOT NULL REFERENCES products(id),
    product_name_snapshot TEXT NOT NULL,
    qty REAL NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    tanda INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'enviado' CHECK (status IN ('enviado','preparando','listo','entregado','cancelado')),
    account_id INTEGER REFERENCES accounts(id),
    sent_at TEXT NOT NULL DEFAULT (datetime('now')),
    ready_at TEXT,
    delivered_at TEXT,
    cancelled_reason TEXT,
    cancelled_by INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS order_item_modifiers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_item_id INTEGER NOT NULL REFERENCES order_items(id),
    modifier_option_id INTEGER REFERENCES modifier_options(id),
    name_snapshot TEXT NOT NULL,
    price_delta_snapshot REAL NOT NULL DEFAULT 0
);

-- ==================== Cuentas y Cobro ====================
CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_id INTEGER REFERENCES tables_(id),
    order_id INTEGER REFERENCES orders(id),
    status TEXT NOT NULL DEFAULT 'abierta' CHECK (status IN ('abierta','cerrada')),
    -- Cómo se originó la cuenta: completa (una sola cuenta con toda la mesa)
    -- o producto (mesa dividida en varias cuentas por partidas). La división
    -- "por persona" / "monto libre" se resuelve como pagos múltiples sobre
    -- la misma cuenta (ver tabla payments), no como cuentas separadas.
    split_mode TEXT NOT NULL DEFAULT 'completa' CHECK (split_mode IN ('completa','producto')),
    subtotal REAL NOT NULL DEFAULT 0,
    discount_total REAL NOT NULL DEFAULT 0,
    tip_total REAL NOT NULL DEFAULT 0,
    total REAL NOT NULL DEFAULT 0,
    opened_at TEXT NOT NULL DEFAULT (datetime('now')),
    closed_at TEXT,
    closed_by INTEGER REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    method TEXT NOT NULL CHECK (method IN ('efectivo','tarjeta_debito','tarjeta_credito','transferencia','vale_despensa')),
    amount REAL NOT NULL,
    tip_amount REAL NOT NULL DEFAULT 0,
    reference TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL REFERENCES users(id),
    shift_id INTEGER REFERENCES cash_shifts(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS discounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    kind TEXT NOT NULL CHECK (kind IN ('descuento','cortesia')),
    amount REAL NOT NULL,
    percent REAL,
    reason TEXT NOT NULL,
    authorized_by INTEGER NOT NULL REFERENCES users(id),
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ==================== Cortes de caja ====================
CREATE TABLE IF NOT EXISTS cash_shifts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cashier_id INTEGER NOT NULL REFERENCES users(id),
    opening_fund REAL NOT NULL DEFAULT 0,
    opened_at TEXT NOT NULL DEFAULT (datetime('now')),
    closed_at TEXT,
    counted_cash REAL,
    status TEXT NOT NULL DEFAULT 'abierto' CHECK (status IN ('abierto','cerrado'))
);

CREATE TABLE IF NOT EXISTS cash_cuts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    shift_id INTEGER NOT NULL REFERENCES cash_shifts(id),
    type TEXT NOT NULL CHECK (type IN ('x','z')),
    totals_json TEXT NOT NULL,
    expected_cash REAL NOT NULL DEFAULT 0,
    counted_cash REAL,
    difference REAL,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ==================== Contaduría ====================
CREATE TABLE IF NOT EXISTS expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    supplier TEXT NOT NULL DEFAULT '',
    date TEXT NOT NULL DEFAULT (date('now')),
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS purchases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    supplier TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    date TEXT NOT NULL DEFAULT (date('now')),
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS inventory_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    unit TEXT NOT NULL DEFAULT 'pza',
    stock REAL NOT NULL DEFAULT 0,
    min_stock REAL NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS inventory_movements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inventory_item_id INTEGER NOT NULL REFERENCES inventory_items(id),
    type TEXT NOT NULL CHECK (type IN ('entrada','salida','merma')),
    qty REAL NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- CFDI 4.0: stub de facturación electrónica. La integración real con un PAC
-- autorizado por el SAT requiere credenciales/contrato externo; aquí se deja
-- el modelo y flujo listos, con timbrado "simulado".
CREATE TABLE IF NOT EXISTS invoices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    rfc TEXT NOT NULL,
    razon_social TEXT NOT NULL,
    uso_cfdi TEXT NOT NULL DEFAULT 'G03',
    status TEXT NOT NULL DEFAULT 'pendiente' CHECK (status IN ('pendiente','timbrada_simulada','cancelada')),
    uuid_fiscal TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ==================== Auditoría ====================
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    action TEXT NOT NULL,
    entity TEXT NOT NULL,
    entity_id INTEGER,
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_account ON order_items(account_id);
CREATE INDEX IF NOT EXISTS idx_accounts_table ON accounts(table_id);
CREATE INDEX IF NOT EXISTS idx_payments_account ON payments(account_id);
CREATE INDEX IF NOT EXISTS idx_orders_table ON orders(table_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);
