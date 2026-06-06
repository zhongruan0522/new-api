# web/classic/AGENTS.md

旧版 UI 规则。上级规则见 [../AGENTS.md](../AGENTS.md)。

## 定位

classic 是旧版本兼容 UI，主要保障升级用户稳定运行。只修 bug、兼容性和安全问题；
新功能默认在 `web/default` 实现，除非用户明确要求同步到 classic。

## 技术栈

- React 18、JavaScript/JSX、Vite。
- UI: Semi Design、Semi Icons、少量 lucide/react-icons。
- 路由: `react-router-dom`。
- 状态: Context + reducer。
- 请求: `src/helpers/api.js` 中的 `API` 实例。
- i18n: `src/i18n/i18n.js`，目前只维护 `zh`。

不要迁入 default 的 Base UI、TanStack Router、Zustand 或 TypeScript 架构。

## 命令

- 安装依赖: `bun install`
- 开发服务: `bun run dev`
- Prettier 检查: `bun run lint`
- Prettier 修复: `bun run lint:fix`
- ESLint: `bun run eslint`
- 生产构建: `DISABLE_ESLINT_PLUGIN='true' bun run build`

改 JSX/JS 后按影响执行 `bun run eslint` 和 `bun run build`。只改文案或样式时至少执行
`bun run lint`。

## 文件组织

- 页面在 `src/pages/`，通用布局和业务组件在 `src/components/`。
- hooks 放 `src/hooks/`，常量放 `src/constants/`，工具放 `src/helpers/`。
- API 请求复用 `src/helpers/api.js`，不要新建无拦截器的 axios 实例。
- 保持现有 JSX、分号和 Prettier 风格。

## 维稳规则

- 优先最小修复，不做大规模重构或技术栈升级。
- 不要改变路由路径、localStorage key、登录态、主题态等旧用户依赖的兼容行为。
- 修改系统设置、登录、渠道、令牌、日志、充值、模型定价等路径时，确认 classic 入口仍可达。
- 涉及 `theme.frontend` 时，只提交 `default` 或 `classic`。
- 不要用 mock 数据、硬编码成功响应或隐藏错误来绕过后端问题。

## i18n 与提示

- 用户可见文案使用 `useTranslation()` 或现有 `t()` 调用。
- 新增中文 key 同步到 `src/i18n/locales/zh.json`。
- toast 使用 `src/helpers/utils.jsx` 中的 `showSuccess`、`showError`、`showWarning`、`showInfo`。
