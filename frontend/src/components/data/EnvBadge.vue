<template>
  <span v-if="!env" class="text-xs text-ink-3">未配置环境</span>
  <span v-else class="badge" :class="TONE_BADGE[envTone(env)]" :title="mode">{{ env.name }}</span>
</template>

<script setup>
import { computed } from 'vue'
import { envTone } from '@/constants/env'
import { TONE_BADGE } from '@/constants/status'

const props = defineProps({
  env: { type: Object, default: null },
})
const mode = computed(() => (props.env?.isProd ? 'Tag 构建 / 生产环境' : '分支构建'))
</script>