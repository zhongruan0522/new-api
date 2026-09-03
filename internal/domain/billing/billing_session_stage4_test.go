package billing

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/domain/shared"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	dbstore "github.com/NookMux/NookMux/internal/store/db"
	tokenstore "github.com/NookMux/NookMux/internal/store/token"
	userstore "github.com/NookMux/NookMux/internal/store/user"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stage4FundingSource struct {
	mu          sync.Mutex
	preConsumed []int
	settled     []int
	refunded    int
}

func (f *stage4FundingSource) Source() string { return "stage4-test" }

func (f *stage4FundingSource) PreConsume(amount int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.preConsumed = append(f.preConsumed, amount)
	return nil
}

func (f *stage4FundingSource) Settle(delta int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.settled = append(f.settled, delta)
	return nil
}

func (f *stage4FundingSource) Refund() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refunded++
	return nil
}

func (f *stage4FundingSource) operations() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.settled), f.refunded
}

// Concurrent settlement calls must be idempotent: only the first delta may
// touch funding and token quota.
func TestStage4ConcurrentBillingSettleIsIdempotent(t *testing.T) {
	setupApplyQuotaTestDB(t)
	gin.SetMode(gin.TestMode)
	relayInfo := newApplyQuotaTestRelayInfo()
	funding := &stage4FundingSource{}
	session := &BillingSession{
		relayInfo:        relayInfo,
		funding:          funding,
		preConsumedQuota: 100,
		tokenConsumed:    100,
	}
	var wg sync.WaitGroup
	const goroutines = 8
	settleErrs := make(chan error, goroutines)
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			settleErrs <- session.Settle(120)
		}()
	}
	wg.Wait()
	close(settleErrs)
	for err := range settleErrs {
		require.NoError(t, err)
	}

	var settledCount, refundedCount int
	require.Eventually(t, func() bool {
		settledCount, refundedCount = funding.operations()
		return settledCount+refundedCount == 1
	}, 10*time.Second, 5*time.Millisecond, fmt.Sprintf("settles=%d refunds=%d", settledCount, refundedCount))

	time.Sleep(25 * time.Millisecond)
	settledCount, refundedCount = funding.operations()
	assert.Equal(t, 1, settledCount, "funding delta must settle exactly once")
	assert.Zero(t, refundedCount)
	assert.False(t, session.NeedsRefund(), "completed settle or refund must close the lifecycle")
}

// A refunded request is terminal. The guard must cover the window after
// Refund marks its winner but before its asynchronous funding/token rollback
// finishes, or a late Settle can charge refunded quota again.
func TestStage4RefundPreventsLateSettlement(t *testing.T) {
	setupApplyQuotaTestDB(t)
	gin.SetMode(gin.TestMode)
	relayInfo := newApplyQuotaTestRelayInfo()
	funding := &stage4FundingSource{}
	session := &BillingSession{
		relayInfo:        relayInfo,
		funding:          funding,
		preConsumedQuota: 100,
		// Funding preconsume keeps Refund meaningful without scheduling the
		// unrelated token-cache rollback goroutine past test cleanup.
		tokenConsumed: 0,
	}

	// Put the terminal state directly into the guarded window: after the refund
	// winner has marked the session but while asynchronous rollback can still be
	// in flight. This avoids coupling the state-machine assertion to global
	// token-cache goroutines.
	session.mu.Lock()
	session.refunded = true
	session.mu.Unlock()
	require.True(t, session.refunded)
	require.NoError(t, session.Settle(120))
	assert.Zero(t, len(funding.settled), "settlement must not race a completed refund")
	assert.False(t, session.NeedsRefund())
}

// Refund must actually roll back both legs of the pre-consume: the funding
// source gets its asynchronous Refund call and the token's recorded usage is
// returned. A late Settle after the refund must stay a no-op.
func TestStage4RefundRollsBackFundingAndTokenUsage(t *testing.T) {
	setupApplyQuotaTestDB(t)
	ctx := newEntryTestContext("tk-stage4-refund")
	relayInfo := newEntryTestRelayInfo(relayconstant.UsageSourceOpenAIChat)
	funding := &stage4FundingSource{}
	session := &BillingSession{
		relayInfo:        relayInfo,
		funding:          funding,
		preConsumedQuota: 100,
		// tokenConsumed > 0 is the needsRefund precondition; simulate the
		// pre-consume's recorded usage so the rollback has something to undo.
		tokenConsumed: 100,
	}
	require.NoError(t, dbstore.DB.Model(&tokenstore.Token{}).
		Where("id = ?", applyQuotaTestTokenId).
		Update("used_quota", 100).Error)

	session.Refund(ctx)
	require.False(t, session.NeedsRefund(), "refund must close the lifecycle")

	require.Eventually(t, func() bool {
		_, refunded := funding.operations()
		return refunded == 1
	}, 10*time.Second, 5*time.Millisecond, "funding refund must run exactly once")

	var token tokenstore.Token
	require.NoError(t, dbstore.DB.First(&token, applyQuotaTestTokenId).Error)
	assert.Zero(t, token.UsedQuota, "token used quota must be rolled back by the refund")

	require.NoError(t, session.Settle(120))
	_, refunded := funding.operations()
	assert.Equal(t, 1, refunded, "late settle must not add funding operations after refund")
	assert.Zero(t, len(funding.settled))
}

// Realtime charges each event directly. The request-level session only holds
// the initial preconsume, so successful finalization must release that amount
// without treating the aggregate summary quota as another funding delta.
func TestStage4PostWssReleasesInitialSessionOnly(t *testing.T) {
	setupApplyQuotaTestDB(t)
	gin.SetMode(gin.TestMode)
	ctx := newEntryTestContext("tk-stage4-wss-session")
	relayInfo := newEntryTestRelayInfo(relayconstant.UsageSourceOpenAIResponses)
	funding := &stage4FundingSource{}
	relayInfo.Billing = &BillingSession{
		relayInfo:        relayInfo,
		funding:          funding,
		preConsumedQuota: 100,
	}
	// quota_type=0 tracks usage instead of remaining quota; simulate the initial
	// preconsume's recorded usage so releasing it does not underflow the counter.
	require.NoError(t, dbstore.DB.Model(&tokenstore.Token{}).
		Where("id = ?", applyQuotaTestTokenId).
		Update("used_quota", 100).Error)

	usage := &shared.RealtimeUsage{
		TotalTokens:  100,
		InputTokens:  60,
		OutputTokens: 40,
	}
	require.Nil(t, PostWssConsumeQuota(ctx, relayInfo, relayInfo.OriginModelName, usage, ""))
	require.Equal(t, []int{-100}, funding.settled)
	require.Zero(t, funding.refunded)
	require.False(t, relayInfo.Billing.NeedsRefund())
	var user userstore.User
	require.NoError(t, dbstore.DB.First(&user, applyQuotaTestUserId).Error)
	assert.Equal(t, 360, user.UsedQuota, "summary usage remains after releasing the initial preconsume")

	// Keep the asynchronous summary log inside the test database lifetime.
	waitForConsumeLogByTokenName(t, "tk-stage4-wss-session")
}
