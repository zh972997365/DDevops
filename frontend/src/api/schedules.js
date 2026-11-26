/** src/api/schedules.js — 定时任务接口 */
import { http } from '@/utils/request'

const url = (id) => `/tasks/${encodeURIComponent(id)}`

export const scheduleApi = {
  list: (status = '') => http.get(`/tasks${status ? `?status=${encodeURIComponent(status)}` : ''}`),
  create: (payload) => http.post('/tasks', payload),
  cancel: (id) => http.post(`${url(id)}/cancel`),
  remove: (id) => http.del(url(id)),
}
