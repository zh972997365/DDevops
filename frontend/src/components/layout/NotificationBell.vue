<template>
  <div ref="root" class="relative">
    <button
      type="button"
      class="top-icon relative"
      :aria-expanded="open"
      aria-haspopup="dialog"
      aria-label="通知"
      title="通知"
      @click="toggle"
    >
      <Bell class="h-4 w-4" />
      <span
        v-if="count > 0"
        class="absolute right-1 top-1 grid min-w-[14px] place-items-center rounded-full bg-failed px-1 text-[9px] font-semibold leading-[14px] text-white ring-2 ring-topbar"
      >
        {{ count > 9 ? '9+' : count }}
      </span>
    </button>

    <Transition name="pop">
      <div
        v-if="open"
        class="absolute right-0 top-[calc(100%+8px)] z-50 w-[360px] overflow-hidden rounded-card border border-line bg-solid shadow-pop max-sm:fixed max-sm:left-3 max-sm:right-3 max-sm:top-[60px] max-sm:w-auto"
        role="dialog"
        aria-label="通知面板"
      >
        <div class="flex items-center justify-between border-b border-line px-4 py-3">
          <div>
            <p class="text-[13px] font-semibold text-ink">通知</p>
            <p class="mt-0.5 text-[11px] text-ink-3">今日失败与待执行计划</p>
          </div>
          <button class="icon-btn h-7 w-7" type="button" aria-label="刷新通知" title="刷新" @click="load">
            <RefreshCw class="h-3.5 w-3.5" :class="loading && 'animate-spin'" />
          </button>
        </div>

        <div class="max-h-[400px] overflow-y-auto">
          <div v-if="loading" class="space-y-3 px-4 py-4">
            <span v-for="i in 3" :key="i" class="skeleton block h-9 w-full" />
          </div>

          <div v-else-if="error" class="px-4 py-6 text-center">
            <p class="text-xs text-ink-3">{{ error }}</p>
            <button class="btn btn-secondary btn-xs mt-3" type="button" @click="load">重试</button>
          </div>

          <template v-else>
            <section v-if="failures.length" class="py-1">
              <p class="section-label px-4 pb-1 pt-3">失败构建 / 今日</p>
              <button
                v-for="row in failures"
                :key="row.id"
                type="button"
                class="flex w-full items-start gap-2.5 px-4 py-2 text-left transition-colors hover:bg-sunken"
                @click="goHistory(row)"
              >
                <span class="mt-[7px] h-1.5 w-1.5 shrink-0 rounded-full bg-failed" />
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-xs font-medium text-ink">{{ row.taskId }}</span>
                  <span class="mt-0.5 block truncate text-[11px] text-ink-3">
                    {{ row.refValue || '默认分支' }} / {{ shortError(row.errorMessage, 60) || '构建未通过' }}
                  </span>
                </span>
                <span class="num mt-0.5 shrink-0 text-[10.5px] text-ink-3">{{ timeOf(row.triggerTime) }}</span>
              </button>
            </section>

            <section v-if="pending.length" class="py-1" :class="failures.length && 'border-t border-line'">
              <p class="section-label px-4 pb-1 pt-3">待执行定时</p>
              <button
                v-for="row in pending.slice(0, 4)"
                :key="row.id"
                type="button"
                class="flex w-full items-start gap-2.5 px-4 py-2 text-left transition-colors hover:bg-sunken"
                @click="goTask(row)"
              >
                <span class="mt-[7px] h-1.5 w-1.5 shrink-0 rounded-full bg-warn" />
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-xs font-medium text-ink">{{ row.taskId }}</span>
                  <span class="mt-0.5 block truncate text-[11px] text-ink-3">
                    {{ row.refValue || '默认分支' }}
                  </span>
                </span>
                <span class="num mt-0.5 shrink-0 text-[10.5px] text-ink-3">{{ timeOf(row.scheduleTime) }}</span>
              </button>
            </section>

            <div v-if="!failures.length && !pending.length" class="px-4 py-10 text-center">
              <span class="mx-auto grid h-9 w-9 place-items-center rounded-full bg-success-soft text-success">
                <CircleCheck class="h-4 w-4" />
              </span>
              <p class="mt-2 text-xs text-ink-2">暂无需要处理的通知</p>
              <p class="mt-0.5 text-[11px] text-ink-3">今日构建无失败，队列为空</p>
            </div>
          </template>
        </div>

        <div class="flex items-center justify-between border-t border-line bg-sunken px-4 py-2.5">
          <button class="link-inline" type="button" @click="goHistory()">发布历史</button>
          <button class="link-inline" type="button" @click="goTasks">定时任务</button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, CircleCheck, RefreshCw } from 'lucide-vue-next'
import { historyApi } from '@/api/history'
import { scheduleApi } from '@/api/schedules'
import { formatDateTime, shortError } from '@/utils/format'

const router = useRouter()
const root = ref(null)
const open = ref(false)
const loading = ref(false)
const error = ref('')
const failures = ref([])
const failureTotal = ref(0)
const pending = ref([])

const count = computed(() => failureTotal.value + pending.value.length)

function today() {
  const d = new Date()
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
function timeOf(v) {
  return formatDateTime(v).slice(5)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [history, tasks] = await Promise.all([
      historyApi.list({ status: 'failed', from: today(), page: 1, size: 5 }),
      scheduleApi.list('pending'),
    ])
    failures.value = history?.data || []
    failureTotal.value = history?.total || 0
    pending.value = (tasks || [])
      .filter((t) => t.status === 'pending')
      .sort((a, b) => String(a.scheduleTime).localeCompare(String(b.scheduleTime)))
  } catch (e) {
    error.value = e.message || '通知加载失败'
  } finally {
    loading.value = false
  }
}

function toggle() {
  open.value = !open.value
  if (open.value) load()
}

function goHistory(row) {
  open.value = false
  router.push({ path: '/history', query: row ? { status: 'failed', q: row.taskId } : { status: 'failed' } })
}
function goTask(row) {
  open.value = false
  router.push({ path: '/tasks', query: row ? { q: row.taskId } : {} })
}
function goTasks() {
  open.value = false
  router.push('/tasks')
}
function onDoc(e) {
  if (open.value && root.value && !root.value.contains(e.target)) open.value = false
}
function onKey(e) {
  if (e.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('mousedown', onDoc)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDoc)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.top-icon {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 7px;
  color: var(--sidebar-ink-2);
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}
.top-icon:hover {
  background: var(--sidebar-hover-bg);
  color: var(--sidebar-ink);
}
.pop-enter-active,
.pop-leave-active {
  transition: opacity 140ms var(--ease), transform 140ms var(--ease);
  transform-origin: top right;
}
.pop-enter-from,
.pop-leave-to {
  transform: translateY(-4px) scale(0.985);
}
</style>