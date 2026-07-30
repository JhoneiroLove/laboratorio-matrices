import { describe, expect, it } from 'vitest'
import { parseMatrixInput } from './matrix'

describe('parseMatrixInput', () => {
  it('interpreta matrices rectangulares con los separadores admitidos', () => {
    expect(parseMatrixInput('1, 2; 3\n4 5 6')).toEqual({ matrix: [[1, 2, 3], [4, 5, 6]], error: null })
  })

  it('acepta decimales, negativos y notación científica', () => {
    expect(parseMatrixInput('-1.5 2e2\n0 .25').matrix).toEqual([[-1.5, 200], [0, 0.25]])
  })

  it('rechaza filas con diferentes longitudes', () => {
    expect(parseMatrixInput('1 2\n3').error).toContain('La fila 2 tiene 1 valores; se esperaban 2')
  })

  it('rechaza valores no numéricos e indica sus coordenadas', () => {
    expect(parseMatrixInput('1 2\n3 nope').error).toContain('fila 2, columna 2')
  })
})
