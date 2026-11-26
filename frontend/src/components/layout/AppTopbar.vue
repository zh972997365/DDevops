<template>
  <header class="app-topbar sticky top-0 z-30 h-topbar shrink-0 border-b border-sidebar-line bg-topbar shadow-topbar">
    <div class="flex h-full items-center gap-3 px-4 lg:px-5">
      <button
        class="grid h-8 w-8 place-items-center rounded-[7px] text-sidebar-ink-2 transition-colors hover:bg-sidebar-soft hover:text-sidebar-ink lg:hidden"
        type="button"
        aria-label="打开菜单"
        @click="ui.mobileNav = true"
      >
        <Menu class="h-4 w-4" />
      </button>

      <nav class="flex min-w-0 items-center gap-1.5 text-xs" aria-label="面包屑">
        <template v-for="(c, i) in trail" :key="`${c.label}-${i}`">
          <span v-if="i > 0" class="hidden text-sidebar-ink-2 opacity-60 sm:inline" aria-hidden="true">/</span>
          <RouterLink
            v-if="c.to"
            :to="c.to"
            class="hidden truncate text-sidebar-ink-2 transition-colors hover:text-sidebar-ink sm:inline"
          >
            {{ c.label }}
          </RouterLink>
          <span
            v-else
            class="truncate"
            :class="i === trail.length - 1 ? 'font-medium text-sidebar-ink' : 'hidden text-sidebar-ink-2 sm:inline'"
          >
            {{ c.label }}
          </span>
        </template>
      </nav>

      <div class="ml-auto flex items-center gap-2">
        <button
          class="hidden h-8 max-w-[280px] flex-1 items-center gap-2 rounded-[8px] border border-sidebar-line bg-topbar-field px-2.5 text-left text-[11.5px] text-sidebar-ink-2 transition-colors hover:border-line-strong hover:bg-sidebar-soft hover:text-sidebar-ink md:flex md:w-[216px]"
          type="button"
          aria-label="打开全局搜索"
          @click="ui.commandOpen = true"
        >
          <Search class="h-3.5 w-3.5 shrink-0" />
          <span class="truncate">搜索页面、任务、环境…</span>
          <span class="ml-auto flex shrink-0 items-center gap-0.5" aria-hidden="true">
            <kbd class="top-kbd">⌘</kbd>
            <kbd class="top-kbd">K</kbd>
          </span>
        </button>
        <button
          class="grid h-8 w-8 place-items-center rounded-[7px] text-sidebar-ink-2 transition-colors hover:bg-sidebar-soft hover:text-sidebar-ink md:hidden"
          type="button"
          aria-label="全局搜索"
          @click="ui.commandOpen = true"
        >
          <Search class="h-4 w-4" />
        </button>

        <NotificationBell />

        <button
          class="grid h-8 w-8 place-items-center rounded-[7px] text-sidebar-ink-2 transition-colors hover:bg-sidebar-soft hover:text-sidebar-ink"
          type="button"
          :aria-label="ui.theme === 'dark' ? '切换浅色主题' : '切换深色主题'"
          :title="ui.theme === 'dark' ? '切换浅色主题' : '切换深色主题'"
          @click="ui.toggleTheme()"
        >
          <Sun v-if="ui.theme === 'dark'" class="h-4 w-4" />
          <Moon v-else class="h-4 w-4" />
        </button>

        <span class="mx-0.5 h-5 w-px bg-sidebar-line" aria-hidden="true" />

        <div ref="userRoot" class="relative">
          <button
            class="flex h-8 items-center gap-2 whitespace-nowrap rounded-[8px] px-1.5 transition-colors hover:bg-sidebar-soft"
            type="button"
            :aria-expanded="userOpen"
            aria-haspopup="menu"
            aria-label="用户菜单"
            @click="userOpen = !userOpen"
          >
            <span class="grid h-6 w-6 place-items-center rounded-full bg-sidebar-soft text-[10.5px] font-semibold text-sidebar-ink">
              {{ initial }}
            </span>
            <span class="hidden min-w-0 text-left sm:block">
              <span class="block max-w-[110px] truncate text-[11.5px] font-medium leading-[14px] text-sidebar-ink">
                {{ auth.displayName }}
              </span>
              <span class="mt-0.5 block text-[9.5px] leading-3 text-sidebar-ink-3">
                {{ auth.isAdmin ? '管理员' : '成员' }}
              </span>
            </span>
            <ChevronDown class="h-3 w-3 shrink-0 text-sidebar-ink-3 transition-transform" :class="userOpen && 'rotate-180'" />
          </button>

          <Transition name="menu">
            <div
              v-if="userOpen"
              class="absolute right-0 top-[calc(100%+8px)] z-50 w-[240px] overflow-hidden rounded-card border border-line bg-solid shadow-pop"
              role="menu"
            >
              <div class="border-b border-line px-3.5 py-3">
                <p class="truncate text-[12.5px] font-semibold text-ink">{{ auth.displayName }}</p>
                <p class="mt-0.5 text-[11px] text-ink-3">
                  {{ auth.isAdmin ? '管理员 / 全量项目' : '普通用户 / 按项目授权' }}
                </p>
              </div>

              <div class="p-1">
                <button class="menu-item" type="button" role="menuitem" @click="ui.toggleDensity(); userOpen = false">
                  <Rows3 class="h-3.5 w-3.5" />
                  <span class="flex-1 text-left">紧凑模式</span>
                  <Check v-if="ui.density === 'compact'" class="h-3.5 w-3.5 text-primary" />
                </button>
                <button class="menu-item" type="button" role="menuitem" @click="ui.toggleMotion(); userOpen = false">
                  <Gauge class="h-3.5 w-3.5" />
                  <span class="flex-1 text-left">增强动效</span>
                  <Check v-if="ui.motion === 'rich'" class="h-3.5 w-3.5 text-primary" />
                </button>
                <button class="menu-item" type="button" role="menuitem" @click="ui.toggleTheme(); userOpen = false">
                  <component :is="ui.theme === 'dark' ? Sun : Moon" class="h-3.5 w-3.5" />
                  <span class="flex-1 text-left">{{ ui.theme === 'dark' ? '浅色主题' : '深色主题' }}</span>
                </button>
              </div>

              <div class="border-t border-line p-1">
                <button class="menu-item text-failed hover:bg-failed-soft" type="button" role="menuitem" @click="onLogout">
                  <LogOut class="h-3.5 w-3.5" />
                  <span class="flex-1 text-left">退出登录</span>
                </button>
              </div>
            </div>
          </Transition>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Check, ChevronDown, Gauge, LogOut, Menu, Moon, Rows3, Search, Sun } from 'lucide-vue-next'
import NotificationBell from './NotificationBell.vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useAuth } from '@/composables/useAuth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const { logout } = useAuth()

const userOpen = ref(false)
const userRoot = ref(null)
const initial = computed(() => (auth.displayName || 'U').slice(0, 1).toUpperCase())

const trail = computed(() => {
  const meta = route.meta || {}
  const out = []
  if (meta.crumb) out.push({ label: meta.crumb })
  if (meta.parent) out.push({ label: meta.parent.label, to: meta.parent.to })
  out.push({ label: meta.title || '概括总览' })
  return out
})

async function onLogout() {
  userOpen.value = false
  await logout()
  router.replace('/login')
}
function onDoc(e) {
  if (userOpen.value && userRoot.value && !userRoot.value.contains(e.target)) userOpen.value = false
}
function onKey(e) {
  if (e.key === 'Escape') userOpen.value = false
}

onMounted(() => {
  document.addEventListener('mousedown', onDoc)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDoc)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.app-topbar {
}
.top-kbd {
  display: inline-grid;
  height: 18px;
  min-width: 18px;
  place-items: center;
  border: 1px solid var(--sidebar-line);
  border-radius: 4px;
  background: var(--topbar-field);
  padding: 0 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 9.5px;
  color: var(--sidebar-ink-2);
}
.menu-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  padding: 8px;
  font-size: 12px;
  color: var(--ink-2);
  transition:
    background-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}
.menu-item:hover {
  background: var(--surface-sunken);
  color: var(--ink);
}
.menu-enter-active,
.menu-leave-active {
  transition: opacity 140ms var(--ease), transform 140ms var(--ease);
  transform-origin: top right;
}
.menu-enter-from,
.menu-leave-to {
  transform: translateY(-4px) scale(0.985);
}
</style>