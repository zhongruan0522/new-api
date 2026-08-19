# constant 包 (`/internal/constant`)

跨领域的全局常量/全局配置变量。渠道域常量已迁至
`internal/domain/channel/constant/`，relay 域常量（`RelayMode`、`FinishReason`、`RelayFormat`）
在 `internal/relay/constant/`。

## 当前文件

| 文件              | 说明                                                                                     |
|-----------------|------------------------------------------------------------------------------------------|
| `cache_key.go`   | 缓存键格式字符串及 Token 相关字段常量，统一缓存命名规则。                                        |
| `context_key.go` | 定义 `ContextKey` 类型以及在整个项目中使用的上下文键常量（请求时间、Token/Channel/User 相关信息等）。 |
| `env.go`         | 环境配置相关的全局变量，在启动阶段由 `internal/common/init.go` 根据配置或环境变量注入。                     |
| `setup.go`       | 标识项目是否已完成初始化安装 (`Setup` 布尔值)。                                                    |

## 为什么这些文件还在这里（过渡期说明）

架构迁移 PRD（`docs/PRD/prd-architecture-migration.md` 阶段 5.1）原计划把它们并入
`internal/domain/shared/`，但执行时发现循环导入：

- `internal/domain/shared/`（原 `dto/`+`types/`）中的 `error.go`、`claude.go`、
  `openai_request.go` 依赖 `internal/common` 的工具函数；
- 而 `internal/common`（`gin.go`、`init.go`、`url_validator.go` 等）和 `internal/infra/log`
  又依赖本包的 `ContextKey` / env 变量。

同包合并会形成 `shared → common → shared` 的环，因此这四个文件留守本包，
待阶段 4（`common/` 拆解）/ 5.4（`gin.go` 归 `httpapi/`）落地时随之解散归位。

## 使用约定

1. 本包**只能被其他包引用**（import），**禁止引用项目内其他自定义包**，仅允许标准库。
2. 不允许在此目录编写业务流程、数据库操作、第三方调用相关逻辑。
3. 新增跨领域常量前先确认归属：渠道域进 `internal/domain/channel/constant/`，
   relay 域进 `internal/relay/constant/`；确属跨领域的才进本包，并在上表登记。
