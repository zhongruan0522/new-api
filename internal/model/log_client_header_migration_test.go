package model

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/gin-gonic/gin"
)

func TestSerializeLogOtherNormalizesNilAndEmpty(t *testing.T) {
	if got := serializeLogOther(nil); got != "{}" {
		t.Fatalf("serializeLogOther(nil) = %q, want {}", got)
	}
	if got := serializeLogOther(map[string]interface{}{}); got != "{}" {
		t.Fatalf("serializeLogOther(empty) = %q, want {}", got)
	}
	got := serializeLogOther(map[string]interface{}{"k": "v"})
	if got != `{"k":"v"}` {
		t.Fatalf("serializeLogOther(map) = %q, want {\"k\":\"v\"}", got)
	}
}

func TestRecordErrorLogStoresObjectForNilOther(t *testing.T) {
	setupLogAdminInfoTestDB(t)
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "http://example.test/v1/chat/completions", nil)
	c.Set("username", "target-user")

	// nil Other must serialize to a JSON object, not the literal "null".
	RecordErrorLog(c, 1, 0, "gpt-test", "token", "boom", 0, 10, false, "default", nil)

	var stored Log
	if err := LOG_DB.First(&stored).Error; err != nil {
		t.Fatalf("query stored log: %v", err)
	}
	if stored.Other != "{}" {
		t.Fatalf("stored Other for nil input = %q, want {}", stored.Other)
	}
}

func TestExtractBackfillUpdatesMigratesAndCleansOther(t *testing.T) {
	other := `{"ua":"MyClient/1.0","x_title":"","http_referer":"https://example.test/path","keep":"yes"}`
	updates := extractBackfillUpdates(other)

	if got := updates["ua"]; got != "MyClient/1.0" {
		t.Fatalf("updates[ua] = %v, want MyClient/1.0", got)
	}
	if got := updates["http_referer"]; got != "https://example.test/path" {
		t.Fatalf("updates[http_referer] = %v, want https://example.test/path", got)
	}
	// Empty x_title should not be copied to the dedicated column.
	if _, ok := updates["x_title"]; ok {
		t.Fatalf("updates should not include empty x_title, got %v", updates)
	}
	// The rewritten `other` JSON must drop all three header keys but keep others.
	rewritten, ok := updates["other"].(string)
	if !ok {
		t.Fatalf("updates should include rewritten other JSON, got %v", updates)
	}
	for _, key := range []string{"ua", "x_title", "http_referer"} {
		if strings.Contains(rewritten, key) {
			t.Fatalf("rewritten other still contains %q: %s", key, rewritten)
		}
	}
	if !strings.Contains(rewritten, `"keep":"yes"`) {
		t.Fatalf("rewritten other dropped unrelated key: %s", rewritten)
	}

	// Re-running on the rewritten other must be a no-op (idempotent).
	secondUpdates := extractBackfillUpdates(rewritten)
	if len(secondUpdates) != 0 {
		t.Fatalf("second pass should produce no updates, got %v", secondUpdates)
	}
}

func TestExtractBackfillUpdatesReturnsEmptyObjectAfterFullMigration(t *testing.T) {
	// When the only contents are header keys, the rewritten other must be "{}"
	// rather than the literal "null".
	updates := extractBackfillUpdates(`{"ua":"OnlyUA/1.0"}`)
	if got := updates["ua"]; got != "OnlyUA/1.0" {
		t.Fatalf("updates[ua] = %v, want OnlyUA/1.0", got)
	}
	rewritten, ok := updates["other"].(string)
	if !ok {
		t.Fatalf("updates should include rewritten other JSON, got %v", updates)
	}
	if rewritten != "{}" {
		t.Fatalf("rewritten other for emptied map = %q, want {}", rewritten)
	}
}

func TestBackfillLogClientHeaderColumnsWritesColumnsAndCleansOther(t *testing.T) {
	setupLogAdminInfoTestDB(t)
	// The migration marker lives in the options table, which the shared test
	// helper does not migrate; create it here so the one-shot marker works.
	if err := DB.AutoMigrate(&Option{}); err != nil {
		t.Fatalf("migrate options table: %v", err)
	}

	legacy := &Log{
		UserId:    1,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeConsume,
		Other:     `{"ua":"LegacyAgent/2.0","x_title":"LegacyTitle","http_referer":"https://legacy.test/refer","model_ratio":1.5}`,
	}
	if err := LOG_DB.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy log: %v", err)
	}

	// Pre-seed the marker so the real backfill is not skipped; clear it right before
	// the call to ensure the migration under test actually executes.
	if err := DB.Where(&Option{Key: logClientHeaderBackfillMarker}).Delete(&Option{}).Error; err != nil {
		t.Fatalf("clear marker before backfill: %v", err)
	}
	backfillLogClientHeaderColumns()

	var stored Log
	if err := LOG_DB.First(&stored, legacy.Id).Error; err != nil {
		t.Fatalf("query backfilled log: %v", err)
	}
	if stored.Ua != "LegacyAgent/2.0" {
		t.Fatalf("backfilled ua = %q, want LegacyAgent/2.0", stored.Ua)
	}
	if stored.XTitle != "LegacyTitle" {
		t.Fatalf("backfilled x_title = %q, want LegacyTitle", stored.XTitle)
	}
	if stored.HttpReferer != "https://legacy.test/refer" {
		t.Fatalf("backfilled http_referer = %q, want https://legacy.test/refer", stored.HttpReferer)
	}
	for _, key := range []string{"ua", "x_title", "http_referer"} {
		if strings.Contains(stored.Other, key) {
			t.Fatalf("other still contains migrated header key %q: %s", key, stored.Other)
		}
	}
	if !strings.Contains(stored.Other, `"model_ratio":1.5`) {
		t.Fatalf("other dropped unrelated field model_ratio: %s", stored.Other)
	}

	// A successful backfill must persist the one-shot marker.
	var count int64
	if err := DB.Model(&Option{}).Where(&Option{Key: logClientHeaderBackfillMarker}).Count(&count).Error; err != nil {
		t.Fatalf("count marker after backfill: %v", err)
	}
	if count != 1 {
		t.Fatalf("marker count after successful backfill = %d, want 1", count)
	}
}
