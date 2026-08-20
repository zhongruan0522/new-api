// 用于迁移检测的旧键，该文件下个版本会删除

package controller

import (
	"encoding/json"
	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/option"
	"github.com/gin-gonic/gin"
	"net/http"
)

// MigrateConsoleSetting 迁移旧的控制台相关配置到 console.*
func MigrateConsoleSetting(c *gin.Context) {
	// 读取全部 option
	opts, err := optionstore.AllOption()
	if err != nil {
		common.SysError("failed to get all options: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": i18n.T(c, i18n.MsgConfigFetchFailed)})
		return
	}
	// 建立 map
	valMap := map[string]string{}
	for _, o := range opts {
		valMap[o.Key] = o.Value
	}

	// 处理 APIInfo
	if v := valMap["ApiInfo"]; v != "" {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			if len(arr) > 50 {
				arr = arr[:50]
			}
			bytes, _ := json.Marshal(arr)
			optionstore.UpdateOption("console.api_info", string(bytes))
		}
		optionstore.UpdateOption("ApiInfo", "")
	}
	// Announcements 直接搬
	if v := valMap["Announcements"]; v != "" {
		optionstore.UpdateOption("console.announcements", v)
		optionstore.UpdateOption("Announcements", "")
	}
	// FAQ 转换
	if v := valMap["FAQ"]; v != "" {
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			out := []map[string]interface{}{}
			for _, item := range arr {
				q, _ := item["question"].(string)
				if q == "" {
					q, _ = item["title"].(string)
				}
				a, _ := item["answer"].(string)
				if a == "" {
					a, _ = item["content"].(string)
				}
				if q != "" && a != "" {
					out = append(out, map[string]interface{}{"question": q, "answer": a})
				}
			}
			if len(out) > 50 {
				out = out[:50]
			}
			bytes, _ := json.Marshal(out)
			optionstore.UpdateOption("console.faq", string(bytes))
		}
		optionstore.UpdateOption("FAQ", "")
	}
	// Uptime Kuma 迁移到新的 groups 结构（console.uptime_kuma_groups）
	url := valMap["UptimeKumaUrl"]
	slug := valMap["UptimeKumaSlug"]
	if url != "" && slug != "" {
		// 仅当同时存在 URL 与 Slug 时才进行迁移
		groups := []map[string]interface{}{
			{
				"id":           1,
				"categoryName": "old",
				"url":          url,
				"slug":         slug,
				"description":  "",
			},
		}
		bytes, _ := json.Marshal(groups)
		optionstore.UpdateOption("console.uptime_kuma_groups", string(bytes))
	}
	// 清空旧键内容
	if url != "" {
		optionstore.UpdateOption("UptimeKumaUrl", "")
	}
	if slug != "" {
		optionstore.UpdateOption("UptimeKumaSlug", "")
	}

	// 删除旧键记录
	oldKeys := []string{"ApiInfo", "Announcements", "FAQ", "UptimeKumaUrl", "UptimeKumaSlug"}
	dbstore.DB.Where("key IN ?", oldKeys).Delete(&optionstore.Option{})

	// 重新加载 OptionMap
	optionstore.InitOptionMap()
	common.SysLog("console setting migrated")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": i18n.T(c, i18n.MsgMiscMigrated)})
}
