<template>
  <AppModal
    :model-value="modelValue"
    :title="title"
    width="440px"
    :persistent="loading"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <p class="text-base leading-6 text-ink-2">{{ message }}</p>
    <ul v-if="items.length" class="mt-3 space-y-2 rounded-field border border-line bg-sunken px-3.5 py-3">
      <li v-for="(item, i) in items" :key="i" class="flex gap-2 text-xs leading-5 text-ink-2">
        <span class="mt-[7px] h-1 w-1 shrink-0 rounded-full bg-ink-3" />
        <span>{{ item }}</span>
      </li>
    </ul>
    <p v-if="detail" class="mt-3 text-xs leading-5 text-ink-3">{{ detail }}</p>

    <template #footer>
      <BaseButton variant="secondary" :disabled="loading" @click="$emit('update:modelValue', false)">
        {{ cancelText }}
      </BaseButton>
      <BaseButton :variant="danger ? 'danger' : 'primary'" :loading="loading" @click="$emit('confirm')">
        {{ confirmText }}
      </BaseButton>
    </template>
  </AppModal>
</template>

<script setup>
import AppModal from './AppModal.vue'
import BaseButton from './BaseButton.vue'

defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '确认操作' },
  message: { type: String, default: '' },
  detail: { type: String, default: '' },
  items: { type: Array, default: () => [] },
  confirmText: { type: String, default: '确认' },
  cancelText: { type: String, default: '取消' },
  danger: { type: Boolean, default: true },
  loading: { type: Boolean, default: false },
})
defineEmits(['update:modelValue', 'confirm'])
</script>