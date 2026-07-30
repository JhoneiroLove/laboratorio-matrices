import { onUnmounted, readonly, ref } from 'vue'
import { api, RequestCancelledError } from '../services/api'
import type { Credentials } from '../types'

const authenticated = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)
const expiresIn = ref<number | null>(null)
let expiryTimer: ReturnType<typeof setTimeout> | null = null
let sessionVersion = 0
let loginController: AbortController | null = null

function clearExpiryTimer() {
  if (expiryTimer !== null) clearTimeout(expiryTimer)
  expiryTimer = null
}

export function useAuth() {
  function logout() {
    sessionVersion += 1
    loginController?.abort()
    loginController = null
    clearExpiryTimer()
    api.logout()
    authenticated.value = false
    loading.value = false
    expiresIn.value = null
    error.value = null
  }

  async function login(credentials: Credentials) {
    loginController?.abort()
    const controller = new AbortController()
    loginController = controller
    const version = ++sessionVersion
    clearExpiryTimer()
    api.logout()
    authenticated.value = false
    expiresIn.value = null
    loading.value = true
    error.value = null
    try {
      const session = await api.login(credentials, controller.signal)
      if (version !== sessionVersion) return
      expiresIn.value = session.expiresIn
      authenticated.value = true
      expiryTimer = setTimeout(logout, session.expiresIn * 1000)
    } catch (cause) {
      if (version === sessionVersion && !(cause instanceof RequestCancelledError)) {
        error.value = cause instanceof Error ? cause.message : 'No se pudo iniciar sesión.'
      }
    } finally {
      if (version === sessionVersion) {
        loginController = null
        loading.value = false
      }
    }
  }

  onUnmounted(logout)

  return {
    authenticated: readonly(authenticated),
    loading: readonly(loading),
    error: readonly(error),
    expiresIn: readonly(expiresIn),
    login,
    logout,
  }
}
