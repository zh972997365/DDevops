<template>
  <div ref="root" class="relative">
    <BaseInput
      v-model="keyword"
      label="从钉钉通讯录选择"
      placeholder="输入姓名或拼音搜索"
      hint="选中后自动填充用户 ID、显示名称与登录名"
    >
      <template #prefix><Users class="h-3.5 w-3.5" /></template>
    </BaseInput>

    <Transition name="pop">
      <div
        v-if="open"
        class="absolute left-0 right-0 top-[calc(100%+4px)] z-30 max-h-[220px] overflow-y-auto rounded-field border border-line bg-solid p-1 shadow-pop"
        role="listbox"
      >
        <button
          v-for="u in results"
          :key="u.userid"
          type="button"
          class="flex w-full items-center gap-2 rounded-[6px] px-2.5 py-2 text-left transition-colors hover:bg-primary-soft"
          @click="pick(u)"
        >
          <span class="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-sunken text-[10px] font-semibold text-ink-2">
            {{ (u.name || 'U').slice(0, 1) }}
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-xs font-medium text-ink">{{ u.name }}</span>
            <span class="block truncate font-mono text-[10px] text-ink-3">{{ u.pinyin || u.userid }}</span>
          </span>
        </button>
        <p v-if="!results.length" class="px-2.5 py-5 text-center text-[11px] text-ink-3">
          {{ loading ? '正在搜索…' : '没有匹配的通讯录成员' }}
        </p>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Users } from 'lucide-vue-next'
import BaseInput from '@/components/base/BaseInput.vue'
import { dingtalkApi } from '@/api/dingtalk'
import { useDebounce } from '@/composables/useDebounce'

const emit = defineEmits(['pick'])
const { run } = useDebounce(280)

const root = ref(null)
const keyword = ref('')
const results = ref([])
const open = ref(false)
const loading = ref(false)

watch(keyword, (v) => {
  const kw = v.trim()
  if (!kw) {
    results.value = []
    open.value = false
    return
  }
  run(async () => {
    loading.value = true
    try {
      results.value = await dingtalkApi.searchUsers(kw)
      open.value = true
    } catch {
      results.value = []
      open.value = true
    } finally {
      loading.value = false
    }
  })
})

function pick(user) {
  emit('pick', user)
  keyword.value = user.name
  open.value = false
  results.value = []
}

function onDoc(e) {
  if (open.value && root.value && !root.value.contains(e.target)) open.value = false
}

onMounted(() => document.addEventListener('mousedown', onDoc))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDoc))
</script>

<style scoped>
.pop-enter-active,
.pop-leave-active {
  transition: opacity 120ms var(--ease), transform 120ms var(--ease);
}
.pop-enter-from,
.pop-leave-to {
  transform: translateY(-3px);
}
</style>