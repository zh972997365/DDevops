<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed right-4 top-[68px] z-[90] flex w-[min(360px,calc(100vw-32px))] flex-col gap-2">
      <TransitionGroup name="toast">
        <div
          v-for="t in items"
          :key="t.id"
          class="toast pointer-events-auto flex items-start gap-2.5 rounded-card border border-line bg-solid px-3.5 py-3 shadow-pop"
          :class="`tone-${t.type || 'info'}`"
          role="status"
        >
          <component :is="ICON[t.type] || ICON.info" class="mt-0.5 h-4 w-4 shrink-0" />
          <div class="min-w-0 flex-1">
            <p v-if="t.title" class="text-xs font-semibold text-ink">{{ t.title }}</p>
            <p class="text-xs leading-5 text-ink-2">{{ t.message }}</p>
          </div>
          <button
            class="grid h-5 w-5 shrink-0 place-items-center rounded text-ink-3 transition-colors hover:bg-sunken hover:text-ink"
            type="button"
            aria-label="关闭提示"
            @click="dismiss(t.id)"
          >
            <X class="h-3 w-3" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { CircleCheck, CircleX, Info, X } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'

const { items, dismiss } = useToast()
const ICON = { success: CircleCheck, error: CircleX, info: Info }
</script>

<style scoped>
.toast {
  animation: toast-in 200ms var(--ease-out) both;
}
.tone-success {
  color: var(--success);
}
.tone-error {
  color: var(--failed);
}
.tone-info {
  color: var(--info);
}
.toast p {
  color: var(--ink-2);
}
.toast p:first-child {
  color: var(--ink);
}
@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateY(-6px) scale(0.99);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
.toast-leave-active {
  transition: opacity 160ms var(--ease), transform 160ms var(--ease);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(10px);
}
</style>