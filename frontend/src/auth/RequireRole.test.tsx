import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { Route, Routes } from 'react-router-dom'
import { renderWithProviders } from '../test/renderWithProviders'
import { RequireRole } from './RequireRole'

vi.mock('../api/endpoints', () => ({
  AuthApi: { login: vi.fn(), me: vi.fn() },
}))

beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
})

function renderProtected(route: string) {
  return renderWithProviders(
    <Routes>
      <Route path="/login" element={<div>Página de login</div>} />
      <Route
        path="/secreta"
        element={
          <RequireRole roles={['admin']}>
            <div>Contenido protegido</div>
          </RequireRole>
        }
      />
    </Routes>,
    { route },
  )
}

describe('RequireRole', () => {
  it('redirige a /login cuando no hay sesión', async () => {
    renderProtected('/secreta')
    expect(await screen.findByText('Página de login')).toBeInTheDocument()
  })
})
