# internal/domain/

领域层：承载被 controller / middleware / relay / store 多包共享的领域模型、错误、
常量与领域服务（阶段 5.3 起，原 `internal/service/` 的业务逻辑按资源迁入）。
目标方向是各业务包依赖 domain，而不是互相依赖。

## 子包

| 子包 | 内容 |
|---|---|
| `billing/` | ★ 计费核心：`service.go`（PreConsumeBilling / SettleBilling / BillingSession 会话）、`quota.go`（预扣/后扣/用量重试）、`pricing.go`（上下文阶梯计价）、`violation_fee.go`、`usage_helper.go`、`gemini_usage.go`、`log_info.go`、`funding_source.go`。 |
| `billing/contract/` | 计费契约叶子包：`PriceData`、`GroupRatioInfo`、`ContextPricing*`。供 config/ratio、store/pricing、relay 引用（见"依赖方向约束"）。 |
| `billing/plan_quota/` | 供应商套餐配额（GLM / Kimi / MiniMax）拉取与归一化。 |
| `channel/` | 渠道域服务：自动禁用/启用、加权随机选择与重试、渠道亲和性缓存；含 `channel_error.go`（`ChannelError`）。 |
| `channel/constant/` | 渠道域常量（`APIType*`、`ChannelType*`、`EndpointType`、`MultiKeyMode`、Azure 时间点、套餐 BaseURL 表）。 |
| `audit/` | 审计日志服务（`RecordAudit` 唯一入口），见 [audit/AGENTS.md](audit/AGENTS.md)。 |
| `rankings/` | 模型/供应商用量排行榜聚合。 |
| `ticket/` | 工单领域服务。 |
| `sensitive/` | 敏感词匹配（`AcSearch` / `SundaySearch`），调用方 relay 与渠道自动禁用。 |
| `group/` | 用户分组与分组倍率解析。 |
| `shared/` | **过渡性收容包**：原 `dto/`+`types/` 中暂无明确归属的协议 DTO 与共享类型。 |

## `shared/` 的纪律：只出不进

- `shared/` 是 `dto/`+`types/` 的过渡收容包，**不是长期归宿**。
- **新代码禁止向 `shared/` 添加文件**；新协议 DTO 优先放进对应领域子包（渠道协议族
  待 relay PRD 收口时归 `internal/relay/`，计费相关归 `billing/`）。
- 每次领域拆分时把可归位的文件迁出 `shared/`，持续收缩直至解散。

## 依赖方向约束

- `domain` 下的包**不得 import** controller / middleware / router；不得被 store 反向依赖。
- `domain/channel/constant/` 与 `domain/billing/contract/` 必须保持**契约叶子包**
  （仅依赖标准库或 `pkg/`）：config/ratio、store/pricing、relay 等引用方依赖它们取类型，
  一旦引入业务 import 就会与领域服务成环。这是契约从 `billing/` 根包下沉到
  `billing/contract/` 的原因（阶段 5.3，`billing` 服务依赖 store/config）。
- 领域服务包（`billing/`、`channel/` 等）可以依赖 store、config、infra，但领域包之间
  和对 infra 的依赖必须保持无环。当前既定方向：
  `billing → {channel, infra/notify, infra/tokenizer, infra/httpclient}`、
  `channel → {sensitive, group, infra/notify}`。新增依赖前先确认不成环。
- `domain/shared/` 目前依赖 `internal/common` 与 `internal/infra/log`（原 dto/types 的既有
  依赖），这是 `internal/constant` 四个跨领域文件暂缓并入 shared 的原因（详见
  `internal/constant/README.md`）；阶段 4 拆解 `common/` 时需消解此依赖。

## 检查清单

- 领域服务承载业务规则；契约子包只做契约（结构体、常量、错误类型、纯函数），不写 I/O、不碰数据库。
- 控制器输入已校验也不能假设内部状态可信；跨系统边界继续校验。
- 计费、quota、usage、渠道选择、动态倍率、违规费用等逻辑必须保持可追踪和可测试。
- 外部 HTTP 调用复用现有客户端和超时配置（`infra/httpclient`）；不要无超时请求或吞掉上游错误。
- 不要模拟成功或生成假 usage 来掩盖上游/业务失败。
- 迁移文件进/出时同步更新本文件子包表与根 `AGENTS.md` 的结构概览。

## 验证

- `go build ./... && go test ./internal/domain/...`。
- 改计费、quota、倍率、渠道选择逻辑后执行对应 billing/channel 测试与
  `go test ./internal/relay/... ./internal/controller/...`。
