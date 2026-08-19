package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/NookMux/NookMux/internal/config/operation"
)

// buildChannelAffinityContextForTest 构造一个已命中亲和规则的请求上下文。
func buildChannelAffinityContextForTest(meta channelAffinityMeta) *gin.Context {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Request = req
	setChannelAffinityContext(ctx, meta)
	return ctx
}

// TestClearCurrentChannelAffinityCache 验证清空当前粘性条目：
// 修复前不存在该能力（禁用渠道命中后条目残留，后续请求继续命中失效渠道）。
func TestClearCurrentChannelAffinityCache(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cacheKeySuffix := fmt.Sprintf("rule-x:default:clear-current-%d", time.Now().UnixNano())
	cacheKeyFull := channelAffinityCacheNamespace + ":" + cacheKeySuffix
	cache := getChannelAffinityCache()
	require.NoError(t, cache.SetWithTTL(cacheKeySuffix, 9527, time.Minute))
	t.Cleanup(func() {
		_, _ = cache.DeleteMany([]string{cacheKeySuffix})
	})

	ctx := buildChannelAffinityContextForTest(channelAffinityMeta{
		CacheKey:   cacheKeyFull,
		TTLSeconds: 60,
		RuleName:   "rule-x",
		SkipRetry:  true,
	})
	ctx.Set(ginKeyChannelAffinitySkipRetry, true)
	require.True(t, ShouldSkipRetryAfterChannelAffinityFailure(ctx))

	deleted := ClearCurrentChannelAffinityCache(ctx)
	require.True(t, deleted)
	_, found, err := cache.Get(cacheKeySuffix)
	require.NoError(t, err)
	require.False(t, found)
	require.False(t, ShouldSkipRetryAfterChannelAffinityFailure(ctx))
}

// TestClearCurrentChannelAffinityCacheMissingContext 无亲和上下文时应返回 false 且不报错。
func TestClearCurrentChannelAffinityCacheMissingContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	require.False(t, ClearCurrentChannelAffinityCache(ctx))
	require.False(t, ClearCurrentChannelAffinityCache(nil))
}

// TestShouldKeepChannelAffinityOnChannelDisabled 默认关闭（禁用渠道时清空粘性），
// 可通过配置打开。
func TestShouldKeepChannelAffinityOnChannelDisabled(t *testing.T) {
	setting := operation.GetChannelAffinitySetting()
	orig := setting.KeepOnChannelDisabled
	t.Cleanup(func() {
		setting.KeepOnChannelDisabled = orig
	})

	setting.KeepOnChannelDisabled = false
	require.False(t, ShouldKeepChannelAffinityOnChannelDisabled())

	setting.KeepOnChannelDisabled = true
	require.True(t, ShouldKeepChannelAffinityOnChannelDisabled())
}
