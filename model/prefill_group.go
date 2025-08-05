package model

import (
    "one-api/common"

    "gorm.io/datatypes"
    "gorm.io/gorm"
)

// PrefillGroup 用于存储可复用的“组”信息，例如模型组、标签组、端点组等。
// Name 字段保持唯一，用于在前端下拉框中展示。
// Type 字段用于区分组的类别，可选值如：model、tag、endpoint。
// Items 字段使用 JSON 数组保存对应类型的字符串集合，示例：
// ["gpt-4o", "gpt-3.5-turbo"]
// 设计遵循 3NF，避免冗余，提供灵活扩展能力。

type PrefillGroup struct {
    Id          int            `json:"id"`
    Name        string         `json:"name" gorm:"size:64;not null;uniqueIndex:uk_prefill_name,where:deleted_at IS NULL"`
    Type        string         `json:"type" gorm:"size:32;index;not null"`
    Items       datatypes.JSON `json:"items" gorm:"type:json"`
    Description string         `json:"description,omitempty" gorm:"type:varchar(255)"`
    CreatedTime int64          `json:"created_time" gorm:"bigint"`
    UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Insert 新建组
func (g *PrefillGroup) Insert() error {
    now := common.GetTimestamp()
    g.CreatedTime = now
    g.UpdatedTime = now
    return DB.Create(g).Error
}

// Update 更新组
func (g *PrefillGroup) Update() error {
    g.UpdatedTime = common.GetTimestamp()
    return DB.Save(g).Error
}

// DeleteByID 根据 ID 删除组
func DeletePrefillGroupByID(id int) error {
    return DB.Delete(&PrefillGroup{}, id).Error
}

// GetAllPrefillGroups 获取全部组，可按类型过滤（为空则返回全部）
func GetAllPrefillGroups(groupType string) ([]*PrefillGroup, error) {
    var groups []*PrefillGroup
    query := DB.Model(&PrefillGroup{})
    if groupType != "" {
        query = query.Where("type = ?", groupType)
    }
    if err := query.Order("updated_time DESC").Find(&groups).Error; err != nil {
        return nil, err
    }
    return groups, nil
}
