<template>
  <label class="relative block" :style="{ width }">
    <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-3" />
    <input
      :value="modelValue"
      type="search"
      class="input h-8 pl-8 pr-2 text-xs"
      :placeholder="placeholder"
      :aria-label="placeholder"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  </label>
</template>

<script setup>
import { Search } from 'lucide-vue-next'

defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '搜索' },
  width: { type: String, default: '240px' },
})
defineEmits(['update:modelValue'])
</script>