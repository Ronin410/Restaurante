import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { EmployeesApi } from '../api/endpoints'
import type { Role, User } from '../api/types'
import { useAuth } from '../auth/AuthContext'
import { formatCurrency } from '../utils/money'
import { formatDateTime, formatDuration } from '../utils/time'
import { Field } from '../components/Field'

const ROLES: { value: Role; label: string }[] = [
  { value: 'admin', label: 'Administrador' },
  { value: 'gerente', label: 'Gerente' },
  { value: 'cajero', label: 'Cajero' },
  { value: 'mesero', label: 'Mesero' },
  { value: 'cocina', label: 'Cocina' },
  { value: 'contador', label: 'Contador' },
]
const roleLabel = (r: Role) => ROLES.find((x) => x.value === r)?.label ?? r

export function EmployeesPage() {
  const queryClient = useQueryClient()
  const { user: me } = useAuth()
  const { data: employees, isLoading } = useQuery({ queryKey: ['employees'], queryFn: () => EmployeesApi.list(false) })
  const [editing, setEditing] = useState<{ user: User | null } | null>(null)
  const [incidentFor, setIncidentFor] = useState<User | null>(null)
  const [historyFor, setHistoryFor] = useState<User | null>(null)
  const [showInactive, setShowInactive] = useState(false)
  const [search, setSearch] = useState('')

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['employees'] })
  const setActive = useMutation({
    mutationFn: ({ u, active }: { u: User; active: boolean }) =>
      EmployeesApi.update(u.id, { name: u.name, role: u.role, position: u.position, salary: u.salary, active }),
    onSuccess: refresh,
  })
  const remove = useMutation({ mutationFn: (id: number) => EmployeesApi.remove(id), onSuccess: refresh })

  const visible = (employees ?? [])
    .filter((e) => showInactive || e.active)
    .filter((e) => `${e.name} ${e.email} ${e.position}`.toLowerCase().includes(search.toLowerCase()))
  const errorMsg = (setActive.error ?? remove.error) as Error | null

  return (
    <div>
      <div className="page-header">
        <div>
          <h2>Personal</h2>
          <p className="muted">Altas, cambios de rol y sueldo, asistencia e incidencias.</p>
        </div>
        <button className="btn primary" onClick={() => setEditing({ user: null })}>
          + Nuevo empleado
        </button>
      </div>

      <div className="toolbar">
        <input className="search" placeholder="Buscar por nombre, correo o puesto…" value={search} onChange={(e) => setSearch(e.target.value)} />
        <label className="check">
          <input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} />
          Mostrar dados de baja
        </label>
      </div>

      {isLoading && <p>Cargando…</p>}
      {errorMsg && <div className="error-banner">{errorMsg.message}</div>}

      <div className="cards-grid">
        {visible.map((e) => (
          <div className={`card person-card ${e.active ? '' : 'inactive'}`} key={e.id}>
            <div className="person-head">
              <div className="avatar">{e.name.slice(0, 1).toUpperCase()}</div>
              <div style={{ minWidth: 0 }}>
                <strong>{e.name}</strong>
                <div className="muted ellipsis">{e.email}</div>
              </div>
              <span className={`status-badge role-${e.role}`}>{roleLabel(e.role)}</span>
            </div>
            <div className="person-meta">
              <span>{e.position || 'Sin puesto'}</span>
              <span>{formatCurrency(e.salary)}</span>
              <span className="muted">Ingreso {e.hire_date}</span>
              {!e.active && <span className="status-badge">Baja</span>}
            </div>
            <div className="manage-actions">
              <button className="btn small" onClick={() => setEditing({ user: e })}>
                Editar
              </button>
              <button className="btn small" onClick={() => setHistoryFor(e)}>
                Asistencia
              </button>
              <button className="btn small" onClick={() => setIncidentFor(e)}>
                Incidencia
              </button>
              {e.id !== me?.id &&
                (e.active ? (
                  <button
                    className="btn small"
                    onClick={() => confirm(`¿Dar de baja a ${e.name}? Ya no podrá iniciar sesión (su historial se conserva).`) && setActive.mutate({ u: e, active: false })}
                  >
                    Dar de baja
                  </button>
                ) : (
                  <button className="btn small primary" onClick={() => setActive.mutate({ u: e, active: true })}>
                    Reactivar
                  </button>
                ))}
              {e.id !== me?.id && (
                <button
                  className="btn small danger"
                  onClick={() =>
                    confirm(`¿Eliminar definitivamente a ${e.name}? Solo es posible si nunca ha vendido ni cobrado; si no, dalo de baja.`) &&
                    remove.mutate(e.id)
                  }
                >
                  Eliminar
                </button>
              )}
            </div>
          </div>
        ))}
      </div>

      {editing && <EmployeeFormModal user={editing.user} onClose={() => setEditing(null)} onSaved={refresh} />}
      {incidentFor && <IncidentModal user={incidentFor} onClose={() => setIncidentFor(null)} />}
      {historyFor && <AttendanceModal user={historyFor} onClose={() => setHistoryFor(null)} />}
    </div>
  )
}

function EmployeeFormModal({ user, onClose, onSaved }: { user: User | null; onClose: () => void; onSaved: () => void }) {
  const [f, setF] = useState({
    name: user?.name ?? '',
    email: user?.email ?? '',
    password: '',
    role: (user?.role ?? 'mesero') as Role,
    position: user?.position ?? '',
    salary: String(user?.salary ?? 0),
    hire_date: user?.hire_date ?? '',
    active: user?.active ?? true,
  })
  const set = (patch: Partial<typeof f>) => setF({ ...f, ...patch })

  const save = useMutation({
    mutationFn: () =>
      user
        ? EmployeesApi.update(user.id, {
            name: f.name,
            email: f.email,
            password: f.password || undefined,
            role: f.role,
            position: f.position,
            salary: Number(f.salary),
            hire_date: f.hire_date || undefined,
            active: f.active,
          })
        : EmployeesApi.create({
            name: f.name,
            email: f.email,
            password: f.password,
            role: f.role,
            position: f.position,
            salary: Number(f.salary),
            hire_date: f.hire_date || undefined,
          }),
    onSuccess: () => {
      onSaved()
      onClose()
    },
  })
  const passwordOk = user ? f.password === '' || f.password.length >= 6 : f.password.length >= 6

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>{user ? `Editar a ${user.name}` : 'Nuevo empleado'}</h3>
        <Field label="Nombre">
          <input value={f.name} onChange={(e) => set({ name: e.target.value })} />
        </Field>
        <Field label="Correo (usuario para entrar)">
          <input type="email" value={f.email} onChange={(e) => set({ email: e.target.value })} />
        </Field>
        <Field label={user ? 'Nueva contraseña (déjala vacía para no cambiarla)' : 'Contraseña (mín. 6 caracteres)'}>
          <input type="password" value={f.password} onChange={(e) => set({ password: e.target.value })} />
        </Field>
        <div className="form-row">
          <Field label="Rol (permisos)">
            <select value={f.role} onChange={(e) => set({ role: e.target.value as Role })}>
              {ROLES.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Puesto">
            <input value={f.position} onChange={(e) => set({ position: e.target.value })} />
          </Field>
        </div>
        <div className="form-row">
          <Field label="Sueldo">
            <input type="number" value={f.salary} onChange={(e) => set({ salary: e.target.value })} />
          </Field>
          <Field label="Fecha de ingreso">
            <input type="date" value={f.hire_date} onChange={(e) => set({ hire_date: e.target.value })} />
          </Field>
        </div>
        {user && (
          <label className="check" style={{ marginBottom: '0.75rem' }}>
            <input type="checkbox" checked={f.active} onChange={(e) => set({ active: e.target.checked })} />
            Activo (puede iniciar sesión)
          </label>
        )}
        {save.isError && <div className="error-banner">{(save.error as Error).message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={!f.name || !f.email || !passwordOk || save.isPending} onClick={() => save.mutate()}>
            {user ? 'Guardar cambios' : 'Crear'}
          </button>
          <button className="btn" onClick={onClose}>
            Cancelar
          </button>
        </div>
      </div>
    </div>
  )
}

function AttendanceModal({ user, onClose }: { user: User; onClose: () => void }) {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['attendance', user.id], queryFn: () => EmployeesApi.attendance(user.id) })
  const open = (data ?? []).find((r) => !r.clock_out)
  const refresh = () => queryClient.invalidateQueries({ queryKey: ['attendance', user.id] })
  const clockIn = useMutation({ mutationFn: () => EmployeesApi.clockIn(user.id), onSuccess: refresh })
  const clockOut = useMutation({ mutationFn: () => EmployeesApi.clockOut(user.id), onSuccess: refresh })
  const err = (clockIn.error ?? clockOut.error) as Error | null

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>Asistencia · {user.name}</h3>
        <p className="muted">Cada empleado puede marcar lo suyo en “Mi checador”; aquí puedes marcarlo por él si lo olvidó.</p>
        <div className="toolbar">
          {open ? (
            <button className="btn" disabled={clockOut.isPending} onClick={() => clockOut.mutate()}>
              Marcar salida
            </button>
          ) : (
            <button className="btn primary" disabled={clockIn.isPending} onClick={() => clockIn.mutate()}>
              Marcar entrada
            </button>
          )}
        </div>
        {err && <div className="error-banner">{err.message}</div>}
        <table className="data-table">
          <thead>
            <tr>
              <th>Entrada</th>
              <th>Salida</th>
              <th>Duración</th>
            </tr>
          </thead>
          <tbody>
            {(data ?? []).slice(0, 30).map((r) => (
              <tr key={r.id}>
                <td>{formatDateTime(r.clock_in)}</td>
                <td>{r.clock_out ? formatDateTime(r.clock_out) : <span className="status-badge">en turno</span>}</td>
                <td>{formatDuration(r.clock_in, r.clock_out)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <button className="btn" style={{ marginTop: '0.75rem' }} onClick={onClose}>
          Cerrar
        </button>
      </div>
    </div>
  )
}

function IncidentModal({ user, onClose }: { user: User; onClose: () => void }) {
  const [type, setType] = useState<'falta' | 'retardo' | 'permiso'>('retardo')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [note, setNote] = useState('')
  const { data: incidents, refetch } = useQuery({ queryKey: ['incidents', user.id], queryFn: () => EmployeesApi.incidents(user.id) })

  const mutation = useMutation({
    mutationFn: () => EmployeesApi.createIncident(user.id, { type, date, note }),
    onSuccess: () => {
      setNote('')
      refetch()
    },
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3 style={{ marginTop: 0 }}>Incidencias · {user.name}</h3>
        <div className="form-row">
          <Field label="Tipo">
            <select value={type} onChange={(e) => setType(e.target.value as typeof type)}>
              <option value="falta">Falta</option>
              <option value="retardo">Retardo</option>
              <option value="permiso">Permiso</option>
            </select>
          </Field>
          <Field label="Fecha">
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
        </div>
        <Field label="Nota">
          <textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} />
        </Field>
        {mutation.isError && <div className="error-banner">{(mutation.error as Error).message}</div>}
        <div className="toolbar">
          <button className="btn primary" disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            Registrar
          </button>
          <button className="btn" onClick={onClose}>
            Cerrar
          </button>
        </div>
        <div className="section-title">Historial</div>
        {(incidents ?? []).length === 0 && <p className="muted">Sin incidencias.</p>}
        {(incidents ?? []).map((i) => (
          <div className="manage-row" key={i.id}>
            <div className="manage-info">
              <span>
                <span className="status-badge">{i.type}</span> {i.date}
              </span>
              {i.note && <span className="muted">{i.note}</span>}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
