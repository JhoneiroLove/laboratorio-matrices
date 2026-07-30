export type Matrix = readonly (readonly number[])[]

export interface Credentials {
  username: string
  password: string
}

export interface StatisticSet {
  minimum: number
  maximum: number
  average: number
  sum: number
  elements: number
}

export interface MatrixStatistic extends StatisticSet {
  name: string
  diagonal: boolean
}

export interface MatrixResult {
  requestId: string
  rotationDirection: 'clockwise'
  rotated: Matrix
  q: Matrix
  r: Matrix
  globalStatistics: StatisticSet
  matrixStatistics: readonly MatrixStatistic[]
  anyDiagonal: boolean
}
