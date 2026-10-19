# Desplegar en Render (paso a paso)

Esta guía monta el sistema completo en [Render](https://render.com) con tres piezas:

| Pieza | Qué es | Tipo en Render |
|---|---|---|
| `restaurante-db` | Base de datos PostgreSQL | PostgreSQL |
| `restaurante-api` | Backend (Go) | Web Service (Docker) |
| `restaurante-web` | Frontend (React) | Static Site |

Todo está descrito en el archivo [`render.yaml`](../render.yaml) de la raíz del repo (un *Blueprint*), así que
Render crea las tres piezas con un solo paso. Tiempo aproximado: 15 minutos.

---

## 0. Antes de empezar

* Una cuenta en Render (puedes entrar con GitHub).
* El código en GitHub (`Ronin410/Restaurante`) y **`render.yaml` en la rama que vas a desplegar**.
  Hoy está en la rama `claude/restaurant-app-react-go-4ux77t`; si quieres desplegar desde `master`,
  primero haz que `master` tenga estos cambios (por ejemplo con un Pull Request a `master`).
* Elige de antemano el **correo y la contraseña del administrador** inicial (mínimo 6 caracteres; usa una contraseña real, no `admin123`).

> **Planes:** el Blueprint usa el plan `free` para que pruebes sin costo. En el plan gratuito los servicios web se
> "duermen" tras un rato sin visitas (la primera petición tarda en responder) y la base de datos gratuita tiene
> límites y caducidad. **Para un restaurante en operación, sube la base de datos y el backend a un plan de pago**
> (Render → servicio → *Settings → Instance Type*). Revisa los precios y límites vigentes en render.com/pricing.

---

## 1. Crear el Blueprint

1. En el panel de Render: **New + → Blueprint**.
2. Conecta tu cuenta de GitHub si te lo pide y elige el repositorio **`Ronin410/Restaurante`**.
3. Elige la **rama** que contiene `render.yaml`. Render lee el archivo y muestra las 3 piezas que va a crear.
4. Te pedirá los valores de las variables marcadas como "sync: false". Llena así por ahora:

| Variable | Servicio | Qué poner |
|---|---|---|
| `ADMIN_EMAIL` | restaurante-api | Tu correo de administrador, ej. `dueno@mirestaurante.com` |
| `ADMIN_PASSWORD` | restaurante-api | Una contraseña segura (mín. 6 caracteres) |
| `ALLOWED_ORIGIN` | restaurante-api | Escribe `*` por ahora (lo cambiaremos en el paso 4) |
| `VITE_API_URL` | restaurante-web | Escribe `http://pendiente` por ahora (lo cambiaremos en el paso 3) |

5. Toca **Apply** (o *Create*). Render empieza a construir. El backend tarda unos minutos la primera vez (compila Go).

`JWT_SECRET` (la clave con la que se firman las sesiones) **la genera Render sola**; no tienes que inventarla.

---

## 2. Verificar el backend

1. Cuando `restaurante-api` diga **Live**, abre su dirección (arriba en la página del servicio), algo como
   `https://restaurante-api-xxxx.onrender.com`.
2. Visita `https://restaurante-api-xxxx.onrender.com/healthz` → debe responder `{"status":"ok"}`.
3. En **Logs** del backend debes ver: `administrador inicial creado: tu-correo`. Eso significa que la base se
   preparó sola (las migraciones corren al arrancar) y se creó tu usuario administrador.

**Copia la dirección del backend**, la necesitas en el siguiente paso.

---

## 3. Apuntar el frontend al backend

El frontend necesita saber dónde está el backend, y esa dirección se "graba" al compilar.

1. Entra a **restaurante-web → Environment**.
2. Cambia `VITE_API_URL` a la dirección del backend **más `/api/v1`**, por ejemplo:

   ```
   https://restaurante-api-xxxx.onrender.com/api/v1
   ```

   (sin `/` al final).
3. Guarda y haz **Manual Deploy → Deploy latest commit** (o *Clear build cache & deploy*) para recompilar.
4. Cuando termine, **copia la dirección del frontend**: `https://restaurante-web-xxxx.onrender.com`.

---

## 4. Permitir al frontend (CORS)

Por seguridad, el backend debe saber qué página puede hablar con él.

1. Entra a **restaurante-api → Environment**.
2. Cambia `ALLOWED_ORIGIN` de `*` a la dirección **exacta** del frontend, sin `/` al final:

   ```
   https://restaurante-web-xxxx.onrender.com
   ```
3. Guarda; Render reinicia el backend solo.

---

## 5. Primer ingreso y configuración inicial

1. Abre la dirección del frontend e inicia sesión con `ADMIN_EMAIL` / `ADMIN_PASSWORD`.
2. Ve a **Configuración** y pon el **nombre y el ícono** de tu restaurante.
3. Crea tu operación (todo se hace desde la app; ver [`MANUAL_DE_USO.md`](MANUAL_DE_USO.md)):
   * **Mesas → Administrar zonas y mesas**: tus zonas (Interior, Terraza…) y mesas.
   * **Catálogo**: categorías, productos y modificadores.
   * **Personal**: un usuario por cada empleado, con su rol (cajero, mesero, cocina, contador…).
4. Recomendado: en el servicio `restaurante-api → Environment`, **borra `ADMIN_PASSWORD`** una vez que ya entraste
   (solo se usa cuando la base está vacía, pero no hace falta dejarla ahí).

> No ejecutes el comando de datos de ejemplo (`seed`) en producción: crea usuarios con contraseñas públicas.

---

## 6. Dominio propio (opcional)

* **restaurante-web → Settings → Custom Domains → Add**, y sigue las instrucciones de DNS que te muestra Render
  (HTTPS se configura solo).
* Si usas un dominio propio en el frontend, actualiza `ALLOWED_ORIGIN` en el backend con ese dominio.
* Si además le pones dominio al backend, actualiza `VITE_API_URL` y vuelve a desplegar el frontend.

---

## 7. Usarlo en la tablet

Abre la dirección del frontend en el navegador de la tablet. Para tenerlo como app: menú del navegador →
**Agregar a pantalla de inicio** (la app es instalable y guarda en el dispositivo las comandas si se va el internet).

---

## 8. Actualizar cuando haya cambios

* Cada `git push` a la rama conectada **despliega solo** (Render lo detecta). Las migraciones de la base de datos se
  aplican automáticamente al arrancar el backend; los datos no se pierden.
* Si cambias `VITE_API_URL`, recompila el frontend (paso 3).
* Después de actualizar, si ves la versión vieja, recarga con Ctrl+Shift+R.

---

## 9. Respaldos y producción

* **Respalda la base de datos.** Los planes de pago de Render incluyen respaldos; en el gratuito no cuentes con ellos.
  También puedes sacar uno manual con `pg_dump` usando la *External Database URL* de `restaurante-db`.
* Usa **planes de pago** para base de datos y backend si el sistema es la caja del restaurante (evita que se "duerma"
  y que caduque la base).
* Mantén `ALLOWED_ORIGIN` con la dirección exacta del frontend (no `*`).
* Cambia las contraseñas de todo usuario de prueba y no compartas cuentas entre empleados.

---

## Problemas frecuentes

| Síntoma | Causa y solución |
|---|---|
| La pantalla de login dice "Error de red" / no carga | `VITE_API_URL` mal escrita o sin recompilar. Debe terminar en `/api/v1`, sin `/` final (paso 3). |
| En la consola del navegador: error de **CORS** | `ALLOWED_ORIGIN` no coincide exactamente con la dirección del frontend (paso 4). |
| Al recargar `/mesas` aparece "Not Found" | Falta la regla de reescritura. Ya viene en `render.yaml`; si creaste el sitio a mano: *Redirects/Rewrites* → `/*` → `/index.html` (tipo Rewrite). |
| El backend queda en "Deploy failed" | Mira **Logs**. Lo común: la base de datos aún no estaba lista; usa *Manual Deploy → Deploy latest commit*. |
| No puedo entrar con el administrador | El administrador solo se crea si la base estaba **vacía** cuando arrancó con `ADMIN_EMAIL` y `ADMIN_PASSWORD`. Revisa los Logs. Si ya había usuarios, usa uno existente. |
| La primera visita tarda ~1 minuto | Plan gratuito: el servicio estaba dormido. Se resuelve con un plan de pago. |
| Veo la versión vieja de la app | Recarga con Ctrl+Shift+R (la app guarda archivos en caché para funcionar sin internet). |

---

## Resumen de variables

**restaurante-api (backend)**

| Variable | Valor |
|---|---|
| `DATABASE_URL` | La pone Render desde la base (automática) |
| `JWT_SECRET` | La genera Render (automática) |
| `ALLOWED_ORIGIN` | Dirección exacta del frontend |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` / `ADMIN_NAME` | Primer administrador (solo si la base está vacía) |
| `PORT` | La define Render; el backend la lee solo |

**restaurante-web (frontend)**

| Variable | Valor |
|---|---|
| `VITE_API_URL` | `https://<tu-backend>.onrender.com/api/v1` |
