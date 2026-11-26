/** src/constants/role.js — 角色映射与受保护账号 */

export const ROLE_MAP = {
  admin: { label: '管理员', tone: 'primary' },
  user: { label: '普通用户', tone: 'neutral' },
}

export const PROTECTED_USER_ID = 'admin'

export function roleOf(role) {
  return ROLE_MAP[role] || ROLE_MAP.user
}
