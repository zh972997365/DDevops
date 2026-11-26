/** src/api/history.js — 发布历史（后端分页） */
import { http } from '@/utils/request'

export const historyApi = {
  list: (query = {}) => {
    const q = new URLSearchParams()
    Object.entries(query).forEach(([k, v]) => {
      if (v !== '' && v !== undefined && v !== null) q.set(k, v)
    })
    return http.get(`/history?${q.toString()}`)
  },
  stats: () => http.get('/history/stats'),
}
