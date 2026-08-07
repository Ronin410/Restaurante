#!/usr/bin/env bash
# Detiene y borra los 3 contenedores. El volumen de datos de Postgres
# (taqueria-db-data) NO se borra, así que los datos sobreviven; bórralo
# aparte con `podman volume rm taqueria-db-data` si de verdad quieres
# empezar desde cero.
set -euo pipefail

for c in taqueria-frontend taqueria-backend taqueria-db; do
  podman rm -f "$c" >/dev/null 2>&1 && echo "Detenido: $c" || echo "(no estaba corriendo: $c)"
done
