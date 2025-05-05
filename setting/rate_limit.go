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
var ModelRequestRateLimitGroup map[string][2]int

var ModelRequestRateLimitGroupMutex sync.RWMutex

func UpdateModelRequestRateLimitGroup(jsonStr string) error {
	ModelRequestRateLimitGroupMutex.Lock()
	defer ModelRequestRateLimitGroupMutex.Unlock()

	var newConfig map[string][2]int
	if jsonStr == "" || jsonStr == "{}" {
		ModelRequestRateLimitGroup = make(map[string][2]int)
		common.SysLog("Model request rate limit group config cleared")
		return nil
	}

	err := json.Unmarshal([]byte(jsonStr), &newConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ModelRequestRateLimitGroup config: %w", err)
	}

	ModelRequestRateLimitGroup = newConfig
	return nil
}

func GetGroupRateLimit(group string) (totalCount, successCount int, found bool) {
	ModelRequestRateLimitGroupMutex.RLock()
	defer ModelRequestRateLimitGroupMutex.RUnlock()

	if ModelRequestRateLimitGroup == nil {
		return 0, 0, false
	}

	limits, found := ModelRequestRateLimitGroup[group]
	if !found {
		return 0, 0, false
	}
	return limits[0], limits[1], true
}
