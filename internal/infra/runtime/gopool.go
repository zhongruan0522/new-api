package runtime

import (
	"context"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
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
const defaultRelayPoolIdleTimeout = 30 * time.Second

type relayTask struct {
	ctx context.Context
	f   func()
}

type boundedRelayPool struct {
	tasks       chan relayTask
	maxWorkers  int
	idleTimeout time.Duration
	mu          sync.Mutex
	workers     int
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
	return newBoundedRelayPoolWithIdleTimeout(workerCount, queueSize, defaultRelayPoolIdleTimeout)
}

func newBoundedRelayPoolWithIdleTimeout(workerCount int, queueSize int, idleTimeout time.Duration) *boundedRelayPool {
	if workerCount <= 0 {
		workerCount = defaultRelayPoolCap
	}
	if queueSize <= 0 {
		queueSize = defaultRelayPoolQueueSize
	}
	if idleTimeout <= 0 {
		idleTimeout = defaultRelayPoolIdleTimeout
	}

	return &boundedRelayPool{
		tasks:       make(chan relayTask, queueSize),
		maxWorkers:  workerCount,
		idleTimeout: idleTimeout,
	}
}

func (p *boundedRelayPool) worker() {
	idleTimer := time.NewTimer(p.idleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case task := <-p.tasks:
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			runRelayTask(task)
			idleTimer.Reset(p.idleTimeout)
		case <-idleTimer.C:
			if len(p.tasks) > 0 {
				idleTimer.Reset(p.idleTimeout)
				continue
			}
			p.mu.Lock()
			p.workers--
			p.mu.Unlock()
			return
		}
	}
}

func (p *boundedRelayPool) startWorkerIfNeeded() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.workers >= p.maxWorkers {
		return
	}
	if p.workers > 0 && len(p.tasks) < p.workers {
		return
	}
	p.workers++
	go p.worker()
}

func (p *boundedRelayPool) activeWorkers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workers
}

func runRelayTask(task relayTask) {
	defer func() {
		if r := recover(); r != nil {
			if stopChan, ok := task.ctx.Value("stop_chan").(chan bool); ok {
				SafeSendBool(stopChan, true)
			}
			common.SysError(fmt.Sprintf("panic in gopool.RelayPool: %v\n%s", r, debug.Stack()))
		}
	}()
	task.f()
}

func (p *boundedRelayPool) CtxGo(ctx context.Context, f func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	p.startWorkerIfNeeded()
	p.tasks <- relayTask{ctx: ctx, f: f}
	p.startWorkerIfNeeded()
}

func RelayCtxGo(ctx context.Context, f func()) {
	relayGoPool.CtxGo(ctx, f)
}

func RelayGo(f func()) {
	RelayCtxGo(context.Background(), f)
}
