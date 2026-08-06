import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AccountingApi } from '../api/endpoints'
import { formatCurrency } from '../utils/money'
import { useAuth } from '../auth/AuthContext'
import { Field } from '../components/Field'

type Tab = 'ventas' | 'gastos' | 'compras' | 'inventario' | 'facturacion' | 'impuestos' | 'resultados'

export function AccountingPage() {
  const [tab, setTab] = useState<Tab>('ventas')
  const tabs: { key: Tab; label: string }[] = [
    { key: 'ventas', label: 'Reporte de ventas' },
    { key: 'gastos', label: 'Gastos' },
    { key: 'compras', label: 'Compras' },
    { key: 'inventario', label: 'Inventario' },
    { key: 'facturacion', label: 'Facturación (CFDI)' },
    { key: 'impuestos', label: 'Impuestos' },
    { key: 'resultados', label: 'Estado de resultados' },
  ]

  return (
    <div>
      <h2>Contaduría</h2>
      <div className="chip-row" style={{ marginBottom: '1rem' }}>
        {tabs.map((t) => (
          <button key={t.key} className={`chip ${tab === t.key ? 'selected' : ''}`} onClick={() => setTab(t.key)}>
            {t.label}
          </button>
        ))}
      </div>
      {tab === 'ventas' && <SalesTab />}
      {tab === 'gastos' && <ExpensesTab />}
      {tab === 'compras' && <PurchasesTab />}
      {tab === 'inventario' && <InventoryTab />}
      {tab === 'facturacion' && <InvoicesTab />}
      {tab === 'impuestos' && <TaxesTab />}
      {tab === 'resultados' && <PnlTab />}
    </div>
  )
}

function useDateRange() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const picker = (
    <div className="toolbar">
      <Field label="Desde" style={{ marginBottom: 0 }}>
        <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
      </Field>
      <Field label="Hasta" style={{ marginBottom: 0 }}>
        <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
      </Field>
    </div>
  )
  return { from, to, picker }
}

function SalesTab() {
  const { from, to, picker } = useDateRange()
  const { data } = useQuery({ queryKey: ['sales-report', from, to], queryFn: () => AccountingApi.salesReport(from, to) })

  return (
    <div>
      {picker}
      <div className="toolbar">
        <a
          className="btn"
          href={`${import.meta.env.VITE_API_URL ?? '/api/v1'}/reports/sales/export?from=${from}&to=${to}`}
          target="_blank"
          rel="noreferrer"
        >
          Exportar CSV
        </a>
      </div>
      {data && (
        <div className="grid" style={{ gridTemplateColumns: '1fr 1fr 1fr', alignItems: 'start' }}>
          <div className="card">
            <h3>Total: {formatCurrency(data.total)}</h3>
            <div className="section-title">Por día</div>
            <table className="data-table">
              <tbody>
                {(data.by_day ?? []).map((d) => (
                  <tr key={d.date}>
                    <td>{d.date}</td>
                    <td>{formatCurrency(d.total)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="card">
            <div className="section-title" style={{ marginTop: 0 }}>
              Por producto
            </div>
            <table className="data-table">
              <tbody>
                {(data.by_product ?? []).map((p) => (
                  <tr key={p.product_name}>
                    <td>{p.product_name}</td>
                    <td>{p.qty}</td>
                    <td>{formatCurrency(p.total)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="card">
            <div className="section-title" style={{ marginTop: 0 }}>
              Por categoría
            </div>
            <table className="data-table">
              <tbody>
                {(data.by_category ?? []).map((c) => (
                  <tr key={c.category}>
                    <td>{c.category}</td>
                    <td>{formatCurrency(c.total)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}

function ExpensesTab() {
  const { hasRole } = useAuth()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['expenses'], queryFn: () => AccountingApi.expenses() })
  const [form, setForm] = useState({ category: '', description: '', amount: '', supplier: '' })

  const create = useMutation({
    mutationFn: () =>
      AccountingApi.createExpense({
        category: form.category,
        description: form.description,
        amount: Number(form.amount),
        supplier: form.supplier,
      }),
    onSuccess: () => {
      setForm({ category: '', description: '', amount: '', supplier: '' })
      queryClient.invalidateQueries({ queryKey: ['expenses'] })
    },
  })

  return (
    <div className="grid" style={{ gridTemplateColumns: hasRole('admin') ? '2fr 1fr' : '1fr', alignItems: 'start' }}>
      <div className="card">
        <table className="data-table">
          <thead>
            <tr>
              <th>Fecha</th>
              <th>Categoría</th>
              <th>Descripción</th>
              <th>Proveedor</th>
              <th>Monto</th>
            </tr>
          </thead>
          <tbody>
            {(data ?? []).map((e) => (
              <tr key={e.id}>
                <td>{e.date}</td>
                <td>{e.category}</td>
                <td>{e.description}</td>
                <td>{e.supplier}</td>
                <td>{formatCurrency(e.amount)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <a className="btn" href={`${import.meta.env.VITE_API_URL ?? '/api/v1'}/expenses/export`} target="_blank" rel="noreferrer">
          Exportar CSV
        </a>
      </div>
      {hasRole('admin') && (
        <div className="card">
          <h3 style={{ marginTop: 0 }}>Registrar gasto</h3>
          <Field label="Categoría">
            <input value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} />
          </Field>
          <Field label="Descripción">
            <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </Field>
          <Field label="Proveedor">
            <input value={form.supplier} onChange={(e) => setForm({ ...form, supplier: e.target.value })} />
          </Field>
          <Field label="Monto">
            <input type="number" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} />
          </Field>
          {create.isError && <div className="error-banner">{(create.error as Error).message}</div>}
          <button className="btn primary" disabled={!form.description || !form.amount} onClick={() => create.mutate()}>
            Guardar
          </button>
        </div>
      )}
    </div>
  )
}

function PurchasesTab() {
  const { hasRole } = useAuth()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['purchases'], queryFn: () => AccountingApi.purchases() })
  const [form, setForm] = useState({ supplier: '', description: '', amount: '' })

  const create = useMutation({
    mutationFn: () => AccountingApi.createPurchase({ supplier: form.supplier, description: form.description, amount: Number(form.amount) }),
    onSuccess: () => {
      setForm({ supplier: '', description: '', amount: '' })
      queryClient.invalidateQueries({ queryKey: ['purchases'] })
    },
  })

  return (
    <div className="grid" style={{ gridTemplateColumns: hasRole('admin') ? '2fr 1fr' : '1fr', alignItems: 'start' }}>
      <div className="card">
        <table className="data-table">
          <thead>
            <tr>
              <th>Fecha</th>
              <th>Proveedor</th>
              <th>Descripción</th>
              <th>Monto</th>
            </tr>
          </thead>
          <tbody>
            {(data ?? []).map((p) => (
              <tr key={p.id}>
                <td>{p.date}</td>
                <td>{p.supplier}</td>
                <td>{p.description}</td>
                <td>{formatCurrency(p.amount)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {hasRole('admin') && (
        <div className="card">
          <h3 style={{ marginTop: 0 }}>Registrar compra</h3>
          <Field label="Proveedor">
            <input value={form.supplier} onChange={(e) => setForm({ ...form, supplier: e.target.value })} />
          </Field>
          <Field label="Descripción">
            <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </Field>
          <Field label="Monto">
            <input type="number" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} />
          </Field>
          {create.isError && <div className="error-banner">{(create.error as Error).message}</div>}
          <button className="btn primary" disabled={!form.supplier || !form.amount} onClick={() => create.mutate()}>
            Guardar
          </button>
        </div>
      )}
    </div>
  )
}

function InventoryTab() {
  const { hasRole } = useAuth()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['inventory'], queryFn: AccountingApi.inventory })
  const [movement, setMovement] = useState<{ id: number; type: string; qty: string; reason: string } | null>(null)

  const addMovement = useMutation({
    mutationFn: () => AccountingApi.addMovement(movement!.id, { type: movement!.type, qty: Number(movement!.qty), reason: movement!.reason }),
    onSuccess: () => {
      setMovement(null)
      queryClient.invalidateQueries({ queryKey: ['inventory'] })
    },
  })

  return (
    <div className="card">
      <table className="data-table">
        <thead>
          <tr>
            <th>Insumo</th>
            <th>Existencia</th>
            <th>Mínimo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {(data ?? []).map((i) => (
            <tr key={i.id}>
              <td>{i.name}</td>
              <td style={{ color: i.low_stock ? 'var(--color-danger)' : 'inherit' }}>
                {i.stock} {i.unit}
              </td>
              <td>{i.min_stock}</td>
              <td>
                {hasRole('admin') && (
                  <button className="btn" onClick={() => setMovement({ id: i.id, type: 'entrada', qty: '', reason: '' })}>
                    Movimiento
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {movement && (
        <div className="modal-backdrop" onClick={() => setMovement(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Movimiento de inventario</h3>
            <Field label="Tipo">
              <select value={movement.type} onChange={(e) => setMovement({ ...movement, type: e.target.value })}>
                <option value="entrada">Entrada</option>
                <option value="salida">Salida</option>
                <option value="merma">Merma</option>
              </select>
            </Field>
            <Field label="Cantidad">
              <input type="number" value={movement.qty} onChange={(e) => setMovement({ ...movement, qty: e.target.value })} />
            </Field>
            <Field label="Motivo">
              <input value={movement.reason} onChange={(e) => setMovement({ ...movement, reason: e.target.value })} />
            </Field>
            {addMovement.isError && <div className="error-banner">{(addMovement.error as Error).message}</div>}
            <button className="btn primary" disabled={!movement.qty} onClick={() => addMovement.mutate()}>
              Guardar
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function InvoicesTab() {
  const { data } = useQuery({ queryKey: ['invoices'], queryFn: AccountingApi.invoices })
  return (
    <div className="card">
      <p className="muted">
        Facturación CFDI 4.0: esta versión simula el timbrado (no está conectada a un PAC autorizado por el SAT). Sirve
        para probar el flujo; antes de facturar de verdad hay que contratar un proveedor de timbrado.
      </p>
      <table className="data-table">
        <thead>
          <tr>
            <th>Cuenta</th>
            <th>RFC</th>
            <th>Razón social</th>
            <th>Uso CFDI</th>
            <th>Estatus</th>
            <th>UUID (simulado)</th>
          </tr>
        </thead>
        <tbody>
          {(data ?? []).map((inv) => (
            <tr key={inv.id}>
              <td>#{inv.account_id}</td>
              <td>{inv.rfc}</td>
              <td>{inv.razon_social}</td>
              <td>{inv.uso_cfdi}</td>
              <td>{inv.status}</td>
              <td className="muted">{inv.uuid_fiscal}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function TaxesTab() {
  const { from, to, picker } = useDateRange()
  const { data } = useQuery({ queryKey: ['tax-report', from, to], queryFn: () => AccountingApi.taxReport(from, to) })
  return (
    <div>
      {picker}
      {data && (
        <div className="card">
          <table className="data-table">
            <tbody>
              <tr>
                <td>Venta total</td>
                <td>{formatCurrency(data.venta_total)}</td>
              </tr>
              <tr>
                <td>IVA trasladado</td>
                <td>{formatCurrency(data.iva_trasladado)}</td>
              </tr>
              <tr>
                <td>ISR retenido</td>
                <td>{formatCurrency(data.isr_retenido)}</td>
              </tr>
            </tbody>
          </table>
          <p className="muted">{data.note}</p>
        </div>
      )}
    </div>
  )
}

function PnlTab() {
  const { from, to, picker } = useDateRange()
  const { data } = useQuery({ queryKey: ['pnl', from, to], queryFn: () => AccountingApi.pnl(from, to) })
  return (
    <div>
      {picker}
      {data && (
        <div className="card">
          <table className="data-table">
            <tbody>
              <tr>
                <td>Ingresos</td>
                <td>{formatCurrency(data.ingresos)}</td>
              </tr>
              <tr>
                <td>Gastos</td>
                <td>-{formatCurrency(data.gastos)}</td>
              </tr>
              <tr>
                <td>Compras</td>
                <td>-{formatCurrency(data.compras)}</td>
              </tr>
              <tr>
                <td>Nómina</td>
                <td>-{formatCurrency(data.nomina)}</td>
              </tr>
              <tr style={{ fontWeight: 600 }}>
                <td>Utilidad</td>
                <td>{formatCurrency(data.utilidad)}</td>
              </tr>
            </tbody>
          </table>
          <p className="muted">{data.note}</p>
        </div>
      )}
    </div>
  )
}
