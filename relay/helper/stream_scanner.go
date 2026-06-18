package helper

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/logger"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

const (
	// InitialScannerBufferSize is the starting buffer size for each SSE stream
	// scanner. The previous value (64KB) was far larger than typical SSE data
	// frames (a few hundred bytes). At 100+ concurrent streams this alone
	// reserves 6.4MB+ of heap that is never used. 8KB comfortably covers most
	// SSE lines while keeping per-request footprint small. The scanner still
	// grows on demand up to the configured maximum.
	InitialScannerBufferSize    = 8 << 10 // 8KB
	DefaultMaxScannerBufferSize = 8 << 20 // 8MB default SSE buffer size
	DefaultPingInterval         = 10 * time.Second
)

// scannerBufferPool reuses the initial scanner buffer across concurrent stream
// requests. Without this pool every streaming relay allocates a fresh
// InitialScannerBufferSize buffer that lives until the scanner is garbage
// collected. At 100 concurrent streams with a 64KB buffer that is ~6.4MB of
// transient allocations. The pool returns already-grown buffers to subsequent
// callers so the scanner rarely needs to reallocate.
//
// Buffers that grew beyond initialBufferReclaimCap are still returned: the
// scanner copies the buffer when it grows, so reusing a larger one is safe and
// avoids re-growing on the next request.
var scannerBufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, InitialScannerBufferSize)
		return &b
	},
}

func getScannerBufferSize() int {
	if constant.StreamScannerMaxBufferMB > 0 {
		return constant.StreamScannerMaxBufferMB << 20
	}
	return DefaultMaxScannerBufferSize
}

func StreamScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(data string) bool) {

	if resp == nil || dataHandler == nil {
		return
	}

	// 确保响应体总是被关闭
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	if streamingTimeout <= 0 {
		streamingTimeout = time.Duration(common.RelayTimeout) * time.Second
	}
	var timeout <-chan time.Time
	var timeoutTimer *time.Timer
	if streamingTimeout > 0 {
		timeoutTimer = time.NewTimer(streamingTimeout)
		defer timeoutTimer.Stop()
		timeout = timeoutTimer.C
	}

	scanner := bufio.NewScanner(resp.Body)

	generalSettings := operation_setting.GetGeneralSetting()
	pingEnabled := generalSettings.PingIntervalEnabled && !info.DisablePing
	pingInterval := time.Duration(generalSettings.PingIntervalSeconds) * time.Second
	if pingInterval <= 0 {
		pingInterval = DefaultPingInterval
	}

	var ping <-chan time.Time
	var pingTicker *time.Ticker
	if pingEnabled {
		pingTicker = time.NewTicker(pingInterval)
		defer pingTicker.Stop()
		ping = pingTicker.C
	}

	if common.DebugEnabled {
		// print timeout and ping interval for debugging
		println("relay timeout seconds:", common.RelayTimeout)
		println("relay max idle conns:", common.RelayMaxIdleConns)
		println("relay max idle conns per host:", common.RelayMaxIdleConnsPerHost)
		println("streaming timeout seconds:", int64(streamingTimeout.Seconds()))
		println("ping interval seconds:", int64(pingInterval.Seconds()))
	}

	bufPtr := scannerBufferPool.Get().(*[]byte)
	scanner.Buffer(*bufPtr, getScannerBufferSize())
	scanner.Split(bufio.ScanLines)
	SetEventStreamHeaders(c)

	// Return the buffer to the pool when streaming finishes. We must use a
	// separate variable because scanner.Buffer may internally reference the
	// slice and grow it via copying; the original slice we put into the pool
	// is no longer referenced by the scanner after the last scan completes.
	defer scannerBufferPool.Put(bufPtr)

	stop := make(chan struct{})
	defer close(stop)
	dataChan := make(chan string, 16)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			data := scanner.Text()
			if common.DebugEnabled {
				println(data)
			}

			if len(data) < 6 {
				continue
			}
			if data[:5] != "data:" && data[:6] != "[DONE]" {
				continue
			}
			data = data[5:]
			data = strings.TrimSpace(data)
			if data == "" {
				continue
			}
			if !strings.HasPrefix(data, "[DONE]") {
				info.SetFirstResponseTime()
				info.ReceivedResponseCount++
				select {
				case dataChan <- data:
				case <-stop:
					return
				case <-c.Request.Context().Done():
					return
				}
			} else {
				// done, 处理完成标志，直接退出停止读取剩余数据防止出错
				if common.DebugEnabled {
					println("received [DONE], stopping scanner")
				}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				logger.LogError(c, "scanner error: "+err.Error())
			}
		}
	}()

	for {
		select {
		case data := <-dataChan:
			resetTimer(timeoutTimer, streamingTimeout)
			if !dataHandler(data) {
				return
			}
		case <-done:
			for {
				select {
				case data := <-dataChan:
					resetTimer(timeoutTimer, streamingTimeout)
					if !dataHandler(data) {
						return
					}
				default:
					logger.LogDebug(c, "streaming finished")
					return
				}
			}
		case <-ping:
			if err := PingData(c); err != nil {
				logger.LogError(c, "ping data error: "+err.Error())
				return
			}
			if common.DebugEnabled {
				println("ping data sent")
			}
		case <-timeout:
			logger.LogError(c, "streaming timeout")
			return
		case <-c.Request.Context().Done():
			logger.LogDebug(c, "client disconnected")
			return
		}
	}
}

func resetTimer(timer *time.Timer, timeout time.Duration) {
	if timer == nil || timeout <= 0 {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timeout)
}
