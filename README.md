# Sistema de Administración para Taquería

POS + back office para gestionar la operación diaria de una taquería:
comandas, mesas, cobro, cortes de caja, personal y contaduría, con acceso
por roles y reportes en tiempo real. Implementado a partir de
[`docs/especificaciones_software_taqueria.md`](docs/especificaciones_software_taqueria.md).

- **Backend:** Go (API REST) — [`backend/`](backend/)
- **Frontend:** React + TypeScript (Vite) — [`frontend/`](frontend/)
- **Base de datos:** PostgreSQL

¿Quieres correrlo en contenedores? Tres contenedores (Postgres, backend,
frontend) en puertos no estándar, documentados dos formas:
**[PODMAN.md](PODMAN.md)** (comandos `podman run` explícitos, sin
`podman-compose`) o **[DOCKER.md](DOCKER.md)** (`docker compose up`, un
solo comando).

---

## 1. Arquitectura

```
Navegador / tablet (React SPA, PWA instalable)
        │  HTTPS, JSON
        ▼
API REST en Go (chi router, JWT)
        │
        ▼
PostgreSQL
```

- **Backend:** Go 1.25, router [chi](https://github.com/go-chi/chi), JWT
  (`golang-jwt`) + bcrypt para autenticación, PostgreSQL como base de datos
  (driver `pgx`). El acceso a datos usa SQL estándar (sin ORM); los
  `store.go` de cada módulo siguen escribiendo sus consultas con
  placeholders `?` como en SQLite (de cuando el proyecto empezó ahí) —
  `internal/dbx.DB` los traduce a `$1, $2, ...` de forma transparente, así
  que no hay cientos de consultas reescritas por el cambio de motor.
- **Frontend:** React 19 + TypeScript, Vite, `react-router-dom` para rutas,
  `@tanstack/react-query` para datos remotos, CSS propio (sin librería de
  componentes) con variables de tema claro/oscuro. **Pensado primero para
  tablet** (el uso principal en el piso de venta), con escritorio como
  segundo uso: ver "Diseño pensado para tablet" más abajo.
- **Módulos del backend** (cada uno en `internal/<módulo>` con
  `model.go` / `store.go` / `service.go` / `handlers.go`):
  `employees`, `catalog`, `tables`, `orders`, `accounts`, `cashcuts`,
  `accounting`, `kpis`, `audit`, más `authn` (JWT/roles) y `dbx` (conexión +
  migraciones). `internal/app` ensambla el router completo; lo usan tanto
  `cmd/server` como las pruebas de integración.

### 1.1 Diseño pensado para tablet

El uso principal es desde una tablet en el piso de venta (mesero tomando
comandas, cajero cobrando), no desde escritorio — aunque sigue funcionando
igual de bien ahí. Por eso:

- **Menú colapsable:** en pantallas angostas (`frontend/src/index.css`,
  punto de quiebre en 960px) la barra lateral se vuelve un cajón que se
  abre con el botón de hamburguesa (barra superior) y se cierra solo al
  elegir una sección o tocar fuera, para no robarle pantalla al mapa de
  mesas o a la comanda. En pantallas anchas (escritorio) la barra lateral
  vuelve a quedar fija, como antes.
- **Objetivos táctiles grandes:** botones, campos de formulario y fichas de
  mesa con al menos 44px de alto (guía de Apple/Google para tocar con el
  dedo sin fallar), en vez de los controles compactos pensados para mouse.
- **Limitación conocida:** las tablas de datos con muchas columnas
  (Personal, reportes de Contaduría) se desplazan horizontalmente en
  pantallas angostas en vez de reacomodarse en tarjetas — funcionan, pero
  no son tan cómodas en celular. Es de menor prioridad porque esas
  pantallas las usa más que nada el administrador/gerente, típicamente
  desde una tablet en horizontal o una computadora.

### Roles y permisos

| Rol | Accesos |
|---|---|
| **admin** | Todos los módulos y reportes |
| **gerente** | Mesas, comandas, cortes, personal, autorización de descuentos/cancelaciones/traspasos |
| **cajero** | Cobro, cuentas, sus turnos y cortes de caja |
| **mesero** | Captura de comandas, estado de mesas |
| **cocina** | KDS (comandas entrantes), marcar como listo/preparando |
| **contador** | Solo lectura: reportes financieros, facturación, gastos |

Cada rol tiene su propio inicio de sesión (usuario/contraseña individual) y
el backend hace cumplir los permisos en cada endpoint (no solo en la UI).

### Zonas y mesas personalizables

Admin/gerente pueden crear, renombrar y eliminar zonas (interior, terraza,
etc.) y mesas desde la pantalla de Mesas → "Administrar zonas y mesas":

- **Mesas:** "eliminar" las desactiva en vez de borrarlas de verdad, para no
  perder el historial de comandas/cuentas que ya las referencian; quedan
  fuera del mapa de mesas y no se pueden volver a abrir hasta
  reactivarlas (botón "Reactivar" en la misma pantalla, marcando
  "Mostrar mesas eliminadas"). No se puede eliminar una mesa
  ocupada/con cuenta solicitada — hay que liberarla primero.
- **Zonas:** sí se pueden borrar de verdad, pero solo cuando ya no tienen
  ninguna mesa asignada (ni activa ni eliminada), para no dejar mesas
  huérfanas apuntando a una zona que ya no existe.

---

## 2. Cómo correr el proyecto en desarrollo

Con contenedores (recomendado, no requiere instalar Go/Node/Postgres a
mano): **[PODMAN.md](PODMAN.md)** o **[DOCKER.md](DOCKER.md)**. Lo de
abajo es para correrlo directo en tu máquina.

### Backend

Necesitas un PostgreSQL accesible (local, Podman, o el que sea). Con Podman
suelto, sin nada más:

```bash
podman run -d --name taqueria-db -p 6543:6543 \
  -e POSTGRES_USER=taqueria -e POSTGRES_PASSWORD=taqueria -e POSTGRES_DB=taqueria \
  docker.io/library/postgres:16-alpine postgres -c port=6543
```

Luego:

```bash
cd backend
go run ./cmd/seed     # crea datos de demo (usuarios, mesas, catálogo) — solo la primera vez
go run ./cmd/server   # arranca la API en :9090 (o el puerto de $PORT)
```

Variables de entorno (ver `backend/.env.example`):

| Variable | Default | Descripción |
|---|---|---|
| `PORT` | `9090` | Puerto HTTP (no estándar a propósito, ver [PODMAN.md](PODMAN.md)) |
| `DATABASE_URL` | — | DSN completo de Postgres; si se define, ignora las variables `PG*` de abajo |
| `PGHOST` / `PGPORT` / `PGUSER` / `PGPASSWORD` / `PGDATABASE` / `PGSSLMODE` | `localhost` / `6543` / `taqueria` / `taqueria` / `taqueria` / `disable` | Piezas del DSN si no defines `DATABASE_URL` |
| `JWT_SECRET` | (valor de desarrollo) | Secreto para firmar tokens — **cambiar en producción** |
| `ALLOWED_ORIGIN` | `*` | Origen permitido por CORS (poner la URL del frontend en producción) |

Usuarios de demo creados por `cmd/seed` (contraseñas de ejemplo, cámbialas
antes de un uso real):

| Rol | Correo | Contraseña |
|---|---|---|
| admin | admin@taqueria.mx | admin123 |
| gerente | gerente@taqueria.mx | gerente123 |
| cajero | cajero@taqueria.mx | cajero123 |
| mesero | mesero@taqueria.mx | mesero123 |
| cocina | cocina@taqueria.mx | cocina123 |
| contador | contador@taqueria.mx | contador123 |

### Frontend

```bash
cd frontend
npm install
npm run dev   # http://localhost:5173, apunta a VITE_API_URL (ver .env.development)
```

`frontend/.env.development` ya apunta a `http://localhost:9090/api/v1`. Para
producción, `VITE_API_URL` normalmente se deja vacío y el frontend se sirve
detrás de un proxy inverso que expone la API en `/api/v1` del mismo origen.
Si lo empaquetas con Podman, ojo: `VITE_API_URL` se incrusta en el build
(`--build-arg`, no variable de entorno del contenedor) — ver
[PODMAN.md](PODMAN.md).

---

## 3. Pruebas

### Backend (Go)

```bash
cd backend
go test ./...
```

Incluye pruebas unitarias (auth/JWT, cálculo de totales, ocupación de
mesas) y una suite de integración (`internal/app/app_test.go`) que levanta
el router completo contra un **PostgreSQL real** y recorre los flujos de
negocio reales vía HTTP: login y roles, mesas → comanda → tandas → KDS →
cuenta → descuento autorizado → pagos mixtos → cierre → corte de caja Z →
KPIs → bitácora; además de reglas de autorización (cancelar requiere
supervisor y motivo, no se puede cerrar una cuenta con saldo pendiente,
etc.).

Estas pruebas necesitan Postgres alcanzable — por defecto usan la misma
base que levanta `podman/run.sh` (`postgres://taqueria:taqueria@localhost:6543/taqueria_test`);
para apuntar a otra instancia exporta `TEST_DATABASE_URL`. Si no hay
ningún Postgres disponible, las pruebas se omiten (`SKIP`) en vez de
fallar.

### Frontend (Vitest + Testing Library)

```bash
cd frontend
npx vitest run
```

Cubre utilidades puras (formato de moneda, división de cuenta, diferencia
de caja, cola offline) y componentes clave (formulario de login, selector
de producto con modificadores obligatorios, guardas de ruta por rol).

---

## 4. Qué es real y qué es una simulación

Este sistema fue construido y probado end-to-end (incluyendo una pasada
manual en navegador real con capturas de pantalla) para los flujos
centrales: mesas, comandas, KDS, cuentas, pagos mixtos, descuentos con
autorización, cortes de caja y reportes. Tres piezas del documento de
especificaciones dependen de hardware o de un proveedor externo que no se
puede probar en este entorno, así que quedan como **simulación con el
modelo de datos y el flujo listos**, documentado en el propio código:

- **Facturación CFDI 4.0** (`internal/accounting`): el flujo de captura y
  el modelo de datos están completos, pero el "timbrado" es simulado
  (`status: timbrada_simulada`, UUID falso). Facturar de verdad requiere
  contratar un PAC autorizado por el SAT.
- **Impresión térmica / pantalla de cocina dedicada:** no hay driver de
  impresora ni app nativa de cocina; en su lugar, `/accounts/{id}/ticket`
  entrega el ticket como datos estructurados (el frontend lo imprime desde
  el navegador con `window.print()`) y el KDS es la pantalla web
  `/kds`, pensada para abrirse en una tablet en la cocina.
- **Checador biométrico:** el checador de asistencia (entrada/salida) es
  manual vía la app (`attendance.source = "manual"`); el campo ya existe
  para que un lector biométrico futuro solo tenga que agregar
  `source = "biometrico"`.

## 5. Modo offline

Las especificaciones piden que comandas y cobro sigan funcionando sin
internet. Se implementó con una cola en `localStorage`
(`frontend/src/utils/offlineQueue.ts`) en vez de un Service Worker con
Background Sync (soporte desigual entre navegadores/tablets):

- Si el navegador está offline (o la red falla) al crear una comanda,
  agregar productos o registrar un pago, la operación se guarda en la cola
  en vez de fallar.
- Al recuperar conexión (`window.addEventListener('online', …)`), la cola
  se reenvía en orden automáticamente.
- Un Service Worker mínimo (`frontend/public/sw.js`) cachea el "app shell"
  (HTML/JS/CSS) para que la interfaz cargue offline; deja pasar toda
  llamada a `/api/` sin interceptar, para que la lógica de arriba vea los
  fallos de red reales.
- El manifest (`frontend/public/manifest.webmanifest`) permite instalar la
  app como PWA en una tablet/celular.

## 6. Próximos pasos sugeridos para producción

- Poner el backend detrás de HTTPS y ajustar `JWT_SECRET`/`ALLOWED_ORIGIN`.
- Ninguna de las consultas usa transacciones explícitas (heredado de cuando
  el proyecto usaba SQLite con una sola conexión, donde no hacía falta);
  con Postgres y varios workers concurrentes de verdad, vale la pena
  envolver en transacción las operaciones multi-tabla (cerrar cuenta +
  liberar mesa, registrar pago + recalcular totales, etc.).
- Contratar un PAC para CFDI real y una impresora térmica ESC/POS si se
  requiere ticket impreso físico.
- Definir política de respaldo automático de la base de datos (la
  especificación lo pide como requisito de seguridad).
