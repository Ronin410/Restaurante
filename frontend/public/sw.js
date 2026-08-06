// Service Worker mínimo: solo cachea el "app shell" (HTML/JS/CSS estáticos)
// para que la interfaz cargue aunque se pierda la conexión en una
// tablet/celular. Las llamadas a la API (/api/...) se dejan pasar siempre a
// la red sin interceptar: la cola offline de comandas/pagos vive en
// JavaScript (ver src/utils/offlineQueue.ts) y necesita ver los fallos de
// red reales para decidir cuándo encolar, no una respuesta cacheada.

const CACHE_NAME = 'taqueria-shell-v1'
const SHELL_URLS = ['/', '/index.html', '/manifest.webmanifest', '/icon.svg']

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(SHELL_URLS)).then(() => self.skipWaiting()),
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url)

  // Nunca interceptar llamadas a la API: deben llegar a la red tal cual.
  if (url.pathname.startsWith('/api/')) return
  if (event.request.method !== 'GET') return

  event.respondWith(
    caches.match(event.request).then((cached) => {
      const network = fetch(event.request)
        .then((response) => {
          if (response.ok) {
            const clone = response.clone()
            caches.open(CACHE_NAME).then((cache) => cache.put(event.request, clone))
          }
          return response
        })
        .catch(() => cached)
      return cached || network
    }),
  )
})
