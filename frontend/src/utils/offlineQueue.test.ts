import { beforeEach, describe, expect, it, vi } from 'vitest'
import { clear, count, enqueue, flush, list } from './offlineQueue'

beforeEach(() => {
  clear()
})

describe('offlineQueue', () => {
  it('empieza vacía', () => {
    expect(count()).toBe(0)
    expect(list()).toEqual([])
  })

  it('encola y lista en orden', () => {
    enqueue({ method: 'POST', path: '/orders/', body: { a: 1 }, description: 'comanda 1' })
    enqueue({ method: 'POST', path: '/orders/2/items', body: { b: 2 }, description: 'agregar producto' })

    const items = list()
    expect(items).toHaveLength(2)
    expect(items[0].description).toBe('comanda 1')
    expect(items[1].path).toBe('/orders/2/items')
    // cada elemento debe tener id único y createdAt
    expect(items[0].id).not.toBe(items[1].id)
  })

  it('flush reenvía en orden y los quita de la cola al tener éxito', async () => {
    enqueue({ method: 'POST', path: '/a', body: {}, description: 'uno' })
    enqueue({ method: 'POST', path: '/b', body: {}, description: 'dos' })

    const calls: string[] = []
    const synced = await flush(async (req) => {
      calls.push(req.path)
    })

    expect(synced).toBe(2)
    expect(calls).toEqual(['/a', '/b'])
    expect(count()).toBe(0)
  })

  it('flush se detiene en el primer fallo y conserva el resto en orden', async () => {
    enqueue({ method: 'POST', path: '/a', body: {}, description: 'uno' })
    enqueue({ method: 'POST', path: '/b', body: {}, description: 'dos' })
    enqueue({ method: 'POST', path: '/c', body: {}, description: 'tres' })

    const send = vi
      .fn()
      .mockResolvedValueOnce(undefined) // /a ok
      .mockRejectedValueOnce(new Error('sin red')) // /b falla
      .mockResolvedValueOnce(undefined) // /c no debería intentarse

    const synced = await flush(send)

    expect(synced).toBe(1)
    expect(send).toHaveBeenCalledTimes(2)
    const remaining = list()
    expect(remaining).toHaveLength(2)
    expect(remaining.map((r) => r.path)).toEqual(['/b', '/c'])
  })
})
