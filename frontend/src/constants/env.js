/** src/constants/env.js — 环境展示与色调 */
export function envLabel(env) {
  return env?.name || '未配置环境'
}

export function envTone(env) {
  if (!env) return 'neutral'
  return env.isProd ? 'warn' : 'info'
}

export function envMode(env) {
  if (!env) return '未配置'
  return env.isProd ? 'Tag 构建' : '分支构建'
}