package common

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
)

var relayGoPool *boundedRelayPool

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
const defaultRelayPoolQueueSize = 4096

type relayTask struct {
	ctx context.Context
	f   func()
}

type boundedRelayPool struct {
	tasks chan relayTask
	once  sync.Once
}

func init() {
	cap := int32(defaultRelayPoolCap)
	if v := os.Getenv("RELAY_POOL_CAP"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= math.MaxInt32 {
			cap = int32(n)
		}
	}
	queueSize := defaultRelayPoolQueueSize
	if v := os.Getenv("RELAY_POOL_QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			queueSize = n
		}
	}
	relayGoPool = newBoundedRelayPool(int(cap), queueSize)
}

func newBoundedRelayPool(workerCount int, queueSize int) *boundedRelayPool {
	if workerCount <= 0 {
		workerCount = defaultRelayPoolCap
	}
	if queueSize <= 0 {
		queueSize = defaultRelayPoolQueueSize
	}

	p := &boundedRelayPool{
		tasks: make(chan relayTask, queueSize),
	}
	p.once.Do(func() {
		for i := 0; i < workerCount; i++ {
			go p.worker()
		}
	})
	return p
}

func (p *boundedRelayPool) worker() {
	for task := range p.tasks {
		runRelayTask(task)
	}
}

func runRelayTask(task relayTask) {
	defer func() {
		if r := recover(); r != nil {
			if stopChan, ok := task.ctx.Value("stop_chan").(chan bool); ok {
				SafeSendBool(stopChan, true)
			}
			SysError(fmt.Sprintf("panic in gopool.RelayPool: %v\n%s", r, debug.Stack()))
		}
	}()
	task.f()
}

func (p *boundedRelayPool) CtxGo(ctx context.Context, f func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	p.tasks <- relayTask{ctx: ctx, f: f}
}

func RelayCtxGo(ctx context.Context, f func()) {
	relayGoPool.CtxGo(ctx, f)
}

func RelayGo(f func()) {
	RelayCtxGo(context.Background(), f)
}
