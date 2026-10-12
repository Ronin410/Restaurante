import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { AccountsApi, CatalogApi, OrdersApi, TablesApi } from '../api/endpoints'
import type { NewItemInput } from '../api/endpoints'
import type { Product } from '../api/types'
import { formatCurrency } from '../utils/money'
import { ProductPickerModal } from '../components/ProductPickerModal'
import { useAuth } from '../auth/AuthContext'
import { Field } from '../components/Field'

interface CartLine extends NewItemInput {
  key: string
  productName: string
  lineNote: string
}

export function TableDetailPage() {
  const { id } = useParams()
  const tableId = Number(id)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { hasRole } = useAuth()

  const [cart, setCart] = useState<CartLine[]>([])
  const [activeProduct, setActiveProduct] = useState<Product | null>(null)
  const [cancelTarget, setCancelTarget] = useState<number | null>(null)
  const [cancelReason, setCancelReason] = useState('')

  const { data: table } = useQuery({ queryKey: ['table', tableId], queryFn: () => TablesApi.get(tableId) })
  const { data: products } = useQuery({ queryKey: ['products'], queryFn: () => CatalogApi.products(true) })
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: CatalogApi.categories })
  const { data: openOrders, refetch: refetchOrders } = useQuery({
    queryKey: ['orders', 'table', tableId],
    queryFn: () => OrdersApi.listOpen(tableId),
    refetchInterval: 8000,
  })

  const { data: tableAccounts } = useQuery({
    queryKey: ['accounts', 'table', tableId],
    queryFn: () => AccountsApi.byTable(tableId),
    refetchInterval: 8000,
  })
  const openAccounts = (tableAccounts ?? []).filter((a) => a.status === 'abierta')

  const order = openOrders?.[0]

  function addToCart(item: NewItemInput, product: Product) {
    setCart((prev) => [
      ...prev,
      { ...item, key: `${Date.now()}-${Math.random()}`, productName: product.name, lineNote: item.notes ?? '' },
    ])
  }

  function removeFromCart(key: string) {
    setCart((prev) => prev.filter((l) => l.key !== key))
  }

  const cartTotal = cart.reduce((sum, l) => sum + l.qty * (products?.find((p) => p.id === l.product_id)?.price ?? 0), 0)

  const sendMutation = useMutation({
    mutationFn: async () => {
      const items: NewItemInput[] = cart.map(({ product_id, qty, notes, modifier_option_ids }) => ({
        product_id,
        qty,
        notes,
        modifier_option_ids,
      }))
      if (order) {
        return OrdersApi.addItems(order.id, items)
      }
      return OrdersApi.create({ table_id: tableId, type: 'mesa', items })
    },
    onSuccess: (result) => {
      setCart([])
      queryClient.invalidateQueries({ queryKey: ['orders', 'table', tableId] })
      if (result.status === 'queued') {
        window.alert('Sin conexión: la comanda se guardó localmente y se enviará al reconectar.')
      }
    },
  })

  const statusMutation = useMutation({
    mutationFn: (status: string) => TablesApi.setStatus(tableId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tables'] })
      queryClient.invalidateQueries({ queryKey: ['table', tableId] })
    },
  })

  const cancelMutation = useMutation({
    mutationFn: () => OrdersApi.cancelItem(cancelTarget!, cancelReason),
    onSuccess: () => {
      setCancelTarget(null)
      setCancelReason('')
      refetchOrders()
    },
  })

  if (!table) return <p>Cargando…</p>

  return (
    <div>
      <div className="toolbar">
        <button className="btn" onClick={() => navigate('/mesas')}>
          ← Mesas
        </button>
        <h2 style={{ margin: 0 }}>
          Mesa {table.number} · <span className="status-badge">{table.status}</span>
        </h2>
        <div style={{ flex: 1 }} />
        <button className="btn primary" onClick={() => navigate(`/cuentas/nueva?table=${tableId}`)}>
          {openAccounts.length > 0 ? 'Cobrar cuenta' : 'Generar cuenta'}
        </button>
        {!order && table.status !== 'libre' && (
          <button
            className="btn"
            title="Para una mesa abierta por error, sin consumo"
            onClick={() => statusMutation.mutate('libre', { onSuccess: () => navigate('/mesas') })}
          >
            Liberar mesa (sin consumo)
          </button>
        )}
      </div>

      {openAccounts.map((a) => (
        <div className="callout" key={a.id}>
          <div>
            <strong>Cuenta #{a.id} generada, falta cobrar</strong>
            <div className="muted">
              Saldo pendiente {formatCurrency(a.balance)}. La mesa se libera sola cuando la cuenta se cobra y se cierra.
            </div>
          </div>
          <button className="btn primary" onClick={() => navigate(`/cuentas/${a.id}`)}>
            Cobrar / ver cuenta
          </button>
        </div>
      ))}

      <div className="grid" style={{ gridTemplateColumns: '2fr 1fr', alignItems: 'start' }}>
        <div>
          <div className="section-title" style={{ marginTop: 0 }}>
            Catálogo
          </div>
          {(categories ?? []).map((cat) => (
            <div key={cat.id} style={{ marginBottom: '0.75rem' }}>
              <strong>{cat.name}</strong>
              <div className="chip-row" style={{ marginTop: '0.4rem' }}>
                {(products ?? [])
                  .filter((p) => p.category_id === cat.id)
                  .map((p) => (
                    <button key={p.id} className="chip" onClick={() => setActiveProduct(p)}>
                      {p.name} · {formatCurrency(p.price)}
                    </button>
                  ))}
              </div>
            </div>
          ))}

          <div className="section-title">Comanda actual</div>
          {!order && <p className="muted">Sin comandas abiertas todavía en esta mesa.</p>}
          {order && (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Producto</th>
                  <th>Cant.</th>
                  <th>Notas</th>
                  <th>Tanda</th>
                  <th>Estatus</th>
                  <th>Total</th>
                  {hasRole('admin', 'gerente') && <th></th>}
                </tr>
              </thead>
              <tbody>
                {order.items.map((item) => (
                  <tr key={item.id}>
                    <td>{item.product_name}</td>
                    <td>{item.qty}</td>
                    <td className="muted">
                      {item.notes} {item.modifiers?.map((m) => m.name).join(', ')}
                    </td>
                    <td>{item.tanda}</td>
                    <td>
                      <span className="status-badge">{item.status}</span>
                    </td>
                    <td>{formatCurrency(item.line_total)}</td>
                    {hasRole('admin', 'gerente') && (
                      <td>
                        {item.status !== 'cancelado' && item.account_id === null && (
                          <button className="btn danger" onClick={() => setCancelTarget(item.id)}>
                            Cancelar
                          </button>
                        )}
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <div className="card">
          <div className="section-title" style={{ marginTop: 0 }}>
            Carrito
          </div>
          {cart.length === 0 && <p className="muted">Agrega productos del catálogo.</p>}
          {cart.map((line) => (
            <div className="cart-line" key={line.key}>
              <span>
                {line.qty}× {line.productName}
              </span>
              <button className="btn" onClick={() => removeFromCart(line.key)}>
                ✕
              </button>
            </div>
          ))}
          {cart.length > 0 && (
            <>
              <div className="cart-line" style={{ fontWeight: 600 }}>
                <span>Total</span>
                <span>{formatCurrency(cartTotal)}</span>
              </div>
              <button
                className="btn primary"
                style={{ width: '100%', marginTop: '0.75rem' }}
                disabled={sendMutation.isPending}
                onClick={() => sendMutation.mutate()}
              >
                Enviar a cocina
              </button>
            </>
          )}
          {sendMutation.isError && <div className="error-banner">{(sendMutation.error as Error).message}</div>}
        </div>
      </div>

      {activeProduct && (
        <ProductPickerModal
          product={activeProduct}
          onClose={() => setActiveProduct(null)}
          onAdd={(item) => addToCart(item, activeProduct)}
        />
      )}

      {cancelTarget !== null && (
        <div className="modal-backdrop" onClick={() => setCancelTarget(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Cancelar partida</h3>
            <p className="muted">Requiere motivo (control de mermas/fraude).</p>
            <Field label="Motivo">
              <textarea value={cancelReason} onChange={(e) => setCancelReason(e.target.value)} rows={3} />
            </Field>
            {cancelMutation.isError && <div className="error-banner">{(cancelMutation.error as Error).message}</div>}
            <div className="toolbar">
              <button className="btn danger" disabled={!cancelReason} onClick={() => cancelMutation.mutate()}>
                Confirmar cancelación
              </button>
              <button className="btn" onClick={() => setCancelTarget(null)}>
                Cerrar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
