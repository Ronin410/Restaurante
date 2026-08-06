// Cola de operaciones pendientes para el "modo offline" del punto 9 de las
// especificaciones: permite seguir tomando comandas y cobrando aunque se
// pierda la conexión, sincronizando al recuperarla. Se implementa sobre
// localStorage (en vez de un Service Worker con Background Sync, de soporte
// desigual entre navegadores) para que funcione en cualquier tablet/celular
// y sea fácil de probar.

export interface QueuedRequest {
  id: string
  method: 'POST' | 'PUT' | 'PATCH'
  path: string
  body: unknown
  createdAt: string
  description: string
}

const STORAGE_KEY = 'taqueria.offline_queue.v1'

function readAll(): QueuedRequest[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeAll(items: QueuedRequest[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
}

export function enqueue(req: Omit<QueuedRequest, 'id' | 'createdAt'>): QueuedRequest {
  const full: QueuedRequest = {
    ...req,
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    createdAt: new Date().toISOString(),
  }
  const items = readAll()
  items.push(full)
  writeAll(items)
  return full
}

export function list(): QueuedRequest[] {
  return readAll()
}

export function remove(id: string): void {
  writeAll(readAll().filter((i) => i.id !== id))
}

export function clear(): void {
  writeAll([])
}

export function count(): number {
  return readAll().length
}

// flush intenta reenviar cada petición encolada, en orden, usando `send`.
// Se detiene ante el primer fallo para no perder el orden de comandas
// (una comanda de la "tanda 2" no debe llegar antes que la de la "tanda 1").
// Regresa cuántas peticiones se sincronizaron con éxito.
export async function flush(send: (req: QueuedRequest) => Promise<void>): Promise<number> {
  const items = readAll()
  let synced = 0
  for (const item of items) {
    try {
      await send(item)
      remove(item.id)
      synced++
    } catch {
      break
    }
  }
  return synced
}
