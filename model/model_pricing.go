package model

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
	"github.com/zhongruan0522/new-api/types"
	"gorm.io/gorm"
)

const (
	modelPricingOptionModelRatio           = "ModelRatio"
	modelPricingOptionModelPrice           = "ModelPrice"
	modelPricingOptionCacheRatio           = "CacheRatio"
	modelPricingOptionCreateCacheRatio     = "CreateCacheRatio"
	modelPricingOptionCompletionRatio      = "CompletionRatio"
	modelPricingOptionAudioRatio           = "AudioRatio"
	modelPricingOptionAudioCompletionRatio = "AudioCompletionRatio"
	modelPricingOptionContextPricing       = "ContextPricing"
)

var modelPricingNumericColumns = map[string]string{
	modelPricingOptionModelRatio:           "model_ratio",
	modelPricingOptionModelPrice:           "model_price",
	modelPricingOptionCacheRatio:           "cache_ratio",
	modelPricingOptionCreateCacheRatio:     "create_cache_ratio",
	modelPricingOptionCompletionRatio:      "completion_ratio",
	modelPricingOptionAudioRatio:           "audio_ratio",
	modelPricingOptionAudioCompletionRatio: "audio_completion_ratio",
}

// ModelPricing stores all model-level pricing dimensions in one row per model
// name or wildcard pattern. Pointer fields are intentional: nil means the
// dimension is not configured, while a non-nil pointer to 0 means explicitly
// configured as free/zero-ratio.
type ModelPricing struct {
	Id                   int64    `json:"id" gorm:"primaryKey"`
	ModelName            string   `json:"model_name" gorm:"size:191;not null;uniqueIndex"`
	ModelPrice           *float64 `json:"model_price,omitempty"`
	ModelRatio           *float64 `json:"model_ratio,omitempty"`
	CompletionRatio      *float64 `json:"completion_ratio,omitempty"`
	CacheRatio           *float64 `json:"cache_ratio,omitempty"`
	CreateCacheRatio     *float64 `json:"create_cache_ratio,omitempty"`
	AudioRatio           *float64 `json:"audio_ratio,omitempty"`
	AudioCompletionRatio *float64 `json:"audio_completion_ratio,omitempty"`
	ContextPricing       *string  `json:"context_pricing,omitempty" gorm:"type:text"`
	CreatedAt            int64    `json:"created_at" gorm:"not null"`
	UpdatedAt            int64    `json:"updated_at" gorm:"not null"`
}

func (ModelPricing) TableName() string {
	return "model_pricings"
}

func IsModelPricingOptionKey(key string) bool {
	if key == modelPricingOptionContextPricing {
		return true
	}
	_, ok := modelPricingNumericColumns[key]
	return ok
}

func HasModelPricingRows() bool {
	if DB == nil || !DB.Migrator().HasTable(&ModelPricing{}) {
		return false
	}
	var count int64
	if err := DB.Model(&ModelPricing{}).Count(&count).Error; err != nil {
		common.SysError("failed to count model pricing rows: " + err.Error())
		return false
	}
	return count > 0
}

func SyncModelPricingTableAndCache() error {
	if DB == nil || !DB.Migrator().HasTable(&ModelPricing{}) {
		return nil
	}
	if !HasModelPricingRows() {
		if err := replaceModelPricingTableFromCurrentSettings(); err != nil {
			return err
		}
	}
	if err := RefreshModelPricingCacheFromDatabase(); err != nil {
		return err
	}
	syncModelPricingOptionsFromCache()
	return nil
}

func RefreshModelPricingCacheFromDatabase() error {
	rows, err := GetModelPricingRows()
	if err != nil {
		return err
	}
	return applyModelPricingRowsToCache(rows)
}

func GetModelPricingRows() ([]ModelPricing, error) {
	var rows []ModelPricing
	err := DB.Order("model_name ASC").Find(&rows).Error
	return rows, err
}

func UpdateModelPricingByOption(key string, jsonValue string) error {
	if !IsModelPricingOptionKey(key) {
		return nil
	}
	if DB == nil || !DB.Migrator().HasTable(&ModelPricing{}) {
		if err := updateModelPricingCacheByOption(key, jsonValue); err != nil {
			return err
		}
		syncModelPricingOptionsFromCache()
		return nil
	}

	if key == modelPricingOptionContextPricing {
		configs, err := parseContextPricingOption(jsonValue)
		if err != nil {
			return err
		}
		if err := replaceContextPricingColumn(configs); err != nil {
			return err
		}
	} else {
		values, err := parseNumericPricingOption(jsonValue)
		if err != nil {
			return err
		}
		if err := replaceNumericPricingColumn(key, values); err != nil {
			return err
		}
	}

	if err := RefreshModelPricingCacheFromDatabase(); err != nil {
		return err
	}
	syncModelPricingOptionsFromCache()
	return nil
}

func updateModelPricingCacheByOption(key string, jsonValue string) error {
	switch key {
	case modelPricingOptionModelRatio:
		return ratio_setting.UpdateModelRatioByJSONString(jsonValue)
	case modelPricingOptionModelPrice:
		return ratio_setting.UpdateModelPriceByJSONString(jsonValue)
	case modelPricingOptionCacheRatio:
		return ratio_setting.UpdateCacheRatioByJSONString(jsonValue)
	case modelPricingOptionCreateCacheRatio:
		return ratio_setting.UpdateCreateCacheRatioByJSONString(jsonValue)
	case modelPricingOptionCompletionRatio:
		return ratio_setting.UpdateCompletionRatioByJSONString(jsonValue)
	case modelPricingOptionAudioRatio:
		return ratio_setting.UpdateAudioRatioByJSONString(jsonValue)
	case modelPricingOptionAudioCompletionRatio:
		return ratio_setting.UpdateAudioCompletionRatioByJSONString(jsonValue)
	case modelPricingOptionContextPricing:
		return ratio_setting.UpdateContextPricingByJSONString(jsonValue)
	default:
		return nil
	}
}

func replaceModelPricingTableFromCurrentSettings() error {
	rowsByModelName := make(map[string]*ModelPricing)
	now := time.Now().Unix()

	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetModelPriceMap(), func(row *ModelPricing, value float64) {
		row.ModelPrice = float64Pointer(value)
	})
	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetModelRatioCopy(), func(row *ModelPricing, value float64) {
		row.ModelRatio = float64Pointer(value)
	})
	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetCompletionRatioCopy(), func(row *ModelPricing, value float64) {
		row.CompletionRatio = float64Pointer(value)
	})
	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetCacheRatioCopy(), func(row *ModelPricing, value float64) {
		row.CacheRatio = float64Pointer(value)
	})
	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetCreateCacheRatioCopy(), func(row *ModelPricing, value float64) {
		row.CreateCacheRatio = float64Pointer(value)
	})
	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetAudioRatioCopy(), func(row *ModelPricing, value float64) {
		row.AudioRatio = float64Pointer(value)
	})
	mergeNumericPricingMap(rowsByModelName, ratio_setting.GetAudioCompletionRatioCopy(), func(row *ModelPricing, value float64) {
		row.AudioCompletionRatio = float64Pointer(value)
	})

	for modelName, config := range ratio_setting.GetContextPricingCopy() {
		row := getOrCreateModelPricingRow(rowsByModelName, modelName, now)
		configJSON, err := common.Marshal(config)
		if err != nil {
			return fmt.Errorf("marshal context pricing for %s: %w", modelName, err)
		}
		contextPricing := string(configJSON)
		row.ContextPricing = &contextPricing
	}

	if len(rowsByModelName) == 0 {
		return nil
	}
	rows := make([]ModelPricing, 0, len(rowsByModelName))
	for _, row := range rowsByModelName {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(leftIndex, rightIndex int) bool {
		return rows[leftIndex].ModelName < rows[rightIndex].ModelName
	})

	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&ModelPricing{}).Error; err != nil {
			return err
		}
		return tx.Create(&rows).Error
	})
}

func mergeNumericPricingMap(rowsByModelName map[string]*ModelPricing, values map[string]float64, assign func(*ModelPricing, float64)) {
	now := time.Now().Unix()
	for modelName, value := range values {
		row := getOrCreateModelPricingRow(rowsByModelName, modelName, now)
		assign(row, value)
	}
}

func getOrCreateModelPricingRow(rowsByModelName map[string]*ModelPricing, modelName string, now int64) *ModelPricing {
	row, exists := rowsByModelName[modelName]
	if exists {
		return row
	}
	row = &ModelPricing{ModelName: modelName, CreatedAt: now, UpdatedAt: now}
	rowsByModelName[modelName] = row
	return row
}

func applyModelPricingRowsToCache(rows []ModelPricing) error {
	modelPriceMap := make(map[string]float64)
	modelRatioMap := make(map[string]float64)
	completionRatioMap := make(map[string]float64)
	cacheRatioMap := make(map[string]float64)
	createCacheRatioMap := make(map[string]float64)
	audioRatioMap := make(map[string]float64)
	audioCompletionRatioMap := make(map[string]float64)
	contextPricingMap := make(map[string]types.ContextPricingConfig)

	for _, row := range rows {
		modelName := row.ModelName
		if row.ModelPrice != nil {
			modelPriceMap[modelName] = *row.ModelPrice
		}
		if row.ModelRatio != nil {
			modelRatioMap[modelName] = *row.ModelRatio
		}
		if row.CompletionRatio != nil {
			completionRatioMap[modelName] = *row.CompletionRatio
		}
		if row.CacheRatio != nil {
			cacheRatioMap[modelName] = *row.CacheRatio
		}
		if row.CreateCacheRatio != nil {
			createCacheRatioMap[modelName] = *row.CreateCacheRatio
		}
		if row.AudioRatio != nil {
			audioRatioMap[modelName] = *row.AudioRatio
		}
		if row.AudioCompletionRatio != nil {
			audioCompletionRatioMap[modelName] = *row.AudioCompletionRatio
		}
		if row.ContextPricing != nil && strings.TrimSpace(*row.ContextPricing) != "" {
			var config types.ContextPricingConfig
			if err := common.UnmarshalJsonStr(*row.ContextPricing, &config); err != nil {
				return fmt.Errorf("parse context pricing for %s: %w", modelName, err)
			}
			contextPricingMap[modelName] = config
		}
	}

	return updatePricingCacheMaps(modelPriceMap, modelRatioMap, completionRatioMap, cacheRatioMap, createCacheRatioMap, audioRatioMap, audioCompletionRatioMap, contextPricingMap)
}

func updatePricingCacheMaps(
	modelPriceMap map[string]float64,
	modelRatioMap map[string]float64,
	completionRatioMap map[string]float64,
	cacheRatioMap map[string]float64,
	createCacheRatioMap map[string]float64,
	audioRatioMap map[string]float64,
	audioCompletionRatioMap map[string]float64,
	contextPricingMap map[string]types.ContextPricingConfig,
) error {
	serializedModelPrice, err := common.Marshal(modelPriceMap)
	if err != nil {
		return err
	}
	serializedModelRatio, err := common.Marshal(modelRatioMap)
	if err != nil {
		return err
	}
	serializedCompletionRatio, err := common.Marshal(completionRatioMap)
	if err != nil {
		return err
	}
	serializedCacheRatio, err := common.Marshal(cacheRatioMap)
	if err != nil {
		return err
	}
	serializedCreateCacheRatio, err := common.Marshal(createCacheRatioMap)
	if err != nil {
		return err
	}
	serializedAudioRatio, err := common.Marshal(audioRatioMap)
	if err != nil {
		return err
	}
	serializedAudioCompletionRatio, err := common.Marshal(audioCompletionRatioMap)
	if err != nil {
		return err
	}
	serializedContextPricing, err := common.Marshal(contextPricingMap)
	if err != nil {
		return err
	}

	if err := ratio_setting.UpdateModelPriceByJSONString(string(serializedModelPrice)); err != nil {
		return err
	}
	if err := ratio_setting.UpdateModelRatioByJSONString(string(serializedModelRatio)); err != nil {
		return err
	}
	if err := ratio_setting.UpdateCompletionRatioByJSONString(string(serializedCompletionRatio)); err != nil {
		return err
	}
	if err := ratio_setting.UpdateCacheRatioByJSONString(string(serializedCacheRatio)); err != nil {
		return err
	}
	if err := ratio_setting.UpdateCreateCacheRatioByJSONString(string(serializedCreateCacheRatio)); err != nil {
		return err
	}
	if err := ratio_setting.UpdateAudioRatioByJSONString(string(serializedAudioRatio)); err != nil {
		return err
	}
	if err := ratio_setting.UpdateAudioCompletionRatioByJSONString(string(serializedAudioCompletionRatio)); err != nil {
		return err
	}
	return ratio_setting.UpdateContextPricingByJSONString(string(serializedContextPricing))
}

func replaceNumericPricingColumn(key string, values map[string]float64) error {
	columnName, exists := modelPricingNumericColumns[key]
	if !exists {
		return fmt.Errorf("unsupported model pricing option key: %s", key)
	}
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ModelPricing{}).Where("1 = 1").Update(columnName, nil).Error; err != nil {
			return err
		}
		for modelName, value := range values {
			if err := upsertModelPricingColumn(tx, modelName, columnName, value, now); err != nil {
				return err
			}
		}
		return deleteEmptyModelPricingRows(tx)
	})
}

func replaceContextPricingColumn(configs map[string]types.ContextPricingConfig) error {
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ModelPricing{}).Where("1 = 1").Update("context_pricing", nil).Error; err != nil {
			return err
		}
		for modelName, config := range configs {
			serializedConfig, err := common.Marshal(config)
			if err != nil {
				return err
			}
			if err := upsertModelPricingColumn(tx, modelName, "context_pricing", string(serializedConfig), now); err != nil {
				return err
			}
		}
		return deleteEmptyModelPricingRows(tx)
	})
}

func upsertModelPricingColumn(tx *gorm.DB, modelName string, columnName string, value any, now int64) error {
	row := ModelPricing{ModelName: modelName}
	if err := tx.Where("model_name = ?", modelName).Attrs(ModelPricing{CreatedAt: now, UpdatedAt: now}).FirstOrCreate(&row).Error; err != nil {
		return err
	}
	return tx.Model(&ModelPricing{}).Where("model_name = ?", modelName).Updates(map[string]any{
		columnName:   value,
		"updated_at": now,
	}).Error
}

func deleteEmptyModelPricingRows(tx *gorm.DB) error {
	return tx.Where(
		"model_price IS NULL AND model_ratio IS NULL AND completion_ratio IS NULL AND cache_ratio IS NULL AND create_cache_ratio IS NULL AND audio_ratio IS NULL AND audio_completion_ratio IS NULL AND context_pricing IS NULL",
	).Delete(&ModelPricing{}).Error
}

func parseNumericPricingOption(jsonValue string) (map[string]float64, error) {
	values := make(map[string]float64)
	if err := common.UnmarshalJsonStr(jsonValue, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func parseContextPricingOption(jsonValue string) (map[string]types.ContextPricingConfig, error) {
	if err := ratio_setting.ValidateContextPricing(jsonValue); err != nil {
		return nil, err
	}
	configs := make(map[string]types.ContextPricingConfig)
	if err := common.UnmarshalJsonStr(jsonValue, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

func syncModelPricingOptionsFromCache() {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[modelPricingOptionModelRatio] = ratio_setting.ModelRatio2JSONString()
	common.OptionMap[modelPricingOptionModelPrice] = ratio_setting.ModelPrice2JSONString()
	common.OptionMap[modelPricingOptionCacheRatio] = ratio_setting.CacheRatio2JSONString()
	common.OptionMap[modelPricingOptionCreateCacheRatio] = ratio_setting.CreateCacheRatio2JSONString()
	common.OptionMap[modelPricingOptionCompletionRatio] = ratio_setting.CompletionRatio2JSONString()
	common.OptionMap[modelPricingOptionAudioRatio] = ratio_setting.AudioRatio2JSONString()
	common.OptionMap[modelPricingOptionAudioCompletionRatio] = ratio_setting.AudioCompletionRatio2JSONString()
	common.OptionMap[modelPricingOptionContextPricing] = ratio_setting.ContextPricing2JSONString()
}

func float64Pointer(value float64) *float64 {
	return &value
}
