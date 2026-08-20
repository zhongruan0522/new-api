# internal/infra/AGENTS.md

`internal/infra/` 是基础设施层（阶段 5.3 起，原 `internal/service/` 中无领域归属的
能力型实现按职责迁入；`log/` 由阶段 3 提前落位）。

## 子包

| 子包 | 内容 |
|---|---|
| `log/` | 日志工具（原 `logger/`）。 |
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
- infra 包之间可以按需单向依赖（当前既定方向：`tokenizer → media → httpclient`、
  `notify → {media, httpclient}`），新增依赖前确认无环；infra 不得 import
  controller / middleware / router / relay（`media` 不得引用 relay 包）。
  既有例外须保持现状并注明：`tokenizer/token_counter.go` 直接 import
  `relay/common`（`EstimateRequestToken`/`CountTokenRealtime` 的 `RelayInfo` 参数）
  与 `relay/constant`（`RelayFormat`/`RelayMode` 常量），系阶段 5.3 迁移前的既有
  签名依赖，待后续阶段收敛（`config/reasoning` 还会间接拉入
  `relay/channel/openrouter`）；除此之外不得新增 infra → relay 依赖。
- 待机内存相关默认值保守（连接池、缓存、后台任务），调大须保留环境变量覆盖。

## 验证

- `go build ./... && go test ./internal/infra/...`。
- 改 SSRF 防护 / 代理客户端后执行 `go test ./internal/infra/httpclient/...` 与
  relay 侧渠道测试。
