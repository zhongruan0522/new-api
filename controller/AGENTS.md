# controller/AGENTS.md

`controller/` 是 HTTP 边界层，负责输入校验、权限检查、调用 service/model、组织响应。

## 规则

- 外部输入必须在这里或更近边界校验：path/query/body/form/file/header。
- 控制器不要沉淀复杂业务逻辑；可复用业务放 `service/`，持久化放 `model/`。
- 响应结构保持现有 `{ success, message, data }` 风格，避免为单个前端新增不兼容格式。
- 不要为了 default UI 改后端业务 API。字段不匹配时优先改前端适配本项目接口。
- `theme.frontend` 只允许 `default` 或 `classic`，并要同步 `setting/system_setting` 与 `common`。
- 安全相关控制器要保留二次验证、角色校验、限速和审计日志。

## 验证

- 改请求校验、权限或响应字段后执行对应 controller 测试。
- 影响系统设置、登录、渠道、令牌、计费、文件或 relay 入口时执行 `go test ./controller/... ./service/...`。
