import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError, type ProcessResponseDto } from './api'

const processResponse: ProcessResponseDto = {
  requestId: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
  rotation: { direction: 'clockwise', matrix: [[3, 1], [4, 2]] },
  qr: { q: [[1, 0], [0, 1]], r: [[1, 2], [0, 3]] },
  statistics: {
    global: { minimum: 0, maximum: 4, sum: 17, average: 1.4167, elements: 12 },
    matrices: [
      { name: 'rotated', minimum: 1, maximum: 4, sum: 10, average: 2.5, elements: 4, diagonal: false },
      { name: 'Q', minimum: 0, maximum: 1, sum: 2, average: 0.5, elements: 4, diagonal: true },
      { name: 'R', minimum: 0, maximum: 3, sum: 6, average: 1.5, elements: 4, diagonal: false },
    ],
    anyDiagonal: true,
  },
}

afterEach(() => {
  api.logout()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('api', () => {
  it('usa el token exacto del inicio de sesión y mapea la respuesta exacta del procesamiento', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({ accessToken: 'jwt-token', tokenType: 'Bearer', expiresIn: 900 }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(processResponse), { status: 200 }))

    await api.login({ username: 'analyst', password: 'secret' })
    const result = await api.processMatrix([[1, 2], [3, 4]])

    const loginInit = fetchMock.mock.calls[0][1] as RequestInit
    expect(loginInit.body).toBe(JSON.stringify({ username: 'analyst', password: 'secret' }))
    const processInit = fetchMock.mock.calls[1][1] as RequestInit
    expect((processInit.headers as Headers).get('Authorization')).toBe('Bearer jwt-token')
    expect(processInit.body).toBe(JSON.stringify({ matrix: [[1, 2], [3, 4]] }))
    expect(result).toEqual({
      requestId: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
      rotationDirection: 'clockwise',
      rotated: processResponse.rotation.matrix,
      q: processResponse.qr.q,
      r: processResponse.qr.r,
      globalStatistics: processResponse.statistics.global,
      matrixStatistics: processResponse.statistics.matrices,
      anyDiagonal: true,
    })
  })

  it('usa el campo detail de RFC 9457 para los errores de la API', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ type: 'about:blank', title: 'Matriz no válida', status: 422, detail: 'Todas las filas deben tener la misma longitud.' }), { status: 422 }),
    )

    await expect(api.processMatrix([[1, 2]])).rejects.toMatchObject({
      name: 'ApiError',
      status: 422,
      message: 'Todas las filas deben tener la misma longitud.',
    } satisfies Partial<ApiError>)
  })

  it('rechaza respuestas exitosas de inicio de sesión con formato inválido', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ accessToken: 'jwt-token', tokenType: 'Bearer' }), { status: 200 }),
    )

    await expect(api.login({ username: 'analyst', password: 'secret' })).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      message: 'La API devolvió una respuesta de inicio de sesión no válida.',
    } satisfies Partial<ApiError>)
  })

  it('rechaza un tokenType diferente de Bearer', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ accessToken: 'jwt-token', tokenType: 'bearer', expiresIn: 900 }), { status: 200 }),
    )

    await expect(api.login({ username: 'analyst', password: 'secret' })).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      message: 'La API devolvió una respuesta de inicio de sesión no válida.',
    } satisfies Partial<ApiError>)
  })

  it('rechaza respuestas exitosas de procesamiento con formato inválido', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ...processResponse, qr: { q: [[1]], r: 'invalid' } }), { status: 200 }),
    )

    await expect(api.processMatrix([[1]])).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      message: 'La API devolvió una respuesta de procesamiento de matriz no válida.',
    } satisfies Partial<ApiError>)
  })

  it('rechaza un requestId que no sea UUID', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ...processResponse, requestId: 'req-42' }), { status: 200 }),
    )

    await expect(api.processMatrix([[1]])).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      message: 'La API devolvió una respuesta de procesamiento de matriz no válida.',
    } satisfies Partial<ApiError>)
  })

  it('rechaza estadísticas con menos de un elemento', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({
        ...processResponse,
        statistics: {
          ...processResponse.statistics,
          global: { ...processResponse.statistics.global, elements: 0 },
        },
      }), { status: 200 }),
    )

    await expect(api.processMatrix([[1]])).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      message: 'La API devolvió una respuesta de procesamiento de matriz no válida.',
    } satisfies Partial<ApiError>)
  })

  it('rechaza una lista vacía de estadísticas por matriz', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({
        ...processResponse,
        statistics: { ...processResponse.statistics, matrices: [] },
      }), { status: 200 }),
    )

    await expect(api.processMatrix([[1]])).rejects.toMatchObject({
      name: 'ApiError',
      status: 502,
      message: 'La API devolvió una respuesta de procesamiento de matriz no válida.',
    } satisfies Partial<ApiError>)
  })

  it('aborta una solicitud de procesamiento cuando el cliente la cancela', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((_input, init) => new Promise((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
    }))
    const controller = new AbortController()
    const pending = api.processMatrix([[1]], controller.signal)

    controller.abort()

    await expect(pending).rejects.toMatchObject({ name: 'RequestCancelledError' })
  })

  it('agota el tiempo de las solicitudes que superan el límite predeterminado', async () => {
    vi.useFakeTimers()
    vi.spyOn(globalThis, 'fetch').mockImplementation((_input, init) => new Promise((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
    }))
    const pending = api.processMatrix([[1]])
    const assertion = expect(pending).rejects.toMatchObject({
      name: 'ApiError',
      status: 0,
      message: 'La solicitud a la API superó el tiempo límite de 15000 ms.',
    } satisfies Partial<ApiError>)

    await vi.advanceTimersByTimeAsync(15_000)
    await assertion
  })
})
