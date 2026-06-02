# web/AGENTS.md

本目录包含两套独立前端。子目录规则优先：

- [default 新版 UI](default/AGENTS.md)
- [classic 旧版 UI](classic/AGENTS.md)

## 双 UI 边界

- `web/default` 和 `web/classic` 独立构建、独立依赖、独立 i18n。不要跨目录 import 源码。
- default 是新功能主线；classic 是退化兼容路径，只做维稳和 bug 修复。
- 后端通过 `theme.frontend` 和 `common.GetTheme()` 选择当前 UI。前端改动不要假设只有一个 UI 会被加载。
- `web/default/dist` 和 `web/classic/dist` 都是后端 embed 资源，Docker 构建要求两套 dist 同时存在。
- 如果 default 调用的接口与本项目后端不一致，改 default 前端对齐后端；不要为了 UI 改后端业务 API。
- 不存在的功能入口应在前端隐藏或降级为明确不可用状态，不得用 mock 数据。

## 常用命令

default:

- `cd web/default && bun install`
- `cd web/default && bun run typecheck`
- `cd web/default && bun run lint`
- `cd web/default && bun run build`

classic:

- `cd web/classic && bun install`
- `cd web/classic && bun run eslint`
- `cd web/classic && bun run lint`
- `cd web/classic && DISABLE_ESLINT_PLUGIN='true' bun run build`

改共享 API、认证、系统设置、主题切换或静态资源路径时，两套 UI 都要构建。

## API 与状态

- default 统一从 `web/default/src/lib/api.ts` 和各 feature 的 `api.ts` 发请求。
- classic 统一从 `web/classic/src/helpers/api.js` 发请求。
- 不要新建绕过统一拦截器的 axios 实例，除非是 SSE、文件下载等确有协议需求，并说明错误处理方式。
- 请求和响应字段必须与本项目后端实际接口一致。参考项目字段只能作为线索，不能直接假定可用。

## i18n

- default 维护 `en` 和 `zh`，动态 key 同步到 `src/i18n/static-keys.ts`。
- classic 目前只维护 `zh`，不要套用 default 的多语言结构。
- 面向用户的新增文案不得裸写为不可翻译常量；按对应 UI 的 i18n 机制处理。

## 视觉与交互

- default 使用 Base UI、Tailwind、Hugeicons/lucide、TanStack 体系。
- classic 使用 Semi Design、react-router-dom、Context 体系。
- 不要把 default 的 UI 原语迁移到 classic，也不要把 Semi 组件引入 default。
- 涉及主题设置、系统设置、登录、渠道、令牌、日志、充值、模型定价等核心路径时，至少检查两套 UI 的对应入口是否仍可用。
