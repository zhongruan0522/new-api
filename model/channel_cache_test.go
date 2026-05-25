package model

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"gorm.io/gorm"
)

func setupChannelCacheTestDB(t *testing.T) {
	t.Helper()

	oldDB := DB
	oldRedisEnabled := common.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	oldUsingSQLite := common.UsingSQLite
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldUsingMySQL := common.UsingMySQL
	oldGroup2Model2Channels := group2model2channels
	oldChannelsIDM := channelsIDM

	common.RedisEnabled = false
	common.MemoryCacheEnabled = true
	common.BatchUpdateEnabled = false
	common.UsingSQLite = true
	common.UsingPostgreSQL = false
	common.UsingMySQL = false
	initCol()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&Channel{}, &Ability{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	DB = db
	group2model2channels = nil
	channelsIDM = nil

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		common.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
		common.UsingSQLite = oldUsingSQLite
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.UsingMySQL = oldUsingMySQL
		initCol()
		group2model2channels = oldGroup2Model2Channels
		channelsIDM = oldChannelsIDM
	})
}

func createChannelCacheTestChannel(t *testing.T, channel Channel) Channel {
	t.Helper()

	if channel.Type == 0 {
		channel.Type = constant.ChannelTypeOpenAI
	}
	if channel.Key == "" {
		channel.Key = "test-key"
	}
	if channel.Name == "" {
		channel.Name = "test-channel"
	}
	if channel.Status == 0 {
		channel.Status = common.ChannelStatusEnabled
	}
	if channel.Models == "" {
		channel.Models = "claude-haiku-4-5-20251001"
	}
	if channel.Group == "" {
		channel.Group = "Coding"
	}
	if channel.Priority == nil {
		priority := int64(0)
		channel.Priority = &priority
	}
	if channel.Weight == nil {
		weight := uint(1)
		channel.Weight = &weight
	}

	if err := DB.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if err := channel.AddAbilities(DB); err != nil {
		t.Fatalf("create abilities: %v", err)
	}
	return channel
}

func TestCacheUpdateChannelStatusReaddsEnabledChannel(t *testing.T) {
	setupChannelCacheTestDB(t)
	channel := createChannelCacheTestChannel(t, Channel{})

	InitChannelCache()

	selected, err := GetRandomSatisfiedChannel("Coding", "claude-haiku-4-5-20251001", 0, -1, 0)
	if err != nil {
		t.Fatalf("initial channel selection failed: %v", err)
	}
	if selected == nil || selected.Id != channel.Id {
		t.Fatalf("initial selected channel = %+v, want channel %d", selected, channel.Id)
	}

	CacheUpdateChannelStatus(channel.Id, common.ChannelStatusAutoDisabled)
	selected, err = GetRandomSatisfiedChannel("Coding", "claude-haiku-4-5-20251001", 0, -1, 0)
	if err != nil {
		t.Fatalf("selection after disable returned error: %v", err)
	}
	if selected != nil {
		t.Fatalf("selected disabled channel %+v", selected)
	}

	CacheUpdateChannelStatus(channel.Id, common.ChannelStatusEnabled)
	selected, err = GetRandomSatisfiedChannel("Coding", "claude-haiku-4-5-20251001", 0, -1, 0)
	if err != nil {
		t.Fatalf("selection after re-enable failed: %v", err)
	}
	if selected == nil || selected.Id != channel.Id {
		t.Fatalf("selected after re-enable = %+v, want channel %d", selected, channel.Id)
	}
	if !IsChannelEnabledForGroupModel("Coding", "claude-haiku-4-5-20251001", channel.Id) {
		t.Fatalf("channel %d was not restored to group/model cache", channel.Id)
	}
}

func TestUpdateChannelStatusRestoresMultiKeyChannelToCache(t *testing.T) {
	setupChannelCacheTestDB(t)
	channel := createChannelCacheTestChannel(t, Channel{
		Key: "key-1\nkey-2",
		ChannelInfo: ChannelInfo{
			IsMultiKey:   true,
			MultiKeySize: 2,
		},
	})

	InitChannelCache()

	if !UpdateChannelStatus(channel.Id, "key-1", common.ChannelStatusAutoDisabled, "test disable key 1") {
		t.Fatalf("disable first key returned false")
	}
	selected, err := GetRandomSatisfiedChannel("Coding", "claude-haiku-4-5-20251001", 0, -1, 0)
	if err != nil {
		t.Fatalf("selection after first key disable failed: %v", err)
	}
	if selected == nil || selected.Id != channel.Id {
		t.Fatalf("channel should remain selectable while one key is enabled, got %+v", selected)
	}

	if !UpdateChannelStatus(channel.Id, "key-2", common.ChannelStatusAutoDisabled, "test disable key 2") {
		t.Fatalf("disable second key returned false")
	}
	selected, err = GetRandomSatisfiedChannel("Coding", "claude-haiku-4-5-20251001", 0, -1, 0)
	if err != nil {
		t.Fatalf("selection after all keys disabled returned error: %v", err)
	}
	if selected != nil {
		t.Fatalf("selected channel with all keys disabled: %+v", selected)
	}

	if !UpdateChannelStatus(channel.Id, "key-1", common.ChannelStatusEnabled, "") {
		t.Fatalf("enable first key returned false")
	}
	selected, err = GetRandomSatisfiedChannel("Coding", "claude-haiku-4-5-20251001", 0, -1, 0)
	if err != nil {
		t.Fatalf("selection after key re-enable failed: %v", err)
	}
	if selected == nil || selected.Id != channel.Id {
		t.Fatalf("selected after key re-enable = %+v, want channel %d", selected, channel.Id)
	}
}
