# Levantar el sistema con Docker Compose

Alternativa a [PODMAN.md](PODMAN.md): mismo resultado (Postgres + backend +
frontend conectados, puertos no estándar), pero con un solo comando en vez
de los `podman run` explicados uno por uno. Útil si en tu equipo usan
Docker Desktop/Engine en vez de Podman, o si prefieres no escribir cada
comando a mano.

Requiere **Docker Compose V2** (el subcomando `docker compose`, con
espacio — viene incluido en Docker Desktop y en Docker Engine recientes).
Si solo tienes el viejo `docker-compose` con guion, instala el plugin
Compose V2 o usa la guía de Podman.

## Puertos (igual que con Podman)

| Servicio | Puerto |
|---|---|
| Base de datos (Postgres) | **6543** (en vez de 5432) |
| Backend (API Go) | **9090** (en vez de 8080) |
| Frontend (React) | **4321** (en vez de 3000/5173) |

## Arrancar todo

```bash
docker compose up -d --build
```

Esto construye las imágenes de `backend/` y `frontend/` (usan el mismo
`Containerfile` de la guía de Podman — Compose lo entiende igual que un
`Dockerfile`) y levanta los 3 contenedores en una red que crea
automáticamente (`restaurante_default`), donde se resuelven entre sí por el
nombre del servicio: el backend se conecta a la base con `PGHOST=db`, no
con `localhost`.

Espera a que el backend responda:

```bash
curl --retry 10 --retry-delay 2 --retry-connrefused http://localhost:9090/healthz
```

## Datos de demostración

El servicio `seed` no arranca solo (tiene `profiles: ["seed"]` en el
`docker-compose.yml` para que un `up` normal no lo repita cada vez). Córrelo
cuando quieras sembrar usuarios/mesas/catálogo de ejemplo:

```bash
docker compose run --rm seed
```

Crea `admin@taqueria.mx` / `admin123` y el resto de roles (ver
`backend/cmd/seed/main.go`). Si ya hay usuarios, no hace nada — es seguro
correrlo de más.

Abre **http://localhost:4321**.

## Ver logs / estado

```bash
docker compose ps
docker compose logs -f backend   # o db / frontend
```

## Actualizar tras un cambio de código

```bash
docker compose up -d --build backend   # o frontend
```

## Detener

```bash
docker compose down        # detiene y borra los contenedores; conserva el volumen de datos
docker compose down -v     # además borra el volumen (pierdes los datos de Postgres)
```

## Variables de entorno

`JWT_SECRET` se puede sobreescribir sin tocar el `docker-compose.yml`,
poniéndolo en un archivo `.env` junto al `docker-compose.yml` (Compose lo
lee solo) o exportándolo antes del `up`:

```bash
JWT_SECRET=algo-mas-seguro docker compose up -d --build
```

Si cambias el puerto del frontend u otros valores, edita directamente
`docker-compose.yml` (los `ports:` y el `VITE_API_URL` del build-arg) —
igual que con Podman, si cambias `BACKEND_PORT` tienes que reconstruir el
frontend, porque esa URL queda incrustada en el JavaScript.

## ¿Docker Compose o Podman?

Hacen lo mismo; usa el que ya tengas instalado:

- **Docker Compose**: un comando (`docker compose up`), bueno para
  arrancar rápido o en CI.
- **[Podman](PODMAN.md)**: comandos `podman run` explícitos, uno por
  servicio — mejor si quieres entender/ajustar exactamente cómo se conecta
  cada pieza, o si tu política es no depender del daemon de Docker.

Ninguno de los dos reemplaza al [README.md](README.md) principal
(arquitectura, cómo correr en desarrollo sin contenedores, pruebas, qué es
simulación todavía).
