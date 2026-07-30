import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import type { MatrixResult } from '../types'
import { useAuth } from './useAuth'
import { useMatrixProcessor } from './useMatrixProcessor'

const matrixResult: MatrixResult = {
  requestId: 'late-request',
  rotationDirection: 'clockwise',
  rotated: [[1]],
  q: [[1]],
  r: [[1]],
  globalStatistics: { minimum: 1, maximum: 1, sum: 3, average: 1, elements: 3 },
  matrixStatistics: [],
  anyDiagonal: true,
}

const mountedApps: ReturnType<typeof createApp>[] = []

afterEach(() => {
  mountedApps.splice(0).forEach((app) => app.unmount())
  document.body.replaceChildren()
  vi.useRealTimers()
  vi.restoreAllMocks()
  api.logout()
})

describe('ciclo de vida de la sesión', () => {
  it('aborta el procesamiento e ignora una respuesta resuelta después del reinicio', async () => {
    let processor!: ReturnType<typeof useMatrixProcessor>
    let resolveRequest!: (result: MatrixResult) => void
    let requestSignal: AbortSignal | undefined
    vi.spyOn(api, 'processMatrix').mockImplementation((_matrix, signal) => {
      requestSignal = signal
      return new Promise((resolve) => { resolveRequest = resolve })
    })
    const app = createApp(defineComponent({
      setup() {
        processor = useMatrixProcessor()
        return () => h('div')
      },
    }))
    app.mount(document.createElement('div'))
    mountedApps.push(app)

    const pending = processor.process([[1]])
    processor.reset()
    expect(requestSignal?.aborted).toBe(true)
    resolveRequest(matrixResult)
    await pending

    expect(processor.result.value).toBeNull()
    expect(processor.loading.value).toBe(false)
  })

  it('vence la autenticación según expiresIn', async () => {
    vi.useFakeTimers()
    vi.spyOn(api, 'login').mockResolvedValue({ accessToken: 'token', tokenType: 'Bearer', expiresIn: 2 })
    let auth!: ReturnType<typeof useAuth>
    const app = createApp(defineComponent({
      setup() {
        auth = useAuth()
        return () => h('div')
      },
    }))
    app.mount(document.createElement('div'))
    mountedApps.push(app)

    await auth.login({ username: 'analyst', password: 'secret' })
    expect(auth.authenticated.value).toBe(true)
    expect(auth.expiresIn.value).toBe(2)

    vi.advanceTimersByTime(2_000)
    await nextTick()
    expect(auth.authenticated.value).toBe(false)
    expect(auth.expiresIn.value).toBeNull()
  })
})
