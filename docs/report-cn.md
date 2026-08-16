# 安全审计报告：new-api

## 审计范围

对位于 /workspace/new-api 的 NookMux LLM 网关进行标准仓库级安全审计。审查了 Go 后端路由、中间件鉴权、中继/代理、支付回调、OAuth/passkey/2FA 流程、存储型媒体以及 Web 前端 DOM 注入点。

- 扫描模式：deep_repository
- 目标类型：git_worktree
- 目标 ID：target_sha256_0152d45ef1f01a40cb2ce0e67e04ee1608bc3b6f9b8b26baf8d01fdd1f7a5f72
- 版本：da0f1241a211c245890d82bec00ff33826ccd259
- 快照摘要：codex-security-snapshot/v1:sha256:779c49cb769398b4d6dfef8042e74a39b56262768c4e6fe859ec3dbc09720225
- 清单策略：repository
- 包含路径：.
- 排除路径：无
- 运行时或测试状态：未记录
- 审查产物：AGENTS.md, Dockerfile, README.md, VERSION, common/, constant/, controller/, dto/, go.mod, go.sum, i18n/, logger/, main.go, makefile, middleware/, model/, nookmux.service, oauth/, pkg/, relay/, router/, scripts/, service/, setting/, types/
- 扫描上下文：对 NookMux Go 仓库（多供应商 LLM API 网关/代理，含中继、计费、OAuth、passkey/2FA 鉴权、管理后台、存储型媒体）进行标准单次安全审计。Worker 标签 discovery-0001；子代理数=3。无继承的 SECURITY.md 策略；无用户上下文；无知识库。

局限与排除项：
- 中继渠道适配器注入点审查（包 C/D）部分委托给一名后来不可用的调查员；剩余审查由协调员顺序完成，覆盖 api_request.go、存储资产处理器、custom_voice 上传以及 Web DOM 注入点。Node Web 前端在源码信号层面（dangerouslySetInnerHTML/redirect/open）进行了审查，但未逐一审计每个组件文件。Vendored Go 模块源码仅对 CORS 库进行了查阅。
- 排除 web/node_modules/\*\*：vendored 依赖，非产品源码
- 排除 web/dist/\*\*：构建产物，已 embed
- 排除 logs/\*\*：运行时日志文件
- 排除 .serena/\*\*：IDE 缓存
- 排除 .git/\*\*：版本控制元数据
- 排除 docs/\*\*：文档，仅作为不可信分析输入审查

### 扫描摘要

| 字段 | 值 |
| --- | --- |
| 可报告 DSS 发现 | 2 |
| 报告实例 | 2 |
| 报告严重性分布 | 中危：1，低危：1 |
| 报告置信度分布 | 高：2 |
| 覆盖范围 | 部分 |
| 验证模式 | static_source_only |

规范产物：`scan-manifest.json`、`findings.json` 和 `coverage.json`。本报告是这些文件的确定性投影。

## 威胁模型

NookMux 是一个 Go (Gin) 实现的多供应商 LLM API 网关与计费平台。它使用管理员配置的上游渠道代理 OpenAI/Claude/Gemini 等模型请求，通过基于令牌和基于会话的鉴权配合角色分级（guest/common/admin/root）进行访问控制，管理用户额度和充值（Epay/Stripe/兑换码），支持 OAuth 登录（GitHub/LinuxDO）和 passkey/2FA，存储用户上传的媒体（图片/视频）并通过 HMAC 签名 URL 提供，以及提供管理后台（React SPA）。信任边界：未认证网络攻击者 ↔ 公开 API/中继/MCP-资产路由；已认证普通用户令牌/会话 ↔ 用户级 API 路由；管理员 ↔ 管理员级路由；root ↔ 系统配置/数据库迁移路由；支付平台 ↔ 匿名回调端点；上游 LLM 供应商 ↔ 中继响应解析。

### 资产

- 用户和管理员会话 cookie（30 天签名 cookie）
- API 访问令牌（Authorization 头 bearer / sk- 令牌）
- 用户额度/计费账本（充值、兑换、转账）
- 渠道供应商 API 密钥和凭据
- 用户拥有的存储型媒体（图片/视频）
- 系统配置项（OptionMap，RootAuth 受控）
- OAuth 供应商绑定和身份关联

### 信任边界

- 未认证网络 ↔ 公开 API 路由（GET /api/status、/api/setup、/api/user/login、/api/oauth/\*、webhook、/mcp/image|video 签名 URL）
- API 令牌 / 会话 cookie ↔ UserAuth 保护路由
- 管理员会话/令牌 ↔ AdminAuth 保护路由（渠道、用户、日志、令牌、存储媒体管理）
- root 会话/令牌 ↔ RootAuth 保护路由（系统配置、数据库迁移、动态倍率规则）
- 支付平台（Epay/Stripe）↔ 匿名 webhook 回调（POST /api/stripe/webhook、/api/user/epay/notify）
- 上游 LLM 供应商 ↔ 中继响应解析器和流式处理器
- 用户可控的通知 URL（webhook/bark/gotify）↔ 服务端出站 HTTP 请求

### 攻击者能力

- 可访问公开 API 和中继路由的未认证网络攻击者
- 持有有效 API 令牌或会话的已认证普通用户
- 持有管理员级会话/令牌的已认证管理员（非 root）
- 支付平台 webhook 发送方（Epay/Stripe），可发送签名或伪造的回调
- 恶意或被入侵的上游 LLM 供应商，返回构造的响应
- 与 Web 前端交互的跨域浏览器攻击者

### 安全目标

- 防止鉴权绕过、会话固定和 OAuth state CSRF
- 强制角色层级：common \< admin \< root 用于特权变更操作
- 防止 IDOR 和跨租户访问令牌、日志、工单、存储媒体、充值记录
- 保护用户额度/计费账本免受重复入账、重放或已付款未入账状态
- 防止中继/代理和文件处理中的 SSRF、开放重定向、路径穿越和注入
- 避免敏感凭据或配置向未认证或低权限用户暴露

### 假设

- 不存在继承的 SECURITY.md 策略；审计应用通用安全设计原则。
- 30 天会话 cookie 生命周期是有意为之；Secure 通过 COOKIE_SECURE 可选开启。
- 初始化（POST /api/setup）按设计为匿名，用于首次运行初始化；保护未初始化实例是部署方的责任。
- 渠道 base URL 和 API 密钥是管理员配置的可信输入；用户中继请求在其中选择，但不提供任意上游主机。
- SSRF 防护（ValidateURL + 拨号时复查）通过 FetchSetting 配置；中继上游调用是管理员配置的目标，非用户提供的 URL。

## 发现

| 发现 | 报告 | 严重性 | 置信度 | 详细说明 |
| --- | --- | --- | --- | --- |
| AdminResetPasskey 跳过角色层级检查，允许非 root 管理员删除 root 用户的 passkey 凭据 | [occ_f4343b16a101826a81994861](#finding-1) | 中危 | 高 | occ_f4343b16a101826a81994861：见下文 |
| 2FA 登录待定会话无时间限制，允许通过 30 天会话 cookie 进行长时间窗口的 2FA 暴力破解 | [occ_d6438fd51f4bfb01701c7339](#finding-2) | 低危 | 高 | occ_d6438fd51f4bfb01701c7339：见下文 |

### 置信度等级

| 标签 | 含义 |
| --- | --- |
| 高 | 直接证据支持该发现，无实质性未解决阻碍。 |
| 中 | 证据支持一个合理的问题，但仍有实质性的运行时或可达性证明待补充。 |
| 低 | 证据不完整，仅保留以供明确跟进。 |

<a id="finding-1"></a>

### [1] AdminResetPasskey 跳过角色层级检查，允许非 root 管理员删除 root 用户的 passkey 凭据

| 字段 | 值 |
| --- | --- |
| 严重性 | 中危 |
| 置信度 | 高 |
| 置信度依据 | 直接源码审查确认 AdminResetPasskey（controller/passkey.go:475-509）缺少任何角色比较，与 AdminDisable2FA（controller/twofa.go:451）和 ManageUser（controller/user.go:762）形成对比，后两者均强制 `myRole <= targetUser.Role && myRole != common.RoleRootUser`。 |
| 类别 | 访问控制 |
| CWE | CWE-862 |
| 受影响行 | controller/passkey.go:475, router/api_router.go:119, controller/twofa.go:451, controller/user.go:762 |

#### 摘要

DELETE /api/user/:id/reset_passkey 挂载在 AdminAuth 下（router/api_router.go:119），但 AdminResetPasskey（controller/passkey.go:475-509）从未将调用者角色与目标用户角色进行比较。其他所有管理员凭据变更处理器都强制 `myRole <= targetUser.Role && myRole != common.RoleRootUser` —— AdminDisable2FA（controller/twofa.go:451）和 ManageUser（controller/user.go:762）。AdminResetPasskey 加载目标用户、统计 passkey 数量，然后无条件调用 model.DeletePasskeyByUserID(user.Id)，没有任何角色比较。因此，非 root 管理员可以删除 root 用户的 passkey 凭据，移除 root 的强认证因子。

#### 根因

AdminResetPasskey 实现时缺少其他所有管理员凭据变更处理器（AdminDisable2FA、ManageUser）所强制的角色层级守卫：`myRole <= targetUser.Role && myRole != common.RoleRootUser`。

#### 验证

通过阅读 controller/passkey.go:475-509 确认：AdminResetPasskey 加载目标用户、统计 passkey 数量，然后调用 model.DeletePasskeyByUserID(user.Id)，无角色比较。与 controller/twofa.go:451（AdminDisable2FA）和 controller/user.go:762（ManageUser）对比，两者在继续执行前均强制 myRole \<= targetUser.Role && myRole != common.RoleRootUser。

#### 数据流

已认证管理员（非 root）发送 DELETE /api/user/:id/reset_passkey → AdminAuth 中间件放行（router/api_router.go:119）→ AdminResetPasskey 按 :id 加载目标用户 → 统计 passkey → 无角色检查地无条件调用 model.DeletePasskeyByUserID(user.Id) → root 用户的 passkey 凭据被删除。

#### 可达性

任何已认证管理员（非 root）都可以针对 root 用户的 :id 来删除其 passkey 凭据。AdminAuth 中间件仅检查调用者是否具有管理员或更高角色；不比较调用者与目标用户的角色。

#### 严重性

**中危** —— 权限提升：非 root 管理员可以删除 root 用户的 passkey 凭据，绕过角色层级。需要管理员级访问权限，限制了攻击者群体，但能够危及 root 账户的强认证因子。

额外的运行时或部署证据可能提升或降低此严重性。

#### 修复建议

在 AdminResetPasskey（controller/passkey.go:475-509）中添加角色层级检查，与 AdminDisable2FA（controller/twofa.go:451）和 ManageUser（controller/user.go:762）使用的模式一致：当 myRole \<= targetUser.Role 或目标用户为 root 用户（myRole != common.RoleRootUser）时拒绝请求。

预防性控制：
- topup_stripe.go:159 的 Stripe webhook.ConstructEvent
- CompleteEpayTopUp 和 Recharge 中的原子 WHERE id=? AND status=? CAS 防止重放导致的重复入账

<a id="finding-2"></a>

### [2] 2FA 登录待定会话无时间限制，允许通过 30 天会话 cookie 进行长时间窗口的 2FA 暴力破解

| 字段 | 值 |
| --- | --- |
| 严重性 | 低危 |
| 置信度 | 高 |
| 置信度依据 | 源码直接显示未存储或检查任何时间戳；与 SecureVerification 的 300 秒超时对比清晰。 |
| 类别 | 认证失败 |
| CWE | CWE-613 |
| 受影响行 | controller/user.go:87, controller/user.go:88, controller/twofa.go:353, main.go:148, controller/secure_verification.go:19 |

#### 摘要

登录（controller/user.go:84-93）在启用 2FA 时设置 session pending_username/pending_user_id。Verify2FALogin（controller/twofa.go:344-415）仅检查 pending_user_id 是否存在；从不记录或验证时间戳。会话 cookie 的 MaxAge 为 2592000（30 天，main.go:148）。2FA 锁定在每个 300 秒窗口后重置，因此持有有效第一阶段凭据的攻击者可以在较长时间内反复尝试 TOTP/备用码。安全验证的逐步认证路径已强制 SecureVerificationTimeout=300 秒；此登录流程缺少等价的限制。

#### 根因

2FA 登录待定会话实现时未存储时间戳或强制生命周期，不同于已强制 300 秒超时的 SecureVerification 逐步认证流程。

#### 验证

通过阅读 controller/user.go:84-93 确认：session.Set("pending_username", user.Username); session.Set("pending_user_id", user.Id) 未存储时间戳。以及 controller/twofa.go:351-362：pendingUserId := session.Get("pending_user_id"); if pendingUserId == nil {...}; userId, ok := pendingUserId.(int) 无过期检查。main.go:148 设置 MaxAge 2592000。controller/secure_verification.go:19 定义 SecureVerificationTimeout=300 并对逐步认证强制执行，表明登录流程遗漏了既有模式。

#### 数据流

POST /api/user/login → user.ValidateAndFill 成功 → 若 IsTwoFAEnabled：在 user.go:88 处 session.Set("pending_user_id", user.Id) 无时间戳。POST /api/user/login/2fa → Verify2FALogin 在 twofa.go:353 读取 session.Get("pending_user_id")，仅检查 nil/类型，验证 TOTP/备用码，调用 setupLogin。

#### 可达性

知道用户密码的未认证攻击者可以发起登录、接收 2FA 挑战，然后拥有最多 30 天时间提交验证码。2FA 锁定（5 次尝试 / 300 秒）在每次锁定后重置，允许恢复猜测。

#### 严重性

**低危** —— 暴力破解窗口延长，但 2FA 锁定和速率限制提供了部分缓解；攻击者必须已经知道密码。成功利用的可能性较低。

额外的运行时或部署证据可能提升或降低此严重性。

#### 修复建议

在设置待定会话时，同时存储时间戳（例如 session.Set("pending_user_id_set_at", now)）。在 Verify2FALogin 中，当 now - pending_user_id_set_at \>= 300 时拒绝（复用 SecureVerificationTimeout）。超时时清除待定字段。

测试：
- 单元测试：待定会话超过 300 秒时 Verify2FALogin 应拒绝

预防性控制：
- POST /api/user/login 和 /api/user/login/2fa 上的 CriticalRateLimit 中间件（router/api_router.go:56-57）
- 按账户强制的 2FA 锁定（model/twofa.go：MaxFailAttempts=5，LockoutDuration=300s）
- TOTP/备用码验证逻辑正确（model/twofa.go）

## 已审查面

| 面 | 风险领域 | 结果 | 备注 |
| --- | --- | --- | --- |
| 鉴权与授权中间件 | 未记录 | 未发现问题 | 无额外规范备注。 |
| 用户账户与 OAuth 流程 | 未记录 | 未发现问题 | 无额外规范备注。 |
| Passkey 与 2FA 生命周期 | 未记录 | 已报告 | 无额外规范备注。 |
| 令牌管理路由 | 未记录 | 未发现问题 | 无额外规范备注。 |
| 支付回调（Epay/Stripe）与计费账本 | 未记录 | 已报告 | 无额外规范备注。 |
| 工单系统归属检查 | 未记录 | 未发现问题 | 无额外规范备注。 |
| 存储媒体访问控制与签名 URL | 未记录 | 未发现问题 | 无额外规范备注。 |
| 系统配置与管理员设置 | 未记录 | 未发现问题 | 无额外规范备注。 |
| 中继代理头透传与请求构造 | 未记录 | 未发现问题 | 无额外规范备注。 |
| SSRF 防护与出站 HTTP 客户端 | 未记录 | 未发现问题 | 无额外规范备注。 |
| 自定义语音上传与上游调用 | 未记录 | 未发现问题 | 无额外规范备注。 |
| Web 前端 DOM 注入点（dangerouslySetInnerHTML、重定向） | 未记录 | 未发现问题 | 无额外规范备注。 |
| 中继渠道适配器（30+ 供应商实现） | 未记录 | 需跟进 | 无额外规范备注。 |
| Web React 组件树（990 个文件） | 未记录 | 需跟进 | 无额外规范备注。 |

## 待解决问题与跟进

- 第二名专项调查员不可用；协调员仅对最高风险的中继文件执行了顺序回退审查
  - 跟进提示：审查延期单元 deferred-1ed885dd224b6291 并关闭其所述的证明缺口。
- 在源码信号层面审查（grep dangerouslySetInnerHTML/redirect/open），但未逐一审计每个组件
  - 跟进提示：审查延期单元 deferred-747a60b572d97ccd 并关闭其所述的证明缺口。
