# internal/domain/

领域契约层：承载被 controller / service / store 多包共享的领域模型、错误与常量。
目标方向是各业务包依赖 domain，而不是互相依赖。

## 子包

| 子包            | 内容                                                                                     |
|----------------|------------------------------------------------------------------------------------------|
| `channel/`      | 渠道域契约（`channel_error.go` 的 `ChannelError`）。                                        |
| `channel/constant/` | 渠道域常量（`APIType*`、`ChannelType*`、`EndpointType`、`MultiKeyMode`、Azure 时间点、套餐 BaseURL 表）。 |
| `billing/`      | 计费域契约（`PriceData`、`ContextPricing*`、`GroupRatioInfo`）。后续阶段（PRD 5.3）会并入计费服务。 |
| `shared/`       | **过渡性收容包**：原 `dto/`+`types/` 中暂无明确归属的协议 DTO 与共享类型。                        |

## `shared/` 的纪律：只出不进

- `shared/` 是 `dto/`+`types/` 的过渡收容包，**不是长期归宿**。
- **新代码禁止向 `shared/` 添加文件**；新协议 DTO 优先放进对应领域子包（渠道协议族
  待 relay PRD 收口时归 `internal/relay/`，计费相关归 `billing/`）。
- 每次领域拆分时把可归位的文件迁出 `shared/`，持续收缩直至解散。

## 依赖方向约束

- `domain` 下的包是契约层，**不得 import** controller / service / model / middleware / router。
- `domain/channel/constant/`、`domain/channel/`、`domain/billing/` 必须保持叶子包
  （仅依赖标准库或 `pkg/`），任何业务 import 都会造成环。
- `domain/shared/` 目前依赖 `internal/common` 与 `internal/infra/log`（原 dto/types 的既有
  依赖），这是 `internal/constant` 四个跨领域文件暂缓并入 shared 的原因（详见
  `internal/constant/README.md`）；阶段 4 拆解 `common/` 时需消解此依赖。

## 检查清单

- 改动本层只做契约（结构体、常量、错误类型、纯函数），不写 I/O、不碰数据库。
- 迁移文件进/出时同步更新本文件子包表与根 `AGENTS.md` 的结构概览。
- 验证：`go build ./... && go test ./internal/domain/...`。
