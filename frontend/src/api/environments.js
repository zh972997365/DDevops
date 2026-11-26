/** src/api/environments.js — 环境接口 */
import { http } from '@/utils/request'

const url = (id) => `/environments/${encodeURIComponent(id)}`

export const envApi = {
  list: () => http.get('/environments'),
  create: (env) => http.post('/environments', env),
  update: (id, env) => http.put(url(id), env),
  remove: (id) => http.del(url(id)),
  reorder: (ids) => http.put('/environments/reorder', { ids }),
}
