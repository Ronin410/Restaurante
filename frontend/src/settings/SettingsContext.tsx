import { createContext, useContext, useEffect, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { SettingsApi } from '../api/endpoints'
import type { BusinessSettings } from '../api/types'

const DEFAULTS: BusinessSettings = { business_name: 'Mi Restaurante', business_icon: '🍽️' }

const SettingsContext = createContext<BusinessSettings>(DEFAULTS)

/** Nombre e ícono del negocio (configurables); también ponen el título de la pestaña. */
export function SettingsProvider({ children }: { children: ReactNode }) {
  const { data } = useQuery({ queryKey: ['settings'], queryFn: SettingsApi.get, staleTime: 5 * 60_000 })
  const value = data ?? DEFAULTS

  useEffect(() => {
    document.title = value.business_name
  }, [value.business_name])

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

export function useBusiness(): BusinessSettings {
  return useContext(SettingsContext)
}
