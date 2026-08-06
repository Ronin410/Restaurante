import { useOnlineStatus } from '../hooks/useOnlineStatus'

export function OfflineBanner() {
  const { online, pending, syncing } = useOnlineStatus()

  if (online && pending === 0) return null

  if (!online) {
    return (
      <div className="offline-banner">
        Sin conexión. Puedes seguir tomando comandas y cobrando: se guardarán y se enviarán solas al reconectar.
        {pending > 0 && ` (${pending} pendiente${pending === 1 ? '' : 's'})`}
      </div>
    )
  }

  return (
    <div className="offline-banner">
      {syncing ? 'Sincronizando operaciones pendientes…' : `${pending} operación(es) pendiente(s) de sincronizar.`}
    </div>
  )
}
