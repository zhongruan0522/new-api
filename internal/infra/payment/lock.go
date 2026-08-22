package payment

import "sync"

var orderLocks sync.Map
var createLock sync.Mutex

// LockOrder 尝试对给定订单号加锁
func LockOrder(tradeNo string) {
	lock, ok := orderLocks.Load(tradeNo)
	if !ok {
		createLock.Lock()
		defer createLock.Unlock()
		lock, ok = orderLocks.Load(tradeNo)
		if !ok {
			lock = new(sync.Mutex)
			orderLocks.Store(tradeNo, lock)
		}
	}
	lock.(*sync.Mutex).Lock()
}

// UnlockOrder 释放给定订单号的锁
func UnlockOrder(tradeNo string) {
	lock, ok := orderLocks.Load(tradeNo)
	if ok {
		lock.(*sync.Mutex).Unlock()
	}
}
