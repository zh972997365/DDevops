/** src/constants/nav.js — 全站导航唯一来源（侧栏 / 命令面板共用） */
import { Clock3, History, Layers3, LayoutDashboard, ServerCog, UsersRound, Workflow } from 'lucide-vue-next'

export const NAV_GROUPS = [
  {
    label: '工作台',
    items: [
      { to: '/', label: '概括总览', caption: '发布态势与待办', icon: LayoutDashboard },
      { to: '/jobs', label: '任务管理', caption: '流水线、环境与构建', icon: Workflow },
      { to: '/tasks', label: '定时任务', caption: '计划发布与倒计时', icon: Clock3 },
      { to: '/history', label: '发布历史', caption: '触发记录与结果', icon: History },
    ],
  },
  {
    label: '配置',
    items: [
      { to: '/environments', label: '环境管理', caption: '构建规则与参数名', icon: Layers3, admin: true },
      { to: '/servers', label: '服务管理', caption: 'Jenkins 实例连接', icon: ServerCog, admin: true },
      { to: '/permissions', label: '用户管理', caption: '账号、角色与项目授权', icon: UsersRound, admin: true },
    ],
  },
]

export function navFor(isAdmin) {
  return NAV_GROUPS.map((g) => ({
    ...g,
    items: g.items.filter((i) => !i.admin || isAdmin),
  })).filter((g) => g.items.length)
}

export function isNavActive(path, to) {
  if (to === '/') return path === '/'
  return path === to || path.startsWith(`${to}/`)
}

export function flatNav(isAdmin) {
  return navFor(isAdmin).flatMap((g) => g.items)
}