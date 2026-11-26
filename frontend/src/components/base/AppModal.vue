<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="modal-root" @click.self="onOverlay">
        <div
          ref="panel"
          class="modal-panel"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          :style="{ width }"
          tabindex="-1"
        >
          <header class="flex items-start gap-3 border-b border-line px-5 py-4">
            <div class="min-w-0 flex-1">
              <h2 class="text-[15px] font-semibold leading-6 text-ink">{{ title }}</h2>
              <p v-if="subtitle" class="mt-0.5 text-xs leading-5 text-ink-3">{{ subtitle }}</p>
            </div>
            <button class="icon-btn -mr-1 -mt-0.5" type="button" aria-label="关闭" @click="close">
              <X class="h-4 w-4" />
            </button>
          </header>

          <div class="modal-body">
            <slot />
          </div>

          <footer v-if="$slots.footer" class="flex items-center justify-end gap-2 border-t border-line px-5 py-3.5">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  width: { type: String, default: '520px' },
  persistent: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])
const panel = ref(null)
let prevOverflow = ''

function close() {
  emit('update:modelValue', false)
}
function onOverlay() {
  if (!props.persistent) close()
}
function onKey(e) {
  if (e.key === 'Escape' && props.modelValue && !props.persistent) close()
}

watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      prevOverflow = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      window.addEventListener('keydown', onKey)
      await nextTick()
      const first = panel.value?.querySelector('input, select, textarea, [data-autofocus]')
      ;(first || panel.value)?.focus?.({ preventScroll: true })
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
.modal-root {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  overflow-y: auto;
  padding: 12vh 16px 40px;
  background: var(--overlay);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}
.modal-panel {
  max-width: 100%;
  max-height: 84vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: var(--surface-solid);
  box-shadow: var(--shadow-pop);
  animation: pop-in 160ms var(--ease-out) both;
}
.modal-body {
  overflow-y: auto;
  padding: 20px;
}
@media (max-width: 640px) {
  .modal-root {
    padding: 8vh 12px 24px;
  }
  .modal-panel {
    max-height: 88vh;
  }
}
</style>