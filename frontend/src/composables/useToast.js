/** src/composables/useToast.js — 全局轻提示队列 */
import { ref } from 'vue'

const items = ref([])
let seq = 0

export function useToast() {
  function dismiss(id) {
    items.value = items.value.filter((t) => t.id !== id)
  }

  function push(message, type = 'success', title = '') {
    const id = ++seq
    items.value.push({ id, message, type, title })
    setTimeout(() => dismiss(id), 3000)
  }

  return {
    items,
    dismiss,
    success: (m, t) => push(m, 'success', t),
    error: (m, t) => push(m, 'error', t),
    info: (m, t) => push(m, 'info', t),
  }
}
