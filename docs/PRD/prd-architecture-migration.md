# PRD：项目结构重构（脱离上游 + 业务边界治理）

> 版本：v3（整文重写：官方指南依据融入「依据」一节；前版校勘发现的映射错误已直接修入各阶段映射表，映射以本文为准）
> 日期：2026-08-15
> 关联文档：[prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md)、[prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)
> 前置条件：本 PRD 是 relay 层 SDK 迁移 / IR 重构的**结构前置**，必须在两份 relay PRD 启动前完成阶段 1–5，否则协议转换和计费重构会被旧 import 路径反复拖累。

## 概述

new-api 长期跟随上游 one-api 衍生分支合并，结构上形成了「扁平分层 + 一级包直挂」的形态。决定脱离上游、对核心计费和协议转换做重构后，当前结构暴露出三类阻碍：

1. **业务包扁平化**：`controller/` 64 文件、`service/` 68 文件、`model/` 69 文件、`common/` 58 文件，子系统的物理边界丢失，找不到"哪个文件属于哪个领域"。
2. **`common/` 角色过载**：JSON 封装、磁盘缓存、邮件、SSRF、加密、系统监控、业务全局变量同处一个包，靠 AGENTS 规则约束反向依赖，没有结构隔离。
3. **缺乏 `internal/` 隔离**：业务代码全部在 module 根，`pkg/cachex` 也能反向 import 业务包；没有标准 `cmd/internal/pkg` 三件套。

本 PRD 的目标是用一套**可独立验证的分阶段迁移路径**，把项目从「扁平分层单体」迁到「`internal/` 隔离 + 领域垂直切片 + pkg 严格无业务依赖」的现代化 Go 布局，为后续计费和协议转换重构提供干净的物理边界。

> 本 PRD 只覆盖**文件结构和包边界治理**，不修改业务逻辑。计费 / 协议转换 / IR 化的逻辑重构见 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)，本 PRD 完成阶段 1–5 后启动。

---

## 依据：官方指南与生态约定

结构决策的引用来源必须分层，避免把社区模板当官方标准背书：

| 层级 | 来源 | 对本 PRD 的约束 |
|---|---|---|
| **官方指南（go.dev）** | [Organizing a Go module](https://go.dev/doc/modules/layout)「Server project」 | server 是自包含二进制、通常无对外导出包 → 实现逻辑进 `internal/`、命令进 `cmd/`。本 PRD 阶段 1/3 与此一致 |
| **官方指南（go.dev）** | 同文「Basic command」 | 单命令仓库 `main.go` 放根目录是官方认可的写法。本 PRD 拆 `cmd/server` **超出官方建议**，属自加约束，动机是给 embed 载体（`internal/app/webdist/`）找合法落点并统一装配层 |
| **官方风格（Google Style Guide，非 Go 团队规范）** | [Go Style Best Practices](https://google.github.io/styleguide/go/best-practices)「Util Packages」「Package Size」 | 包名不应叫 `util`/`helper`/`common`——拆解 `common/` 的直接依据；包内文件平铺本身合规（`net/http`、`os` 即示范），问题是单包过大、未按资源拆 |
| **社区惯例（非官方）** | golang-standards/project-layout | `cmd/`+`internal/`+`pkg/` 三件套的流行出处，其 README 自声明非官方标准。仅作参考，不单独作为决策依据 |

两个反直觉点，明确**不**作为本 PRD 的动机，避免目标漂移：

1. **"根目录一级包多"不是官方反对的形态。** 官方布局指南刻意不规定目录清单，Go 生态审美是扁平优于深层级。本 PRD 收敛根目录的真正收益是 `internal/` 隔离和领域边界，不是"根目录变干净"。
2. **"一级包下文件平铺"不是官方反对的形态。** 风格指南只要求"文件聚焦到维护者能猜到内容在哪"。`controller/` 63 文件平铺的问题在**包没有按资源拆**（单包过大），阶段 5 的拆分依据是包规模和领域边界，不是文件摆放方式。

此外，目标布局自身有两处**有意保留的偏离**，执行时需持续评估：

- `internal/domain/shared/`：`shared` 仍是风格指南点名的低信息量命名（比 `common` 好得有限）。本 PRD 把它定位为**过渡性收容包**（吸收 `dto/`+`types/`），并以"只出不进"的纪律持续收缩（见阶段 5.1），而非长期归宿。
- `pkg/cachex`：全仓唯一 importer 是 `service/channel_affinity.go`，按官方定义不满足 `pkg/` 门槛（无外部使用者）。但其依赖干净（不 import 任何业务包），且阶段 2 后 `pkg/` 将形成纯工具库区，故保留在 `pkg/`，用完成标准中的 `go list -deps` 检查兜底：一旦长出业务依赖即降级 `internal/`。

## 设计原则

- **领域优先于层**：原按层切（controller/service/model），新结构在层内再按资源切，每个资源是 controller + store 的垂直切片，领域契约（domain）独立成公共层。
- **`internal/` 强制隔离**：`cmd/`、`web/`、`pkg/` 之外的全部业务代码进 `internal/`。
- **`pkg/` 严格无业务依赖**：只放真正可独立复用且依赖干净的库。进入 `pkg/` 前必须核查直接 import：不得出现 `internal/`、`constant/`、`types/`、`dto/`、`common/` 任何一处。
- **机械迁移优先**：先搬位置不改逻辑，业务逻辑重构留到阶段 6 之后的 relay PRD。
- **每阶段可回滚**：一阶段一 PR，每 PR 必须 `go build ./... && go test ./...` 通过。

---

## 目标布局

```
new-api/
├── cmd/
│   └── server/
│       └── main.go                  ← 启动入口，只做 os.Exit + 调 app.Run
├── internal/                        ← 所有业务代码，强制不可外部引用
│   ├── app/                         ← 启动装配 + 资源初始化
│   │   ├── bootstrap.go             ← 原 main.go 的 InitResources
│   │   ├── server.go                ← gin 装配 + 路由挂载
│   │   ├── analytics.go             ← Umami / GA 注入
│   │   ├── env.go                   ← 原 common/init.go + env.go
│   │   └── webdist/                 ← //go:embed web/dist 的载体
│   ├── httpapi/                     ← 原 router + middleware + controller
│   │   ├── router/
│   │   ├── middleware/
│   │   └── controller/              ← 按资源拆子包（见阶段 5.4）
│   ├── domain/                      ← 核心领域模型 + 错误 + 类型
│   │   ├── billing/                 ← ★ 计费领域（新核心）
│   │   ├── channel/                 ← 含 constant/（原 constant/ 的 channel 域常量）
│   │   ├── user/
│   │   ├── token/
│   │   ├── log/
│   │   ├── redemption/
│   │   ├── ticket/
│   │   ├── checkin/
│   │   ├── topup/
│   │   ├── audit/
│   │   ├── sensitive/               ← 敏感词匹配（service/sensitive.go + str.go）
│   │   ├── shared/                  ← 过渡性收容包：dto/ + types/ 中暂无明确归属的契约，只出不进
│   │   └── ...
│   ├── relay/                       ← ★ 协议转换领域（保留 relay 名）
│   │   ├── core/                    ← adaptor 调度 + websocket
│   │   ├── wire/                    ← OpenAI wire 协议族
│   │   ├── handler/                 ← 各模态 handler
│   │   ├── channel/                 ← 原 relay/channel/<provider>/ 不动
│   │   ├── common_handler/          ← 原位保留（rerank 处理）
│   │   ├── reasonmap/               ← 原位保留（推理链映射）
│   │   ├── common/
│   │   ├── helper/                  ← + service/error.go 的 relay 错误包装
│   │   └── constant/                ← + 原 constant/finish_reason.go 并入
│   ├── store/                       ← 原 model/，按资源拆
│   │   ├── db/
│   │   │   ├── migrate/             ← db_premigrate_* / db_same_type_migrate_*
│   │   │   ├── cleanup/             ← cleanup_removed_*
│   │   │   └── identity.go
│   │   ├── channel/
│   │   ├── user/
│   │   ├── token/
│   │   ├── log/
│   │   └── ...
│   ├── infra/                       ← 基础设施（只放无业务语义的实现）
│   │   ├── db/                      ← GORM 初始化
│   │   ├── redis/
│   │   ├── cache/                   ← common/disk_cache*
│   │   ├── httpclient/              ← service/http_client.go（通用传输层；渠道代理规则留 domain/channel）
│   │   ├── oss/                     ← stored_image / stored_video / stored_media
│   │   ├── email/                   ← common/email*
│   │   ├── oauth/                   ← 原 oauth/
│   │   ├── payment/                 ← stripe / epay
│   │   ├── passkey/                 ← 原 service/passkey
│   │   ├── tokenizer/               ← service/tokenizer* + token_counter + token_estimator
│   │   ├── media/                   ← file_decoder / image / audio / heif
│   │   ├── security/                ← ssrf / ip / url_validator / crypto / totp
│   │   ├── notify/                  ← user_notify / notify_limit / webhook
│   │   ├── runtime/                 ← system_monitor / pprof / pyro / gopool
│   │   └── log/                     ← 合并 logger/ + common/sys_log.go
│   ├── config/                      ← 原 setting/，按子域已分好
│   │   ├── manager/                 ← 原 setting/config/（ConfigManager，避免 internal/config/config 路径重复）
│   │   ├── system/                  ← 原 setting/system_setting
│   │   ├── dashboard/
│   │   ├── operation/
│   │   ├── model/
│   │   ├── ratio/
│   │   ├── performance/
│   │   ├── reasoning/
│   │   └── console/
│   └── i18n/
├── pkg/                             ← 可独立复用的库（无业务依赖，进入前须核查 import）
│   ├── cachex/                      ← 不动
│   └── jsonx/                       ← common/json.go
├── web/                             ← 前端不动
├── docs/
└── scripts/
```

> 与前版目标树的差异：`relay/common_handler/`、`relay/reasonmap/` 补全（阶段 5.5 原位保留）；`config/manager/` 新增（消解 `setting/config` 改名冲突）；`pkg/ginext` 移除（依赖核查不通过，见阶段 2）；`domain/constant/` 改为 `domain/channel/constant/`（见阶段 5.1）。

---

## 范围

迁移按 6 个阶段推进，建议执行顺序：阶段 1 → 2 → 3 → 5.1 → 5.2 → 5.3 → 5.4 → 4 → 5.5 → 6。把 `common/` 拆解（阶段 4）放到阶段 5 之后，是因为阶段 5 完成后 `common/` 剩余文件很多可以就近归位，减少阶段 4 工作量。

### 阶段 0：准备（不动业务）

- [ ] 建立 `docs/migration-target-layout.md`，固化上面目标树作为唯一真相源。
- [ ] 跑一遍 `go build ./... && go test ./...`，截 baseline，标记 skip / flaky 的测试。
- [ ] 统一文件命名：`controller/channel-billing.go` 等连字符文件名改成下划线；`channel_test_handler.go` 与 `channel_test.go` 容易混淆，重命名清理。
- [ ] 在 `docs/` 下建迁移清单（迁移单元 → 目标路径 → 验证命令）。

**验证**：`go build ./... && go test ./...` 与 baseline 一致。

### 阶段 1：入口和装配外移（低风险）

把根 `main.go` 拆成：

```
cmd/server/main.go             ← 只 os.Exit + 调 app.Run
internal/app/bootstrap.go      ← 原 InitResources() 包一层
internal/app/server.go         ← gin 装配 + router.SetRouter
internal/app/analytics.go      ← InjectUmamiAnalytics / InjectGoogleAnalytics
```

`main.go` 里的 `model.InitDB` / `common.InitRedisClient` / `i18n.Init` 等调用原地不动，只是被 `app.Bootstrap()` 包了一层。`//go:embed web/dist` 移到 `internal/app/webdist/`，因为 embed 不支持 `..` 相对路径。

**只动 main.go**，业务包路径不变。

注意：`web/dist` 本身不移动。Dockerfile 第 16 行 `test -f web/dist/index.html` 依赖该目录原地存在；embed 载体迁移只改 Go 侧声明位置，不影响构建产物检查。

**验证**：`go run ./cmd/server` 能起，`go build ./...` 通过。

### 阶段 2：`pkg/` 抽离（无业务依赖，最安全）

`pkg/` 的进入门槛是**依赖核查通过**：候选文件不得 import `constant/`、`types/`、`dto/`、`common/` 或任何业务包。据此核查 `common/` 候选：

| 来源 | 目标 | 依赖核查结果 |
|---|---|---|
| `common/json.go` | `pkg/jsonx/` | 仅标准库（bytes / encoding/json / io），通过 |
| `common/body_storage.go`（含 `gin_body_storage_test.go`） | `pkg/ginext/` | 仅标准库，通过 |
| `pkg/cachex/` | 不动 | 依赖干净 |

> 前版曾把 `common/gin.go`、`common/response.go`、`common/request_body_limit.go` 列入 `pkg/ginext` 候选。核查发现三者均 import `constant`（ContextKey / MaxTextResponseBodyMB / AnonymousRequestBodyLimitKB 等），不满足 `pkg/` 无业务依赖约束。**改为随阶段 3 进 `internal/`**：`gin.go` + `request_body_limit.go` 是 HTTP 边界工具，阶段 5.4 归 `internal/httpapi/`；`response.go` 是 relay 侧响应体读取，阶段 5 按调用方归位。待 `constant` 依赖消解后才可重新评估抽 `pkg/`。因此目标布局中**不设 `pkg/ginext`**，`body_storage.go` 并入 `pkg/jsonx` 同级新包 `pkg/ginext/` 或暂留 `internal/`，执行时按其测试文件对 gin 的依赖情况二选一。

抽完后 `common/` 还剩业务工具（database / redis / sys_log / init / env / constants / model / quota / topup_ratio / timezone / trusted_proxies / pprof / pyro / system_monitor* / gopool* / go_channel / page_info / performance_config / audio / api_type / endpoint_* / crypto / email* / embed_file_system / ip / rate_limit / ssrf_protection / totp / url_validator / validate / verification / custom_event / disk_cache* / limiter/ / response / gin / request_body_limit / str / utils / hash / copy / condition 等）—— 等阶段 4 一起处理。

**验证**：`go build ./... && go test ./pkg/...`；`go list -deps ./pkg/...` 输出不出现 `internal/`、`constant`、`types`、`dto`、`common` 路径。

### 阶段 3：`internal/` 外壳 + 业务包整体平移（机械迁移）

把所有业务包从根目录搬进 `internal/`，**内部结构不变**，仅路径前缀变化：

| 原路径 | 新路径 |
|---|---|
| `common/` | `internal/common/` |
| `controller/` | `internal/controller/` |
| `service/` | `internal/service/` |
| `model/` | `internal/model/`（阶段 5.2 再改成 `internal/store/`） |
| `middleware/` | `internal/middleware/`（阶段 5 后并入 `internal/httpapi/`） |
| `router/` | `internal/router/`（同上） |
| `setting/` | `internal/config/`（**这一步顺便改名 + 去掉 `*_setting` 后缀**） |
| `setting/config/` | `internal/config/manager/`（**改名消歧**，见下） |
| `relay/` | `internal/relay/` |
| `oauth/` | `internal/oauth/` |
| `dto/` | `internal/dto/`（阶段 5.1 并入 `domain/`） |
| `types/` | `internal/types/`（同上） |
| `constant/` | `internal/constant/`（阶段 5.1 按域拆入 `domain/`） |
| `i18n/` | `internal/i18n/` |
| `logger/` | `internal/infra/log/`（**一步到位**，见下） |

注意事项：

- module path 不变（仍是 `github.com/NookMux/NookMux`），只是包路径多一层 `internal/`。Go 的 import fix 由 IDE / `gofmt -r` 完成。
- `setting/` 改名 `config/` 是顺带的，子目录如 `system_setting/` 同步去掉后缀变成 `config/system/`，避免 `internal/config/system_setting` 这种冗余命名。
- **`setting/config/` 必须同步改名**：它是被 20+ 文件 import 的 ConfigManager（`setting/config/config.go`），若只做整体改名会出现 `internal/config/config` 路径重复。改为 `internal/config/manager/`，包名 `manager`。
- **`logger/` 不走"先平移后合并"**：`logger/logger.go` 仅 1 个文件，平移到 `internal/logger/` 再在阶段 4 搬一次是纯 churn。直接落到 `internal/infra/log/`，阶段 4 把 `common/sys_log.go` 并入同包，全仓只改一次 import。
- `web/dist` 的 embed 已在阶段 1 处理好。

**验证**：`go build ./... && go test ./...` 必须全绿。完成此阶段后业务代码已在 `internal/` 壳里，子包结构仍和现状一样乱，由阶段 5 处理。

### 阶段 4：`common/` 拆解（高风险，分多个子 PR）

`common/` 被全项目引用（282 处 import），拆它会触发最多 import 变更。按用途归口：

| common 文件 | 去向 |
|---|---|
| `database.go` / `redis.go` / `init.go` / `env.go` | `internal/infra/db/`、`internal/infra/redis/`、`internal/app/env.go` |
| `disk_cache*.go` / `limiter/` | `internal/infra/cache/`（和 `pkg/cachex` 区分：cachex 是底层库，这是业务缓存） |
| `email*.go` | `internal/infra/email/` |
| `embed_file_system.go` | `internal/app/webdist/` |
| `ssrf_protection.go` / `ip.go` / `url_validator.go` / `crypto.go` / `totp.go` / `verification.go` / `validate.go` / `trusted_proxies.go` | `internal/infra/security/` |
| `system_monitor*.go` / `pprof.go` / `pyro.go` / `gopool*.go` / `go_channel.go` | `internal/infra/runtime/` |
| `sys_log.go` | `internal/infra/log/`（并入阶段 3 已就位的 `logger/`） |
| `gin.go` / `request_body_limit.go` | `internal/httpapi/`（HTTP 边界工具 + ContextKey；若阶段 5.4 已建 httpapi 则顺路，否则先留 `internal/common/`） |
| `response.go` | relay 侧响应体读取，按调用方归 `internal/relay/` 或暂留 |
| `hash.go` / `copy.go` / `str.go` / `utils.go` / `condition.go` / `rate_limit.go` | 依赖干净，可评估进 `pkg/`（如 `pkg/jsonx` 旁）或随调用方归位；`condition.go` 仅被 `setting/operation_setting/tool_billing.go` 使用，优先评估归 setting 侧 |
| `constants.go` / `model.go` / `quota.go` / `topup_ratio.go` / `audio.go` / `api_type.go` / `endpoint_*.go` / `performance_config.go` / `timezone.go` / `page_info.go` / `custom_event.go` | **暂时留 `internal/common/`**，等阶段 5 拆领域时各自归位 |

阶段 4 完成后 `internal/common/` 应该只剩"业务全局变量和零碎工具"，体积减半。建议按 `infra/db`、`infra/email`、`infra/security` 等拆成多个小 PR 逐个推。

**回退策略**：拆 `common/` 时可以先用 type alias 在 `internal/common/` 里转一层（如 `type RedisClient = infra.RedisClient`），让旧 import 不全断，等下游都改完再删 alias。

**验证**：每个子 PR 跑 `go build ./... && go test ./...`。

### 阶段 5：领域垂直切片（核心，给计费 / 协议转换铺路）

把 `controller/`、`service/`、`model/` 三个大包按资源拆成垂直子包。

#### 5.1 抽领域契约层 `internal/domain/`

把"被多包共享的领域模型和错误"先抽出来，让 controller / service / store 都依赖 domain 而不是互相依赖。

`internal/constant/` 按内容归属拆（不做整体 `domain/constant` 平移——`constant` 是风格指南点名的语法角色命名，且内容并非同域）：

| 原 constant 文件 | 去向 |
|---|---|
| `api_type.go` / `azure.go` / `channel.go` / `endpoint_type.go` / `multi_key_mode.go` | `internal/domain/channel/constant/`（均为渠道域常量） |
| `finish_reason.go` | `internal/relay/constant/`（并入现有包） |
| `context_key.go` | 随 `gin.go` 归 `internal/httpapi/`（过渡期暂 `domain/shared/`） |
| `cache_key.go` / `env.go` / `setup.go` | `internal/domain/shared/` |

`internal/dto/` + `internal/types/` 机械合并进 `internal/domain/shared/`，随后按明确归属疏散：

| 文件 | 疏散目标 |
|---|---|
| `types/channel_error.go` | `internal/domain/channel/` |
| `types/context_pricing.go` / `types/price_data.go` | `internal/domain/billing/` |
| `types/relay_format.go` | `internal/relay/` |
| 其余（`error.go` / `request_meta.go` / `rw_map.go` / `set.go` / `file_data.go` / `file_source.go` + `dto/` 协议族） | 暂留 `domain/shared/` |

`domain/shared/` 的纪律：**只出不进**。新代码禁止向 `shared/` 添加文件；每次领域拆分时把可归位的文件迁出。它是 `dto/`+`types/` 的过渡收容包，不是长期归宿。

这一步完成后再做 5.2 / 5.3 / 5.4，因为 controller 子包要 import domain 而不是 import 整个 dto。

**验证**：`go build ./... && go test ./...`。

#### 5.2 `model/` → `internal/store/`，按资源拆

按现有文件名前缀归类（左侧为 model 现有文件，右侧为目标子包）：

```
internal/store/
├── db/
│   ├── migrate/                  ← service/db_premigrate_* 全部下沉
│   ├── cleanup/                  ← model/cleanup_removed_* 下沉
│   ├── identity.go               ← service/db_identity.go
│   └── init.go                   ← model/main.go + model.InitDB
├── channel/                      ← channel*.go + ability.go + channel_cache.go + channel_satisfy.go + channel_api_type_select.go + channel_request_format_preference.go + dynamic_ratio_*.go
├── user/                         ← user*.go + user_cache.go
├── token/                        ← token*.go + token_cache.go + token_window.go + token_quota_snapshot.go
├── log/                          ← log*.go + log_client_header_migration.go
├── pricing/                      ← pricing*.go + pricing_default.go + pricing_refresh.go
├── option/                       ← option*.go + setup*.go
├── redemption/                   ← redemption.go
├── ticket/                       ← ticket.go
├── topup/                        ← topup.go + topup_search_test.go
├── checkin/                      ← checkin.go
├── usedata/                      ← usedata*.go + usedata_rankings.go
├── audit/                        ← audit_log.go
├── twofa/                        ← twofa.go
├── passkey/                      ← passkey.go
├── minimax_voice/                ← minimax_voice*.go
├── missing_models/               ← missing_models.go
├── prefill_group/                ← prefill_group.go
├── stored_media/                 ← stored_image.go + stored_video.go + stored_media.go
├── vendor_meta/                  ← vendor_meta.go + model_meta.go
└── model_extra/                  ← model_extra.go
```

`model/errors.go`（ErrDatabase 等存储层错误）随 `store/db/`；`model/utils.go`（批量更新器）随 `store/db/`；`data_migration_marker.go` 随 `store/db/migrate/`。

**验证**：`go build ./... && go test ./internal/store/...`。

#### 5.3 `service/` → 按资源 / 能力拆

```
internal/
├── domain/
│   ├── billing/                  ← ★ 计费核心
│   │   ├── service.go            ← service/billing.go + billing_session.go
│   │   ├── quota.go              ← service/quota.go
│   │   ├── pricing.go            ← service/context_pricing.go
│   │   ├── violation_fee.go      ← service/violation_fee.go
│   │   ├── usage_helper.go       ← service/usage_helpr.go
│   │   ├── gemini_usage.go       ← service/gemini_usage.go
│   │   ├── log_info.go           ← service/log_info_generate.go
│   │   ├── funding_source.go     ← service/funding_source.go
│   │   └── plan_quota/           ← service/plan_quota_*.go 全部
│   ├── channel/                  ← service/channel.go + channel_select.go + channel_affinity.go + http.go（NewProxyHttpClient 渠道代理规则）
│   ├── user/                     ← service/user_notify.go + ...
│   ├── audit/                    ← service/audit.go（RecordAudit 入口）
│   ├── rankings/                 ← service/rankings.go
│   ├── ticket/                   ← service/ticket.go
│   ├── sensitive/                ← service/sensitive.go + str.go（AcSearch / SundaySearch 敏感词匹配，唯一调用方 relay/channel/openai/audio.go）
│   └── group/                    ← service/group.go
├── infra/
│   ├── httpclient/               ← service/http_client.go + http_client_test.go（通用传输层；渠道代理选择是业务规则，归 domain/channel，不得混入 infra）
│   ├── payment/                  ← service/epay.go（stripe 在 controller/topup_stripe.go，挪过来）
│   ├── media/                    ← service/image.go + image_heif_test.go + audio.go + file_decoder.go + file_service.go + download.go + convert.go
│   ├── passkey/                  ← service/passkey/
│   ├── notify/                   ← service/notify_limit.go + user_notify.go + webhook.go
│   ├── tokenizer/                ← service/tokenizer.go + token_estimator.go + token_counter.go（三者同域，唯一归属，勿拆两处）
│   └── custom_voice/             ← service/custom_voice.go + custom_voice_test.go
├── relay/
│   └── helper/                   ← service/error.go（RelayErrorHandler / ClaudeErrorWrapper 等 relay 错误包装，调用方为 controller + relay）
```

> 与前版差异：`service/error.go`、`service/str.go` 补齐归属；`token_counter.go` 消除双重映射（前版同时列在 `domain/token/` 和 `infra/tokenizer/`）；`http.go` 与 `http_client.go` 拆开——前者是"每渠道代理拨号"业务规则（见 `service/AGENTS.md` 的代理约束），后者才是可进 infra 的传输层。`domain/token/` 因此没有 service 侧内容，仅保留 store 侧切片。

**验证**：`go build ./... && go test ./internal/domain/... ./internal/infra/... ./internal/relay/...`。

#### 5.4 `controller/` → 按资源拆

```
internal/httpapi/controller/
├── channel/             ← channel.go + channel_proxy.go + channel_billing.go + channel_affinity_cache.go + channel_test_handler.go + channel_fetch_models_headers_test.go + missing_models.go
├── user/                ← user.go + login_test.go + email_bind_test.go + password_reset_test.go + status_user_modules_test.go + user_manage_quota_test.go + user_access_token_test.go + user_batch_update_test.go
├── token/               ← token.go + token_feedback.go
├── billing/             ← billing.go + pricing.go + pricing_anon_test.go
├── topup/               ← topup.go + topup_stripe.go
├── redemption/          ← redemption.go
├── checkin/             ← checkin.go
├── ticket/              ← ticket.go
├── audit/               ← audit.go
├── rankings/            ← rankings.go
├── oauth/               ← oauth.go + oauth_test.go
├── passkey/             ← passkey.go
├── twofa/               ← twofa.go
├── option/              ← option.go + option_test.go + dashboard_config.go + console_migrate.go
├── db_migrate/          ← db_premigrate.go + db_same_type_migrate.go
├── dynamic_ratio/       ← dynamic_ratio.go
├── prefill_group/       ← prefill_group.go
├── group/               ← group.go
├── model/               ← model.go + model_meta.go + model_sync.go + model_test.go + model_sync_test.go
├── vendor_meta/         ← vendor_meta.go
├── relay/               ← relay.go + relay_test.go + relay_retry_test.go
├── log/                 ← log.go
├── performance/         ← performance.go
├── playground/          ← playground.go
├── stored_media/        ← stored_media.go
├── usedata/             ← usedata.go + usedata_test.go
├── uptime_kuma/         ← uptime_kuma.go
├── image/               ← image.go
├── custom_voice/        ← custom_voice.go + minimax_voice.go
├── secure_verification/ ← secure_verification.go + secure_verification_test.go
├── setup/               ← setup.go
└── misc/                ← misc.go
```

> 与前版差异：`model_sync.go` 消除双重映射（前版同时列在 `channel/` 和 `model/`）——其内容是 `SyncUpstreamModels`（从上游拉取模型/厂商元数据），与 `channel_test_handler.go`（渠道连通性测试）不同域，唯一归属 `model/`；`log.go` 补齐归属。

同时把 `internal/router/` + `internal/middleware/` + `internal/controller/` 整体上移到 `internal/httpapi/` 下。

**验证**：`go build ./... && go test ./internal/httpapi/...`。

#### 5.5 `relay/` 内部精修（不动外部 import 路径）

`relay/` 已经做得不错，主要是把顶层散落的 `openai_wire_*` 和 handler 收口：

```
internal/relay/
├── core/                ← relay_adaptor.go + websocket.go + stored_asset_signature*.go
├── wire/                ← openai_wire_* 全部下沉
│   ├── convert/         ← openai_wire_convert_* （原在 relay/common/，挪过来）
│   ├── stream/          ← openai_wire_stream_* + openai_wire_capture_writer.go
│   └── auto_convert.go
├── handler/             ← audio_handler.go + claude_handler.go + compatible_handler.go + embedding_handler.go + gemini_handler.go + image_handler.go + rerank_handler.go + responses_handler.go + stored_image_handler.go + stored_video_handler.go + third_party_media_to_text*.go
├── channel/             ← 不动
├── common_handler/      ← 原位保留（rerank.go + rerank_error_test.go）
├── reasonmap/           ← 原位保留（reasonmap.go）
├── common/              ← 留 billing.go + image_handling.go + media_text_handling.go，openai_wire_convert_* 挪到 wire/convert/
├── helper/              ← 不动（阶段 5.3 已并入 service/error.go 的错误包装）
└── constant/            ← 不动（阶段 5.1 已并入 finish_reason.go）
```

> `relay/common/billing.go` 是计费和协议转换的交汇点，阶段 5 不要动它，留到阶段 6 / relay IR 重构 PRD 一起处理。

**验证**：`go test ./internal/relay/...`。

### 阶段 6：计费 + 协议转换边界就位（这才动逻辑，但只做边界迁移）

前面 5 个阶段做完后，已经有了：

```
internal/domain/billing/         ← 计费的纯领域模型 + 服务
internal/relay/wire/             ← OpenAI wire 协议转换的独立画布
internal/relay/common/billing.go ← 计费在 relay 侧的入口（待消解）
```

阶段 6 只做边界迁移，不动计费 / 协议转换的实际算法（那是 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md) 的事）：

1. **计费边界**：把 `relay/common/billing.go` 的 usage 计算逻辑下沉到 `domain/billing/`，relay 只调 `domain/billing.CalculateUsage(meta, rawUsage)` 和 `domain/billing.ApplyQuota(...)`，不让协议层接触 quota 表。
2. **协议转换边界**：`relay/wire/convert/` 里 chat ↔ responses 互转整理成单点入口，为后续 IR 化留接口位（实际 IR 化见 relay IR PRD）。

**验证**：`go build ./... && go test ./...` 全绿，行为与阶段 5 完全一致（无逻辑变更）。

---

## 风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| `common/` 拆解触发全项目 import 变更 | 编译大面积失败 | 阶段 4 拆成多个小 PR；type alias 兜底，旧 import 不全断 |
| `internal/` 平移后 embed 路径失效 | `web/dist` 无法嵌入 | 阶段 1 把 embed 移到 `internal/app/webdist/`，避免 `..` 路径；`web/dist` 目录本身不移动（Dockerfile 依赖其原地存在） |
| 一阶段一 PR 之间互相依赖卡死 | 中途无法编译 | 严格按建议顺序（1→2→3→5.1→5.2→5.3→5.4→4→5.5→6）推，每阶段独立可编译 |
| AGENTS.md 引用旧路径 | AI 协作 / 人工 review 引用错位置 | 每阶段同步更新对应 AGENTS.md 的路径引用 |
| module path 改动破坏外部引用 | Docker / CI / heroku buildpack 失效 | 本 PRD **不改** module path，只增加 `internal/` 前缀；module path 改名留到结构稳定后单独 commit |
| 阶段 5.2 / 5.3 / 5.4 同名文件跨子包冲突 | 包内符号重复定义 | 按资源切包后，同资源文件聚合到一个子包；跨资源同名（如多个 `errors.go`）按子包隔离，不冲突 |
| `setting/` 改名 `config/` 触发配置键路径假设 | 配置读取代码假设包名 | 阶段 3 顺手改但保留所有 option key 字符串不变，仅改 Go 包路径；`setting/config/` 同步改名 `config/manager/` 消歧 |
| `pkg/` 候选依赖核查不严（如前版 `gin.go` 踩坑） | `pkg/` 长出业务依赖，破坏"可独立复用"定位 | 阶段 2 及后续每次 `pkg/` 变更跑 `go list -deps ./pkg/...` 检查；完成标准兜底 |
| 前端依赖后端路径 | 重构破坏前端调用 | 路由 URL（`/api/...`）不变，仅 Go 包路径变化，前端零影响 |

## 完成标准

- [ ] 阶段 0–6 全部完成，每个阶段独立 PR 已合并
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全绿（与阶段 0 baseline 一致或更好）
- [ ] `internal/common/` 仅剩业务全局变量，体积较初始减少 ≥ 50%
- [ ] `controller/`、`service/`、`model/` 不再以扁平 60+ 文件形式存在，按资源拆成子包
- [ ] `pkg/` 下无任何 `internal/`、`constant/`、`types/`、`dto/`、`common/` 依赖：`go list -deps ./pkg/...` 输出不出现上述路径；`pkg/cachex` 依赖干净故保留，若后续长出业务依赖则降级 `internal/`
- [ ] `domain/shared/` 无新增文件（只出不进纪律生效），channel / billing / relay 域契约已各自归位
- [ ] 所有 AGENTS.md 路径引用已同步更新
- [ ] `docs/migration-target-layout.md` 与最终代码结构一致
- [ ] 计费（`internal/domain/billing/`）和协议转换（`internal/relay/wire/`）边界就位，可启动 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)

---

## 分支与版本控制

```
main
  └── 架构重构分支（architecture-migration）
       ├── 阶段 0：准备 + 文件名统一
       ├── 阶段 1：cmd/server + internal/app 抽离
       ├── 阶段 2：pkg/jsonx 抽离（依赖核查通过的候选）
       ├── 阶段 3：internal/ 外壳平移 + setting → config 改名（含 config/manager 消歧、logger 一步到位）
       ├── 阶段 5.1：domain 契约层抽取（constant 按域拆 + dto/types 合并疏散）
       ├── 阶段 5.2：model → store 按资源拆
       ├── 阶段 5.3：service 按资源 / 能力拆
       ├── 阶段 5.4：controller 按资源拆 + httpapi 聚合
       ├── 阶段 4：common 拆解（分多个子 PR）
       ├── 阶段 5.5：relay/wire + relay/handler 收口
       └── 阶段 6：计费 / 协议转换边界就位
            └── 合并回 main → 启动 relay IR 重构 PRD
```

**衔接规则**：

- 本 PRD 是 relay SDK 迁移 / IR 重构两份 PRD 的**结构前置**。
- 阶段 1–5 完成后即可启动 [prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md)。
- 阶段 6 完成后即可启动 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)。
- 每个阶段合并后保证 main 始终可发布。

---

## 不在范围内

以下内容明确不在本 PRD 范围内：

- 业务逻辑重构（计费算法、协议转换算法、IR 化）—— 见 [prd-relay-ir-refactor.md](./prd-relay-ir-refactor.md)
- adaptor 接口 Convert 方法迁移到 SDK params —— 见 [prd-relay-sdk-migration.md](./prd-relay-sdk-migration.md)
- module path 改名（`github.com/NookMux/NookMux` → 其他）—— 留到结构稳定后单独 commit
- AGPL / 版权头 / Docker 镜像名 / CI workflow 名等元数据变更
- 数据库 schema 变更
- 路由 URL 变更（`/api/...`、`/v1/...` 对外路径不变）
- 前端结构调整（`web/` 已经是 feature-based，不动）
- 新增渠道类型
- 模型倍率数值调整
- 审计日志逻辑变更
- 删除现有功能或渠道

---

## 修订记录

- **v1**（2026-08-15）：初版，确立六阶段迁移路径与目标布局。
- **v2**（2026-08-15）：附加「与官方指南的对齐」章节与 v1 映射校勘旁注（未改正文映射表）。
- **v3**（2026-08-15，当前）：整文重写。官方依据分层融入「依据」与「设计原则」；v2 校勘修正直接落入各阶段映射表——`pkg/ginext` 候选因依赖 `constant` 撤销、`model_sync.go` 与 `token_counter.go` 消除双重映射、`service/error.go` 与 `str.go` 补齐归属、`setting/config` 改名冲突消解为 `config/manager/`、`logger/` 一步到位、`relay/common_handler` 与 `reasonmap` 补全、`domain/constant` 改为按域拆分并为 `domain/shared` 设立"只出不进"纪律。
