package constant

type MultiKeyMode string

const (
	MultiKeyModeRandom  MultiKeyMode = "random"  // 随机
	MultiKeyModePolling MultiKeyMode = "polling" // 轮询
)

// Valid 报告模式是否为受支持的多密钥取用策略，渠道写入边界必须拒绝非法值，
// 避免运行时静默落入 GetNextEnabledKey 的 default 分支。
func (m MultiKeyMode) Valid() bool {
	return m == MultiKeyModeRandom || m == MultiKeyModePolling
}
