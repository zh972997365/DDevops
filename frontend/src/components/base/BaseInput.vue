<template>
  <div class="min-w-0">
    <label v-if="label" class="field-label" :for="id">
      {{ label }}
      <span v-if="required" class="text-failed" aria-hidden="true">*</span>
    </label>

    <div class="relative">
      <span v-if="$slots.prefix" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-3">
        <slot name="prefix" />
      </span>

      <input
        :id="id"
        v-model="model"
        :type="reveal ? 'text' : type"
        :placeholder="placeholder"
        :disabled="disabled"
        :autocomplete="autocomplete"
        :maxlength="maxlength"
        :inputmode="inputmode"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="hint || error ? `${id}-desc` : undefined"
        class="input"
        :class="[
          mono && 'font-mono text-xs',
          $slots.prefix && 'pl-9',
          type === 'password' && 'pr-10',
          error && 'is-error',
        ]"
      />

      <button
        v-if="type === 'password'"
        type="button"
        class="absolute right-1 top-1/2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-[6px] text-ink-3 transition-colors hover:bg-sunken hover:text-ink"
        :aria-label="reveal ? '隐藏密码' : '显示密码'"
        @click="reveal = !reveal"
      >
        <EyeOff v-if="reveal" class="h-3.5 w-3.5" />
        <Eye v-else class="h-3.5 w-3.5" />
      </button>

      <span v-else-if="$slots.suffix" class="absolute right-3 top-1/2 -translate-y-1/2 text-ink-3">
        <slot name="suffix" />
      </span>
    </div>

    <p v-if="error" :id="`${id}-desc`" class="field-error">{{ error }}</p>
    <p v-else-if="hint" :id="`${id}-desc`" class="field-hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { useId, ref } from 'vue'
import { Eye, EyeOff } from 'lucide-vue-next'

defineProps({
  label: { type: String, default: '' },
  hint: { type: String, default: '' },
  error: { type: String, default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' },
  mono: { type: Boolean, default: false },
  required: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  autocomplete: { type: String, default: 'off' },
  maxlength: { type: [String, Number], default: undefined },
  inputmode: { type: String, default: undefined },
})

const model = defineModel({ type: [String, Number], default: '' })
const id = useId()
const reveal = ref(false)
</script>