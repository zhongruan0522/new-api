# model/AGENTS.md

`model/` 是 GORM 模型、迁移、缓存和数据库访问层。

## 数据库兼容

必须同时支持 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6。

- 优先使用 GORM 查询、更新、迁移能力。
- 原始 SQL 必须参数化，不能拼接外部输入。
- 保留字列、布尔值、引号、JSON 存储、ALTER 行为要处理三库差异。
- JSON 存储优先 `TEXT`，不要引入缺少回退方案的 JSONB/MySQL 专有能力。
- SQLite 不支持 `ALTER COLUMN`，迁移按现有 add-column/兼容模式处理。

## 缓存与配置

- `OptionMap`、channel cache、dynamic ratio cache 等全局缓存要注意锁、同步频率和多节点行为。
- 主题配置最终要同步到 `common.SetTheme`，只允许 `default` / `classic`。
- 迁移和 cleanup 必须幂等，可重复运行。

## 验证

- 改模型、迁移或缓存后执行相关 model 测试。
- 涉及 SQL 或迁移时至少做 SQLite 路径验证；能配置 MySQL/PostgreSQL 时补充对应验证。
- 跨层影响执行 `go test ./model/... ./service/... ./controller/...`。
