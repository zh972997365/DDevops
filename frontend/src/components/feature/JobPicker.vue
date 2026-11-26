<template>
  <div ref="root" class="relative">
    <button
      type="button"
      class="input flex items-center gap-2 text-left"
      :class="!model && 'text-ink-3'"
      :disabled="disabled"
      :aria-expanded="open"
      aria-haspopup="listbox"
      @click="toggle"
    >
      <span class="min-w-0 flex-1 truncate font-mono text-xs">{{ selectedLabel || placeholder }}</span>
      <ChevronDown class="h-3.5 w-3.5 shrink-0 text-ink-3 transition-transform" :class="open && 'rotate-180'" />
    </button>

    <Transition name="pop">
      <div
        v-if="open"
        class="mt-1.5 w-full overflow-hidden rounded-field border border-line bg-solid shadow-card"
      >
        <div class="border-b border-line p-2">
          <label class="relative block">
            <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-3" />
            <input
              ref="searchEl"
              v-model="keyword"
              type="search"
              class="input h-8 pl-8 text-xs"
              placeholder="搜索任务 ID"
              aria-label="搜索任务"
            />
          </label>
        </div>

        <div class="max-h-[200px] overflow-y-auto p-1" role="listbox" aria-label="任务列表">
          <template v-for="g in visibleGroups" :key="g.label">
            <p class="section-label px-2 pb-1 pt-2">{{ g.label }}</p>
            <button
              v-for="o in g.options"
              :key="o.value"
              type="button"
              role="option"
              :aria-selected="o.value === model"
              class="flex w-full items-center gap-2 rounded-[6px] px-2 py-1.5 text-left font-mono text-[11.5px] transition-colors hover:bg-sunken"
              :class="o.value === model ? 'bg-primary-soft text-primary-ink' : 'text-ink-2'"
              @click="pick(o.value)"
            >
              <span class="min-w-0 flex-1 truncate">{{ o.label }}</span>
              <Check v-if="o.value === model" class="h-3 w-3 shrink-0" />
            </button>
          </template>
          <p v-if="!visibleGroups.length" class="px-2 py-6 text-center text-[11px] text-ink-3">没有匹配的任务</p>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Check, ChevronDown, Search } from 'lucide-vue-next'

const props = defineProps({
  modelValue: { type: String, default: '' },
  groups: { type: Array, default: () => [] },
  placeholder: { type: String, default: '选择任务' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const root = ref(null)
const searchEl = ref(null)
const open = ref(false)
const keyword = ref('')

const selectedLabel = computed(() => {
  for (const g of props.groups) {
    const hit = g.options.find((o) => o.value === props.modelValue)
    if (hit) return hit.label
  }
  return ''
})

const visibleGroups = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return props.groups
  return props.groups
    .map((g) => ({
      label: g.label,
      options: g.options.filter((o) => o.label.toLowerCase().includes(kw)),
    }))
    .filter((g) => g.options.length)
})

async function toggle() {
  if (props.disabled) return
  open.value = !open.value
  if (open.value) {
    keyword.value = ''
    await nextTick()
    searchEl.value?.focus()
  }
}
function pick(value) {
  emit('update:modelValue', value)
  open.value = false
}
function onDoc(e) {
  if (open.value && root.value && !root.value.contains(e.target)) open.value = false
}
function onKey(e) {
  if (e.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('mousedown', onDoc)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDoc)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.pop-enter-active,
.pop-leave-active {
  transition: opacity 120ms var(--ease), transform 120ms var(--ease);
  transform-origin: top left;
}
.pop-enter-from,
.pop-leave-to {
  transform: translateY(-3px) scale(0.99);
}
</style>