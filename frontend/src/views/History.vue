<template>
  <div>
    <PageHeader
      title="发布历史"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="refresh">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
      </template>
    </PageHeader>

    <div class="stagger mb-4 grid grid-cols-2 gap-3 xl:grid-cols-4">
      <KpiCard label="构建总数" :value="stats.total" suffix="次" />
      <KpiCard
        label="成功"
        :value="stats.success"
        suffix="次"
        :hint="`成功率 ${stats.rate}%`"
      />
      <KpiCard
        label="失败"
        :value="stats.failed"
        suffix="次"
        :hint="stats.failed ? '需要逐条确认' : '没有失败记录'"
      />
      <KpiCard
        label="今日构建"
        :value="stats.today"
        suffix="次"
        hint="仅统计今日触发的记录"
      />
    </div>

    <DataTable
      :columns="columns"
      :rows="rows"
      :total="total"
      :page="page"
      :page-size="pageSize"
      :loading="loading"
      :error="error"
      selectable
      :selected="selected"
      sticky-key="task"
      clickable
      :row-class="(row) => (row.buildStatus === 'failed' ? 'row-failed' : '')"
      min-width="980px"
      :empty-title="dirty ? '没有匹配的发布记录' : '暂无发布记录'"
      :empty-description="dirty ? '调整任务、环境、结果或日期范围后再试' : '触发一次构建后，这里会出现对应的发布记录'"
      @update:page="setPage"
      @update:page-size="setPageSize"
      @update:selected="selected = $event"
      @row-click="openDetail"
      @retry="refresh"
    >
      <template #toolbar>
        <FilterBar :dirty="dirty" @reset="resetFilters">
          <SearchField v-model="filters.search" placeholder="搜索任务 ID" width="210px" />
          <FilterSelect
            v-model="filters.envId"
            :options="envFilterOptions"
            placeholder="全部环境"
            aria-label="按环境筛选"
          />
          <FilterSelect
            v-model="filters.status"
            :options="statusFilterOptions"
            placeholder="全部结果"
            aria-label="按结果筛选"
          />
          <input v-model="filters.from" type="date" class="filter-date" aria-label="开始日期" />
          <span class="text-[10.5px] text-ink-3">至</span>
          <input v-model="filters.to" type="date" class="filter-date" aria-label="结束日期" />
          <template #right>
            <span class="text-[10.5px] text-ink-3">
              共 <CountUp :value="total" class="text-ink-2" /> 条
            </span>
          </template>
        </FilterBar>
      </template>

      <template #bulk>
        <BaseButton size="xs" variant="secondary" @click="exportSelected">
          <template #icon><Download class="h-3 w-3" /></template>
          导出 CSV
        </BaseButton>
        <BaseButton size="xs" variant="secondary" @click="copyLinks">
          <template #icon><Link2 class="h-3 w-3" /></template>
          复制构建链接
        </BaseButton>
      </template>

      <template #cell-task="{ row }">
        <p class="mono truncate text-[11.5px] font-medium text-ink">{{ row.taskId }}</p>
        <p class="mt-0.5 truncate text-[10.5px] text-ink-3">{{ row.triggerName || '未知触发人' }}</p>
      </template>

      <template #cell-env="{ row }">
        <EnvBadge :env="envMap[row.envId]" />
      </template>

      <template #cell-ref="{ row }">
        <span class="mono text-[11px] text-ink-2">{{ row.refValue || '默认分支' }}</span>
      </template>

      <template #cell-trigger="{ row }">
        <span class="inline-flex items-center gap-1.5 text-[11px] text-ink-2">
          <component :is="triggerOf(row.triggerType).icon" class="h-3.5 w-3.5 text-ink-3" />
          {{ triggerOf(row.triggerType).label }}
        </span>
      </template>

      <template #cell-status="{ row }">
        <StatusBadge :status="row.buildStatus" />
      </template>

      <template #cell-time="{ row }">
        <span class="num text-[11px] text-ink-2" :title="formatDateTime(row.triggerTime)">
          {{ relativeTime(row.triggerTime) }}
        </span>
      </template>

      <template #cell-actions="{ row }">
        <RowActions
          :label="row.taskId"
          :items="[
            { label: '详情', onSelect: () => openDetail(row) },
            ...(row.buildUrl ? [{ label: '构建', variant: 'primary', href: row.buildUrl }] : []),
          ]"
        />
      </template>
    </DataTable>

    <DetailDrawer
      v-model="drawerOpen"
      :title="active?.taskId || ''"
      :subtitle="active ? `${envName(active.envId)} / ${active.refValue || '默认分支'}` : ''"
      width="480px"
    >
      <div v-if="active" class="space-y-4">
        <div class="flex flex-wrap items-center gap-2">
          <StatusBadge :status="active.buildStatus" />
          <span class="chip">{{ triggerOf(active.triggerType).label }}触发</span>
          <span class="text-[10.5px] text-ink-3">{{ formatDateTime(active.triggerTime) }}</span>
        </div>

        <dl class="grid grid-cols-[78px_1fr] gap-x-3 gap-y-2.5 rounded-field border border-line px-3.5 py-3 text-xs">
          <dt class="text-ink-3">任务</dt>
          <dd class="font-mono text-[11px] text-ink">{{ active.taskId }}</dd>
          <dt class="text-ink-3">环境</dt>
          <dd><EnvBadge :env="envMap[active.envId]" /></dd>
          <dt class="text-ink-3">引用</dt>
          <dd class="font-mono text-[11px] text-ink-2">{{ active.refValue || '默认分支' }}</dd>
          <dt class="text-ink-3">触发人</dt>
          <dd class="text-ink">{{ active.triggerName || '—' }}</dd>
          <dt class="text-ink-3">触发时间</dt>
          <dd class="num text-ink">{{ formatDateTime(active.triggerTime) }}</dd>
          <dt class="text-ink-3">记录 ID</dt>
          <dd class="num text-[11px] text-ink-2">#{{ active.id }}</dd>
        </dl>

        <div v-if="active.errorMessage" class="rounded-field border border-failed bg-failed-soft px-3.5 py-3">
          <p class="section-label !text-failed">失败原因</p>
          <p class="mt-1 break-words text-[11px] leading-5 text-failed">{{ shortError(active.errorMessage, 420) }}</p>
        </div>

        <div v-if="active.buildUrl" class="rounded-field border border-line px-3.5 py-3">
          <p class="section-label">构建链接</p>
          <a :href="active.buildUrl" target="_blank" rel="noopener" class="mt-1 block break-all font-mono text-[10.5px] text-primary-ink hover:underline">
            {{ active.buildUrl }}
          </a>
        </div>

        <p class="text-[10.5px] leading-5 text-ink-3">
          发布历史保留近一年，环境名称按当前配置实时换算，因此不会出现改名前的旧名称。
        </p>
      </div>

      <template #footer>
        <a
          v-if="active?.buildUrl"
          :href="active.buildUrl"
          target="_blank"
          rel="noopener"
          class="btn btn-primary btn-sm"
        >
          打开 Jenkins
          <ExternalLink class="h-3 w-3" />
        </a>
        <BaseButton size="sm" variant="secondary" @click="copyText(active?.taskId)">复制任务 ID</BaseButton>
      </template>
    </DetailDrawer>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  CalendarClock,
  Download,
  ExternalLink,
  Hand,
  Link2,
  MessageSquare,
  RefreshCw,
} from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import RowActions from '@/components/base/RowActions.vue'
import FilterSelect from '@/components/base/FilterSelect.vue'
import SearchField from '@/components/base/SearchField.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import DataTable from '@/components/data/DataTable.vue'
import DetailDrawer from '@/components/base/DetailDrawer.vue'
import StatusBadge from '@/components/data/StatusBadge.vue'
import EnvBadge from '@/components/data/EnvBadge.vue'
import KpiCard from '@/components/data/KpiCard.vue'
import CountUp from '@/components/ui/CountUp.vue'
import { historyApi } from '@/api/history'
import { envApi } from '@/api/environments'
import { useToast } from '@/composables/useToast'
import { useDebounce } from '@/composables/useDebounce'
import { downloadCsv, stamp } from '@/utils/export'
import { formatDateTime, relativeTime, shortError } from '@/utils/format'

const route = useRoute()
const toast = useToast()
const { run } = useDebounce(280)

const envs = ref([])
const rows = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const error = ref('')
const selected = ref([])
const stats = reactive({ total: 0, success: 0, failed: 0, today: 0, rate: 0 })

const filters = reactive({ search: '', envId: '', status: '', from: '', to: '' })
const ready = ref(false)

const envMap = computed(() => Object.fromEntries(envs.value.map((e) => [e.id, e])))
const envFilterOptions = computed(() => [
  { label: '全部环境', value: '' },
  ...envs.value.map((e) => ({ label: e.name, value: e.id })),
])
const statusFilterOptions = [
  { label: '全部结果', value: '' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
]
const envName = (id) => envMap.value[id]?.name || '未配置环境'
const dirty = computed(() => !!(filters.search || filters.envId || filters.status || filters.from || filters.to))

const columns = [
  { key: 'task', title: '任务', width: '22%' },
  { key: 'env', title: '环境', width: '16%' },
  { key: 'ref', title: '分支 / Tag', width: '14%' },
  { key: 'trigger', title: '触发', width: '10%' },
  { key: 'status', title: '结果', width: '9%' },
  { key: 'time', title: '触发时间', width: '14%' },
  { key: 'actions', title: '', align: 'right', width: '140px' },
]

const TRIGGER = {
  manual: { label: '手动', icon: Hand },
  scheduled: { label: '定时', icon: CalendarClock },
  dingtalk: { label: '钉钉', icon: MessageSquare },
}
const triggerOf = (t) => TRIGGER[t] || { label: t || '手动', icon: Hand }

function ymd(d) {
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

async function load() {
  loading.value = true
  try {
    const res = await historyApi.list({
      page: page.value,
      size: pageSize.value,
      search: filters.search,
      envId: filters.envId,
      status: filters.status,
      from: filters.from,
      to: filters.to,
    })
    rows.value = res?.data || []
    total.value = res?.total || 0
    error.value = ''
  } catch (e) {
    rows.value = []
    total.value = 0
    error.value = e.message || '发布历史加载失败'
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    Object.assign(stats, await historyApi.stats())
    stats.rate = Math.round((stats.rate || 0) * 100)
  } catch {
    /* 统计失败不影响列表 */
  }
}

function refresh() {
  load()
  loadStats()
}

function scheduleReload() {
  run(() => load())
}

watch(filters, () => {
  if (!ready.value) return
  page.value = 1
  selected.value = []
  scheduleReload()
})

function setPage(p) {
  page.value = p
  selected.value = []
  load()
}
function setPageSize(size) {
  pageSize.value = size
  page.value = 1
  selected.value = []
  load()
}
function resetFilters() {
  Object.assign(filters, { search: '', envId: '', status: '', from: '', to: '' })
}

const drawerOpen = ref(false)
const active = ref(null)
function openDetail(row) {
  active.value = row
  drawerOpen.value = true
}

function exportSelected() {
  const set = new Set(selected.value)
  const list = rows.value.filter((r) => set.has(r.id))
  if (!list.length) return
  downloadCsv(
    `发布历史-${stamp()}.csv`,
    ['任务', '环境', '分支/Tag', '触发方式', '触发人', '结果', '触发时间', '构建链接'],
    list.map((r) => [
      r.taskId,
      envName(r.envId),
      r.refValue || '默认分支',
      triggerOf(r.triggerType).label,
      r.triggerName || '',
      r.buildStatus,
      formatDateTime(r.triggerTime),
      r.buildUrl || '',
    ])
  )
  toast.success(`已导出 ${list.length} 条`)
}

function copyLinks() {
  const set = new Set(selected.value)
  const links = rows.value.filter((r) => set.has(r.id) && r.buildUrl).map((r) => r.buildUrl)
  if (!links.length) {
    toast.info('选中的记录没有构建链接')
    return
  }
  navigator.clipboard?.writeText(links.join('\n')).then(
    () => toast.success(`已复制 ${links.length} 个链接`),
    () => toast.error('复制失败，请手动选择')
  )
}

function copyText(text) {
  if (!text) return
  navigator.clipboard?.writeText(String(text)).then(
    () => toast.success('已复制'),
    () => toast.error('复制失败，请手动选择')
  )
}

function consumeQuery() {
  if (route.query.q != null) filters.search = String(route.query.q)
  if (route.query.status != null) filters.status = String(route.query.status)
  if (route.query.env != null) filters.envId = String(route.query.env)
  if (route.query.from != null) filters.from = String(route.query.from)
  if (route.query.to != null) filters.to = String(route.query.to)
}

watch(
  () => route.query,
  () => {
    consumeQuery()
    page.value = 1
    load()
  },
  { deep: true }
)

onMounted(async () => {
  consumeQuery()
  const [e] = await Promise.allSettled([envApi.list(), load(), loadStats()])
  if (e.status === 'fulfilled') envs.value = e.value || []
  ready.value = true
})
</script>

<style scoped>
:deep(tr.row-failed) {
  background: var(--failed-soft);
}
:deep(tr.row-failed td:first-child) {
  box-shadow: inset 2px 0 0 var(--failed);
}
:deep(tr.row-failed:hover) {
  background: color-mix(in srgb, var(--failed-soft) 70%, var(--surface-hover));
}
@media (max-width: 1023px) {
  :deep(tr.row-failed .sticky-col) {
    background: var(--failed-soft);
  }
}
</style>