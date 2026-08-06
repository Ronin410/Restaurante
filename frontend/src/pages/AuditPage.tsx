import { useQuery } from '@tanstack/react-query'
import { AuditApi } from '../api/endpoints'

export function AuditPage() {
  const { data, isLoading } = useQuery({ queryKey: ['audit'], queryFn: () => AuditApi.list() })

  return (
    <div>
      <h2>Bitácora de auditoría</h2>
      <p className="muted">Registro de acciones sensibles: cancelaciones, descuentos, traspasos, cortes, etc.</p>
      {isLoading && <p>Cargando…</p>}
      <table className="data-table">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Usuario</th>
            <th>Acción</th>
            <th>Entidad</th>
            <th>Detalle</th>
          </tr>
        </thead>
        <tbody>
          {(data ?? []).map((entry) => (
            <tr key={entry.id}>
              <td>{entry.created_at}</td>
              <td>{entry.user_name ?? '—'}</td>
              <td>{entry.action}</td>
              <td>
                {entry.entity} {entry.entity_id ? `#${entry.entity_id}` : ''}
              </td>
              <td className="muted">{entry.detail}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
