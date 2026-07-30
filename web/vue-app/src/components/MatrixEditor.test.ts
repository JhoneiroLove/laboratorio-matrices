import { createApp, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import MatrixEditor from './MatrixEditor.vue'

const mountedApps: ReturnType<typeof createApp>[] = []

function mountEditor(onProcess = vi.fn()) {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(MatrixEditor, { loading: false, onProcess })
  app.mount(root)
  mountedApps.push(app)
  return { root, onProcess }
}

afterEach(() => {
  mountedApps.splice(0).forEach((app) => app.unmount())
  document.body.replaceChildren()
})

describe('MatrixEditor', () => {
  it('emite la matriz rectangular interpretada', async () => {
    const { root, onProcess } = mountEditor()
    const textarea = root.querySelector('textarea') as HTMLTextAreaElement
    textarea.value = '1 2\n3 4\n5 6'
    textarea.dispatchEvent(new Event('input'))
    await nextTick()
    root.querySelector('button')?.click()
    expect(onProcess).toHaveBeenCalledWith([[1, 2], [3, 4], [5, 6]])
  })

  it('deshabilita el procesamiento e informa las filas inválidas', async () => {
    const { root } = mountEditor()
    const textarea = root.querySelector('textarea') as HTMLTextAreaElement
    textarea.value = '1 2\n3'
    textarea.dispatchEvent(new Event('input'))
    await nextTick()
    expect((root.querySelector('button') as HTMLButtonElement).disabled).toBe(true)
    expect(root.querySelector('[role="alert"]')?.textContent).toContain('La fila 2')
  })
})
