<template>
  <AppModal
    :model-value="modelValue"
    title="触发构建"
    :subtitle="job ? `任务 ${job.id}` : ''"
    width="470px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="space-y-4">
      <dl class="grid grid-cols-[64px_1fr] gap-x-3 gap-y-2.5 rounded-field border border-line bg-sunken px-3.5 py-3 text-xs">
        <dt class="text-ink-3">任务</dt>
        <dd class="truncate font-mono text-ink">{{ job?.id || '—' }}</dd>
        <dt class="text-ink-3">环境</dt>
        <dd class="flex items-center gap-2">
          <span class="text-ink">{{ env?.name || '未配置' }}</span>
          <span v-if="env?.isProd" class="badge badge-warn">生产</span>
        </dd>
        <dt class="text-ink-3">方式</dt>
        <dd class="text-ink">{{ isProd ? 'Tag 构建' : '分支构建' }}</dd>
        <dt class="text-ink-3">参数名</dt>
        <dd class="font-mono text-ink-2">{{ paramName }}</dd>
      </dl>

      <template v-if="!isProd">
        <BaseInput
          v-model="refValue"
          label="分支 / 版本"
          mono
          :placeholder="env?.defaultRef || 'origin/test'"
          :hint="`留空使用环境默认分支 ${env?.defaultRef || 'origin/test'}`"
          :error="error"
        />
      </template>

      <template v-else>
        <div class="rounded-field border border-warn bg-warn-soft px-3 py-2 text-[11px] leading-5 text-warn">
          生产环境必须显式选择版本 Tag，缺失时不会回落到默认分支。
        </div>

        <div>
          <label class="field-label" for="prod-tag">
            版本 Tag
            <span class="text-failed" aria-hidden="true">*</span>
          </label>
          <div class="relative">
            <select
              id="prod-tag"
              v-model="refValue"
              class="input cursor-pointer appearance-none pr-9"
              :class="error && 'is-error'"
              :disabled="tagsLoading || !tagsAvailable || !tags.length"
            >
              <option value="" disabled>{{ tagsLoading ? '正在获取 Tag…' : '请选择版本 Tag' }}</option>
              <option v-for="t in tags" :key="t" :value="t">{{ t }}</option>
            </select>
            <ChevronDown class="select-chevron" />
          </div>
          <p v-if="error" class="field-error">{{ error }}</p>
          <p v-else-if="tagsLoading" class="field-hint">正在从 Jenkins 读取 Git Parameter…</p>
          <p v-else-if="!tagsAvailable || !tags.length" class="field-hint">
            未能获取可用 Tag。
            <button type="button" class="text-primary-ink underline underline-offset-2" @click="fetchTags">重试</button>
          </p>
          <p v-else class="field-hint">仅展示最近 10 个 Tag，最新在最上方；该请求只读，不会触发构建。</p>
        </div>
      </template>
    </div>

    <template #footer>
      <BaseButton variant="secondary" @click="$emit('update:modelValue', false)">取消</BaseButton>
      <BaseButton :loading="loading" :disabled="isProd && !refValue" @click="submit">执行构建</BaseButton>
    </template>
  </AppModal>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import AppModal from '@/components/base/AppModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { jobApi } from '@/api/jobs'
import { required, tagFormat } from '@/utils/validate'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  job: { type: Object, default: null },
  env: { type: Object, default: null },
  loading: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'submit'])

const refValue = ref('')
const error = ref('')
const tags = ref([])
const tagsAvailable = ref(false)
const tagsLoading = ref(false)

const isProd = computed(() => !!props.env?.isProd)
const paramName = computed(() =>
  isProd.value
    ? props.env?.defaultTagParam || 'Tags'
    : props.env?.defaultBranchParam || 'branch'
)

async function fetchTags() {
  tags.value = []
  tagsAvailable.value = false
  if (!isProd.value || !props.job?.id) return
  tagsLoading.value = true
  try {
    const res = await jobApi.gitTags(props.job.id)
    tags.value = [...(res.tags || [])].sort().reverse().slice(0, 10)
    tagsAvailable.value = !!res.available && tags.value.length > 0
  } catch {
    tags.value = []
    tagsAvailable.value = false
  } finally {
    tagsLoading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    refValue.value = isProd.value ? '' : props.env?.defaultRef || ''
    error.value = ''
    fetchTags()
  }
)

function submit() {
  if (isProd.value) {
    const msg = required(refValue.value, '版本 Tag') || tagFormat(refValue.value)
    if (msg) {
      error.value = msg
      return
    }
  }
  emit('submit', refValue.value.trim())
}
</script>