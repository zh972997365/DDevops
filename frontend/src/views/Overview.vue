<template>
  <div>
    <PageHeader
      title="概括总览"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="loadAll">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
        <BaseButton v-if="auth.isAdmin" @click="router.push('/jobs/new')">
          <template #icon><Plus class="h-3.5 w-3.5" /></template>
          新建任务
        </BaseButton>
      </template>
    </PageHeader>

    <div v-if="fatalError" class="card card-pad">
      <ErrorState :message="fatalError" @retry="loadAll" />
    </div>

    <template v-else>
      <!-- 结论先行：一句话概括今天，不堆指标 -->
      <div class="card mb-4 flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2.5">
        <div class="flex min-w-0 items-center gap-2">
          <p class="text-xs leading-5 text-ink-2">
            <template v-if="loading && !loaded">正在汇总今日数据…</template>
            <template v-else>{{ summary }}</template>
          </p>
        </div>
        <div class="ml-auto flex flex-wrap items-center gap-x-3 gap-y-1 text-[10.5px] text-ink-3">
          <span><span class="num text-ink-2">{{ totalJobs }}</span> 条流水线</span>
          <span class="text-line-strong">/</span>
          <span><span class="num text-ink-2">{{ envs.length }}</span> 个环境</span>
          <span class="text-line-strong">/</span>
          <span>累计 <span class="num text-ink-2">{{ stats.total }}</span> 次发布</span>
          <span v-if="updatedAt" class="hidden sm:inline">/ {{ updatedAt }} 更新</span>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-12">
        <!-- KPI -->
        <div class="stagger grid grid-cols-2 gap-3 xl:col-span-12 xl:grid-cols-4">
          <template v-if="loading && !loaded">
            <div v-for="i in 4" :key="i" class="card card-pad">
              <span class="skeleton h-3 w-16" />
              <span class="skeleton mt-4 block h-7 w-24" />
              <span class="skeleton mt-3 block h-3 w-32" />
            </div>
          </template>
          <template v-else>
            <KpiCard
              label="今日发布"
              :value="todayCount"
              suffix="次"
              :hint="`成功 ${todaySuccess} / 失败 ${todayFailed}`"
            />
            <KpiCard
              label="近 14 日成功率"
              :value="windowRate"
              suffix="%"
              :hint="`${windowSuccess} 成功 / ${windowTotal} 次`"
            />
            <KpiCard
              label="运行中"
              :value="running"
              suffix="个"
              hint="调度器正在执行"
            />
            <KpiCard
              label="待执行"
              :value="pendingList.length"
              suffix="个"
              :hint="pendingList.length ? `下一班 ${formatDateTime(pendingList[0].scheduleTime)}` : '队列为空'"
            />
          </template>
        </div>

        <!-- 趋势 -->
        <section class="card overflow-hidden xl:col-span-8">
          <header class="flex flex-wrap items-start justify-between gap-3 border-b border-line px-5 py-4">
            <div>
              <h2 class="panel-title">发布趋势</h2>
              <p class="mt-0.5 text-[11px] text-ink-3">近 14 日 / 按触发日聚合</p>
            </div>
            <div class="flex items-center gap-3 text-[10.5px] text-ink-3">
              <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-[2px] bg-[var(--chart-bar)]" />成功</span>
              <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-[2px] bg-[var(--chart-bar-alt)]" />失败</span>
              <span v-if="peakDay" class="hidden sm:inline">峰值 {{ peakDay.label }} / {{ peakDay.total }} 次</span>
            </div>
          </header>

          <div class="px-5 pb-4 pt-5">
            <div v-if="loading && !loaded" class="flex h-[190px] items-end gap-2">
              <span v-for="i in 14" :key="i" class="skeleton flex-1" :style="{ height: `${20 + ((i * 13) % 60)}%` }" />
            </div>

            <template v-else>
              <div class="flex gap-2.5">
                <div class="flex h-[190px] w-6 shrink-0 flex-col justify-between pb-0 text-right font-mono text-[10px] text-ink-3">
                  <span>{{ maxTrend }}</span>
                  <span>{{ Math.round(maxTrend / 2) }}</span>
                  <span>0</span>
                </div>

                <div class="min-w-0 flex-1">
                  <div class="relative h-[190px]">
                    <div class="pointer-events-none absolute inset-0 flex flex-col justify-between" aria-hidden="true">
                      <span v-for="i in 3" :key="i" class="h-px w-full bg-[var(--chart-grid)]" />
                    </div>

                    <div v-if="!windowTotal" class="absolute inset-0 z-[1] grid place-items-center">
                      <p class="rounded-field border border-line bg-solid px-3 py-2 text-[11px] text-ink-3">
                        近 14 日没有发布记录
                      </p>
                    </div>

                    <div class="relative flex h-full items-end gap-[5px]">
                      <div
                        v-for="d in trend"
                        :key="d.key"
                        class="group relative flex h-full flex-1 flex-col justify-end rounded-[3px] transition-colors hover:bg-surface-hover"
                      >
                        <div
                          class="relative w-full overflow-hidden rounded-[2px]"
                          :style="{ height: d.total ? `max(3px, ${(d.total / maxTrend) * 100}%)` : '0' }"
                        >
                          <span
                            class="absolute bottom-0 left-0 w-full transition-opacity group-hover:opacity-85"
                            :class="d.isToday ? 'bg-[var(--accent)]' : 'bg-[var(--chart-bar)]'"
                            :style="{ height: `${d.successPct}%` }"
                          />
                          <span
                            v-if="d.failed"
                            class="absolute left-0 top-0 w-full rounded-t-[2px] bg-[var(--chart-bar-alt)]"
                            :style="{ height: `${d.failedPct}%` }"
                          />
                        </div>
                        <div class="pointer-events-none absolute bottom-[calc(100%+6px)] left-1/2 z-10 hidden w-max -translate-x-1/2 rounded-[6px] border border-line bg-solid px-2 py-1.5 text-left shadow-pop group-hover:block">
                          <p class="num text-[10px] font-medium text-ink">{{ d.label }}{{ d.isToday ? ' / 今天' : '' }}</p>
                          <p class="num mt-0.5 text-[10px] text-ink-3">
                            共 {{ d.total }} / 成功 {{ d.success }} / 失败 {{ d.failed }}
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="mt-2 flex gap-[5px]">
                    <span
                      v-for="d in trend"
                      :key="`x-${d.key}`"
                      class="flex-1 text-center font-mono text-[10px]"
                      :class="d.isToday ? 'font-semibold text-ink' : 'text-ink-3'"
                    >
                      {{ d.isToday ? '今天' : d.tick }}
                    </span>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </section>

        <!-- 定时队列 + 环境 -->
        <div class="flex flex-col gap-4 xl:col-span-4">
          <section class="card flex flex-1 flex-col overflow-hidden">
            <header class="flex items-center justify-between border-b border-line px-4 py-3">
              <div class="flex items-center gap-2">
                <h2 class="panel-title">定时队列</h2>
                <span class="badge badge-neutral">{{ pendingList.length }}</span>
              </div>
              <RouterLink to="/tasks" class="link-inline">全部</RouterLink>
            </header>
            <div v-if="pendingList.length" class="flex-1 divide-y divide-line-soft">
              <button
                v-for="t in pendingList.slice(0, 4)"
                :key="t.id"
                type="button"
                class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-sunken"
                @click="router.push({ path: '/tasks', query: { q: t.taskId } })"
              >
                <span class="num w-[52px] shrink-0 font-mono text-[10.5px] text-ink-3">
                  {{ formatDateTime(t.scheduleTime).slice(5) }}
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-mono text-[11.5px] text-ink">{{ t.taskId }}</span>
                  <span class="mt-0.5 block truncate text-[10.5px] text-ink-3">
                    {{ t.refValue || '默认分支' }} / {{ t.createdName || '定时' }}
                  </span>
                </span>
                <span class="badge" :class="remain(t.scheduleTime).urgent ? 'badge-failed' : 'badge-neutral'">
                  {{ remain(t.scheduleTime).text }}
                </span>
              </button>
            </div>
            <div v-else class="flex flex-1 flex-col items-center justify-center gap-3 px-4 py-8 text-center">
              <div>
                <p class="text-xs text-ink-2">队列为空</p>
                <p class="mt-1 text-[10.5px] text-ink-3">接下来没有自动发布</p>
              </div>
              <RouterLink to="/tasks?new=1" class="btn btn-secondary btn-xs">新建定时</RouterLink>
            </div>
          </section>

          <section class="card overflow-hidden">
            <header class="flex items-center justify-between border-b border-line px-4 py-3">
              <h2 class="panel-title">环境矩阵</h2>
              <RouterLink v-if="auth.isAdmin" to="/environments" class="link-inline">
                管理
              </RouterLink>
            </header>
            <div v-if="envs.length" class="divide-y divide-line-soft">
              <button
                v-for="e in envs.slice(0, 5)"
                :key="e.id"
                type="button"
                class="flex w-full items-center gap-2.5 px-4 py-2 text-left transition-colors hover:bg-sunken"
                @click="router.push({ path: '/jobs', query: { env: e.id } })"
              >
                <span class="min-w-0 flex-1 truncate text-xs text-ink">{{ e.name }}</span>
                <span class="text-[10px] text-ink-3">{{ e.isProd ? '标签' : '分支' }}</span>
                <span class="num w-10 text-right text-[10.5px] text-ink-3">{{ e.jobCount ?? 0 }}</span>
              </button>
            </div>
            <p v-else class="px-4 py-6 text-center text-[11px] text-ink-3">还没有环境配置</p>
          </section>
        </div>

        <!-- 最近发布 -->
        <section class="card overflow-hidden xl:col-span-8">
          <header class="flex items-center justify-between border-b border-line px-5 py-3.5">
            <div>
              <h2 class="panel-title">最近发布</h2>
              <p class="mt-0.5 text-[11px] text-ink-3">按触发时间倒序，点击行可查看详情</p>
            </div>
            <RouterLink to="/history" class="link-inline">全部历史</RouterLink>
          </header>

          <div v-if="loading && !loaded" class="space-y-2 p-4">
            <span v-for="i in 5" :key="i" class="skeleton block h-8 w-full" />
          </div>
          <div v-else-if="recent.length">
            <div class="divide-y divide-line-soft sm:hidden">
              <RouterLink
                v-for="r in recent.slice(0, 6)"
                :key="`m-${r.id}`"
                :to="{ path: '/history', query: { q: r.taskId } }"
                class="flex items-center gap-3 px-4 py-2.5"
              >
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-mono text-[11.5px] font-medium text-ink">{{ r.taskId }}</span>
                  <span class="mt-0.5 block truncate text-[10.5px] text-ink-3">
                    {{ envName(r.envId) }} / {{ r.refValue || '默认分支' }}
                  </span>
                </span>
                <StatusBadge :status="r.buildStatus" />
                <span class="shrink-0 text-[10.5px] text-ink-3">{{ relativeTime(r.triggerTime) }}</span>
              </RouterLink>
            </div>

            <div class="hidden overflow-x-auto sm:block">
              <table class="data-table min-w-[560px]">
                <thead>
                  <tr>
                    <th>任务</th>
                    <th>环境</th>
                    <th>引用</th>
                    <th>结果</th>
                    <th class="text-right">触发时间</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="r in recent.slice(0, 6)"
                    :key="r.id"
                    class="cursor-pointer"
                    @click="router.push({ path: '/history', query: { q: r.taskId } })"
                  >
                    <td>
                      <span class="mono text-[11.5px] font-medium text-ink">{{ r.taskId }}</span>
                    </td>
                    <td><EnvBadge :env="envMap[r.envId]" /></td>
                    <td><span class="mono text-[11px] text-ink-2">{{ r.refValue || '默认分支' }}</span></td>
                    <td><StatusBadge :status="r.buildStatus" /></td>
                    <td class="text-right">
                      <span class="text-[11px] text-ink-2" :title="formatDateTime(r.triggerTime)">
                        {{ relativeTime(r.triggerTime) }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <EmptyState
            v-else
            compact
            title="暂无发布记录"
            description="触发一次构建后，这里会显示最近的发布结果"
          />
        </section>

        <!-- 需要接手 -->
        <section class="card flex flex-col overflow-hidden xl:col-span-4">
          <header class="flex items-center justify-between border-b border-line px-4 py-3">
            <div class="flex items-center gap-2">
              <h2 class="panel-title">需要接手</h2>
              <span class="badge" :class="failedWindow.length ? 'badge-failed' : 'badge-success'">
                {{ failedWindow.length }}
              </span>
            </div>
            <RouterLink to="/history?status=failed" class="link-inline">查看</RouterLink>
          </header>

          <div v-if="failedWindow.length" class="flex-1 divide-y divide-line-soft">
            <a
              v-for="r in failedWindow.slice(0, 4)"
              :key="r.id"
              :href="r.buildUrl || undefined"
              :target="r.buildUrl ? '_blank' : undefined"
              rel="noopener"
              class="flex items-start gap-2.5 px-4 py-2.5 transition-colors hover:bg-sunken"
            >
              <span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-failed" />
              <span class="min-w-0 flex-1">
                <span class="flex items-baseline gap-2">
                  <span class="truncate font-mono text-[11.5px] font-medium text-ink">{{ r.taskId }}</span>
                  <span class="shrink-0 text-[10px] text-ink-3">{{ relativeTime(r.triggerTime) }}</span>
                </span>
                <span class="mt-0.5 line-clamp-1 text-[10.5px] leading-4 text-ink-3">
                  {{ shortError(r.errorMessage) || `${envName(r.envId)} / ${r.refValue || '默认分支'} 构建未通过` }}
                </span>
              </span>
              <ArrowUpRight v-if="r.buildUrl" class="mt-0.5 h-3 w-3 shrink-0 text-ink-3" />
            </a>
          </div>
          <div v-else class="flex flex-1 flex-col items-center justify-center px-4 py-8 text-center">
            <span class="grid h-8 w-8 place-items-center rounded-full bg-success-soft text-success">
              <Check class="h-4 w-4" />
            </span>
            <p class="mt-2 text-xs text-ink-2">近 14 日没有失败构建</p>
            <p class="mt-0.5 text-[10.5px] text-ink-3">保持这个状态</p>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowUpRight,
  Check,
  Plus,
  RefreshCw,
} from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import KpiCard from '@/components/data/KpiCard.vue'
import StatusBadge from '@/components/data/StatusBadge.vue'
import EnvBadge from '@/components/data/EnvBadge.vue'
import EmptyState from '@/components/data/EmptyState.vue'
import ErrorState from '@/components/data/ErrorState.vue'
import { overviewApi } from '@/api/overview'
import { historyApi } from '@/api/history'
import { jobApi } from '@/api/jobs'
import { envApi } from '@/api/environments'
import { scheduleApi } from '@/api/schedules'
import { useAuthStore } from '@/stores/auth'
import { useCountdown } from '@/composables/useCountdown'
import { countdownText, formatDateTime, relativeTime, shortError } from '@/utils/format'

const router = useRouter()
const auth = useAuthStore()
const { now } = useCountdown()

const loading = ref(false)
const loaded = ref(false)
const fatalError = ref('')
const updatedAt = ref('')

const overview = ref(null)
const stats = ref({ total: 0, success: 0, failed: 0, today: 0, rate: 0 })
const jobs = ref([])
const envs = ref([])
const tasks = ref([])
const history = ref([])

const envMap = computed(() => Object.fromEntries(envs.value.map((e) => [e.id, e])))
const envName = (id) => envMap.value[id]?.name || '未配置环境'

const startOfDay = (d) => {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  return x
}
const today0 = startOfDay(new Date())

const ymd = (d) => {
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

const trend = computed(() => {
  const days = []
  for (let i = 13; i >= 0; i -= 1) {
    const d = new Date(today0)
    d.setDate(d.getDate() - i)
    const idx = 13 - i
    const rows = history.value.filter((r) => {
      const t = new Date(r.triggerTime)
      return t.getFullYear() === d.getFullYear() && t.getMonth() === d.getMonth() && t.getDate() === d.getDate()
    })
    const success = rows.filter((r) => r.buildStatus === 'success').length
    const failed = rows.filter((r) => r.buildStatus === 'failed').length
    const total = rows.length
    const label = `${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`
    days.push({
      key: ymd(d),
      label,
      tick: idx % 3 === 0 ? label : '',
      total,
      success,
      failed,
      successPct: total ? (success / total) * 100 : 0,
      failedPct: total ? (failed / total) * 100 : 0,
      isToday: i === 0,
    })
  }
  return days
})

const maxTrend = computed(() => Math.max(1, ...trend.value.map((d) => d.total)))
const windowTotal = computed(() => trend.value.reduce((n, d) => n + d.total, 0))
const windowSuccess = computed(() => trend.value.reduce((n, d) => n + d.success, 0))
const windowRate = computed(() => (windowTotal.value ? Math.round((windowSuccess.value / windowTotal.value) * 100) : 0))
const peakDay = computed(() => {
  const hit = trend.value.reduce((a, b) => (b.total > a.total ? b : a), trend.value[0])
  return hit && hit.total > 0 ? hit : null
})

const totalJobs = computed(() => overview.value?.jobs?.total ?? jobs.value.length)
const todayCount = computed(() => overview.value?.today?.builds ?? stats.value.today ?? 0)
const todaySuccess = computed(() => overview.value?.today?.success ?? 0)
const todayFailed = computed(() => Math.max(0, todayCount.value - todaySuccess.value))
const todayRate = computed(() => (todayCount.value ? Math.round((todaySuccess.value / todayCount.value) * 100) : 0))
const running = computed(() => overview.value?.running ?? 0)

const pendingList = computed(() =>
  tasks.value
    .filter((t) => t.status === 'pending')
    .sort((a, b) => String(a.scheduleTime).localeCompare(String(b.scheduleTime)))
)

const recent = computed(() =>
  [...history.value]
    .filter((r) => r.buildStatus === 'success' || r.buildStatus === 'failed')
    .sort((a, b) => new Date(b.triggerTime) - new Date(a.triggerTime))
)

const failedWindow = computed(() =>
  recent.value.filter((r) => r.buildStatus === 'failed').slice(0, 8)
)

const summary = computed(() => {
  if (!todayCount.value) {
    return pendingList.value.length
      ? `今日还没有发布记录，队列中有 ${pendingList.value.length} 个计划等待执行。`
      : '今日还没有发布记录，定时队列为空，系统保持静默。'
  }
  const base = `今日已发布 ${todayCount.value} 次，成功率 ${todayRate.value}%`
  const tail = todayFailed.value > 0 ? `，${todayFailed.value} 次失败待确认。` : '，没有失败构建。'
  const next = pendingList.value.length ? ` 下一班 ${formatDateTime(pendingList.value[0].scheduleTime)}。` : ''
  return base + tail + next
})

function remain(target) {
  void now.value
  return countdownText(target)
}

async function loadAll() {
  loading.value = true
  const from = new Date(today0)
  from.setDate(from.getDate() - 13)

  const [o, s, j, e, t, h] = await Promise.allSettled([
    overviewApi.get(),
    historyApi.stats(),
    jobApi.list(),
    envApi.list(),
    scheduleApi.list(),
    historyApi.list({ page: 1, size: 200, from: ymd(from) }),
  ])

  if (o.status === 'fulfilled') overview.value = o.value
  if (s.status === 'fulfilled') stats.value = s.value || stats.value
  if (j.status === 'fulfilled') jobs.value = j.value || []
  if (e.status === 'fulfilled') envs.value = e.value || []
  if (t.status === 'fulfilled') tasks.value = t.value || []
  if (h.status === 'fulfilled') history.value = h.value?.data || []

  const results = [o, s, j, e, t, h]
  const failed = results.filter((r) => r.status === 'rejected')
  fatalError.value = failed.length === results.length ? failed[0].reason?.message || '数据加载失败' : ''
  loading.value = false
  loaded.value = true

  const d = new Date()
  updatedAt.value = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

onMounted(loadAll)
</script>