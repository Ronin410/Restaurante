import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { EmployeesApi } from '../api/endpoints'
import { useAuth } from '../auth/AuthContext'
import { formatDateTime, formatDuration } from '../utils/time'

/**
 * Checador personal: cada empleado marca su propia entrada y salida desde su
 * sesión. Gerencia puede ver y corregir la asistencia de todos en Personal.
 */
export function TimeclockPage() {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const userId = user!.id

  const { data: records } = useQuery({
    queryKey: ['attendance', userId],
    queryFn: () => EmployeesApi.attendance(userId),
    refetchInterval: 30000,
  })
  const open = (records ?? []).find((r) => !r.clock_out)

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['attendance', userId] })
  const clockIn = useMutation({ mutationFn: () => EmployeesApi.clockIn(userId), onSuccess: refresh })
  const clockOut = useMutation({ mutationFn: () => EmployeesApi.clockOut(userId), onSuccess: refresh })
  const error = (clockIn.error ?? clockOut.error) as Error | null

  return (
    <div>
      <div className="page-header">
        <div>
          <h2>Mi checador</h2>
          <p className="muted">Marca tu entrada al llegar y tu salida al terminar tu turno.</p>
        </div>
      </div>

      <div className="card clock-card">
        <div className={`clock-state ${open ? 'on' : 'off'}`}>
          {open ? `Trabajando desde ${formatDateTime(open.clock_in)}` : 'Fuera de turno'}
        </div>
        {open && <div className="muted">Llevas {formatDuration(open.clock_in, null)}</div>}
        {error && <div className="error-banner">{error.message}</div>}
        {open ? (
          <button className="btn danger big" disabled={clockOut.isPending} onClick={() => clockOut.mutate()}>
            Marcar salida
          </button>
        ) : (
          <button className="btn primary big" disabled={clockIn.isPending} onClick={() => clockIn.mutate()}>
            Marcar entrada
          </button>
        )}
      </div>

      <div className="section-title">Mis últimos registros</div>
      <div className="card">
        <table className="data-table">
          <thead>
            <tr>
              <th>Entrada</th>
              <th>Salida</th>
              <th>Duración</th>
            </tr>
          </thead>
          <tbody>
            {(records ?? []).slice(0, 15).map((r) => (
              <tr key={r.id}>
                <td>{formatDateTime(r.clock_in)}</td>
                <td>{r.clock_out ? formatDateTime(r.clock_out) : <span className="status-badge">en turno</span>}</td>
                <td>{formatDuration(r.clock_in, r.clock_out)}</td>
              </tr>
            ))}
            {(records ?? []).length === 0 && (
              <tr>
                <td colSpan={3} className="muted">
                  Aún no tienes registros.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
