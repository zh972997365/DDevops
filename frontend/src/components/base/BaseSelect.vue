<template>
  <div class="min-w-0">
    <label v-if="label" class="field-label" :for="id">
      {{ label }}
      <span v-if="required" class="text-failed" aria-hidden="true">*</span>
    </label>

    <div class="relative">
      <select
        :id="id"
        v-model="model"
        class="input cursor-pointer appearance-none pr-9"
        :class="[error && 'is-error', mono && 'font-mono text-xs']"
        :disabled="disabled"
        :aria-invalid="error ? 'true' : undefined"
      >
        <option value="" disabled>{{ placeholder }}</option>
        <template v-if="groups?.length">
          <optgroup v-for="g in groups" :key="g.label" :label="g.label">
            <option v-for="o in g.options" :key="o.value" :value="o.value" :disabled="o.disabled">
              {{ o.label }}
            </option>
          </optgroup>
        </template>
        <template v-else>
          <option v-for="o in options" :key="o.value" :value="o.value" :disabled="o.disabled">
            {{ o.label }}
          </option>
        </template>
      </select>
      <ChevronDown class="select-chevron" />
    </div>

    <p v-if="error" class="field-error">{{ error }}</p>
    <p v-else-if="hint" class="field-hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { useId } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

defineProps({
  label: { type: String, default: '' },
  hint: { type: String, default: '' },
  error: { type: String, default: '' },
  placeholder: { type: String, default: '请选择' },
  options: { type: Array, default: () => [] },
  groups: { type: Array, default: () => [] },
  disabled: { type: Boolean, default: false },
  required: { type: Boolean, default: false },
  mono: { type: Boolean, default: false },
})

const model = defineModel({ type: [String, Number], default: '' })
const id = useId()
</script>