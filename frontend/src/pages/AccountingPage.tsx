import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AccountingApi } from '../api/endpoints'
import { formatCurrency } from '../utils/money'
import { useAuth } from '../auth/AuthContext'
import { Field } from '../components/Field'
import type { Expense, InventoryItem, Purchase } from '../api/types'

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

/** Quien puede dar de alta, editar y dar de baja registros de contaduría. */
function useCanManage() {
  const { hasRole } = useAuth()
  return hasRole('admin', 'contador')
}

interface FieldDef {
  key: string
  label: string
  type?: 'text' | 'number' | 'date' | 'select'
  options?: { value: string; label: string }[]
  required?: boolean
}

/** Formulario modal genérico para alta/edición de un registro. */
function RecordFormModal({
  title,
  fields,
  initial,
  onSubmit,
  onClose,
  error,
  pending,
}: {
  title: string
  fields: FieldDef[]
  initial: Record<string, string>
  onSubmit: (values: Record<string, string>) => void
  onClose: () => void
  error?: Error | null
  pending?: boolean
}) {
  const [values, setValues] = useState(initial)
  const ready = fields.every((f) => !f.required || values[f.key])
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>{title}</h3>
        {fields.map((f) => (
          <Field key={f.key} label={f.label}>
            {f.type === 'select' ? (
              <select value={values[f.key]} onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}>
                {f.options!.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
            ) : (
              <input
                type={f.type ?? 'text'}
                value={values[f.key]}
                onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}
              />
            )}
          </Field>
        ))}
        {error && <div className="error-banner">{error.message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={!ready || pending} onClick={() => onSubmit(values)}>
            Guardar
          </button>
          <button className="btn" onClick={onClose}>
            Cancelar
          </button>
        </div>
      </div>
    </div>
  )
}

function ExpensesTab() {
  const canManage = useCanManage()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['expenses'], queryFn: () => AccountingApi.expenses() })
  const [editing, setEditing] = useState<{ item: Expense | null } | null>(null)

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['expenses'] })
  const save = useMutation({
    mutationFn: (v: Record<string, string>) => {
      const input = { category: v.category, description: v.description, amount: Number(v.amount), supplier: v.supplier, date: v.date || undefined }
      return editing?.item ? AccountingApi.updateExpense(editing.item.id, input) : AccountingApi.createExpense(input)
    },
    onSuccess: () => {
      setEditing(null)
      refresh()
    },
  })
  const remove = useMutation({ mutationFn: (id: number) => AccountingApi.deleteExpense(id), onSuccess: refresh })

  const fields: FieldDef[] = [
    { key: 'category', label: 'Categoría' },
    { key: 'description', label: 'Descripción', required: true },
    { key: 'supplier', label: 'Proveedor' },
    { key: 'amount', label: 'Monto', type: 'number', required: true },
    { key: 'date', label: 'Fecha (vacío = hoy)', type: 'date' },
  ]

  return (
    <div className="card">
      <div className="toolbar">
        {canManage && (
          <button className="btn primary" onClick={() => setEditing({ item: null })}>
            + Registrar gasto
          </button>
        )}
        <a className="btn" href={`${import.meta.env.VITE_API_URL ?? '/api/v1'}/expenses/export`} target="_blank" rel="noreferrer">
          Exportar CSV
        </a>
      </div>
      {remove.isError && <div className="error-banner">{(remove.error as Error).message}</div>}
      <table className="data-table">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Categoría</th>
            <th>Descripción</th>
            <th>Proveedor</th>
            <th>Monto</th>
            {canManage && <th></th>}
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
              {canManage && (
                <td>
                  <RowActions
                    onEdit={() => setEditing({ item: e })}
                    onDelete={() => confirm(`¿Eliminar el gasto "${e.description}"?`) && remove.mutate(e.id)}
                  />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {editing && (
        <RecordFormModal
          title={editing.item ? 'Editar gasto' : 'Registrar gasto'}
          fields={fields}
          initial={{
            category: editing.item?.category ?? '',
            description: editing.item?.description ?? '',
            supplier: editing.item?.supplier ?? '',
            amount: editing.item ? String(editing.item.amount) : '',
            date: editing.item?.date ?? '',
          }}
          onSubmit={(v) => save.mutate(v)}
          onClose={() => setEditing(null)}
          error={save.error as Error | null}
          pending={save.isPending}
        />
      )}
    </div>
  )
}

function RowActions({ onEdit, onDelete, deleteLabel = 'Eliminar' }: { onEdit?: () => void; onDelete: () => void; deleteLabel?: string }) {
  return (
    <div className="manage-actions">
      {onEdit && (
        <button className="btn small" onClick={onEdit}>
          Editar
        </button>
      )}
      <button className="btn small danger" onClick={onDelete}>
        {deleteLabel}
      </button>
    </div>
  )
}

function PurchasesTab() {
  const canManage = useCanManage()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['purchases'], queryFn: () => AccountingApi.purchases() })
  const [editing, setEditing] = useState<{ item: Purchase | null } | null>(null)

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['purchases'] })
  const save = useMutation({
    mutationFn: (v: Record<string, string>) => {
      const input = { supplier: v.supplier, description: v.description, amount: Number(v.amount), date: v.date || undefined }
      return editing?.item ? AccountingApi.updatePurchase(editing.item.id, input) : AccountingApi.createPurchase(input)
    },
    onSuccess: () => {
      setEditing(null)
      refresh()
    },
  })
  const remove = useMutation({ mutationFn: (id: number) => AccountingApi.deletePurchase(id), onSuccess: refresh })

  const fields: FieldDef[] = [
    { key: 'supplier', label: 'Proveedor', required: true },
    { key: 'description', label: 'Descripción' },
    { key: 'amount', label: 'Monto', type: 'number', required: true },
    { key: 'date', label: 'Fecha (vacío = hoy)', type: 'date' },
  ]

  return (
    <div className="card">
      {canManage && (
        <div className="toolbar">
          <button className="btn primary" onClick={() => setEditing({ item: null })}>
            + Registrar compra
          </button>
        </div>
      )}
      {remove.isError && <div className="error-banner">{(remove.error as Error).message}</div>}
      <table className="data-table">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Proveedor</th>
            <th>Descripción</th>
            <th>Monto</th>
            {canManage && <th></th>}
          </tr>
        </thead>
        <tbody>
          {(data ?? []).map((p) => (
            <tr key={p.id}>
              <td>{p.date}</td>
              <td>{p.supplier}</td>
              <td>{p.description}</td>
              <td>{formatCurrency(p.amount)}</td>
              {canManage && (
                <td>
                  <RowActions
                    onEdit={() => setEditing({ item: p })}
                    onDelete={() => confirm(`¿Eliminar la compra a "${p.supplier}"?`) && remove.mutate(p.id)}
                  />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {editing && (
        <RecordFormModal
          title={editing.item ? 'Editar compra' : 'Registrar compra'}
          fields={fields}
          initial={{
            supplier: editing.item?.supplier ?? '',
            description: editing.item?.description ?? '',
            amount: editing.item ? String(editing.item.amount) : '',
            date: editing.item?.date ?? '',
          }}
          onSubmit={(v) => save.mutate(v)}
          onClose={() => setEditing(null)}
          error={save.error as Error | null}
          pending={save.isPending}
        />
      )}
    </div>
  )
}

function InventoryTab() {
  const canManage = useCanManage()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['inventory'], queryFn: AccountingApi.inventory })
  const [movement, setMovement] = useState<{ id: number; type: string; qty: string; reason: string } | null>(null)
  const [editing, setEditing] = useState<{ item: InventoryItem | null } | null>(null)
  const [historyFor, setHistoryFor] = useState<InventoryItem | null>(null)

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['inventory'] })

  const addMovement = useMutation({
    mutationFn: () => AccountingApi.addMovement(movement!.id, { type: movement!.type, qty: Number(movement!.qty), reason: movement!.reason }),
    onSuccess: () => {
      setMovement(null)
      refresh()
    },
  })
  const save = useMutation({
    mutationFn: (v: Record<string, string>) => {
      const input = { name: v.name, unit: v.unit || 'pza', min_stock: Number(v.min_stock || 0) }
      return editing?.item ? AccountingApi.updateInventoryItem(editing.item.id, input) : AccountingApi.createInventoryItem(input)
    },
    onSuccess: () => {
      setEditing(null)
      refresh()
    },
  })
  const remove = useMutation({ mutationFn: (id: number) => AccountingApi.deleteInventoryItem(id), onSuccess: refresh })

  const fields: FieldDef[] = [
    { key: 'name', label: 'Nombre del insumo', required: true },
    { key: 'unit', label: 'Unidad (kg, lt, pza…)' },
    { key: 'min_stock', label: 'Existencia mínima (alerta)', type: 'number' },
  ]

  return (
    <div className="card">
      {canManage && (
        <div className="toolbar">
          <button className="btn primary" onClick={() => setEditing({ item: null })}>
            + Nuevo insumo
          </button>
        </div>
      )}
      {remove.isError && <div className="error-banner">{(remove.error as Error).message}</div>}
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
                <div className="manage-actions">
                  <button className="btn small" onClick={() => setHistoryFor(i)}>
                    Historial
                  </button>
                  {canManage && (
                    <>
                      <button className="btn small" onClick={() => setMovement({ id: i.id, type: 'entrada', qty: '', reason: '' })}>
                        Movimiento
                      </button>
                      <button className="btn small" onClick={() => setEditing({ item: i })}>
                        Editar
                      </button>
                      <button
                        className="btn small danger"
                        onClick={() => confirm(`¿Eliminar "${i.name}" y todo su historial de movimientos?`) && remove.mutate(i.id)}
                      >
                        Eliminar
                      </button>
                    </>
                  )}
                </div>
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

      {editing && (
        <RecordFormModal
          title={editing.item ? `Editar ${editing.item.name}` : 'Nuevo insumo'}
          fields={fields}
          initial={{
            name: editing.item?.name ?? '',
            unit: editing.item?.unit ?? 'pza',
            min_stock: editing.item ? String(editing.item.min_stock) : '0',
          }}
          onSubmit={(v) => save.mutate(v)}
          onClose={() => setEditing(null)}
          error={save.error as Error | null}
          pending={save.isPending}
        />
      )}

      {historyFor && (
        <MovementHistoryModal
          item={historyFor}
          canManage={canManage}
          onClose={() => setHistoryFor(null)}
          onChanged={(updated) => {
            setHistoryFor(updated)
            refresh()
          }}
        />
      )}
    </div>
  )
}

function MovementHistoryModal({
  item,
  canManage,
  onClose,
  onChanged,
}: {
  item: InventoryItem
  canManage: boolean
  onClose: () => void
  onChanged: (updated: InventoryItem) => void
}) {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['inventory-movements', item.id], queryFn: () => AccountingApi.movements(item.id) })
  const remove = useMutation({
    mutationFn: (movementId: number) => AccountingApi.deleteMovement(item.id, movementId),
    onSuccess: (updated) => {
      queryClient.invalidateQueries({ queryKey: ['inventory-movements', item.id] })
      onChanged(updated)
    },
  })
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" style={{ width: 'min(560px, 95vw)' }} onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>
          Historial · {item.name} ({item.stock} {item.unit})
        </h3>
        {(data ?? []).length === 0 && <p className="muted">Sin movimientos.</p>}
        {(data ?? []).map((m) => (
          <div className="manage-row" key={m.id}>
            <div className="manage-info">
              <span>
                {m.type} · {m.qty} {item.unit}
              </span>
              <span className="muted">
                {m.created_at} {m.reason && `· ${m.reason}`}
              </span>
            </div>
            {canManage && (
              <RowActions
                deleteLabel="Deshacer"
                onDelete={() => confirm('¿Eliminar este movimiento? El existente se corrige automáticamente.') && remove.mutate(m.id)}
              />
            )}
          </div>
        ))}
        {remove.isError && <div className="error-banner">{(remove.error as Error).message}</div>}
        <button className="btn" onClick={onClose}>
          Cerrar
        </button>
      </div>
    </div>
  )
}

function InvoicesTab() {
  const canManage = useCanManage()
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['invoices'], queryFn: AccountingApi.invoices })
  const [creating, setCreating] = useState(false)

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['invoices'] })
  const create = useMutation({
    mutationFn: (v: Record<string, string>) =>
      AccountingApi.createInvoice({ account_id: Number(v.account_id), rfc: v.rfc, razon_social: v.razon_social, uso_cfdi: v.uso_cfdi || undefined }),
    onSuccess: () => {
      setCreating(false)
      refresh()
    },
  })
  const cancel = useMutation({ mutationFn: (id: number) => AccountingApi.cancelInvoice(id), onSuccess: refresh })

  return (
    <div className="card">
      <p className="muted">
        Facturación CFDI 4.0: esta versión simula el timbrado (no está conectada a un PAC autorizado por el SAT). Sirve
        para probar el flujo; antes de facturar de verdad hay que contratar un proveedor de timbrado. Una factura no se
        borra: se cancela.
      </p>
      {canManage && (
        <div className="toolbar">
          <button className="btn primary" onClick={() => setCreating(true)}>
            + Nueva factura
          </button>
        </div>
      )}
      {cancel.isError && <div className="error-banner">{(cancel.error as Error).message}</div>}
      <table className="data-table">
        <thead>
          <tr>
            <th>Cuenta</th>
            <th>RFC</th>
            <th>Razón social</th>
            <th>Uso CFDI</th>
            <th>Estatus</th>
            <th>UUID (simulado)</th>
            {canManage && <th></th>}
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
              {canManage && (
                <td>
                  {inv.status !== 'cancelada' && (
                    <RowActions
                      deleteLabel="Cancelar"
                      onDelete={() => confirm(`¿Cancelar la factura de ${inv.razon_social}?`) && cancel.mutate(inv.id)}
                    />
                  )}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {creating && (
        <RecordFormModal
          title="Nueva factura"
          fields={[
            { key: 'account_id', label: 'Número de cuenta', type: 'number', required: true },
            { key: 'rfc', label: 'RFC', required: true },
            { key: 'razon_social', label: 'Razón social', required: true },
            { key: 'uso_cfdi', label: 'Uso CFDI (vacío = G03)' },
          ]}
          initial={{ account_id: '', rfc: '', razon_social: '', uso_cfdi: '' }}
          onSubmit={(v) => create.mutate(v)}
          onClose={() => setCreating(false)}
          error={create.error as Error | null}
          pending={create.isPending}
        />
      )}
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
