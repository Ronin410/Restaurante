import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { AccountsApi, CatalogApi, OrdersApi } from '../api/endpoints'
import type { NewItemInput } from '../api/endpoints'
import type { Order, Product } from '../api/types'
import { formatCurrency } from '../utils/money'
import { Field } from '../components/Field'
import { ProductPickerModal } from '../components/ProductPickerModal'

const TYPE_LABEL: Record<string, string> = { llevar: 'Para llevar', domicilio: 'A domicilio', mesa: 'Mesa' }
const ITEM_LABEL: Record<string, string> = {
  enviado: 'En cocina',
  preparando: 'Preparando',
  listo: 'Listo',
  entregado: 'Entregado',
  cancelado: 'Cancelado',
}

/**
 * Pedidos sin mesa (para llevar y a domicilio): se capturan igual que una
 * comanda, pero ligados a un cliente en vez de a una mesa, y se cobran desde
 * aquí mismo con "Cobrar".
 */
export function OrdersPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [creating, setCreating] = useState(false)

  const { data: orders, isLoading } = useQuery({
    queryKey: ['orders', 'pending'],
    queryFn: OrdersApi.listPending,
    refetchInterval: 8000,
  })

  const charge = useMutation({
    mutationFn: (orderId: number) => AccountsApi.generateForOrder(orderId),
    onSuccess: (account) => navigate(`/cuentas/${account.id}`),
  })

  return (
    <div>
      <div className="page-header">
        <div>
          <h2>Pedidos para llevar y a domicilio</h2>
          <p className="muted">Pedidos que no pertenecen a una mesa. Cóbralos cuando el cliente pague o recoja.</p>
        </div>
        <button className="btn primary" onClick={() => setCreating(true)}>
          + Nuevo pedido
        </button>
      </div>

      {isLoading && <p>Cargando…</p>}
      {charge.isError && <div className="error-banner">{(charge.error as Error).message}</div>}
      {!isLoading && (orders ?? []).length === 0 && (
        <div className="empty-state card">No hay pedidos pendientes. Toca “Nuevo pedido” para capturar uno.</div>
      )}

      <div className="cards-grid">
        {(orders ?? []).map((o: Order) => (
          <div className="card order-card" key={o.id}>
            <div className="order-card-head">
              <strong>
                #{o.id} · {TYPE_LABEL[o.type] ?? o.type}
              </strong>
              <span className="status-badge">{formatCurrency(o.total)}</span>
            </div>
            <div className="muted">
              {o.customer_name || 'Sin nombre'}
              {o.customer_phone && ` · ${o.customer_phone}`}
            </div>
            {o.customer_address && <div className="muted">📍 {o.customer_address}</div>}
            <ul className="plain-list">
              {o.items
                .filter((i) => i.status !== 'cancelado')
                .map((i) => (
                  <li key={i.id}>
                    {i.qty}× {i.product_name}{' '}
                    <span className="status-badge">{ITEM_LABEL[i.status] ?? i.status}</span>
                  </li>
                ))}
            </ul>
            <button className="btn primary" disabled={charge.isPending} onClick={() => charge.mutate(o.id)}>
              Cobrar
            </button>
          </div>
        ))}
      </div>

      {creating && (
        <NewOrderModal
          onClose={() => setCreating(false)}
          onCreated={() => queryClient.invalidateQueries({ queryKey: ['orders', 'pending'] })}
        />
      )}
    </div>
  )
}

interface CartLine extends NewItemInput {
  key: string
  productName: string
}

function NewOrderModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [type, setType] = useState<'llevar' | 'domicilio'>('llevar')
  const [customer, setCustomer] = useState({ name: '', phone: '', address: '' })
  const [cart, setCart] = useState<CartLine[]>([])
  const [active, setActive] = useState<Product | null>(null)
  const [categoryId, setCategoryId] = useState<number | null>(null)

  const { data: products } = useQuery({ queryKey: ['products'], queryFn: () => CatalogApi.products(true) })
  const { data: categories } = useQuery({ queryKey: ['categories'], queryFn: CatalogApi.categories })

  const shownCategory = categoryId ?? categories?.[0]?.id ?? null
  const total = cart.reduce((sum, l) => sum + l.qty * (products?.find((p) => p.id === l.product_id)?.price ?? 0), 0)
  const needsAddress = type === 'domicilio' && (!customer.address || !customer.phone)

  const send = useMutation({
    mutationFn: () =>
      OrdersApi.create({
        type,
        customer_name: customer.name,
        customer_phone: customer.phone,
        customer_address: customer.address,
        items: cart.map(({ product_id, qty, notes, modifier_option_ids }) => ({ product_id, qty, notes, modifier_option_ids })),
      }),
    onSuccess: () => {
      onCreated()
      onClose()
    },
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>Nuevo pedido</h3>

        <div className="segmented">
          {(['llevar', 'domicilio'] as const).map((t) => (
            <button key={t} className={type === t ? 'active' : ''} onClick={() => setType(t)}>
              {TYPE_LABEL[t]}
            </button>
          ))}
        </div>

        <div className="form-row">
          <Field label="Nombre del cliente">
            <input value={customer.name} onChange={(e) => setCustomer({ ...customer, name: e.target.value })} />
          </Field>
          <Field label={type === 'domicilio' ? 'Teléfono (obligatorio)' : 'Teléfono'}>
            <input value={customer.phone} onChange={(e) => setCustomer({ ...customer, phone: e.target.value })} />
          </Field>
        </div>
        {type === 'domicilio' && (
          <Field label="Domicilio de entrega (obligatorio)">
            <input value={customer.address} onChange={(e) => setCustomer({ ...customer, address: e.target.value })} />
          </Field>
        )}

        <div className="chip-row" style={{ marginBottom: '0.5rem' }}>
          {(categories ?? []).map((c) => (
            <button key={c.id} className={`chip ${shownCategory === c.id ? 'selected' : ''}`} onClick={() => setCategoryId(c.id)}>
              {c.name}
            </button>
          ))}
        </div>
        <div className="product-grid">
          {(products ?? [])
            .filter((p) => p.category_id === shownCategory)
            .map((p) => (
              <button key={p.id} className="product-tile" onClick={() => setActive(p)}>
                <strong>{p.name}</strong>
                <span className="muted">{formatCurrency(p.price)}</span>
              </button>
            ))}
        </div>

        <div className="section-title">Pedido</div>
        {cart.length === 0 && <p className="muted">Agrega productos del catálogo.</p>}
        {cart.map((l) => (
          <div className="cart-line" key={l.key}>
            <span>
              {l.qty}× {l.productName}
            </span>
            <button className="btn small" onClick={() => setCart(cart.filter((x) => x.key !== l.key))}>
              ✕
            </button>
          </div>
        ))}
        {cart.length > 0 && (
          <div className="cart-line" style={{ fontWeight: 600 }}>
            <span>Total</span>
            <span>{formatCurrency(total)}</span>
          </div>
        )}
        {send.isError && <div className="error-banner">{(send.error as Error).message}</div>}
        <div className="toolbar" style={{ marginTop: '0.75rem' }}>
          <button className="btn primary" disabled={cart.length === 0 || needsAddress || send.isPending} onClick={() => send.mutate()}>
            Enviar a cocina
          </button>
          <button className="btn" onClick={onClose}>
            Cancelar
          </button>
        </div>
      </div>

      {active && (
        <ProductPickerModal
          product={active}
          onClose={() => setActive(null)}
          onAdd={(item) =>
            setCart((prev) => [...prev, { ...item, key: `${Date.now()}-${Math.random()}`, productName: active.name }])
          }
        />
      )}
    </div>
  )
}
