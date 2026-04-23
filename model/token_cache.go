package model

import (
	"fmt"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
)

func cacheSetToken(token Token) error {
	key := common.GenerateHMAC(token.Key)
	token.Clean()
	err := common.RedisHSetObj(fmt.Sprintf("token:%s", key), &token, time.Duration(common.RedisKeyCacheSeconds())*time.Second)
	if err != nil {
		return err
	}
	return nil
}

func cacheDeleteToken(key string) error {
	key = common.GenerateHMAC(key)
	err := common.RedisDelKey(fmt.Sprintf("token:%s", key))
	if err != nil {
		return err
	}
	return nil
}

func cacheIncrTokenQuota(key string, increment int64) error {
	key = common.GenerateHMAC(key)
	err := common.RedisHIncrBy(fmt.Sprintf("token:%s", key), constant.TokenFiledRemainQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

func cacheDecrTokenQuota(key string, decrement int64) error {
	return cacheIncrTokenQuota(key, -decrement)
}

func cacheIncrWindowUsedQuota(key string, increment int64) error {
	key = common.GenerateHMAC(key)
	err := common.RedisHIncrBy(fmt.Sprintf("token:%s", key), constant.TokenFieldWindowUsedQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

func cacheIncrCycleUsedQuota(key string, increment int64) error {
	key = common.GenerateHMAC(key)
	err := common.RedisHIncrBy(fmt.Sprintf("token:%s", key), constant.TokenFieldCycleUsedQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

// cacheDecrWindowQuotaCond 原子扣减窗口额度，仅当 window_used_quota + quota <= window_quota 时才成功。
func cacheDecrWindowQuotaCond(key string, quota int) (bool, error) {
	key = common.GenerateHMAC(key)
	return common.RedisHIncrByCond(fmt.Sprintf("token:%s", key), constant.TokenFieldWindowUsedQuota, constant.TokenFieldWindowQuota, int64(quota))
}

// cacheDecrCycleQuotaCond 原子扣减周期额度，仅当 cycle_used_quota + quota <= cycle_quota 时才成功。
func cacheDecrCycleQuotaCond(key string, quota int) (bool, error) {
	key = common.GenerateHMAC(key)
	return common.RedisHIncrByCond(fmt.Sprintf("token:%s", key), constant.TokenFieldCycleUsedQuota, constant.TokenFieldCycleQuota, int64(quota))
}

func cacheSetTokenField(key string, field string, value string) error {
	key = common.GenerateHMAC(key)
	err := common.RedisHSetField(fmt.Sprintf("token:%s", key), field, value)
	if err != nil {
		return err
	}
	return nil
}

// CacheGetTokenByKey 从缓存中获取 token，如果缓存中不存在，则从数据库中获取
func cacheGetTokenByKey(key string) (*Token, error) {
	hmacKey := common.GenerateHMAC(key)
	if !common.RedisEnabled {
		return nil, fmt.Errorf("redis is not enabled")
	}
	var token Token
	err := common.RedisHGetObj(fmt.Sprintf("token:%s", hmacKey), &token)
	if err != nil {
		return nil, err
	}
	token.Key = key
	return &token, nil
}
