import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from './AuthContext'
import type { Role } from '../api/types'

/** Protege una ruta: exige sesión iniciada y, opcionalmente, uno de los roles indicados. */
export function RequireRole({ roles, children }: { roles?: Role[]; children: ReactNode }) {
  const { user, loading } = useAuth()

  if (loading) return <div className="page-loading">Cargando…</div>
  if (!user) return <Navigate to="/login" replace />
  if (roles && roles.length > 0 && !roles.includes(user.role)) {
    return (
      <div className="empty-state">
        <h2>Acceso no autorizado</h2>
        <p>Tu rol ({user.role}) no tiene permiso para ver esta sección.</p>
      </div>
    )
  }
  return <>{children}</>
}
