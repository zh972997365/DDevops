/** src/composables/usePermission.js — 权限规则唯一出口 */
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function usePermission() {
  const auth = useAuthStore()

  const isAdmin = computed(() => auth.isAdmin)
  const canManage = computed(() => auth.isAdmin)
  const canEdit = computed(() => auth.isAdmin)
  const canDelete = computed(() => auth.isAdmin)

  // 定时任务：admin 可取消任意，普通用户仅限本人创建
  function canCancel(task) {
    if (auth.isAdmin) return true
    return !!task && task.createdBy === auth.user?.username
  }

  return { isAdmin, canManage, canEdit, canDelete, canCancel }
}
