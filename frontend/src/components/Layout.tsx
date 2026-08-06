import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { OfflineBanner } from './OfflineBanner'
import type { Role } from '../api/types'

interface NavItem {
  to: string
  label: string
  roles?: Role[]
}

const NAV_ITEMS: NavItem[] = [
  { to: '/mesas', label: 'Mesas', roles: ['admin', 'gerente', 'cajero', 'mesero'] },
  { to: '/kds', label: 'Cocina (KDS)', roles: ['admin', 'gerente', 'cocina'] },
  { to: '/cortes', label: 'Cortes de caja', roles: ['admin', 'gerente', 'cajero', 'contador'] },
  { to: '/dashboard', label: 'Panel / KPIs', roles: ['admin', 'gerente', 'contador'] },
  { to: '/personal', label: 'Personal', roles: ['admin', 'gerente'] },
  { to: '/catalogo', label: 'Catálogo', roles: ['admin', 'gerente'] },
  { to: '/contaduria', label: 'Contaduría', roles: ['admin', 'contador'] },
  { to: '/auditoria', label: 'Auditoría', roles: ['admin', 'gerente'] },
]

export function Layout() {
  const { user, logout } = useAuth()

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>🌮 Taquería</h1>
        <nav>
          {NAV_ITEMS.filter((item) => !item.roles || (user && item.roles.includes(user.role))).map((item) => (
            <NavLink key={item.to} to={item.to} className={({ isActive }) => (isActive ? 'active' : '')}>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="user-box">
          <div>
            <strong>{user?.name}</strong>
          </div>
          <div className="muted">{user?.role}</div>
          <button className="btn" style={{ marginTop: '0.5rem', width: '100%' }} onClick={logout}>
            Cerrar sesión
          </button>
        </div>
      </aside>
      <main className="main-content">
        <OfflineBanner />
        <Outlet />
      </main>
    </div>
  )
}
