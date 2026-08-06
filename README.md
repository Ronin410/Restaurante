# Sistema de Administración para Taquería

POS + back office para gestionar la operación diaria de una taquería:
comandas, mesas, cobro, cortes de caja, personal y contaduría, con acceso
por roles y reportes en tiempo real. Implementado a partir de
[`docs/especificaciones_software_taqueria.md`](docs/especificaciones_software_taqueria.md).

- **Backend:** Go (API REST) — [`backend/`](backend/)
- **Frontend:** React + TypeScript (Vite) — [`frontend/`](frontend/)

---

## 1. Arquitectura

```
Navegador / tablet (React SPA, PWA instalable)
        │  HTTPS, JSON
        ▼
API REST en Go (chi router, JWT)
        │
        ▼
SQLite (archivo local, WAL)
```

- **Backend:** Go 1.25, router [chi](https://github.com/go-chi/chi), JWT
  (`golang-jwt`) + bcrypt para autenticación, SQLite puro-Go
  (`modernc.org/sqlite`, sin cgo) como base de datos. El acceso a datos usa
  SQL estándar (sin ORM) para que migrar a PostgreSQL en el futuro —cuando
  se necesite escalar a varias sucursales— sea un cambio acotado al paquete
  `internal/dbx` y a los `store.go` de cada módulo.
- **Frontend:** React 19 + TypeScript, Vite, `react-router-dom` para rutas,
  `@tanstack/react-query` para datos remotos, CSS propio (sin librería de
  componentes) con variables de tema claro/oscuro.
- **Módulos del backend** (cada uno en `internal/<módulo>` con
  `model.go` / `store.go` / `service.go` / `handlers.go`):
  `employees`, `catalog`, `tables`, `orders`, `accounts`, `cashcuts`,
  `accounting`, `kpis`, `audit`, más `authn` (JWT/roles) y `dbx` (conexión +
  migraciones). `internal/app` ensambla el router completo; lo usan tanto
  `cmd/server` como las pruebas de integración.

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

---

## 2. Cómo correr el proyecto en desarrollo

### Backend

```bash
cd backend
go run ./cmd/seed     # crea datos de demo (usuarios, mesas, catálogo) — solo la primera vez
go run ./cmd/server   # arranca la API en :8080 (o el puerto de $PORT)
```

Variables de entorno (ver `backend/.env.example`):

| Variable | Default | Descripción |
|---|---|---|
| `PORT` | `8080` | Puerto HTTP |
| `DB_PATH` | `./data/taqueria.db` | Ruta del archivo SQLite |
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

`frontend/.env.development` ya apunta a `http://localhost:8080/api/v1`. Para
producción, `VITE_API_URL` normalmente se deja vacío y el frontend se sirve
detrás de un proxy inverso que expone la API en `/api/v1` del mismo origen.

---

## 3. Pruebas

### Backend (Go)

```bash
cd backend
go test ./...
```

Incluye pruebas unitarias (auth/JWT, cálculo de totales, ocupación de
mesas) y una suite de integración (`internal/app/app_test.go`) que levanta
el router completo contra una base SQLite en memoria y recorre los flujos
de negocio reales vía HTTP: login y roles, mesas → comanda → tandas → KDS →
cuenta → descuento autorizado → pagos mixtos → cierre → corte de caja Z →
KPIs → bitácora; además de reglas de autorización (cancelar requiere
supervisor y motivo, no se puede cerrar una cuenta con saldo pendiente,
etc.).

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

- Migrar de SQLite a PostgreSQL si se agregan varias sucursales
  concurrentes (el acceso a datos ya está aislado por módulo para que el
  cambio sea localizado).
- Poner el backend detrás de HTTPS y ajustar `JWT_SECRET`/`ALLOWED_ORIGIN`.
- Contratar un PAC para CFDI real y una impresora térmica ESC/POS si se
  requiere ticket impreso físico.
- Definir política de respaldo automático de la base de datos (la
  especificación lo pide como requisito de seguridad).
