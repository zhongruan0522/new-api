package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/logger"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type Log struct {
	Id               int    `json:"id" gorm:"index:idx_created_at_id,priority:1;index:idx_user_id_id,priority:2"`
	UserId           int    `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:2;index:idx_created_at_type"`
	Type             int    `json:"type" gorm:"index:idx_created_at_type"`
	Content          string `json:"content"`
	Username         string `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName        string `json:"token_name" gorm:"index;default:''"`
	ModelName        string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota            int    `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	UseTime          int    `json:"use_time" gorm:"default:0"`
	IsStream         bool   `json:"is_stream"`
	ChannelId        int    `json:"channel" gorm:"index"`
	ChannelName      string `json:"channel_name" gorm:"->"`
	TokenId          int    `json:"token_id" gorm:"default:0;index"`
	Group            string `json:"group" gorm:"index"`
	Ip               string `json:"ip" gorm:"index;default:''"`
	RequestId        string `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	Other            string `json:"other"`
}

// don't use iota, avoid change log type value
const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
	LogTypeError   = 5
	LogTypeRefund  = 6
)

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap != nil {
			// Remove admin-only debug fields.
			delete(otherMap, "admin_info")
			delete(otherMap, "reject_reason")
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(common.MaxRecentItems).Find(&logs).Error
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeMs int,
	isStream bool, group string, other map[string]interface{}) {
	logger.LogInfo(c, fmt.Sprintf("record error log: userId=%d, channelId=%d, modelName=%s, tokenName=%s, content=%s", userId, channelId, modelName, tokenName, content))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	otherStr := common.MapToJsonStr(other)
	// 记录请求与错误日志的 IP（强制开启，用于滥用追踪）
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeError,
		Content:          content,
		PromptTokens:     0,
		CompletionTokens: 0,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            0,
		ChannelId:        channelId,
		TokenId:          tokenId,
		UseTime:          useTimeMs,
		IsStream:         isStream,
		Group:            group,
		Ip:               c.ClientIP(),
		RequestId:        requestId,
		Other:            otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
	if common.DataExportEnabled {
		gopool.Go(func() {
			LogQuotaErrorData(userId, username, modelName, common.GetTimestamp())
		})
	}
}

type RecordConsumeLogParams struct {
	ChannelId        int                    `json:"channel_id"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	ModelName        string                 `json:"model_name"`
	TokenName        string                 `json:"token_name"`
	Quota            int                    `json:"quota"`
	Content          string                 `json:"content"`
	TokenId          int                    `json:"token_id"`
	UseTimeMs        int                    `json:"use_time_ms"`
	IsStream         bool                   `json:"is_stream"`
	Group            string                 `json:"group"`
	Other            map[string]interface{} `json:"other"`
	InputTokens      int                    `json:"input_tokens"` // 原始输入 token（未被计费逻辑修改），用于缓存率统计
}

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	logger.LogInfo(c, fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.GetJsonString(params)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	createdAt := common.GetTimestamp()
	clientIP := c.ClientIP()
	other := params.Other
	if other == nil {
		other = make(map[string]interface{})
	}
	appendConsumeLogClientHeaders(c, other)
	otherStr := common.MapToJsonStr(other)
	// 记录请求与错误日志的 IP（强制开启，用于滥用追踪）
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        createdAt,
		Type:             LogTypeConsume,
		Content:          params.Content,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		TokenName:        params.TokenName,
		ModelName:        params.ModelName,
		Quota:            params.Quota,
		ChannelId:        params.ChannelId,
		TokenId:          params.TokenId,
		UseTime:          params.UseTimeMs,
		IsStream:         params.IsStream,
		Group:            params.Group,
		Ip:               clientIP,
		RequestId:        requestId,
		Other:            otherStr,
	}
	// 消费日志不影响主流程，异步写入以避免高并发下在请求尾部阻塞数据库。
	gopool.Go(func() {
		err := LOG_DB.Create(log).Error
		if err != nil {
			common.SysError(fmt.Sprintf("failed to record consume log (request_id=%s): %s", requestId, err.Error()))
		}
		if common.DataExportEnabled {
			cacheHitTokens := 0
			cacheCreationTokens := 0
			if other != nil {
				if v, ok := other["cache_tokens"].(float64); ok {
					cacheHitTokens = int(v)
				} else if v, ok := other["cache_tokens"].(int); ok {
					cacheHitTokens = v
				}
				if v, ok := other["cache_creation_tokens"].(float64); ok {
					cacheCreationTokens = int(v)
				} else if v, ok := other["cache_creation_tokens"].(int); ok {
					cacheCreationTokens = v
				}
			}
			inputTokens := params.InputTokens
			if inputTokens == 0 {
				inputTokens = params.PromptTokens
			}
			LogQuotaData(userId, username, params.ModelName, params.Quota, createdAt, params.PromptTokens+params.CompletionTokens, inputTokens, cacheHitTokens, cacheCreationTokens)
		}
	})
}

func appendConsumeLogClientHeaders(c *gin.Context, other map[string]interface{}) {
	if c == nil || other == nil {
		return
	}

	if _, exists := other["http_referer"]; !exists {
		// Prefer the OpenAI-compatible `HTTP-Referer` header, fall back to standard `Referer`.
		httpReferer := c.GetHeader("HTTP-Referer")
		if strings.TrimSpace(httpReferer) == "" {
			httpReferer = c.GetHeader("Referer")
		}
		other["http_referer"] = sanitizeConsumeLogHeaderValue(httpReferer)
	}

	if _, exists := other["x_title"]; !exists {
		other["x_title"] = sanitizeConsumeLogHeaderValue(c.GetHeader("X-Title"))
	}

	if _, exists := other["ua"]; !exists {
		ua := c.GetHeader("User-Agent")
		if strings.TrimSpace(ua) == "" && c.Request != nil {
			ua = c.Request.UserAgent()
		}
		other["ua"] = sanitizeConsumeLogHeaderValue(ua)
	}
}

func sanitizeConsumeLogHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	// Keep DB logs single-line to avoid rendering/log injection issues in the UI.
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("logs.type = ?", logType)
	}

	if modelName != "" {
		tx = tx.Where("logs.model_name like ?", modelName)
	}
	if username != "" {
		tx = tx.Where("logs.username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	channelIds := types.NewSet[int]()
	for _, log := range logs {
		if log.ChannelId != 0 {
			channelIds.Add(log.ChannelId)
		}
	}

	if channelIds.Len() > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err = DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
			return logs, total, err
		}
		channelMap := make(map[int]string, len(channels))
		for _, channel := range channels {
			channelMap[channel.Id] = channel.Name
		}
		for i := range logs {
			logs[i].ChannelName = channelMap[logs[i].ChannelId]
		}
	}

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("logs.user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("logs.user_id = ? and logs.type = ?", userId, logType)
	}

	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return nil, 0, err
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	formatUserLogs(logs, startIdx)
	return logs, total, err
}

type Stat struct {
	Quota        int `json:"quota"`
	Rpm          int `json:"rpm"`
	Tpm          int `json:"tpm"`
	SuccessCount int `json:"success_count"`
	FailCount    int `json:"fail_count"`
}

// buildStatConditions 构建统计查询的通用 WHERE 条件
func buildStatConditions(tx *gorm.DB, username string, tokenName string, startTimestamp int64, endTimestamp int64, modelName string, channel int, group string) (*gorm.DB, error) {
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return nil, err
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
	}
	return tx, nil
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	// 额度统计查询
	tx := LOG_DB.Table("logs").Select("sum(quota) quota")
	tx, err = buildStatConditions(tx, username, tokenName, startTimestamp, endTimestamp, modelName, channel, group)
	if err != nil {
		return stat, err
	}
	if logType != LogTypeUnknown {
		tx = tx.Where("type = ?", logType)
	} else {
		tx = tx.Where("type = ?", LogTypeConsume)
	}

	// rpm和tpm查询（最近60秒）
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm")
	rpmTpmQuery, err = buildStatConditions(rpmTpmQuery, username, tokenName, 0, 0, modelName, channel, group)
	if err != nil {
		return stat, err
	}
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	// 成功次数查询
	successQuery := LOG_DB.Table("logs").Select("count(*) success_count")
	successQuery, err = buildStatConditions(successQuery, username, tokenName, startTimestamp, endTimestamp, modelName, channel, group)
	if err != nil {
		return stat, err
	}
	if logType != LogTypeUnknown {
		successQuery = successQuery.Where("type = ?", logType)
	} else {
		successQuery = successQuery.Where("type = ?", LogTypeConsume)
	}

	// 失败次数查询
	failQuery := LOG_DB.Table("logs").Select("count(*) fail_count")
	failQuery, err = buildStatConditions(failQuery, username, tokenName, startTimestamp, endTimestamp, modelName, channel, group)
	if err != nil {
		return stat, err
	}
	failQuery = failQuery.Where("type = ?", LogTypeError)

	// 执行查询
	if err := tx.Scan(&stat).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := rpmTpmQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := successQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query success count stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := failQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query fail count stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}

	return stat, nil
}

// QueryRpmTpm 实时查询最近60秒的 RPM 和 TPM，供 DataExport 模式复用
func QueryRpmTpm(username string, tokenName string, modelName string, channel int, group string) (rpm int, tpm int, err error) {
	q := LOG_DB.Table("logs").Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm")
	q, buildErr := buildStatConditions(q, username, tokenName, 0, 0, modelName, channel, group)
	if buildErr != nil {
		return 0, 0, buildErr
	}
	q = q.Where("type = ?", LogTypeConsume)
	q = q.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	var result struct {
		Rpm int `json:"rpm"`
		Tpm int `json:"tpm"`
	}
	if err := q.Scan(&result).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return 0, 0, errors.New("查询RPM/TPM统计数据失败")
	}
	return result.Rpm, result.Tpm, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func DeleteOldLog(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0

	for {
		if nil != ctx.Err() {
			return total, ctx.Err()
		}

		result := LOG_DB.Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&Log{})
		if nil != result.Error {
			return total, result.Error
		}

		total += result.RowsAffected

		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}
