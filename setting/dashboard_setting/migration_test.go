package dashboard_setting

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/NookMux/NookMux/common"
	"github.com/NookMux/NookMux/model"
	"github.com/NookMux/NookMux/setting/console_setting"
	"gorm.io/gorm"
)

// setupMigrationTestDB 初始化 sqlite 内存数据库并迁移 Option 表，
// 保存并恢复 model.DB 和 common.OptionMap 以保证测试隔离。
func setupMigrationTestDB(t *testing.T) func() {
	t.Helper()

	oldDB := model.DB
	oldOptionMap := common.OptionMap

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Option{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	model.DB = db
	common.OptionMap = make(map[string]string)

	return func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = oldDB
		common.OptionMap = oldOptionMap
	}
}

// resetDashboardConfig 将全局 dashboardConfig 恢复为默认值，
// 保证多次测试之间状态隔离。
func resetDashboardConfig() {
	dashboardConfig = defaultDashboardConfig
}

// resetConsoleSetting 将全局 consoleSetting 的面板开关恢复为默认值，
// 保证多次测试之间状态隔离。
func resetConsoleSetting() {
	cs := console_setting.GetConsoleSetting()
	cs.ApiInfoEnabled = true
	cs.UptimeKumaEnabled = true
	cs.AnnouncementsEnabled = true
	cs.FAQEnabled = true
}

// TestShouldMigrate_NoMarker 验证 OptionMap 中没有 migrated 标记时返回 true。
func TestShouldMigrate_NoMarker(t *testing.T) {
	setupMigrationTestDB(t)
	// OptionMap 为空，不应跳过迁移
	if !ShouldMigrate() {
		t.Fatal("ShouldMigrate 应返回 true：OptionMap 无 migrated 标记")
	}
}

// TestShouldMigrate_MarkerTrue 验证 migrated 标记为 "true" 时返回 false。
func TestShouldMigrate_MarkerTrue(t *testing.T) {
	cleanup := setupMigrationTestDB(t)
	defer cleanup()

	common.OptionMap["dashboard_config.migrated"] = "true"
	if ShouldMigrate() {
		t.Fatal("ShouldMigrate 应返回 false：已标记 migrated=true")
	}
}

// TestShouldMigrate_MarkerNotTrue 验证 migrated 标记存在但非 "true" 时返回 true。
func TestShouldMigrate_MarkerNotTrue(t *testing.T) {
	cleanup := setupMigrationTestDB(t)
	defer cleanup()

	common.OptionMap["dashboard_config.migrated"] = "false"
	if !ShouldMigrate() {
		t.Fatal("ShouldMigrate 应返回 true：migrated 标记为 false 而非 true")
	}
}

// TestMigrateFromConsoleSetting_Idempotent 验证迁移的幂等性：
// 第一次调用执行迁移并写入标记，第二次调用应直接跳过。
func TestMigrateFromConsoleSetting_Idempotent(t *testing.T) {
	cleanup := setupMigrationTestDB(t)
	defer cleanup()
	resetDashboardConfig()
	resetConsoleSetting()

	// 第一次调用：应执行迁移
	if err := MigrateFromConsoleSetting(); err != nil {
		t.Fatalf("第一次迁移失败: %v", err)
	}

	// 验证迁移标记已写入
	if common.OptionMap["dashboard_config.migrated"] != "true" {
		t.Fatalf("迁移后 migrated 标记应为 true，实际为 %q", common.OptionMap["dashboard_config.migrated"])
	}

	// 记录第一次迁移后的配置值
	firstApiInfo := common.OptionMap["dashboard_config.api_info_enabled"]

	// 修改 console_setting 的值，模拟后续变更
	// 如果第二次迁移真的执行了，dashboard_config 的值会变化
	cs := console_setting.GetConsoleSetting()
	cs.ApiInfoEnabled = !cs.ApiInfoEnabled

	// 第二次调用：应跳过迁移（幂等）
	if err := MigrateFromConsoleSetting(); err != nil {
		t.Fatalf("第二次迁移失败: %v", err)
	}

	// 验证值未变化（证明第二次跳过了迁移）
	secondApiInfo := common.OptionMap["dashboard_config.api_info_enabled"]
	if firstApiInfo != secondApiInfo {
		t.Fatalf("幂等性失败：第二次迁移后值从 %q 变为 %q", firstApiInfo, secondApiInfo)
	}
}

// TestMigrateFromConsoleSetting_CopiesPanelToggles 验证迁移正确复制面板开关。
func TestMigrateFromConsoleSetting_CopiesPanelToggles(t *testing.T) {
	cleanup := setupMigrationTestDB(t)
	defer cleanup()
	resetDashboardConfig()
	resetConsoleSetting()

	// 设置 console_setting 的面板开关为已知值
	cs := console_setting.GetConsoleSetting()
	cs.ApiInfoEnabled = false
	cs.UptimeKumaEnabled = true
	cs.AnnouncementsEnabled = false
	cs.FAQEnabled = true

	if err := MigrateFromConsoleSetting(); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	// 验证 dashboard_config 中对应字段已复制
	checks := map[string]string{
		"dashboard_config.api_info_enabled":      "false",
		"dashboard_config.uptime_kuma_enabled":   "true",
		"dashboard_config.announcements_enabled": "false",
		"dashboard_config.faq_enabled":           "true",
	}
	for key, expected := range checks {
		actual := common.OptionMap[key]
		if actual != expected {
			t.Errorf("迁移后 %s 应为 %q，实际为 %q", key, expected, actual)
		}
	}
}
