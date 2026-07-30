import { createApp } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import MatrixTable from './MatrixTable.vue'

const mountedApps: ReturnType<typeof createApp>[] = []

afterEach(() => {
  mountedApps.splice(0).forEach((app) => app.unmount())
  document.body.replaceChildren()
})

describe('MatrixTable', () => {
  it('etiqueta la matriz y expone las coordenadas de filas y columnas', () => {
    const root = document.createElement('div')
    const app = createApp(MatrixTable, { title: 'Q ortogonal', label: 'Factor / Q', matrix: [[1, 0], [0, 1]] })
    app.mount(root)
    mountedApps.push(app)

    expect(root.querySelector('caption')?.textContent).toContain('Matriz Q ortogonal con 2 filas y 2 columnas')
    expect([...root.querySelectorAll('thead th')].map((cell) => cell.textContent)).toEqual(['', 'C1', 'C2'])
    expect([...root.querySelectorAll('tbody th')].map((cell) => cell.textContent)).toEqual(['F1', 'F2'])
    expect(root.querySelectorAll('thead th[scope="col"]')).toHaveLength(3)
    expect(root.querySelectorAll('tbody th[scope="row"]')).toHaveLength(2)
  })
})
