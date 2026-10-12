import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
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
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()

  // Cierra el menú automáticamente al navegar a otra sección (en tablet el
  // cajón flota sobre el contenido, así que si no se cierra solo tapa la
  // pantalla nueva).
  useEffect(() => {
    setMenuOpen(false)
  }, [location.pathname])

  const visibleItems = NAV_ITEMS.filter((item) => !item.roles || (user && item.roles.includes(user.role)))
  const currentTitle = visibleItems.find((item) => location.pathname.startsWith(item.to))?.label ?? 'Taquería'

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
        <h1>🌮 Taquería</h1>
        <nav>
          {visibleItems.map((item) => (
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
