package dbmigrate

import (
	"fmt"

	"github.com/NookMux/NookMux/internal/store/log"
	"gorm.io/gorm"
)

// ensureLogBillingDetailsColumn protects rolling upgrades. Slave nodes do not
// run migrations, but the process model now includes billing_details. If the
// shared database has not been migrated yet, GORM's INSERT would include the
// missing column and lose consume logs after quota settlement. Fail startup
// instead of writing silently into an incompatible schema.
func ensureLogBillingDetailsColumn(dbHandle *gorm.DB, scope string) error {
	if dbHandle == nil {
		return fmt.Errorf("%s database is not initialized", scope)
	}
	if dbHandle.Migrator().HasTable(&logstore.Log{}) &&
		!dbHandle.Migrator().HasColumn(&logstore.Log{}, "billing_details") {
		return fmt.Errorf(
			"%s database is missing logs.billing_details; start the master node first so it can complete the billing migration", scope,
		)
	}
	return nil
}
