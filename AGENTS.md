# AGENTS.md

本文件是仓库级统一入口。按 https://agents.md/ 的约定，子目录中更近的
`AGENTS.md` 会补充或覆盖这里的规则；用户在对话中的明确要求优先级最高。

## 子规则索引

前端双 UI:

- [web/AGENTS.md](web/AGENTS.md)
- [web/default/AGENTS.md](web/default/AGENTS.md)
- [web/classic/AGENTS.md](web/classic/AGENTS.md)

后端 Go 包:

- [common/AGENTS.md](common/AGENTS.md)
- [router/AGENTS.md](router/AGENTS.md)
- [controller/AGENTS.md](controller/AGENTS.md)
- [middleware/AGENTS.md](middleware/AGENTS.md)
- [service/AGENTS.md](service/AGENTS.md)
- [model/AGENTS.md](model/AGENTS.md)
- [setting/AGENTS.md](setting/AGENTS.md)
- [relay/AGENTS.md](relay/AGENTS.md)

文档:

- [docs/AGENTS.md](docs/AGENTS.md)

`参考项目/` 是本地参考源码，已被忽略；除非用户明确要求，不要修改其中内容。

## 项目概览

这是 Go 实现的 AI API 网关和管理后台。后端聚合 OpenAI、Claude、Gemini、
Azure、AWS Bedrock 等上游能力，提供用户、渠道、计费、限速、认证和管理接口。

主要结构:

- `main.go`: 启动、资源初始化、双 UI embed 注入。
- `router/`: API、relay、dashboard、web 静态路由。
- `controller/`: HTTP 边界、请求校验、响应组织。
- `middleware/`: 认证、限速、日志、分发、安全校验。
- `service/`: 业务逻辑、外部请求、计费、迁移编排。
- `model/`: GORM 模型、迁移、缓存、数据库访问。
- `setting/`: 系统、运营、模型、倍率、性能等配置。
- `common/`: JSON、缓存、环境变量、主题静态文件、安全工具。
- `relay/`: AI 请求中继、协议转换、供应商适配。
- `web/default/`: 新版 UI，React 19 + TypeScript + Rsbuild。
- `web/classic/`: 旧版 UI，React 18 + JavaScript + Vite，维稳。

## 全局工作规则

- 先建立证据链再改代码：现象、入口、相关代码/配置、根因层级、最小修复点、验证方式。
- 保持工作区脏改隔离。不要回滚、覆盖或格式化与当前任务无关的用户改动。
- 不做破坏性 Git 操作，不自动 commit/push；需要提交时只 add 相关具体文件。
- 不写入 secrets。环境变量、数据库 DSN、OAuth 密钥、API key 都不得硬编码到源码或文档示例的真实值。
- 不用模拟成功、静默降级、吞错或假数据让流程“看起来能跑”。失败必须清晰暴露。
- 外部输入必须在系统边界校验：HTTP 参数、表单、文件、网络、数据库、缓存、权限、安全逻辑。
- 新增通用能力前先搜索现有工具函数；确有复用价值再放入 `common/` 或对应前端 `lib/`。
- 不要顺手删除、替换或改名项目标识、AGPL/版权头、Go module path、Docker/CI 镜像名等元数据。

## 双 UI 架构不变量

- `web/default` 是完整升级的新 UI；`web/classic` 是旧 UI 退化兼容路径。
- `theme.frontend` 只允许 `default` 或 `classic`，后端通过 `common.GetTheme()` 选择资源。
- `main.go` 必须 embed 两套 dist，`Dockerfile` 会检查 `web/default/dist/index.html`
  和 `web/classic/dist/index.html` 同时存在。
- 后端业务 API 对新旧 UI 保持一致。default UI 字段或路径不匹配时，优先改前端适配本项目后端。
- classic 进入维稳模式：只修 bug 和兼容性问题，不在 classic 上扩展新功能或引入新技术栈。
- 任何影响共享 API、认证、主题切换、静态资源服务、系统设置的改动，都要考虑两套 UI 的运行路径。
- 不允许在 default 或 classic 中用 mock 数据替代真实后端能力。

## 后端规则

- Go 版本以 `go.mod` 为准。
- JSON 序列化/反序列化调用使用 `common/json.go` 的包装函数；不要在业务代码里直接调用
  `encoding/json` 的 marshal/unmarshal/decode。
- 数据库必须兼容 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6。优先 GORM；原始 SQL 必须参数化并处理三库差异。
- 路由层不要承载业务逻辑；控制器只做边界处理；服务层承载业务；模型层承载持久化。
- relay 改动要保护流式输出、usage 统计、错误映射、计费和供应商协议差异。
- relay 请求 DTO 中需要转发给上游的可选标量字段，优先用指针类型配合 `omitempty`，保留客户端显式传入的 `0`、`0.0`、`false`。

常用验证:

- `go test ./...`
- `go test ./relay/... ./controller/... ./service/...`
- `go build -ldflags "-X 'github.com/zhongruan0522/new-api/common.Version=$(git rev-parse HEAD)'" -o new-api`

## 前端规则

- 前端包管理器使用 Bun。各 UI 子项目都有独立 `package.json` 和 `bun.lock`。
- 不要跨 `web/default` 与 `web/classic` 直接 import 源码或共享 node_modules。
- 改 `web/default` 后按影响执行 `bun run typecheck`、`bun run lint`、`bun run build`。
- 改 `web/classic` 后按影响执行 `bun run eslint`、`bun run lint`、`bun run build`。
- 同时影响共享 API 或主题切换时，两个前端都要构建。

## 文档与参考项目

- `参考项目/` 仅用于比对上游实现。复制代码前必须适配本项目 API、配置和双 UI 约束。