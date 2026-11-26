/** src/composables/useDragSort.js — 卡片拖拽排序（乐观更新 + 失败回滚） */
import { ref } from 'vue'
import { useToast } from './useToast'

export function useDragSort({ all, reorder, load }) {
  const toast = useToast()
  const dragging = ref('')
  const over = ref('')

  function onDragStart(item, e) {
    dragging.value = item.id
    e.dataTransfer.effectAllowed = 'move'
    // Firefox 要求设置 data 才会真正开始拖拽
    e.dataTransfer.setData('text/plain', item.id)
  }

  function onDragOver(item, e) {
    if (!dragging.value || dragging.value === item.id) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    over.value = item.id
  }

  function onDragLeave(item) {
    if (over.value === item.id) over.value = ''
  }

  async function onDrop(item, e) {
    e.preventDefault()
    const fromId = dragging.value
    const toId = item.id
    dragging.value = ''
    over.value = ''
    if (!fromId || fromId === toId) return

    // 基于全量列表重排，避免搜索过滤时丢失未显示的项目
    const next = [...all.value]
    const from = next.findIndex((x) => x.id === fromId)
    const to = next.findIndex((x) => x.id === toId)
    if (from < 0 || to < 0) return
    next.splice(to, 0, ...next.splice(from, 1))

    const previous = all.value
    all.value = next
    try {
      await reorder(next.map((x) => x.id))
      toast.success('顺序已保存')
    } catch (err) {
      all.value = previous
      toast.error(err.message)
      await load()
    }
  }

  function onDragEnd() {
    dragging.value = ''
    over.value = ''
  }

  return { dragging, over, onDragStart, onDragOver, onDragLeave, onDrop, onDragEnd }
}
