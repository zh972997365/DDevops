/** src/composables/useCountdown.js — 全局共享秒级时钟（多组件只起一个定时器） */
import { onBeforeUnmount, onMounted, ref } from 'vue'

const now = ref(Date.now())
let timer = null
let refCount = 0

export function useCountdown() {
  onMounted(() => {
    refCount += 1
    if (!timer) timer = setInterval(() => (now.value = Date.now()), 1000)
  })
  onBeforeUnmount(() => {
    refCount -= 1
    if (refCount <= 0 && timer) {
      clearInterval(timer)
      timer = null
    }
  })
  return { now }
}
