/** src/api/servers.js — Jenkins 服务接口 */
import { http } from '@/utils/request'

const url = (id) => `/servers/${encodeURIComponent(id)}`

export const serverApi = {
  list: () => http.get('/servers'),
  // 精简列表（仅 id/name），所有登录用户可读，用于任务列表展示所属服务器
  options: () => http.get('/servers/options'),
  create: (server) => http.post('/servers', server),
  update: (id, server) => http.put(url(id), server),
  remove: (id) => http.del(url(id)),
  reorder: (ids) => http.put('/servers/reorder', { ids }),
}
