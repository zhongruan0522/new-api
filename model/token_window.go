package model

import (
	"time"
)

// GetCurrentWindow 计算当前窗口的起止时间。
// 窗口从 WindowStartHour 小时开始，每 WindowHours 小时一个窗口，按小时对齐。
func (token *Token) GetCurrentWindow() (windowStart, windowEnd int64) {
	return token.getCurrentWindow(time.Now().Unix())
}

func floorDiv(a, b int64) int64 {
	if b <= 0 {
		return 0
	}
	if a >= 0 {
		return a / b
	}
	// 向下取整，例如 -1/5 = -1
	return -((-a + b - 1) / b)
}

func (token *Token) getCurrentWindow(now int64) (windowStart, windowEnd int64) {
	hours := int64(token.WindowHours)
	if hours <= 0 {
		return 0, 0
	}

	// windowLen 是窗口长度（秒）
	windowLen := hours * 3600

	var anchor int64
	if hours <= 24 {
		// WindowHours <= 24：按天对齐锚点，每天 WindowStartHour 小时启动新窗口。
		// 若当前时间尚未到达今天的锚点，则使用昨天的锚点保持窗口连续性，
		// 防止跨午夜后 windowStart 回退导致 ShouldResetWindow 判断异常。
		dayStart := (now / 86400) * 86400
		anchor = dayStart + int64(token.WindowStartHour)*3600
		if now < anchor {
			anchor -= 86400
		}
	} else {
		// WindowHours > 24：使用固定锚点，保证多日窗口序列连续不中断。
		// 按天对齐会导致窗口在每天 WindowStartHour 时刻被错误截断，
		// 例如 window_start_hour=8, window_hours=30 时，第二天 08:00 会错误地
		// 开始新窗口，而实际窗口应持续到第二天 14:00 才结束。
		// 以 Unix 纪元当天的 WindowStartHour 作为全局锚点。
		anchor = int64(token.WindowStartHour) * 3600
	}

	elapsed := now - anchor
	windowIndex := floorDiv(elapsed, windowLen)
	windowStart = anchor + windowIndex*windowLen
	windowEnd = windowStart + windowLen

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
