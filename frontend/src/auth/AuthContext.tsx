import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { AuthApi } from '../api/endpoints'
import { loadStoredToken, setAuthToken } from '../api/client'
import type { Role, User } from '../api/types'

interface AuthContextValue {
  user: User | null
  loading: boolean
  error: string | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  hasRole: (...roles: Role[]) => boolean
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = loadStoredToken()
    if (!token) {
      setLoading(false)
      return
    }
    AuthApi.me()
      .then(setUser)
      .catch(() => setAuthToken(null))
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    setError(null)
    try {
      const result = await AuthApi.login(email, password)
      setAuthToken(result.token)
      setUser(result.user)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'No se pudo iniciar sesión')
      throw err
    }
  }, [])

  const logout = useCallback(() => {
    setAuthToken(null)
    setUser(null)
  }, [])

  const hasRole = useCallback((...roles: Role[]) => !!user && roles.includes(user.role), [user])

  return (
    <AuthContext.Provider value={{ user, loading, error, login, logout, hasRole }}>{children}</AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth debe usarse dentro de <AuthProvider>')
  return ctx
}
