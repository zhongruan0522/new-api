package missingmodelstore

import (
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/vendor_meta"
)

// GetMissingModels returns model names that are referenced in the system
func GetMissingModels() ([]string, error) {
	// 1. 获取所有已启用模型（去重）
	models := channelstore.GetEnabledModels()
	if len(models) == 0 {
		return []string{}, nil
	}

	// 2. 查询已有的元数据模型名
	var existing []string
	if err := dbstore.DB.Model(&vendormetastore.Model{}).Where("model_name IN ?", models).Pluck("model_name", &existing).Error; err != nil {
		return nil, err
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		existingSet[e] = struct{}{}
	}

	// 3. 收集缺失模型
	var missing []string
	for _, name := range models {
		if _, ok := existingSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing, nil
}
