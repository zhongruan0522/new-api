# 新版本 UI 方案

> 本文档为前端双主题热切换方案的技术设计，请勿删除。

## 0. 施工信息

| 项目 | 说明 |
|------|------|
| 施工分支 | `test`（从 `dev` 分支创建） |
| 参考项目位置 | `参考项目/new-api/`（上游 QuantumNous/new-api，main 分支） |
| 新版 UI 源码来源 | `参考项目/new-api/web/default/` |

所有实施工作必须在 `test` 分支上进行。参考项目仅作为代码参考和复制来源，不直接修改。

## 1. 背景

本项目 Fork 自上游 QuantumNous/new-api，当前前端基于 Semi Design + React 18 + react-router-dom v6 + Vite 构建（下称 **classic 主题**）。

上游在 `v1.0.0-alpha.1` 之后引入了全新前端 `web/default/`（下称 **default 主题**），技术栈为：

| 维度 | classic (当前) | default (新版) |
|------|---------------|----------------|
| 框架 | React 18, JSX | React 19, TSX |
| UI 库 | Semi Design | shadcn/Base UI + Tailwind 4 |
| 路由 | react-router-dom v6 | @tanstack/react-router (文件路由) |
| 构建 | Vite | Rsbuild |
| 状态 | Context API | Zustand |
| 数据获取 | axios 直接调用 | @tanstack/react-query |
| 表格 | Semi Table | @tanstack/react-table |
| 图表 | VChart (Semi 主题) | VChart v2 |
| 语言 | JavaScript | TypeScript |

## 2. 核心原则

### 2.1 后端业务 API 零修改

新旧两套前端调用的是完全相同的后端 API。换 UI 不等于换接口。如果这个功能调用的接口不存在，那么就要考虑下这个功能是否存在。

### 2.2 唯一允许的后端改动：静态文件服务层 + 主题标识

后端唯一需要改动的是"serve 哪个 HTML/JS/CSS"以及"告诉前端当前是哪个主题"

### 2.3 前端适配方向

如果 default 前端的某个 API 调用路径或字段与本项目后端不一致，修改前端代码去对齐后端，不反过来改后端。

### 2.4 classic 主题维稳

classic 主题进入维稳模式：只修 bug，不加新功能。新功能只在 default 主题中开发。

## 3. 总体架构

```
项目根目录/
├── main.go
├── web/
│   ├── classic/               # 现有前端（维稳，只修 bug）
│   │   ├── src/
│   │   ├── package.json
│   │   └── dist/
│   └── default/               # 新版前端（从参考项目引入）
│       ├── src/
│       ├── package.json
│       └── dist/
├── common/                    # 新增 ThemeAwareFS + GetTheme/SetTheme
├── setting/system_setting/    # 新增主题配置持久化
└── router/                    # web-router 适配双主题
```

## 4. 实施步骤

每一步都是一个独立的、可验证的提交。

### Step 1: 迁移现有前端到 `web/classic/`

目标：纯目录重组，零代码改动，确保编译通过、构建产物正常、服务能跑。

操作：
- 将 `web/` 下的前端源码、配置文件、public 目录等移入 `web/classic/`
- 调整 `main.go` 的 embed 指令路径（从 `web/dist` 改为 `web/classic/dist`）
- 调整 `router/web-router.go` 中的 embed 路径引用
- 运行 `cd web/classic && bun install && bun run build`
- 运行 `go build` 确认编译通过
- 启动服务确认前端正常加载

验证通过后提交。

### Step 2: 引入新版 UI 到 `web/default/`

目标：把参考项目的 `web/default/` 复制进来，确保它自身能独立编译出产物。

操作：
- 从参考项目复制 `web/default/` 目录到本项目
- 运行 `cd web/default && bun install && bun run build`
- 确认构建产物生成在 `web/default/dist/`

此时后端还没有 serve 这套前端，只是确保它能编译。验证通过后提交。

### Step 3: 实现主题切换模块

目标：后端支持双主题 embed 和运行时热切换，管理员可在新旧 UI 之间切换。

操作：
- `common/constants.go` 新增主题原子变量（GetTheme/SetTheme）
- `common/embed-file-system.go` 新增 ThemeAwareFS，根据当前主题 serve 对应的静态文件
- `setting/system_setting/` 下新增主题配置持久化模块
- `router/web-router.go` 改为接收双主题资源，使用 ThemeAwareFS
- `router/main.go` 适配新的函数签名
- `main.go` embed 两套 dist，传入双主题资源
- `controller/misc.go` 的 `/api/status` 响应中追加 `theme` 字段
- 默认值设为 `classic`，确保升级后用户无感

验证：
- `go build` 编译通过
- 启动服务，默认加载 classic 前端
- 通过配置切换到 default，确认加载 default 前端
- 切回 classic，确认正常

验证通过后提交。

### Step 4: default 前端 API 适配（最小化可运行）

目标：让 default 前端能在本项目后端上最小化运行起来。

操作：
- 对比 default 前端的 API 调用与本项目后端实际提供的接口
- 上游有但本项目没有的功能/接口：在前端侧砍掉对应的页面入口或调用
- 请求路径或字段名不一致的：在前端侧修改对齐后端
- 不改后端，不加接口

验证：
- 切换到 default 主题
- 核心页面能正常加载和操作（登录、Dashboard、渠道管理、令牌管理、日志、充值、模型定价）
- 不存在的功能入口已隐藏，不会报 404 或白屏

验证通过后提交。

### Step 5: 逐步补齐本项目独有页面

目标：将 classic 中有但 default 中没有的页面，在 default 中逐步实现。

需要补齐的页面可能但不限于以下内容：

| 页面 | 路由 | 优先级 | 说明 |
|------|------|--------|------|
| DynamicRatio | /dynamic-ratio | P1 | 动态倍率管理 |
| KeyQuery | /key-query | P2 | API Key 查询 |
| MultimodalFiles | /multimodal-files | P2 | 多模态文件管理 |
| Ticket | /ticket | P3 | 工单系统 |
| OrderQuery | /order-query | P3 | 订单查询 |

每个页面独立提交，逐步推进。

### Step 6: 确保云端工作流能够正常打包

目标: 能够正常构建Docker镜像，确保Dockerfile、.github/workflows/docker-image.yml均能正常的构建Docker镜像

## 5. 切换机制

管理员可以在系统设置中的运营设置选择使用新版本前端还是旧版本前端，由于当前为test分支，主要是测试，所以默认请设置为新分支

## 6. 参考提交

以下为上游引入新 UI 的关键提交（均来自 `参考项目/new-api` 仓库），实施时可参考其中的实现方式：

| Commit | 说明 | 规模 |
|--------|------|------|
| `a42b39760` | feat: launch v1.0 — next-generation frontend built from the ground up (#4265) | 1290 files, +158,786 lines |
| `8b2b03d27` | feat(web/default): unified UI overhaul — Base UI migration, theme presets (#4633) | 317 files, +19,871 / -7,008 |
| `a7d019e3a` | feat(default): redesign dashboard overview | dashboard 重构 |

这些提交中的后端改动仅限于静态文件服务层（共 14 个文件，+1,449 行），不涉及任何业务 API。

可参考的后端文件：
- `参考项目/new-api/common/constants.go` — GetTheme/SetTheme 实现
- `参考项目/new-api/common/embed-file-system.go` — ThemeAwareFS 实现
- `参考项目/new-api/setting/system_setting/theme.go` — 主题配置持久化
- `参考项目/new-api/router/web-router.go` — 双主题静态文件服务
- `参考项目/new-api/router/main.go` — ThemeAssets 参数传递
- `参考项目/new-api/main.go` — embed 指令
- `参考项目/new-api/controller/misc.go` — /api/status 返回 theme 字段

## 7. 风险与注意事项

1. **二进制体积**：embed 两套前端会增加约 15-20MB，gzip 压缩可缓解
2. **前端 API 差异**：default 前端可能调用本项目后端不存在的接口，这些在前端侧屏蔽，不改后端
3. **i18n**：default 使用 i18next 多语言（en/zh/ja/fr/ru/vi），classic 仅 zh，两者独立互不影响
4. **构建时间**：CI 需要构建两套前端，建议并行执行
5. **升级路径**：用户从旧版本升级后默认仍为 classic，不会突然变 UI
6. **禁止使用模拟数据**：不得出现使用模拟数据的情况