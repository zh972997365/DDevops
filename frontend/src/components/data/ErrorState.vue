<template>
  <div class="flex flex-col items-center justify-center px-6 py-12 text-center">
    <span class="grid h-11 w-11 place-items-center rounded-full border border-line bg-failed-soft text-failed">
      <TriangleAlert class="h-5 w-5" />
    </span>
    <p class="mt-3 text-[13px] font-medium text-ink">数据加载失败</p>
    <p class="mt-1 max-w-md text-xs leading-5 text-ink-3">{{ message || '请检查服务是否可用，然后重试。' }}</p>
    <button v-if="retryable" class="btn btn-secondary btn-sm mt-4" type="button" @click="$emit('retry')">
      <RotateCcw class="h-3.5 w-3.5" />
      重新加载
    </button>
  </div>
</template>

<script setup>
import { RotateCcw, TriangleAlert } from 'lucide-vue-next'

defineProps({
  message: { type: String, default: '' },
  retryable: { type: Boolean, default: true },
})
defineEmits(['retry'])
</script>