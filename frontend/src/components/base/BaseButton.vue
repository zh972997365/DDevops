<template>
  <button
    :type="type"
    class="btn"
    :class="[variantClass, sizeClass, block && 'w-full']"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
  >
    <span v-if="loading" class="spinner" aria-hidden="true" />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  variant: { type: String, default: 'primary' },
  size: { type: String, default: 'md' },
  type: { type: String, default: 'button' },
  loading: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  block: { type: Boolean, default: false },
})

const variantClass = computed(
  () =>
    ({
      primary: 'btn-primary',
      secondary: 'btn-secondary',
      soft: 'btn-soft',
      ghost: 'btn-ghost',
      danger: 'btn-danger',
    })[props.variant] || 'btn-primary'
)

const sizeClass = computed(
  () =>
    ({
      sm: 'btn-sm',
      xs: 'btn-xs',
      md: '',
      lg: 'h-10 px-4 text-md',
    })[props.size] || ''
)
</script>

<style scoped>
.spinner {
  width: 13px;
  height: 13px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 650ms linear infinite;
  opacity: 0.85;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>