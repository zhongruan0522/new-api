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
| `pricing/` | `pricingstore` | 定价缓存与刷新（含 model_extra 查询） |
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
- SQLite 不支持 `ALTER COLUMN`，迁移按现有 add-column/兼容模式处理。

## 缓存与配置

- `OptionMap`、channel cache、dynamic ratio cache 等全局缓存要注意锁、同步频率和多节点行为。
- 迁移和 cleanup 必须幂等，可重复运行。

## 测试

- 跨包 fixture 的测试用外部测试包（`package <pkg>_test`，如 `logstore_test`），
  避免内部测试包与被测包依赖成环；仅用本包未导出符号的测试保持内部测试包。
- 测试间复位批量暂存用 `dbstore.ResetBatchUpdateStores()`。

## 验证

- 改模型、迁移或缓存后执行相关 store 测试。
- 涉及 SQL 或迁移时至少做 SQLite 路径验证；能配置 MySQL/PostgreSQL 时补充对应验证。
- 跨层影响执行 `go test ./internal/store/... ./internal/service/... ./internal/controller/...`。
