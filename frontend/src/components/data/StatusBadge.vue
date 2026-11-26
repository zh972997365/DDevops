<template>
  <span class="badge" :class="TONE_BADGE[tone]">{{ label }}</span>
</template>

<script setup>
import { computed } from 'vue'
import { statusOf, TONE_BADGE } from '@/constants/status'

const props = defineProps({
  status: { type: String, default: '' },
})

const info = computed(() => statusOf(props.status))
const label = computed(() => info.value.label)
const tone = computed(() => info.value.tone)
</script>