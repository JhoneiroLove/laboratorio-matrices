<script setup lang="ts">
import type { MatrixResult } from '../types'
import MatrixTable from './MatrixTable.vue'
import StatisticsPanel from './StatisticsPanel.vue'

defineProps<{ result: MatrixResult }>()
</script>

<template>
  <div class="results" aria-live="polite">
    <section class="result-heading">
      <div><p class="eyebrow">Salida / 02</p><h2>Análisis de descomposición</h2></div>
      <div class="diagonal-chip" :class="{ positive: result.anyDiagonal }">
        <span class="status-dot" aria-hidden="true" />
        <span><small>Resultados combinados</small><strong>{{ result.anyDiagonal ? 'Se encontró una diagonal' : 'Ninguna es diagonal' }}</strong></span>
      </div>
    </section>
    <div class="matrix-grid">
      <MatrixTable class="rotated-card" title="Matriz rotada" label="Rotación / sentido horario" :matrix="result.rotated" />
      <MatrixTable title="Q ortogonal" label="Factor / Q" :matrix="result.q" />
      <MatrixTable title="R triangular superior" label="Factor / R" :matrix="result.r" />
    </div>
    <StatisticsPanel :global="result.globalStatistics" :matrices="result.matrixStatistics" />
    <p class="request-id">Solicitud <code>{{ result.requestId }}</code></p>
  </div>
</template>
