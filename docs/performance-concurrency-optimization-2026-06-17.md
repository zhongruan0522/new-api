# 高并发内存与前端资源优化记录

日期：2026-06-17

## 背景

本轮目标是降低 API 高并发场景下的内存占用和 CPU 压力。用户反馈的核心现象是：功能逐渐丰富后，20 个 API 并发请求已经接近 300MB，历史版本不到 100MB；同时需要注意前端 CSS/JS 等静态资源请求也会形成额外并发压力。

约束：

- 禁止直接解析 Docker。
- 避免占满磁盘，构建后清理临时二进制。
- 优先修根因，不用模拟数据或静默降级。

## Commit 清单

### 1. `2c7d212c perf: reduce relay streaming memory overhead`

问题层级：

- relay 流式请求路径存在多余 goroutine、较大的默认扫描缓冲和重复流式队列。
- OpenAI 流式 usage/token 统计曾积累完整流式片段后再统一处理，放大高并发下的内存驻留。
- HTTP client 连接池默认值偏大，闲置连接生命周期不够明确。

解决办法：

- 移除 relay API request 路径中重复的 SSE ping goroutine。
- 重写 `relay/helper/stream_scanner.go` 的扫描/转发逻辑，减少 goroutine、timer 和队列开销。
- 将默认 SSE frame buffer 从 64MB 下调为 8MB，并保留环境变量可调。
- OpenAI 流式 token accounting 改为增量处理，避免累积完整 `streamItems` 后再拼接大 JSON。
- 调整 HTTP transport 默认连接池：
  - `RELAY_MAX_IDLE_CONNS` 默认 200。
  - `RELAY_MAX_IDLE_CONNS_PER_HOST` 默认 32。
  - 新增 `RELAY_IDLE_CONN_TIMEOUT` 默认 90 秒。

验证：

- `go test ./...`
- `go build -ldflags "-X 'github.com/zhongruan0522/new-api/common.Version=$(git rev-parse HEAD)'" -o new-api`
- 构建后删除 `new-api` 临时二进制。

### 2. `96e54b94 perf: cap token cache and trim realtime buffers`

问题层级：

- tokenizer encoder cache 对动态/未知模型名可能持续增长。
- Realtime relay 路径保留了未使用的 per-session 队列。
- disk-backed request body 已落盘时，仍可能在 gin context 里保留一份 legacy byte slice，造成大请求体重复驻留。

解决办法：

- `service/tokenizer.go` 给 token encoder cache 加上上限。
- 规范化 OpenAI 动态模型名，减少同一类模型名造成的 cache key 膨胀。
- 未知模型不再缓存默认 encoder，避免无界增长。
- 移除 OpenAI realtime 未使用的 `sendChan` / `receiveChan`。
- Realtime token 事件日志从 Info 降为 Debug，降低高频日志开销。
- `common/gin.go` 在请求体使用 disk-backed storage 时，不再额外写入 legacy `KeyRequestBody` byte slice。

验证：

- `go test ./...`
- `go build -ldflags "-X 'github.com/zhongruan0522/new-api/common.Version=$(git rev-parse HEAD)'" -o new-api`
- 构建后删除 `new-api` 临时二进制。

### 3. `e605d71f perf: precompress static assets and add allocation benches`

问题层级：

- API 并发之外，浏览器访问还会并发请求 CSS/JS/font/image 等静态资源。
- 之前 web router 主要依赖运行时 gzip 中间件；在大量静态资源请求下，动态压缩会额外消耗 CPU。
- 同时生成 `.br` 和 `.gz` 会让 `web/dist` 以及最终 `go:embed` 包体重复增大。

解决办法：

- 新增 `web/scripts/compress-assets.mjs`，在 `bun run build` / `bun run build:check` 后只为 `dist/static` 下可压缩资源生成 `.br`。
- 不生成 `.gz`：
  - 支持 Brotli 的现代浏览器直接拿 `.br`。
  - 只支持 gzip 的客户端继续走原有 `gin-contrib/gzip` 动态压缩。
  - 不支持压缩的客户端继续拿原始资源。
- `router/web-router.go` 在动态 gzip 中间件之前优先命中 `.br` 静态资源，命中后设置：
  - `Content-Encoding: br`
  - `Vary: Accept-Encoding`
  - 原始资源对应的 `Content-Type`
- 为 `/static/*` 和 `/favicon.ico` 增加长缓存：
  - `Cache-Control: public, max-age=31536000, immutable`
  - SPA `index.html` 仍保持 `no-cache`。
- 新增 allocation benchmark，方便后续持续对比：
  - disk-backed request body
  - unknown token encoder model
  - stream scanner handler
  - 128 并发 stream scanner handler

验证：

- `go test ./router/... ./common/...`
- `bun run typecheck`
- `bun run build`
- `go test ./...`
- `go build -ldflags "-X 'github.com/zhongruan0522/new-api/common.Version=$(git rev-parse HEAD)'" -o new-api`
- 构建后删除 `new-api` 临时二进制。

关键证据：

- `bun run build` 后预压缩脚本输出：
  - `Compressed 191 assets: 16399.8 kB -> br 2816.7 kB`
- `web/dist/static` 校验：
  - `.br` 文件数：191
  - `.gz` 文件数：0
- benchmark 当前样本：
  - `BenchmarkGetRequestBodyDiskStorage`: 约 142167 ns/op，106266 B/op，44 allocs/op
  - `BenchmarkUnknownTokenEncoderModels`: 约 608.7 ns/op，40 B/op，1 alloc/op
  - `BenchmarkStreamScannerHandler`: 约 179729 ns/op，90734 B/op，243 allocs/op

### 4. 追加验证：128 并发流式核心路径

追加变更：

- `relay/helper/stream_scanner_test.go` 新增 `BenchmarkStreamScannerHandlerConcurrent128`。
- benchmark 在单进程内启动 128 个并发 stream scanner，每个 scanner 处理 200 个 SSE data frame。
- 每个并发请求都使用独立 gin context、HTTP response body、scanner goroutine 和 data handler。
- 第一个 frame 会同步阻塞，让 128 个 scanner 同时驻留，再释放并处理完整流。
- 不连接外网、不依赖 Docker、不写大文件。

验证命令：

- `go test ./relay/helper -run '^$' -bench 'BenchmarkStreamScannerHandler(Concurrent128)?$' -benchmem -count=1`

当前样本：

- `BenchmarkStreamScannerHandler`: 约 175989 ns/op，90734 B/op，243 allocs/op
- `BenchmarkStreamScannerHandlerConcurrent128`: 约 17.43 ms/op，11.64 MB/op，31395 allocs/op
- `BenchmarkStreamScannerHandlerConcurrent128` 额外指标：
  - `peak_heap_mb/op`: 约 10.61MB
  - `retained_heap_mb/op`: 约 0.0026MB

边界说明：

- 该 benchmark 证明的是 relay 流式扫描/转发核心路径在 128 并发下的内存形态。
- 它不等价于完整 HTTP + 鉴权 + 数据库 + 真实上游的端到端压测。
- 完整目标仍建议用真实运行服务采集 RSS、heap profile、goroutine profile 和 CPU profile。

## 总体思路

本轮没有从单点“表面省一点内存”下手，而是按高并发下的长期驻留来源拆分：

1. relay 流式路径：减少 goroutine、timer、队列、大 buffer 和完整流式内容累积。
2. cache 和请求体路径：限制可增长 map，避免动态模型名造成无界 cache；避免大请求体内存/磁盘双份保存。
3. 前端静态资源：构建时预压缩，运行时优先返回 Brotli 文件，减少 CSS/JS 并发请求的动态压缩 CPU；长缓存减少重复请求。
4. 验证路径：保留 benchmark，后续可以用同一命令对比优化前后 allocation 趋势。

## 后续建议

- 用真实上游或压测替身跑 `>100` API 并发，并采集 RSS、heap profile、goroutine profile、CPU profile。
- 对照 benchmark 持续观察 scanner、tokenizer、request body 三条热点路径。
- 若前端资源仍需继续压缩，下一步优先做按页面拆分重型依赖，例如 chart/highlight 相关 chunk，而不是再增加压缩格式。
