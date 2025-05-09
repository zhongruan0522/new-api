package helper

import (
	"bufio"
	"context"
	"github.com/bytedance/gopkg/util/gopool"
	"io"
	"net/http"
	"one-api/common"
	"one-api/constant"
	relaycommon "one-api/relay/common"
	"one-api/setting/operation_setting"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	InitialScannerBufferSize = 1 << 20  // 1MB (1*1024*1024)
	MaxScannerBufferSize     = 10 << 20 // 10MB (10*1024*1024)
	DefaultPingInterval      = 10 * time.Second
)

type DoRequestFunc func(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error)

// Optional SSE Ping keep-alive mechanism
//
// Used to solve the problem of the connection with the client timing out due to no data being sent when the upstream
// channel response time is long (e.g., thinking model).
// When enabled, it will send ping data packets to the client via SSE at the specified interval to maintain the connection.
func DoStreamRequestWithPinger(doRequest DoRequestFunc, c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	SetEventStreamHeaders(c)

	generalSettings := operation_setting.GetGeneralSetting()
	pingEnabled := generalSettings.PingIntervalEnabled
	pingInterval := time.Duration(generalSettings.PingIntervalSeconds) * time.Second

	pingerCtx, stopPinger := context.WithCancel(c.Request.Context())
	var pingerWg sync.WaitGroup
	var doRequestErr error
	var resp any

	if pingEnabled {
		pingerWg.Add(1)

		gopool.Go(func() {
			defer pingerWg.Done()

			if pingInterval <= 0 {
				pingInterval = DefaultPingInterval
			}

			ticker := time.NewTicker(pingInterval)
			defer ticker.Stop()
			var pingMutex sync.Mutex

			if common.DebugEnabled {
				println("SSE ping goroutine started.")
			}

			for {
				select {
				case <-ticker.C:
					pingMutex.Lock()
					err := PingData(c)
					pingMutex.Unlock()
					if err != nil {
						common.LogError(c, "SSE ping error: "+err.Error())
						return
					}
					if common.DebugEnabled {
						println("SSE ping data sent.")
					}
				case <-pingerCtx.Done():
					if common.DebugEnabled {
						println("SSE ping goroutine stopped.")
					}
					return
				}
			}
		})
	}

	resp, doRequestErr = doRequest(c, info, requestBody)

	stopPinger()
	if pingEnabled {
		pingerWg.Wait()
	}

	return resp, doRequestErr
}

func StreamScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(data string) bool) {

	if resp == nil || dataHandler == nil {
		return
	}

	defer resp.Body.Close()

	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	if strings.HasPrefix(info.UpstreamModelName, "o") {
		// twice timeout for thinking model
		streamingTimeout *= 2
	}

	var (
		stopChan   = make(chan bool, 2)
		scanner    = bufio.NewScanner(resp.Body)
		ticker     = time.NewTicker(streamingTimeout)
		writeMutex sync.Mutex // Mutex to protect concurrent writes
	)

	defer func() {
		ticker.Stop()
		close(stopChan)
	}()
	scanner.Buffer(make([]byte, InitialScannerBufferSize), MaxScannerBufferSize)
	scanner.Split(bufio.ScanLines)
	SetEventStreamHeaders(c)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = context.WithValue(ctx, "stop_chan", stopChan)

	common.RelayCtxGo(ctx, func() {
		for scanner.Scan() {
			ticker.Reset(streamingTimeout)
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
			data = strings.TrimLeft(data, " ")
			data = strings.TrimSuffix(data, "\r")
			if !strings.HasPrefix(data, "[DONE]") {
				info.SetFirstResponseTime()
				writeMutex.Lock() // Lock before writing
				success := dataHandler(data)
				writeMutex.Unlock() // Unlock after writing
				if !success {
					break
				}
			}
		}

		if err := scanner.Err(); err != nil {
			if err != io.EOF {
				common.LogError(c, "scanner error: "+err.Error())
			}
		}

		common.SafeSendBool(stopChan, true)
	})

	select {
	case <-ticker.C:
		// 超时处理逻辑
		common.LogError(c, "streaming timeout")
		common.SafeSendBool(stopChan, true)
	case <-stopChan:
		// 正常结束
		common.LogInfo(c, "streaming finished")
	}
}
