<script setup lang="ts">
import { watch } from 'vue'
import LoginPanel from './components/LoginPanel.vue'
import MatrixEditor from './components/MatrixEditor.vue'
import ResultsPanel from './components/ResultsPanel.vue'
import { useAuth } from './composables/useAuth'
import { useMatrixProcessor } from './composables/useMatrixProcessor'

const auth = useAuth()
const processor = useMatrixProcessor(logout)

function logout() {
  processor.reset()
  auth.logout()
}

async function login(credentials: { username: string; password: string }) {
  processor.reset()
  await auth.login(credentials)
}

watch(auth.authenticated, (isAuthenticated) => {
  if (!isAuthenticated) processor.reset()
}, { flush: 'sync' })
</script>

<template>
  <LoginPanel v-if="!auth.authenticated.value" :loading="auth.loading.value" :error="auth.error.value" @submit="login" />
  <div v-else class="app-shell">
    <header class="site-header">
      <a class="wordmark" href="#workspace" aria-label="Inicio del Laboratorio de Matrices"><span>M</span> Laboratorio de Matrices</a>
      <div class="header-rule" aria-hidden="true" />
      <button class="text-button" type="button" @click="logout">Cerrar sesión</button>
    </header>
    <main id="workspace">
      <section class="workspace-intro">
        <p class="eyebrow">Laboratorio de matrices / Sesión activa</p>
        <h1>Define la entrada.<br><em>Interpretá la estructura.</em></h1>
      </section>
      <MatrixEditor :loading="processor.loading.value" @process="processor.process" />
      <p v-if="processor.error.value" class="api-error" role="alert"><strong>Se interrumpió el procesamiento.</strong> {{ processor.error.value }}</p>
      <div v-if="processor.loading.value" class="loading-panel" role="status"><span class="loader" aria-hidden="true" /><p>Analizando la estructura de la matriz…</p></div>
      <ResultsPanel v-else-if="processor.result.value" :result="processor.result.value" />
      <section v-else class="empty-state" aria-label="Todavía no hay resultados"><span aria-hidden="true">[ A ]</span><p>La descomposición aparecerá acá después del procesamiento.</p></section>
    </main>
    <footer><span>Laboratorio de Matrices</span><span>Precisión antes que aproximación</span></footer>
  </div>
</template>
