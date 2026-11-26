/** src/constants/status.js — 状态文案与色调唯一来源 */

export const STATUS_MAP = {
  pending: { label: '待执行', tone: 'pending' },
  executing: { label: '进行中', tone: 'running' },
  running: { label: '进行中', tone: 'running' },
  completed: { label: '已完成', tone: 'success' },
  success: { label: '成功', tone: 'success' },
  failed: { label: '失败', tone: 'failed' },
  cancelled: { label: '已取消', tone: 'cancelled' },
}

export const TONE_BADGE = {
  pending: 'badge-neutral',
  running: 'badge-warn',
  success: 'badge-success',
  failed: 'badge-failed',
  cancelled: 'badge-cancelled',
  neutral: 'badge-neutral',
  info: 'badge-info',
}

export const TONE_DOT = {
  pending: 'bg-pending',
  running: 'bg-warn',
  success: 'bg-success',
  failed: 'bg-failed',
  cancelled: 'bg-cancelled',
  neutral: 'bg-pending',
  info: 'bg-info',
}

export function statusOf(status) {
  return STATUS_MAP[status] || { label: status || '—', tone: 'neutral' }
}