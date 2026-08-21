package tokenstore

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/infra/redis"
	"github.com/NookMux/NookMux/internal/infra/security"
	"time"
)

func cacheSetToken(token Token) error {
	key := security.GenerateHMAC(token.Key)
	token.Clean()
	err := redis.RedisHSetObj(fmt.Sprintf("token:%s", key), &token, time.Duration(redis.RedisKeyCacheSeconds())*time.Second)
	if err != nil {
		return err
	}
	return nil
}

func cacheDeleteToken(key string) error {
	key = security.GenerateHMAC(key)
	err := redis.RedisDelKey(fmt.Sprintf("token:%s", key))
	if err != nil {
		return err
	}
	return nil
}

func cacheIncrTokenQuota(key string, increment int64) error {
	key = security.GenerateHMAC(key)
	err := redis.RedisHIncrBy(fmt.Sprintf("token:%s", key), shared.TokenFiledRemainQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

func cacheDecrTokenQuota(key string, decrement int64) error {
	return cacheIncrTokenQuota(key, -decrement)
}

func cacheIncrTokenUsedQuota(key string, increment int64) error {
	key = security.GenerateHMAC(key)
	err := redis.RedisHIncrBy(fmt.Sprintf("token:%s", key), shared.TokenFieldUsedQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

func cacheIncrWindowUsedQuota(key string, increment int64) error {
	key = security.GenerateHMAC(key)
	err := redis.RedisHIncrBy(fmt.Sprintf("token:%s", key), shared.TokenFieldWindowUsedQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

func cacheIncrCycleUsedQuota(key string, increment int64) error {
	key = security.GenerateHMAC(key)
	err := redis.RedisHIncrBy(fmt.Sprintf("token:%s", key), shared.TokenFieldCycleUsedQuota, increment)
	if err != nil {
		return err
	}
	return nil
}

// CacheGetTokenByKey 从缓存中获取 token，如果缓存中不存在，则从数据库中获取
func cacheGetTokenByKey(key string) (*Token, error) {
	hmacKey := security.GenerateHMAC(key)
	if !redis.RedisEnabled {
		return nil, fmt.Errorf("redis is not enabled")
	}
	var token Token
	err := redis.RedisHGetObj(fmt.Sprintf("token:%s", hmacKey), &token)
	if err != nil {
		return nil, err
	}
	token.Key = key
	return &token, nil
}
