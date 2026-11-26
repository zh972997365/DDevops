/** src/stores/ui.js — 主题 / 信息密度 / 侧栏 / 全局搜索 */
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const LS_THEME = 'devops_theme'
const LS_DENSITY = 'devops_density'
const LS_COLLAPSE = 'devops_nav_collapsed'
const LS_MOTION = 'devops_motion'

export const useUiStore = defineStore('ui', () => {
  const theme = ref(localStorage.getItem(LS_THEME) === 'dark' ? 'dark' : 'light')
  const density = ref(localStorage.getItem(LS_DENSITY) === 'compact' ? 'compact' : 'comfortable')
  const collapsed = ref(localStorage.getItem(LS_COLLAPSE) === '1')
  const motion = ref(localStorage.getItem(LS_MOTION) === 'rich' ? 'rich' : 'standard')
  const mobileNav = ref(false)
  const commandOpen = ref(false)

  function apply() {
    const el = document.documentElement
    el.classList.toggle('dark', theme.value === 'dark')
    el.classList.toggle('light', theme.value !== 'dark')
    el.classList.toggle('density-compact', density.value === 'compact')
    el.classList.toggle('motion-rich', motion.value === 'rich')
    const meta = document.querySelector('meta[name="theme-color"]')
    if (meta) meta.setAttribute('content', theme.value === 'dark' ? '#171d22' : '#f4f6f7')
  }

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  function toggleDensity() {
    density.value = density.value === 'compact' ? 'comfortable' : 'compact'
  }

  function toggleCollapse() {
    collapsed.value = !collapsed.value
  }

  function toggleMotion() {
    motion.value = motion.value === 'rich' ? 'standard' : 'rich'
  }

  watch(theme, (v) => {
    localStorage.setItem(LS_THEME, v)
    apply()
  })
  watch(density, (v) => {
    localStorage.setItem(LS_DENSITY, v)
    apply()
  })
  watch(motion, (v) => {
    localStorage.setItem(LS_MOTION, v)
    apply()
  })
  watch(collapsed, (v) => localStorage.setItem(LS_COLLAPSE, v ? '1' : '0'))

  apply()

  return {
    theme,
    density,
    motion,
    collapsed,
    mobileNav,
    commandOpen,
    toggleTheme,
    toggleDensity,
    toggleMotion,
    toggleCollapse,
  }
})