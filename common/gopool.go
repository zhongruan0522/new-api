package common

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/bytedance/gopkg/util/gopool"
)

var relayGoPool gopool.Pool

// defaultRelayPoolCap limits the number of background goroutines spawned by the
// relay pool for async work (billing, logging, audit, etc.).
//
// The previous value math.MaxInt32 effectively removed the cap, so under high
// concurrency every submitted task spawned a new worker goroutine. Each worker
// starts with an 8KB stack that grows, and the tasks they run frequently
// allocate buffers, causing RSS to balloon (observed >300MB at 20 concurrent
// requests). A bounded cap keeps memory predictable while still allowing enough
// parallelism for async bookkeeping.
const defaultRelayPoolCap = 256

func init() {
	cap := int32(defaultRelayPoolCap)
	if v := os.Getenv("RELAY_POOL_CAP"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= math.MaxInt32 {
			cap = int32(n)
		}
	}
	relayGoPool = gopool.NewPool("gopool.RelayPool", cap, gopool.NewConfig())
	relayGoPool.SetPanicHandler(func(ctx context.Context, i interface{}) {
		if stopChan, ok := ctx.Value("stop_chan").(chan bool); ok {
			SafeSendBool(stopChan, true)
		}
		SysError(fmt.Sprintf("panic in gopool.RelayPool: %v", i))
	})
}

func RelayCtxGo(ctx context.Context, f func()) {
	relayGoPool.CtxGo(ctx, f)
}

func RelayGo(f func()) {
	RelayCtxGo(context.Background(), f)
}
