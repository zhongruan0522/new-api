# 代码风格和约定

## Go 后端

### 命名约定
- **包名**：小写，单词，如 `common`、`service`、`model`
- **函数/方法**：PascalCase（导出）或 camelCase（非导出）
- **常量**：PascalCase，如 `DefaultPort`
- **变量**：camelCase，如 `userId`、`channelName`
- **接口**：以 `er` 结尾，如 `Reader`、`Writer`

### 代码组织
- **分层架构**：Router → Controller → Service → Model
- **关注点分离**：
  - `router/`：HTTP 路由定义
  - `controller/`：请求处理和验证
  - `service/`：业务逻辑
  - `model/`：数据库访问和数据模型
  - `middleware/`：中间件（认证、日志、速率限制等）
  - `common/`：共享工具和常量

### JSON 处理
- **必须使用** `common/json.go` 中的包装函数：
  - `common.Marshal(v any) ([]byte, error)`
  - `common.Unmarshal(data []byte, v any) error`
  - `common.UnmarshalJsonStr(data string, v any) error`
  - `common.DecodeJson(reader io.Reader, v any) error`
  - `common.GetJsonType(data json.RawMessage) string`
- **禁止**直接导入或调用 `encoding/json`

### 数据库
- **ORM**：使用 GORM 抽象，优先使用 GORM 方法而不是原始 SQL
- **兼容性**：所有数据库代码必须同时兼容 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6
- **列引用**：
  - PostgreSQL：`"column"`
  - MySQL/SQLite：`` `column` ``
  - 使用 `commonGroupCol`、`commonKeyCol` 处理保留字
- **布尔值**：
  - PostgreSQL：`true`/`false`
  - MySQL/SQLite：`1`/`0`
  - 使用 `commonTrueVal`/`commonFalseVal`
- **数据库检测**：使用 `common.UsingPostgreSQL`、`common.UsingSQLite`、`common.UsingMySQL` 标志

### 错误处理
- 使用 `error` 接口，不要使用 panic（除非必要）
- 错误应尽早抛出，由上层统一处理
- 业务层不应自行捕获或吞掉错误

### 日志
- 使用 `common.SysLog()`、`common.SysError()`、`common.FatalLog()` 等日志函数
- 避免使用 `fmt.Println()` 或 `log.Println()`

### 注释
- 导出的函数/类型应有注释
- 复杂逻辑应有解释性注释
- 使用 `//` 单行注释

### 格式化
- 运行 `go fmt ./...` 保持一致的格式
- 使用 `go vet ./...` 检查常见问题

## React 前端

### 命名约定
- **组件**：PascalCase，如 `UserProfile`、`ChannelList`
- **文件**：与组件同名，如 `UserProfile.jsx`
- **变量/函数**：camelCase，如 `userId`、`handleSubmit`
- **常量**：UPPER_SNAKE_CASE，如 `MAX_RETRIES`、`API_TIMEOUT`
- **CSS 类**：kebab-case，如 `user-profile`、`channel-list`

### 代码组织
- **组件结构**：
  ```
  src/
    components/     — 可复用组件
    pages/          — 页面组件
    hooks/          — 自定义 hooks
    services/       — API 调用和业务逻辑
    utils/          — 工具函数
    styles/         — 全局样式
    i18n/           — 国际化配置
  ```

### 国际化
- **库**：i18next + react-i18next
- **语言**：仅中文（zh）
- **翻译文件**：`web/src/i18n/locales/zh.json`
- **格式**：扁平 JSON，键和值都是中文
- **使用**：`const { t } = useTranslation(); t('中文key')`
- **Semi UI 语言**：固定为 `zh_CN`

### 代码质量
- **格式化**：Prettier（`bun run lint:fix`）
- **Linting**：ESLint（`bun run eslint:fix`）
- **配置**：
  - Prettier：单引号（`singleQuote: true`）、JSX 单引号（`jsxSingleQuote: true`）
  - ESLint：extends `react-app` 和 `react-app/jest`

### React 最佳实践
- 使用函数组件和 Hooks
- 使用 `useCallback` 优化回调函数
- 使用 `useMemo` 优化计算
- 避免在 render 中创建新对象/函数
- 使用 `React.memo` 优化组件重渲染
- 正确处理依赖数组

### 样式
- **CSS 框架**：Tailwind CSS v3.4.19
- **UI 库**：Semi Design v2.72.2
- **方法**：优先使用 Semi Design 组件，然后 Tailwind，最后自定义 CSS

### 注释
- 复杂逻辑应有解释性注释
- 使用 `//` 单行注释
- 避免过度注释

## 通用约定

### 提交信息
- 格式：`<type>: <subject>`
- 类型：feat、fix、docs、style、refactor、test、chore
- 示例：`feat: add user authentication`、`fix: resolve database connection issue`

### 文件编码
- **编码**：UTF-8
- **行尾**：LF（Unix 风格）

### 环境变量
- 使用 `.env` 文件管理配置
- 参考 `.env.example` 了解所有可用配置项
- 敏感信息（密钥、密码）不要提交到版本控制

### 测试
- 编写测试来验证功能
- 测试应能证明问题的存在（先失败）
- 使用 `*_test.go` 命名测试文件
- 前端测试使用 Jest（如果配置）

### 性能
- 避免 N+1 查询问题
- 使用缓存（Redis 或内存缓存）
- 优化数据库查询
- 前端：避免不必要的重渲染，使用虚拟化长列表

### 安全
- 验证所有用户输入
- 使用参数化查询防止 SQL 注入
- 不要在代码中硬编码敏感信息
- 使用 HTTPS 在生产环境
- 正确处理 CORS 和 CSRF
