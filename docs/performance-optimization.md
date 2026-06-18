# 性能优化报告

本文件记录了针对高并发场景下内存和 CPU 占用过高的系统性优化。每节对应一个独立的 commit，包含问题分析、修复方案和预期效果。

## 背景

项目在多次功能迭代后出现严重性能退化：20 并发请求时内存占用约 300MB（此前不足 100MB），高并发（>100）场景下内存持续增长。

经过对 `main.go`、`relay/`、`model/`、`service/`、`common/` 和 `Dockerfile` 的全链路分析，定位到以下根因类别：

1. 无限 goroutine 池导致并发时 goroutine 爆炸
2. 每请求大缓冲区分配且不复用
3. 上游响应体无大小限制
4. 后台任务无效唤醒
5. 连接池配置过小导致 TLS 连接频繁重建
6. Docker 镜像包含不必要的 ASan 运行时

---

## Commit 1: `64d3257c` — Docker 镜像从 debian 切换到 alpine，移除 libasan8

**问题**

`Dockerfile` 运行时阶段使用 `debian:bookworm-slim` 并安装了：
- `libasan8`：GCC AddressSanitizer 运行时库（约 100MB+）
- `wget`：容器内未使用

`CGO_ENABLED=0` 编译的 Go 二进制是完全静态链接的，使用纯 Go 实现的 `github.com/glebarez/sqlite` 驱动，不依赖 libc。`libasan8` 不仅不必要，还可能被动态链接器预加载，为每次内存分配添加 sanitizer bookkeeping 开销。

**修复**

将运行时基础镜像从 `debian:bookworm-slim`（~80MB）切换到 `alpine:3.21`（~3.5MB），仅安装 `ca-certificates` 和 `tzdata`。

**效果**

| 指标 | 修复前 | 修复后 |
|------|--------|--------|
| 运行时镜像大小 | ~120MB+ | ~30MB |
| 不必要运行时库 | libasan8 (100MB+) | 无 |

---

## Commit 2: `50292f16` — 限制 relay goroutine 池为 256 worker

**问题**

`common/gopool.go` 中的 relay pool 以 `math.MaxInt32` 作为 worker 上限：

```go
relayGoPool = gopool.NewPool("gopool.RelayPool", math.MaxInt32, gopool.NewConfig())
```

bytedance/gopkg 的 gopool 在 `ScaleThreshold=1` 且 `WorkerCount < cap` 时，每提交一个任务就创建新 worker goroutine。`cap=MaxInt32` 意味着 cap 检查永远为真，**每个任务都会创建一个新 goroutine**，pool 完全没有起到限制作用。

高并发时每个请求会提交多个异步任务（计费、日志、审计），导致 goroutine 无限增长：每个 goroutine 初始栈 8KB（会按需增长），加上任务本身持有的 buffer、DB 连接、JSON 结构等分配，20 并发时轻松创建数百个 goroutine。

**修复**

将默认 cap 设为 256（可通过 `RELAY_POOL_CAP` 环境变量配置），并添加 int32 溢出保护。

```go
const defaultRelayPoolCap = 256
cap := int32(defaultRelayPoolCap)
if v := os.Getenv("RELAY_POOL_CAP"); v != "" {
    if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= math.MaxInt32 {
        cap = int32(n)
    }
}
```

**效果**

- goroutine 数量有上限，不再随并发线性增长
- 内存占用可预测
- 256 对异步 bookkeeping 足够充裕

---

## Commit 3: `d823010a` — pprof 条件化注册

**问题**

`main.go` 中有 `_ "net/http/pprof"` 无条件空白导入。该导入在 `init()` 中向 `http.DefaultServeMux` 注册约 10 个 debug handler。虽然单个 handler 开销不大，但在生产构建中完全是死代码。

**修复**

移除 `main.go` 中的无条件导入，将 `_ "net/http/pprof"` 放入 `common/pprof.go` 包级别导入。pprof debug server 仅在 `ENABLE_PPROF=true` 时通过 `common.EnablePprofServer()` 启动。

同时清理了 `main.go` 中不再使用的 `log` 和 `gopool` 导入。

---

## Commit 4: `0f6d841d` — 流式扫描器缓冲区降至 8KB 并用 sync.Pool 复用

**问题**

`relay/helper/stream_scanner.go` 中每个流式请求分配 64KB 初始 scanner buffer：

```go
scanner.Buffer(make([]byte, 64<<10), getScannerBufferSize())
```

SSE data frame 通常只有几百字节，64KB 远超实际需要。100 并发流式请求仅缓冲区就预留 6.4MB+，且每个请求结束后 buffer 被 GC 回收再重新分配。

**修复**

1. 将 `InitialScannerBufferSize` 从 64KB 降至 8KB（仍覆盖绝大多数 SSE 行，scanner 按需增长）
2. 添加 `sync.Pool` 复用 buffer，避免重复分配

```go
var scannerBufferPool = sync.Pool{
    New: func() interface{} {
        b := make([]byte, InitialScannerBufferSize)
        return &b
    },
}
```

buffer 在 scanner goroutine 退出后才放回 pool，避免 data race。

**效果（128 并发流式 benchmark）**

| 指标 | 修复前 | 修复后 | 改善 |
|------|--------|--------|------|
| 峰值堆内存 | 10.63 MB/op | 3.70 MB/op | -65% |
| 总分配字节 | 11.64 MB/op | 4.31 MB/op | -63% |
| 单次操作耗时 | 21.1 ms | 16.7 ms | +21% |

---

## Commit 5: `07b56f0e` — 上游响应体读取限制 128MB

**问题**

所有非流式 relay channel handler 使用 `io.ReadAll(resp.Body)` 将整个上游响应加载到内存，无大小上限。恶意或异常的上游返回超大响应体时，在高并发下可以耗尽服务器内存。

涉及 provider：OpenAI、Claude、Gemini、Ollama、MiniMax、SiliconFlow、Zhipu、rerank handler。

**修复**

新增 `common.ReadResponseBody()`，基于 `io.LimitReader` 限制为 `constant.MaxResponseBodyMB`（默认 128MB，可通过 `MAX_RESPONSE_BODY_MB` 环境变量配置）。替换所有 relay 路径上的 `io.ReadAll(resp.Body)` 调用。

```go
func ReadResponseBody(r io.Reader) ([]byte, error) {
    maxBytes := int64(maxMB) << 20
    data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
    if int64(len(data)) > maxBytes {
        return nil, &MaxResponseBodyExceededError{LimitMB: maxMB}
    }
    return data, nil
}
```

---

## Commit 6: `e1963e06` — 减少 UpdateQuotaData 唤醒频率

**问题**

`model/usedata.go` 的 `UpdateQuotaData` goroutine 每秒 `time.Sleep(time.Second)` 唤醒，即使 `DataExportEnabled=false`（大多数部署的默认值）。这导致每秒一次无意义的 goroutine 唤醒和 cache-line 刷新。

**修复**

根据运行状态动态调整 sleep 间隔：
- `DataExportEnabled=false`：sleep 10s（减少 90% 唤醒）
- 启用但 flush 间隔较长（>5min）：sleep 最多 1min（仍能及时检测运行时开关变更）
- 启用且间隔短：保持 1s 粒度

---

## Commit 7: `eb45929b` — 扩大 Redis 和 HTTP 连接池默认值

**问题**

**Redis pool**：默认 PoolSize=10，在高并发 relay 请求（每个请求可能多次访问 Redis 做 token/channel 缓存）时成为瓶颈。缺少 MinIdleConns、超时和重试配置。

**HTTP transport**：`MaxIdleConnsPerHost=32`。AI 网关流量通常指向少数上游 host（OpenAI、Claude 等），per-host 限制过低导致并发高峰时频繁关闭/重建 TLS 连接（握手开销大）。

**修复**

Redis pool：
- PoolSize：10 → 50
- MinIdleConns：添加，默认 5
- DialTimeout=5s, ReadTimeout=3s, WriteTimeout=3s
- MaxRetries=3

HTTP transport：
- RelayMaxIdleConnsPerHost：32 → 100

所有参数均可通过环境变量覆盖。

---

## Commit 8: `2485cb98` — SiliconFlow JSON 调用规范修复

**问题**

`relay/channel/siliconflow/relay-siliconflow.go` 直接调用 `encoding/json` 的 `Marshal`/`Unmarshal`，违反 `common/AGENTS.md` 中"JSON 序列化必须走 `common/json.go` 包装函数"的规则。

**修复**

替换为 `common.Marshal` 和 `common.Unmarshal`。

---

## Commit 9: `977a9c97` — Code Review 修复

修复了 review 子代理发现的问题：

**P1-1: scanner buffer pool data race**

scanner buffer 原先在外层 handler 的 defer 中放回 pool。当 handler 因 timeout/disconnect 提前退出时，scanner goroutine 可能仍在 `scanner.Scan()` 中使用 buffer 的底层数组。后续流式请求可能从 pool 获取并修改同一个 buffer，导致 data race。

修复：将 `Put()` 移入 scanner goroutine 的 defer 链，确保只在 `scanner.Scan()` 确定退出后才回收。

**P1-2: MiniMax 响应体泄漏**

`handleTTSResponse` 和 `handleChatCompletionResponse` 中 `resp.Body.Close()` 在 `ReadResponseBody` 之后才 defer。如果读取失败（超大响应或读取错误），response body 不会被关闭，泄漏上游 HTTP 连接。

修复：将 `defer resp.Body.Close()` 移到读取操作之前。

**P2-1: pprofinit 简化**

移除了不必要的 `common/pprofinit` 间接包，直接在 `common/pprof.go` 使用 blank import。

**P2-2: RELAY_POOL_CAP 溢出保护**

添加 `math.MaxInt32` 上限检查，防止环境变量配置过大导致 int32 溢出。

---

## 新增环境变量汇总

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `RELAY_POOL_CAP` | 256 | relay 异步任务 goroutine 池上限 |
| `MAX_RESPONSE_BODY_MB` | 128 | 上游非流式响应体最大读取大小（MB） |
| `REDIS_POOL_SIZE` | 50 | Redis 连接池大小 |
| `REDIS_MIN_IDLE_CONNS` | 5 | Redis 最小空闲连接数 |
| `REDIS_DIAL_TIMEOUT` | 5 | Redis 拨号超时（秒） |
| `REDIS_READ_TIMEOUT` | 3 | Redis 读超时（秒） |
| `REDIS_WRITE_TIMEOUT` | 3 | Redis 写超时（秒） |
| `REDIS_MAX_RETRIES` | 3 | Redis 最大重试次数 |
| `RELAY_MAX_IDLE_CONNS_PER_HOST` | 100 | 每上游 host 最大空闲连接数 |

---

## 验证

所有改动通过了以下验证：

```bash
# 全量编译
go build -o /tmp/new-api-verify .

# 全量测试
go test ./common/... ./relay/... ./service/... ./controller/... ./model/...

# data race 检测
go test -race ./relay/helper/

# 并发 benchmark
go test -bench=BenchmarkStreamScannerHandlerConcurrent128 -benchmem ./relay/helper/
```

benchmark 结果（128 并发流式，200 frame/stream）：

```
BenchmarkStreamScannerHandlerConcurrent128-3   99   16889006 ns/op   3.697 peak_heap_mb/op   4310829 B/op   31537 allocs/op
```

峰值堆内存从 10.63 MB/op 降至 3.70 MB/op（-65%）。

---

## 后续优化方向

以下优化因风险较高或需要更大范围改动，未在本次执行：

1. **channel cache 增量更新**：当前每 `SyncFrequency` 秒全量重建 channel cache，可改为增量更新
2. **quota data 批量 UPSERT**：`SaveQuotaDataCache` 逐条 SELECT+UPDATE/INSERT，可用批量操作优化
3. **日志批量写入**：每条日志单独 goroutine + INSERT，可用 buffered channel + batch insert
4. **proxy client cache LRU 驱逐**：`proxyClients` map 无上限，可用 LRU 限制
5. **token encoder cache 缩减**：当前 maxTokenEncoderCacheSize=64，内存紧张时可降至 32
