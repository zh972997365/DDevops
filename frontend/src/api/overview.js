/** src/api/overview.js — 首页统计与最近构建 */
import { http } from '@/utils/request'

export const overviewApi = {
  get: () => http.get('/overview'),
}
