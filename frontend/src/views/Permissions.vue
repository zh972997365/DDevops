<template>
  <div>
    <PageHeader
      title="用户管理"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="load">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
        <BaseButton variant="secondary" :loading="syncing" @click="syncDingtalk">
          <template #icon><RefreshCcw class="h-3.5 w-3.5" /></template>
          同步通讯录
        </BaseButton>
        <BaseButton @click="openCreate">
          <template #icon><Plus class="h-3.5 w-3.5" /></template>
          添加用户
        </BaseButton>
      </template>
    </PageHeader>

    <div class="stagger mb-4 grid grid-cols-2 gap-3 xl:grid-cols-4">
      <KpiCard label="成员总数" :value="users.length" suffix="人" />
      <KpiCard label="管理员" :value="adminCount" suffix="人" hint="全量项目权限" />
      <KpiCard
        label="普通用户"
        :value="users.length - adminCount"
        suffix="人"
        hint="按项目授权"
      />
      <KpiCard
        label="未授权"
        :value="unassigned"
        suffix="人"
        hint="登录后无可用任务"
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
      sticky-key="name"
      min-width="940px"
      :empty-title="dirty ? '没有匹配的成员' : '暂无账号'"
      :empty-description="dirty ? '调整关键词或角色后再试' : '添加一位成员，或从钉钉通讯录同步后选择'"
      @update:page="page = $event"
      @update:page-size="pageSize = $event; page = 1"
      @sort-change="toggleSort"
      @retry="load"
    >
      <template #toolbar>
        <FilterBar :dirty="dirty" @reset="resetFilters">
          <SearchField v-model="filters.search" placeholder="搜索姓名 / 钉钉 ID / 登录名" width="260px" />
          <FilterSelect
            v-model="filters.role"
            :options="roleFilterOptions"
            placeholder="全部角色"
            aria-label="按角色筛选"
          />
          <template #right>
            <span class="text-[10.5px] text-ink-3">
              共 <CountUp :value="filtered.length" class="text-ink-2" /> 人
            </span>
          </template>
        </FilterBar>
      </template>

      <template #cell-name="{ row }">
        <div class="flex items-center gap-2.5">
          <span class="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-primary-soft text-[11px] font-semibold text-primary-ink">
            {{ (row.userName || 'U').slice(0, 1) }}
          </span>
          <div class="min-w-0">
            <p class="truncate text-xs font-medium text-ink">{{ row.userName || '—' }}</p>
            <p class="mt-0.5 truncate text-[10px] text-ink-3">{{ row.userId === PROTECTED_USER_ID ? '受保护账号' : '成员' }}</p>
          </div>
        </div>
      </template>

      <template #cell-userId="{ row }">
        <button
          type="button"
          class="group inline-flex max-w-full items-center gap-1.5 font-mono text-[11px] text-ink-2 transition-colors hover:text-primary-ink"
          title="点击复制钉钉用户 ID"
          @click="copyText(row.userId)"
        >
          <span class="truncate">{{ row.userId }}</span>
          <Copy class="h-3 w-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
        </button>
      </template>

      <template #cell-username="{ row }">
        <span class="font-mono text-[11px] text-ink-2">{{ row.username || '—' }}</span>
      </template>

      <template #cell-role="{ row }">
        <span class="badge" :class="row.webRole === 'admin' ? 'badge-info' : 'badge-neutral'">
          {{ row.webRole === 'admin' ? '管理员' : '普通用户' }}
        </span>
      </template>

      <template #cell-scope="{ row }">
        <span v-if="row.webRole === 'admin'" class="badge badge-info">全部项目</span>
        <span v-else-if="row.allowedJobs?.length" class="badge badge-info">
          {{ row.allowedJobs.length }} 个项目
        </span>
        <span v-else class="badge badge-warn">无授权</span>
      </template>

      <template #cell-actions="{ row }">
        <RowActions
          :label="row.userName || row.userId"
          :items="[
            { label: '编辑', onSelect: () => openEdit(row) },
            { label: '重置密码', onSelect: () => openReset(row) },
            {
              label: '删除',
              tone: 'danger',
              disabled: row.userId === PROTECTED_USER_ID,
              title: row.userId === PROTECTED_USER_ID ? '受保护账号不可删除' : undefined,
              onSelect: () => askDelete(row),
            },
          ]"
        />
      </template>
    </DataTable>

    <AppModal
      v-model="formOpen"
      :title="editingId ? '编辑用户' : '添加用户'"
      subtitle="账号同时用于 Web 登录与机器人指令"
      width="640px"
    >
      <div class="space-y-4">
        <section class="rounded-field border border-line p-3.5">
          <h3 class="mb-3 text-xs font-semibold text-ink">身份</h3>
          <div class="space-y-3">
            <DingTalkPicker v-if="!editingId" @pick="onPickUser" />
            <div class="grid gap-3 sm:grid-cols-2">
              <BaseInput
                v-model="form.userId"
                label="钉钉用户 ID"
                mono
                required
                :disabled="!!editingId"
                placeholder="钉钉 userid"
              />
              <BaseInput v-model="form.userName" label="用户名称" required placeholder="显示名称" />
            </div>
          </div>
        </section>

        <section class="rounded-field border border-line p-3.5">
          <h3 class="mb-3 text-xs font-semibold text-ink">登录与权限</h3>
          <div class="grid gap-3 sm:grid-cols-2">
            <BaseInput v-model="form.username" label="登录用户名" mono required placeholder="英文登录账号" />
            <BaseInput
              v-model="form.password"
              label="登录密码"
              type="password"
              :required="!editingId"
              placeholder="编辑时留空表示不改"
              hint="至少 6 位；修改后旧会话立即失效"
            />
            <div class="sm:col-span-2">
              <BaseSelect
                v-model="form.webRole"
                label="角色"
                :options="roleOptions"
                :disabled="form.userId === PROTECTED_USER_ID"
                :hint="form.userId === PROTECTED_USER_ID ? '受保护管理员账号角色不可修改' : '管理员不受项目授权限制'"
              />
            </div>
          </div>
        </section>

        <section class="rounded-field border border-line p-3.5">
          <div class="mb-2 flex items-center justify-between gap-3">
            <h3 class="text-xs font-semibold text-ink">可构建项目</h3>
            <span class="text-[10.5px] text-ink-3">
              已选 <span class="num text-ink-2">{{ form.allowedJobs.length }}</span> / {{ jobs.length }}
            </span>
          </div>

          <div v-if="form.webRole === 'admin'" class="rounded-field border border-line bg-sunken px-3 py-2.5 text-[11px] text-ink-3">
            管理员可访问所有项目，无需勾选。
          </div>

          <template v-else>
            <p class="mb-2 text-[10.5px] leading-5 text-ink-3">
              留空表示无授权项目：该用户登录后看不到任何任务、无法构建。管理员始终全量。
            </p>
            <SearchField v-model="jobKeyword" placeholder="搜索项目 ID" width="100%" />
            <div class="mt-2 max-h-[190px] overflow-y-auto rounded-field border border-line p-1.5">
              <div v-if="visibleJobs.length" class="grid gap-1 sm:grid-cols-2">
                <label
                  v-for="job in visibleJobs"
                  :key="job.id"
                  class="flex cursor-pointer items-center gap-2 rounded-[6px] px-2 py-1.5 font-mono text-[10.5px] transition-colors"
                  :class="form.allowedJobs.includes(job.id) ? 'bg-primary-soft text-primary-ink' : 'text-ink-2 hover:bg-sunken'"
                >
                  <input v-model="form.allowedJobs" type="checkbox" :value="job.id" class="h-3.5 w-3.5 rounded-[3px] accent-[var(--primary)]" />
                  <span class="truncate">{{ job.id }}</span>
                </label>
              </div>
              <p v-else class="py-5 text-center text-[11px] text-ink-3">
                {{ jobKeyword.trim() ? '没有匹配的项目' : '暂无可选项目，请先在任务管理中创建任务' }}
              </p>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-3 text-[11px]">
              <button type="button" class="text-primary-ink hover:underline" @click="selectAllJobs">
                {{ jobKeyword.trim() ? '全选筛选结果' : '全选全部' }}
              </button>
              <button type="button" class="text-ink-3 hover:underline" @click="form.allowedJobs = []">清空</button>
              <span class="ml-auto text-[10.5px] text-ink-3">未勾选 = 无授权</span>
            </div>
          </template>
        </section>
      </div>

      <template #footer>
        <BaseButton variant="secondary" @click="formOpen = false">取消</BaseButton>
        <BaseButton :loading="saving" @click="saveUser">保存</BaseButton>
      </template>
    </AppModal>

    <AppModal v-model="resetOpen" title="重置密码" :subtitle="`账号 ${resetTarget?.username || ''}`" width="420px">
      <BaseInput v-model="newPassword" label="新密码" type="password" placeholder="至少 6 位" hint="重置后该用户已登录的会话会立即失效。" />
      <template #footer>
        <BaseButton variant="secondary" @click="resetOpen = false">取消</BaseButton>
        <BaseButton :loading="resetting" @click="submitReset">确认重置</BaseButton>
      </template>
    </AppModal>

    <ConfirmDialog
      v-model="deleteOpen"
      title="删除用户"
      :message="`确定删除用户「${pending?.userName || pending?.userId || ''}」吗？`"
      detail="该账号的登录会话会立即失效，历史记录仍会保留。"
      confirm-text="确认删除"
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  Copy,
  Plus,
  RefreshCcw,
  RefreshCw,
} from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import RowActions from '@/components/base/RowActions.vue'
import FilterSelect from '@/components/base/FilterSelect.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import SearchField from '@/components/base/SearchField.vue'
import FilterBar from '@/components/base/FilterBar.vue'
import AppModal from '@/components/base/AppModal.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import DataTable from '@/components/data/DataTable.vue'
import KpiCard from '@/components/data/KpiCard.vue'
import CountUp from '@/components/ui/CountUp.vue'
import DingTalkPicker from '@/components/feature/DingTalkPicker.vue'
import { userApi } from '@/api/users'
import { jobApi } from '@/api/jobs'
import { dingtalkApi } from '@/api/dingtalk'
import { useToast } from '@/composables/useToast'
import { PROTECTED_USER_ID } from '@/constants/role'

const toast = useToast()
const users = ref([])
const jobs = ref([])
const loading = ref(false)
const error = ref('')
const syncing = ref(false)

const filters = ref({ search: '', role: '' })
const page = ref(1)
const pageSize = ref(10)
const sortKey = ref('name')
const sortDir = ref('asc')

const columns = [
  { key: 'name', title: '成员', sortable: true, width: '19%' },
  { key: 'userId', title: '钉钉用户 ID', width: '20%' },
  { key: 'username', title: '登录名', width: '14%' },
  { key: 'role', title: '角色', sortable: true, width: '10%' },
  { key: 'scope', title: '可构建项目', sortable: true, width: '14%' },
  { key: 'actions', title: '', align: 'right', width: '224px' },
]

const roleOptions = [
  { label: '普通用户', value: 'user' },
  { label: '管理员', value: 'admin' },
]
const roleFilterOptions = [{ label: '全部角色', value: '' }, ...roleOptions]

const adminCount = computed(() => users.value.filter((u) => u.webRole === 'admin').length)
const unassigned = computed(
  () => users.value.filter((u) => u.webRole !== 'admin' && !(u.allowedJobs?.length > 0)).length
)

const dirty = computed(() => !!(filters.value.search || filters.value.role))

const filtered = computed(() => {
  const f = filters.value
  const kw = (f.search || '').trim().toLowerCase()
  return users.value.filter((u) => {
    const hitRole = !f.role || u.webRole === f.role
    const hitKw =
      !kw ||
      (u.userName || '').toLowerCase().includes(kw) ||
      (u.userId || '').toLowerCase().includes(kw) ||
      (u.username || '').toLowerCase().includes(kw)
    return hitRole && hitKw
  })
})

const sorted = computed(() => {
  const dir = sortDir.value === 'asc' ? 1 : -1
  return [...filtered.value].sort((a, b) => {
    if (sortKey.value === 'role') return String(a.webRole).localeCompare(String(b.webRole)) * dir
    if (sortKey.value === 'scope') return ((a.allowedJobs?.length || 0) - (b.allowedJobs?.length || 0)) * dir
    return String(a.userName || a.userId).localeCompare(String(b.userName || b.userId)) * dir
  })
})

const rows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return sorted.value.slice(start, start + pageSize.value)
})

watch(filters, () => (page.value = 1), { deep: true })

function toggleSort(key) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else {
    sortKey.value = key
    sortDir.value = 'asc'
  }
}
function resetFilters() {
  filters.value = { search: '', role: '' }
}

async function load() {
  loading.value = true
  const [u, j] = await Promise.allSettled([userApi.list(), jobApi.list()])
  if (u.status === 'fulfilled') {
    users.value = u.value || []
    error.value = ''
  } else {
    error.value = u.reason?.message || '用户加载失败'
  }
  if (j.status === 'fulfilled') jobs.value = j.value || []
  loading.value = false
}

function copyText(text) {
  if (!text) return
  navigator.clipboard?.writeText(String(text)).then(
    () => toast.success('已复制'),
    () => toast.error('复制失败，请手动选择')
  )
}

const formOpen = ref(false)
const editingId = ref('')
const saving = ref(false)
const jobKeyword = ref('')
const form = reactive({ userId: '', userName: '', username: '', password: '', webRole: 'user', allowedJobs: [] })

const visibleJobs = computed(() => {
  const kw = jobKeyword.value.trim().toLowerCase()
  if (!kw) return jobs.value
  return jobs.value.filter((j) => (j.id || '').toLowerCase().includes(kw))
})

function openCreate() {
  editingId.value = ''
  jobKeyword.value = ''
  Object.assign(form, { userId: '', userName: '', username: '', password: '', webRole: 'user', allowedJobs: [] })
  formOpen.value = true
}
function openEdit(user) {
  editingId.value = user.userId
  jobKeyword.value = ''
  Object.assign(form, {
    userId: user.userId,
    userName: user.userName || '',
    username: user.username || '',
    password: '',
    webRole: user.webRole || 'user',
    allowedJobs: [...(user.allowedJobs || [])],
  })
  formOpen.value = true
}

function onPickUser(u) {
  form.userId = u.userid
  form.userName = u.name
  if (u.pinyin) form.username = u.pinyin
}

function selectAllJobs() {
  if (!jobKeyword.value.trim()) {
    form.allowedJobs = jobs.value.map((j) => j.id)
    return
  }
  const set = new Set(form.allowedJobs)
  visibleJobs.value.forEach((j) => set.add(j.id))
  form.allowedJobs = [...set]
}

async function saveUser() {
  if (!form.userId || !form.userName || !form.username) {
    toast.error('请填写用户 ID、名称与登录用户名')
    return
  }
  if (!editingId.value && !form.password) {
    toast.error('新增用户必须设置密码')
    return
  }
  if (form.password && form.password.length < 6) {
    toast.error('密码至少 6 位')
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form,
      webRole: form.userId === PROTECTED_USER_ID ? 'admin' : form.webRole,
    }
    if (editingId.value) await userApi.update(editingId.value, payload)
    else await userApi.create(payload)
    formOpen.value = false
    toast.success(editingId.value ? '用户已更新' : '用户已创建')
    await load()
  } catch (e) {
    toast.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const resetOpen = ref(false)
const resetting = ref(false)
const resetTarget = ref(null)
const newPassword = ref('')

function openReset(user) {
  resetTarget.value = user
  newPassword.value = ''
  resetOpen.value = true
}

async function submitReset() {
  if (newPassword.value.length < 6) {
    toast.error('密码至少 6 位')
    return
  }
  resetting.value = true
  try {
    const u = resetTarget.value
    await userApi.update(u.userId, {
      userId: u.userId,
      userName: u.userName,
      username: u.username,
      password: newPassword.value,
      webRole: u.webRole,
    })
    resetOpen.value = false
    toast.success('密码已重置')
  } catch (e) {
    toast.error(e.message || '重置失败')
  } finally {
    resetting.value = false
  }
}

const deleteOpen = ref(false)
const pending = ref(null)
const deleting = ref(false)

function askDelete(user) {
  if (user.userId === PROTECTED_USER_ID) return
  pending.value = user
  deleteOpen.value = true
}
async function confirmDelete() {
  if (!pending.value) return
  deleting.value = true
  try {
    await userApi.remove(pending.value.userId)
    deleteOpen.value = false
    toast.success('用户已删除')
    await load()
  } catch (e) {
    toast.error(e.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

async function syncDingtalk() {
  syncing.value = true
  try {
    const res = await dingtalkApi.syncUsers()
    toast.success(`同步成功，共 ${res.count} 人`)
  } catch (e) {
    toast.error(e.message || '同步失败')
  } finally {
    syncing.value = false
  }
}

onMounted(load)
</script>

