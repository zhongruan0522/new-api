package model

import (
	"fmt"
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
	lastUpdatedAt := time.Time{}
	for {
		interval := time.Duration(common.DataExportInterval) * time.Minute
		if interval < time.Minute {
			interval = time.Minute
		}
		if common.DataExportEnabled && time.Since(lastUpdatedAt) >= interval {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
			lastUpdatedAt = time.Now()
		}
		time.Sleep(time.Second)
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

const quotaDataAggregateSelect = "model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, sum(fail_count) as fail_count, created_at"

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select(quotaDataAggregateSelect).
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Group("model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select(quotaDataAggregateSelect).
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Group("model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, sum(fail_count) as fail_count").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&quotaDatas).Error
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
	err = DB.Table("quota_data").Select(quotaDataAggregateSelect).Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
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

// RecalculateQuotaData 从 logs 表重新聚合指定时间范围的 quota_data（先删后插）
func RecalculateQuotaData(startTime int64, endTime int64) error {
	// 对齐到小时桶边界，避免部分小时数据不一致
	startTime = startTime - (startTime % 3600)
	endTime = endTime - (endTime % 3600) + 3599

	// 先从日志聚合数据（可失败的操作放在事务外）
	type logRow struct {
		UserId           int
		Username         string
		ModelName        string
		CreatedAt        int64
		PromptTokens     int
		CompletionTokens int
		Quota            int
	}

	// 成功请求（type = 2）
	var successLogs []logRow
	err := LOG_DB.Table("logs").
		Select("user_id, username, model_name, created_at, prompt_tokens, completion_tokens, quota").
		Where("type = 2 and created_at >= ? and created_at <= ?", startTime, endTime).
		Find(&successLogs).Error
	if err != nil {
		return fmt.Errorf("查询成功日志失败: %w", err)
	}

	// 失败请求（type = 5）
	type failRow struct {
		UserId    int
		Username  string
		ModelName string
		CreatedAt int64
	}
	var failLogs []failRow
	err = LOG_DB.Table("logs").
		Select("user_id, username, model_name, created_at").
		Where("type = 5 and created_at >= ? and created_at <= ?", startTime, endTime).
		Find(&failLogs).Error
	if err != nil {
		return fmt.Errorf("查询失败日志失败: %w", err)
	}

	// 按 (userId, username, modelName, hourStart) 聚合
	type aggKey struct {
		UserId    int
		Username  string
		ModelName string
		HourStart int64
	}
	type aggVal struct {
		Count     int
		FailCount int
		Quota     int
		TokenUsed int
	}
	merged := make(map[aggKey]*aggVal)

	// 聚合成功日志
	for _, r := range successLogs {
		hourStart := r.CreatedAt - (r.CreatedAt % 3600)
		key := aggKey{UserId: r.UserId, Username: r.Username, ModelName: r.ModelName, HourStart: hourStart}
		v, ok := merged[key]
		if !ok {
			v = &aggVal{}
			merged[key] = v
		}
		v.Count++
		v.Quota += r.Quota
		v.TokenUsed += r.PromptTokens + r.CompletionTokens
	}

	// 聚合失败日志
	for _, r := range failLogs {
		hourStart := r.CreatedAt - (r.CreatedAt % 3600)
		key := aggKey{UserId: r.UserId, Username: r.Username, ModelName: r.ModelName, HourStart: hourStart}
		v, ok := merged[key]
		if !ok {
			v = &aggVal{}
			merged[key] = v
		}
		v.FailCount++
	}

	// 构建批量插入数据
	batch := make([]*QuotaData, 0, len(merged))
	for k, v := range merged {
		batch = append(batch, &QuotaData{
			UserID:    k.UserId,
			Username:  k.Username,
			ModelName: k.ModelName,
			CreatedAt: k.HourStart,
			Count:     v.Count,
			FailCount: v.FailCount,
			Quota:     v.Quota,
			TokenUsed: v.TokenUsed,
		})
	}

	// 事务保护：先删后插
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Delete(nil).Error; err != nil {
			return fmt.Errorf("删除旧 quota_data 失败: %w", err)
		}
		if len(batch) > 0 {
			if err := tx.Table("quota_data").CreateInBatches(batch, 100).Error; err != nil {
				return fmt.Errorf("插入 quota_data 失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	common.SysLog(fmt.Sprintf("重新计算数据看板完成，时间范围 %d~%d，共 %d 条记录", startTime, endTime, len(batch)))
	return nil
}
