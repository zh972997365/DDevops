/** src/api/users.js — 用户与权限接口 */
import { http } from '@/utils/request'

const url = (id) => `/permissions/${encodeURIComponent(id)}`

export const userApi = {
  list: () => http.get('/permissions'),
  create: (user) => http.post('/permissions', user),
  update: (id, user) => http.put(url(id), user),
  remove: (id) => http.del(url(id)),
}
