/** src/api/jobs.js — Jenkins 任务接口 */
import { http } from '@/utils/request'

const url = (id) => `/jobs/${encodeURIComponent(id)}`

export const jobApi = {
  list: () => http.get('/jobs'),
  create: (job) => http.post('/jobs', job),
  update: (id, job) => http.put(url(id), job),
  remove: (id) => http.del(url(id)),
  build: (id, refValue) => http.post(`${url(id)}/build`, { refValue }),
  gitTags: (id) => http.get(`${url(id)}/git-tags`),
}
