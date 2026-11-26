<template>
  <div class="card overflow-hidden">
    <div v-if="$slots.toolbar">
      <slot name="toolbar" />
    </div>

    <Transition name="bulk">
      <div
        v-if="selectable && selected.length"
        class="flex flex-wrap items-center gap-2 border-b border-line bg-primary-soft px-4 py-2"
      >
        <span class="text-xs font-medium text-primary-ink">已选 {{ selected.length }} 项</span>
        <span class="h-3 w-px bg-line-strong" />
        <slot name="bulk" />
        <button type="button" class="btn btn-ghost btn-xs ml-auto" @click="$emit('update:selected', [])">
          取消选择
        </button>
      </div>
    </Transition>

    <div class="table-scroll overflow-x-auto">
      <table class="data-table" :style="{ minWidth }">
        <thead>
          <tr>
            <th v-if="selectable" class="w-10 pr-0">
              <input
                v-indeterminate="someSelected && !allSelected"
                type="checkbox"
                class="table-check"
                :checked="allSelected"
                :disabled="!selectableRows.length"
                aria-label="选择本页全部"
                @change="toggleAll($event.target.checked)"
              />
            </th>
            <th
              v-for="col in columns"
              :key="col.key"
              :class="[alignClass(col), col.class, stickyKey === col.key && 'sticky-col']"
              :style="col.width ? { width: col.width } : null"
              :aria-sort="ariaSort(col)"
            >
              <button
                v-if="col.sortable"
                type="button"
                class="group inline-flex items-center gap-1 rounded text-left transition-colors hover:text-ink"
                @click="$emit('sort-change', col.key)"
              >
                {{ col.title }}
                <span class="inline-flex flex-col leading-none" aria-hidden="true">
                  <ChevronUp
                    class="h-2.5 w-2.5"
                    :class="sortKey === col.key && sortDir === 'asc' ? 'text-primary' : 'text-line-strong'"
                  />
                  <ChevronDown
                    class="-mt-0.5 h-2.5 w-2.5"
                    :class="sortKey === col.key && sortDir === 'desc' ? 'text-primary' : 'text-line-strong'"
                  />
                </span>
              </button>
              <template v-else>{{ col.title }}</template>
            </th>
          </tr>
        </thead>

        <tbody v-if="loading">
          <tr v-for="r in 6" :key="`sk-${r}`">
            <td v-if="selectable" class="w-10 pr-0" data-label="选择"><span class="skeleton h-3.5 w-3.5 rounded-[3px]" /></td>
            <td v-for="(col, i) in columns" :key="col.key" :class="alignClass(col)" :data-label="col.key === 'actions' ? '操作' : col.title">
              <span class="skeleton h-3" :style="{ width: skeletonWidth(i) }" />
            </td>
          </tr>
        </tbody>

        <tbody v-else-if="error">
          <tr>
            <td :colspan="colSpan" class="p-0">
              <ErrorState :message="error" @retry="$emit('retry')" />
            </td>
          </tr>
        </tbody>

        <tbody v-else-if="rows.length">
          <tr
            v-for="(row, index) in rows"
            :key="keyOf(row, index)"
            :class="[
              isSelected(row) && 'is-selected',
              clickable && 'cursor-pointer',
              rowClass?.(row),
            ]"
            @click="clickable && $emit('row-click', row)"
          >
            <td v-if="selectable" class="w-10 pr-0" data-label="选择" @click.stop>
              <input
                type="checkbox"
                class="table-check"
                :checked="isSelected(row)"
                :disabled="!canSelect(row)"
                :aria-label="`选择 ${keyOf(row, index)}`"
                @change="toggleRow(row, $event.target.checked)"
              />
            </td>
            <td
              v-for="(col, i) in columns"
              :key="col.key"
              :class="[alignClass(col), col.class, stickyKey === col.key && 'sticky-col', i === 0 && 'cell-primary']"
              :data-label="col.key === 'actions' ? '操作' : col.title"
            >
              <slot :name="`cell-${col.key}`" :row="row" :value="cellValue(row, col)" :index="index">
                <span :class="col.mono && 'mono'">{{ cellValue(row, col) ?? '—' }}</span>
              </slot>
            </td>
          </tr>
        </tbody>

        <tbody v-else>
          <tr>
            <td :colspan="colSpan" class="p-0">
              <EmptyState :title="emptyTitle" :description="emptyDescription">
                <template v-if="$slots['empty-action']" #action>
                  <slot name="empty-action" />
                </template>
              </EmptyState>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <Pagination
      v-if="pagination && !error && total > 0"
      :current="page"
      :total="total"
      :page-size="pageSize"
      @update:page="$emit('update:page', $event)"
      @update:page-size="$emit('update:pageSize', $event)"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ChevronDown, ChevronUp } from 'lucide-vue-next'
import Pagination from '@/components/base/Pagination.vue'
import EmptyState from '@/components/data/EmptyState.vue'
import ErrorState from '@/components/data/ErrorState.vue'

const props = defineProps({
  columns: { type: Array, required: true },
  rows: { type: Array, default: () => [] },
  rowKey: { type: [String, Function], default: 'id' },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  total: { type: Number, default: 0 },
  page: { type: Number, default: 1 },
  pageSize: { type: Number, default: 10 },
  pagination: { type: Boolean, default: true },
  selectable: { type: Boolean, default: false },
  selected: { type: Array, default: () => [] },
  rowSelectable: { type: Function, default: null },
  sortKey: { type: String, default: '' },
  sortDir: { type: String, default: 'desc' },
  stickyKey: { type: String, default: '' },
  clickable: { type: Boolean, default: false },
  rowClass: { type: Function, default: null },
  emptyTitle: { type: String, default: '暂无数据' },
  emptyDescription: { type: String, default: '' },
  minWidth: { type: String, default: '860px' },
})
const emit = defineEmits(['update:page', 'update:pageSize', 'update:selected', 'sort-change', 'row-click', 'retry'])

const vIndeterminate = {
  mounted(el, binding) {
    el.indeterminate = !!binding.value
  },
  updated(el, binding) {
    el.indeterminate = !!binding.value
  },
}

const colSpan = computed(() => props.columns.length + (props.selectable ? 1 : 0))

function keyOf(row, index) {
  if (typeof props.rowKey === 'function') return props.rowKey(row)
  return row?.[props.rowKey] ?? index
}
function canSelect(row) {
  return props.rowSelectable ? props.rowSelectable(row) : true
}
const selectableRows = computed(() => props.rows.filter(canSelect))
const allSelected = computed(
  () => selectableRows.value.length > 0 && selectableRows.value.every((r) => props.selected.includes(keyOf(r)))
)
const someSelected = computed(() => selectableRows.value.some((r) => props.selected.includes(keyOf(r))))

function isSelected(row) {
  return props.selected.includes(keyOf(row))
}
function toggleRow(row, checked) {
  const key = keyOf(row)
  const next = checked ? [...props.selected, key] : props.selected.filter((k) => k !== key)
  emit('update:selected', next)
}
function toggleAll(checked) {
  if (checked) {
    const merged = new Set(props.selected)
    selectableRows.value.forEach((r) => merged.add(keyOf(r)))
    emit('update:selected', [...merged])
  } else {
    const keys = new Set(selectableRows.value.map((r) => keyOf(r)))
    emit('update:selected', props.selected.filter((k) => !keys.has(k)))
  }
}

function alignClass(col) {
  if (col.align === 'right') return 'text-right'
  if (col.align === 'center') return 'text-center'
  return 'text-left'
}
function ariaSort(col) {
  if (!col.sortable) return undefined
  if (props.sortKey !== col.key) return 'none'
  return props.sortDir === 'asc' ? 'ascending' : 'descending'
}
function cellValue(row, col) {
  return col.value ? col.value(row) : row?.[col.key]
}
function skeletonWidth(i) {
  return ['72%', '48%', '64%', '40%', '56%', '44%'][i % 6]
}
</script>

<style scoped>
.table-check {
  width: 14px;
  height: 14px;
  border-radius: 3px;
  accent-color: var(--primary);
  cursor: pointer;
}
.table-check:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
.bulk-enter-active,
.bulk-leave-active {
  transition: opacity 140ms var(--ease), transform 140ms var(--ease);
}
.bulk-enter-from,
.bulk-leave-to {
  transform: translateY(-3px);
}
@media (max-width: 767px) {
  .table-scroll {
    overflow: visible;
  }
  .data-table {
    display: block;
    min-width: 0 !important;
  }
  .data-table thead {
    display: none;
  }
  .data-table tbody,
  .data-table tr {
    display: block;
  }
  .data-table tr {
    padding: 12px 16px;
    border-bottom: 1px solid var(--line-soft);
  }
  .data-table tbody tr:last-child {
    border-bottom: 0;
  }
  .data-table td {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 4px 0;
    border: 0;
    text-align: right;
  }
  .data-table td::before {
    content: attr(data-label);
    flex-shrink: 0;
    color: var(--ink-3);
    font-size: 11px;
    font-weight: 400;
    text-align: left;
  }
  .data-table td[data-label=''],
  .data-table td[data-label='选择'],
  .data-table td[data-label='操作'] {
    justify-content: flex-end;
    padding-top: 8px;
  }
  .data-table td[data-label=''],
  .data-table td[data-label='选择']::before,
  .data-table td[data-label='操作']::before {
    content: none;
  }
  .data-table td[data-label='选择'] {
    justify-content: flex-start;
  }
  .data-table .sticky-col {
    position: static;
    background: transparent;
    box-shadow: none;
  }
  .data-table td.cell-primary {
    justify-content: flex-start;
    padding-bottom: 2px;
    font-size: 13px;
    font-weight: 500;
  }
  .data-table td.cell-primary::before {
    content: none;
  }
  .data-table td.cell-primary > * {
    width: 100%;
    text-align: left;
  }
  .table-check {
    width: 18px;
    height: 18px;
  }
}
@media (max-width: 1023px) {
  .sticky-col {
    position: sticky;
    left: 0;
    z-index: 2;
    background: var(--surface-solid);
    box-shadow: 1px 0 0 var(--line);
  }
  thead .sticky-col {
    z-index: 3;
  }
  tbody tr:hover .sticky-col {
    background: color-mix(in srgb, var(--surface-solid) 92%, var(--primary) 4%);
  }
  tbody tr.is-selected .sticky-col {
    background: var(--primary-soft);
  }
}
</style>