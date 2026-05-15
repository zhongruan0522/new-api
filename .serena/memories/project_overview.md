# New API 项目概述

## 项目目的
New API 是一个基于 newapi 二开的 AI API 网关/代理，聚合 40+ 个上游 AI 提供商（OpenAI、Claude、Gemini、Azure、AWS Bedrock 等）在统一的 API 后面，提供用户管理、计费、速率限制和管理仪表板。

**免责声明**：仅供自用，不得用于商业范围，不得使用本项目进行中转、分发、倒卖厂商Plan。

## 技术栈

### 后端
- **语言**：Go 1.26.2+
- **Web 框架**：Gin v1.9.1
- **ORM**：GORM v2 (v1.25.2)
- **数据库驱动**：
  - SQLite (glebarez/sqlite v1.9.0)
  - MySQL (gorm.io/driver/mysql v1.4.3)
  - PostgreSQL (gorm.io/driver/postgres v1.5.2)
- **缓存**：
  - Redis (go-redis/redis/v8 v8.11.5)
  - 内存缓存（可选）
- **认证**：
  - JWT (golang-jwt/jwt/v5 v5.3.0)
  - WebAuthn/Passkeys (go-webauthn/webauthn v0.14.0)
  - OAuth (GitHub、Discord、OIDC、LinuxDo 等)
- **其他关键库**：
  - 国际化：nicksnyder/go-i18n/v2 v2.6.1
  - 支付：stripe-go/v81、Calcium-Ion/go-epay
  - 音频处理：go-audio、oggvorbis、flac、mp3
  - 视频处理：abema/go-mp4、yapingcat/gomedia
  - Token 计数：tiktoken-go/tokenizer v0.6.2
  - 性能分析：grafana/pyroscope-go v1.2.7

### 前端
- **框架**：React 18.3.1
- **构建工具**：Vite 6.4.2
- **UI 库**：Semi Design (@douyinfe/semi-ui v2.72.2)
- **包管理器**：Bun（推荐）
- **国际化**：i18next v26.0.8 + react-i18next v17.0.6
- **其他库**：
  - 路由：react-router-dom v6.30.3
  - HTTP：axios 1.15.2
  - Markdown：react-markdown v10.1.0、marked v4.3.0
  - 图表：@visactor/react-vchart v1.8.11
  - 图标：lucide-react、@lobehub/icons
  - 通知：react-toastify v9.1.3
  - 日期：dayjs v1.11.20
  - 拼音：pinyin-pro v3.28.1
  - 二维码：qrcode.react v4.2.0
  - 文件上传：react-dropzone v14.4.1
  - 代码高亮：rehype-highlight v7.0.2
  - 数学公式：katex、rehype-katex、remark-math
  - Markdown 扩展：remark-gfm、remark-breaks
  - 防机器人：react-turnstile v1.1.5

### 数据库
- **支持**：SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6
- **必须同时兼容所有三个数据库**

## 项目结构

```
router/              — HTTP 路由（API、中继、仪表板、Web）
controller/          — 请求处理器
service/             — 业务逻辑
model/               — 数据模型和数据库访问（GORM）
relay/               — AI API 中继/代理
  relay/channel/     — 提供商特定适配器
middleware/          — 认证、速率限制、CORS、日志、分发
setting/             — 配置管理（比率、模型、操作、系统、性能）
common/              — 共享工具（JSON、加密、Redis、环境、速率限制等）
dto/                 — 数据传输对象（请求/响应结构）
constant/            — 常量（API 类型、渠道类型、上下文键等）
types/               — 类型定义（中继格式、文件源、错误等）
i18n/                — 后端国际化（go-i18n，en/zh）
oauth/               — OAuth 提供商实现
pkg/                 — 内部包（cachex、ionet）
web/                 — React 前端
  web/src/i18n/      — 前端 i18n（i18next，仅中文）
```

## 关键特性
- 多数据库支持（SQLite、MySQL、PostgreSQL）
- 多渠道 AI 提供商聚合
- 用户管理和认证（JWT、WebAuthn、OAuth）
- 计费和配额管理
- 速率限制
- 音频/视频处理
- Token 计数和估算
- 国际化支持（后端 en/zh，前端仅 zh）
- 性能分析（Pyroscope）
- 管理仪表板
