<template>
  <div class="min-h-screen">
    <AppSidebar />

    <div
      class="min-h-screen"
      :class="ui.collapsed ? 'lg:pl-sidebar-sm' : 'lg:pl-sidebar'"
    >
      <AppTopbar />
      <main class="page-shell">
        <RouterView v-slot="{ Component, route }">
          <Transition name="page" mode="out-in">
            <component :is="Component" :key="route.path" />
          </Transition>
        </RouterView>
      </main>
    </div>

    <Transition name="fade">
      <div
        v-if="ui.mobileNav"
        class="fixed inset-0 z-[35] bg-[color:var(--overlay)] lg:hidden"
        aria-hidden="true"
        @click="ui.mobileNav = false"
      />
    </Transition>

    <CommandPalette v-model="ui.commandOpen" />
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from './AppSidebar.vue'
import AppTopbar from './AppTopbar.vue'
import CommandPalette from './CommandPalette.vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const route = useRoute()

function onKey(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    ui.commandOpen = !ui.commandOpen
  }
  if (e.key === 'Escape' && ui.mobileNav) ui.mobileNav = false
}

watch(
  () => route.fullPath,
  () => {
    ui.mobileNav = false
    ui.commandOpen = false
  }
)

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<style scoped>
.page-enter-active,
.page-leave-active {
  transition: opacity 150ms var(--ease), transform 150ms var(--ease);
}
.page-enter-from {
  transform: translateY(4px);
}
.page-leave-to {
  opacity: 0;
}
.fade-enter-active,
.fade-leave-active {
  transition: background-color 180ms var(--ease);
}
.fade-enter-from,
.fade-leave-to {
  background: transparent;
}
</style>