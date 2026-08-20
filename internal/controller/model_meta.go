package controller

import (
	"encoding/json"
	"github.com/NookMux/NookMux/internal/common"
	audit "github.com/NookMux/NookMux/internal/domain/audit"
	"github.com/NookMux/NookMux/internal/domain/channel/constant"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/NookMux/NookMux/internal/store/vendor_meta"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
	"strings"
)

// GetAllModelsMeta 获取模型列表（分页）
func GetAllModelsMeta(c *gin.Context) {

	pageInfo := common.GetPageQuery(c)
	modelsMeta, err := vendormetastore.GetAllModels(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("failed to get all models: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	// 批量填充附加字段，提升列表接口性能
	enrichModels(modelsMeta)
	var total int64
	dbstore.DB.Model(&vendormetastore.Model{}).Count(&total)

	// 统计供应商计数（全部数据，不受分页影响）
	vendorCounts, _ := vendormetastore.GetVendorModelCounts()

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

	modelsMeta, total, err := vendormetastore.SearchModels(keyword, vendor, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.SysError("failed to search models: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	// 批量填充附加字段，提升列表接口性能
	enrichModels(modelsMeta)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(modelsMeta)
	common.ApiSuccess(c, pageInfo)
}

// GetModelMeta 根据 ID 获取单条模型信息
func GetModelMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var m vendormetastore.Model
	if err := dbstore.DB.First(&m, id).Error; err != nil {
		common.SysError("failed to get model by id: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	enrichModels([]*vendormetastore.Model{&m})
	common.ApiSuccess(c, &m)
}

// CreateModelMeta 新建模型
func CreateModelMeta(c *gin.Context) {
	var m vendormetastore.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if m.ModelName == "" {
		common.ApiErrorI18n(c, i18n.MsgModelMetaNameRequired)
		return
	}
	// 名称冲突检查
	if dup, err := vendormetastore.IsModelNameDuplicated(0, m.ModelName); err != nil {
		common.SysError("failed to check model name duplication: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	} else if dup {
		common.ApiErrorI18n(c, i18n.MsgModelMetaNameExists)
		return
	}

	if err := m.Insert(); err != nil {
		common.SysError("failed to insert model: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	pricingstore.RefreshPricing()
	audit.RecordAudit(c, auditstore.AuditModuleModel, auditstore.AuditActionCreate, "新增模型: "+m.ModelName, nil, m)
	common.ApiSuccess(c, &m)
}

// UpdateModelMeta 更新模型
func UpdateModelMeta(c *gin.Context) {
	statusOnly := c.Query("status_only") == "true"

	var m vendormetastore.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if m.Id == 0 {
		common.ApiErrorI18n(c, i18n.MsgModelMetaMissingID)
		return
	}

	// 查询更新前的原始数据用于审计差异对比
	var origin vendormetastore.Model
	if err := dbstore.DB.First(&origin, "id = ?", m.Id).Error; err != nil {
		common.SysError("failed to get model origin: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	if statusOnly {
		// 只更新状态，防止误清空其他字段
		if err := dbstore.DB.Model(&vendormetastore.Model{}).Where("id = ?", m.Id).Update("status", m.Status).Error; err != nil {
			common.SysError("failed to update model status: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			return
		}
		// after 使用 origin 副本+新 status，避免请求体零值字段产生噪声 diff
		afterModel := origin
		afterModel.Status = m.Status
		audit.RecordAudit(c, auditstore.AuditModuleModel, auditstore.AuditActionUpdate, "修改模型: "+origin.ModelName, origin, afterModel)
	} else {
		// 名称冲突检查
		if dup, err := vendormetastore.IsModelNameDuplicated(m.Id, m.ModelName); err != nil {
			common.SysError("failed to check model name duplication: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			return
		} else if dup {
			common.ApiErrorI18n(c, i18n.MsgModelMetaNameExists)
			return
		}

		if err := m.Update(); err != nil {
			common.SysError("failed to update model: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgDatabaseError)
			return
		}
		audit.RecordAudit(c, auditstore.AuditModuleModel, auditstore.AuditActionUpdate, "修改模型: "+m.ModelName, origin, m)
	}
	pricingstore.RefreshPricing()
	common.ApiSuccess(c, &m)
}

// DeleteModelMeta 删除模型
func DeleteModelMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := dbstore.DB.Delete(&vendormetastore.Model{}, id).Error; err != nil {
		common.SysError("failed to delete model: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	pricingstore.RefreshPricing()
	audit.RecordAudit(c, auditstore.AuditModuleModel, auditstore.AuditActionDelete, "删除模型 #"+strconv.Itoa(id), nil, map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

// enrichModels 批量填充附加信息：端点、渠道、分组、计费类型，避免 N+1 查询
func enrichModels(models []*vendormetastore.Model) {
	if len(models) == 0 {
		return
	}

	// 1) 拆分精确与规则匹配
	exactNames := make([]string, 0)
	exactIdx := make(map[string][]int) // modelName -> indices in models
	ruleIndices := make([]int, 0)
	for i, m := range models {
		if m == nil {
			continue
		}
		if m.NameRule == vendormetastore.NameRuleExact {
			exactNames = append(exactNames, m.ModelName)
			exactIdx[m.ModelName] = append(exactIdx[m.ModelName], i)
		} else {
			ruleIndices = append(ruleIndices, i)
		}
	}

	// 2) 批量查询精确模型的绑定渠道
	channelsByModel, _ := vendormetastore.GetBoundChannelsByModelsMap(exactNames)

	// 3) 精确模型：端点从缓存、渠道批量映射、分组/计费类型从缓存
	for name, indices := range exactIdx {
		chs := channelsByModel[name]
		for _, idx := range indices {
			mm := models[idx]
			if mm.Endpoints == "" {
				eps := pricingstore.GetModelSupportEndpointTypes(mm.ModelName)
				if b, err := json.Marshal(eps); err == nil {
					mm.Endpoints = string(b)
				}
			}
			mm.BoundChannels = chs
			mm.EnableGroups = pricingstore.GetModelEnableGroups(mm.ModelName)
			mm.QuotaTypes = pricingstore.GetModelQuotaTypes(mm.ModelName)
		}
	}

	if len(ruleIndices) == 0 {
		return
	}

	// 4) 一次性读取定价缓存，内存匹配所有规则模型
	pricings := pricingstore.GetPricing()

	// 为全部规则模型收集匹配名集合、端点并集、分组并集、配额集合
	matchedNamesByIdx := make(map[int][]string)
	endpointSetByIdx := make(map[int]map[constant.EndpointType]struct{})
	groupSetByIdx := make(map[int]map[string]struct{})
	quotaSetByIdx := make(map[int]map[int]struct{})

	for _, p := range pricings {
		for _, idx := range ruleIndices {
			mm := models[idx]
			var matched bool
			switch mm.NameRule {
			case vendormetastore.NameRulePrefix:
				matched = strings.HasPrefix(p.ModelName, mm.ModelName)
			case vendormetastore.NameRuleSuffix:
				matched = strings.HasSuffix(p.ModelName, mm.ModelName)
			case vendormetastore.NameRuleContains:
				matched = strings.Contains(p.ModelName, mm.ModelName)
			}
			if !matched {
				continue
			}
			matchedNamesByIdx[idx] = append(matchedNamesByIdx[idx], p.ModelName)

			es := endpointSetByIdx[idx]
			if es == nil {
				es = make(map[constant.EndpointType]struct{})
				endpointSetByIdx[idx] = es
			}
			for _, et := range p.SupportedEndpointTypes {
				es[et] = struct{}{}
			}

			gs := groupSetByIdx[idx]
			if gs == nil {
				gs = make(map[string]struct{})
				groupSetByIdx[idx] = gs
			}
			for _, g := range p.EnableGroup {
				gs[g] = struct{}{}
			}

			qs := quotaSetByIdx[idx]
			if qs == nil {
				qs = make(map[int]struct{})
				quotaSetByIdx[idx] = qs
			}
			qs[p.QuotaType] = struct{}{}
		}
	}

	// 5) 汇总所有匹配到的模型名称，批量查询一次渠道
	allMatchedSet := make(map[string]struct{})
	for _, names := range matchedNamesByIdx {
		for _, n := range names {
			allMatchedSet[n] = struct{}{}
		}
	}
	allMatched := make([]string, 0, len(allMatchedSet))
	for n := range allMatchedSet {
		allMatched = append(allMatched, n)
	}
	matchedChannelsByModel, _ := vendormetastore.GetBoundChannelsByModelsMap(allMatched)

	// 6) 回填每个规则模型的并集信息
	for _, idx := range ruleIndices {
		mm := models[idx]

		// 端点并集 -> 序列化
		if es, ok := endpointSetByIdx[idx]; ok && mm.Endpoints == "" {
			eps := make([]constant.EndpointType, 0, len(es))
			for et := range es {
				eps = append(eps, et)
			}
			if b, err := json.Marshal(eps); err == nil {
				mm.Endpoints = string(b)
			}
		}

		// 分组并集
		if gs, ok := groupSetByIdx[idx]; ok {
			groups := make([]string, 0, len(gs))
			for g := range gs {
				groups = append(groups, g)
			}
			mm.EnableGroups = groups
		}

		// 配额类型集合（保持去重并排序）
		if qs, ok := quotaSetByIdx[idx]; ok {
			arr := make([]int, 0, len(qs))
			for k := range qs {
				arr = append(arr, k)
			}
			sort.Ints(arr)
			mm.QuotaTypes = arr
		}

		// 渠道并集
		names := matchedNamesByIdx[idx]
		channelSet := make(map[string]vendormetastore.BoundChannel)
		for _, n := range names {
			for _, ch := range matchedChannelsByModel[n] {
				key := ch.Name + "_" + strconv.Itoa(ch.Type)
				channelSet[key] = ch
			}
		}
		if len(channelSet) > 0 {
			chs := make([]vendormetastore.BoundChannel, 0, len(channelSet))
			for _, ch := range channelSet {
				chs = append(chs, ch)
			}
			mm.BoundChannels = chs
		}

		// 匹配信息
		mm.MatchedModels = names
		mm.MatchedCount = len(names)
	}
}
