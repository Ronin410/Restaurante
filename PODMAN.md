# Levantar el sistema con Podman (sin podman-compose)

Tres contenedores conectados por una red de Podman creada a mano, cada uno
en un puerto **no estándar** (ni el 5432 de Postgres, ni el 8080 típico de
backends, ni el 3000/5173/80 típico de frontends), para que no choquen con
otros servicios en la misma máquina:

| Servicio | Imagen | Puerto (contenedor y host) |
|---|---|---|
| Base de datos | `postgres:16-alpine` | **6543** (en vez de 5432) |
| Backend (API Go) | construida de `backend/Containerfile` | **9090** (en vez de 8080) |
| Frontend (React) | construida de `frontend/Containerfile` | **4321** (en vez de 3000/5173) |

Puedes cambiar cualquiera de los tres exportando `DB_PORT` / `BACKEND_PORT` /
`FRONTEND_PORT` antes de correr los comandos (ver `podman/run.sh`).

No se usa `podman-compose` ni un YAML de orquestación: son comandos
`podman network create` / `podman run` explícitos, uno por servicio, para
que quede claro cómo se conectan entre sí. `podman/run.sh` automatiza
exactamente estos mismos comandos si no quieres teclearlos cada vez.

---

## 1. Red compartida

Los contenedores se van a encontrar por **nombre** (no por IP) a través de
una red que tú creas:

```bash
podman network create taqueria-net
```

Cualquier contenedor en `taqueria-net` puede resolver a los demás usando su
`--name` como hostname — por eso el backend se conecta a la base de datos
con `PGHOST=taqueria-db`, no con `localhost` ni con una IP.

## 2. Base de datos (Postgres)

```bash
podman volume create taqueria-db-data

podman run -d \
  --name taqueria-db \
  --network taqueria-net \
  -e POSTGRES_USER=taqueria \
  -e POSTGRES_PASSWORD=taqueria \
  -e POSTGRES_DB=taqueria \
  -v taqueria-db-data:/var/lib/postgresql/data \
  -p 6543:6543 \
  docker.io/library/postgres:16-alpine \
  postgres -c port=6543
```

- `postgres -c port=6543` le dice al propio servidor de Postgres que
  escuche en 6543 en vez de su puerto por defecto (5432) — no es solo un
  remapeo de puerto del host, adentro del contenedor tampoco se usa 5432.
- `-p 6543:6543` publica ese puerto al host, útil para conectarte con
  `psql`/DBeaver/etc. desde fuera; el backend no lo necesita (usa la red
  interna), así que puedes quitar ese `-p` si no te interesa el acceso
  externo a la base.
- El volumen `taqueria-db-data` persiste los datos aunque borres y vuelvas
  a crear el contenedor.

Espera a que esté lista antes de seguir:

```bash
until podman exec taqueria-db pg_isready -p 6543 -U taqueria; do sleep 1; done
```

## 3. Backend

```bash
cd backend
podman build -t taqueria-backend .

podman run -d \
  --name taqueria-backend \
  --network taqueria-net \
  -e PGHOST=taqueria-db \
  -e PGPORT=6543 \
  -e PGUSER=taqueria \
  -e PGPASSWORD=taqueria \
  -e PGDATABASE=taqueria \
  -e PGSSLMODE=disable \
  -e JWT_SECRET=cambia-este-secreto \
  -e ALLOWED_ORIGIN=http://localhost:4321 \
  -e PORT=9090 \
  -p 9090:9090 \
  taqueria-backend
```

Nota que `PGHOST=taqueria-db` (el nombre del contenedor de la base) y
`PGPORT=6543` (el puerto **interno** de Postgres) — esto es tráfico
contenedor-a-contenedor dentro de `taqueria-net`, no pasa por el `-p` que
publicaste al host.

El backend corre sus migraciones solo, al arrancar.

### Datos de demostración (opcional)

La misma imagen incluye el binario `seed`; se corre una vez, como
contenedor de un solo uso, sobreescribiendo el `ENTRYPOINT`:

```bash
podman run --rm \
  --network taqueria-net \
  -e PGHOST=taqueria-db -e PGPORT=6543 \
  -e PGUSER=taqueria -e PGPASSWORD=taqueria -e PGDATABASE=taqueria -e PGSSLMODE=disable \
  --entrypoint /usr/local/bin/seed \
  taqueria-backend
```

Crea usuarios de cada rol (`admin@taqueria.mx` / `admin123`, etc. — ver
`backend/cmd/seed/main.go`), zonas/mesas y catálogo de ejemplo. Si ya hay
usuarios, no hace nada (es seguro correrlo de más).

## 4. Frontend

```bash
cd ../frontend
podman build --build-arg VITE_API_URL=http://localhost:9090/api/v1 -t taqueria-frontend .

podman run -d \
  --name taqueria-frontend \
  --network taqueria-net \
  -p 4321:4321 \
  taqueria-frontend
```

**Importante sobre `VITE_API_URL`:** Vite incrusta esa URL en el
JavaScript compilado en el momento del `podman build`, no cuando arranca el
contenedor. Como quien la usa es el *navegador* del usuario (no otro
contenedor), tiene que ser una URL alcanzable desde tu máquina —el puerto
9090 publicado al host—, **no** `http://taqueria-backend:9090` (ese nombre
solo se resuelve dentro de `taqueria-net`, el navegador no vive ahí). El
frontend en sí no necesita la red para nada más que consistencia; se une a
`taqueria-net` por si más adelante quieres ponerle un proxy inverso
delante.

Abre **http://localhost:4321**.

---

## Todo junto

```bash
./podman/run.sh    # hace exactamente los pasos de arriba, en orden
./podman/stop.sh   # detiene y borra los contenedores (conserva el volumen de datos)
```

## Actualizar tras un cambio de código

```bash
podman build -t taqueria-backend ./backend   # o ./frontend
podman rm -f taqueria-backend                 # o taqueria-frontend
# vuelve a correr el `podman run` correspondiente de arriba (o ./podman/run.sh)
```

## Verificar que quedó bien conectado

```bash
podman ps                                            # los 3 contenedores como "Up"
curl http://localhost:9090/healthz                   # {"status":"ok"}
podman exec taqueria-backend wget -qO- http://taqueria-db:6543 || true  # confirma que resuelve el nombre (fallará con "not an HTTP server", eso es normal — Postgres no habla HTTP; el punto es que sí conectó)
```

## Notas

- **Rootless:** estos comandos funcionan igual con Podman rootless. La
  imagen del frontend usa `nginx-unprivileged` (escucha en 4321, no en un
  puerto <1024) y la del backend corre como usuario no-root dentro del
  contenedor.
- **Por qué no `podman-compose`:** se pidió explícitamente no usarlo. Si
  más adelante quieres ese flujo, los mismos servicios/variables de este
  documento se pueden pasar directo a un `docker-compose.yml`/Kube YAML
  (`podman generate kube` puede generarlo a partir de los contenedores ya
  corriendo).
- Este documento no reemplaza al [README.md](README.md) (arquitectura,
  cómo correr en desarrollo sin contenedores, pruebas, qué es simulación).
