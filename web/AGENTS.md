# web/AGENTS.md

前端规则。上级规则见 [../AGENTS.md](../AGENTS.md)。

## 技术栈

- React 19、TypeScript、Rsbuild。
- 路由: `@tanstack/react-router`，文件路由在 `src/routes/`。
- 数据: `@tanstack/react-query`、Zustand、统一 axios 实例 `src/lib/api.ts`。
- UI: Base UI、Tailwind CSS 4、`src/components/ui/`、Hugeicons/lucide。
- 表单: React Hook Form + Zod。
- 图表: VChart v2。
- i18n: i18next + react-i18next，`en` 和 `zh`。

## 后端嵌入载体

- `web/embed.go` 是后端专用 Go 源文件，用于在 `web/dist` 旁声明 `//go:embed`；Go embed 不支持 `..`，因此该文件不能移入 `internal/app/webdist/`。
- `internal/app/webdist/` 是启动装配层使用的门面包；修改嵌入路径时必须同步检查该包和 `router/web_router.go`。
- 修改 `web/embed.go` 后至少执行 `go test ./internal/app/... ./router/...`。

## 命令

- 安装依赖: `bun install`
- 开发服务: `bun run dev`
- 类型检查: `bun run typecheck`
- ESLint: `bun run lint`
- 格式检查: `bun run format:check`
- 生产构建: `bun run build`
- 完整构建检查: `bun run build:check`
- i18n 同步: `bun run i18n:sync`

改 TS/TSX 后至少执行 `bun run typecheck`。改路由、API、核心页面或构建配置后执行
`bun run build` 或 `bun run build:check`。

## 文件组织

- 功能模块放 `src/features/<feature>/`，常见结构为 `api.ts`、`types.ts`、`constants.ts`、
`components/`、`hooks/`、`lib/`。
- 路由只负责装配页面和路由级校验，业务逻辑放到 feature。
- 通用组件放 `src/components/`，基础 UI 原语放 `src/components/ui/`。
- 通用工具放 `src/lib/`，状态放 `src/stores/`。
- 类型导入使用 `import type`。

## API 与数据

- 使用 `src/lib/api.ts` 的 `api` 实例，保留 cookie、错误处理、GET 去重和 `New-Api-User` 头。
- 数据获取用 `useQuery`，变更用 `useMutation`，query key 使用数组并保持层级稳定。
- 成功后按影响范围 invalidate 相关 query。不要手动刷新整个页面替代状态更新。
- 服务端响应以本项目后端为准。参考项目 API 不存在时，隐藏入口或改前端适配，不新增后端业务 API。
- 不使用 mock 数据、假分页、假成功状态或静默吞错。

## i18n

- React 组件中使用 `const { t } = useTranslation()`;非 React 模块可用 `i18next.t`。
- 文案按 feature 物理拆分到 `src/i18n/locales/<locale>/<section>.json`(`<locale>` 为 en 或 zh,`<section>` 对应 feature,如 `auth`、`channels`、`system-settings`,共用/品牌/壳层分别用 `common.json` 和 `layout.json`)。每个 section 文件是扁平的 `Record<string, string>`,不包 translation 包装层;运行时在 `src/i18n/config.ts` 用对象展开合并回单一 translation namespace。
- 用户可见文案用语义 key,格式 `<section>.<group>.<name>[.<attr>]`,前缀 `<section>` 与所在文件名一致(如 `channels.json` → `channels.fields.xxx`);`<group>` 收敛在 `fields/actions/status/errors/tips/titles/placeholders`,`<name>` 用 camelCase,section 名含连字符时前缀转 camelCase(`usage-logs` → `usageLogs`)。语义 key 为扁平字符串,禁止写成嵌套对象。
- 新增用户可见文案:在相关 feature 的 `src/i18n/locales/en/<section>.json` 和 `src/i18n/locales/zh/<section>.json` 中各加一行;新增 section 文件后必须在 `src/i18n/config.ts` 里导入并展开到对应语言的 translation。
- 用 `bun run i18n:sync` 校验 en/zh key 是否对齐(脚本会按 base 顺序重排各 section、产出 `_extras`、`_reports`,不再写回单一大文件)。
- 常量、配置、枚举等动态 key 要登记到 `src/i18n/static-keys.ts`,或确保以 `t('...')` 字面量出现。
- `keySeparator: false`、`nsSeparator: false`;`supportedLngs` 目前只有 en 和 zh,不要添加未维护的语言入口。

## 类型、表单与错误

- 避免 `any`；确实无法确定时用 `unknown` 并在边界收窄。
- 表单 schema 放在 feature 的 `lib/` 或相邻模块，用 Zod 定义并通过 `z.infer` 导出类型。
- 组件 props 保持明确类型。复杂页面优先拆出小组件、hooks、纯函数。
- toast 使用 `sonner`，文案要走 i18n。
- 禁止 `no-console` 违规；调试日志不要留在生产路径。

## 样式与交互

- Tailwind 为主，动态类名用项目 `cn()`/`tailwind-merge` 体系。
- 组件优先复用 `src/components/ui/` 和现有 feature 组件。
- UI 应支持深浅色、主题 preset、移动端和键盘操作。
- 不要把页面章节做成嵌套卡片；管理后台优先信息密度、清晰扫描和稳定布局。

## 列表/表格类页面规范

管理后台凡是以"分页/可筛选的数据行列表"为主体的页面，必须使用 `DataTablePage`
（`@/components/data-table`）+ `SectionPageLayout`（`@/components/layout`），不得
手拼 `Table` 或用 `Card` 包裹表格。标准参照页面为"使用日志"
（`features/usage-logs/components/usage-logs-table.tsx`）。

关键约定：

- **Sticky 表头**：传 `tableClassName='overflow-x-auto'` +
  `tableHeaderClassName='bg-muted/30 sticky top-0 z-10'`，有 `size` 时配 `applyHeaderSize`。
- **分页**：默认走 `PageFooterPortal`；无分页传 `showPagination={false}`。
- **筛选区**：作为 `toolbar` slot 传入，简单走 `toolbarProps`，复杂传自定义节点。
- **骨架屏/空状态**：必须用内置 `TableSkeleton` / `TableEmpty` / `MobileCardList`，
  禁止纯文字或 `Loader2` 占位。
- **Card 禁用**：表格、筛选区、分页区均不得包成独立圆角卡片；统计信息走
  `afterTable` slot 或 `SectionPageLayout.Actions`。
- **批量操作**：`enableRowSelection` + `bulkActions` slot。

纯表单/配置页（如系统设置定价、公开定价目录页）不适用本规范。完整规范、Props
列表、标准模板和检查清单见
[docs/开发规范/list-page-table-spec.md](../docs/开发规范/list-page-table-spec.md)。

