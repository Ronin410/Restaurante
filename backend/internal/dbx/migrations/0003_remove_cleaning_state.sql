-- El sistema ya no maneja el estado "limpieza": al cerrar la cuenta la mesa
-- queda libre. Las mesas que hubieran quedado en limpieza se liberan.
UPDATE tables_ SET status = 'libre', current_waiter_id = NULL, opened_at = NULL, joined_with = NULL WHERE status = 'limpieza';
