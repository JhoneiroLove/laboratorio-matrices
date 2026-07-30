<script setup lang="ts">
import type { Matrix } from '../types'
import { formatNumber } from '../utils/matrix'

defineProps<{ title: string; label: string; matrix: Matrix }>()
</script>

<template>
  <article class="matrix-card">
    <header><p class="eyebrow">{{ label }}</p><h3>{{ title }}</h3></header>
    <div v-if="matrix.length" class="matrix-scroll" tabindex="0" :aria-label="`${title}: ${matrix.length} filas por ${matrix[0].length} columnas`">
      <table>
        <caption class="sr-only">Matriz {{ title }} con {{ matrix.length }} filas y {{ matrix[0].length }} columnas</caption>
        <thead>
          <tr>
            <th scope="col" aria-label="Coordenadas de filas y columnas" />
            <th v-for="(_, columnIndex) in matrix[0]" :key="columnIndex" scope="col">C{{ columnIndex + 1 }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, rowIndex) in matrix" :key="rowIndex">
            <th scope="row">F{{ rowIndex + 1 }}</th>
            <td v-for="(cell, columnIndex) in row" :key="columnIndex">{{ formatNumber(cell) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-else class="empty-matrix">La API no devolvió una matriz</p>
  </article>
</template>
