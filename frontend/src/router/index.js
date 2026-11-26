/** src/router/index.js — 路由、登录与管理员守卫 */
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { public: true, title: '登录' },
  },
  {
    path: '/',
    component: AppLayout,
    children: [
      { path: '', name: 'overview', component: () => import('@/views/Overview.vue'), meta: { title: '概括总览', crumb: '工作台' } },
      { path: 'jobs', name: 'jobs', component: () => import('@/views/Jobs.vue'), meta: { title: '任务管理', crumb: '发布' } },
      {
        path: 'jobs/new',
        name: 'job-create',
        component: () => import('@/views/JobForm.vue'),
        meta: { title: '新建任务', crumb: '发布', parent: { label: '任务管理', to: '/jobs' }, admin: true },
      },
      {
        path: 'jobs/:id/edit',
        name: 'job-edit',
        component: () => import('@/views/JobForm.vue'),
        meta: { title: '编辑任务', crumb: '发布', parent: { label: '任务管理', to: '/jobs' }, admin: true },
      },
      { path: 'tasks', name: 'schedules', component: () => import('@/views/Schedules.vue'), meta: { title: '定时任务', crumb: '发布' } },
      { path: 'history', name: 'history', component: () => import('@/views/History.vue'), meta: { title: '发布历史', crumb: '发布' } },
      {
        path: 'environments',
        name: 'environments',
        component: () => import('@/views/Environments.vue'),
        meta: { title: '环境管理', crumb: '设置', admin: true },
      },
      {
        path: 'servers',
        name: 'servers',
        component: () => import('@/views/Servers.vue'),
        meta: { title: '服务管理', crumb: '设置', admin: true },
      },
      {
        path: 'permissions',
        name: 'permissions',
        component: () => import('@/views/Permissions.vue'),
        meta: { title: '用户管理', crumb: '设置', admin: true },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isAuthed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.public && auth.isAuthed) {
    const redirect = typeof to.query.redirect === 'string' && to.query.redirect.startsWith('/') ? to.query.redirect : null
    return redirect || { path: '/' }
  }
  if (to.meta.admin && !auth.isAdmin) {
    return { path: '/' }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} / DDevOps` : 'DDevOps / 自动化发布平台'
})

export default router