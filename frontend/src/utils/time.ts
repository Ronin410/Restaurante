/** Las fechas del backend vienen como "AAAA-MM-DD HH:MM:SS" (UTC, sin zona). */
export function parseServerDate(value: string): Date {
  return new Date(value.replace(' ', 'T') + (value.includes('Z') || value.includes('+') ? '' : 'Z'))
}

export function formatDateTime(value: string): string {
  return parseServerDate(value).toLocaleString('es-MX', { dateStyle: 'medium', timeStyle: 'short' })
}

export function formatDuration(from: string, to: string | null): string {
  const end = to ? parseServerDate(to).getTime() : Date.now()
  const mins = Math.max(0, Math.round((end - parseServerDate(from).getTime()) / 60000))
  const h = Math.floor(mins / 60)
  return h > 0 ? `${h} h ${mins % 60} min` : `${mins} min`
}
