package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/go-redis/redis/v8"
	"sync"
)

//go:embed lua/rate_limit.lua
var rateLimitScript string

type RedisLimiter struct {
	client         *redis.Client
	limitScriptSHA string
}

var (
	instance *RedisLimiter
	once     sync.Once
)

func New(ctx context.Context, r *redis.Client) *RedisLimiter {
	once.Do(func() {
		client := r
		_, err := client.Ping(ctx).Result()
		if err != nil {
			panic(err) // 或者处理连接错误
		}
		// 预加载脚本
		limitSHA, err := client.ScriptLoad(ctx, rateLimitScript).Result()
		if err != nil {
			fmt.Println(err)
		}

		instance = &RedisLimiter{
			client:         client,
			limitScriptSHA: limitSHA,
		}
	})

	return instance
}

func (rl *RedisLimiter) Allow(ctx context.Context, key string, opts ...Option) (bool, error) {
	// 默认配置
	config := &Config{
		Capacity:  10,
		Rate:      1,
		Requested: 1,
	}

	// 应用选项模式
	for _, opt := range opts {
		opt(config)
	}

	// 执行限流
	result, err := rl.client.EvalSha(
		ctx,
		rl.limitScriptSHA,
		[]string{key},
		config.Requested,
		config.Rate,
		config.Capacity,
	).Int()

	if err != nil {
		return false, fmt.Errorf("rate limit failed: %w", err)
	}
	return result == 1, nil
}

// Config 配置选项模式
type Config struct {
	Capacity  int64
	Rate      int64
	Requested int64
}

type Option func(*Config)

func WithCapacity(c int64) Option {
	return func(cfg *Config) { cfg.Capacity = c }
}

func WithRate(r int64) Option {
	return func(cfg *Config) { cfg.Rate = r }
}

func WithRequested(n int64) Option {
	return func(cfg *Config) { cfg.Requested = n }
}
