<template>
  <div>
    <PageHeader
      title="服务管理"
    >
      <template #actions>
        <BaseButton variant="secondary" :loading="loading" @click="load">
          <template #icon><RefreshCw class="h-3.5 w-3.5" /></template>
          刷新
        </BaseButton>
        <BaseButton @click="openCreate">
          <template #icon><Plus class="h-3.5 w-3.5" /></template>
          添加服务
        </BaseButton>
      </template>
    </PageHeader>

    <div class="mb-4 flex items-start gap-2.5 rounded-card border border-line bg-surface px-4 py-2.5 text-[11px] leading-5 text-ink-3">
      <ShieldCheck class="mt-0.5 h-3.5 w-3.5 shrink-0 text-ink-3" />
      <p>
        API Token 只在后端调用 Jenkins 时使用，列表与接口都不会回显；编辑时留空表示保持原值。
        默认服务 <span class="text-ink-2">jenkins</span> 不可删除。
      </p>
    </div>

    <div v-if="loading && !all.length" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="i in 3" :key="i" class="card card-pad">
        <span class="skeleton h-8 w-8 rounded-[8px]" />
        <span class="skeleton mt-3 block h-4 w-32" />
        <span class="skeleton mt-2 block h-3 w-48" />
        <span class="skeleton mt-5 block h-16 w-full" />
      </div>
    </div>

    <div v-else-if="error" class="card card-pad">
      <ErrorState :message="error" @retry="load" />
    </div>

    <div v-else-if="all.length" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="s in all"
        :key="s.id"
        class="card card-pad group flex h-full flex-col transition-[border-color,box-shadow,opacity,transform] duration-150 hover:-translate-y-px hover:border-line-strong hover:shadow-pop"
        :class="{
          'cursor-grab active:cursor-grabbing': true,
          'opacity-45': dragging === s.id,
          'border-primary ring-2 ring-primary-soft': over === s.id,
        }"
        draggable="true"
        @dragstart="onDragStart(s, $event)"
        @dragover="onDragOver(s, $event)"
        @dragleave="onDragLeave(s)"
        @drop="onDrop(s, $event)"
        @dragend="onDragEnd"
      >
        <div class="flex items-start gap-3">
          <span class="grid h-8 w-8 shrink-0 place-items-center rounded-[8px] bg-primary-soft text-primary-ink">
            <Server class="h-4 w-4" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="truncate text-[13.5px] font-semibold text-ink">{{ s.name }}</h3>
              <span v-if="s.id === 'jenkins'" class="badge badge-info">DEFAULT</span>
            </div>
            <p class="mt-0.5 truncate text-[10px] text-ink-3">ID / {{ s.id }}</p>
          </div>
          <GripVertical class="mt-0.5 h-4 w-4 shrink-0 text-ink-3 opacity-0 transition-opacity group-hover:opacity-100" />
        </div>

        <p class="mt-3 truncate rounded-field border border-line bg-sunken px-2.5 py-2 font-mono text-[10.5px] text-ink-2" :title="s.baseUrl">
          {{ s.baseUrl }}
        </p>

        <dl class="mt-3 grid grid-cols-2 gap-3">
          <div class="min-w-0">
            <dt class="section-label">用户名</dt>
            <dd class="mt-1 truncate text-xs text-ink">{{ s.user }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="section-label">创建时间</dt>
            <dd class="num mt-1 truncate text-[11px] text-ink-2">{{ formatDateTime(s.createdAt) }}</dd>
          </div>
        </dl>

        <div class="mt-auto flex items-center justify-between border-t border-line-soft pt-2.5">
          <span class="text-[10px] text-ink-3">{{ s.id === 'jenkins' ? '系统默认服务' : '自定义服务' }}</span>
          <div class="flex items-center gap-0.5">
            <BaseButton size="xs" variant="ghost" @click="openEdit(s)">
              <template #icon><Pencil class="h-3 w-3" /></template>
              编辑
            </BaseButton>
            <BaseButton
              v-if="s.id !== 'jenkins'"
              size="xs"
              variant="ghost"
              class="hover:text-failed"
              @click="askDelete(s)"
            >
              <template #icon><Trash2 class="h-3 w-3" /></template>
              删除
            </BaseButton>
          </div>
        </div>
      </article>
    </div>

    <div v-else class="card card-pad">
      <EmptyState title="还没有 Jenkins 服务" description="添加一个服务地址，任务才能绑定并触发构建">
        <template #action>
          <BaseButton size="sm" @click="openCreate">添加服务</BaseButton>
        </template>
      </EmptyState>
    </div>

    <AppModal
      v-model="formOpen"
      :title="editingId ? '编辑服务' : '添加服务'"
      subtitle="连接信息仅用于后端调用 Jenkins"
      width="540px"
    >
      <div class="space-y-4">
        <section>
          <h3 class="mb-3 text-xs font-semibold text-ink">基本信息</h3>
          <div class="grid gap-3 sm:grid-cols-2">
            <BaseInput v-model="form.name" label="名称" required placeholder="例：后端 Jenkins" />
            <BaseInput
              v-model="form.id"
              label="服务 ID"
              mono
              required
              :disabled="!!editingId"
              placeholder="例：jenkins-2"
              :hint="editingId ? '服务 ID 创建后不可修改' : '唯一标识，任务通过它绑定服务'"
            />
          </div>
        </section>

        <section class="space-y-3">
          <h3 class="text-xs font-semibold text-ink">连接信息</h3>
          <BaseInput
            v-model="form.baseUrl"
            label="Base URL"
            mono
            required
            placeholder="https://jenkins.example.com"
            :error="urlError"
            hint="末尾斜杠会自动归一，不会拼出 //"
          />
          <BaseInput v-model="form.user" label="用户名" required placeholder="admin" />
          <BaseInput
            v-model="form.token"
            label="API Token"
            type="password"
            :required="!editingId"
            placeholder="Token 或密码"
            :hint="editingId ? '留空表示保持原有 Token 不变' : '仅用于调用 Jenkins API 的凭据'"
          />
        </section>
      </div>

      <template #footer>
        <BaseButton variant="secondary" @click="formOpen = false">取消</BaseButton>
        <BaseButton :loading="saving" @click="saveServer">保存</BaseButton>
      </template>
    </AppModal>

    <ConfirmDialog
      v-model="deleteOpen"
      title="删除服务"
      :message="`确定删除「${pending?.name}」吗？`"
      detail="绑定该服务的任务将无法构建；默认服务不可删除。"
      confirm-text="确认删除"
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { GripVertical, Pencil, Plus, RefreshCw, Server, ShieldCheck, Trash2 } from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import AppModal from '@/components/base/AppModal.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import EmptyState from '@/components/data/EmptyState.vue'
import ErrorState from '@/components/data/ErrorState.vue'
import { serverApi } from '@/api/servers'
import { useDragSort } from '@/composables/useDragSort'
import { useToast } from '@/composables/useToast'
import { formatDateTime } from '@/utils/format'
import { urlFormat } from '@/utils/validate'

const toast = useToast()
const all = ref([])
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  try {
    all.value = await serverApi.list()
    error.value = ''
  } catch (e) {
    error.value = e.message || '服务加载失败'
  } finally {
    loading.value = false
  }
}

const { dragging, over, onDragStart, onDragOver, onDragLeave, onDrop, onDragEnd } = useDragSort({
  all,
  reorder: (ids) => serverApi.reorder(ids),
  load,
})

const formOpen = ref(false)
const editingId = ref('')
const saving = ref(false)
const form = reactive({ id: '', name: '', baseUrl: '', user: '', token: '' })

const urlError = computed(() => (form.baseUrl ? urlFormat(form.baseUrl) : ''))

function openCreate() {
  editingId.value = ''
  Object.assign(form, { id: '', name: '', baseUrl: '', user: '', token: '' })
  formOpen.value = true
}
function openEdit(s) {
  editingId.value = s.id
  Object.assign(form, { id: s.id, name: s.name, baseUrl: s.baseUrl, user: s.user, token: '' })
  formOpen.value = true
}

async function saveServer() {
  if (!form.id || !form.name || !form.baseUrl || !form.user) {
    toast.error('请填写服务 ID、名称、Base URL 与用户名')
    return
  }
  if (urlError.value) {
    toast.error(urlError.value)
    return
  }
  saving.value = true
  try {
    const payload = { ...form, id: form.id.trim(), name: form.name.trim(), baseUrl: form.baseUrl.trim() }
    if (editingId.value) await serverApi.update(editingId.value, payload)
    else await serverApi.create(payload)
    formOpen.value = false
    toast.success(editingId.value ? '服务已更新' : '服务已添加')
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

function askDelete(s) {
  if (s.id === 'jenkins') return
  pending.value = s
  deleteOpen.value = true
}
async function confirmDelete() {
  if (!pending.value) return
  deleting.value = true
  try {
    await serverApi.remove(pending.value.id)
    deleteOpen.value = false
    toast.success('服务已删除')
    await load()
  } catch (e) {
    toast.error(e.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

onMounted(load)
</script>