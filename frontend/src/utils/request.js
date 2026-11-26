/** src/utils/request.js — 唯一网络入口：token、401 跳转、统一错误 */
import { useAuthStore } from '@/stores/auth'

const BASE = '/api'

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function request(method, path, body) {
  const auth = useAuthStore()
  const headers = {}
  if (auth.token) headers.Authorization = auth.token

  let payload
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  let resp
  try {
    resp = await fetch(BASE + path, { method, headers, body: payload })
  } catch (e) {
    throw new ApiError('网络连接失败，请检查服务是否运行', 0)
  }

  if (resp.status === 401) {
    auth.logout()
    // 交给 main.js 里的监听器跳转，避免 request 反向依赖 router 形成循环引用
    window.dispatchEvent(new Event('auth:expired'))
    throw new ApiError('登录已过期，请重新登录', 401)
  }

  const text = await resp.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = null
    }
  }

  if (!resp.ok) {
    throw new ApiError(data?.error || `请求失败（${resp.status}）`, resp.status)
  }
  return data
}

export const http = {
  get: (p) => request('GET', p),
  post: (p, b) => request('POST', p, b),
  put: (p, b) => request('PUT', p, b),
  del: (p) => request('DELETE', p),
}
