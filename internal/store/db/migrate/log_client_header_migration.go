package dbmigrate

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/internal/store/option"
)

// logClientHeaderBackfillMarker 是一次性数据迁移标记，存储在主库 options 表。
// 删除该 options 行可强制重新执行回填（迁移本身幂等）。
const logClientHeaderBackfillMarker = "data_migration.log_client_headers.v1.done"

// logClientHeaderKeys 是从 Other JSON 迁移到专用列的客户端请求头键名。
var logClientHeaderKeys = []string{"ua", "x_title", "http_referer"}

// backfillLogClientHeaderColumns 是一次性数据迁移：
// 将历史日志存放在 Other JSON 里的 ua / x_title / http_referer 回填到新增的专用列，
// 并从 Other JSON 中移除这些已迁移的键，避免历史行长期双份存储。
//
// 背景：这三个客户端请求头原本序列化进 Other 列，现已独立为列作为唯一数据来源。
// 新写入只写列；历史数据需要从 Other JSON 中提取回填，以便检索与展示。
//
// 幂等性：通过 options 表的 data-migration marker 保证只完整执行一次。
// 迁移按 id 游标分批推进、单条更新。任意批次查询失败，或任意单行更新失败时，
// 均不写 marker，下次启动会重试未完成的行（迁移本身幂等）。注意 marker 存于
// 主库 DB，日志数据在 LOG_DB，二者可分离（marker 仅作全局开关）。
func backfillLogClientHeaderColumns() {
	if dbstore.LOG_DB == nil {
		return
	}
	if optionstore.IsDataMigrationDone(logClientHeaderBackfillMarker) {
		return
	}

	const batchSize = 1000
	updated := 0
	failed := 0
	maxId := 0
	for {
		type logOtherRow struct {
			Id    int    `gorm:"column:id"`
			Other string `gorm:"column:other"`
		}
		var rows []logOtherRow
		err := dbstore.LOG_DB.Model(&logstore.Log{}).
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
			if uerr := dbstore.LOG_DB.Model(&logstore.Log{}).Where("id = ?", row.Id).Updates(updates).Error; uerr != nil {
				// 单行更新失败：计数并继续，但最终不写 marker，保证下次启动可重试。
				failed++
				common.SysError(fmt.Sprintf("backfillLogClientHeaderColumns: update id=%d failed: %v", row.Id, uerr))
				continue
			}
			updated++
		}

		maxId = rows[len(rows)-1].Id
		if len(rows) < batchSize {
			break
		}
	}

	// 仅当所有候选行都成功更新时才写 marker；存在失败时让下次启动补跑。
	if failed > 0 {
		common.SysError(fmt.Sprintf("backfillLogClientHeaderColumns: finished with %d rows updated, %d failed; marker not set, will retry next startup", updated, failed))
		return
	}
	optionstore.MarkDataMigrationDone(logClientHeaderBackfillMarker)
	common.SysLog(fmt.Sprintf("backfillLogClientHeaderColumns: completed, %d rows updated", updated))
}

// extractBackfillUpdates 从 Other JSON 中提取三个客户端请求头的值，
// 回填到专用列，并同时从 Other JSON 中移除这些已迁移的键（写回 other 列），
// 使历史行的客户端头不再冗余存于两处。仅返回存在且非空的字段。
func extractBackfillUpdates(otherJSON string) map[string]interface{} {
	otherMap, err := common.StrToMap(otherJSON)
	if err != nil || otherMap == nil {
		return nil
	}
	updates := make(map[string]interface{})
	migrated := false
	for _, key := range logClientHeaderKeys {
		if v, ok := otherMap[key].(string); ok && v != "" {
			updates[key] = v
		}
		// 无论该键是否有非空值，只要 Other JSON 中存在该键就删除，
		// 避免空值残留导致以后被认为“未迁移”。
		if _, exists := otherMap[key]; exists {
			delete(otherMap, key)
			migrated = true
		}
	}
	// 同步写回清理后的 Other JSON，保证空对象仍存为 "{}"。
	if migrated {
		updates["other"] = common.MapToJsonStr(otherMap)
	}
	return updates
}
