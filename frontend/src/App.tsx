import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { RequireRole } from './auth/RequireRole'
import { LoginPage } from './pages/LoginPage'
import { TablesPage } from './pages/TablesPage'
import { TableDetailPage } from './pages/TableDetailPage'
import { AccountPage, NewAccountPage } from './pages/AccountPage'
import { KdsPage } from './pages/KdsPage'
import { CashCutsPage } from './pages/CashCutsPage'
import { DashboardPage } from './pages/DashboardPage'
import { EmployeesPage } from './pages/EmployeesPage'
import { CatalogPage } from './pages/CatalogPage'
import { AccountingPage } from './pages/AccountingPage'
import { AuditPage } from './pages/AuditPage'

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />

      <Route
        element={
          <RequireRole>
            <Layout />
          </RequireRole>
        }
      >
        <Route path="/" element={<Navigate to="/mesas" replace />} />

        <Route
          path="/mesas"
          element={
            <RequireRole roles={['admin', 'gerente', 'cajero', 'mesero']}>
              <TablesPage />
            </RequireRole>
          }
        />
        <Route
          path="/mesas/:id"
          element={
            <RequireRole roles={['admin', 'gerente', 'cajero', 'mesero']}>
              <TableDetailPage />
            </RequireRole>
          }
        />
        <Route
          path="/cuentas/nueva"
          element={
            <RequireRole roles={['admin', 'gerente', 'cajero']}>
              <NewAccountPage />
            </RequireRole>
          }
        />
        <Route
          path="/cuentas/:id"
          element={
            <RequireRole roles={['admin', 'gerente', 'cajero']}>
              <AccountPage />
            </RequireRole>
          }
        />
        <Route
          path="/kds"
          element={
            <RequireRole roles={['admin', 'gerente', 'cocina']}>
              <KdsPage />
            </RequireRole>
          }
        />
        <Route
          path="/cortes"
          element={
            <RequireRole roles={['admin', 'gerente', 'cajero', 'contador']}>
              <CashCutsPage />
            </RequireRole>
          }
        />
        <Route
          path="/dashboard"
          element={
            <RequireRole roles={['admin', 'gerente', 'contador']}>
              <DashboardPage />
            </RequireRole>
          }
        />
        <Route
          path="/personal"
          element={
            <RequireRole roles={['admin', 'gerente']}>
              <EmployeesPage />
            </RequireRole>
          }
        />
        <Route
          path="/catalogo"
          element={
            <RequireRole roles={['admin', 'gerente']}>
              <CatalogPage />
            </RequireRole>
          }
        />
        <Route
          path="/contaduria"
          element={
            <RequireRole roles={['admin', 'contador']}>
              <AccountingPage />
            </RequireRole>
          }
        />
        <Route
          path="/auditoria"
          element={
            <RequireRole roles={['admin', 'gerente']}>
              <AuditPage />
            </RequireRole>
          }
        />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
