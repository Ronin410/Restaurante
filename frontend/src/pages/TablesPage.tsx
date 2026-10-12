import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { TablesApi } from '../api/endpoints'
import type { RestaurantTable } from '../api/types'
import { formatOccupiedMinutes } from '../utils/money'
import { useAuth } from '../auth/AuthContext'
import { TableManagementModal } from '../components/TableManagementModal'

const STATUS_LABEL: Record<string, string> = {
  libre: 'Libre',
  ocupada: 'Ocupada',
  cuenta_solicitada: 'Cuenta solicitada',
  limpieza: 'En limpieza',
}

export function TablesPage() {
  const { hasRole } = useAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [showManage, setShowManage] = useState(false)

  const { data: tables, isLoading, error } = useQuery({
    queryKey: ['tables'],
    queryFn: () => TablesApi.list(),
    refetchInterval: 10000,
  })

  const openMutation = useMutation({
    mutationFn: (id: number) => TablesApi.open(id),
    onSuccess: (table) => {
      queryClient.invalidateQueries({ queryKey: ['tables'] })
      navigate(`/mesas/${table.id}`)
    },
  })

  function handleTileClick(table: RestaurantTable) {
    if (table.status === 'libre') {
      openMutation.mutate(table.id)
    } else {
      navigate(`/mesas/${table.id}`)
    }
  }

  const grouped = new Map<string, RestaurantTable[]>()
  for (const t of tables ?? []) {
    const key = t.zone_name ?? 'Sin zona'
    if (!grouped.has(key)) grouped.set(key, [])
    grouped.get(key)!.push(t)
  }

  return (
    <div>
      <div className="toolbar">
        <h2 style={{ margin: 0 }}>Mesas</h2>
        {hasRole('admin', 'gerente') && (
          <button className="btn" onClick={() => setShowManage(true)}>
            Administrar zonas y mesas
          </button>
        )}
      </div>

      {isLoading && <p>Cargando…</p>}
      {error && <div className="error-banner">{(error as Error).message}</div>}
      {!isLoading && (tables ?? []).length === 0 && (
        <div className="empty-state">
          <p>Todavía no hay mesas registradas.</p>
          {hasRole('admin', 'gerente') && (
            <button className="btn primary" onClick={() => setShowManage(true)}>
              Crear la primera zona y mesa
            </button>
          )}
        </div>
      )}

      {[...grouped.entries()].map(([zoneName, list]) => (
        <div key={zoneName} style={{ marginBottom: '1.25rem' }}>
          <h3 style={{ marginBottom: '0.5rem' }}>{zoneName}</h3>
          <div className="tables-grid">
            {list.map((table) => (
              <button
                key={table.id}
                className="table-tile"
                data-status={table.status}
                onClick={() => handleTileClick(table)}
              >
                <strong>Mesa {table.number}</strong>
                <span className="status-badge">{STATUS_LABEL[table.status] ?? table.status}</span>
                <span className="muted">Cap. {table.capacity}</span>
                {table.status === 'ocupada' && (
                  <>
                    <span className="muted">{table.current_waiter_name}</span>
                    <span className="muted">{formatOccupiedMinutes(table.occupied_minutes)}</span>
                  </>
                )}
              </button>
            ))}
          </div>
        </div>
      ))}

      {showManage && <TableManagementModal onClose={() => setShowManage(false)} />}
    </div>
  )
}
