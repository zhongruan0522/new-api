package model

import (
	"context"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// StoredVideo stores user-provided video bytes for the "multimodal auto convert to URL" feature.
// These assets are intended for tool access (e.g. video understanding MCP) and can be cleaned
// up via the existing log cleanup flow.
type StoredVideo struct {
	Id        string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId    int       `json:"user_id" gorm:"index"`
	ChannelId int       `json:"channel_id" gorm:"index"`
	CreatedAt int64     `json:"created_at" gorm:"bigint;index"`
	MimeType  string    `json:"mime_type" gorm:"type:varchar(255);default:''"`
	SizeBytes int       `json:"size_bytes" gorm:"default:0"`
	Sha256    string    `json:"sha256" gorm:"type:char(64);index"`
	Data      LargeBlob `json:"-" gorm:"not null"`
}

func (v *StoredVideo) Insert(ctx context.Context) error {
	if v == nil {
		return errors.New("stored video is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if v.Id == "" {
		v.Id = common.GetUUID()
	}
	if v.CreatedAt == 0 {
		v.CreatedAt = common.GetTimestamp()
	}
	return DB.WithContext(ctx).Create(v).Error
}

func GetStoredVideoByID(ctx context.Context, id string) (*StoredVideo, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var v StoredVideo
	if err := DB.WithContext(ctx).Where("id = ?", id).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func GetStoredVideoByUserAndSha(ctx context.Context, userId int, sha256 string) (*StoredVideo, error) {
	if userId <= 0 {
		return nil, errors.New("user_id is required")
	}
	sha256 = strings.TrimSpace(sha256)
	if sha256 == "" {
		return nil, errors.New("sha256 is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var v StoredVideo
	if err := DB.WithContext(ctx).Where("user_id = ? AND sha256 = ?", userId, sha256).Order("created_at asc").First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func GetStoredVideoMetaByID(ctx context.Context, id string) (*StoredVideo, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var v StoredVideo
	if err := DB.WithContext(ctx).Model(&StoredVideo{}).
		Select("id", "user_id", "channel_id", "created_at", "mime_type", "size_bytes", "sha256").
		Where("id = ?", id).
		First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func DeleteStoredVideosByIDs(ctx context.Context, ids []string, userId int) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db := DB.WithContext(ctx).Where("id IN ?", ids)
	if userId > 0 {
		db = db.Where("user_id = ?", userId)
	}
	result := db.Delete(&StoredVideo{})
	return result.RowsAffected, result.Error
}

func DeleteOldStoredVideos(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}

		result := DB.WithContext(ctx).Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&StoredVideo{})
		if result.Error != nil {
			return total, result.Error
		}

		total += result.RowsAffected

		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}

func EnsureStoredVideosPoolLimit(ctx context.Context, maxBytes int64, batchSize int) (int64, error) {
	if maxBytes <= 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var deleted int64 = 0

	for {
		if ctx.Err() != nil {
			return deleted, ctx.Err()
		}

		var totalBytes int64
		if err := DB.WithContext(ctx).Model(&StoredVideo{}).Select("coalesce(sum(size_bytes),0)").Scan(&totalBytes).Error; err != nil {
			return deleted, err
		}
		if totalBytes <= maxBytes {
			return deleted, nil
		}

		var oldest []StoredVideo
		if err := DB.WithContext(ctx).Model(&StoredVideo{}).
			Select("id", "size_bytes").
			Order("created_at asc").
			Order("id asc").
			Limit(batchSize).
			Find(&oldest).Error; err != nil {
			return deleted, err
		}
		if len(oldest) == 0 {
			return deleted, nil
		}

		ids := make([]string, 0, len(oldest))
		for i := range oldest {
			if oldest[i].Id != "" {
				ids = append(ids, oldest[i].Id)
			}
		}
		if len(ids) == 0 {
			return deleted, nil
		}

		result := DB.WithContext(ctx).Where("id IN ?", ids).Delete(&StoredVideo{})
		if result.Error != nil {
			return deleted, result.Error
		}
		if result.RowsAffected == 0 {
			return deleted, nil
		}
		deleted += result.RowsAffected
	}
}
