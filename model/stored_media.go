package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// StoredMediaListItem is a lightweight row used by the multimodal file management UI.
// It intentionally does NOT include the binary payload.
type StoredMediaListItem struct {
	MediaType string `json:"media_type" gorm:"column:media_type"` // image | video
	Id        string `json:"id" gorm:"column:id"`
	UserId    int    `json:"user_id" gorm:"column:user_id"`
	ChannelId int    `json:"channel_id" gorm:"column:channel_id"`
	CreatedAt int64  `json:"created_at" gorm:"column:created_at"`
	MimeType  string `json:"mime_type" gorm:"column:mime_type"`
	SizeBytes int    `json:"size_bytes" gorm:"column:size_bytes"`
	Sha256    string `json:"sha256" gorm:"column:sha256"`
}

func GetAllStoredMedia(ctx context.Context, startTimestamp, endTimestamp int64, startIdx, pageSize int) ([]StoredMediaListItem, int64, error) {
	return queryStoredMedia(ctx, 0, startTimestamp, endTimestamp, startIdx, pageSize)
}

func GetUserStoredMedia(ctx context.Context, userId int, startTimestamp, endTimestamp int64, startIdx, pageSize int) ([]StoredMediaListItem, int64, error) {
	if userId <= 0 {
		return nil, 0, errors.New("user_id is required")
	}
	return queryStoredMedia(ctx, userId, startTimestamp, endTimestamp, startIdx, pageSize)
}

func queryStoredMedia(ctx context.Context, userId int, startTimestamp, endTimestamp int64, startIdx, pageSize int) ([]StoredMediaListItem, int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if startIdx < 0 {
		startIdx = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	whereClauses := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if userId > 0 {
		whereClauses = append(whereClauses, "user_id = ?")
		args = append(args, userId)
	}
	if startTimestamp > 0 {
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, startTimestamp)
	}
	if endTimestamp > 0 {
		whereClauses = append(whereClauses, "created_at <= ?")
		args = append(args, endTimestamp)
	}

	cond := ""
	if len(whereClauses) > 0 {
		cond = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// NOTE: keep this SQL portable across SQLite/MySQL/PostgreSQL.
	baseUnionSQL := fmt.Sprintf(
		"SELECT 'image' AS media_type, id, user_id, channel_id, created_at, mime_type, size_bytes, sha256 FROM stored_images%s "+
			"UNION ALL "+
			"SELECT 'video' AS media_type, id, user_id, channel_id, created_at, mime_type, size_bytes, sha256 FROM stored_videos%s",
		cond,
		cond,
	)

	// The WHERE conditions apply to both SELECTs, so we must provide the args twice.
	unionArgs := make([]any, 0, len(args)*2+2)
	unionArgs = append(unionArgs, args...)
	unionArgs = append(unionArgs, args...)

	var total int64
	countSQL := "SELECT COUNT(1) FROM (" + baseUnionSQL + ") t"
	if err := DB.WithContext(ctx).Raw(countSQL, unionArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	listSQL := baseUnionSQL + " ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	listArgs := append(unionArgs, pageSize, startIdx)

	var items []StoredMediaListItem
	if err := DB.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

