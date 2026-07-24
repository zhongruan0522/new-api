package model

import (
	"fmt"

	"github.com/NookMux/NookMux/common"
)

// logClientHeaderBackfillMarker 是一次性数据迁移标记，存储在主库 options 表。
// 删除该 options 行可强制重新执行回填（迁移本身幂等）。
const logClientHeaderBackfillMarker = "data_migration.log_client_headers.v1.done"

// backfillLogClientHeaderColumns 是一次性数据迁移：
// 将历史日志存放在 Other JSON 里的 ua / x_title / http_referer 回填到新增的专用列。
//
// 背景：这三个客户端请求头原本序列化进 Other 列，现已独立为列作为唯一数据来源。
// 新写入只写列；历史数据需要从 Other JSON 中提取回填，以便检索与展示。
//
// 幂等性：通过 options 表的 data-migration marker 保证只完整执行一次。
// 迁移按 id 游标分批推进、单条更新；任意批次查询失败时直接返回且不写 marker，
// 下次启动会重试（迁移本身幂等）。注意 marker 存于主库 DB，日志数据在 LOG_DB，
// 二者可分离（marker 仅作全局开关）。
func backfillLogClientHeaderColumns() {
	if LOG_DB == nil {
		return
	}
	if isDataMigrationDone(logClientHeaderBackfillMarker) {
		return
	}

	const batchSize = 1000
	processed := 0
	maxId := 0
	for {
		type logOtherRow struct {
			Id    int    `gorm:"column:id"`
			Other string `gorm:"column:other"`
		}
		var rows []logOtherRow
		err := LOG_DB.Model(&Log{}).
			Select("id, other").
			Where("id > ? AND other <> ''", maxId).
			Order("id asc").
			Limit(batchSize).
			Find(&rows).Error
		if err != nil {
			// 查询失败保守终止且不写 marker，下次启动重试。
			common.SysError(fmt.Sprintf("backfillLogClientHeaderColumns: query batch failed (maxId=%d): %v", maxId, err))
			return
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			updates := extractBackfillUpdates(row.Other)
			if len(updates) == 0 {
				continue
			}
			if uerr := LOG_DB.Model(&Log{}).Where("id = ?", row.Id).Updates(updates).Error; uerr != nil {
				common.SysError(fmt.Sprintf("backfillLogClientHeaderColumns: update id=%d failed: %v", row.Id, uerr))
			}
			processed++
		}

		maxId = rows[len(rows)-1].Id
		if len(rows) < batchSize {
			break
		}
	}

	markDataMigrationDone(logClientHeaderBackfillMarker)
	common.SysLog(fmt.Sprintf("backfillLogClientHeaderColumns: completed, %d rows updated", processed))
}

// extractBackfillUpdates 从 Other JSON 中提取三个客户端请求头的值，
// 仅返回存在且非空的字段，用于回填到专用列。
func extractBackfillUpdates(otherJSON string) map[string]interface{} {
	otherMap, err := common.StrToMap(otherJSON)
	if err != nil || otherMap == nil {
		return nil
	}
	updates := make(map[string]interface{})
	if v, ok := otherMap["ua"].(string); ok && v != "" {
		updates["ua"] = v
	}
	if v, ok := otherMap["x_title"].(string); ok && v != "" {
		updates["x_title"] = v
	}
	if v, ok := otherMap["http_referer"].(string); ok && v != "" {
		updates["http_referer"] = v
	}
	return updates
}
