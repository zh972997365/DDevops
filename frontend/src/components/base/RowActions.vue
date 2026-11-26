<template>
  <div class="row-actions" @click.stop>
    <div class="row-actions-inline">
      <component
        :is="item.href ? 'a' : 'button'"
        v-for="(item, i) in visibleItems"
        :key="`${item.label}-${i}`"
        :href="item.href || undefined"
        :target="item.href ? '_blank' : undefined"
        :rel="item.href ? 'noopener' : undefined"
        :type="item.href ? undefined : 'button'"
        class="btn btn-xs"
        :class="variantClass(item)"
        :disabled="!item.href && item.disabled"
        :aria-disabled="item.disabled || undefined"
        :title="item.title || undefined"
        @click="choose(item)"
      >
        <component :is="item.icon" v-if="item.icon" class="h-3 w-3 shrink-0" />
        {{ item.label }}
      </component>
    </div>

    <button
      ref="triggerEl"
      class="row-actions-trigger"
      type="button"
      aria-haspopup="menu"
      :aria-expanded="open"
      :aria-label="label ? `${label}的操作` : '更多操作'"
      :title="label ? `${label}的操作` : '更多操作'"
      @click.stop="toggle"
      @keydown.down.prevent="openAt(true)"
      @keydown.esc.prevent="close(true)"
    >
      <MoreHorizontal class="h-4 w-4" />
    </button>

    <Teleport to="body">
      <Transition name="row-menu">
        <div
          v-if="open"
          ref="menuEl"
          class="row-menu"
          role="menu"
          :style="{ left: `${pos.left}px`, top: `${pos.top}px` }"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.home.prevent="move(-visibleItems.length)"
          @keydown.end.prevent="move(visibleItems.length)"
          @keydown.esc.prevent="close(true)"
          @keydown.tab="close(false)"
        >
          <component
            :is="item.href ? 'a' : 'button'"
            v-for="(item, i) in visibleItems"
            :key="`${item.label}-${i}`"
            :href="item.href || undefined"
            :target="item.href ? '_blank' : undefined"
            :rel="item.href ? 'noopener' : undefined"
            :type="item.href ? undefined : 'button'"
            class="row-menu-item"
            :class="[item.tone === 'danger' && 'is-danger', item.disabled && 'is-disabled']"
            role="menuitem"
            :tabindex="-1"
            :aria-disabled="item.disabled || undefined"
            :title="item.title || undefined"
            @click="choose(item)"
          >
            <component :is="item.icon" v-if="item.icon" class="h-3.5 w-3.5 shrink-0" />
            <span class="truncate">{{ item.label }}</span>
          </component>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'

const props = defineProps({
  items: { type: Array, default: () => [] },
  label: { type: String, default: '' },
})

const visibleItems = computed(() => props.items.filter((it) => it.show !== false))
const triggerEl = ref(null)
const menuEl = ref(null)
const open = ref(false)
const pos = reactive({ left: 0, top: 0 })
let lastFocus = null

function variantClass(item) {
  if (item.disabled) return 'row-action-ghost opacity-45'
  if (item.tone === 'danger') return 'row-action-danger'
  if (item.variant === 'primary') return 'row-action-primary'
  return 'row-action-ghost'
}

function enabledIndexes() {
  return visibleItems.value.map((it, i) => (it.disabled ? -1 : i)).filter((i) => i >= 0)
}

function place() {
  const rect = triggerEl.value?.getBoundingClientRect()
  if (!rect) return
  const width = 176
  const height = visibleItems.value.length * 38 + 8
  const gap = 6
  const left = Math.min(Math.max(8, rect.right - width), window.innerWidth - width - 8)
  let top = rect.bottom + gap
  if (top + height > window.innerHeight - 8) top = Math.max(8, rect.top - height - gap)
  pos.left = Math.round(left)
  pos.top = Math.round(top)
}

async function openAt(focusFirst) {
  place()
  open.value = true
  lastFocus = document.activeElement
  await nextTick()
  if (focusFirst) {
    const first = enabledIndexes()[0]
    if (first !== undefined) menuEl.value?.children[first]?.focus()
  }
}

function toggle() {
  if (open.value) close(false)
  else openAt(false)
}

function close(returnFocus) {
  open.value = false
  if (returnFocus && lastFocus instanceof HTMLElement) lastFocus.focus()
}

function move(step) {
  const list = enabledIndexes()
  if (!list.length) return
  const nodes = [...(menuEl.value?.children || [])]
  const current = nodes.indexOf(document.activeElement)
  let at = list.indexOf(current)
  at = at === -1 ? 0 : (at + step + list.length) % list.length
  nodes[list[at]]?.focus()
}

function choose(item) {
  if (item.disabled) return
  close(false)
  item.onSelect?.()
}

function onOutside(e) {
  if (!open.value) return
  if (triggerEl.value?.contains(e.target) || menuEl.value?.contains(e.target)) return
  close(false)
}
function dismiss() {
  if (open.value) close(false)
}

document.addEventListener('mousedown', onOutside)
window.addEventListener('scroll', dismiss, true)
window.addEventListener('resize', dismiss)

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onOutside)
  window.removeEventListener('scroll', dismiss, true)
  window.removeEventListener('resize', dismiss)
})
</script>

<style scoped>
.row-actions {
  display: flex;
  justify-content: flex-end;
}
.row-actions-inline {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.row-actions-inline .btn {
  font-size: 11.5px;
}
.row-actions-inline .btn:hover {
  transform: none;
}
.row-action-ghost {
  color: var(--ink-2);
}
.row-action-ghost:hover:not(:disabled) {
  background: var(--surface-sunken);
  color: var(--ink);
}
.row-action-primary {
  color: var(--primary-ink);
  font-weight: 500;
}
.row-action-primary:hover:not(:disabled) {
  background: var(--primary-soft);
  color: var(--primary-ink);
}
.row-action-danger {
  color: var(--failed);
}
.row-action-danger:hover:not(:disabled) {
  background: var(--failed-soft);
  color: var(--failed);
}
.row-actions-trigger {
  display: none;
  width: 32px;
  height: 28px;
  place-items: center;
  border-radius: 6px;
  color: var(--ink-3);
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}
.row-actions-trigger:hover,
.row-actions-trigger[aria-expanded='true'] {
  background: var(--surface-sunken);
  color: var(--ink);
}
.row-actions-trigger:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}
.row-menu {
  position: fixed;
  z-index: 75;
  width: 176px;
  padding: 4px;
  border-radius: 9px;
  background: var(--surface-solid);
  box-shadow: var(--shadow-pop);
}
.dark .row-menu {
  background: var(--surface-raised);
}
.row-menu-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  padding: 8px 12px;
  color: var(--ink-2);
  font-size: 12.5px;
  line-height: 20px;
  text-align: left;
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}
.row-menu-item + .row-menu-item {
  margin-top: 1px;
}
.row-menu-item:hover,
.row-menu-item:focus-visible {
  background: var(--surface-sunken);
  color: var(--ink);
  outline: none;
}
.row-menu-item.is-danger {
  color: var(--failed);
}
.row-menu-item.is-danger:hover,
.row-menu-item.is-danger:focus-visible {
  background: var(--failed-soft);
}
.row-menu-item.is-disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
.row-menu-enter-active,
.row-menu-leave-active {
  transition: transform 120ms var(--ease-out);
  transform-origin: top right;
}
.row-menu-enter-from,
.row-menu-leave-to {
  transform: translateY(-3px) scale(0.985);
}
@media (max-width: 767px) {
  .row-actions-inline {
    display: none;
  }
  .row-actions-trigger {
    display: inline-grid;
    width: 36px;
    height: 36px;
  }
}
</style>
