# internal/infra/AGENTS.md

`internal/infra/` 是基础设施层（阶段 5.3 起原 `internal/service/` 中无领域归属的
能力型实现按职责迁入；`log/` 由阶段 3 提前落位；`db/`、`redis/`、`cache/`、
`email/`、`security/`、`runtime/` 由阶段 4 自 `internal/common/` 迁入）。

## 子包

| 子包 | 内容 |
|---|---|
| `db/` | 数据库类型常量与连接状态变量（`UsingSQLite`/`LogSqlType`/`SQLitePath` 等；GORM 初始化在 `internal/store/db`，本包只持有状态）。 |
| `redis/` | Redis 客户端（`RDB`/`InitRedisClient`/`Redis*` 读写族）。 |
| `cache/` | 业务缓存：磁盘缓存（`disk_cache*.go`）、请求体存储（`body_storage.go`，含 `ErrRequestBodyTooLarge`）与 Redis 限流器（`limiter/`）。与 `pkg/cachex` 的区分：cachex 是无业务依赖的底层库，这里是带统计与配置的业务缓存。 |
| `email/` | SMTP 邮件发送（含 Outlook LOGIN 兼容认证）。SMTP 配置变量在 `internal/common/constants.go`。 |
| `security/` | 安全工具：SSRF 防护与连接时复查、IP/CIDR 工具、URL 校验、HMAC/bcrypt 哈希、TOTP/备用码、验证码、validator 实例、Gin 受信代理配置。 |
| `runtime/` | 运行时设施：系统监控（CPU/内存/磁盘）、pprof、Pyroscope、有界 relay goroutine 池（`RelayGo`/`RelayCtxGo`）、安全 channel 发送。 |
| `log/` | 业务日志工具（`LogInfo`/`LogError` 等，原 `logger/`）；`Dir` 由 app 层注入。 |
| `httpclient/` | 通用 HTTP 传输层：全局 client、SSRF 校验与连接时复查（防 DNS rebinding）、代理客户端构造与缓存（`NewProxyHttpClient` / `NewProxyWebSocketDialer`）。渠道"走哪个代理"的选择是业务规则，留在调用方（relay / domain/channel），本包只提供按 URL 构造代理客户端的能力。 |
| `media/` | 文件/图片/音频的下载、类型探测、解码与 base64 处理；worker 下载请求。 |
| `tokenizer/` | token 计数与估算（`tokenizer.go` + `token_estimator.go` + `token_counter.go` 三者同域，唯一归属）。 |
| `notify/` | 用户通知发送（邮件/webhook/bark/gotify）与频控。 |
| `payment/` | 支付集成：epay 回调地址、stripe Checkout 链接生成与 webhook 事件入账、订单锁（`LockOrder`/`UnlockOrder`）。gin 边界 handler 仍在 controller。 |
| `passkey/` | WebAuthn/Passkey 会话与用户转换。 |
| `custom_voice/` | MiniMax 定制语音（音色克隆、预览、确认计费）。 |

## 规则

- infra 包不承载 HTTP 边界逻辑；gin handler 留在 controller，本层提供可复用实现。
  既有例外须保持现状并注明（如 `payment.GenStripeLink` 接收 `*gin.Context` 仅为
  本地化 API Key 错误文案，行为与迁移前一致）。
- 外部调用必须复用 `httpclient` 的客户端、超时与 SSRF 防护，不允许裸 `http.Client` 出站。
- 文件下载、解析、存储要校验大小、类型、来源和错误路径。
- 失败必须清晰暴露：不要模拟成功、静默降级或吞错。
- infra 包之间可以按需单向依赖，新增依赖前确认无环。当前既定方向：
  `tokenizer → media → httpclient`、`notify → {media, httpclient}`、
  `log → runtime`（日志滚动异步化用 `RelayGo`）、`runtime → cache`
  （磁盘缓存路径用于磁盘空间监控）。
- infra 不得 import controller / middleware / router / relay（`media` 不得引用 relay 包）。
  既有例外须保持现状并注明，不得新增：
  - `tokenizer/token_counter.go` 直接 import `relay/common` 与 `relay/constant`
    （阶段 5.3 迁移前的既有签名依赖，待后续阶段收敛；`config/reasoning` 还会间接
    拉入 `relay/channel/openrouter`）；
  - `tokenizer/token_counter.go` import `internal/httpapi` 根包（gin 请求体/上下文
    键工具，阶段 4 自 `internal/common` 迁出后的既定依赖；该根包是仅依赖
    `infra/cache`/`domain/shared`/`internal/common`/`pkg/jsonx` 的叶子工具包，不承载
    路由/中间件/控制器）；
    除此之外不得新增 infra → httpapi 子包依赖。
- infra 可以依赖 `internal/common`（业务全局状态内核，如 `SysLog`/`DebugEnabled`/
  `QuotaPerUnit`/SMTP 配置）与 `internal/domain/shared`（跨层契约与运行时限值），
  方向单向：`common` 与 `shared` 不得反向 import infra（`shared → infra/log` 为
  dto/types 既有依赖，是唯一历史例外，见 `internal/domain/AGENTS.md`）。
- 待机内存相关默认值保守（连接池、缓存、后台任务），调大须保留环境变量覆盖。
  本层涉及的环境变量（Redis 池参数、`RELAY_POOL_CAP`、pprof/pyroscope、
  磁盘缓存阈值等）变更须同步 `.env.example` 与中英文环境变量文档。

## 验证

- `go build ./... && go test ./internal/infra/...`。
- 改 SSRF 防护 / 代理客户端后执行 `go test ./internal/infra/httpclient/...` 与
  relay 侧渠道测试。
- 改请求体存储/磁盘缓存后执行 `go test ./internal/httpapi/... ./internal/infra/cache/...`。

