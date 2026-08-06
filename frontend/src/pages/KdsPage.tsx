import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { OrdersApi } from '../api/endpoints'
import type { OrderItem } from '../api/types'

const COLUMNS: { status: OrderItem['status']; label: string; next?: OrderItem['status'] }[] = [
  { status: 'enviado', label: 'Enviado', next: 'preparando' },
  { status: 'preparando', label: 'Preparando', next: 'listo' },
  { status: 'listo', label: 'Listo para entregar' },
]

export function KdsPage() {
  const queryClient = useQueryClient()
  const { data: items, isLoading } = useQuery({
    queryKey: ['kds'],
    queryFn: OrdersApi.kdsItems,
    refetchInterval: 4000,
  })

  const advance = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => OrdersApi.updateItemStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['kds'] }),
  })

  return (
    <div>
      <h2>Cocina (KDS)</h2>
      {isLoading && <p>Cargando…</p>}
      <div className="grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)', alignItems: 'start' }}>
        {COLUMNS.map((col) => (
          <div key={col.status} className="card">
            <h3>{col.label}</h3>
            {(items ?? [])
              .filter((i) => i.status === col.status)
              .map((item) => (
                <div key={item.id} className="cart-line" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <strong>
                      {item.qty}× {item.product_name}
                    </strong>
                    <span className="muted">
                      {item.order_type === 'mesa' ? `Mesa ${item.table_number}` : item.order_type}
                    </span>
                  </div>
                  {(item.notes || (item.modifiers && item.modifiers.length > 0)) && (
                    <span className="muted">
                      {item.notes} {item.modifiers?.map((m) => m.name).join(', ')}
                    </span>
                  )}
                  <span className="muted">Tanda {item.tanda} · {item.sent_at.slice(11, 16)}</span>
                  {col.next && (
                    <button
                      className="btn primary"
                      style={{ marginTop: '0.4rem' }}
                      onClick={() => advance.mutate({ id: item.id, status: col.next! })}
                    >
                      {col.next === 'preparando' ? 'Empezar a preparar' : 'Marcar listo'}
                    </button>
                  )}
                  {!col.next && (
                    <button
                      className="btn"
                      style={{ marginTop: '0.4rem' }}
                      onClick={() => advance.mutate({ id: item.id, status: 'entregado' })}
                    >
                      Marcar entregado
                    </button>
                  )}
                </div>
              ))}
            {(items ?? []).filter((i) => i.status === col.status).length === 0 && (
              <p className="muted">Sin partidas.</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
