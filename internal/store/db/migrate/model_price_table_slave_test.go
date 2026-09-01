package dbmigrate

import (
	"fmt"
	"strings"
	"testing"

	pricingstore "github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEnsureModelPriceTableTablesBlocksUnmigratedSlaveSchema(t *testing.T) {
	tests := []struct {
		name    string
		missing any
		table   string
	}{
		{
			name:    "missing plan table",
			missing: &pricingstore.ModelPricePlan{},
			table:   "model_price_plans",
		},
		{
			name:    "missing component table",
			missing: &pricingstore.ModelPriceComponent{},
			table:   "model_price_components",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbHandle, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite: %v", err)
			}
			if err := dbHandle.AutoMigrate(&pricingstore.ModelPricePlan{}, &pricingstore.ModelPriceComponent{}); err != nil {
				t.Fatalf("migrate current schema: %v", err)
			}
			if err := ensureModelPriceTableTables(dbHandle, "test"); err != nil {
				t.Fatalf("current schema should be accepted: %v", err)
			}

			if err := dbHandle.Migrator().DropTable(tt.missing); err != nil {
				t.Fatalf("drop %s: %v", tt.table, err)
			}
			err = ensureModelPriceTableTables(dbHandle, "test")
			if err == nil {
				t.Fatalf("missing %s must block slave startup", tt.table)
			}
			want := "test database is missing " + tt.table
			if !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("error = %q, want prefix %q", err.Error(), want)
			}
		})
	}
}
