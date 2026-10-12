import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { TablesApi } from '../api/endpoints'
import type { RestaurantTable, Zone } from '../api/types'
import { Field } from './Field'

/**
 * Panel de administración de zonas y mesas (terrazas, interior, etc.):
 * crear, renombrar/editar y eliminar. Las mesas no se borran de verdad
 * (quedarían huérfanas sus comandas/cuentas pasadas) sino que se desactivan
 * — el backend las saca del mapa de mesas y no deja abrirlas de nuevo hasta
 * reactivarlas. Las zonas sí se borran de verdad, pero solo si ya no tienen
 * ninguna mesa asignada.
 */
export function TableManagementModal({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient()
  const [showInactive, setShowInactive] = useState(false)
  const [zoneForm, setZoneForm] = useState<{ id: number; name: string } | null>(null)
  const [tableForm, setTableForm] = useState<{ table: RestaurantTable | null } | null>(null)

  const { data: zones } = useQuery({ queryKey: ['zones'], queryFn: TablesApi.zones })
  // Siempre se trae el inventario completo (activas + eliminadas): así el
  // conteo de mesas por zona es siempre real, sin importar si la casilla
  // "mostrar eliminadas" está prendida o no (si dependiera de ella, una
  // zona podría mostrar "0 mesas" y aun así no dejarse borrar porque sí
  // tiene una mesa eliminada, lo cual es muy confuso). El checkbox solo
  // filtra qué filas se muestran en la sección "Mesas" de abajo.
  const { data: allTables } = useQuery({
    queryKey: ['tables', 'manage-all'],
    queryFn: () => TablesApi.list(true),
  })
  const tables = showInactive ? allTables : allTables?.filter((t) => t.active)

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ['tables'] })
    queryClient.invalidateQueries({ queryKey: ['zones'] })
  }

  const deleteZone = useMutation({
    mutationFn: (id: number) => TablesApi.deleteZone(id),
    onSuccess: invalidate,
  })
  const deleteTable = useMutation({
    mutationFn: (id: number) => TablesApi.remove(id),
    onSuccess: invalidate,
  })
  const reactivateTable = useMutation({
    mutationFn: (id: number) => TablesApi.reactivate(id),
    onSuccess: invalidate,
  })

  const tablesByZone = new Map<number, RestaurantTable[]>()
  for (const t of tables ?? []) {
    if (!tablesByZone.has(t.zone_id)) tablesByZone.set(t.zone_id, [])
    tablesByZone.get(t.zone_id)!.push(t)
  }

  // Conteo real por zona (activas + eliminadas), independiente del filtro
  // de la sección "Mesas" — ver comentario arriba de allTables.
  const totalByZone = new Map<number, number>()
  for (const t of allTables ?? []) {
    totalByZone.set(t.zone_id, (totalByZone.get(t.zone_id) ?? 0) + 1)
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" style={{ width: 'min(680px, 95vw)' }} onClick={(e) => e.stopPropagation()}>
        <div className="toolbar" style={{ marginBottom: '0.25rem' }}>
          <h3 style={{ margin: 0, flex: 1 }}>Administrar zonas y mesas</h3>
          <button className="btn small" onClick={onClose}>
            Cerrar
          </button>
        </div>

        <div className="section-title" style={{ marginTop: '0.5rem' }}>
          Zonas
        </div>
        {(zones ?? []).map((zone) => (
          <div className="manage-row" key={zone.id}>
            <div className="manage-info">
              <strong>{zone.name}</strong>
              <span className="muted">{totalByZone.get(zone.id) ?? 0} mesa(s)</span>
            </div>
            <div className="manage-actions">
              <button className="btn small" onClick={() => setZoneForm({ id: zone.id, name: zone.name })}>
                Renombrar
              </button>
              <button
                className="btn small danger"
                disabled={deleteZone.isPending || (totalByZone.get(zone.id) ?? 0) > 0}
                title={(totalByZone.get(zone.id) ?? 0) > 0 ? 'Mueve o elimina sus mesas antes de borrar la zona' : undefined}
                onClick={() => {
                  if (confirm(`¿Eliminar la zona "${zone.name}"?`)) {
                    deleteZone.mutate(zone.id)
                  }
                }}
              >
                Eliminar
              </button>
            </div>
          </div>
        ))}
        {deleteZone.isError && <div className="error-banner">{(deleteZone.error as Error).message}</div>}
        <button className="btn small" style={{ marginTop: '0.5rem' }} onClick={() => setZoneForm({ id: 0, name: '' })}>
          + Nueva zona
        </button>

        <div className="section-title">Mesas</div>
        <label style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '0.5rem' }}>
          <input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} />
          Mostrar mesas eliminadas
        </label>

        {[...tablesByZone.entries()].map(([zoneId, list]) => {
          const zoneName = zones?.find((z) => z.id === zoneId)?.name ?? `Zona #${zoneId}`
          return (
            <div key={zoneId} style={{ marginBottom: '0.5rem' }}>
              <strong className="muted">{zoneName}</strong>
              {list.map((table) => (
                <div className="manage-row" key={table.id}>
                  <div className="manage-info">
                    <span>
                      Mesa {table.number} {!table.active && <span className="status-badge">eliminada</span>}
                    </span>
                    <span className="muted">Capacidad {table.capacity}</span>
                  </div>
                  <div className="manage-actions">
                    {table.active ? (
                      <>
                        <button className="btn small" onClick={() => setTableForm({ table })}>
                          Editar
                        </button>
                        <button
                          className="btn small danger"
                          disabled={deleteTable.isPending || table.status === 'ocupada' || table.status === 'cuenta_solicitada'}
                          title={
                            table.status === 'ocupada' || table.status === 'cuenta_solicitada'
                              ? 'Libera la mesa antes de eliminarla'
                              : undefined
                          }
                          onClick={() => {
                            if (confirm(`¿Eliminar la mesa ${table.number}? Podrás reactivarla después.`)) {
                              deleteTable.mutate(table.id)
                            }
                          }}
                        >
                          Eliminar
                        </button>
                      </>
                    ) : (
                      <button className="btn small primary" disabled={reactivateTable.isPending} onClick={() => reactivateTable.mutate(table.id)}>
                        Reactivar
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )
        })}
        {deleteTable.isError && <div className="error-banner">{(deleteTable.error as Error).message}</div>}

        <button
          className="btn small"
          style={{ marginTop: '0.5rem' }}
          disabled={!zones || zones.length === 0}
          onClick={() => setTableForm({ table: null })}
        >
          + Nueva mesa
        </button>
        {(!zones || zones.length === 0) && <p className="muted">Crea una zona primero.</p>}
      </div>

      {zoneForm && (
        <ZoneFormModal
          zoneId={zoneForm.id || null}
          initialName={zoneForm.name}
          onClose={() => setZoneForm(null)}
          onSaved={invalidate}
        />
      )}
      {tableForm && zones && (
        <TableFormModal table={tableForm.table} zones={zones} onClose={() => setTableForm(null)} onSaved={invalidate} />
      )}
    </div>
  )
}

function ZoneFormModal({
  zoneId,
  initialName,
  onClose,
  onSaved,
}: {
  zoneId: number | null
  initialName: string
  onClose: () => void
  onSaved: () => void
}) {
  const [name, setName] = useState(initialName)

  const save = useMutation({
    mutationFn: () => (zoneId ? TablesApi.updateZone(zoneId, name) : TablesApi.createZone(name)),
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>{zoneId ? 'Renombrar zona' : 'Nueva zona'}</h3>
        <Field label="Nombre (ej. Interior, Terraza)">
          <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        </Field>
        {save.isError && <div className="error-banner">{(save.error as Error).message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={!name || save.isPending} onClick={() => save.mutate()}>
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

function TableFormModal({
  table,
  zones,
  onClose,
  onSaved,
}: {
  table: RestaurantTable | null
  zones: Zone[]
  onClose: () => void
  onSaved: () => void
}) {
  const [zoneId, setZoneId] = useState(table?.zone_id ?? zones[0]?.id ?? 0)
  const [number, setNumber] = useState(table?.number ?? '')
  const [capacity, setCapacity] = useState(table?.capacity ?? 4)

  const save = useMutation({
    mutationFn: () => {
      const input = { zone_id: zoneId, number, capacity }
      return table ? TablesApi.update(table.id, input) : TablesApi.create(input)
    },
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>{table ? `Editar mesa ${table.number}` : 'Nueva mesa'}</h3>
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
          <input value={number} onChange={(e) => setNumber(e.target.value)} autoFocus />
        </Field>
        <Field label="Capacidad">
          <input type="number" value={capacity} onChange={(e) => setCapacity(Number(e.target.value))} />
        </Field>
        {save.isError && <div className="error-banner">{(save.error as Error).message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={!number || save.isPending} onClick={() => save.mutate()}>
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
