package optionstore

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/store/db"
)

// One-shot, versioned data-migration markers.
//
// Some startup data cleanups (e.g. sanitizing legacy JSON columns across all
// users/channels) are expensive and only need to run once. To avoid re-scanning
// large tables on every boot, we record a marker row in the options table after
// such a migration has fully completed. On subsequent startups the migration is
// skipped when the marker is present.
//
// Key shape: data_migration.<name>.v<n>.done
//   - <name>  identifies the logical migration
//   - v<n>    bumps when the migration logic itself changes; bumping the version
//     (which changes the key) forces a fresh run of the new logic.
//
// To force a re-run (e.g. after importing legacy data from an older release),
// delete the corresponding options row.

// IsDataMigrationDone reports whether a one-shot data-migration marker is set.
//
// It reads the options table directly rather than common.OptionMap, because
// startup migrations run before InitOptionMap() populates the in-memory map.
//
// On read failure it conservatively returns false (treated as "not done") and
// logs a warning, so a flaky read can never silently skip a migration. The
// migrations themselves are idempotent, so an extra run on the next startup is
// safe.
func IsDataMigrationDone(marker string) bool {
	if dbstore.DB == nil {
		return false
	}
	var count int64
	if err := dbstore.DB.Model(&Option{}).Where(dbstore.CommonKeyCol+" = ?", marker).Count(&count).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to read data migration marker %s: %v (will conservatively run migration)", marker, err))
		return false
	}
	return count > 0
}

// MarkDataMigrationDone persists a one-shot data-migration marker.
//
// It is best-effort by design: a write failure only logs a warning and never
// aborts startup, because the substantive migration has already succeeded and is
// idempotent. A missing marker just causes one extra (safe) re-run on the next
// startup, which is strictly safer than failing a boot after successful work.
func MarkDataMigrationDone(marker string) {
	if dbstore.DB == nil {
		return
	}
	opt := Option{Key: marker}
	if err := dbstore.DB.FirstOrCreate(&opt, Option{Key: marker}).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to persist data migration marker %s: %v (migration will re-run next startup, which is safe)", marker, err))
		return
	}
	if opt.Value == "true" {
		return
	}
	if err := dbstore.DB.Model(&Option{}).Where(dbstore.CommonKeyCol+" = ?", marker).Update("value", "true").Error; err != nil {
		common.SysError(fmt.Sprintf("failed to persist data migration marker %s: %v (migration will re-run next startup, which is safe)", marker, err))
	}
}
