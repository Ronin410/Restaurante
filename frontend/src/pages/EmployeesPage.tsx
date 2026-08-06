import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { EmployeesApi } from '../api/endpoints'
import type { Role } from '../api/types'
import { formatCurrency } from '../utils/money'
import { Field } from '../components/Field'

const ROLES: Role[] = ['admin', 'gerente', 'cajero', 'mesero', 'cocina', 'contador']

export function EmployeesPage() {
  const queryClient = useQueryClient()
  const { data: employees, isLoading } = useQuery({ queryKey: ['employees'], queryFn: () => EmployeesApi.list(false) })
  const [showNew, setShowNew] = useState(false)
  const [incidentFor, setIncidentFor] = useState<number | null>(null)

  const clockIn = useMutation({
    mutationFn: (id: number) => EmployeesApi.clockIn(id),
  })
  const clockOut = useMutation({
    mutationFn: (id: number) => EmployeesApi.clockOut(id),
  })

  return (
    <div>
      <div className="toolbar">
        <h2 style={{ margin: 0 }}>Personal</h2>
        <button className="btn primary" onClick={() => setShowNew(true)}>
          + Nuevo empleado
        </button>
      </div>

      {isLoading && <p>Cargando…</p>}

      <table className="data-table">
        <thead>
          <tr>
            <th>Nombre</th>
            <th>Correo</th>
            <th>Rol</th>
            <th>Puesto</th>
            <th>Sueldo</th>
            <th>Ingreso</th>
            <th>Activo</th>
            <th>Checador</th>
            <th>Incidencias</th>
          </tr>
        </thead>
        <tbody>
          {(employees ?? []).map((e) => (
            <tr key={e.id}>
              <td>{e.name}</td>
              <td>{e.email}</td>
              <td>{e.role}</td>
              <td>{e.position}</td>
              <td>{formatCurrency(e.salary)}</td>
              <td>{e.hire_date}</td>
              <td>{e.active ? 'Sí' : 'No'}</td>
              <td>
                <button className="btn" onClick={() => clockIn.mutate(e.id)} style={{ marginRight: 4 }}>
                  Entrada
                </button>
                <button className="btn" onClick={() => clockOut.mutate(e.id)}>
                  Salida
                </button>
              </td>
              <td>
                <button className="btn" onClick={() => setIncidentFor(e.id)}>
                  Registrar
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {showNew && (
        <NewEmployeeModal
          onClose={() => setShowNew(false)}
          onCreated={() => queryClient.invalidateQueries({ queryKey: ['employees'] })}
        />
      )}
      {incidentFor !== null && <IncidentModal userId={incidentFor} onClose={() => setIncidentFor(null)} />}
    </div>
  )
}

function NewEmployeeModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('mesero')
  const [position, setPosition] = useState('')
  const [salary, setSalary] = useState('0')

  const createMutation = useMutation({
    mutationFn: () =>
      EmployeesApi.create({ name, email, password, role, position, salary: Number(salary) }),
    onSuccess: () => {
      onCreated()
      onClose()
    },
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>Nuevo empleado</h3>
        <Field label="Nombre">
          <input value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="Correo">
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </Field>
        <Field label="Contraseña">
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </Field>
        <Field label="Rol">
          <select value={role} onChange={(e) => setRole(e.target.value as Role)}>
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Puesto">
          <input value={position} onChange={(e) => setPosition(e.target.value)} />
        </Field>
        <Field label="Sueldo">
          <input type="number" value={salary} onChange={(e) => setSalary(e.target.value)} />
        </Field>
        {createMutation.isError && <div className="error-banner">{(createMutation.error as Error).message}</div>}
        <div className="toolbar">
          <button
            className="btn primary"
            disabled={!name || !email || password.length < 6}
            onClick={() => createMutation.mutate()}
          >
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

function IncidentModal({ userId, onClose }: { userId: number; onClose: () => void }) {
  const [type, setType] = useState<'falta' | 'retardo' | 'permiso'>('retardo')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [note, setNote] = useState('')

  const mutation = useMutation({
    mutationFn: () => EmployeesApi.createIncident(userId, { type, date, note }),
    onSuccess: onClose,
  })

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>Registrar incidencia</h3>
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
        <Field label="Nota">
          <textarea value={note} onChange={(e) => setNote(e.target.value)} rows={3} />
        </Field>
        {mutation.isError && <div className="error-banner">{(mutation.error as Error).message}</div>}
        <div className="toolbar">
          <button className="btn primary" onClick={() => mutation.mutate()}>
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
