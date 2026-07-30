import { onUnmounted, readonly, ref } from 'vue'
import { api, ApiError, RequestCancelledError } from '../services/api'
import type { Matrix, MatrixResult } from '../types'

export function useMatrixProcessor(onUnauthorized?: () => void) {
  const result = ref<MatrixResult | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let activeController: AbortController | null = null
  let requestVersion = 0

  function reset() {
    requestVersion += 1
    activeController?.abort()
    activeController = null
    result.value = null
    loading.value = false
    error.value = null
  }

  async function process(matrix: Matrix) {
    activeController?.abort()
    const controller = new AbortController()
    activeController = controller
    const version = ++requestVersion
    result.value = null
    loading.value = true
    error.value = null
    try {
      const nextResult = await api.processMatrix(matrix, controller.signal)
      if (version === requestVersion && !controller.signal.aborted) result.value = nextResult
    } catch (cause) {
      if (version !== requestVersion || cause instanceof RequestCancelledError) return
      if (cause instanceof ApiError && cause.status === 401) onUnauthorized?.()
      error.value = cause instanceof Error ? cause.message : 'No se pudo procesar la matriz.'
    } finally {
      if (version === requestVersion) {
        activeController = null
        loading.value = false
      }
    }
  }

  onUnmounted(reset)

  return { result: readonly(result), loading: readonly(loading), error: readonly(error), process, reset }
}
