import type { Credentials, Matrix, MatrixResult } from '../types'

const baseUrl = (import.meta.env.VITE_MATRIX_API_URL ?? '').replace(/\/$/, '')
const configuredTimeout = Number(import.meta.env.VITE_API_TIMEOUT_MS)
const requestTimeoutMs = Number.isFinite(configuredTimeout) && configuredTimeout > 0 ? configuredTimeout : 15_000
let accessToken: string | null = null

export interface LoginResponseDto {
  accessToken: string
  tokenType: 'Bearer'
  expiresIn: number
}

interface StatisticsDto {
  minimum: number
  maximum: number
  sum: number
  average: number
  elements: number
}

interface MatrixStatisticsDto extends StatisticsDto {
  name: string
  diagonal: boolean
}

export interface ProcessResponseDto {
  requestId: string
  rotation: {
    direction: 'clockwise'
    matrix: Matrix
  }
  qr: {
    q: Matrix
    r: Matrix
  }
  statistics: {
    global: StatisticsDto
    matrices: MatrixStatisticsDto[]
    anyDiagonal: boolean
  }
}

interface ProblemDetails {
  detail?: string
}

type UnknownRecord = Record<string, unknown>
type Validator<T> = (value: unknown) => value is T
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

export class ApiError extends Error {
  constructor(message: string, public readonly status: number) {
    super(message)
    this.name = 'ApiError'
  }
}

export class RequestCancelledError extends Error {
  constructor() {
    super('Se canceló la solicitud.')
    this.name = 'RequestCancelledError'
  }
}

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function hasExactKeys(value: UnknownRecord, keys: readonly string[]): boolean {
  const actual = Object.keys(value).sort()
  const expected = [...keys].sort()
  return actual.length === expected.length && actual.every((key, index) => key === expected[index])
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function isMatrix(value: unknown): value is Matrix {
  if (!Array.isArray(value) || value.length === 0 || !Array.isArray(value[0]) || value[0].length === 0) return false
  const columns = value[0].length
  return value.every((row) => Array.isArray(row) && row.length === columns && row.every(isFiniteNumber))
}

function isStatistics(value: unknown): value is StatisticsDto {
  return isRecord(value)
    && hasExactKeys(value, ['minimum', 'maximum', 'sum', 'average', 'elements'])
    && isFiniteNumber(value.minimum)
    && isFiniteNumber(value.maximum)
    && isFiniteNumber(value.sum)
    && isFiniteNumber(value.average)
    && Number.isInteger(value.elements)
    && (value.elements as number) >= 1
}

function isMatrixStatistics(value: unknown): value is MatrixStatisticsDto {
  if (!isRecord(value) || !hasExactKeys(value, ['name', 'minimum', 'maximum', 'sum', 'average', 'elements', 'diagonal'])) return false
  const { name, diagonal, ...statistics } = value
  return typeof name === 'string' && name.length > 0 && typeof diagonal === 'boolean' && isStatistics(statistics)
}

function isLoginResponse(value: unknown): value is LoginResponseDto {
  return isRecord(value)
    && hasExactKeys(value, ['accessToken', 'tokenType', 'expiresIn'])
    && typeof value.accessToken === 'string'
    && value.accessToken.length > 0
    && value.tokenType === 'Bearer'
    && Number.isFinite(value.expiresIn)
    && Number.isInteger(value.expiresIn)
    && (value.expiresIn as number) > 0
}

function isProcessResponse(value: unknown): value is ProcessResponseDto {
  if (!isRecord(value) || !hasExactKeys(value, ['requestId', 'rotation', 'qr', 'statistics'])) return false
  if (typeof value.requestId !== 'string' || !uuidPattern.test(value.requestId)) return false
  if (!isRecord(value.rotation) || !hasExactKeys(value.rotation, ['direction', 'matrix'])) return false
  if (value.rotation.direction !== 'clockwise' || !isMatrix(value.rotation.matrix)) return false
  if (!isRecord(value.qr) || !hasExactKeys(value.qr, ['q', 'r']) || !isMatrix(value.qr.q) || !isMatrix(value.qr.r)) return false
  if (!isRecord(value.statistics) || !hasExactKeys(value.statistics, ['global', 'matrices', 'anyDiagonal'])) return false
  return isStatistics(value.statistics.global)
    && Array.isArray(value.statistics.matrices)
    && value.statistics.matrices.length > 0
    && value.statistics.matrices.every(isMatrixStatistics)
    && typeof value.statistics.anyDiagonal === 'boolean'
}

async function request<T>(path: string, init: RequestInit, validator: Validator<T>, responseName: string): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Content-Type', 'application/json')
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)

  const controller = new AbortController()
  let timedOut = false
  const abortFromCaller = () => controller.abort(init.signal?.reason)
  if (init.signal?.aborted) abortFromCaller()
  else init.signal?.addEventListener('abort', abortFromCaller, { once: true })
  const timeout = globalThis.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, requestTimeoutMs)

  try {
    const response = await fetch(`${baseUrl}${path}`, { ...init, headers, signal: controller.signal })
    const body: unknown = await response.json().catch(() => undefined)
    if (!response.ok) {
      const problem = isRecord(body) ? body as ProblemDetails : {}
      const message = typeof problem.detail === 'string' ? problem.detail : `La solicitud falló (${response.status}).`
      throw new ApiError(message, response.status)
    }
    if (!validator(body)) throw new ApiError(`La API devolvió una respuesta de ${responseName} no válida.`, 502)
    return body
  } catch (cause) {
    if (cause instanceof ApiError) throw cause
    if (timedOut) throw new ApiError(`La solicitud a la API superó el tiempo límite de ${requestTimeoutMs} ms.`, 0)
    if (controller.signal.aborted) throw new RequestCancelledError()
    throw new ApiError('No se puede acceder a la API. Verificá el servicio e intentá de nuevo.', 0)
  } finally {
    globalThis.clearTimeout(timeout)
    init.signal?.removeEventListener('abort', abortFromCaller)
  }
}

function mapProcessResponse(response: ProcessResponseDto): MatrixResult {
  return {
    requestId: response.requestId,
    rotationDirection: response.rotation.direction,
    rotated: response.rotation.matrix,
    q: response.qr.q,
    r: response.qr.r,
    globalStatistics: response.statistics.global,
    matrixStatistics: response.statistics.matrices,
    anyDiagonal: response.statistics.anyDiagonal,
  }
}

export const api = {
  async login(credentials: Credentials, signal?: AbortSignal): Promise<LoginResponseDto> {
    const result = await request('/api/v1/auth/token', {
      method: 'POST',
      body: JSON.stringify(credentials),
      signal,
    }, isLoginResponse, 'inicio de sesión')
    if (signal?.aborted) throw new RequestCancelledError()
    accessToken = result.accessToken
    return result
  },
  logout(): void {
    accessToken = null
  },
  async processMatrix(matrix: Matrix, signal?: AbortSignal): Promise<MatrixResult> {
    const result = await request('/api/v1/matrices/process', {
      method: 'POST',
      body: JSON.stringify({ matrix }),
      signal,
    }, isProcessResponse, 'procesamiento de matriz')
    return mapProcessResponse(result)
  },
}
