<template>
  <div ref="root" class="filter-select-root">
    <button
      ref="trigger"
      type="button"
      class="filter-select"
      :class="[open && 'is-open', !hasSelection && 'is-placeholder']"
      :disabled="disabled"
      :aria-label="ariaLabel || placeholder"
      :aria-expanded="open"
      aria-haspopup="listbox"
      @click="toggle"
      @keydown.down.prevent="openMenu('first')"
      @keydown.up.prevent="openMenu('last')"
    >
      <span class="filter-select-label">{{ selectedLabel }}</span>
      <ChevronDown class="filter-chevron" aria-hidden="true" />
    </button>

    <Teleport to="body">
      <Transition name="filter-pop">
        <div
          v-if="open"
          :id="menuId"
          ref="menu"
          class="filter-menu"
          :style="menuStyle"
          role="listbox"
          tabindex="-1"
          :aria-label="ariaLabel || placeholder"
          :aria-activedescendant="activeIndex >= 0 ? optionId(activeIndex) : undefined"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.home.prevent="moveTo(0)"
          @keydown.end.prevent="moveTo(options.length - 1)"
          @keydown.enter.prevent="chooseActive"
          @keydown.esc.prevent="close(true)"
          @keydown.tab="close(true)"
        >
          <button
            v-for="(option, index) in options"
            :id="optionId(index)"
            :key="String(option.value)"
            type="button"
            role="option"
            tabindex="-1"
            class="filter-option"
            :class="[
              option.value === modelValue && 'is-selected',
              index === activeIndex && 'is-active',
            ]"
            :aria-selected="option.value === modelValue"
            @mouseenter="activeIndex = index"
            @click="pick(option.value)"
          >
            <span class="filter-option-label" :title="option.label">{{ option.label }}</span>
            <Check v-if="option.value === modelValue" class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
          </button>
          <p v-if="!options.length" class="filter-empty">没有可选项</p>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { Check, ChevronDown } from 'lucide-vue-next'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '请选择' },
  ariaLabel: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const uid = useId()
const menuId = `filter-menu-${uid}`
const root = ref(null)
const trigger = ref(null)
const menu = ref(null)
const open = ref(false)
const activeIndex = ref(-1)
const menuStyle = ref({})

const hasSelection = computed(() => props.options.some((option) => option.value === props.modelValue))
const selectedLabel = computed(
  () => props.options.find((option) => option.value === props.modelValue)?.label || props.placeholder
)

function optionId(index) {
  return `filter-option-${uid}-${index}`
}

function updatePosition() {
  if (!trigger.value) return
  const rect = trigger.value.getBoundingClientRect()
  const gap = 6
  const edge = 12
  const width = Math.max(120, Math.min(320, Math.max(rect.width, 200), window.innerWidth - edge * 2))
  const left = Math.min(Math.max(edge, rect.left), window.innerWidth - width - edge)
  const spaceBelow = window.innerHeight - rect.bottom - edge
  const spaceAbove = rect.top - edge
  const openUp = spaceBelow < 180 && spaceAbove > spaceBelow
  const maxHeight = Math.max(96, Math.min(280, openUp ? spaceAbove : spaceBelow))

  menuStyle.value = openUp
    ? {
        left: `${left}px`,
        bottom: `${window.innerHeight - rect.top + gap}px`,
        width: `${width}px`,
        maxHeight: `${maxHeight}px`,
      }
    : {
        left: `${left}px`,
        top: `${rect.bottom + gap}px`,
        width: `${width}px`,
        maxHeight: `${maxHeight}px`,
      }
}

function openMenu(position = 'selected') {
  if (props.disabled || open.value) return
  const selectedIndex = props.options.findIndex((option) => option.value === props.modelValue)
  if (position === 'first') activeIndex.value = 0
  else if (position === 'last') activeIndex.value = props.options.length - 1
  else activeIndex.value = selectedIndex >= 0 ? selectedIndex : 0

  updatePosition()
  open.value = true
  nextTick(() => menu.value?.focus())
}

function close(returnFocus = false) {
  if (!open.value) return
  open.value = false
  activeIndex.value = -1
  if (returnFocus) trigger.value?.focus()
}

function toggle() {
  if (open.value) close()
  else openMenu()
}

function move(step) {
  if (!props.options.length) return
  const start = activeIndex.value < 0 ? (step > 0 ? -1 : 0) : activeIndex.value
  activeIndex.value = (start + step + props.options.length) % props.options.length
}

function moveTo(index) {
  if (!props.options.length) return
  activeIndex.value = Math.min(Math.max(0, index), props.options.length - 1)
}

function chooseActive() {
  if (activeIndex.value >= 0 && props.options[activeIndex.value]) {
    pick(props.options[activeIndex.value].value)
  }
}

function pick(value) {
  emit('update:modelValue', value)
  close(true)
}

function onDocumentPointerDown(event) {
  if (!open.value) return
  if (root.value?.contains(event.target) || menu.value?.contains(event.target)) return
  close()
}

function onViewportChange() {
  updatePosition()
}

watch(open, (isOpen) => {
  if (isOpen) {
    window.addEventListener('scroll', onViewportChange, true)
    window.addEventListener('resize', onViewportChange)
  } else {
    window.removeEventListener('scroll', onViewportChange, true)
    window.removeEventListener('resize', onViewportChange)
  }
})

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) close()
  }
)

onMounted(() => {
  document.addEventListener('mousedown', onDocumentPointerDown)
  document.addEventListener('keydown', onKey)
})

function onKey(event) {
  if (event.key === 'Escape' && open.value) close(true)
}

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentPointerDown)
  document.removeEventListener('keydown', onKey)
  window.removeEventListener('scroll', onViewportChange, true)
  window.removeEventListener('resize', onViewportChange)
})
</script>