import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CashCutsApi } from '../api/endpoints'
import { formatCurrency } from '../utils/money'
import { useAuth } from '../auth/AuthContext'
import { Field } from '../components/Field'

export function CashCutsPage() {
  const { hasRole } = useAuth()
  const queryClient = useQueryClient()
  const [openingFund, setOpeningFund] = useState('500')
  const [countedCash, setCountedCash] = useState('')

  const { data: shifts } = useQuery({ queryKey: ['shifts'], queryFn: () => CashCutsApi.listShifts() })
  const openShift = shifts?.find((s) => s.status === 'abierto')

  const { data: history } = useQuery({ queryKey: ['cuts-history'], queryFn: () => CashCutsApi.listCuts() })

  const openMutation = useMutation({
    mutationFn: () => CashCutsApi.openShift(Number(openingFund)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['shifts'] }),
  })

  const xCutMutation = useMutation({
    mutationFn: () => CashCutsApi.xCut(openShift!.id, countedCash ? Number(countedCash) : undefined),
  })

  const closeMutation = useMutation({
    mutationFn: () => CashCutsApi.closeShift(openShift!.id, Number(countedCash)),
    onSuccess: () => {
      setCountedCash('')
      queryClient.invalidateQueries({ queryKey: ['shifts'] })
      queryClient.invalidateQueries({ queryKey: ['cuts-history'] })
    },
  })

  return (
    <div>
      <h2>Cortes de caja</h2>

      {!openShift && hasRole('admin', 'gerente', 'cajero') && (
        <div className="card" style={{ maxWidth: 360, marginBottom: '1rem' }}>
          <h3 style={{ marginTop: 0 }}>Abrir turno</h3>
          <Field label="Fondo inicial">
            <input type="number" value={openingFund} onChange={(e) => setOpeningFund(e.target.value)} />
          </Field>
          {openMutation.isError && <div className="error-banner">{(openMutation.error as Error).message}</div>}
          <button className="btn primary" onClick={() => openMutation.mutate()}>
            Abrir turno
          </button>
        </div>
      )}

      {openShift && (
        <div className="card" style={{ maxWidth: 420, marginBottom: '1rem' }}>
          <h3 style={{ marginTop: 0 }}>
            Turno abierto — {openShift.cashier_name} (fondo {formatCurrency(openShift.opening_fund)})
          </h3>
          <Field label="Efectivo contado">
            <input type="number" value={countedCash} onChange={(e) => setCountedCash(e.target.value)} />
          </Field>
          <div className="toolbar">
            <button className="btn" disabled={xCutMutation.isPending} onClick={() => xCutMutation.mutate()}>
              Corte X (parcial, no cierra)
            </button>
            <button
              className="btn primary"
              disabled={!countedCash || closeMutation.isPending}
              onClick={() => closeMutation.mutate()}
            >
              Cerrar turno (corte Z)
            </button>
          </div>
          {xCutMutation.isError && <div className="error-banner">{(xCutMutation.error as Error).message}</div>}
          {closeMutation.isError && <div className="error-banner">{(closeMutation.error as Error).message}</div>}
          {xCutMutation.data && <CutSummary cut={xCutMutation.data} />}
          {closeMutation.data && <CutSummary cut={closeMutation.data.cut} />}
        </div>
      )}

      <div className="section-title">Historial de cortes</div>
      <table className="data-table">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Turno</th>
            <th>Tipo</th>
            <th>Venta</th>
            <th>Efectivo esperado</th>
            <th>Contado</th>
            <th>Diferencia</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {(history ?? []).map((c) => (
            <tr key={c.id}>
              <td>{c.created_at.slice(0, 16)}</td>
              <td>#{c.shift_id}</td>
              <td>{c.type.toUpperCase()}</td>
              <td>{formatCurrency(c.breakdown.sales_total)}</td>
              <td>{formatCurrency(c.expected_cash)}</td>
              <td>{c.counted_cash !== null ? formatCurrency(c.counted_cash) : '—'}</td>
              <td style={{ color: (c.difference ?? 0) < 0 ? 'var(--color-danger)' : 'inherit' }}>
                {c.difference !== null ? formatCurrency(c.difference) : '—'}
              </td>
              <td>
                <a href={`${import.meta.env.VITE_API_URL ?? '/api/v1'}${CashCutsApi.exportUrl(c.id)}`} target="_blank" rel="noreferrer">
                  Exportar CSV
                </a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CutSummary({ cut }: { cut: import('../api/types').CashCut }) {
  return (
    <div style={{ marginTop: '0.75rem' }}>
      <div className="section-title">Desglose por método</div>
      {(cut.breakdown.by_method ?? []).map((m) => (
        <div key={m.method} className="cart-line">
          <span>{m.method}</span>
          <span>{formatCurrency(m.total)}</span>
        </div>
      ))}
      <div className="cart-line">
        <span>Propinas</span>
        <span>{formatCurrency(cut.breakdown.tips_total)}</span>
      </div>
      <div className="cart-line">
        <span>Descuentos/cortesías</span>
        <span>{formatCurrency(cut.breakdown.discounts_total)}</span>
      </div>
      {cut.difference !== null && (
        <div className="cart-line" style={{ fontWeight: 600 }}>
          <span>Diferencia</span>
          <span>{formatCurrency(cut.difference)}</span>
        </div>
      )}
    </div>
  )
}
