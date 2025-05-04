package setting

import (
	"encoding/json"
	"fmt"
	"one-api/common"
	"sync"
)

var ModelRequestRateLimitEnabled = false
var ModelRequestRateLimitDurationMinutes = 1
var ModelRequestRateLimitCount = 0
var ModelRequestRateLimitSuccessCount = 1000

// ModelRequestRateLimitGroupKey 定义了模型请求按组速率限制的配置键
const ModelRequestRateLimitGroupKey = "ModelRequestRateLimitGroup"

// ModelRequestRateLimitGroupConfig 存储按用户组解析后的速率限制配置
// map[groupName][2]int{totalCount, successCount}
var ModelRequestRateLimitGroupConfig map[string][2]int
var ModelRequestRateLimitGroupMutex sync.RWMutex

// UpdateModelRequestRateLimitGroupConfig 解析、校验并更新内存中的用户组速率限制配置
func UpdateModelRequestRateLimitGroupConfig(jsonStr string) error {
	ModelRequestRateLimitGroupMutex.Lock()
	defer ModelRequestRateLimitGroupMutex.Unlock()

	var newConfig map[string][2]int
	if jsonStr == "" || jsonStr == "{}" {
		// 如果配置为空或空JSON对象，则清空内存配置
		ModelRequestRateLimitGroupConfig = make(map[string][2]int)
		common.SysLog("Model request rate limit group config cleared")
		return nil
	}

	err := json.Unmarshal([]byte(jsonStr), &newConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ModelRequestRateLimitGroup config: %w", err)
	}

	// 校验配置值
	for group, limits := range newConfig {
		if len(limits) != 2 {
			return fmt.Errorf("invalid config for group '%s': limits array length must be 2", group)
		}
		if limits[1] <= 0 { // successCount must be greater than 0
			return fmt.Errorf("invalid config for group '%s': successCount (limits[1]) must be greater than 0", group)
		}
		if limits[0] < 0 { // totalCount can be 0 (no limit) or positive
			return fmt.Errorf("invalid config for group '%s': totalCount (limits[0]) cannot be negative", group)
		}
		if limits[0] > 0 && limits[0] < limits[1] { // If totalCount is set, it must be >= successCount
			return fmt.Errorf("invalid config for group '%s': totalCount (limits[0]) must be greater than or equal to successCount (limits[1]) when totalCount > 0", group)
		}
	}

	ModelRequestRateLimitGroupConfig = newConfig
	common.SysLog("Model request rate limit group config updated")
	return nil
}

// GetGroupRateLimit 安全地获取指定用户组的速率限制值
func GetGroupRateLimit(group string) (totalCount, successCount int, found bool) {
	ModelRequestRateLimitGroupMutex.RLock()
	defer ModelRequestRateLimitGroupMutex.RUnlock()

	if ModelRequestRateLimitGroupConfig == nil {
		return 0, 0, false // 配置尚未初始化
	}

	limits, found := ModelRequestRateLimitGroupConfig[group]
	if !found {
		return 0, 0, false
	}
	return limits[0], limits[1], true
}
