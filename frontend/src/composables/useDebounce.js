/** src/composables/useDebounce.js — 防抖执行器，卸载自动清理 */
import { onBeforeUnmount } from 'vue'

export function useDebounce(delay = 300) {
  let timer = null

  function run(fn) {
    clearTimeout(timer)
    timer = setTimeout(fn, delay)
  }

  onBeforeUnmount(() => clearTimeout(timer))
  return { run }
}
