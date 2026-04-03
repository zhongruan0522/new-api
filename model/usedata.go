package model

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
	FailCount int    `json:"fail_count" gorm:"default:0"`
	Quota     int    `json:"quota" gorm:"default:0"`
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(userId int, username string, modelName string, quota int, createdAt int64, tokenUsed int) {
	key := fmt.Sprintf("%d-%s-%s-%d", userId, username, modelName, createdAt)
	quotaData, ok := CacheQuotaData[key]
	if ok {
		quotaData.Count += 1
		quotaData.Quota += quota
		quotaData.TokenUsed += tokenUsed
	} else {
		quotaData = &QuotaData{
			UserID:    userId,
			Username:  username,
			ModelName: modelName,
			CreatedAt: createdAt,
			Count:     1,
			Quota:     quota,
			TokenUsed: tokenUsed,
		}
	}
	CacheQuotaData[key] = quotaData
}

// LogQuotaData 记录成功请求数据到内存缓存
func LogQuotaData(userId int, username string, modelName string, quota int, createdAt int64, tokenUsed int) {
	createdAt = createdAt - (createdAt % 3600)

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(userId, username, modelName, quota, createdAt, tokenUsed)
}

// LogQuotaErrorData 记录失败请求数据到内存缓存
func LogQuotaErrorData(userId int, username string, modelName string, createdAt int64) {
	createdAt = createdAt - (createdAt % 3600)

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	key := fmt.Sprintf("%d-%s-%s-%d", userId, username, modelName, createdAt)
	quotaData, ok := CacheQuotaData[key]
	if ok {
		quotaData.FailCount += 1
	} else {
		quotaData = &QuotaData{
			UserID:    userId,
			Username:  username,
			ModelName: modelName,
			CreatedAt: createdAt,
			Count:     0,
			Quota:     0,
			TokenUsed: 0,
			FailCount: 1,
		}
	}
	CacheQuotaData[key] = quotaData
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt).First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			increaseQuotaData(quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.Count, quotaData.Quota, quotaData.CreatedAt, quotaData.TokenUsed, quotaData.FailCount)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(userId int, username string, modelName string, count int, quota int, createdAt int64, tokenUsed int, failCount int) {
	err := DB.Table("quota_data").Where("user_id = ? and username = ? and model_name = ? and created_at = ?",
		userId, username, modelName, createdAt).Updates(map[string]interface{}{
		"count":      gorm.Expr("count + ?", count),
		"quota":      gorm.Expr("quota + ?", quota),
		"token_used": gorm.Expr("token_used + ?", tokenUsed),
		"fail_count": gorm.Expr("fail_count + ?", failCount),
	}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
	//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
	err = DB.Table("quota_data").Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, sum(fail_count) as fail_count, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
	return quotaDatas, err
}

// QuotaStat 预聚合的统计数据
type QuotaStat struct {
	Quota        int `json:"quota"`
	Tpm          int `json:"tpm"`
	SuccessCount int `json:"success_count"`
	FailCount    int `json:"fail_count"`
}

// GetQuotaStatByUserId 从 quota_data 表查询指定用户的预聚合成功/失败次数
func GetQuotaStatByUserId(userId int, startTime int64, endTime int64) (QuotaStat, error) {
	var stat QuotaStat
	err := DB.Table("quota_data").Select("coalesce(sum(quota), 0) as quota, coalesce(sum(token_used), 0) as tpm, coalesce(sum(count), 0) as success_count, coalesce(sum(fail_count), 0) as fail_count").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Scan(&stat).Error
	return stat, err
}

// GetQuotaStatByUsername 从 quota_data 表查询指定用户的预聚合成功/失败次数
func GetQuotaStatByUsername(username string, startTime int64, endTime int64) (QuotaStat, error) {
	var stat QuotaStat
	err := DB.Table("quota_data").Select("coalesce(sum(quota), 0) as quota, coalesce(sum(token_used), 0) as tpm, coalesce(sum(count), 0) as success_count, coalesce(sum(fail_count), 0) as fail_count").
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Scan(&stat).Error
	return stat, err
}

// GetAllQuotaStat 从 quota_data 表查询所有用户的预聚合成功/失败次数
func GetAllQuotaStat(startTime int64, endTime int64) (QuotaStat, error) {
	var stat QuotaStat
	err := DB.Table("quota_data").Select("coalesce(sum(quota), 0) as quota, coalesce(sum(token_used), 0) as tpm, coalesce(sum(count), 0) as success_count, coalesce(sum(fail_count), 0) as fail_count").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Scan(&stat).Error
	return stat, err
}

// ModelRankItem 模型调用排行单项
type ModelRankItem struct {
	ModelName    string  `json:"model_name"`
	SuccessCount int     `json:"success_count"`
	FailCount    int     `json:"fail_count"`
	SuccessRate  float64 `json:"success_rate"`
}

// ModelRankResponse 模型调用排行响应（全部/国内/海外三组排行）
type ModelRankResponse struct {
	All      []ModelRankItem `json:"all"`
	Domestic []ModelRankItem `json:"domestic"`
	Overseas []ModelRankItem `json:"overseas"`
}

// computeModelRank 从 quota_data 列表计算按模型分组的排行数据
func computeModelRank(quotaDatas []*QuotaData) ModelRankResponse {
	modelMap := make(map[string]*ModelRankItem)
	for _, item := range quotaDatas {
		rank, ok := modelMap[item.ModelName]
		if !ok {
			rank = &ModelRankItem{ModelName: item.ModelName}
			modelMap[item.ModelName] = rank
		}
		rank.SuccessCount += item.Count
		rank.FailCount += item.FailCount
	}

	// 计算成功率并分类
	var allList, domesticList, overseasList []ModelRankItem
	for _, rank := range modelMap {
		total := rank.SuccessCount + rank.FailCount
		if total > 0 {
			rank.SuccessRate = math.Floor(float64(rank.SuccessCount)/float64(total)*1000) / 10
		}
		allList = append(allList, *rank)
		if matchModelPrefix(rank.ModelName, domesticPrefixes) {
			domesticList = append(domesticList, *rank)
		} else if matchModelPrefix(rank.ModelName, overseasPrefixes) {
			overseasList = append(overseasList, *rank)
		}
	}

	sortModelRank := func(list []ModelRankItem) {
		sort.Slice(list, func(i, j int) bool {
			return list[i].SuccessCount+list[i].FailCount > list[j].SuccessCount+list[j].FailCount
		})
	}
	sortModelRank(allList)
	sortModelRank(domesticList)
	sortModelRank(overseasList)

	return ModelRankResponse{
		All:      allList,
		Domestic: domesticList,
		Overseas: overseasList,
	}
}

// GetModelRankByUserId 查询指定用户的模型调用排行（全部/国内/海外）
func GetModelRankByUserId(userId int, startTime int64, endTime int64) (ModelRankResponse, error) {
	var quotaDatas []*QuotaData
	err := DB.Table("quota_data").
		Select("model_name, sum(count) as count, sum(fail_count) as fail_count").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Group("model_name").
		Find(&quotaDatas).Error
	if err != nil {
		return ModelRankResponse{}, err
	}
	return computeModelRank(quotaDatas), nil
}

// GetModelRankByUsername 查询指定用户名的模型调用排行（全部/国内/海外）
func GetModelRankByUsername(username string, startTime int64, endTime int64) (ModelRankResponse, error) {
	var quotaDatas []*QuotaData
	err := DB.Table("quota_data").
		Select("model_name, sum(count) as count, sum(fail_count) as fail_count").
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Group("model_name").
		Find(&quotaDatas).Error
	if err != nil {
		return ModelRankResponse{}, err
	}
	return computeModelRank(quotaDatas), nil
}

// GetAllModelRank 查询所有用户的模型调用排行（全部/国内/海外）
func GetAllModelRank(startTime int64, endTime int64) (ModelRankResponse, error) {
	var quotaDatas []*QuotaData
	err := DB.Table("quota_data").
		Select("model_name, sum(count) as count, sum(fail_count) as fail_count").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("model_name").
		Find(&quotaDatas).Error
	if err != nil {
		return ModelRankResponse{}, err
	}
	return computeModelRank(quotaDatas), nil
}

// RegionStat 区域统计数据（国内/海外）
type RegionStat struct {
	SuccessCount int     `json:"success_count"`
	FailCount    int     `json:"fail_count"`
	SuccessRate  float64 `json:"success_rate"`
}

// regionModelPrefixes 国内模型名前缀（统一小写）
var domesticPrefixes = []string{"glm", "minimax", "qwen", "kimi"}

// overseasPrefixes 海外模型名前缀（统一小写）
var overseasPrefixes = []string{"claude", "gemini", "gpt"}

// matchModelPrefix 不区分大小写匹配模型名前缀
func matchModelPrefix(modelName string, prefixes []string) bool {
	lower := strings.ToLower(modelName)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// computeRegionStats 从 quota_data 列表计算国内/海外统计
func computeRegionStats(quotaDatas []*QuotaData) (domestic RegionStat, overseas RegionStat) {
	for _, item := range quotaDatas {
		if matchModelPrefix(item.ModelName, domesticPrefixes) {
			domestic.SuccessCount += item.Count
			domestic.FailCount += item.FailCount
		} else if matchModelPrefix(item.ModelName, overseasPrefixes) {
			overseas.SuccessCount += item.Count
			overseas.FailCount += item.FailCount
		}
	}
	total := domestic.SuccessCount + domestic.FailCount
	if total > 0 {
		domestic.SuccessRate = math.Floor(float64(domestic.SuccessCount)/float64(total)*1000) / 10
	}
	total = overseas.SuccessCount + overseas.FailCount
	if total > 0 {
		overseas.SuccessRate = math.Floor(float64(overseas.SuccessCount)/float64(total)*1000) / 10
	}
	return
}

// RegionStatsResponse 区域统计响应
type RegionStatsResponse struct {
	Domestic RegionStat `json:"domestic"`
	Overseas RegionStat `json:"overseas"`
}

// GetRegionStatsByUserId 查询指定用户的国内/海外模型成功率
func GetRegionStatsByUserId(userId int, startTime int64, endTime int64) (RegionStatsResponse, error) {
	var quotaDatas []*QuotaData
	err := DB.Table("quota_data").
		Select("model_name, sum(count) as count, sum(fail_count) as fail_count").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Group("model_name").
		Find(&quotaDatas).Error
	if err != nil {
		return RegionStatsResponse{}, err
	}
	domestic, overseas := computeRegionStats(quotaDatas)
	return RegionStatsResponse{Domestic: domestic, Overseas: overseas}, nil
}

// GetRegionStatsByUsername 查询指定用户名的国内/海外模型成功率
func GetRegionStatsByUsername(username string, startTime int64, endTime int64) (RegionStatsResponse, error) {
	var quotaDatas []*QuotaData
	err := DB.Table("quota_data").
		Select("model_name, sum(count) as count, sum(fail_count) as fail_count").
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Group("model_name").
		Find(&quotaDatas).Error
	if err != nil {
		return RegionStatsResponse{}, err
	}
	domestic, overseas := computeRegionStats(quotaDatas)
	return RegionStatsResponse{Domestic: domestic, Overseas: overseas}, nil
}

// GetAllRegionStats 查询所有用户的国内/海外模型成功率
func GetAllRegionStats(startTime int64, endTime int64) (RegionStatsResponse, error) {
	var quotaDatas []*QuotaData
	err := DB.Table("quota_data").
		Select("model_name, sum(count) as count, sum(fail_count) as fail_count").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("model_name").
		Find(&quotaDatas).Error
	if err != nil {
		return RegionStatsResponse{}, err
	}
	domestic, overseas := computeRegionStats(quotaDatas)
	return RegionStatsResponse{Domestic: domestic, Overseas: overseas}, nil
}
