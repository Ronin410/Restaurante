import { describe, expect, it } from 'vitest'
import { cashDifference, formatCurrency, formatOccupiedMinutes, splitEvenly } from './money'

describe('formatCurrency', () => {
  it('formatea montos como pesos mexicanos', () => {
    expect(formatCurrency(93.6)).toContain('93.60')
    expect(formatCurrency(0)).toContain('0.00')
  })

  it('trata NaN como cero en vez de romperse', () => {
    expect(formatCurrency(NaN)).toContain('0.00')
  })
})

describe('splitEvenly', () => {
  it('reparte un total exacto entre partes iguales', () => {
    expect(splitEvenly(100, 4)).toEqual([25, 25, 25, 25])
  })

  it('ajusta los centavos restantes en las primeras partes para que la suma cuadre', () => {
    const shares = splitEvenly(100, 3)
    expect(shares).toEqual([33.34, 33.33, 33.33])
    const sum = shares.reduce((a, b) => a + b, 0)
    expect(Math.round(sum * 100) / 100).toBe(100)
  })

  it('regresa arreglo vacío si count es cero o negativo', () => {
    expect(splitEvenly(100, 0)).toEqual([])
    expect(splitEvenly(100, -2)).toEqual([])
  })
})

describe('cashDifference', () => {
  it('es positiva cuando sobra efectivo', () => {
    expect(cashDifference(500, 520)).toBe(20)
  })

  it('es negativa cuando falta efectivo', () => {
    expect(cashDifference(500, 480)).toBe(-20)
  })

  it('es cero cuando cuadra exactamente', () => {
    expect(cashDifference(550.4, 550.4)).toBe(0)
  })
})

describe('formatOccupiedMinutes', () => {
  it('regresa un guion cuando no hay dato', () => {
    expect(formatOccupiedMinutes(null)).toBe('—')
    expect(formatOccupiedMinutes(undefined)).toBe('—')
  })

  it('formatea minutos simples', () => {
    expect(formatOccupiedMinutes(45)).toBe('45 min')
  })

  it('formatea horas y minutos', () => {
    expect(formatOccupiedMinutes(125)).toBe('2h 5min')
  })
})
