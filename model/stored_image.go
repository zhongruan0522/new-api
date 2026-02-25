package model

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// LargeBlob is a cross-database "large binary" type.
//
// - MySQL:     LONGBLOB (up to 4GB)
// - PostgreSQL: BYTEA
// - SQLite:    BLOB
//
// This avoids MySQL's default BLOB (64KB) limitation for image storage while
// keeping the schema compatible across SQLite/MySQL/PostgreSQL.
type LargeBlob []byte

func (LargeBlob) GormDataType() string {
	return "large_blob"
}

func (LargeBlob) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql":
		return "LONGBLOB"
	case "postgres":
		return "BYTEA"
	case "sqlite":
		return "BLOB"
	default:
		return "BLOB"
	}
}

func (b LargeBlob) Value() (driver.Value, error) {
	if b == nil {
		return nil, nil
	}
	return []byte(b), nil
}

func (b *LargeBlob) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*b = nil
		return nil
	case []byte:
		buf := make([]byte, len(v))
		copy(buf, v)
		*b = LargeBlob(buf)
		return nil
	case string:
		*b = LargeBlob([]byte(v))
		return nil
	default:
		return fmt.Errorf("unsupported scan type for LargeBlob: %T", value)
	}
}

// StoredImage stores user-provided image bytes for the "image auto convert to URL" feature.
// These images are intended for short-term tool access (e.g. image understanding MCP),
// and can be cleaned up via the existing log cleanup flow.
type StoredImage struct {
	Id        string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId    int       `json:"user_id" gorm:"index"`
	ChannelId int       `json:"channel_id" gorm:"index"`
	CreatedAt int64     `json:"created_at" gorm:"bigint;index"`
	MimeType  string    `json:"mime_type" gorm:"type:varchar(255);default:''"`
	SizeBytes int       `json:"size_bytes" gorm:"default:0"`
	Sha256    string    `json:"sha256" gorm:"type:char(64);index"`
	Data      LargeBlob `json:"-" gorm:"not null"`
}

func (img *StoredImage) Insert(ctx context.Context) error {
	if img == nil {
		return errors.New("stored image is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if img.Id == "" {
		img.Id = common.GetUUID()
	}
	if img.CreatedAt == 0 {
		img.CreatedAt = common.GetTimestamp()
	}
	return DB.WithContext(ctx).Create(img).Error
}

func GetStoredImageByID(ctx context.Context, id string) (*StoredImage, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var img StoredImage
	if err := DB.WithContext(ctx).Where("id = ?", id).First(&img).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

func GetStoredImageByUserAndSha(ctx context.Context, userId int, sha256 string) (*StoredImage, error) {
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
	var img StoredImage
	if err := DB.WithContext(ctx).Where("user_id = ? AND sha256 = ?", userId, sha256).Order("created_at asc").First(&img).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

func GetStoredImageMetaByID(ctx context.Context, id string) (*StoredImage, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var img StoredImage
	if err := DB.WithContext(ctx).Model(&StoredImage{}).
		Select("id", "user_id", "channel_id", "created_at", "mime_type", "size_bytes", "sha256").
		Where("id = ?", id).
		First(&img).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

func DeleteStoredImagesByIDs(ctx context.Context, ids []string, userId int) (int64, error) {
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
	result := db.Delete(&StoredImage{})
	return result.RowsAffected, result.Error
}

func DeleteOldStoredImages(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if ctx != nil && ctx.Err() != nil {
			return total, ctx.Err()
		}

		result := DB.WithContext(ctx).Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&StoredImage{})
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

func EnsureStoredImagesPoolLimit(ctx context.Context, maxBytes int64, batchSize int) (int64, error) {
	if maxBytes <= 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = 100
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
		if err := DB.WithContext(ctx).Model(&StoredImage{}).Select("coalesce(sum(size_bytes),0)").Scan(&totalBytes).Error; err != nil {
			return deleted, err
		}
		if totalBytes <= maxBytes {
			return deleted, nil
		}

		// Delete oldest images in batches until within limit.
		var oldest []StoredImage
		if err := DB.WithContext(ctx).Model(&StoredImage{}).
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

		result := DB.WithContext(ctx).Where("id IN ?", ids).Delete(&StoredImage{})
		if result.Error != nil {
			return deleted, result.Error
		}
		if result.RowsAffected == 0 {
			return deleted, nil
		}
		deleted += result.RowsAffected
	}
}
