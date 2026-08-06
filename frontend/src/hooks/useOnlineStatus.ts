import { useEffect, useState } from 'react'
import { flushOfflineQueue, offlineQueue } from '../api/client'

/**
 * Sigue el estado de conexión del navegador y, al recuperar internet,
 * reenvía automáticamente lo que se haya encolado en modo offline
 * (comandas/pagos capturados sin conexión).
 */
export function useOnlineStatus() {
  const [online, setOnline] = useState(navigator.onLine)
  const [pending, setPending] = useState(offlineQueue.count())
  const [syncing, setSyncing] = useState(false)

  useEffect(() => {
    const handleOnline = async () => {
      setOnline(true)
      setSyncing(true)
      try {
        await flushOfflineQueue()
      } finally {
        setPending(offlineQueue.count())
        setSyncing(false)
      }
    }
    const handleOffline = () => setOnline(false)

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    const interval = window.setInterval(() => setPending(offlineQueue.count()), 2000)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
      window.clearInterval(interval)
    }
  }, [])

  return { online, pending, syncing }
}
