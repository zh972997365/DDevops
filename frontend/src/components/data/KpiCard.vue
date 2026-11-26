<template>
  <article class="card kpi-tilt flex h-full min-h-[104px] flex-col card-pad">
    <p class="text-[11.5px] leading-4 text-ink-3">{{ label }}</p>

    <p class="mt-2 flex items-baseline gap-1">
      <CountUp :value="value" :decimals="decimals" class="num text-[28px] font-semibold leading-8 tracking-[-0.025em] text-ink" />
      <span v-if="suffix" class="text-[11.5px] font-medium text-ink-3">{{ suffix }}</span>
    </p>

    <p v-if="hint" class="mt-auto pt-1 text-[11.5px] leading-5 text-ink-3">{{ hint }}</p>
    <slot />
  </article>
</template>

<script setup>
import CountUp from '@/components/ui/CountUp.vue'

defineProps({
  label: { type: String, required: true },
  value: { type: Number, default: 0 },
  decimals: { type: Number, default: 0 },
  suffix: { type: String, default: '' },
  hint: { type: String, default: '' },
})
</script>
