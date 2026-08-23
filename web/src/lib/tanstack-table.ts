// TanStack Table v9 兼容层：项目统一从这里导入。
// v9 重构了核心 API（features 注册、createXxxRowModel 工厂等），官方提供
// `/legacy` 入口保留 v8 用法；此文件把 v8 名称集中映射到该入口，后续迁移
// 原生 v9 API 时只需替换本文件。
export * from '@tanstack/react-table'
export {
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  getPaginationRowModel,
  getExpandedRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  useLegacyTable as useReactTable,
} from '@tanstack/react-table/legacy'
export type {
  LegacyColumnDef as ColumnDef,
  LegacyReactTable as Table,
  LegacyRow as Row,
  LegacyColumn as Column,
  LegacyCell as Cell,
} from '@tanstack/react-table/legacy'
export type { ColumnVisibilityState as VisibilityState } from '@tanstack/react-table'
