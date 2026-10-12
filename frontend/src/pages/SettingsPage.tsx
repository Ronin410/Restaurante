import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { SettingsApi } from '../api/endpoints'
import { useBusiness } from '../settings/SettingsContext'
import { Field } from '../components/Field'

const ICON_SUGGESTIONS = ['🍽️', '🌮', '🍕', '🍔', '🍣', '🥩', '🍝', '🦐', '☕', '🥐', '🍺', '🍜']

/** Configuración general del negocio (solo administrador). */
export function SettingsPage() {
  const business = useBusiness()
  const queryClient = useQueryClient()
  const [name, setName] = useState(business.business_name)
  const [icon, setIcon] = useState(business.business_icon)

  const save = useMutation({
    mutationFn: () => SettingsApi.update({ business_name: name, business_icon: icon }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['settings'] }),
  })

  return (
    <div>
      <div className="page-header">
        <div>
          <h2>Configuración</h2>
          <p className="muted">Datos de tu negocio. Se muestran en el menú, en la pantalla de inicio de sesión y en la pestaña del navegador.</p>
        </div>
      </div>
      <div className="card" style={{ maxWidth: 520 }}>
        <Field label="Nombre del negocio">
          <input value={name} maxLength={60} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="Ícono (emoji)">
          <input value={icon} maxLength={8} onChange={(e) => setIcon(e.target.value)} />
        </Field>
        <div className="chip-row" style={{ marginBottom: '1rem' }}>
          {ICON_SUGGESTIONS.map((i) => (
            <button key={i} type="button" className={`chip ${icon === i ? 'selected' : ''}`} onClick={() => setIcon(i)}>
              {i}
            </button>
          ))}
        </div>
        {save.isError && <div className="error-banner">{(save.error as Error).message}</div>}
        {save.isSuccess && <p style={{ color: 'var(--color-accent)' }}>Guardado.</p>}
        <button className="btn primary" disabled={!name.trim() || save.isPending} onClick={() => save.mutate()}>
          Guardar
        </button>
      </div>
    </div>
  )
}
