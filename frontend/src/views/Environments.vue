<template>
  <div>
    <PageHeader
      title="环境管理"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="load">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
        <BaseButton @click="openCreate">
          <template #icon><Plus class="h-3.5 w-3.5" /></template>
          新建环境
        </BaseButton>
      </template>
    </PageHeader>

    <div class="stagger mb-4 grid grid-cols-2 gap-3 xl:grid-cols-4">
      <KpiCard label="环境总数" :value="all.length" suffix="个" />
      <KpiCard
        label="Tag 环境"
        :value="prodCount"
        suffix="个"
        hint="强制选择版本 Tag"
      />
      <KpiCard
        label="分支环境"
        :value="all.length - prodCount"
        suffix="个"
        hint="可留空走默认分支"
      />
      <KpiCard label="关联任务" :value="jobTotal" suffix="条" hint="改动会影响后续构建" />
    </div>

    <div class="mb-3 flex flex-wrap items-center gap-2">
      <SearchField v-model="keyword" placeholder="搜索环境名 / ID" width="260px" />
      <span class="text-[10.5px] text-ink-3">
        <span class="num text-ink-2">{{ visible.length }}</span> 个环境
      </span>
      <span class="ml-auto hidden items-center gap-1.5 text-[10.5px] text-ink-3 sm:flex">
        <GripVertical class="h-3 w-3" />
        拖拽卡片可调整顺序
      </span>
    </div>

    <div v-if="loading && !all.length" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="i in 6" :key="i" class="card card-pad">
        <span class="skeleton h-8 w-8 rounded-[8px]" />
        <span class="skeleton mt-3 block h-4 w-28" />
        <span class="skeleton mt-2 block h-3 w-48" />
        <span class="skeleton mt-5 block h-14 w-full" />
      </div>
    </div>

    <div v-else-if="error" class="card card-pad">
      <ErrorState :message="error" @retry="load" />
    </div>

    <div v-else-if="visible.length" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="env in visible"
        :key="env.id"
        class="card card-pad group flex h-full flex-col transition-[border-color,box-shadow,opacity,transform] duration-150 hover:-translate-y-px hover:border-line-strong hover:shadow-pop"
        :class="{
          'cursor-grab active:cursor-grabbing': true,
          'opacity-45': dragging === env.id,
          'border-primary ring-2 ring-primary-soft': over === env.id,
        }"
        draggable="true"
        @dragstart="onDragStart(env, $event)"
        @dragover="onDragOver(env, $event)"
        @dragleave="onDragLeave(env)"
        @drop="onDrop(env, $event)"
        @dragend="onDragEnd"
      >
        <div class="flex items-start gap-3">
          <span
            class="grid h-8 w-8 shrink-0 place-items-center rounded-[8px]"
            :class="env.isProd ? 'bg-warn-soft text-warn' : 'bg-info-soft text-info'"
          >
            <Tag v-if="env.isProd" class="h-4 w-4" />
            <GitBranch v-else class="h-4 w-4" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="truncate text-[13.5px] font-semibold text-ink">{{ env.name }}</h3>
              <span class="badge" :class="env.isProd ? 'badge-warn' : 'badge-info'">
                {{ env.isProd ? '生产' : '分支' }}
              </span>
            </div>
            <p class="mt-1 text-[11px] leading-5 text-ink-3">
              {{ env.isProd ? 'Tag 构建 / 必须显式填写版本 Tag' : `分支构建 / 默认 ${env.defaultRef || 'origin/test'}` }}
            </p>
          </div>
          <GripVertical class="mt-0.5 h-4 w-4 shrink-0 text-ink-3 opacity-0 transition-opacity group-hover:opacity-100" />
        </div>

        <dl class="mt-4 grid grid-cols-2 gap-3 rounded-field border border-line bg-sunken px-3 py-2.5">
          <div>
            <dt class="section-label">关联任务</dt>
            <dd class="mt-1 text-xs font-medium text-ink">
              <span class="num">{{ env.jobCount ?? 0 }}</span> 条
            </dd>
          </div>
          <div class="min-w-0">
            <dt class="section-label">{{ env.isProd ? '标签参数名' : '分支参数名' }}</dt>
            <dd class="mt-1 truncate font-mono text-[11px] text-ink">{{ paramOf(env) }}</dd>
          </div>
        </dl>

        <div class="mt-auto flex items-center justify-between pt-3">
          <span class="truncate text-[10px] text-ink-3">ID / {{ env.id }}</span>
          <div class="flex items-center gap-0.5">
            <BaseButton size="xs" variant="ghost" @click="openEdit(env)">
              <template #icon><Pencil class="h-3 w-3" /></template>
              编辑
            </BaseButton>
            <BaseButton size="xs" variant="ghost" class="hover:text-failed" @click="askDelete(env)">
              <template #icon><Trash2 class="h-3 w-3" /></template>
              删除
            </BaseButton>
          </div>
        </div>
      </article>
    </div>

    <div v-else class="card card-pad">
      <EmptyState
        :title="keyword ? '没有匹配的环境' : '还没有环境'"
        :description="keyword ? '换一个关键词试试' : '先创建测试 / 预发 / 生产等环境，再建任务'"
      >
        <template #action>
          <BaseButton size="sm" @click="openCreate">新建环境</BaseButton>
        </template>
      </EmptyState>
    </div>

    <AppModal v-model="formOpen" :title="editingId ? '编辑环境' : '新建环境'" width="560px">
      <div v-if="editingId" class="mb-4 flex items-start gap-2.5 rounded-field border border-info bg-info-soft px-3.5 py-2.5 text-[11px] leading-5 text-info">
        <Info class="mt-0.5 h-3.5 w-3.5 shrink-0" />
        <span>
          当前环境关联 <b class="num">{{ editingJobCount }}</b> 个任务。构建方式、默认分支和参数名会影响这些任务
          <b>下一次及以后</b>的构建，正在执行的不受影响。
        </span>
      </div>

      <div class="space-y-4">
        <section>
          <h3 class="mb-3 text-xs font-semibold text-ink">基本信息</h3>
          <BaseInput
            v-model="form.name"
            label="环境名称"
            required
            placeholder="例：测试环境"
            hint="仅用于展示：钉钉卡片、历史筛选；改名不直接改变构建参数"
          />
        </section>

        <section>
          <h3 class="mb-2 text-xs font-semibold text-ink">构建方式</h3>
          <p class="mb-3 text-[11px] text-ink-3">决定触发构建时需要填写什么，切换会立即影响所有关联任务。</p>
          <div class="grid gap-2 sm:grid-cols-2">
            <button
              v-for="opt in BUILD_MODES"
              :key="opt.label"
              type="button"
              class="flex items-start gap-2.5 rounded-field border p-3 text-left transition-colors"
              :class="form.isProd === opt.value ? 'border-primary bg-primary-soft' : 'border-line bg-solid hover:border-line-strong'"
              @click="form.isProd = opt.value"
            >
              <component
                :is="opt.icon"
                class="mt-0.5 h-4 w-4 shrink-0"
                :class="form.isProd === opt.value ? 'text-primary' : 'text-ink-3'"
              />
              <span class="min-w-0">
                <span class="block text-xs font-medium text-ink">{{ opt.label }}</span>
                <span class="mt-0.5 block text-[10.5px] leading-4 text-ink-3">{{ opt.desc }}</span>
              </span>
            </button>
          </div>
        </section>

        <section class="space-y-3">
          <template v-if="!form.isProd">
            <BaseInput
              v-model="form.defaultRef"
              label="默认分支"
              mono
              placeholder="origin/test"
              hint="手动构建留空、或定时任务未填分支时使用"
            />
            <BaseInput
              v-model="form.defaultBranchParam"
              label="分支参数名"
              mono
              placeholder="branch"
              hint="传给 Jenkins 的参数名，填错 Jenkins 会读不到分支"
            />
          </template>
          <template v-else>
            <BaseInput
              v-model="form.defaultTagParam"
              label="标签参数名"
              mono
              placeholder="Tags"
              hint="Jenkins Job 上定义的 Tag 参数名，例如 Tags / GIT_TAG"
            />
          </template>
        </section>
      </div>

      <template #footer>
        <BaseButton variant="secondary" @click="formOpen = false">取消</BaseButton>
        <BaseButton :loading="saving" @click="saveEnv">保存</BaseButton>
      </template>
    </AppModal>

    <ConfirmDialog
      v-model="impactOpen"
      title="确认修改环境配置"
      :message="`此环境关联 ${editingJobCount} 个任务，以下改动会影响它们后续的构建：`"
      :items="impactItems"
      detail="正在执行或已完成的构建不受影响。"
      confirm-text="确认修改"
      :danger="false"
      :loading="saving"
      @confirm="doSave"
    />

    <ConfirmDialog
      v-model="deleteOpen"
      title="删除环境"
      :message="`确定删除环境「${pending?.name}」吗？`"
      detail="若仍有任务关联该环境，删除会被拒绝。"
      confirm-text="确认删除"
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import {
  GitBranch,
  GripVertical,
  Info,
  Pencil,
  Plus,
  RefreshCw,
  Tag,
  Trash2,
} from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import SearchField from '@/components/base/SearchField.vue'
import AppModal from '@/components/base/AppModal.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import EmptyState from '@/components/data/EmptyState.vue'
import ErrorState from '@/components/data/ErrorState.vue'
import KpiCard from '@/components/data/KpiCard.vue'
import { envApi } from '@/api/environments'
import { useDragSort } from '@/composables/useDragSort'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const all = ref([])
const loading = ref(false)
const error = ref('')
const keyword = ref('')

const BUILD_MODES = [
  { value: false, label: '分支构建', desc: '按分支名触发，可留空走默认分支', icon: GitBranch },
  { value: true, label: 'Tag 构建', desc: '按版本 Tag 触发，必须填写', icon: Tag },
]

async function load() {
  loading.value = true
  try {
    all.value = await envApi.list()
    error.value = ''
  } catch (e) {
    error.value = e.message || '环境加载失败'
  } finally {
    loading.value = false
  }
}

const { dragging, over, onDragStart, onDragOver, onDragLeave, onDrop, onDragEnd } = useDragSort({
  all,
  reorder: (ids) => envApi.reorder(ids),
  load,
})

const visible = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return all.value
  return all.value.filter((e) => (e.name || '').toLowerCase().includes(kw) || (e.id || '').toLowerCase().includes(kw))
})

const prodCount = computed(() => all.value.filter((e) => e.isProd).length)
const jobTotal = computed(() => all.value.reduce((n, e) => n + (e.jobCount || 0), 0))
const paramOf = (env) => (env.isProd ? env.defaultTagParam : env.defaultBranchParam) || (env.isProd ? 'Tags' : 'branch')

const formOpen = ref(false)
const editingId = ref('')
const saving = ref(false)
const original = ref(null)
const impactOpen = ref(false)
const impactItems = ref([])
const form = reactive({
  name: '',
  isProd: false,
  defaultRef: '',
  defaultBranchParam: 'branch',
  defaultTagParam: 'Tags',
})

const editingJobCount = computed(() => all.value.find((e) => e.id === editingId.value)?.jobCount ?? 0)

function openCreate() {
  editingId.value = ''
  original.value = null
  Object.assign(form, {
    name: '',
    isProd: false,
    defaultRef: 'origin/test',
    defaultBranchParam: 'branch',
    defaultTagParam: 'Tags',
  })
  formOpen.value = true
}

function openEdit(env) {
  editingId.value = env.id
  original.value = {
    name: env.name,
    isProd: !!env.isProd,
    defaultRef: env.defaultRef || '',
    defaultBranchParam: env.defaultBranchParam || 'branch',
    defaultTagParam: env.defaultTagParam || 'Tags',
  }
  Object.assign(form, { ...original.value })
  formOpen.value = true
}

function buildImpactLines() {
  const o = original.value
  if (!o) return []
  const lines = []
  if (form.isProd !== o.isProd) {
    lines.push(
      form.isProd
        ? '构建方式改为 Tag 构建：后续将强制填写版本 Tag'
        : `构建方式改为分支构建：不再强制填写 Tag，留空时使用 ${form.defaultRef || 'origin/test'}`
    )
  }
  if (!form.isProd && form.defaultRef !== o.defaultRef) {
    lines.push(`默认分支：${o.defaultRef || '空'} → ${form.defaultRef || '空'}（留空构建时使用）`)
  }
  if (!form.isProd && form.defaultBranchParam !== o.defaultBranchParam) {
    lines.push(`分支参数名：${o.defaultBranchParam} → ${form.defaultBranchParam}（传给 Jenkins 的参数名）`)
  }
  if (form.isProd && form.defaultTagParam !== o.defaultTagParam) {
    lines.push(`标签参数名：${o.defaultTagParam} → ${form.defaultTagParam}（传给 Jenkins 的参数名）`)
  }
  if (form.name !== o.name) {
    lines.push('环境名称变化：影响钉钉卡片文案与历史筛选展示')
  }
  return lines
}

function saveEnv() {
  form.name = form.name.trim()
  if (!form.name) {
    toast.error('请填写环境名称')
    return
  }
  const param = form.isProd ? form.defaultTagParam.trim() : form.defaultBranchParam.trim()
  if (param && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(param)) {
    toast.error('参数名只能包含字母、数字、下划线，且不能以数字开头')
    return
  }
  if (editingId.value && editingJobCount.value > 0) {
    const lines = buildImpactLines()
    if (lines.length) {
      impactItems.value = lines
      impactOpen.value = true
      return
    }
  }
  doSave()
}

async function doSave() {
  saving.value = true
  try {
    if (editingId.value) await envApi.update(editingId.value, { ...form })
    else await envApi.create({ ...form })
    formOpen.value = false
    impactOpen.value = false
    toast.success(editingId.value ? '环境已更新' : '环境已创建')
    await load()
  } catch (e) {
    toast.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const deleteOpen = ref(false)
const pending = ref(null)
const deleting = ref(false)

function askDelete(env) {
  pending.value = env
  deleteOpen.value = true
}
async function confirmDelete() {
  if (!pending.value) return
  deleting.value = true
  try {
    await envApi.remove(pending.value.id)
    deleteOpen.value = false
    toast.success('环境已删除')
    await load()
  } catch (e) {
    toast.error(e.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

onMounted(load)
</script>