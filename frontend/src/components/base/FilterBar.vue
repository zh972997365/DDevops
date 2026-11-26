<template>
  <div class="flex flex-wrap items-center gap-2.5 border-b border-line px-4 py-2.5">
    <slot />
    <button v-if="dirty" type="button" class="btn btn-ghost btn-xs" @click="$emit('reset')">
      <RotateCcw class="h-3 w-3" />
      重置
    </button>
    <div v-if="$slots.right" class="ml-auto flex flex-wrap items-center gap-2">
      <slot name="right" />
    </div>
  </div>
</template>

<script setup>
import { RotateCcw } from 'lucide-vue-next'

defineProps({ dirty: { type: Boolean, default: false } })
defineEmits(['reset'])
</script>