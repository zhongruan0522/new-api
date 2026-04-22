package model

import (
	"time"
)

// GetCurrentWindow 计算当前窗口的起止时间。
// 窗口从 WindowStartHour 小时开始，每 WindowHours 小时一个窗口，按小时对齐。
func (token *Token) GetCurrentWindow() (windowStart, windowEnd int64) {
	return token.getCurrentWindow(time.Now().Unix())
}

func (token *Token) getCurrentWindow(now int64) (windowStart, windowEnd int64) {
	hours := int64(token.WindowHours)
	if hours <= 0 {
		return 0, 0
	}

	// 将 now 转换为 UTC 时间，然后取当天 0 点
	t := time.Unix(now, 0).UTC()
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()

	// 从当天 0 点偏移到 WindowStartHour
	alignmentOffset := int64(token.WindowStartHour) * 3600
	epochStart := startOfDay + alignmentOffset

	// 如果 epochStart 在 now 之后，说明今天第一个窗口还没开始，回退到昨天
	if epochStart > now {
		epochStart -= 24 * 3600
	}

	// 找到 <= now 的最后一个窗口起始时间
	elapsed := now - epochStart
	windowIndex := elapsed / (hours * 3600)
	windowStart = epochStart + windowIndex*hours*3600
	windowEnd = windowStart + hours*3600

	return windowStart, windowEnd
}

// GetCurrentCycle 计算当前周期的起止时间。
// 周期从 CycleStartTime 开始，每 CycleDays 天一个周期。
func (token *Token) GetCurrentCycle() (cycleStart, cycleEnd int64) {
	return token.getCurrentCycle(time.Now().Unix())
}

func (token *Token) getCurrentCycle(now int64) (cycleStart, cycleEnd int64) {
	days := int64(token.CycleDays)
	if days <= 0 {
		return 0, 0
	}

	// 如果没有周期起始时间，使用当前时间
	cycleStart = token.CycleStartTime
	if cycleStart <= 0 {
		cycleStart = now
	}

	cycleLength := days * 24 * 3600
	// 找到 <= now 的最后一个周期起始时间
	elapsed := now - cycleStart
	if elapsed < 0 {
		return cycleStart, cycleStart + cycleLength
	}
	cycleIndex := elapsed / cycleLength
	cycleStart = cycleStart + cycleIndex*cycleLength
	cycleEnd = cycleStart + cycleLength

	return cycleStart, cycleEnd
}

// ShouldResetWindow 判断是否需要重置窗口额度
func (token *Token) ShouldResetWindow() bool {
	return token.shouldResetWindow(time.Now().Unix())
}

func (token *Token) shouldResetWindow(now int64) bool {
	if token.WindowStartTime <= 0 {
		return true
	}
	windowStart, _ := token.getCurrentWindow(now)
	return token.WindowStartTime < windowStart
}

// ShouldResetCycle 判断是否需要重置周期额度
func (token *Token) ShouldResetCycle() bool {
	return token.shouldResetCycle(time.Now().Unix())
}

func (token *Token) shouldResetCycle(now int64) bool {
	if token.CycleStartTime <= 0 {
		return true
	}
	cycleStart, _ := token.getCurrentCycle(now)
	return token.CycleStartTime < cycleStart
}
