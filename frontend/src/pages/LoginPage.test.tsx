import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders } from '../test/renderWithProviders'
import { LoginPage } from './LoginPage'
import { AuthApi } from '../api/endpoints'

vi.mock('../api/endpoints', () => ({
  AuthApi: {
    login: vi.fn(),
    me: vi.fn(),
  },
}))

beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
})

describe('LoginPage', () => {
  it('envía las credenciales capturadas al iniciar sesión', async () => {
    vi.mocked(AuthApi.login).mockResolvedValue({
      token: 'token-123',
      expires_at: new Date().toISOString(),
      user: { id: 1, name: 'Ada Admin', email: 'admin@test.mx', role: 'admin', position: '', salary: 0, hire_date: '', active: true },
    })

    renderWithProviders(<LoginPage />, { route: '/login' })

    await userEvent.type(screen.getByLabelText('Correo'), 'admin@test.mx')
    await userEvent.type(screen.getByLabelText('Contraseña'), 'password123')
    await userEvent.click(screen.getByRole('button', { name: /entrar/i }))

    await waitFor(() => {
      expect(AuthApi.login).toHaveBeenCalledWith('admin@test.mx', 'password123')
    })
  })

  it('muestra un mensaje de error si las credenciales son inválidas', async () => {
    vi.mocked(AuthApi.login).mockRejectedValue(new Error('credenciales inválidas'))

    renderWithProviders(<LoginPage />, { route: '/login' })

    await userEvent.type(screen.getByLabelText('Correo'), 'admin@test.mx')
    await userEvent.type(screen.getByLabelText('Contraseña'), 'incorrecta')
    await userEvent.click(screen.getByRole('button', { name: /entrar/i }))

    expect(await screen.findByText('credenciales inválidas')).toBeInTheDocument()
  })

  it('no permite enviar el formulario con campos vacíos (HTML required)', () => {
    renderWithProviders(<LoginPage />, { route: '/login' })
    const emailInput = screen.getByLabelText('Correo') as HTMLInputElement
    expect(emailInput.required).toBe(true)
    const passwordInput = screen.getByLabelText('Contraseña') as HTMLInputElement
    expect(passwordInput.required).toBe(true)
  })
})
