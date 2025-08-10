package controller

import (
	"encoding/json"
	"strconv"
	"strings"

	"one-api/common"
	"one-api/constant"
	"one-api/model"

	"github.com/gin-gonic/gin"
)

// GetAllModelsMeta 获取模型列表（分页）
func GetAllModelsMeta(c *gin.Context) {

	pageInfo := common.GetPageQuery(c)
	modelsMeta, err := model.GetAllModels(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 填充附加字段
	for _, m := range modelsMeta {
		fillModelExtra(m)
	}
	var total int64
	model.DB.Model(&model.Model{}).Count(&total)

	// 统计供应商计数（全部数据，不受分页影响）
	vendorCounts, _ := model.GetVendorModelCounts()

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(modelsMeta)
	common.ApiSuccess(c, gin.H{
		"items":         modelsMeta,
		"total":         total,
		"page":          pageInfo.GetPage(),
		"page_size":     pageInfo.GetPageSize(),
		"vendor_counts": vendorCounts,
	})
}

// SearchModelsMeta 搜索模型列表
func SearchModelsMeta(c *gin.Context) {

	keyword := c.Query("keyword")
	vendor := c.Query("vendor")
	pageInfo := common.GetPageQuery(c)

	modelsMeta, total, err := model.SearchModels(keyword, vendor, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, m := range modelsMeta {
		fillModelExtra(m)
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(modelsMeta)
	common.ApiSuccess(c, pageInfo)
}

// GetModelMeta 根据 ID 获取单条模型信息
func GetModelMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var m model.Model
	if err := model.DB.First(&m, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	fillModelExtra(&m)
	common.ApiSuccess(c, &m)
}

// CreateModelMeta 新建模型
func CreateModelMeta(c *gin.Context) {
	var m model.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	if m.ModelName == "" {
		common.ApiErrorMsg(c, "模型名称不能为空")
		return
	}
	// 名称冲突检查
	if dup, err := model.IsModelNameDuplicated(0, m.ModelName); err != nil {
		common.ApiError(c, err)
		return
	} else if dup {
		common.ApiErrorMsg(c, "模型名称已存在")
		return
	}

	if err := m.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, &m)
}

// UpdateModelMeta 更新模型
func UpdateModelMeta(c *gin.Context) {
	statusOnly := c.Query("status_only") == "true"

	var m model.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	if m.Id == 0 {
		common.ApiErrorMsg(c, "缺少模型 ID")
		return
	}

	if statusOnly {
		// 只更新状态，防止误清空其他字段
		if err := model.DB.Model(&model.Model{}).Where("id = ?", m.Id).Update("status", m.Status).Error; err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		// 名称冲突检查
		if dup, err := model.IsModelNameDuplicated(m.Id, m.ModelName); err != nil {
			common.ApiError(c, err)
			return
		} else if dup {
			common.ApiErrorMsg(c, "模型名称已存在")
			return
		}

		if err := m.Update(); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	model.RefreshPricing()
	common.ApiSuccess(c, &m)
}

// DeleteModelMeta 删除模型
func DeleteModelMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Delete(&model.Model{}, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, nil)
}

// 辅助函数：填充 Endpoints 和 BoundChannels 和 EnableGroups
func fillModelExtra(m *model.Model) {
	// 若为精确匹配，保持原有逻辑
	if m.NameRule == model.NameRuleExact {
		if m.Endpoints == "" {
			eps := model.GetModelSupportEndpointTypes(m.ModelName)
			if b, err := json.Marshal(eps); err == nil {
				m.Endpoints = string(b)
			}
		}
		if channels, err := model.GetBoundChannels(m.ModelName); err == nil {
			m.BoundChannels = channels
		}
		m.EnableGroups = model.GetModelEnableGroups(m.ModelName)
		m.QuotaType = model.GetModelQuotaType(m.ModelName)
		return
	}

	// 非精确匹配：计算并集
	pricings := model.GetPricing()

	// 匹配到的模型名称集合
	matchedNames := make([]string, 0)

	// 端点去重集合
	endpointSet := make(map[constant.EndpointType]struct{})

	// 已绑定渠道去重集合
	channelSet := make(map[string]model.BoundChannel)
	// 分组去重集合
	groupSet := make(map[string]struct{})
	// 计费类型（若有任意模型为 1，则返回 1）
	quotaTypeSet := make(map[int]struct{})

	for _, p := range pricings {
		var matched bool
		switch m.NameRule {
		case model.NameRulePrefix:
			matched = strings.HasPrefix(p.ModelName, m.ModelName)
		case model.NameRuleSuffix:
			matched = strings.HasSuffix(p.ModelName, m.ModelName)
		case model.NameRuleContains:
			matched = strings.Contains(p.ModelName, m.ModelName)
		}
		if !matched {
			continue
		}

		// 记录匹配到的模型名称
		matchedNames = append(matchedNames, p.ModelName)

		// 收集端点
		for _, et := range p.SupportedEndpointTypes {
			endpointSet[et] = struct{}{}
		}

		// 收集分组
		for _, g := range p.EnableGroup {
			groupSet[g] = struct{}{}
		}

		// 收集计费类型
		quotaTypeSet[p.QuotaType] = struct{}{}
	}

	// 序列化端点
	if len(endpointSet) > 0 && m.Endpoints == "" {
		eps := make([]constant.EndpointType, 0, len(endpointSet))
		for et := range endpointSet {
			eps = append(eps, et)
		}
		if b, err := json.Marshal(eps); err == nil {
			m.Endpoints = string(b)
		}
	}

	// 序列化分组
	if len(groupSet) > 0 {
		groups := make([]string, 0, len(groupSet))
		for g := range groupSet {
			groups = append(groups, g)
		}
		m.EnableGroups = groups
	}

	// 确定计费类型：仅当所有匹配模型计费类型一致时才返回该类型，否则返回 -1 表示未知/不确定
	if len(quotaTypeSet) == 1 {
		for k := range quotaTypeSet {
			m.QuotaType = k
		}
	} else {
		m.QuotaType = -1
	}

	// 批量查询并序列化渠道
	if len(matchedNames) > 0 {
		if channels, err := model.GetBoundChannelsForModels(matchedNames); err == nil {
			for _, ch := range channels {
				key := ch.Name + "_" + strconv.Itoa(ch.Type)
				channelSet[key] = ch
			}
		}
		if len(channelSet) > 0 {
			chs := make([]model.BoundChannel, 0, len(channelSet))
			for _, ch := range channelSet {
				chs = append(chs, ch)
			}
			m.BoundChannels = chs
		}
	}

	// 设置匹配信息
	m.MatchedModels = matchedNames
	m.MatchedCount = len(matchedNames)
}
