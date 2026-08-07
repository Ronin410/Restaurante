#!/usr/bin/env bash
# Levanta los 3 contenedores (Postgres, backend, frontend) conectados por una
# red de Podman, con puertos no estándar. Esto NO es podman-compose: son los
# mismos comandos `podman network create` / `podman run` que se explican uno
# por uno en PODMAN.md, solo que encadenados para no escribirlos a mano cada
# vez. Ejecutar desde la raíz del repo: ./podman/run.sh
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

NETWORK=taqueria-net
DB_NAME=taqueria-db
DB_VOLUME=taqueria-db-data
BACKEND_NAME=taqueria-backend
FRONTEND_NAME=taqueria-frontend

DB_PORT="${DB_PORT:-6543}"
BACKEND_PORT="${BACKEND_PORT:-9090}"
FRONTEND_PORT="${FRONTEND_PORT:-4321}"
PGUSER="${PGUSER:-taqueria}"
PGPASSWORD="${PGPASSWORD:-taqueria}"
PGDATABASE="${PGDATABASE:-taqueria}"
JWT_SECRET="${JWT_SECRET:-cambia-este-secreto-en-produccion}"

echo "==> Red de Podman"
podman network exists "$NETWORK" || podman network create "$NETWORK"

echo "==> Volumen de datos de Postgres"
podman volume exists "$DB_VOLUME" || podman volume create "$DB_VOLUME"

echo "==> Base de datos (Postgres, puerto $DB_PORT en vez del 5432 estándar)"
if ! podman container exists "$DB_NAME"; then
  podman run -d \
    --name "$DB_NAME" \
    --network "$NETWORK" \
    -e POSTGRES_USER="$PGUSER" \
    -e POSTGRES_PASSWORD="$PGPASSWORD" \
    -e POSTGRES_DB="$PGDATABASE" \
    -v "$DB_VOLUME":/var/lib/postgresql/data \
    -p "$DB_PORT:$DB_PORT" \
    docker.io/library/postgres:16-alpine \
    postgres -c port="$DB_PORT"
else
  podman start "$DB_NAME" >/dev/null
fi

echo "==> Esperando a que Postgres acepte conexiones..."
until podman exec "$DB_NAME" pg_isready -p "$DB_PORT" -U "$PGUSER" >/dev/null 2>&1; do
  sleep 1
done

echo "==> Construyendo imagen del backend"
podman build -t taqueria-backend ./backend

echo "==> Backend (puerto $BACKEND_PORT en vez del 8080 típico)"
podman rm -f "$BACKEND_NAME" >/dev/null 2>&1 || true
podman run -d \
  --name "$BACKEND_NAME" \
  --network "$NETWORK" \
  -e PGHOST="$DB_NAME" \
  -e PGPORT="$DB_PORT" \
  -e PGUSER="$PGUSER" \
  -e PGPASSWORD="$PGPASSWORD" \
  -e PGDATABASE="$PGDATABASE" \
  -e PGSSLMODE=disable \
  -e JWT_SECRET="$JWT_SECRET" \
  -e ALLOWED_ORIGIN="http://localhost:$FRONTEND_PORT" \
  -e PORT="$BACKEND_PORT" \
  -p "$BACKEND_PORT:$BACKEND_PORT" \
  taqueria-backend

echo "==> Esperando a que el backend responda..."
until curl -sf "http://localhost:$BACKEND_PORT/healthz" >/dev/null 2>&1; do
  sleep 1
done

echo "==> Sembrando datos de demostración (solo si la base está vacía)"
podman run --rm \
  --network "$NETWORK" \
  -e PGHOST="$DB_NAME" \
  -e PGPORT="$DB_PORT" \
  -e PGUSER="$PGUSER" \
  -e PGPASSWORD="$PGPASSWORD" \
  -e PGDATABASE="$PGDATABASE" \
  -e PGSSLMODE=disable \
  --entrypoint /usr/local/bin/seed \
  taqueria-backend

echo "==> Construyendo imagen del frontend (apuntando al backend en localhost:$BACKEND_PORT)"
podman build --build-arg VITE_API_URL="http://localhost:$BACKEND_PORT/api/v1" -t taqueria-frontend ./frontend

echo "==> Frontend (puerto $FRONTEND_PORT)"
podman rm -f "$FRONTEND_NAME" >/dev/null 2>&1 || true
podman run -d \
  --name "$FRONTEND_NAME" \
  --network "$NETWORK" \
  -p "$FRONTEND_PORT:$FRONTEND_PORT" \
  taqueria-frontend

cat <<EOF

Listo. Abre http://localhost:$FRONTEND_PORT

Usuarios de demo (ver backend/cmd/seed): admin@taqueria.mx / admin123, etc.

Para detener todo: ./podman/stop.sh
EOF
