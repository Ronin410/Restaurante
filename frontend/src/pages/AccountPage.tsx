import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { AccountsApi } from '../api/endpoints'
import { formatCurrency, splitEvenly } from '../utils/money'
import { useAuth } from '../auth/AuthContext'
import { Field } from '../components/Field'

const PAYMENT_METHODS = [
  { value: 'efectivo', label: 'Efectivo' },
  { value: 'tarjeta_debito', label: 'Tarjeta débito' },
  { value: 'tarjeta_credito', label: 'Tarjeta crédito' },
  { value: 'transferencia', label: 'Transferencia' },
  { value: 'vale_despensa', label: 'Vale de despensa' },
]

/** Genera la cuenta de una mesa y redirige a su detalle. Ruta: /cuentas/nueva?table=ID */
export function NewAccountPage() {
  const [params] = useSearchParams()
  const tableId = Number(params.get('table'))
  const navigate = useNavigate()

  const generateMutation = useMutation({
    mutationFn: () => AccountsApi.generate(tableId),
    onSuccess: (account) => navigate(`/cuentas/${account.id}`, { replace: true }),
  })

  useEffect(() => {
    if (tableId) generateMutation.mutate()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableId])

  if (generateMutation.isError) {
    return (
      <div>
        <div className="error-banner">{(generateMutation.error as Error).message}</div>
        <button className="btn" onClick={() => navigate(-1)}>
          Volver
        </button>
      </div>
    )
  }
  return <p>Generando cuenta…</p>
}

export function AccountPage() {
  const { id } = useParams()
  const accountId = Number(id)
  const queryClient = useQueryClient()
  const { hasRole } = useAuth()

  const { data: account, isLoading } = useQuery({
    queryKey: ['account', accountId],
    queryFn: () => AccountsApi.get(accountId),
    refetchInterval: 6000,
  })

  const [paymentMethod, setPaymentMethod] = useState('efectivo')
  const [paymentAmount, setPaymentAmount] = useState('')
  const [tipAmount, setTipAmount] = useState('')
  const [splitCount, setSplitCount] = useState(1)

  const [discountKind, setDiscountKind] = useState<'descuento' | 'cortesia'>('descuento')
  const [discountPercent, setDiscountPercent] = useState('')
  const [discountAmount, setDiscountAmount] = useState('')
  const [discountReason, setDiscountReason] = useState('')

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['account', accountId] })

  const payMutation = useMutation({
    mutationFn: () =>
      AccountsApi.addPayment(accountId, {
        method: paymentMethod,
        amount: Number(paymentAmount),
        tip_amount: Number(tipAmount || 0),
      }),
    onSuccess: (result) => {
      setPaymentAmount('')
      setTipAmount('')
      invalidate()
      if (result.status === 'queued') {
        window.alert('Sin conexión: el pago se guardó localmente y se sincronizará al reconectar.')
      }
    },
  })

  const discountMutation = useMutation({
    mutationFn: () =>
      AccountsApi.addDiscount(accountId, {
        kind: discountKind,
        percent: discountPercent ? Number(discountPercent) : undefined,
        amount: discountAmount ? Number(discountAmount) : undefined,
        reason: discountReason,
      }),
    onSuccess: () => {
      setDiscountPercent('')
      setDiscountAmount('')
      setDiscountReason('')
      invalidate()
    },
  })

  const closeMutation = useMutation({
    mutationFn: () => AccountsApi.close(accountId),
    onSuccess: () => {
      invalidate()
      queryClient.invalidateQueries({ queryKey: ['tables'] })
    },
  })

  if (isLoading || !account) return <p>Cargando…</p>

  const suggestedShares = splitEvenly(account.balance, splitCount)

  return (
    <div>
      <h2>Cuenta #{account.id} {account.table_number && <span className="muted">· Mesa {account.table_number}</span>}</h2>
      <span className="status-badge">{account.status}</span>

      <div className="grid" style={{ gridTemplateColumns: '1.3fr 1fr', alignItems: 'start', marginTop: '1rem' }}>
        <div className="card">
          <h3>Productos</h3>
          <table className="data-table">
            <tbody>
              {account.items.map((item) => (
                <tr key={item.id}>
                  <td>
                    {item.qty}× {item.product_name}
                  </td>
                  <td>{formatCurrency(item.line_total)}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <table className="data-table" style={{ marginTop: '1rem' }}>
            <tbody>
              <tr>
                <td>Subtotal</td>
                <td>{formatCurrency(account.subtotal)}</td>
              </tr>
              <tr>
                <td>Descuentos/cortesías</td>
                <td>-{formatCurrency(account.discount_total)}</td>
              </tr>
              <tr style={{ fontWeight: 600 }}>
                <td>Total</td>
                <td>{formatCurrency(account.total)}</td>
              </tr>
              <tr>
                <td>Pagado</td>
                <td>{formatCurrency(account.paid)}</td>
              </tr>
              <tr style={{ fontWeight: 600 }}>
                <td>Saldo pendiente</td>
                <td>{formatCurrency(account.balance)}</td>
              </tr>
              <tr>
                <td>Propinas</td>
                <td>{formatCurrency(account.tip_total)}</td>
              </tr>
            </tbody>
          </table>

          {account.discounts.length > 0 && (
            <>
              <div className="section-title">Descuentos aplicados</div>
              <ul>
                {account.discounts.map((d) => (
                  <li key={d.id} className="muted">
                    {d.kind} — {formatCurrency(d.amount)} ({d.reason})
                  </li>
                ))}
              </ul>
            </>
          )}

          {account.payments.length > 0 && (
            <>
              <div className="section-title">Pagos registrados</div>
              <ul>
                {account.payments.map((p) => (
                  <li key={p.id} className="muted">
                    {p.method}: {formatCurrency(p.amount)} {p.tip_amount > 0 && `(propina ${formatCurrency(p.tip_amount)})`}
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>

        <div>
          {account.status === 'abierta' && (
            <div className="card" style={{ marginBottom: '0.75rem' }}>
              <h3 style={{ marginTop: 0 }}>Registrar pago</h3>
              <Field label="Dividir entre (para sugerir el monto por persona)">
                <input type="number" min={1} value={splitCount} onChange={(e) => setSplitCount(Number(e.target.value))} />
              </Field>
              {splitCount > 1 && (
                <p className="muted" style={{ marginTop: '-0.5rem' }}>
                  Sugerido por persona: {suggestedShares.map((s) => formatCurrency(s)).join(' / ')}
                </p>
              )}
              <Field label="Método">
                <select value={paymentMethod} onChange={(e) => setPaymentMethod(e.target.value)}>
                  {PAYMENT_METHODS.map((m) => (
                    <option key={m.value} value={m.value}>
                      {m.label}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="Monto">
                <input type="number" step="0.01" value={paymentAmount} onChange={(e) => setPaymentAmount(e.target.value)} />
              </Field>
              <Field label="Propina (opcional, no se descuenta del saldo)">
                <input type="number" step="0.01" value={tipAmount} onChange={(e) => setTipAmount(e.target.value)} />
              </Field>
              {payMutation.isError && <div className="error-banner">{(payMutation.error as Error).message}</div>}
              <button
                className="btn primary"
                style={{ width: '100%' }}
                disabled={!paymentAmount || payMutation.isPending}
                onClick={() => payMutation.mutate()}
              >
                Registrar pago
              </button>
            </div>
          )}

          {account.status === 'abierta' && hasRole('admin', 'gerente') && (
            <div className="card" style={{ marginBottom: '0.75rem' }}>
              <h3 style={{ marginTop: 0 }}>Aplicar descuento/cortesía</h3>
              <Field label="Tipo">
                <select value={discountKind} onChange={(e) => setDiscountKind(e.target.value as 'descuento' | 'cortesia')}>
                  <option value="descuento">Descuento</option>
                  <option value="cortesia">Cortesía</option>
                </select>
              </Field>
              <Field label="Porcentaje">
                <input type="number" value={discountPercent} onChange={(e) => setDiscountPercent(e.target.value)} placeholder="ej. 10" />
              </Field>
              <Field label="o monto fijo">
                <input type="number" value={discountAmount} onChange={(e) => setDiscountAmount(e.target.value)} />
              </Field>
              <Field label="Motivo (obligatorio)">
                <textarea value={discountReason} onChange={(e) => setDiscountReason(e.target.value)} rows={2} />
              </Field>
              {discountMutation.isError && <div className="error-banner">{(discountMutation.error as Error).message}</div>}
              <button
                className="btn"
                style={{ width: '100%' }}
                disabled={!discountReason || (!discountPercent && !discountAmount)}
                onClick={() => discountMutation.mutate()}
              >
                Aplicar
              </button>
            </div>
          )}

          <div className="card">
            {account.status === 'abierta' ? (
              <button
                className="btn primary"
                style={{ width: '100%' }}
                disabled={account.balance > 0.01 || closeMutation.isPending}
                onClick={() => closeMutation.mutate()}
              >
                Cerrar cuenta
              </button>
            ) : (
              <>
                <p>
                  ✅ Cuenta cerrada. La mesa se libera sola cuando ya no queda nada por cobrar en ella
                  (el sistema no maneja el paso de limpieza).
                </p>
                <button className="btn primary" style={{ width: '100%' }} onClick={() => navigate('/mesas')}>
                  Volver a mesas
                </button>
              </>
            )}
            {closeMutation.isError && <div className="error-banner">{(closeMutation.error as Error).message}</div>}
            <button className="btn" style={{ width: '100%', marginTop: '0.5rem' }} onClick={() => window.print()}>
              Imprimir / reimprimir ticket
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
