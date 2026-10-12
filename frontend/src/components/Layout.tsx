import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { OfflineBanner } from './OfflineBanner'
import { useBusiness } from '../settings/SettingsContext'
import type { Role } from '../api/types'

interface NavItem {
  to: string
  label: string
  icon: string
  group: 'Operación' | 'Administración'
  roles?: Role[]
}

const NAV_ITEMS: NavItem[] = [
  { to: '/mesas', label: 'Mesas', icon: '🍽️', group: 'Operación', roles: ['admin', 'gerente', 'cajero', 'mesero'] },
  { to: '/pedidos', label: 'Pedidos / llevar', icon: '🛍️', group: 'Operación', roles: ['admin', 'gerente', 'cajero', 'mesero'] },
  { to: '/kds', label: 'Cocina', icon: '👨‍🍳', group: 'Operación', roles: ['admin', 'gerente', 'cocina'] },
  { to: '/cortes', label: 'Cortes de caja', icon: '💵', group: 'Operación', roles: ['admin', 'gerente', 'cajero', 'contador'] },
  { to: '/dashboard', label: 'Panel / KPIs', icon: '📊', group: 'Administración', roles: ['admin', 'gerente', 'contador'] },
  { to: '/catalogo', label: 'Catálogo', icon: '📖', group: 'Administración', roles: ['admin', 'gerente'] },
  { to: '/personal', label: 'Personal', icon: '👥', group: 'Administración', roles: ['admin', 'gerente'] },
  { to: '/contaduria', label: 'Contaduría', icon: '🧾', group: 'Administración', roles: ['admin', 'contador'] },
  { to: '/auditoria', label: 'Auditoría', icon: '🔍', group: 'Administración', roles: ['admin', 'gerente'] },
  { to: '/configuracion', label: 'Configuración', icon: '⚙️', group: 'Administración', roles: ['admin'] },
]

const CLOCK_ITEM = { to: '/checador', label: 'Mi checador', icon: '⏱️' }

/**
 * El uso principal de esta app es en tablet, en el piso de venta: la barra
 * lateral fija de escritorio se vuelve aquí un cajón que se abre con un
 * botón de hamburguesa (barra superior) y se cierra tocando fuera o al
 * elegir una sección, para no robarle pantalla al mapa de mesas / la
 * comanda. En pantallas de escritorio (≥960px, ver index.css) el cajón
 * vuelve a quedar fijo y visible, como antes.
 */
export function Layout() {
  const { user, logout } = useAuth()
  const business = useBusiness()
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()

  // Cierra el menú automáticamente al navegar a otra sección (en tablet el
  // cajón flota sobre el contenido, así que si no se cierra solo tapa la
  // pantalla nueva).
  useEffect(() => {
    setMenuOpen(false)
  }, [location.pathname])

  const visibleItems = NAV_ITEMS.filter((item) => !item.roles || (user && item.roles.includes(user.role)))
  const currentTitle =
    [...visibleItems, CLOCK_ITEM].find((item) => location.pathname.startsWith(item.to))?.label ?? business.business_name

  return (
    <div className="app-shell">
      <div className="topbar">
        <button
          className="hamburger-btn"
          aria-label={menuOpen ? 'Cerrar menú' : 'Abrir menú'}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
        >
          {menuOpen ? '✕' : '☰'}
        </button>
        <span className="topbar-title">{currentTitle}</span>
      </div>

      <div className={`sidebar-backdrop ${menuOpen ? 'open' : ''}`} onClick={() => setMenuOpen(false)} />

      <aside className={`sidebar ${menuOpen ? 'open' : ''}`}>
        <h1>
          {business.business_icon} {business.business_name}
        </h1>
        <nav>
          {(['Operación', 'Administración'] as const).map((group) => {
            const items = visibleItems.filter((i) => i.group === group)
            if (items.length === 0) return null
            return (
              <div key={group} className="nav-group">
                <div className="nav-group-title">{group}</div>
                {items.map((item) => (
                  <NavLink key={item.to} to={item.to} className={({ isActive }) => (isActive ? 'active' : '')}>
                    <span className="nav-icon" aria-hidden>
                      {item.icon}
                    </span>
                    {item.label}
                  </NavLink>
                ))}
              </div>
            )
          })}
          <div className="nav-group">
            <NavLink to={CLOCK_ITEM.to} className={({ isActive }) => (isActive ? 'active' : '')}>
              <span className="nav-icon" aria-hidden>
                {CLOCK_ITEM.icon}
              </span>
              {CLOCK_ITEM.label}
            </NavLink>
          </div>
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
