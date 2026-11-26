<template>
  <aside
    class="app-sidebar w-[260px]"
    :class="[
      collapsed ? 'lg:w-sidebar-sm' : 'lg:w-sidebar',
      mobileOpen ? 'max-lg:translate-x-0' : 'max-lg:-translate-x-full',
    ]"
  >
    <div
      class="flex h-topbar shrink-0 items-center gap-2.5 border-b border-sidebar-line px-4"
      :class="!showText && 'justify-center px-0'"
    >
      <RouterLink to="/" class="flex min-w-0 items-center gap-2.5" aria-label="DDevOps 概括总览">
        <span class="brand-logo" aria-hidden="true"><img src="/brand-mark.svg" alt="" /></span>
        <span v-if="showText" class="min-w-0">
          <span class="block truncate text-[13.5px] font-semibold leading-4 tracking-[-0.01em] text-sidebar-ink">
            DDevOps
          </span>
          <span class="mt-1 block truncate text-[10.5px] leading-none text-sidebar-ink-3">自动化发布平台</span>
        </span>
      </RouterLink>

      <button
        v-if="showText"
        class="ml-auto grid h-7 w-7 place-items-center rounded-[6px] text-sidebar-ink-2 transition-colors hover:bg-sidebar-soft hover:text-sidebar-ink lg:hidden"
        type="button"
        aria-label="关闭菜单"
        @click="ui.mobileNav = false"
      >
        <X class="h-4 w-4" />
      </button>
    </div>

    <nav ref="navEl" class="relative flex-1 overflow-y-auto overflow-x-hidden px-2.5 py-4" aria-label="主导航">
      <span
        class="nav-indicator"
        :class="{ 'is-ready': ready }"
        :style="{
          transform: `translate(${left}px, ${top}px)`,
          width: `${width}px`,
          height: `${height}px`,
        }"
        aria-hidden="true"
      />

      <div v-for="(group, gi) in groups" :key="group.label" :class="gi > 0 && 'mt-6'">
        <p
          v-if="showText"
          class="mb-2 px-2.5 text-[10.5px] font-medium text-sidebar-ink-3"
        >
          {{ group.label }}
        </p>
        <div v-else-if="gi > 0" class="mx-auto mb-2 h-px w-6 bg-sidebar-line" />

        <div class="flex flex-col gap-0.5">
          <RouterLink
            v-for="item in group.items"
            :key="item.to"
            :ref="(el) => setItemRef(item.to, el)"
            :to="item.to"
            class="nav-item"
            :class="{
              'is-active': isNavActive(route.path, item.to),
              'is-icon-only': !showText,
            }"
            :title="!showText ? item.label : undefined"
          >
            <component :is="item.icon" class="h-4 w-4 shrink-0" />
            <span v-if="showText" class="truncate">{{ item.label }}</span>
          </RouterLink>
        </div>
      </div>
    </nav>

    <div class="shrink-0 border-t border-sidebar-line p-2.5">
      <div class="flex items-center gap-2.5 rounded-[8px] px-1.5 py-1.5" :class="!showText && 'justify-center px-0'">
        <span class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-sidebar-soft text-[12px] font-semibold text-sidebar-ink">
          {{ initial }}
        </span>
        <div v-if="showText" class="min-w-0 flex-1">
          <p class="truncate text-[12.5px] font-medium leading-4 text-sidebar-ink">{{ auth.displayName }}</p>
          <p class="mt-0.5 truncate text-[10.5px] leading-none text-sidebar-ink-2">
            {{ auth.isAdmin ? '管理员 / 全量权限' : '普通用户 / 项目授权' }}
          </p>
        </div>
        <button
          v-if="showText"
          class="grid h-7 w-7 shrink-0 place-items-center rounded-[6px] text-sidebar-ink-3 transition-colors hover:bg-sidebar-soft hover:text-sidebar-ink"
          type="button"
          aria-label="退出登录"
          title="退出登录"
          @click="onLogout"
        >
          <LogOut class="h-3.5 w-3.5" />
        </button>
      </div>

      <button
        class="mt-1 hidden h-8 w-full items-center justify-center gap-1.5 rounded-[6px] text-[11px] text-sidebar-ink-2 transition-colors hover:bg-sidebar-soft hover:text-sidebar-ink lg:flex"
        type="button"
        :aria-label="collapsed ? '展开侧栏' : '收起侧栏'"
        :title="collapsed ? '展开侧栏' : '收起侧栏'"
        @click="ui.toggleCollapse()"
      >
        <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="h-3.5 w-3.5" />
        <span v-if="showText">收起侧栏</span>
      </button>
    </div>
  </aside>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut, PanelLeftClose, PanelLeftOpen, X } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useAuth } from '@/composables/useAuth'
import { isNavActive, navFor } from '@/constants/nav'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const { logout } = useAuth()

const groups = computed(() => navFor(auth.isAdmin))
const mobileOpen = computed(() => ui.mobileNav)
const collapsed = computed(() => ui.collapsed)
const isMobile = ref(window.matchMedia('(max-width: 1023px)').matches)
const showText = computed(() => !collapsed.value || isMobile.value)
const initial = computed(() => (auth.displayName || 'U').slice(0, 1).toUpperCase())

const navEl = ref(null)
const items = new Map()
const top = ref(0)
const left = ref(10)
const width = ref(0)
const height = ref(38)
const ready = ref(false)

function setItemRef(to, el) {
  const node = el?.$el || el
  if (node) items.set(to, node)
  else items.delete(to)
}

// 滑块几何直接取自当前选中行，避免折叠/展开过渡期间的测量偏差
function measure() {
  const node = navEl.value
  if (!node) return
  const active = [...items.entries()].find(([to]) => isNavActive(route.path, to))?.[1]
  if (!active) {
    height.value = 0
    return
  }
  top.value = active.offsetTop
  left.value = active.offsetLeft
  width.value = active.offsetWidth
  height.value = active.offsetHeight
  if (!ready.value) requestAnimationFrame(() => (ready.value = true))
}

// DOM 变化（标签显隐、字体加载、折叠过渡）后重新对齐，避免滑块与行错位
let rafId = 0
function scheduleMeasure() {
  cancelAnimationFrame(rafId)
  rafId = requestAnimationFrame(() => requestAnimationFrame(measure))
}

let observer = null
watch(
  () => [route.path, collapsed.value, isMobile.value, auth.isAdmin],
  () => nextTick(scheduleMeasure)
)

const mq = window.matchMedia('(max-width: 1023px)')
function onMq(e) {
  isMobile.value = e.matches
  nextTick(scheduleMeasure)
}
function onResize() {
  scheduleMeasure()
}
function onTransitionEnd(e) {
  if (e.target === e.currentTarget) scheduleMeasure()
}

onMounted(() => {
  mq.addEventListener('change', onMq)
  window.addEventListener('resize', onResize)
  window.addEventListener('load', scheduleMeasure)
  navEl.value?.addEventListener('transitionend', onTransitionEnd)
  // 侧栏宽度过渡、标签显隐都会改变行几何，用 ResizeObserver 兜底
  if (typeof ResizeObserver !== 'undefined' && navEl.value) {
    observer = new ResizeObserver(scheduleMeasure)
    observer.observe(navEl.value)
  }
  nextTick(scheduleMeasure)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(rafId)
  mq.removeEventListener('change', onMq)
  window.removeEventListener('resize', onResize)
  window.removeEventListener('load', scheduleMeasure)
  navEl.value?.removeEventListener('transitionend', onTransitionEnd)
  observer?.disconnect()
})

async function onLogout() {
  await logout()
  router.replace('/login')
}
</script>

<style scoped>
.app-sidebar {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 40;
  display: flex;
  flex-direction: column;
  background: var(--sidebar);
  border-right: 1px solid var(--sidebar-line);
  transition: transform 220ms var(--ease);
}
@media (min-width: 1024px) {
  .app-sidebar {
    position: fixed;
  }
}

.nav-indicator {
  position: absolute;
  top: 0;
  left: 0;
  z-index: 0;
  border: 1px solid var(--sidebar-active-border);
  border-radius: 8px;
  background: var(--sidebar-active-bg);
  box-shadow: var(--shadow-control);
  opacity: 0;
  pointer-events: none;
}
.dark .nav-indicator {
  box-shadow: none;
}
.nav-indicator::before {
  content: '';
  position: absolute;
  left: 4px;
  top: 50%;
  width: 2px;
  height: 12px;
  transform: translateY(-50%);
  border-radius: 2px;
  background: var(--accent);
}
.nav-indicator.is-ready {
  opacity: 1;
  transition:
    transform 380ms var(--ease-spring),
    opacity 160ms var(--ease);
}

.nav-item {
  position: relative;
  z-index: 1;
  display: flex;
  height: 36px;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  padding: 0 12px;
  font-size: 12.5px;
  color: var(--sidebar-ink-2);
  transition:
    color var(--dur-fast) var(--ease),
    background-color var(--dur-fast) var(--ease);
}
.nav-item:hover {
  color: var(--sidebar-ink);
  background: var(--sidebar-hover-bg);
}
.nav-item:active {
  transform: scale(0.995);
}
.nav-item.is-active {
  color: var(--primary-ink);
  font-weight: 600;
}
.nav-item.is-active svg {
  color: var(--accent);
}
.dark .nav-item.is-active {
  color: var(--ink);
}
.nav-item.is-icon-only {
  justify-content: center;
  padding: 0;
}
</style>