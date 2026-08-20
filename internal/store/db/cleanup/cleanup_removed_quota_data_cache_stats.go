package dbcleanup

import (
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/store/usedata"
	"gorm.io/gorm"
)

// CleanupRemovedQuotaDataCacheStats applies the cleanup to the provided DB.
// It is used both by normal startup migration and DB pre-migration targets.
func CleanupRemovedQuotaDataCacheStats(db *gorm.DB) {
	if db == nil {
		return
	}

	migrator := db.Migrator()
	for _, col := range []string{"input_tokens", "cache_hit_tokens", "cache_creation_tokens"} {
		if migrator.HasColumn(&usedatastore.QuotaData{}, col) {
			if err := migrator.DropColumn(&usedatastore.QuotaData{}, col); err != nil {
				common.SysError("failed to drop quota_data." + col + ": " + err.Error())
			} else {
				common.SysLog("dropped column quota_data." + col)
			}
		}
	}
}
