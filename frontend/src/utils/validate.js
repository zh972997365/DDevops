/** src/utils/validate.js — 校验规则集中定义，返回错误文案（空串表示通过） */

export const required = (v, label = '此项') => (String(v ?? '').trim() ? '' : `${label}不能为空`)

export const maxLen = (v, n) => (String(v ?? '').length <= n ? '' : `长度不能超过 ${n} 个字符`)

export const tagFormat = (v) => {
  const s = String(v ?? '').trim()
  if (!s) return ''
  return /^[A-Za-z0-9._/-]+$/.test(s) ? '' : '只能包含字母、数字、点、横线、斜杠'
}

export const urlFormat = (v) => {
  const s = String(v ?? '').trim()
  if (!s) return ''
  return /^https?:\/\/[^\s]+$/.test(s) ? '' : '需以 http:// 或 https:// 开头'
}

export function firstError(rules) {
  for (const msg of rules) {
    if (msg) return msg
  }
  return ''
}
