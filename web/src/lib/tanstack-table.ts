// TanStack Table v9 统一入口：项目所有表格相关导入都从这里走。
//
// 历史背景：dependabot 升级到 v9 后，此文件曾把 v8 API 集中映射到官方
// deprecated 的 `/legacy` 入口完成止血。`/legacy` 默认注册全部 stock
// features、不做 tree-shaking，且上游随时可能删除该入口。
//
// 迁移目标：全面切换到 v9 原生 API（`useTable` + `features` 注册）。
// `appTableFeatures` 在这里一次性注册项目用到的 feature 与 row model
// 工厂，所有表格实例共享；下方 legacy 导出在全部调用点切换完成前暂时
// 保留，切换完成后整段删除。
import {
  columnFacetingFeature,
  columnFilteringFeature,
  columnVisibilityFeature,
  createExpandedRowModel,
  createFacetedRowModel,
  createFacetedUniqueValues,
  createFilteredRowModel,
  createPaginatedRowModel,
  createSortedRowModel,
  globalFilteringFeature,
  rowExpandingFeature,
  rowPaginationFeature,
  rowSelectionFeature,
  rowSortingFeature,
  tableFeatures,
} from '@tanstack/react-table'

/**
 * 项目级 feature 注册表，供所有 `useTable` 实例通过
 * `features: appTableFeatures` 共享。
 *
 * 覆盖项目全部用法：列筛选 / 全局筛选 / 列可见性 / faceted / 行展开 /
 * 行分页 / 行选择 / 行排序；core row model 由 v9 内置，无需注册。
 * row model 工厂从 v8 的每表 options 搬进 features 槽位。
 */
export const appTableFeatures = tableFeatures({
  columnFacetingFeature,
  columnFilteringFeature,
  columnVisibilityFeature,
  globalFilteringFeature,
  rowExpandingFeature,
  rowPaginationFeature,
  rowSelectionFeature,
  rowSortingFeature,
  expandedRowModel: createExpandedRowModel(),
  facetedRowModel: createFacetedRowModel(),
  facetedUniqueValues: createFacetedUniqueValues(),
  filteredRowModel: createFilteredRowModel(),
  paginatedRowModel: createPaginatedRowModel(),
  sortedRowModel: createSortedRowModel(),
})

export type AppTableFeatures = typeof appTableFeatures

// --- legacy 兼容导出（v8 API → `/legacy` 入口），迁移完成后删除 ---------
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
