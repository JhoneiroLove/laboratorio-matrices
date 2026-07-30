<script setup lang="ts">
import type { MatrixStatistic, StatisticSet } from '../types'
import { formatNumber } from '../utils/matrix'

defineProps<{ global: StatisticSet; matrices: readonly MatrixStatistic[] }>()
const metrics: Array<{ key: keyof StatisticSet; label: string }> = [
  { key: 'minimum', label: 'Mínimo' },
  { key: 'maximum', label: 'Máximo' },
  { key: 'average', label: 'Promedio' },
  { key: 'sum', label: 'Suma' },
  { key: 'elements', label: 'Elementos' },
]

function matrixName(name: string): string {
  return name === 'rotated' ? 'Rotada' : name
}
</script>

<template>
  <section class="statistics-panel" aria-labelledby="statistics-title">
    <div class="section-heading"><div><p class="eyebrow">Medición / Σ</p><h2 id="statistics-title">Estadísticas</h2></div></div>
    <div class="global-stats">
      <div v-for="metric in metrics" :key="metric.key">
        <span>{{ metric.label }}</span><strong>{{ formatNumber(global[metric.key]) }}</strong>
      </div>
    </div>
    <div class="stats-table-wrap" tabindex="0">
      <table class="stats-table">
        <caption class="sr-only">Estadísticas de cada matriz resultante</caption>
        <thead><tr><th>Matriz</th><th v-for="metric in metrics" :key="metric.key">{{ metric.label }}</th></tr></thead>
        <tbody>
          <tr v-for="matrix in matrices" :key="matrix.name">
            <th><span>{{ matrixName(matrix.name) }}</span><small class="matrix-diagonal" :class="{ positive: matrix.diagonal }">{{ matrix.diagonal ? 'Diagonal' : 'No diagonal' }}</small></th>
            <td v-for="metric in metrics" :key="metric.key">{{ formatNumber(matrix[metric.key]) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
