<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="modelValue" class="drawer-root" @click.self="close">
        <aside
          class="drawer-panel"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          :style="{ width }"
        >
          <header class="flex items-start gap-3 border-b border-line px-5 py-4">
            <div class="min-w-0 flex-1">
              <h2 class="truncate text-[15px] font-semibold leading-6 text-ink">{{ title }}</h2>
              <p v-if="subtitle" class="mt-1 text-xs leading-5 text-ink-3">{{ subtitle }}</p>
            </div>
            <button class="icon-btn -mr-1 -mt-0.5" type="button" aria-label="关闭" @click="close">
              <X class="h-4 w-4" />
            </button>
          </header>

          <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            <slot />
          </div>

          <footer v-if="$slots.footer" class="flex items-center justify-end gap-2 border-t border-line bg-canvas px-5 py-3">
            <slot name="footer" />
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { onBeforeUnmount, watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  width: { type: String, default: '480px' },
})
const emit = defineEmits(['update:modelValue'])
let prevOverflow = ''

function close() {
  emit('update:modelValue', false)
}
function onKey(e) {
  if (e.key === 'Escape' && props.modelValue) close()
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      prevOverflow = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      window.addEventListener('keydown', onKey)
    } else {
      document.body.style.overflow = prevOverflow
      window.removeEventListener('keydown', onKey)
    }
  }
)

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  document.body.style.overflow = prevOverflow
})
</script>

<style scoped>
.drawer-root {
  position: fixed;
  inset: 0;
  z-index: 65;
  display: flex;
  justify-content: flex-end;
  background: var(--overlay);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}
.drawer-panel {
  display: flex;
  max-width: 100vw;
  height: 100%;
  flex-direction: column;
  border-left: 1px solid var(--line);
  background: var(--surface-solid);
  box-shadow: var(--shadow-pop);
  animation: drawer-in 220ms var(--ease-out) both;
}
@media (max-width: 640px) {
  .drawer-panel {
    width: 100% !important;
  }
}
</style>