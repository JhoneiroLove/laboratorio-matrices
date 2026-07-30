<script setup lang="ts">
import { reactive } from 'vue'

defineProps<{ loading: boolean; error: string | null }>()
const emit = defineEmits<{ submit: [credentials: { username: string; password: string }] }>()
const form = reactive({ username: '', password: '' })

function submit() {
  emit('submit', { ...form })
}
</script>

<template>
  <main class="login-shell">
    <section class="login-intro" aria-labelledby="login-title">
      <p class="eyebrow">Laboratorio numérico / 01</p>
      <h1 id="login-title">Matrices,<br><em>con claridad.</em></h1>
      <p class="lede">Rotá, descomponé y analizá matrices rectangulares en un espacio de trabajo enfocado.</p>
      <div class="axis-mark" aria-hidden="true"><span>x</span><span>y</span></div>
    </section>
    <section class="login-card">
      <div>
        <p class="step-number">Acceso</p>
        <h2>Ingresá al laboratorio</h2>
      </div>
      <form @submit.prevent="submit">
        <label for="username">Usuario</label>
        <input id="username" v-model.trim="form.username" name="username" autocomplete="username" required :disabled="loading">
        <label for="password">Contraseña</label>
        <input id="password" v-model="form.password" name="password" type="password" autocomplete="current-password" required :disabled="loading">
        <p v-if="error" class="form-error" role="alert">{{ error }}</p>
        <button class="primary-button" type="submit" :disabled="loading">
          <span>{{ loading ? 'Autenticando…' : 'Abrir laboratorio' }}</span><span aria-hidden="true">→</span>
        </button>
      </form>
      <p class="security-note">Tu token de acceso permanece solo en la memoria de esta pestaña y se elimina cuando termina la sesión.</p>
    </section>
  </main>
</template>
