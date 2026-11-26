<template>
  <div>
    <PageHeader
      title="任务管理"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="load">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
        <BaseButton v-if="canEdit" @click="router.push('/jobs/new')">
          <template #icon><Plus class="h-3.5 w-3.5" /></template>
          新建任务
        </BaseButton>
      </template>
    </PageHeader>

    <div class="stagger mb-4 grid grid-cols-2 gap-3 xl:grid-cols-4">
      <KpiCard label="任务总数" :value="stats.total" suffix="条" />
      <KpiCard
        label="今日构建"
        :value="stats.today"
        suffix="次"
        :hint="`覆盖 ${stats.total} 条流水线`"
      />
      <KpiCard
        label="今日成功率"
        :value="stats.rate"
        suffix="%"
        :hint="`成功 ${stats.success} 次`"
      />
      <KpiCard
        label="运行中"
        :value="stats.running"
        suffix="个"
        hint="调度器正在执行"
      />
    </div>

    <DataTable
      :columns="columns"
      :rows="rows"
      :total="filtered.length"
      :page="page"
      :page-size="pageSize"
      :loading="loading"
      :error="error"
      selectable
      :selected="selected"
      :sort-key="sortKey"
      :sort-dir="sortDir"
      sticky-key="name"
      clickable
      :empty-title="dirty ? '没有匹配的任务' : '还没有任务'"
      :empty-description="dirty ? '调整关键词或筛选条件后再试' : '把一条 Jenkins 流水线接入平台，绑定环境后即可触发构建'"
      @update:page="page = $event"
      @update:page-size="onPageSize"
      @update:selected="selected = $event"
      @sort-change="toggleSort"
      @row-click="openDetail"
      @retry="load"
    >
      <template #toolbar>
        <FilterBar :dirty="dirty" @reset="resetFilters">
          <SearchField v-model="filters.search" placeholder="搜索任务 ID 或路径" width="230px" />
          <FilterSelect
            v-model="filters.envId"
            :options="envFilterOptions"
            placeholder="全部环境"
            aria-label="按环境筛选"
          />
          <FilterSelect
            v-model="filters.serverId"
            :options="serverFilterOptions"
            placeholder="全部服务"
            aria-label="按服务筛选"
          />
          <template #right>
            <span class="text-[10.5px] text-ink-3">
              共 <CountUp :value="filtered.length" class="text-ink-2" /> 条
            </span>
          </template>
        </FilterBar>
      </template>

      <template #bulk>
        <BaseButton size="xs" variant="secondary" @click="copySelected">
          <template #icon><Copy class="h-3 w-3" /></template>
          复制 ID
        </BaseButton>
        <BaseButton size="xs" variant="secondary" @click="exportSelected">
          <template #icon><Download class="h-3 w-3" /></template>
          导出 CSV
        </BaseButton>
      </template>

      <template #cell-name="{ row }">
        <div class="min-w-0">
          <p class="truncate font-mono text-[11.5px] font-medium text-ink">{{ row.id }}</p>
          <p class="mt-0.5 truncate text-[10.5px] text-ink-3">{{ serverName(row.serverId) }}</p>
        </div>
      </template>

      <template #cell-env="{ row }">
        <EnvBadge :env="envMap[row.envId]" />
      </template>

      <template #cell-latest="{ row }">
        <StatusBadge v-if="latest(row.id)" :status="latest(row.id).status" />
        <span v-else class="text-[11px] text-ink-3">未构建</span>
      </template>

      <template #cell-time="{ row }">
        <span v-if="latest(row.id)" class="text-[11px] text-ink-2" :title="formatDateTime(latest(row.id).time)">
          {{ relativeTime(latest(row.id).time) }}
        </span>
        <span v-else class="text-[11px] text-ink-3">未记录</span>
      </template>

      <template #cell-actions="{ row }">
        <RowActions
          :label="row.id"
          :items="[
            { label: '构建', variant: 'primary', onSelect: () => openBuild(row) },
            { label: '详情', onSelect: () => openDetail(row) },
            ...(canEdit ? [{ label: '编辑', onSelect: () => editJob(row) }] : []),
            ...(canEdit ? [{ label: '删除', tone: 'danger', onSelect: () => askDelete(row) }] : []),
          ]"
        />
      </template>

      <template #empty-action>
        <BaseButton v-if="canEdit" size="sm" @click="router.push('/jobs/new')">新建任务</BaseButton>
      </template>
    </DataTable>

    <DetailDrawer
      v-model="drawerOpen"
      :title="activeJob?.id || ''"
      :subtitle="activeJob ? `${envName(activeJob.envId)} / ${serverName(activeJob.serverId)}` : ''"
      width="500px"
    >
      <div v-if="activeJob" class="space-y-4">
        <section class="overflow-hidden rounded-field border border-line">
          <div class="flex items-center justify-between border-b border-line px-3.5 py-2">
            <p class="section-label">任务信息</p>
            <button class="link-inline" type="button" @click="copyText(activeJob.jobPath)">
              <Copy class="h-3 w-3" />复制路径
            </button>
          </div>
          <dl class="grid grid-cols-[78px_1fr] gap-x-3 gap-y-2.5 px-3.5 py-3 text-xs">
            <dt class="text-ink-3">Jenkins 路径</dt>
            <dd class="break-all font-mono text-[11px] leading-5 text-ink">{{ activeJob.jobPath }}</dd>
            <dt class="text-ink-3">所属服务</dt>
            <dd class="text-ink">{{ serverName(activeJob.serverId) }}</dd>
            <dt class="text-ink-3">环境</dt>
            <dd><EnvBadge :env="envMap[activeJob.envId]" /></dd>
            <dt class="text-ink-3">构建方式</dt>
            <dd class="text-ink">
              {{ envMap[activeJob.envId]?.isProd ? 'Tag 构建' : '分支构建' }}
              <span class="text-ink-3">
                / 参数
                {{ envMap[activeJob.envId]?.isProd
                  ? envMap[activeJob.envId]?.defaultTagParam || 'Tags'
                  : envMap[activeJob.envId]?.defaultBranchParam || 'branch' }}
              </span>
            </dd>
            <dt class="text-ink-3">默认引用</dt>
            <dd class="font-mono text-[11px] text-ink-2">
              {{ envMap[activeJob.envId]?.defaultRef || (envMap[activeJob.envId]?.isProd ? '必须选择 Tag' : 'origin/test') }}
            </dd>
            <dt class="text-ink-3">钉钉短码</dt>
            <dd class="flex items-center gap-2">
              <span class="font-mono text-[11px] text-ink-2">{{ activeJob.shortCode || '—' }}</span>
              <button v-if="activeJob.shortCode" class="text-ink-3 hover:text-ink" type="button" aria-label="复制短码" @click="copyText(activeJob.shortCode)">
                <Copy class="h-3 w-3" />
              </button>
            </dd>
            <dt class="text-ink-3">更新时间</dt>
            <dd class="num text-[11px] text-ink-2">{{ formatDateTime(activeJob.updatedAt) }}</dd>
          </dl>
        </section>

        <section v-if="latest(activeJob.id)" class="rounded-field border border-line px-3.5 py-3">
          <div class="flex items-center justify-between">
            <p class="section-label">最近构建</p>
            <a
              v-if="latest(activeJob.id).url"
              :href="latest(activeJob.id).url"
              target="_blank"
              rel="noopener"
              class="link-inline"
            >
              打开 Jenkins <ArrowUpRight class="h-3 w-3" />
            </a>
          </div>
          <div class="mt-2 flex items-center gap-2.5">
            <StatusBadge :status="latest(activeJob.id).status" />
            <span class="text-[11px] text-ink-2">{{ formatDateTime(latest(activeJob.id).time) }}</span>
            <span class="text-[11px] text-ink-3">/ {{ relativeTime(latest(activeJob.id).time) }}</span>
          </div>
        </section>

        <section class="overflow-hidden rounded-field border border-line">
          <div class="flex items-center justify-between border-b border-line px-3.5 py-2">
            <p class="section-label">最近构建记录</p>
            <RouterLink :to="{ path: '/history', query: { q: activeJob.id } }" class="link-inline">
              全部
            </RouterLink>
          </div>
          <div v-if="historyLoading" class="space-y-2 p-3">
            <span v-for="i in 3" :key="i" class="skeleton block h-7 w-full" />
          </div>
          <div v-else-if="jobHistory.length" class="divide-y divide-line-soft">
            <div v-for="h in jobHistory" :key="h.id" class="flex items-center gap-2 px-3.5 py-2">
              <StatusBadge :status="h.buildStatus" />
              <span class="min-w-0 flex-1 truncate font-mono text-[10.5px] text-ink-2">{{ h.refValue || '默认分支' }}</span>
              <span class="num shrink-0 text-[10px] text-ink-3" :title="formatDateTime(h.triggerTime)">
                {{ relativeTime(h.triggerTime) }}
              </span>
              <a v-if="h.buildUrl" :href="h.buildUrl" target="_blank" rel="noopener" class="text-ink-3 hover:text-ink" aria-label="打开构建">
                <ArrowUpRight class="h-3 w-3" />
              </a>
            </div>
          </div>
          <p v-else class="px-3.5 py-6 text-center text-[11px] text-ink-3">还没有构建记录</p>
        </section>
      </div>

      <template #footer>
        <BaseButton size="sm" @click="openBuild(activeJob)">构建</BaseButton>
        <BaseButton v-if="canEdit" size="sm" variant="secondary" @click="editJob(activeJob)">编辑</BaseButton>
        <BaseButton v-if="canEdit" size="sm" variant="danger" @click="askDelete(activeJob)">删除</BaseButton>
      </template>
    </DetailDrawer>

    <BuildModal
      v-model="buildOpen"
      :job="buildJob"
      :env="envMap[buildJob?.envId]"
      :loading="building"
      @submit="submitBuild"
    />

    <ConfirmDialog
      v-model="deleteOpen"
      title="删除任务"
      :message="`确定删除任务「${pendingJob?.id}」吗？`"
      detail="任务配置会被移除，已产生的发布历史仍会保留。"
      confirm-text="确认删除"
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowUpRight,
  Copy,
  Download,
  Plus,
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
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import BuildModal from '@/components/feature/BuildModal.vue'
import { jobApi } from '@/api/jobs'
import { envApi } from '@/api/environments'
import { serverApi } from '@/api/servers'
import { overviewApi } from '@/api/overview'
import { historyApi } from '@/api/history'
import { usePermission } from '@/composables/usePermission'
import { useToast } from '@/composables/useToast'
import { downloadCsv, stamp } from '@/utils/export'
import { formatDateTime, relativeTime } from '@/utils/format'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const { canEdit } = usePermission()

const jobs = ref([])
const envs = ref([])
const servers = ref([])
const overview = ref(null)
const loading = ref(false)
const error = ref('')

const filters = ref({ search: '', envId: '', serverId: '' })
const page = ref(1)
const pageSize = ref(10)
const sortKey = ref('time')
const sortDir = ref('desc')
const selected = ref([])

const envMap = computed(() => Object.fromEntries(envs.value.map((e) => [e.id, e])))
const envFilterOptions = computed(() => [
  { label: '全部环境', value: '' },
  ...envs.value.map((e) => ({ label: e.name, value: e.id })),
])
const serverFilterOptions = computed(() => [
  { label: '全部服务', value: '' },
  ...servers.value.map((s) => ({ label: s.name, value: s.id })),
])
const envName = (id) => envMap.value[id]?.name || '未配置环境'
const serverName = (id) => servers.value.find((s) => s.id === id)?.name || id || 'jenkins'
const latest = (id) => overview.value?.latestBuilds?.[id] || null

const dirty = computed(() => !!(filters.value.search || filters.value.envId || filters.value.serverId))

const filtered = computed(() => {
  const f = filters.value
  const kw = (f.search || '').trim().toLowerCase()
  return jobs.value.filter((j) => {
    const hitKw = !kw || j.id.toLowerCase().includes(kw) || (j.jobPath || '').toLowerCase().includes(kw)
    const hitEnv = !f.envId || j.envId === f.envId
    const hitServer = !f.serverId || j.serverId === f.serverId
    return hitKw && hitEnv && hitServer
  })
})

const sorted = computed(() => {
  const arr = [...filtered.value]
  const dir = sortDir.value === 'asc' ? 1 : -1
  arr.sort((a, b) => {
    if (sortKey.value === 'name') return a.id.localeCompare(b.id) * dir
    if (sortKey.value === 'env') return envName(a.envId).localeCompare(envName(b.envId)) * dir
    const ta = latest(a.id)?.time || ''
    const tb = latest(b.id)?.time || ''
    if (!ta && !tb) return a.id.localeCompare(b.id)
    if (!ta) return 1
    if (!tb) return -1
    return ta.localeCompare(tb) * dir
  })
  return arr
})

const rows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return sorted.value.slice(start, start + pageSize.value)
})

const stats = computed(() => ({
  total: overview.value?.jobs?.total ?? jobs.value.length,
  today: overview.value?.today?.builds ?? 0,
  success: overview.value?.today?.success ?? 0,
  rate: Math.round((overview.value?.today?.rate ?? 0) * 100),
  running: overview.value?.running ?? 0,
}))

const columns = [
  { key: 'name', title: '任务', sortable: true, width: '32%' },
  { key: 'env', title: '环境', sortable: true, width: '18%' },
  { key: 'latest', title: '最近构建', width: '14%' },
  { key: 'time', title: '最近构建时间', sortable: true },
  { key: 'actions', title: '', align: 'right', width: '220px' },
]

async function load() {
  loading.value = true
  const [j, e, s, o] = await Promise.allSettled([
    jobApi.list(),
    envApi.list(),
    serverApi.options(),
    overviewApi.get(),
  ])
  if (j.status === 'fulfilled') {
    jobs.value = j.value || []
    error.value = ''
    if (activeJob.value) {
      const next = jobs.value.find((x) => x.id === activeJob.value.id)
      if (next) activeJob.value = next
      else drawerOpen.value = false
    }
  } else {
    error.value = j.reason?.message || '任务加载失败'
  }
  if (e.status === 'fulfilled') envs.value = e.value || []
  if (s.status === 'fulfilled') servers.value = s.value || []
  if (o.status === 'fulfilled') overview.value = o.value
  loading.value = false
}

function resetFilters() {
  filters.value = { search: '', envId: '', serverId: '' }
  page.value = 1
}

function onPageSize(size) {
  pageSize.value = size
  page.value = 1
}

function toggleSort(key) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = key === 'time' ? 'desc' : 'asc'
  }
}

watch(
  filters,
  () => {
    page.value = 1
  },
  { deep: true }
)

const drawerOpen = ref(false)
const activeJob = ref(null)
const jobHistory = ref([])
const historyLoading = ref(false)

async function openDetail(job) {
  if (!job) return
  activeJob.value = job
  drawerOpen.value = true
  historyLoading.value = true
  jobHistory.value = []
  try {
    const res = await historyApi.list({ q: job.id, page: 1, size: 12 })
    jobHistory.value = (res?.data || []).filter((h) => h.taskId === job.id).slice(0, 6)
  } catch {
    jobHistory.value = []
  } finally {
    historyLoading.value = false
  }
}

function editJob(job) {
  if (!job) return
  router.push(`/jobs/${encodeURIComponent(job.id)}/edit`)
}

const buildOpen = ref(false)
const buildJob = ref(null)
const building = ref(false)

function openBuild(job) {
  if (!job) return
  buildJob.value = job
  buildOpen.value = true
}

async function submitBuild(refValue) {
  if (!buildJob.value) return
  building.value = true
  try {
    await jobApi.build(buildJob.value.id, refValue)
    buildOpen.value = false
    toast.success('构建已触发')
    await load()
    if (drawerOpen.value && activeJob.value) openDetail(activeJob.value)
  } catch (e) {
    toast.error(e.message || '构建触发失败')
  } finally {
    building.value = false
  }
}

const deleteOpen = ref(false)
const pendingJob = ref(null)
const deleting = ref(false)

function askDelete(job) {
  if (!job) return
  pendingJob.value = job
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!pendingJob.value) return
  deleting.value = true
  try {
    await jobApi.remove(pendingJob.value.id)
    toast.success('任务已删除')
    deleteOpen.value = false
    if (activeJob.value?.id === pendingJob.value.id) drawerOpen.value = false
    selected.value = selected.value.filter((id) => id !== pendingJob.value.id)
    await load()
  } catch (e) {
    toast.error(e.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

function copyText(text) {
  if (!text) return
  navigator.clipboard?.writeText(String(text)).then(
    () => toast.success('已复制'),
    () => toast.error('复制失败，请手动选择')
  )
}

function copySelected() {
  if (!selected.value.length) return
  copyText(selected.value.join('\n'))
}

function exportSelected() {
  const set = new Set(selected.value)
  const list = jobs.value.filter((j) => set.has(j.id))
  if (!list.length) return
  downloadCsv(
    `任务清单-${stamp()}.csv`,
    ['任务 ID', '环境', '所属服务', 'Jenkins 路径', '最近构建', '最近构建时间'],
    list.map((j) => [
      j.id,
      envName(j.envId),
      serverName(j.serverId),
      j.jobPath,
      latest(j.id)?.status || '未构建',
      latest(j.id)?.time ? formatDateTime(latest(j.id).time) : '',
    ])
  )
  toast.success(`已导出 ${list.length} 条`)
}

function consumeQuery() {
  const q = route.query
  if (q.new === '1' && canEdit.value) {
    const rest = { ...q }
    delete rest.new
    router.replace({ path: '/jobs/new', query: rest })
    return
  }
  if (q.q != null) filters.value.search = String(q.q)
  if (q.env != null) filters.value.envId = String(q.env)
  page.value = 1
}

watch(() => route.query, consumeQuery, { deep: true })

onMounted(async () => {
  consumeQuery()
  await load()
})
</script>

