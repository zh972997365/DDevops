/** src/stores/auth.js — 登录态与权限判定 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

const LS_TOKEN = 'devops_token'
const LS_USER = 'devops_user'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(LS_TOKEN) || '')
  const user = ref(JSON.parse(localStorage.getItem(LS_USER) || 'null'))

  const isAuthed = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const displayName = computed(() => user.value?.username || '未登录')

  function setAuth(t, u) {
    token.value = t
    user.value = u
    localStorage.setItem(LS_TOKEN, t)
    localStorage.setItem(LS_USER, JSON.stringify(u))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem(LS_TOKEN)
    localStorage.removeItem(LS_USER)
  }

  return { token, user, isAuthed, isAdmin, displayName, setAuth, logout }
})
