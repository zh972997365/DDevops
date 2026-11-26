<template>
  <div class="flex flex-col items-center justify-center px-6 text-center" :class="compact ? 'py-8' : 'py-14'">
    <span class="grid h-11 w-11 place-items-center rounded-full border border-dashed border-line-strong bg-sunken text-ink-3">
      <component :is="icon || Inbox" class="h-5 w-5" />
    </span>
    <p class="mt-3 text-[13px] font-medium text-ink">{{ title }}</p>
    <p v-if="description" class="mt-1 max-w-sm text-xs leading-5 text-ink-3">{{ description }}</p>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup>
import { Inbox } from 'lucide-vue-next'

defineProps({
  title: { type: String, default: '暂无数据' },
  description: { type: String, default: '' },
  icon: { type: [Object, Function], default: null },
  compact: { type: Boolean, default: false },
})
</script>