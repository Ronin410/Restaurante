// Cliente HTTP central: agrega el token JWT, normaliza errores y ofrece un
// modo "encolable" para las mutaciones que deben seguir funcionando sin
// conexión (comandas y cobro, ver utils/offlineQueue.ts).

import * as offlineQueue from '../utils/offlineQueue'

const API_BASE: string = import.meta.env.VITE_API_URL ?? '/api/v1'

const TOKEN_STORAGE_KEY = 'taqueria.token'

let authToken: string | null = null

export function setAuthToken(token: string | null): void {
  authToken = token
  if (token) {
    localStorage.setItem(TOKEN_STORAGE_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_STORAGE_KEY)
  }
}

export function loadStoredToken(): string | null {
  authToken = localStorage.getItem(TOKEN_STORAGE_KEY)
  return authToken
}

export class ApiClientError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
}

async function rawRequest<T>(path: string, options: RequestOptions): Promise<T> {
  const method = options.method ?? 'GET'
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
    },
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  })

  if (!res.ok) {
    let message = `Error ${res.status}`
    try {
      const data = await res.json()
      if (data && typeof data.error === 'string') message = data.error
    } catch {
      // cuerpo no era JSON; se conserva el mensaje genérico
    }
    throw new ApiClientError(message, res.status)
  }
  if (res.status === 204) return undefined as T
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

/** Petición estándar: siempre va a la red, lanza ApiClientError si falla. */
export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  return rawRequest<T>(path, options)
}

export type QueueableResult<T> = { status: 'ok'; data: T } | { status: 'queued' }

/**
 * Para las mutaciones que la especificación exige que sigan funcionando sin
 * internet (captura de comandas y cobro): si el navegador está offline, o si
 * la red falla, la operación se guarda en la cola local en vez de fallar.
 */
export async function apiMutateOrQueue<T>(
  path: string,
  body: unknown,
  description: string,
  method: 'POST' | 'PUT' | 'PATCH' = 'POST',
): Promise<QueueableResult<T>> {
  if (!navigator.onLine) {
    offlineQueue.enqueue({ method, path, body, description })
    return { status: 'queued' }
  }
  try {
    const data = await rawRequest<T>(path, { method, body })
    return { status: 'ok', data }
  } catch (err) {
    if (err instanceof TypeError) {
      // fetch lanza TypeError cuando la red falla (a diferencia de un 4xx/5xx del servidor)
      offlineQueue.enqueue({ method, path, body, description })
      return { status: 'queued' }
    }
    throw err
  }
}

/** Reenvía todo lo pendiente en la cola offline; usarse al reconectar. */
export async function flushOfflineQueue(): Promise<number> {
  return offlineQueue.flush(async (req) => {
    await rawRequest(req.path, { method: req.method, body: req.body })
  })
}

export { offlineQueue }
