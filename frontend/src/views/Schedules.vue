<template>
  <div>
    <PageHeader
      title="定时任务"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="load">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
        <BaseButton @click="openCreate">
          <template #icon><Plus class="h-3.5 w-3.5" /></template>
          新建定时
        </BaseButton>
      </template>
    </PageHeader>

    <div class="stagger mb-4 grid grid-cols-2 gap-3 xl:grid-cols-4">
      <KpiCard label="待执行" :value="count('pending')" suffix="个" :icon="Clock3" tone="info" :progress="ratio(count('pending'))" />
      <KpiCard label="进行中" :value="count('executing')" suffix="个" :icon="LoaderCircle" tone="warn" :progress="ratio(count('executing'))" />
      <KpiCard label="已完成" :value="count('completed')" suffix="个" :icon="CircleCheck" tone="success" :progress="ratio(count('completed'))" />
      <KpiCard
        label="失败 / 取消"
        :value="count('failed') + count('cancelled')"
        suffix="个"
        :icon="TriangleAlert"
        tone="failed"
        :progress="ratio(count('failed') + count('cancelled'))"
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
      :sort-key="sortKey"
      :sort-dir="sortDir"
      sticky-key="task"
      clickable
      :empty-title="dirty ? '没有匹配的定时任务' : '暂无定时任务'"
      :empty-description="dirty ? '调整关键词或状态后再试' : '创建一个定时任务，让发布在指定时间自动执行'"
      @update:page="page = $event"
      @update:page-size="pageSize = $event"
      @sort-change="toggleSort"
      @row-click="openDetail"
      @retry="load"
    >
      <template #toolbar>
        <FilterBar :dirty="dirty" @reset="resetFilters">
          <SearchField v-model="filters.search" placeholder="搜索任务 ID" width="230px" />
          <FilterSelect
            v-model="filters.status"
            :options="statusFilterOptions"
            placeholder="全部状态"
            aria-label="按状态筛选"
          />
          <template #right>
            <span class="text-[10.5px] text-ink-3">
              共 <CountUp :value="filtered.length" class="text-ink-2" /> 条
            </span>
          </template>
        </FilterBar>
      </template>

      <template #cell-task="{ row }">
        <p class="truncate font-mono text-[11.5px] font-medium text-ink">{{ row.taskId }}</p>
        <p class="mt-0.5 truncate text-[10.5px] text-ink-3">{{ envNameOf(row.taskId) }}</p>
      </template>

      <template #cell-ref="{ row }">
        <span class="mono text-[11px] text-ink-2">{{ row.refValue || '默认分支' }}</span>
      </template>

      <template #cell-time="{ row }">
        <div class="flex flex-col">
          <span class="num text-[11px] text-ink">{{ formatDateTime(row.scheduleTime) }}</span>
          <span
            v-if="row.status === 'pending'"
            class="num mt-0.5 text-[10px]"
            :class="remain(row.scheduleTime).urgent ? 'text-failed' : 'text-ink-3'"
          >
            {{ remain(row.scheduleTime).text }}
          </span>
        </div>
      </template>

      <template #cell-created="{ row }">
        <span class="text-[11px] text-ink-2">{{ row.createdName || '—' }}</span>
      </template>

      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>

      <template #cell-actions="{ row }">
        <RowActions
          :label="row.taskId"
          :items="[
            { label: '详情', onSelect: () => openDetail(row) },
            ...(row.status === 'pending' && canCancel(row)
              ? [{ label: '取消', onSelect: () => askCancel(row) }]
              : []),
            ...(canDelete ? [{ label: '删除', tone: 'danger', onSelect: () => askDelete(row) }] : []),
          ]"
        />
      </template>
    </DataTable>

    <DetailDrawer
      v-model="drawerOpen"
      :title="active?.taskId || ''"
      :subtitle="active ? envNameOf(active.taskId) : ''"
      width="460px"
    >
      <div v-if="active" class="space-y-4">
        <div class="flex items-center gap-2">
          <StatusBadge :status="active.status" />
          <span v-if="active.status === 'pending'" class="badge badge-neutral">
            {{ remain(active.scheduleTime).text }}
          </span>
        </div>

        <dl class="grid grid-cols-[78px_1fr] gap-x-3 gap-y-2.5 rounded-field border border-line px-3.5 py-3 text-xs">
          <dt class="text-ink-3">任务</dt>
          <dd class="font-mono text-[11px] text-ink">{{ active.taskId }}</dd>
          <dt class="text-ink-3">分支 / Tag</dt>
          <dd class="font-mono text-[11px] text-ink-2">{{ active.refValue || '默认分支' }}</dd>
          <dt class="text-ink-3">执行时间</dt>
          <dd class="num text-ink">{{ formatDateTime(active.scheduleTime) }}</dd>
          <dt class="text-ink-3">创建人</dt>
          <dd class="text-ink">{{ active.createdName || '—' }}</dd>
          <dt class="text-ink-3">创建时间</dt>
          <dd class="num text-[11px] text-ink-2">{{ formatDateTime(active.createdAt) }}</dd>
          <template v-if="active.executeTime">
            <dt class="text-ink-3">执行时间</dt>
            <dd class="num text-[11px] text-ink-2">{{ formatDateTime(active.executeTime) }}</dd>
          </template>
        </dl>

        <div v-if="active.errorMessage" class="rounded-field border border-failed bg-failed-soft px-3.5 py-3">
          <p class="section-label !text-failed">失败原因</p>
          <p class="mt-1 break-words text-[11px] leading-5 text-failed">{{ shortError(active.errorMessage, 360) }}</p>
        </div>

        <p class="text-[10.5px] leading-5 text-ink-3">
          调度器每分钟扫描一次待执行任务；到点后状态变为「进行中」，完成后写入发布历史。
        </p>
      </div>

      <template #footer>
        <BaseButton
          v-if="active?.status === 'pending' && canCancel(active)"
          size="sm"
          variant="secondary"
          @click="askCancel(active)"
        >
          取消定时
        </BaseButton>
        <BaseButton v-if="canDelete" size="sm" variant="danger" @click="askDelete(active)">删除记录</BaseButton>
      </template>
    </DetailDrawer>

    <AppModal
      v-model="createOpen"
      title="新建定时任务"
      subtitle="调度器每分钟扫描一次，到点自动触发"
      width="520px"
    >
      <div class="space-y-4">
        <section class="rounded-field border border-line p-3.5">
          <h3 class="mb-3 text-xs font-semibold text-ink">选择任务</h3>
          <div class="space-y-3">
            <div>
              <p class="field-label">任务 <span class="text-failed">*</span></p>
              <JobPicker v-model="createForm.taskId" :groups="jobGroups" placeholder="选择任务" />
              <p v-if="selectedJob" class="field-hint">
                {{ selectedIsProd ? '生产环境 / 必须填写版本 Tag' : `分支构建 / 默认 ${selectedEnv?.defaultRef || 'origin/test'}` }}
              </p>
            </div>

            <BaseInput
              v-model="createForm.refValue"
              :label="selectedIsProd ? '版本 Tag' : '分支 / 版本'"
              mono
              :required="selectedIsProd"
              :placeholder="selectedIsProd ? '例：20261105u01' : '留空使用环境默认分支'"
              :hint="selectedIsProd ? '格式如 20261105u01；缺失或非法会被拒绝' : '留空将使用环境默认分支'"
              :error="refError"
            />
          </div>
        </section>

        <section class="rounded-field border border-line p-3.5">
          <h3 class="mb-3 text-xs font-semibold text-ink">执行时间</h3>
          <label class="field-label" for="schedule-time">
            触发时间 <span class="text-failed" aria-hidden="true">*</span>
          </label>
          <input id="schedule-time" v-model="createForm.scheduleTime" type="datetime-local" class="input num" :min="minTime" />
          <p class="field-hint">按服务器本地时间计算，请在触发时间前至少留出 1 分钟。</p>
        </section>
      </div>

      <template #footer>
        <BaseButton variant="secondary" @click="createOpen = false">取消</BaseButton>
        <BaseButton :loading="creating" @click="submitCreate">保存</BaseButton>
      </template>
    </AppModal>

    <ConfirmDialog
      v-model="cancelOpen"
      title="取消定时"
      message="确定取消这条定时任务吗？"
      detail="状态将置为已取消，不再执行。"
      confirm-text="确认取消"
      :danger="false"
      :loading="acting"
      @confirm="confirmCancel"
    />
    <ConfirmDialog
      v-model="deleteOpen"
      title="删除记录"
      message="确定删除这条定时任务吗？"
      detail="删除后记录不可恢复，已产生的发布历史不受影响。"
      confirm-text="确认删除"
      :loading="acting"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  CircleCheck,
  Clock3,
  LoaderCircle,
  Plus,
  RefreshCw,
  TriangleAlert,
} from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import RowActions from '@/components/base/RowActions.vue'
import FilterSelect from '@/components/base/FilterSelect.vue'
import SearchField from '@/components/base/SearchField.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import AppModal from '@/components/base/AppModal.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import DetailDrawer from '@/components/base/DetailDrawer.vue'
import DataTable from '@/components/data/DataTable.vue'
import StatusBadge from '@/components/data/StatusBadge.vue'
import KpiCard from '@/components/data/KpiCard.vue'
import CountUp from '@/components/ui/CountUp.vue'
import JobPicker from '@/components/feature/JobPicker.vue'
import { scheduleApi } from '@/api/schedules'
import { jobApi } from '@/api/jobs'
import { envApi } from '@/api/environments'
import { usePermission } from '@/composables/usePermission'
import { useToast } from '@/composables/useToast'
import { useCountdown } from '@/composables/useCountdown'
import { countdownText, formatDateTime, shortError } from '@/utils/format'
import { tagFormat } from '@/utils/validate'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { canDelete, canCancel } = usePermission()
const { now } = useCountdown()

const all = ref([])
const jobs = ref([])
const envs = ref([])
const loading = ref(false)
const error = ref('')
const filters = ref({ search: '', status: '' })
const page = ref(1)
const pageSize = ref(10)
const sortKey = ref('time')
const sortDir = ref('desc')

const STATUS_OPTIONS = [
  { label: '待执行', value: 'pending' },
  { label: '进行中', value: 'executing' },
  { label: '已完成', value: 'completed' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
]
const statusFilterOptions = [{ label: '全部状态', value: '' }, ...STATUS_OPTIONS]

const columns = [
  { key: 'task', title: '任务', sortable: true, width: '24%' },
  { key: 'ref', title: '分支 / Tag', width: '18%' },
  { key: 'time', title: '执行时间', sortable: true, width: '20%' },
  { key: 'created', title: '创建人', width: '12%' },
  { key: 'status', title: '状态', width: '12%' },
  { key: 'actions', title: '', align: 'right', width: '188px' },
]

const jobMap = computed(() => Object.fromEntries(jobs.value.map((j) => [j.id, j])))
const envNameOf = (taskId) => {
  const envId = jobMap.value[taskId]?.envId
  return envs.value.find((e) => e.id === envId)?.name || '未配置环境'
}

const dirty = computed(() => !!(filters.value.search || filters.value.status))

const filtered = computed(() => {
  const f = filters.value
  const kw = (f.search || '').trim().toLowerCase()
  return all.value.filter((t) => {
    const hitKw = !kw || (t.taskId || '').toLowerCase().includes(kw)
    const hitStatus = !f.status || t.status === f.status
    return hitKw && hitStatus
  })
})

const sorted = computed(() => {
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...filtered.value].sort((a, b) => {
    if (sortKey.value === 'task') return String(a.taskId).localeCompare(String(b.taskId)) * dir
    return String(a.scheduleTime || '').localeCompare(String(b.scheduleTime || '')) * dir
  })
})

const rows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return sorted.value.slice(start, start + pageSize.value)
})

function count(status) {
  return all.value.filter((t) => t.status === status).length
}
function ratio(n) {
  return all.value.length ? Math.round((n / all.value.length) * 100) : 0
}
function remain(target) {
  void now.value
  return countdownText(target)
}

async function load() {
  loading.value = true
  const [t, j, e] = await Promise.allSettled([scheduleApi.list(), jobApi.list(), envApi.list()])
  if (t.status === 'fulfilled') {
    all.value = t.value || []
    error.value = ''
  } else {
    error.value = t.reason?.message || '定时任务加载失败'
  }
  if (j.status === 'fulfilled') jobs.value = j.value || []
  if (e.status === 'fulfilled') envs.value = e.value || []
  if (active.value) {
    const next = all.value.find((x) => x.id === active.value.id)
    if (next) active.value = next
    else drawerOpen.value = false
  }
  loading.value = false
}

function resetFilters() {
  filters.value = { search: '', status: '' }
  page.value = 1
}
function toggleSort(key) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else {
    sortKey.value = key
    sortDir.value = key === 'time' ? 'desc' : 'asc'
  }
}
watch(filters, () => (page.value = 1), { deep: true })

const drawerOpen = ref(false)
const active = ref(null)
function openDetail(row) {
  active.value = row
  drawerOpen.value = true
}

const jobGroups = computed(() => {
  const map = new Map()
  jobs.value.forEach((j) => {
    const key = j.envId || 'none'
    if (!map.has(key)) map.set(key, [])
    map.get(key).push({ label: j.id, value: j.id })
  })
  return [...map.entries()].map(([envId, options]) => ({
    label: envs.value.find((e) => e.id === envId)?.name || '未配置环境',
    options,
  }))
})

const createOpen = ref(false)
const creating = ref(false)
const refError = ref('')
const createForm = reactive({ taskId: '', refValue: '', scheduleTime: '' })
const minTime = ref('')

const selectedJob = computed(() => jobs.value.find((j) => j.id === createForm.taskId))
const selectedEnv = computed(() => envs.value.find((e) => e.id === selectedJob.value?.envId))
const selectedIsProd = computed(() => !!selectedEnv.value?.isProd)

function pad(n) {
  return String(n).padStart(2, '0')
}
function localInput(d) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function openCreate() {
  const d = new Date(Date.now() + 10 * 60000)
  const min = new Date(Date.now() + 60000)
  Object.assign(createForm, { taskId: '', refValue: '', scheduleTime: localInput(d) })
  minTime.value = localInput(min)
  refError.value = ''
  createOpen.value = true
}

watch(
  () => createForm.taskId,
  () => {
    if (!selectedIsProd.value) createForm.refValue = createForm.refValue.trim()
    refError.value = ''
  }
)

async function submitCreate() {
  refError.value = ''
  if (!createForm.taskId || !createForm.scheduleTime) {
    toast.error('请选择任务与执行时间')
    return
  }
  if (selectedIsProd.value) {
    const tag = createForm.refValue.trim()
    if (!tag) {
      refError.value = '生产环境必须填写版本 Tag'
      return
    }
    const msg = tagFormat(tag)
    if (msg) {
      refError.value = msg
      return
    }
  }
  creating.value = true
  try {
    await scheduleApi.create({
      taskId: createForm.taskId,
      refValue: createForm.refValue.trim(),
      scheduleTime: createForm.scheduleTime.replace('T', ' '),
    })
    createOpen.value = false
    toast.success('定时任务已创建')
    await load()
  } catch (e) {
    toast.error(e.message || '创建失败')
  } finally {
    creating.value = false
  }
}

const cancelOpen = ref(false)
const deleteOpen = ref(false)
const pending = ref(null)
const acting = ref(false)

function askCancel(row) {
  if (!row) return
  pending.value = row
  cancelOpen.value = true
}
function askDelete(row) {
  if (!row) return
  pending.value = row
  deleteOpen.value = true
}
async function confirmCancel() {
  if (!pending.value) return
  acting.value = true
  try {
    await scheduleApi.cancel(pending.value.id)
    toast.success('已取消')
    cancelOpen.value = false
    drawerOpen.value = false
    await load()
  } catch (e) {
    toast.error(e.message || '取消失败')
  } finally {
    acting.value = false
  }
}
async function confirmDelete() {
  if (!pending.value) return
  acting.value = true
  try {
    await scheduleApi.remove(pending.value.id)
    toast.success('已删除')
    deleteOpen.value = false
    drawerOpen.value = false
    await load()
  } catch (e) {
    toast.error(e.message || '删除失败')
  } finally {
    acting.value = false
  }
}

function consumeQuery() {
  if (route.query.new === '1') {
    openCreate()
    const rest = { ...route.query }
    delete rest.new
    router.replace({ query: rest })
  }
  if (route.query.q != null) filters.value.search = String(route.query.q)
  if (route.query.status != null) filters.value.status = String(route.query.status)
}

watch(() => route.query, consumeQuery, { deep: true })

onMounted(async () => {
  consumeQuery()
  await load()
})
</script>

