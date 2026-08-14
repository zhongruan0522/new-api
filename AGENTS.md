# AGENTS.md

本文件是仓库级统一入口。按 https://agents.md/ 的约定，子目录中更近的
`AGENTS.md` 会补充或覆盖这里的规则；用户在对话中的明确要求优先级最高。

## ⚠ 必读：分层规则

**修改某个包/目录下的代码前，必须先阅读该目录下的 `AGENTS.md`。** 根文件只包含
全局规则和概览，每个子目录的 `AGENTS.md` 包含该包特有的约定、模式和检查清单。
跳过子目录规则会导致违反项目约定（如遗漏审计埋点、数据库兼容性问题、前端 i18n 缺失等）。

阅读顺序：根 `AGENTS.md` → 目标目录的 `AGENTS.md` → 如有更深层级继续向下。

涉及跨包改动时，阅读所有受影响包的 `AGENTS.md`。例如改 controller 调用
service 的逻辑时，同时阅读 `controller/AGENTS.md` 和 `service/AGENTS.md`。

## 子规则索引

前端:

- [web/AGENTS.md](web/AGENTS.md)

后端 Go 包:

- [common/AGENTS.md](common/AGENTS.md)
- [router/AGENTS.md](router/AGENTS.md)
- [controller/AGENTS.md](controller/AGENTS.md)
- [middleware/AGENTS.md](middleware/AGENTS.md)
- [service/AGENTS.md](service/AGENTS.md)
- [model/AGENTS.md](model/AGENTS.md)
- [setting/AGENTS.md](setting/AGENTS.md)
- [relay/AGENTS.md](relay/AGENTS.md)
- [i18n/AGENTS.md](i18n/AGENTS.md)

文档:

- [docs/AGENTS.md](docs/AGENTS.md)

`参考项目/` 是本地参考源码，已被忽略；除非用户明确要求，不要修改其中内容。

## 项目概览

这是 Go 实现的 AI API 网关和管理后台。后端聚合 OpenAI、Claude、Gemini、
Azure、AWS Bedrock 等上游能力，提供用户、渠道、计费、限速、认证和管理接口。

主要结构:

- `main.go`: 启动、资源初始化、前端 embed 注入。
- `router/`: API、relay、dashboard、web 静态路由。
- `controller/`: HTTP 边界、请求校验、响应组织。
- `middleware/`: 认证、限速、日志、分发、安全校验。
- `service/`: 业务逻辑、外部请求、计费、迁移编排、审计日志。
- `model/`: GORM 模型、迁移、缓存、数据库访问。
- `setting/`: 系统、运营、模型、倍率、性能、审计等配置。
- `common/`: JSON、缓存、环境变量、静态文件服务、安全工具。
- `relay/`: AI 请求中继、协议转换、供应商适配。
- `i18n/`: 后端 API 响应消息多语言翻译。
- `web/`: 前端 UI，React 19 + TypeScript + Rsbuild。

## 全局工作规则

- 先建立证据链再改代码：现象、入口、相关代码/配置、根因层级、最小修复点、验证方式。
- 保持工作区脏改隔离。不要回滚、覆盖或格式化与当前任务无关的用户改动。
- 不做破坏性 Git 操作，不自动 commit/push；需要提交时只 add 相关具体文件。
- 不写入 secrets。环境变量、数据库 DSN、OAuth 密钥、API key 都不得硬编码到源码或文档示例的真实值。
- 不用模拟成功、静默降级、吞错或假数据让流程"看起来能跑"。失败必须清晰暴露。
- 外部输入必须在系统边界校验：HTTP 参数、表单、文件、网络、数据库、缓存、权限、安全逻辑。
- 新增通用能力前先搜索现有工具函数；确有复用价值再放入 `common/` 或对应前端 `lib/`。
- 不要顺手删除、替换或改名项目标识、AGPL/版权头、Go module path、Docker/CI 镜像名等元数据。

## 后端规则

- Go 版本以 `go.mod` 为准。
- JSON 序列化/反序列化调用使用 `common/json.go` 的包装函数；不要在业务代码里直接调用
  `encoding/json` 的 marshal/unmarshal/decode。
- 数据库必须兼容 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6。优先 GORM；原始 SQL 必须参数化并处理三库差异。
- 渠道相关的外网请求（中继、测试、模型拉取、余额/套餐查询、WebSocket 等）必须走该渠道配置的代理（`service.NewProxyHttpClient` / `NewProxyWebSocketDialer`）。
- 待机内存相关默认值必须保守：连接池 idle 上限、prepared statement 缓存、后台 worker/goroutine 池、
  ticker 唤醒频率等常驻资源不能为追求峰值吞吐随意调大。确需调大时必须保留环境变量覆盖、同步
  `.env.example` 和中英文环境变量文档，并说明低流量/待机场景的内存影响。
- 路由层不要承载业务逻辑；控制器只做边界处理；服务层承载业务；模型层承载持久化。
- relay 改动要保护流式输出、usage 统计、错误映射、计费和供应商协议差异。
- relay 请求 DTO 中需要转发给上游的可选标量字段，优先用指针类型配合 `omitempty`，保留客户端显式传入的 `0`、`0.0`、`false`。
- 后端 API 响应消息的多语言翻译遵守 [i18n/AGENTS.md](i18n/AGENTS.md)：用户可见提示走 `i18n.Msg*` 常量，不要硬编码中英文字符串。

### 审计日志

管理员对系统资源（渠道、用户、令牌、系统设置等）的增删改操作必须接入审计日志。
通过 `service.RecordAudit(...)` 记录，详见 `controller/AGENTS.md` 和 `service/AGENTS.md`。
新增需要审计的资源类型时，按 `controller/AGENTS.md` 中的检查清单同步更新 model、
setting、前端常量和 i18n。

常用验证:

- `go test ./...`
- `go test ./relay/... ./controller/... ./service/...`
- `go build -ldflags "-X 'github.com/NookMux/NookMux/common.Version=$(git rev-parse HEAD)'" -o NookMux`

## 前端规则

- 前端包管理器使用 Bun。`web/` 目录有独立 `package.json` 和 `bun.lock`。
- 改 `web/` 后按影响执行 `bun run typecheck`、`bun run lint`、`bun run build`，适度使用knip。
- 不允许用 mock 数据替代真实后端能力。
- 列表/表格类页面必须使用 `DataTablePage` + `SectionPageLayout`，不得手拼 `Table`
  或用 `Card` 包裹表格。详见 [web/AGENTS.md](web/AGENTS.md) 和
  [docs/开发规范/list-page-table-spec.md](docs/开发规范/list-page-table-spec.md)。
- 前端组件优先复用 `src/components/ui/`、`src/components/data-table/`、
  `src/components/layout/` 等通用组件，避免重复造轮子。

## 文档与参考项目

- `参考项目/` 仅用于比对上游实现。复制代码前必须适配本项目 API 和配置。
- 跨模块详细开发规范文档放在 `docs/开发规范/`，根 `AGENTS.md` 和子目录
  `AGENTS.md` 通过链接引用，避免在 AGENTS.md 中堆砌长篇规范正文。
- `docs/AGENTS.md` 中的规则适用于 `docs/` 目录下的所有文档文件。
