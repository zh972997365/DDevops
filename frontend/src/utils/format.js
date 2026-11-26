/** src/utils/format.js — 时间 / 数字格式化（全站唯一实现） */

const pad = (n) => String(n).padStart(2, '0')

export function toDate(value) {
  if (!value) return null
  const d = value instanceof Date ? value : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDateTime(value) {
  const d = toDate(value)
  if (!d) return '-'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function relativeTime(value) {
  const d = toDate(value)
  if (!d) return '-'
  const diff = Date.now() - d.getTime()
  if (diff < 0) return '即将'
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min} 分钟前`
  const hour = Math.floor(min / 60)
  if (hour < 24) return `${hour} 小时前`
  const day = Math.floor(hour / 24)
  if (day < 30) return `${day} 天前`
  return formatDateTime(d).slice(0, 10)
}

export function formatNumber(n) {
  const v = Number(n) || 0
  return v.toLocaleString('zh-CN')
}

export function formatPercent(v) {
  return `${Math.round((Number(v) || 0) * 100)}%`
}

/** 占比百分比（0–100），分母为空时为 0；用于统计卡底部的占比条 */
export function ratioPercent(n, d) {
  const total = Number(d) || 0
  if (total <= 0) return 0
  return Math.min(100, Math.max(0, Math.round(((Number(n) || 0) / total) * 100)))
}

/** 倒计时：返回 {text, urgent}，urgent 表示 1 小时内 */
export function countdownText(target) {
  const d = toDate(target)
  if (!d) return { text: '-', urgent: false }
  const diff = d.getTime() - Date.now()
  if (diff <= 0) return { text: '已到时间', urgent: true }
  const total = Math.floor(diff / 1000)
  const day = Math.floor(total / 86400)
  const hour = Math.floor((total % 86400) / 3600)
  const min = Math.floor((total % 3600) / 60)
  const sec = total % 60
  const text =
    day > 0
      ? `${day} 天 ${pad(hour)}:${pad(min)}`
      : `${pad(hour)}:${pad(min)}:${pad(sec)}`
  return { text, urgent: diff < 3600 * 1000 }
}

/** Jenkins 错误可能带回整页 HTML，列表与抽屉只展示可读摘要 */
export function shortError(value, max = 180) {
  const text = String(value || '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  if (!text) return ''
  return text.length > max ? `${text.slice(0, max)}…` : text
}
