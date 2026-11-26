/** src/api/dingtalk.js — 钉钉通讯录接口 */
import { http } from '@/utils/request'

export const dingtalkApi = {
  searchUsers: (name) => http.get(`/dingtalk/users?name=${encodeURIComponent(name)}`),
  syncUsers: () => http.post('/dingtalk/sync'),
}
