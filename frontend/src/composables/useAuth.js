/** src/composables/useAuth.js — 登录 / 登出动作 */
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'

export function useAuth() {
  const auth = useAuthStore()

  async function login(username, password) {
    const res = await authApi.login(username, password)
    auth.setAuth(res.token, { username, role: res.role })
    return res
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {
      /* 服务异常也要清本地状态 */
    }
    auth.logout()
  }

  return { login, logout }
}
