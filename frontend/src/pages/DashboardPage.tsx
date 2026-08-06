import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { KpisApi } from '../api/endpoints'
import { Field } from '../components/Field'
import { formatCurrency, formatOccupiedMinutes } from '../utils/money'

function defaultFrom() {
  const d = new Date()
  d.setDate(d.getDate() - 6)
  return d.toISOString().slice(0, 10)
}
function today() {
  return new Date().toISOString().slice(0, 10)
}

export function DashboardPage() {
  const [from, setFrom] = useState(defaultFrom())
  const [to, setTo] = useState(today())

  const { data, isLoading, error } = useQuery({
    queryKey: ['kpis', from, to],
    queryFn: () => KpisApi.dashboard(from, to),
  })

  return (
    <div>
      <h2>Panel de indicadores (KPIs)</h2>
      <div className="toolbar">
        <Field label="Desde" style={{ marginBottom: 0 }}>
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
        </Field>
        <Field label="Hasta" style={{ marginBottom: 0 }}>
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
        </Field>
      </div>

      {isLoading && <p>Cargando…</p>}
      {error && <div className="error-banner">{(error as Error).message}</div>}

      {data && (
        <>
          <div className="kpi-grid">
            <div className="card kpi-tile">
              <div className="value">{formatCurrency(data.venta_total)}</div>
              <div className="label">Venta total (neta)</div>
            </div>
            <div className="card kpi-tile">
              <div className="value">{formatCurrency(data.ticket_promedio)}</div>
              <div className="label">Ticket promedio por mesa</div>
            </div>
            <div className="card kpi-tile">
              <div className="value">{formatCurrency(data.propinas_total)}</div>
              <div className="label">Propinas totales</div>
            </div>
            <div className="card kpi-tile">
              <div className="value">{formatOccupiedMinutes(Math.round(data.rotacion_minutos_promedio))}</div>
              <div className="label">Rotación promedio de mesas</div>
            </div>
            <div className="card kpi-tile">
              <div className="value" style={{ color: data.faltantes_sobrantes_acumulado < 0 ? 'var(--color-danger)' : 'inherit' }}>
                {formatCurrency(data.faltantes_sobrantes_acumulado)}
              </div>
              <div className="label">Faltantes/sobrantes acumulados</div>
            </div>
            <div className="card kpi-tile">
              <div className="value">{data.producto_mas_vendido?.product_name ?? '—'}</div>
              <div className="label">Producto más vendido ({data.producto_mas_vendido?.qty ?? 0} uds)</div>
            </div>
            <div className="card kpi-tile">
              <div className="value">{data.producto_menos_vendido?.product_name ?? '—'}</div>
              <div className="label">Producto menos vendido ({data.producto_menos_vendido?.qty ?? 0} uds)</div>
            </div>
          </div>

          <div className="grid" style={{ gridTemplateColumns: '1fr 1fr', alignItems: 'start' }}>
            <div className="card">
              <h3>Venta por método de pago</h3>
              <table className="data-table">
                <tbody>
                  {(data.venta_por_metodo ?? []).map((m) => (
                    <tr key={m.method}>
                      <td>{m.method}</td>
                      <td>{formatCurrency(m.total)}</td>
                    </tr>
                  ))}
                  {(data.venta_por_metodo ?? []).length === 0 && (
                    <tr>
                      <td colSpan={2} className="muted">
                        Sin datos en el rango
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <div className="card">
              <h3>Propinas por mesero</h3>
              <table className="data-table">
                <tbody>
                  {(data.propinas_por_mesero ?? []).map((w) => (
                    <tr key={w.waiter_id}>
                      <td>{w.waiter_name}</td>
                      <td>{formatCurrency(w.tips)}</td>
                    </tr>
                  ))}
                  {(data.propinas_por_mesero ?? []).length === 0 && (
                    <tr>
                      <td colSpan={2} className="muted">
                        Sin datos en el rango
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <div className="card">
              <h3>Venta por mesero</h3>
              <table className="data-table">
                <tbody>
                  {(data.venta_por_mesero ?? []).map((w) => (
                    <tr key={w.waiter_id}>
                      <td>{w.waiter_name}</td>
                      <td>{formatCurrency(w.total)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="card">
              <h3>Venta por turno de caja</h3>
              <table className="data-table">
                <tbody>
                  {(data.venta_por_turno ?? []).map((s) => (
                    <tr key={s.shift_id}>
                      <td>
                        {s.cashier_name} <span className="muted">({s.opened_at.slice(0, 16)})</span>
                      </td>
                      <td>{formatCurrency(s.total)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
