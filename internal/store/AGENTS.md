# internal/store/AGENTS.md

`internal/store/` 是原 `internal/model/` 按资源垂直拆分后的持久层：GORM 模型、
查询、缓存、迁移与数据清理。本文件规则适用于所有子包。

## 目录结构与包命名

子包按资源拆分；包名统一带 `store` 后缀（避免与调用方常见的局部变量
`user`/`token`/`channel`/`log`/`db` 冲突），`db/` 系子包以 `db` 前缀命名：

| 目录 | 包名 | 职责 |
|---|---|---|
| `db/` | `dbstore` | `DB`/`LOG_DB` 句柄、列名方言变量、连接初始化、批量更新器 |
| `db/migrate/` | `dbmigrate` | `InitDB`/`InitLogDB` 编排、AutoMigrate、pre-migrate / 同类型迁移、日志头回填 |
| `db/cleanup/` | `dbcleanup` | 一次性历史数据清理（cleanup_removed_*） |
| `channel/` | `channelstore` | 渠道、ability、动态倍率 |
| `user/` | `userstore` | 用户、用户缓存、用户动作日志（RecordLog*） |
| `token/` | `tokenstore` | 令牌、令牌缓存、窗口/周期配额 |
| `log/` | `logstore` | 消费/错误日志查询与统计 |
| `pricing/` | `pricingstore` | 定价缓存、模型广场刷新与 [model_price_table.go](pricing/model_price_table.go) 的组件价格表持久化（含旧 ratio 的只读投影） |
| `option/` | `optionstore` | option KV、setup 记录、数据迁移 marker |
| 其余单资源目录 | `<资源>store` | redemption/ticket/topup/checkin/usedata/audit/twofa/passkey/minimax_voice/missing_models/prefill_group/stored_media/vendor_meta |

`vendormetastore`（`vendor_meta/`）持有 `Model`/`Vendor` 模型元数据，是大多数
资源包的公共依赖；`dbstore` 不依赖任何资源包（分层最底层）。

## 分层与依赖方向

- `dbstore` 不 import 任何资源子包；资源子包 → `dbstore` 取 `DB`/`LOG_DB`/列名变量。
- `InitDB`/`InitLogDB`/AutoMigrate 编排在 `dbmigrate`（引用全部资源模型，放
  `dbstore` 会成环）；启动装配从 `dbmigrate` 调用。
- 批量更新器机制（stores/locks/定时 flush）在 `dbstore`，各资源包在 `init()`
  里通过 `dbstore.RegisterBatchFlushers` 注册自己的落库函数（写入与注册同包，
  保证 flusher 永不缺失；nil 时按类型报错丢弃，不静默）。
- `user ↔ log` 的历史循环以"用户动作日志随用户域"解：`RecordLog`/
  `RecordLogWithAdminInfo` 在 `userstore`；logstore 只保留查询/消费/错误日志。
- 一次性数据迁移 marker（`IsDataMigrationDone`/`MarkDataMigrationDone`）在
  `optionstore`（直接读写 options 表）；`dbcleanup` 与 `dbmigrate` 复用。

## 数据库兼容

必须同时支持 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6。

- 优先使用 GORM 查询、更新、迁移能力。
- 原始 SQL 必须参数化，不能拼接外部输入。
- 保留字列（`group`/`key`）用 `dbstore.CommonGroupCol`/`CommonKeyCol`/`LogGroupCol`
  拼接（由 `dbstore.InitCol` 按方言初始化），不要手写反引号/双引号。
- 保留字列、布尔值、引号、JSON 存储、ALTER 行为要处理三库差异。
- JSON 存储优先 `TEXT`，不要引入缺少回退方案的 JSONB/MySQL 专有能力。
- TEXT 列不要带 `default` tag：MySQL 拒绝为 TEXT/BLOB 列设字面默认值
  （Error 1101），空库 AutoMigrate 建表会直接失败（详见
  `internal/store/token/token.go` 的 ModelLimits 与 `internal/store/log/log.go`
  的 ua/x_title 列注释）；可空 JSON text 列用 `*string` 指针（如
  `Log.BillingDetails`）。
- SQLite 不支持 `ALTER COLUMN`，迁移按现有 add-column/兼容模式处理。

## 缓存与配置

- `OptionMap`、channel cache、dynamic ratio cache 等全局缓存要注意锁、同步频率和多节点行为。
- 组件价格表的 `model_price_plans`/`model_price_components` 必须保持分表关系；[model_price_table.go](pricing/model_price_table.go) 的全量替换必须在一个事务中完成，提交后失效表缓存并调用 `RefreshPricing` 重建模型广场。不得写回或删除旧 `ModelRatio`/`ModelPrice`/缓存和音频 ratio 配置。
- 新增组件价格表物理模型时，同步更新主库 `AutoMigrate`、目标库 schema 与 pre-/same-type migration 的复制步骤；父表必须先于组件子表复制。
- 迁移和 cleanup 必须幂等，可重复运行。
- 从节点不执行迁移。新增物理模型或列后，从节点启动必须校验共享主库/独立日志库
  已具备对应表或列；缺失时显式阻断启动，避免 GORM 读写在滚动升级中静默退化或丢失
  数据（如 `billing_details` 的 `ensureLogBillingDetailsColumn` 与组件价格表的
  `ensureModelPriceTableTables`）。

## 测试

- 跨包 fixture 的测试用外部测试包（`package <pkg>_test`，如 `logstore_test`），
  避免内部测试包与被测包依赖成环；仅用本包未导出符号的测试保持内部测试包。
- 测试间复位批量暂存用 `dbstore.ResetBatchUpdateStores()`。

## 验证

- 改模型、迁移或缓存后执行相关 store 测试。
- 涉及 SQL 或迁移时至少做 SQLite 路径验证；能配置 MySQL/PostgreSQL 时补充对应验证。
- 跨层影响执行 `go test ./internal/store/... ./internal/domain/... ./internal/httpapi/controller/...`。

- Token 明细迁移完成标记位于实际日志库的 `log_billing_migration_states`，不是主库 options；slave 须检查完成标记。升级须停止并排空旧版本写入，具体顺序及重试边界见 [计费 PRD](../../docs/PRD/计费.md#43-迁移与兼容)。
