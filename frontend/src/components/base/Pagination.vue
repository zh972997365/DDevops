<template>
  <nav class="flex flex-wrap items-center gap-3 border-t border-line px-4 py-3" aria-label="分页">
    <p class="text-xs text-ink-3">
      <span class="num text-ink-2">{{ from }}</span> 至 <span class="num text-ink-2">{{ to }}</span>
      <span class="mx-1 opacity-60">/</span>
      共 <span class="num text-ink-2">{{ total }}</span> 条
    </p>

    <div class="ml-auto flex items-center gap-1">
      <button class="page-btn" type="button" :disabled="current <= 1" aria-label="上一页" @click="go(current - 1)">
        <ChevronLeft class="h-3.5 w-3.5" />
      </button>

      <template v-for="(p, i) in pages" :key="`${p}-${i}`">
        <span v-if="p === '...'" class="px-1 text-xs text-ink-3">…</span>
        <button
          v-else
          class="page-btn num"
          :class="{ 'is-active': p === current }"
          type="button"
          :aria-current="p === current ? 'page' : undefined"
          @click="go(p)"
        >
          {{ p }}
        </button>
      </template>

      <button
        class="page-btn"
        type="button"
        :disabled="current >= totalPages"
        aria-label="下一页"
        @click="go(current + 1)"
      >
        <ChevronRight class="h-3.5 w-3.5" />
      </button>

      <span class="relative ml-1 inline-flex items-center">
        <select
          class="h-7 cursor-pointer appearance-none rounded-[6px] border border-line bg-solid pl-2 pr-6 text-xs text-ink-2 outline-none focus:border-primary"
          :value="pageSize"
          aria-label="每页条数"
          @change="$emit('update:pageSize', Number($event.target.value))"
        >
          <option v-for="s in pageSizes" :key="s" :value="s">{{ s }} / 页</option>
        </select>
        <ChevronDown class="select-chevron page-size-chevron" />
      </span>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = defineProps({
  current: { type: Number, default: 1 },
  total: { type: Number, default: 0 },
  pageSize: { type: Number, default: 10 },
  pageSizes: { type: Array, default: () => [10, 20, 50] },
})
const emit = defineEmits(['update:page', 'update:pageSize'])

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const from = computed(() => (props.total === 0 ? 0 : (props.current - 1) * props.pageSize + 1))
const to = computed(() => Math.min(props.total, props.current * props.pageSize))

const pages = computed(() => {
  const n = totalPages.value
  const c = props.current
  if (n <= 7) return Array.from({ length: n }, (_, i) => i + 1)
  const out = [1]
  const start = Math.max(2, c - 1)
  const end = Math.min(n - 1, c + 1)
  if (start > 2) out.push('...')
  for (let i = start; i <= end; i += 1) out.push(i)
  if (end < n - 1) out.push('...')
  out.push(n)
  return out
})

function go(p) {
  if (p < 1 || p > totalPages.value || p === props.current) return
  emit('update:page', p)
}
</script>

<style scoped>
.page-btn {
  display: inline-grid;
  height: 28px;
  min-width: 28px;
  place-items: center;
  border: 1px solid transparent;
  border-radius: 6px;
  font-size: 12px;
  color: var(--ink-2);
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}
.page-btn:hover:not(:disabled):not(.is-active) {
  background: var(--surface-sunken);
  color: var(--ink);
}
.page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.page-btn.is-active {
  border-color: var(--line);
  background: var(--surface-solid);
  color: var(--primary-ink);
  font-weight: 600;
  box-shadow: var(--shadow-control);
}
.page-size-chevron {
  right: 6px;
  width: 12px;
  height: 12px;
}
</style>