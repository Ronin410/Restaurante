const formatter = new Intl.NumberFormat('es-MX', { style: 'currency', currency: 'MXN' })

export function formatCurrency(amount: number): string {
  if (Number.isNaN(amount)) return formatter.format(0)
  return formatter.format(amount)
}

/**
 * Calcula las partes de una cuenta dividida "por persona": reparte el total
 * en `count` partes iguales, ajustando la última para que la suma cuadre
 * exactamente con el total (evita perder centavos por redondeo).
 */
export function splitEvenly(total: number, count: number): number[] {
  if (count <= 0) return []
  const rounded = Math.round(total * 100)
  const base = Math.floor(rounded / count)
  const remainder = rounded - base * count
  const shares: number[] = []
  for (let i = 0; i < count; i++) {
    const cents = base + (i < remainder ? 1 : 0)
    shares.push(cents / 100)
  }
  return shares
}

/** Diferencia de un corte de caja: positivo = sobrante, negativo = faltante. */
export function cashDifference(expected: number, counted: number): number {
  return Math.round((counted - expected) * 100) / 100
}

export function formatOccupiedMinutes(minutes: number | null | undefined): string {
  if (minutes === null || minutes === undefined) return '—'
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return `${hours}h ${rest}min`
}
