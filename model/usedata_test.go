package model

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupQuotaDataTestDB 初始化内存 SQLite 数据库并迁移 QuotaData 表。
// 恢复全局 DB/缓存和写入层配置，确保各用例之间互不干扰。
func setupQuotaDataTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	oldLogDB := LOG_DB
	oldCache := CacheQuotaData
	oldTokens := quotaDataTrackTokens
	oldByModel := quotaDataTrackByModel
	oldByUser := quotaDataTrackByUser

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&QuotaData{}, &Log{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db
	LOG_DB = db
	CacheQuotaData = make(map[string]*QuotaData)

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		LOG_DB = oldLogDB
		CacheQuotaData = oldCache
		quotaDataTrackTokens = oldTokens
		quotaDataTrackByModel = oldByModel
		quotaDataTrackByUser = oldByUser
	})
}

// setTrackingConfig 设置写入层聚合粒度配置。
func setTrackingConfig(t *testing.T, tokens, byModel, byUser bool) {
	t.Helper()
	quotaDataTrackTokens = tokens
	quotaDataTrackByModel = byModel
	quotaDataTrackByUser = byUser
}

// snapshotCache 返回缓存副本，便于断言。
func snapshotCache(t *testing.T) map[string]*QuotaData {
	t.Helper()
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	out := make(map[string]*QuotaData, len(CacheQuotaData))
	for k, v := range CacheQuotaData {
		cp := *v
		out[k] = &cp
	}
	return out
}

// TestLogQuotaDataTrackTokensDisabled 验证禁用 track_tokens 时 token 不累计。
func TestLogQuotaDataTrackTokensDisabled(t *testing.T) {
	setupQuotaDataTestDB(t)
	setTrackingConfig(t, false, true, true)

	LogQuotaData(1, "alice", "gpt-4", 100, 1000, 500)

	snap := snapshotCache(t)
	if len(snap) != 1 {
		t.Fatalf("cache size = %d, want 1", len(snap))
	}
	for _, d := range snap {
		if d.TokenUsed != 0 {
			t.Fatalf("TokenUsed = %d, want 0 when track_tokens disabled", d.TokenUsed)
		}
		if d.Count != 1 {
			t.Fatalf("Count = %d, want 1", d.Count)
		}
		if d.Quota != 100 {
			t.Fatalf("Quota = %d, want 100", d.Quota)
		}
	}
}

// TestLogQuotaDataTrackTokensEnabled 验证启用 track_tokens 时 token 正常累计。
func TestLogQuotaDataTrackTokensEnabled(t *testing.T) {
	setupQuotaDataTestDB(t)
	setTrackingConfig(t, true, true, true)

	LogQuotaData(1, "alice", "gpt-4", 100, 1000, 500)

	snap := snapshotCache(t)
	for _, d := range snap {
		if d.TokenUsed != 500 {
			t.Fatalf("TokenUsed = %d, want 500", d.TokenUsed)
		}
	}
}

// TestLogQuotaDataTrackByUserDisabled 验证禁用 by_user 时不同用户聚合到同一匿名桶（userId=0）。
func TestLogQuotaDataTrackByUserDisabled(t *testing.T) {
	setupQuotaDataTestDB(t)
	setTrackingConfig(t, true, true, false)

	LogQuotaData(1, "alice", "gpt-4", 100, 1000, 10)
	LogQuotaData(2, "bob", "gpt-4", 200, 1000, 20)

	snap := snapshotCache(t)
	if len(snap) != 1 {
		t.Fatalf("cache size = %d, want 1 (all users collapse to anonymous bucket)", len(snap))
	}
	for _, d := range snap {
		if d.UserID != 0 {
			t.Fatalf("UserID = %d, want 0 (anonymous bucket)", d.UserID)
		}
		if d.Username != "" {
			t.Fatalf("Username = %q, want empty", d.Username)
		}
		if d.Count != 2 {
			t.Fatalf("Count = %d, want 2 (merged from two users)", d.Count)
		}
		if d.TokenUsed != 30 {
			t.Fatalf("TokenUsed = %d, want 30", d.TokenUsed)
		}
	}
}

// TestLogQuotaDataTrackByModelDisabled 验证禁用 by_model 时不同模型聚合到同一全局模型桶（modelName=""）。
func TestLogQuotaDataTrackByModelDisabled(t *testing.T) {
	setupQuotaDataTestDB(t)
	setTrackingConfig(t, true, false, true)

	LogQuotaData(1, "alice", "gpt-4", 100, 1000, 10)
	LogQuotaData(1, "alice", "claude-3", 200, 1000, 20)

	snap := snapshotCache(t)
	if len(snap) != 1 {
		t.Fatalf("cache size = %d, want 1 (all models collapse to global bucket)", len(snap))
	}
	for _, d := range snap {
		if d.ModelName != "" {
			t.Fatalf("ModelName = %q, want empty (global bucket)", d.ModelName)
		}
		if d.Count != 2 {
			t.Fatalf("Count = %d, want 2 (merged from two models)", d.Count)
		}
	}
}

// TestLogQuotaErrorDataTrackByUserDisabled 验证失败日志也按配置聚合维度。
func TestLogQuotaErrorDataTrackByUserDisabled(t *testing.T) {
	setupQuotaDataTestDB(t)
	setTrackingConfig(t, true, true, false)

	LogQuotaErrorData(1, "alice", "gpt-4", 1000)
	LogQuotaErrorData(2, "bob", "gpt-4", 1000)

	snap := snapshotCache(t)
	if len(snap) != 1 {
		t.Fatalf("cache size = %d, want 1 (error data collapsed by user)", len(snap))
	}
	for _, d := range snap {
		if d.UserID != 0 {
			t.Fatalf("UserID = %d, want 0", d.UserID)
		}
		if d.FailCount != 2 {
			t.Fatalf("FailCount = %d, want 2", d.FailCount)
		}
	}
}

// TestRecalculateQuotaDataRespectsTrackingConfig 验证重算流程按配置聚合维度和 token 记录。
func TestRecalculateQuotaDataRespectsTrackingConfig(t *testing.T) {
	setupQuotaDataTestDB(t)
	// 全部禁用：所有记录聚合为单条，token 为 0
	setTrackingConfig(t, false, false, false)

	// 构造两条成功日志到 logs 表（type=2），重算入口读取 LOG_DB
	logs := []*Log{
		{UserId: 1, Username: "alice", ModelName: "gpt-4", CreatedAt: 1000, PromptTokens: 10, CompletionTokens: 20, Quota: 100, Type: LogTypeConsume},
		{UserId: 2, Username: "bob", ModelName: "claude-3", CreatedAt: 1000, PromptTokens: 30, CompletionTokens: 40, Quota: 200, Type: LogTypeConsume},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatalf("create logs: %v", err)
	}

	if err := RecalculateQuotaData(0, 2000); err != nil {
		t.Fatalf("RecalculateQuotaData error: %v", err)
	}

	var rows []*QuotaData
	if err := DB.Table("quota_data").Find(&rows).Error; err != nil {
		t.Fatalf("query quota_data: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("quota_data rows = %d, want 1 (all dimensions collapsed)", len(rows))
	}
	r := rows[0]
	if r.UserID != 0 || r.Username != "" || r.ModelName != "" {
		t.Fatalf("dimensions = (user=%d, name=%q, model=%q), want anonymous/global bucket", r.UserID, r.Username, r.ModelName)
	}
	if r.TokenUsed != 0 {
		t.Fatalf("TokenUsed = %d, want 0 when track_tokens disabled", r.TokenUsed)
	}
	if r.Count != 2 {
		t.Fatalf("Count = %d, want 2", r.Count)
	}
	if r.Quota != 300 {
		t.Fatalf("Quota = %d, want 300", r.Quota)
	}
}
