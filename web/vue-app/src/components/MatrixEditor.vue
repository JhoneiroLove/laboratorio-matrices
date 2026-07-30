<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Matrix } from '../types'
import { parseMatrixInput } from '../utils/matrix'

const props = defineProps<{ loading: boolean }>()
const emit = defineEmits<{ process: [matrix: Matrix] }>()
const input = ref('1  2  3\n4  5  6')
const parsed = computed(() => parseMatrixInput(input.value))
const dimensions = computed(() => parsed.value.matrix ? `${parsed.value.matrix.length} × ${parsed.value.matrix[0].length}` : '—')

function submit() {
  if (parsed.value.matrix) emit('process', parsed.value.matrix)
}
</script>

<template>
  <section class="editor-panel" aria-labelledby="matrix-input-title">
    <div class="section-heading">
      <div><p class="eyebrow">Entrada / A</p><h2 id="matrix-input-title">Matriz de origen</h2></div>
      <output class="dimension-badge" aria-label="Dimensiones de la matriz">{{ dimensions }}</output>
    </div>
    <label class="sr-only" for="matrix-input">Valores de la matriz, con una fila por línea</label>
    <textarea id="matrix-input" v-model="input" spellcheck="false" :aria-invalid="Boolean(parsed.error)" aria-describedby="matrix-help matrix-error" :disabled="loading" />
    <div class="editor-meta">
      <p id="matrix-help">Separá los valores con espacios, comas o punto y coma. Todas las filas deben tener la misma longitud.</p>
      <p v-if="parsed.error" id="matrix-error" class="inline-error" role="alert">{{ parsed.error }}</p>
    </div>
    <button class="primary-button process-button" type="button" :disabled="loading || Boolean(parsed.error)" @click="submit">
      <span>{{ loading ? 'Calculando…' : 'Procesar matriz' }}</span><span aria-hidden="true">↗</span>
    </button>
  </section>
</template>
