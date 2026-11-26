<template>
  <Teleport to="body">
    <Transition name="palette">
      <div v-if="modelValue" class="palette-root" @click.self="close">
        <div class="palette" role="dialog" aria-modal="true" aria-label="全局搜索">
          <div class="flex h-[54px] items-center gap-2.5 border-b border-line px-4">
            <Search class="h-4 w-4 shrink-0 text-ink-3" />
            <input
              ref="inputEl"
              v-model="query"
              type="text"
              class="min-w-0 flex-1 bg-transparent text-[14px] text-ink outline-none placeholder:text-ink-3"
              placeholder="搜索页面、任务、环境…"
              aria-label="全局搜索"
              @keydown.down.prevent="move(1)"
              @keydown.up.prevent="move(-1)"
              @keydown.enter.prevent="choose"
              @keydown.esc.prevent="close"
            />
            <kbd class="kbd">ESC</kbd>
          </div>

          <div ref="listEl" class="min-h-0 flex-1 overflow-y-auto px-2 py-2">
            <template v-for="block in blocks" :key="block.group">
              <p class="section-label px-2.5 pb-1 pt-2.5">{{ block.group }}</p>
              <button
                v-for="item in block.items"
                :key="item.id"
                :ref="(el) => setRef(item.id, el)"
                type="button"
                class="result group"
                :class="{ 'is-active': activeId === item.id }"
                @pointermove="activeId = item.id"
                @click="run(item)"
              >
                <span class="result-icon">
                  <component :is="item.icon" class="h-3.5 w-3.5" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-[12.5px] font-medium text-ink" v-html="highlight(item.title)" />
                  <span
                    v-if="item.sub"
                    class="mt-0.5 block truncate text-[11px] leading-4 text-ink-3"
                    v-html="highlight(item.sub)"
                  />
                </span>
                <span v-if="item.meta" class="shrink-0 text-[10px] text-ink-3">
                  {{ item.meta }}
                </span>
                <CornerDownLeft v-if="activeId === item.id" class="h-3 w-3 shrink-0 text-ink-3" />
              </button>
            </template>

            <div v-if="!flat.length" class="flex flex-col items-center px-4 py-12 text-center">
              <span class="grid h-10 w-10 place-items-center rounded-full border border-dashed border-line-strong text-ink-3">
                <SearchX class="h-4 w-4" />
              </span>
              <p class="mt-3 text-xs font-medium text-ink">没有匹配项</p>
              <p class="mt-1 text-[11px] text-ink-3">换一个关键词，或输入完整的任务 ID</p>
            </div>
          </div>

          <div class="flex items-center gap-4 border-t border-line bg-sunken px-4 py-2 text-[10.5px] text-ink-3">
            <span class="flex items-center gap-1"><kbd class="kbd">↑</kbd><kbd class="kbd">↓</kbd>选择</span>
            <span class="flex items-center gap-1"><kbd class="kbd">↵</kbd>打开</span>
            <span class="flex items-center gap-1"><kbd class="kbd">ESC</kbd>关闭</span>
            <span class="ml-auto text-[10px]">⌘K 随时打开</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  CornerDownLeft,
  Layers3,
  Moon,
  Plus,
  Rows3,
  Search,
  SearchX,
  Sun,
  Timer,
  Workflow,
} from 'lucide-vue-next'
import { jobApi } from '@/api/jobs'
import { envApi } from '@/api/environments'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { navFor } from '@/constants/nav'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue'])

const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()

const query = ref('')
const activeId = ref('')
const inputEl = ref(null)
const listEl = ref(null)
const itemEls = new Map()
const jobs = ref([])
const envs = ref([])
let loaded = false

const PAGES = computed(() =>
  navFor(auth.isAdmin).map((g) => g.items).flat().map((n) => ({
    id: `page:${n.to}`,
    group: '页面',
    title: n.label,
    sub: n.caption,
    icon: n.icon,
    meta: '页面',
    run: () => router.push(n.to),
  }))
)

const ACTIONS = computed(() => {
  const out = []
  if (auth.isAdmin) {
    out.push({
      id: 'act:new-job',
      group: '快捷操作',
      title: '新建任务',
      sub: '把 Jenkins 流水线接入发布平台',
      icon: Plus,
      meta: '命令',
      run: () => router.push('/jobs/new'),
    })
  }
  out.push(
    {
      id: 'act:new-schedule',
      group: '快捷操作',
      title: '新建定时任务',
      sub: '在指定时间自动触发发布',
      icon: Timer,
      meta: '命令',
      run: () => router.push('/tasks?new=1'),
    },
    {
      id: 'act:theme',
      group: '快捷操作',
      title: ui.theme === 'dark' ? '切换浅色主题' : '切换深色主题',
      sub: '界面明暗随环境调整',
      icon: ui.theme === 'dark' ? Sun : Moon,
      meta: '命令',
      run: () => ui.toggleTheme(),
    },
    {
      id: 'act:density',
      group: '快捷操作',
      title: ui.density === 'compact' ? '切换舒适密度' : '切换紧凑密度',
      sub: '调整表格与卡片的留白',
      icon: Rows3,
      meta: '命令',
      run: () => ui.toggleDensity(),
    }
  )
  return out
})

const JOB_ITEMS = computed(() => {
  const envName = Object.fromEntries(envs.value.map((e) => [e.id, e.name]))
  return jobs.value.map((j) => ({
    id: `job:${j.id}`,
    group: '任务',
    title: j.id,
    sub: `${envName[j.envId] || '未配置环境'} / ${j.jobPath}`,
    icon: Workflow,
    meta: '任务',
    run: () => router.push({ path: '/jobs', query: { q: j.id } }),
  }))
})

const ENV_ITEMS = computed(() =>
  envs.value.map((e) => ({
    id: `env:${e.id}`,
    group: '环境',
    title: e.name,
    sub: e.isProd ? 'Tag 构建 / 生产环境' : `分支构建 / ${e.defaultRef || 'origin/test'}`,
    icon: Layers3,
    meta: e.isProd ? '生产' : '环境',
    run: () => router.push({ path: '/jobs', query: { env: e.id } }),
  }))
)

const all = computed(() => [...PAGES.value, ...ACTIONS.value, ...JOB_ITEMS.value, ...ENV_ITEMS.value])

const flat = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) {
    return [...PAGES.value, ...ACTIONS.value, ...JOB_ITEMS.value.slice(0, 6)]
  }
  return all.value
    .map((item) => {
      const title = item.title.toLowerCase()
      const sub = (item.sub || '').toLowerCase()
      let score = 0
      if (title === q) score += 140
      else if (title.startsWith(q)) score += 96
      else if (title.includes(q)) score += 60
      if (sub.includes(q)) score += 20
      if (item.group === '页面') score += 4
      return { item, score }
    })
    .filter((r) => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, 16)
    .map((r) => r.item)
})

const blocks = computed(() => {
  const order = ['页面', '快捷操作', '任务', '环境']
  const map = new Map()
  flat.value.forEach((i) => {
    if (!map.has(i.group)) map.set(i.group, [])
    map.get(i.group).push(i)
  })
  return order.filter((g) => map.has(g)).map((g) => ({ group: g, items: map.get(g) }))
})

const activeIndex = computed(() => flat.value.findIndex((i) => i.id === activeId.value))

watch(query, () => {
  activeId.value = flat.value[0]?.id || ''
})
watch(
  () => props.modelValue,
  async (open) => {
    document.body.style.overflow = open ? 'hidden' : ''
    if (!open) return
    query.value = ''
    activeId.value = ''
    await nextTick()
    inputEl.value?.focus()
    await ensureData()
    if (!activeId.value) activeId.value = flat.value[0]?.id || ''
  }
)
watch(flat, (v) => {
  if (!v.some((i) => i.id === activeId.value)) activeId.value = v[0]?.id || ''
})

async function ensureData() {
  if (loaded) return
  loaded = true
  const [j, e] = await Promise.allSettled([jobApi.list(), envApi.list()])
  if (j.status === 'fulfilled') jobs.value = j.value || []
  if (e.status === 'fulfilled') envs.value = e.value || []
}

function setRef(id, el) {
  if (el) itemEls.set(id, el)
  else itemEls.delete(id)
}

function move(delta) {
  if (!flat.value.length) return
  const next = (Math.max(0, activeIndex.value) + delta + flat.value.length) % flat.value.length
  activeId.value = flat.value[next].id
  nextTick(() => itemEls.get(activeId.value)?.scrollIntoView({ block: 'nearest' }))
}

function choose() {
  const item = flat.value.find((i) => i.id === activeId.value)
  if (item) run(item)
}

function run(item) {
  close()
  setTimeout(() => item.run?.(), 60)
}
function close() {
  emit('update:modelValue', false)
}

function escapeHtml(text) {
  return String(text).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c])
}
function highlight(text) {
  const q = query.value.trim()
  const raw = escapeHtml(text)
  if (!q) return raw
  const idx = String(text).toLowerCase().indexOf(q.toLowerCase())
  if (idx < 0) return raw
  return `${escapeHtml(text.slice(0, idx))}<mark class="rounded-[2px] bg-primary-soft px-0.5 text-primary-ink">${escapeHtml(
    text.slice(idx, idx + q.length)
  )}</mark>${escapeHtml(text.slice(idx + q.length))}`
}
</script>

<style scoped>
.palette-root {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  justify-content: center;
  padding: 12vh 16px 24px;
  background: var(--overlay);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}
.palette {
  display: flex;
  width: min(660px, 100%);
  max-height: min(620px, 74vh);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: var(--surface-solid);
  box-shadow: var(--shadow-pop);
  animation: pop-in 160ms var(--ease-out) both;
}
.result {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  padding: 8px 12px;
  text-align: left;
  transition: background-color var(--dur-fast) var(--ease);
}
.result.is-active {
  background: var(--primary-soft);
}
.result-icon {
  display: grid;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 7px;
  background: var(--surface-solid);
  color: var(--ink-3);
  transition:
    color var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}
.result.is-active .result-icon {
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
  color: var(--primary-ink);
}
.palette-enter-active .palette {
  animation: pop-in 160ms var(--ease-out) both;
}
</style>