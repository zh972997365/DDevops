<template>
  <span class="num">{{ text }}</span>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  value: { type: Number, default: 0 },
  duration: { type: Number, default: 700 },
  decimals: { type: Number, default: 0 },
})

const display = ref(0)
let raf = 0
let started = false
const reduced = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const text = computed(() =>
  new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: props.decimals,
    maximumFractionDigits: props.decimals,
  }).format(display.value)
)

function animate(to) {
  cancelAnimationFrame(raf)
  if (reduced) {
    display.value = to
    return
  }
  const from = display.value
  const delta = to - from
  if (Math.abs(delta) < 0.001) return
  const start = performance.now()
  const step = (now) => {
    const p = Math.min(1, (now - start) / props.duration)
    const eased = 1 - (1 - p) ** 3
    display.value = from + delta * eased
    if (p < 1) raf = requestAnimationFrame(step)
  }
  raf = requestAnimationFrame(step)
}

watch(
  () => props.value,
  (v) => animate(Number(v) || 0)
)

onMounted(() => {
  if (started) return
  started = true
  animate(Number(props.value) || 0)
})

onBeforeUnmount(() => cancelAnimationFrame(raf))
</script>