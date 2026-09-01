package dbmigrate

import (
	"fmt"

	pricingstore "github.com/NookMux/NookMux/internal/store/pricing"
	"gorm.io/gorm"
)

// ensureModelPriceTableTables protects a rolling upgrade of the component
// price table. Slave nodes do not run AutoMigrate, so they must not start
// against a main database that still lacks either physical table; otherwise
// marketplace reads would silently fall back from the intended configuration.
func ensureModelPriceTableTables(dbHandle *gorm.DB, scope string) error {
	if dbHandle == nil {
		return fmt.Errorf("%s database is not initialized", scope)
	}
	for _, table := range []struct {
		name  string
		model any
	}{
		{name: "model_price_plans", model: &pricingstore.ModelPricePlan{}},
		{name: "model_price_components", model: &pricingstore.ModelPriceComponent{}},
	} {
		if !dbHandle.Migrator().HasTable(table.model) {
			return fmt.Errorf(
				"%s database is missing %s; start the master node first so it can complete the component price table migration",
				scope,
				table.name,
			)
		}
	}
	return nil
}
