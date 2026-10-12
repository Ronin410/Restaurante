-- Permite "eliminar" mesas sin perder el historial de comandas/cuentas que
-- ya las referencian (orders.table_id, accounts.table_id): en vez de un
-- DELETE físico, se desactivan (active = 0) y dejan de aparecer en el mapa
-- de mesas / al generar nuevas comandas, pero se pueden reactivar.
--
-- Las zonas no necesitan este mismo mecanismo porque nada las referencia
-- directamente (solo las mesas, vía zone_id) y por eso sí se pueden borrar
-- de verdad una vez que se quedan sin mesas asociadas.

ALTER TABLE tables_ ADD COLUMN IF NOT EXISTS active INTEGER NOT NULL DEFAULT 1;
