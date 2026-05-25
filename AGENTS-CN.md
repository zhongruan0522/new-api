# AGENTS-CN.md — new-api 项目约定

## 概述

这是一个基于 Go 构建的 AI API 网关/代理。它将 40 多个上游 AI 提供商（OpenAI、Claude、Gemini、Azure、AWS Bedrock 等）聚合在统一的 API 之后，具备用户管理、计费、速率限制和管理后台功能。

## 技术栈

- **后端**: Go 1.22+、Gin Web 框架、GORM v2 ORM
- **前端**: React 18、Vite、Semi Design UI (@douyinfe/semi-ui)
- **数据库**: SQLite、MySQL、PostgreSQL（必须同时支持这三种）
- **缓存**: Redis (go-redis) + 内存缓存
- **认证**: JWT、WebAuthn/Passkeys、OAuth (GitHub、Discord、OIDC 等)
- **前端包管理器**: Bun（优先于 npm/yarn/pnpm）

## 架构

分层架构: 路由层(Router) -> 控制器层(Controller) -> 服务层(Service) -> 模型层(Model)

```
router/        — HTTP 路由 (API、中继、后台、网页)
controller/    — 请求处理器
service/       — 业务逻辑
model/         — 数据模型和数据库访问 (GORM)
relay/         — AI API 中继/代理，包含各提供商适配器
  relay/channel/ — 各提供商专属适配器 (openai/、claude/、gemini/、aws/ 等)
middleware/    — 认证、速率限制、跨域、日志、分发
setting/       — 配置管理 (倍率、模型、运营、系统、性能)
common/        — 共享工具函数 (JSON、加密、Redis、环境变量、速率限制等)
dto/           — 数据传输对象 (请求/响应结构体)
constant/      — 常量 (API 类型、渠道类型、上下文键)
types/         — 类型定义 (中继格式、文件来源、错误)
i18n/          — 后端国际化 (go-i18n, 英文/中文)
oauth/         — OAuth 提供商实现
pkg/           — 内部包 (cachex、ionet)
web/           — React 前端
  web/src/i18n/  — 前端国际化 (i18next, 仅中文)
```

## 国际化 (i18n)

### 后端 (`i18n/`)
- 库: `nicksnyder/go-i18n/v2`
- 语言: 英文、中文

### 前端 (`web/src/i18n/`)
- 库: `i18next` + `react-i18next`
- 语言: 中文（仅中文）
- 翻译文件: `web/src/i18n/locales/zh.json` — 扁平 JSON，键和值均为中文
- 用法: `useTranslation()` 钩子，在组件中调用 `t('中文key')`
- Semi UI 区域设置: 固定为 `zh_CN`

## 规则

### 规则 1: JSON 包 — 使用 `common/json.go`

所有 JSON 序列化/反序列化操作**必须**使用 `common/json.go` 中的包装函数：

- `common.Marshal(v any) ([]byte, error)`
- `common.Unmarshal(data []byte, v any) error`
- `common.UnmarshalJsonStr(data string, v any) error`
- `common.DecodeJson(reader io.Reader, v any) error`
- `common.GetJsonType(data json.RawMessage) string`

**请勿**在业务代码中直接导入或调用 `encoding/json`。这些包装函数用于保持一致性和未来的可扩展性（例如，替换为更快的 JSON 库）。

注意：`json.RawMessage`、`json.Number` 以及来自 `encoding/json` 的其他类型定义仍可作为类型引用，但实际的序列化/反序列化调用必须通过 `common.*` 进行。

### 规则 2: 数据库兼容性 — SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6

所有数据库代码**必须**同时完全兼容这三种数据库。

**使用 GORM 抽象：**
- 优先使用 GORM 方法（`Create`、`Find`、`Where`、`Updates` 等），而非原始 SQL。
- 让 GORM 处理主键生成 —— 不要直接使用 `AUTO_INCREMENT` 或 `SERIAL`。

**当不可避免使用原始 SQL 时：**
- 列引号差异：PostgreSQL 使用 `"column"`，MySQL/SQLite 使用 `` `column` ``。
- 对 `group`、`key` 等保留字列，使用来自 `model/main.go` 的 `commonGroupCol`、`commonKeyCol` 变量。
- 布尔值差异：PostgreSQL 使用 `true`/`false`，MySQL/SQLite 使用 `1`/`0`。使用 `commonTrueVal`/`commonFalseVal`。
- 使用 `common.UsingPostgreSQL`、`common.UsingSQLite`、`common.UsingMySQL` 标志来分支数据库特定逻辑。

**未经跨数据库回退方案禁止使用的功能：**
- MySQL 专有函数（例如，没有 PostgreSQL `STRING_AGG` 等效方案时使用的 `GROUP_CONCAT`）
- PostgreSQL 专有运算符（例如，`@>`、`?`、`JSONB` 运算符）
- SQLite 中的 `ALTER COLUMN`（不支持 —— 请使用添加列变通方案）
- 没有回退方案的数据库特定列类型 —— 对于 JSON 存储，使用 `TEXT` 而非 `JSONB`

**迁移：**
- 确保所有迁移在三种数据库上都能正常工作。
- 对于 SQLite，使用 `ALTER TABLE ... ADD COLUMN` 替代 `ALTER COLUMN`（参见 `model/main.go` 中的模式）。

### 规则 3: 前端 — 优先使用 Bun

在前端 (`web/` 目录) 使用 `bun` 作为首选的包管理器和脚本运行器：
- `bun install` 用于安装依赖
- `bun run dev` 用于开发服务器
- `bun run build` 用于生产构建
- `bun run i18n:*` 用于 i18n 工具
