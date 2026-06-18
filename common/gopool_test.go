package common

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBoundedRelayPoolAppliesBackpressure(t *testing.T) {
	p := newBoundedRelayPool(1, 1)

	release := make(chan struct{})
	p.CtxGo(nil, func() {
		<-release
	})
	p.CtxGo(nil, func() {})

	submitted := make(chan struct{})
	go func() {
		p.CtxGo(nil, func() {})
		close(submitted)
	}()

	select {
	case <-submitted:
		t.Fatal("third task submitted while worker and queue were full")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)
	<-submitted
}

func TestBoundedRelayPoolRunsAllSubmittedTasks(t *testing.T) {
	const taskCount = 32
	p := newBoundedRelayPool(4, taskCount)

	var ran atomic.Int32
	var wg sync.WaitGroup
	wg.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		p.CtxGo(nil, func() {
			ran.Add(1)
			wg.Done()
		})
	}
	wg.Wait()

	if got := ran.Load(); got != taskCount {
		t.Fatalf("ran %d tasks, want %d", got, taskCount)
	}
}
