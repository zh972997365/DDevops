/** src/api/auth.js — 登录相关接口 */
import { http } from '@/utils/request'

export const authApi = {
  login: (username, password) => http.post('/login', { username, password }),
  logout: () => http.post('/logout'),
}
