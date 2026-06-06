package model

import (
	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

// cleanupRemovedQuotaDataCacheStats drops quota_data columns that only backed
// the removed classic dashboard cache-rate cards and charts.
func cleanupRemovedQuotaDataCacheStats() {
	CleanupRemovedQuotaDataCacheStats(DB)
}

// CleanupRemovedQuotaDataCacheStats applies the cleanup to the provided DB.
// It is used both by normal startup migration and DB pre-migration targets.
func CleanupRemovedQuotaDataCacheStats(db *gorm.DB) {
	if db == nil {
		return
	}

	migrator := db.Migrator()
	for _, col := range []string{"input_tokens", "cache_hit_tokens", "cache_creation_tokens"} {
		if migrator.HasColumn(&QuotaData{}, col) {
			if err := migrator.DropColumn(&QuotaData{}, col); err != nil {
				common.SysError("failed to drop quota_data." + col + ": " + err.Error())
			} else {
				common.SysLog("dropped column quota_data." + col)
			}
		}
	}
}
