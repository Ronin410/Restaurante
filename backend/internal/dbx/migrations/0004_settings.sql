-- Configuración general del negocio (nombre, ícono…): el sistema sirve para
-- cualquier tipo de restaurante, así que el nombre ya no está escrito en el código.
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO settings (key, value) VALUES ('business_name', 'Mi Restaurante') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('business_icon', '🍽️') ON CONFLICT (key) DO NOTHING;
