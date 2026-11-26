<template>
  <div>
    <PageHeader
      :title="editing ? '编辑任务' : '新建任务'"
    >
      <template #actions>
        <BaseButton variant="secondary" @click="router.push('/jobs')">
          <template #icon><ArrowLeft class="h-3.5 w-3.5" /></template>
          返回列表
        </BaseButton>
        <BaseButton v-if="!loading && !loadError" type="submit" form="job-form" :loading="saving">
          {{ editing ? '保存修改' : '创建任务' }}
        </BaseButton>
      </template>
    </PageHeader>

    <div v-if="loading" class="grid gap-4 xl:grid-cols-12">
      <div class="card card-pad xl:col-span-8">
        <span class="skeleton h-4 w-24" />
        <span class="skeleton mt-5 block h-9 w-full" />
        <span class="skeleton mt-4 block h-9 w-full" />
        <span class="skeleton mt-8 block h-4 w-24" />
        <span class="skeleton mt-5 block h-9 w-full" />
      </div>
      <div class="card card-pad xl:col-span-4">
        <span class="skeleton h-4 w-20" />
        <span class="skeleton mt-5 block h-24 w-full" />
      </div>
    </div>

    <div v-else-if="loadError" class="card card-pad">
      <ErrorState :message="loadError" :retryable="false" />
      <div class="mt-3 text-center">
        <BaseButton size="sm" variant="secondary" @click="router.push('/jobs')">返回任务列表</BaseButton>
      </div>
    </div>

    <form v-else id="job-form" class="grid gap-4 xl:grid-cols-12" @submit.prevent="submit">
      <div class="space-y-4 xl:col-span-8">
        <div
          v-if="editing && form.jobId.trim() !== originalId"
          class="flex items-start gap-2.5 rounded-card border border-warn bg-warn-soft px-4 py-3 text-xs leading-5 text-warn"
        >
          <TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" />
          <p>
            任务 ID 将改为 <b class="font-mono">{{ form.jobId.trim() || '—' }}</b>。钉钉指令需改用新 ID，旧 ID 立即失效；
            已关联的定时任务与发布历史会自动同步，Jenkins 端路径不变。
          </p>
        </div>

        <div
          v-if="serverError"
          class="rounded-card border border-failed bg-failed-soft px-4 py-3 text-xs leading-5 text-failed"
          role="alert"
        >
          {{ serverError }}
        </div>

        <section class="card card-pad">
          <header class="mb-4">
            <h2 class="panel-title">基本信息</h2>
            <p class="mt-0.5 text-[11px] text-ink-3">任务 ID 既是主键也是任务名，创建后仍可修改。</p>
          </header>
          <div class="grid gap-4 sm:grid-cols-2">
            <BaseInput
              v-model="form.jobId"
              label="任务 ID"
              mono
              required
              placeholder="例：backend-deploy"
              :error="errors.jobId"
              hint="用于钉钉指令与发布历史检索"
            />
            <BaseSelect
              v-model="form.envId"
              label="所属环境"
              required
              placeholder="选择环境"
              :options="envOptions"
              :error="errors.envId"
              hint="环境决定构建方式、默认分支与参数名"
            />
          </div>
        </section>

        <section class="card card-pad">
          <header class="mb-4">
            <h2 class="panel-title">构建配置</h2>
            <p class="mt-0.5 text-[11px] text-ink-3">路径与服务器决定构建请求发往哪里，环境决定参数怎么传。</p>
          </header>
          <div class="space-y-4">
            <BaseInput
              v-model="form.jobPath"
              label="Jenkins 任务路径"
              mono
              required
              placeholder="/job/my-service/job/build"
              :error="errors.jobPath"
              hint="可填 /view/xxx/job/xxx/build 或 /job/xxx；系统会换算为 buildWithParameters"
            />
            <BaseSelect
              v-model="form.serverId"
              label="所属服务器"
              required
              placeholder="选择 Jenkins 服务"
              :options="serverOptions"
              :error="errors.serverId"
            />
          </div>
        </section>

        <section class="card card-pad">
          <header class="mb-3">
            <h2 class="panel-title">请求预览</h2>
            <p class="mt-0.5 text-[11px] text-ink-3">只读预览，实际请求由后端归一化后发出。</p>
          </header>
          <p class="break-all rounded-field border border-line bg-sunken px-3 py-2.5 font-mono text-[11px] leading-5 text-ink-2">
            POST&nbsp;{{ endpointPreview }}
          </p>
        </section>
      </div>

      <aside class="xl:col-span-4">
        <div class="space-y-4 xl:sticky xl:top-[76px]">
          <section class="card card-pad">
            <header class="mb-3 flex items-center justify-between">
              <h2 class="panel-title">规则预览</h2>
              <span class="badge" :class="selectedEnv?.isProd ? 'badge-warn' : 'badge-info'">
                {{ selectedEnv ? (selectedEnv.isProd ? '标签' : '分支') : '—' }}
              </span>
            </header>

            <div v-if="selectedEnv" class="space-y-3">
              <div>
                <p class="section-label">环境</p>
                <p class="mt-1 text-[13px] font-medium text-ink">{{ selectedEnv.name }}</p>
              </div>
              <div class="divider" />
              <dl class="grid grid-cols-[62px_1fr] gap-x-3 gap-y-2 text-xs">
                <dt class="text-ink-3">构建方式</dt>
                <dd class="text-ink">{{ selectedEnv.isProd ? 'Tag 构建' : '分支构建' }}</dd>
                <dt class="text-ink-3">参数名</dt>
                <dd class="font-mono text-[11px] text-ink">{{ paramName }}</dd>
                <dt class="text-ink-3">默认引用</dt>
                <dd class="font-mono text-[11px] text-ink-2">
                  {{ selectedEnv.isProd ? '必须选择 Tag' : selectedEnv.defaultRef || 'origin/test' }}
                </dd>
              </dl>
              <div class="divider" />
              <p class="text-[11px] leading-5 text-ink-3">
                {{
                  selectedEnv.isProd
                    ? '生产环境必须显式选择版本 Tag，缺失时请求会被拒绝，不会静默回落到测试分支。'
                    : '构建时可留空引用，系统会使用环境默认分支；未带前缀时自动补 origin/。'
                }}
              </p>
            </div>
            <p v-else class="py-3 text-[11px] text-ink-3">选择环境后显示构建规则。</p>
          </section>

          <section v-if="editing" class="card card-pad">
            <h2 class="panel-title">影响面</h2>
            <ul class="mt-3 space-y-2.5">
              <li v-for="item in impacts" :key="item" class="flex gap-2 text-[11px] leading-5 text-ink-2">
                <span class="mt-[7px] h-1 w-1 shrink-0 rounded-full bg-ink-3" />
                {{ item }}
              </li>
            </ul>
          </section>

          <div class="flex items-center justify-end gap-2 border-t border-line pt-3">
            <BaseButton variant="secondary" @click="router.push('/jobs')">取消</BaseButton>
            <BaseButton type="submit" :loading="saving">
              {{ editing ? '保存修改' : '创建任务' }}
            </BaseButton>
          </div>
        </div>
      </aside>
    </form>

    <ConfirmDialog
      v-model="renameOpen"
      title="确认修改任务 ID"
      :message="`把「${originalId}」改为「${form.jobId.trim()}」？`"
      :items="renameItems"
      detail="正在执行与历史记录的内容不受影响。"
      confirm-text="确认修改"
      :loading="saving"
      :danger="false"
      @confirm="doSave"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, TriangleAlert } from 'lucide-vue-next'
import PageHeader from '@/components/layout/PageHeader.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import ConfirmDialog from '@/components/base/ConfirmDialog.vue'
import ErrorState from '@/components/data/ErrorState.vue'
import { jobApi } from '@/api/jobs'
import { envApi } from '@/api/environments'
import { serverApi } from '@/api/servers'
import { useToast } from '@/composables/useToast'
import { required } from '@/utils/validate'

const router = useRouter()
const route = useRoute()
const toast = useToast()

const editing = computed(() => !!route.params.id)
const originalId = ref('')
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const serverError = ref('')

const envs = ref([])
const servers = ref([])
const form = reactive({ jobId: '', envId: '', jobPath: '', serverId: '' })
const errors = reactive({ jobId: '', envId: '', jobPath: '', serverId: '' })

const envOptions = computed(() => envs.value.map((e) => ({ label: `${e.name}（${e.isProd ? 'Tag' : '分支'}）`, value: e.id })))
const serverOptions = computed(() => servers.value.map((s) => ({ label: s.name, value: s.id })))
const selectedEnv = computed(() => envs.value.find((e) => e.id === form.envId) || null)
const paramName = computed(() =>
  selectedEnv.value?.isProd
    ? selectedEnv.value?.defaultTagParam || 'Tags'
    : selectedEnv.value?.defaultBranchParam || 'branch'
)

const endpointPreview = computed(() => {
  const base = servers.value.find((s) => s.id === form.serverId)?.baseUrl || 'https://jenkins.example.com'
  const path = (form.jobPath || '/job/example').trim().replace(/\/+$/, '')
  return `${base.replace(/\/+$/, '')}/${path.replace(/^\/+/, '').replace(/\/build$/, '')}/buildWithParameters`
})

const impacts = computed(() => [
  '钉钉指令需改用新 ID，旧 ID 立即失效',
  '已关联的定时任务与发布历史会在同一事务内同步',
  'Jenkins 端任务路径不变，无需修改 Jenkins 配置',
])

const renameItems = computed(() => impacts.value)
const renameOpen = ref(false)

function validate() {
  errors.jobId = ''
  errors.envId = ''
  errors.jobPath = ''
  errors.serverId = ''
  form.jobId = form.jobId.trim()
  form.jobPath = form.jobPath.trim()

  const a = required(form.jobId, '任务 ID')
  if (a) errors.jobId = a
  else if (/\s/.test(form.jobId)) errors.jobId = '任务 ID 不能包含空格'

  const b = required(form.envId, '环境')
  if (b) errors.envId = b

  const c = required(form.jobPath, 'Jenkins 任务路径')
  if (c) errors.jobPath = c

  const d = required(form.serverId, '所属服务器')
  if (d) errors.serverId = d

  return !errors.jobId && !errors.envId && !errors.jobPath && !errors.serverId
}

function submit() {
  serverError.value = ''
  if (!validate()) return
  if (editing.value && form.jobId !== originalId.value) {
    renameOpen.value = true
    return
  }
  doSave()
}

async function doSave() {
  saving.value = true
  serverError.value = ''
  try {
    const payload = {
      id: form.jobId,
      envId: form.envId,
      jobPath: form.jobPath,
      serverId: form.serverId,
    }
    if (editing.value) {
      await jobApi.update(originalId.value, payload)
      toast.success(form.jobId !== originalId.value ? '任务 ID 已更新，关联记录已同步' : '任务已更新')
    } else {
      await jobApi.create(payload)
      toast.success('任务已创建')
    }
    renameOpen.value = false
    router.push('/jobs')
  } catch (e) {
    serverError.value = e.message || '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  loading.value = true
  const [e, s, j] = await Promise.allSettled([envApi.list(), serverApi.options(), jobApi.list()])
  if (e.status === 'fulfilled') envs.value = e.value || []
  if (s.status === 'fulfilled') servers.value = s.value || []
  if (e.status === 'rejected' || s.status === 'rejected') {
    loadError.value = '环境或 Jenkins 服务加载失败，请确认服务可用后重试'
  }

  if (editing.value) {
    const id = decodeURIComponent(String(route.params.id))
    const job = j.status === 'fulfilled' ? (j.value || []).find((x) => x.id === id) : null
    if (!job) {
      loadError.value = j.status === 'rejected' ? j.reason?.message || '任务加载失败' : `任务「${id}」不存在或已被删除`
    } else {
      originalId.value = job.id
      Object.assign(form, {
        jobId: job.id,
        envId: job.envId,
        jobPath: job.jobPath,
        serverId: job.serverId || 'jenkins',
      })
    }
  } else {
    form.envId = envs.value[0]?.id || ''
    form.serverId = servers.value[0]?.id || 'jenkins'
  }
  loading.value = false
})
</script>