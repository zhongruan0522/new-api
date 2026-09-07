package dbmigrate

import (
	"context"
	"fmt"
	"time"

	"github.com/NookMux/NookMux/internal/store/log"
	"gorm.io/gorm"
)

// ensureLogBillingDetailsColumn gates slave startup on schema AND completed data.
// Old writers must already have been drained (see the billing PRD). Slaves do not
// run migrations, but the process model now includes billing_details and its
// row-level migration version. If the shared database has not been migrated
// yet, GORM's INSERT would include the missing columns and lose consume logs
// after quota settlement. Fail startup instead of writing silently into an
// incompatible schema.
func ensureLogBillingDetailsColumn(dbHandle *gorm.DB, scope string) error {
	if dbHandle == nil {
		return fmt.Errorf("%s database is not initialized", scope)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dbHandle = dbHandle.WithContext(ctx)
	if !dbHandle.Migrator().HasTable(&logstore.Log{}) {
		return fmt.Errorf("%s database is missing logs; complete the master migration before starting slaves", scope)
	}
	for _, column := range []string{"billing_details", "billing_details_version"} {
		if !dbHandle.Migrator().HasColumn(&logstore.Log{}, column) {
			return fmt.Errorf(
				"%s database is missing logs.%s; start the master node first so it can complete the billing migration",
				scope, column,
			)
		}
	}
	if !dbHandle.Migrator().HasTable(&logBillingMigrationState{}) {
		return fmt.Errorf("%s database billing details migration is incomplete; stop old writers and complete the master migration before starting slaves", scope)
	}
	var count int64
	if err := dbHandle.Model(&logBillingMigrationState{}).Where("version = ?", logstore.LogBillingDetailsVersion).Count(&count).Error; err != nil {
		return fmt.Errorf("%s database billing migration marker: %w", scope, err)
	}
	if count != 1 {
		return fmt.Errorf("%s database billing details migration is incomplete; wait for the master to finish", scope)
	}
	return nil
}
