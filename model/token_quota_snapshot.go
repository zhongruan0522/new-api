package model

import "time"

type TokenQuotaSnapshot struct {
	QuotaType      int
	TotalGranted   int
	TotalUsed      int
	TotalAvailable int
	Unlimited      bool
}

func (token *Token) GetQuotaSnapshot() TokenQuotaSnapshot {
	return token.getQuotaSnapshot(time.Now().Unix())
}

func (token *Token) getQuotaSnapshot(now int64) TokenQuotaSnapshot {
	quotaType := token.QuotaType
	if quotaType == 0 && !token.UnlimitedQuota {
		quotaType = 1
	}

	if token.UnlimitedQuota || quotaType == 0 {
		return TokenQuotaSnapshot{
			QuotaType:      0,
			TotalGranted:   token.RemainQuota + token.UsedQuota,
			TotalUsed:      token.UsedQuota,
			TotalAvailable: token.RemainQuota,
			Unlimited:      true,
		}
	}

	switch quotaType {
	case 2:
		used := token.WindowUsedQuota
		if token.shouldResetWindow(now) {
			used = 0
		}
		if used < 0 {
			used = 0
		}
		available := token.WindowQuota - used
		if available < 0 {
			available = 0
		}
		return TokenQuotaSnapshot{
			QuotaType:      quotaType,
			TotalGranted:   token.WindowQuota,
			TotalUsed:      used,
			TotalAvailable: available,
		}
	case 3:
		windowUsed := token.WindowUsedQuota
		if token.shouldResetWindow(now) {
			windowUsed = 0
		}
		if windowUsed < 0 {
			windowUsed = 0
		}
		cycleUsed := token.CycleUsedQuota
		if token.shouldResetCycle(now) {
			cycleUsed = 0
		}
		if cycleUsed < 0 {
			cycleUsed = 0
		}

		windowAvailable := token.WindowQuota - windowUsed
		if windowAvailable < 0 {
			windowAvailable = 0
		}
		cycleAvailable := token.CycleQuota - cycleUsed
		if cycleAvailable < 0 {
			cycleAvailable = 0
		}

		if windowAvailable <= cycleAvailable {
			return TokenQuotaSnapshot{
				QuotaType:      quotaType,
				TotalGranted:   token.WindowQuota,
				TotalUsed:      windowUsed,
				TotalAvailable: windowAvailable,
			}
		}

		return TokenQuotaSnapshot{
			QuotaType:      quotaType,
			TotalGranted:   token.CycleQuota,
			TotalUsed:      cycleUsed,
			TotalAvailable: cycleAvailable,
		}
	default:
		totalGranted := token.RemainQuota + token.UsedQuota
		if totalGranted < 0 {
			totalGranted = 0
		}
		available := token.RemainQuota
		if available < 0 {
			available = 0
		}
		return TokenQuotaSnapshot{
			QuotaType:      1,
			TotalGranted:   totalGranted,
			TotalUsed:      token.UsedQuota,
			TotalAvailable: available,
		}
	}
}
