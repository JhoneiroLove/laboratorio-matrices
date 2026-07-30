import type { Matrix } from '../types'

export interface MatrixParseResult {
  matrix: Matrix | null
  error: string | null
}

export function parseMatrixInput(input: string): MatrixParseResult {
  const lines = input.trim().split(/\r?\n/).filter((line) => line.trim())
  if (lines.length === 0) return { matrix: null, error: 'Ingresá al menos una fila.' }

  const matrix: number[][] = []
  let width = 0

  for (let rowIndex = 0; rowIndex < lines.length; rowIndex += 1) {
    const cells = lines[rowIndex].trim().split(/[\s,;]+/).filter(Boolean)
    if (rowIndex === 0) width = cells.length
    if (cells.length !== width) {
      return { matrix: null, error: `La fila ${rowIndex + 1} tiene ${cells.length} valores; se esperaban ${width}.` }
    }

    const row = cells.map(Number)
    const invalidColumn = row.findIndex((value) => !Number.isFinite(value))
    if (invalidColumn >= 0) {
      return { matrix: null, error: `El valor de la fila ${rowIndex + 1}, columna ${invalidColumn + 1} no es un número válido.` }
    }
    matrix.push(row)
  }

  return { matrix, error: null }
}

export function formatNumber(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(value)
}
