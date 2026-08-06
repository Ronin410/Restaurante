import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { TablesApi } from '../api/endpoints'
import { Field } from '../components/Field'
import type { RestaurantTable } from '../api/types'
import { formatOccupiedMinutes } from '../utils/money'
import { useAuth } from '../auth/AuthContext'

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
  const [showNewTable, setShowNewTable] = useState(false)

  const { data: tables, isLoading, error } = useQuery({
    queryKey: ['tables'],
    queryFn: TablesApi.list,
    refetchInterval: 10000,
  })
  const { data: zones } = useQuery({ queryKey: ['zones'], queryFn: TablesApi.zones })

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
          <button className="btn" onClick={() => setShowNewTable(true)}>
            + Nueva mesa
          </button>
        )}
      </div>

      {isLoading && <p>Cargando…</p>}
      {error && <div className="error-banner">{(error as Error).message}</div>}

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

      {showNewTable && zones && (
        <NewTableModal zones={zones} onClose={() => setShowNewTable(false)} />
      )}
    </div>
  )
}

function NewTableModal({ zones, onClose }: { zones: { id: number; name: string }[]; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [zoneId, setZoneId] = useState(zones[0]?.id ?? 0)
  const [number, setNumber] = useState('')
  const [capacity, setCapacity] = useState(4)

  const createMutation = useMutation({
    mutationFn: () => TablesApi.create({ zone_id: zoneId, number, capacity }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tables'] })
      onClose()
    },
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>Nueva mesa</h3>
        <Field label="Zona">
          <select value={zoneId} onChange={(e) => setZoneId(Number(e.target.value))}>
            {zones.map((z) => (
              <option key={z.id} value={z.id}>
                {z.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Número">
          <input value={number} onChange={(e) => setNumber(e.target.value)} />
        </Field>
        <Field label="Capacidad">
          <input type="number" value={capacity} onChange={(e) => setCapacity(Number(e.target.value))} />
        </Field>
        {createMutation.isError && <div className="error-banner">{(createMutation.error as Error).message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={!number} onClick={() => createMutation.mutate()}>
            Crear
          </button>
          <button className="btn" onClick={onClose}>
            Cancelar
          </button>
        </div>
      </div>
    </div>
  )
}
