package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMiniMaxVoiceTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&MiniMaxVoice{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
	})
}

// createMiniMaxVoice 创建一条音色记录，createdAt 由调用方指定（秒级时间戳）。
func createMiniMaxVoice(t *testing.T, voiceId, voiceType string, createdAt int64) *MiniMaxVoice {
	t.Helper()
	voice := &MiniMaxVoice{
		Type:         voiceType,
		OperatorId:   1,
		OperatorKind: "user",
		VoiceId:      voiceId,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	if err := DB.Create(voice).Error; err != nil {
		t.Fatalf("create voice %s: %v", voiceId, err)
	}
	return voice
}

func TestDeleteExpiredMiniMaxVoicePreviews_DeletesOnlyExpiredPreviews(t *testing.T) {
	setupMiniMaxVoiceTestDB(t)

	now := time.Now().Unix()
	// 超过 7 天的试听记录：应被删除。
	createMiniMaxVoice(t, "expired-preview-1", MiniMaxVoiceTypePreview, now-8*24*3600)
	// 未超过 7 天的试听记录：应保留。
	recent := createMiniMaxVoice(t, "recent-preview-1", MiniMaxVoiceTypePreview, now-1*24*3600)
	// 超过 7 天的“已创建”记录：不应被删除（仅清理试听中）。
	createdOld := createMiniMaxVoice(t, "created-old-1", MiniMaxVoiceTypeCreated, now-30*24*3600)

	cutoff := now - 7*24*3600
	affected, err := DeleteExpiredMiniMaxVoicePreviews(cutoff)
	if err != nil {
		t.Fatalf("DeleteExpiredMiniMaxVoicePreviews error: %v", err)
	}
	if affected != 1 {
		t.Fatalf("affected = %d, want 1", affected)
	}

	// 验证过期试听已被删除。
	var cnt int64
	if err := DB.Model(&MiniMaxVoice{}).Where("voice_id = ?", "expired-preview-1").Count(&cnt).Error; err != nil {
		t.Fatalf("count expired preview: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("expired preview still exists")
	}

	// 验证近期试听保留。
	var gotRecent MiniMaxVoice
	if err := DB.Where("voice_id = ?", recent.VoiceId).First(&gotRecent).Error; err != nil {
		t.Fatalf("recent preview should remain: %v", err)
	}

	// 验证旧的 created 保留。
	var gotCreated MiniMaxVoice
	if err := DB.Where("voice_id = ?", createdOld.VoiceId).First(&gotCreated).Error; err != nil {
		t.Fatalf("old created should remain: %v", err)
	}
}

func TestDeleteExpiredMiniMaxVoicePreviews_BoundaryKeepsExactlyCutoffAge(t *testing.T) {
	setupMiniMaxVoiceTestDB(t)

	// created_at == cutoff 的记录（恰好 7 天）不应被删除（条件为 created_at < cutoff）。
	now := time.Now().Unix()
	cutoff := now - 7*24*3600
	createMiniMaxVoice(t, "boundary-preview", MiniMaxVoiceTypePreview, cutoff)

	affected, err := DeleteExpiredMiniMaxVoicePreviews(cutoff)
	if err != nil {
		t.Fatalf("DeleteExpiredMiniMaxVoicePreviews error: %v", err)
	}
	if affected != 0 {
		t.Fatalf("affected = %d, want 0 (boundary record should be kept)", affected)
	}

	var cnt int64
	if err := DB.Model(&MiniMaxVoice{}).Where("voice_id = ?", "boundary-preview").Count(&cnt).Error; err != nil {
		t.Fatalf("count boundary preview: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("boundary preview should remain, got count %d", cnt)
	}
}

func TestDeleteExpiredMiniMaxVoicePreviews_NoopOnNilDB(t *testing.T) {
	// DB 未初始化时应安全跳过，不 panic。
	oldDB := DB
	DB = nil
	t.Cleanup(func() { DB = oldDB })

	affected, err := DeleteExpiredMiniMaxVoicePreviews(time.Now().Unix())
	if err != nil {
		t.Fatalf("unexpected error on nil DB: %v", err)
	}
	if affected != 0 {
		t.Fatalf("affected = %d, want 0 on nil DB", affected)
	}
}
