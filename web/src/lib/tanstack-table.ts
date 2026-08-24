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
  columnSizingFeature,
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
  type Cell as TanstackCell,
  type CellData,
  type Column as TanstackColumn,
  type ColumnDef as TanstackColumnDef,
  type ReactTable,
  type Row as TanstackRow,
  type RowData,
} from '@tanstack/react-table'

/**
 * 项目级 feature 注册表，供所有 `useTable` 实例通过
 * `features: appTableFeatures` 共享。
 *
 * 覆盖项目全部用法：列宽（size/getSize）/ 列筛选 / 全局筛选 / 列可见性 /
 * faceted / 行展开 / 行分页 / 行选择 / 行排序；core row model 由 v9 内置，
 * 无需注册。row model 工厂从 v8 的每表 options 搬进 features 槽位。
 */
export const appTableFeatures = tableFeatures({
  columnFacetingFeature,
  columnFilteringFeature,
  columnSizingFeature,
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

export * from '@tanstack/react-table'

/**
 * 预绑定 AppTableFeatures 的常用类型别名。
 *
 * v9 泛型第一参数固定为项目级 feature 注册表，业务代码统一使用这些别名，
 * 与 v8 的单参数写法保持一致；需要不同 feature 集合的表格可直接使用
 * 原生泛型类型。
 */
export type Table<TData extends RowData> = ReactTable<AppTableFeatures, TData>
export type Row<TData extends RowData> = TanstackRow<AppTableFeatures, TData>
export type Column<
  TData extends RowData,
  TValue extends CellData = CellData,
> = TanstackColumn<AppTableFeatures, TData, TValue>
export type Cell<
  TData extends RowData,
  TValue extends CellData = CellData,
> = TanstackCell<AppTableFeatures, TData, TValue>
export type ColumnDef<
  TData extends RowData,
  TValue extends CellData = CellData,
> = TanstackColumnDef<AppTableFeatures, TData, TValue>
export type { ColumnVisibilityState as VisibilityState } from '@tanstack/react-table'

// --- legacy 兼容导出（v8 API → `/legacy` 入口），迁移完成后删除 ---------
// 仅供尚未切换 useTable 的自建表格（upstream-conflict-dialog、
// key-query-logs-table）使用。
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
export type { LegacyColumnDef } from '@tanstack/react-table/legacy'
