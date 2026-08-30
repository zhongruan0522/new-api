package dbmigrate

import (
	"fmt"
	"testing"

	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestEnsureLogBillingDetailsColumnBlocksUnmigratedSlaveSchema verifies the
// rolling-upgrade guard added for the billing_details column: an old logs table
// must prevent a slave node from starting instead of causing asynchronous log
// INSERTs to fail after quota has already been settled.
func TestEnsureLogBillingDetailsColumnBlocksUnmigratedSlaveSchema(t *testing.T) {
	dbHandle, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := dbHandle.AutoMigrate(&logstore.Log{}); err != nil {
		t.Fatalf("migrate current schema: %v", err)
	}
	if err := ensureLogBillingDetailsColumn(dbHandle, "test"); err != nil {
		t.Fatalf("current schema should be accepted: %v", err)
	}

	if err := dbHandle.Migrator().DropColumn(&logstore.Log{}, "billing_details"); err != nil {
		t.Fatalf("simulate historical schema: %v", err)
	}
	err = ensureLogBillingDetailsColumn(dbHandle, "test")
	if err == nil {
		t.Fatal("historical schema must block slave startup")
	}
	want := "test database is missing logs.billing_details"
	if len(err.Error()) < len(want) || err.Error()[:len(want)] != want {
		t.Fatalf("error = %q, want prefix %q", err.Error(), want)
	}
}
