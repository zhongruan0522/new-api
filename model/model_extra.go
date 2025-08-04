package model

// GetModelEnableGroups 返回指定模型名称可用的用户分组列表。
// 复用缓存的定价映射，避免额外的数据库查询。
func GetModelEnableGroups(modelName string) []string {
    for _, p := range GetPricing() {
        if p.ModelName == modelName {
            return p.EnableGroup
        }
    }
    return make([]string, 0)
}

// GetModelQuotaType 返回指定模型的计费类型（quota_type）。
// 复用缓存的定价映射，避免额外数据库查询。
// 如果未找到对应模型，默认返回 0。
func GetModelQuotaType(modelName string) int {
    for _, p := range GetPricing() {
        if p.ModelName == modelName {
            return p.QuotaType
        }
    }
    return 0
}
