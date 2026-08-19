# i18n/AGENTS.md

`i18n/` 负责后端 API 响应消息的多语言翻译（go-i18n）。

## 适用范围

只处理后端返回给前端的 JSON 响应 `message` 字段（错误/成功提示）。前端 UI 文案
的 i18n 见 `web/AGENTS.md`，两者互不重叠：后端翻译自己决定要返回的提示语，前端
对后端 message 原样展示，不要二次翻译。

## 规则

- 用户可见的 API 错误/成功提示必须走 i18n，用 `i18n.Msg*` 常量配合
  `common.ApiErrorI18n` / `common.ApiSuccessI18n` / `i18n.T` 返回，不要硬编码
  中英文字符串。
- 消息 key 必须定义在 `i18n/keys.go` 的常量中并按模块分组（`MsgUser*`、
  `MsgToken*` 等），不要在调用处写字面量 key。
- 翻译文件为 `locales/<locale>.yaml`（`<locale>` 为 `en` 或 `zh`，即
  `locales/en.yaml` 和 `locales/zh.yaml`），扁平 `key: "value"` 结构。后端为单一
  文件，所有 key 集中在对应 locale 的 yaml 中，不像前端按 feature 拆成多个 section。
  key 用点分命名 `模块.场景`（如 `user.username_or_password_error`）。
- 新增消息时 `locales/zh.yaml` 和 `locales/en.yaml` 必须同步加同一 key，缺失 key
  会原样返回 key 字符串作为 fallback。
- 翻译值支持模板变量 `{{.Var}}`，通过 `i18n.T` / `ApiErrorI18n` 的 args map 传入。
- 语言来自请求 `Accept-Language` header，由 `GetLangFromContext` 解析；只支持
  `zh` 和 `en`，无法识别时回落到默认中文（`DefaultLang`）。不要在业务代码里
  自行解析语言串。
- 内部日志（`SysError` / `SysLog`）、调试输出、发给上游供应商的请求体不翻译。
- i18n 包不承载业务逻辑；JSON 序列化遵守根目录规则（调用 `pkg/jsonx` 包装函数）。
- 不要绕过 `ApiErrorI18n` / `i18n.T` 直接调 `i18n.Translate` 传硬编码语言。

## 初始化

`i18n.Init()` 在 `internal/app/bootstrap.go` 启动时加载 embed 的 yaml 并把 `common.TranslateMessage`
注入为 `i18n.T`。新增翻译文件需在 `Init` 中注册。

## 验证

- 改 `keys.go` 或翻译 yaml 后执行 `go build .` 确认编译通过。
- 新增 key 后人工核对 `locales/zh.yaml` 与 `locales/en.yaml` 已对齐（后端目前
  无类似前端 `i18n:sync` 的自动校验脚本）。
- 影响响应 message 的改动执行相关 controller 测试。
