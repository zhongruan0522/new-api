package service

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/logger"
	"github.com/zhongruan0522/new-api/model"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

type BillingSession struct {
	relayInfo        *relaycommon.RelayInfo
	funding          FundingSource
	preConsumedQuota int
	tokenConsumed    int
	fundingSettled   bool
	settled          bool
	refunded         bool
	mu               sync.Mutex
}

func (s *BillingSession) Settle(actualQuota int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled {
		return nil
	}

	delta := actualQuota - s.preConsumedQuota
	if delta == 0 {
		s.settled = true
		return nil
	}

	if !s.fundingSettled {
		if err := s.funding.Settle(delta); err != nil {
			return err
		}
		s.fundingSettled = true
	}

	var tokenErr error
	if delta > 0 {
		tokenErr = s.decreaseTokenQuota(delta)
	} else {
		tokenErr = s.increaseTokenQuota(-delta)
	}
	if tokenErr != nil {
		common.SysLog(fmt.Sprintf(
			"error adjusting token quota after funding settled (userId=%d, tokenId=%d, delta=%d): %s",
			s.relayInfo.UserId, s.relayInfo.TokenId, delta, tokenErr.Error(),
		))
	}

	s.settled = true
	return tokenErr
}

func (s *BillingSession) Refund(c *gin.Context) {
	s.mu.Lock()
	if s.settled || s.refunded || !s.needsRefundLocked() {
		s.mu.Unlock()
		return
	}
	s.refunded = true
	s.mu.Unlock()

	logger.LogInfo(c, fmt.Sprintf(
		"用户 %d 请求失败, 返还预扣费（token_quota=%s, funding=%s）",
		s.relayInfo.UserId,
		logger.FormatQuota(s.tokenConsumed),
		s.funding.Source(),
	))

	tokenId := s.relayInfo.TokenId
	tokenKey := s.relayInfo.TokenKey
	tokenConsumed := s.tokenConsumed
	funding := s.funding

	gopool.Go(func() {
		if err := funding.Refund(); err != nil {
			common.SysLog("error refunding billing source: " + err.Error())
		}
		if tokenConsumed > 0 {
			if err := s.increaseTokenQuotaByAmount(tokenId, tokenKey, tokenConsumed); err != nil {
				common.SysLog("error refunding token quota: " + err.Error())
			}
		}
	})
}

func (s *BillingSession) NeedsRefund() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.needsRefundLocked()
}

func (s *BillingSession) needsRefundLocked() bool {
	if s.settled || s.refunded || s.fundingSettled {
		return false
	}
	return s.tokenConsumed > 0
}

func (s *BillingSession) GetPreConsumedQuota() int {
	return s.preConsumedQuota
}

func (s *BillingSession) preConsume(c *gin.Context, quota int) *types.NewAPIError {
	effectiveQuota := quota
	if s.shouldTrust(c) {
		effectiveQuota = 0
		logger.LogInfo(c, fmt.Sprintf("用户 %d 额度充足, 信任且不需要预扣费 (funding=%s)", s.relayInfo.UserId, s.funding.Source()))
	} else if effectiveQuota > 0 {
		logger.LogInfo(c, fmt.Sprintf("用户 %d 需要预扣费 %s (funding=%s)", s.relayInfo.UserId, logger.FormatQuota(effectiveQuota), s.funding.Source()))
	}

	if effectiveQuota > 0 {
		if err := PreConsumeTokenQuota(s.relayInfo, effectiveQuota); err != nil {
			return types.NewErrorWithStatusCode(
				err,
				types.ErrorCodePreConsumeTokenQuotaFailed,
				http.StatusForbidden,
				types.ErrOptionWithSkipRetry(),
				types.ErrOptionWithNoRecordErrorLog(),
			)
		}
		s.tokenConsumed = effectiveQuota
	}

	if err := s.funding.PreConsume(effectiveQuota); err != nil {
		if s.tokenConsumed > 0 {
			if rollbackErr := s.increaseTokenQuotaByAmount(s.relayInfo.TokenId, s.relayInfo.TokenKey, s.tokenConsumed); rollbackErr != nil {
				common.SysLog(fmt.Sprintf(
					"error rolling back token quota (userId=%d, tokenId=%d, amount=%d, fundingErr=%s): %s",
					s.relayInfo.UserId, s.relayInfo.TokenId, s.tokenConsumed, err.Error(), rollbackErr.Error(),
				))
			}
			s.tokenConsumed = 0
		}
		return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
	}

	s.preConsumedQuota = effectiveQuota
	s.syncRelayInfo()
	return nil
}

func (s *BillingSession) shouldTrust(c *gin.Context) bool {
	trustQuota := common.GetTrustQuota()
	if trustQuota <= 0 {
		return false
	}

	quotaType := s.relayInfo.TokenQuotaType

	tokenTrusted := quotaType == 0 || s.relayInfo.TokenUnlimited
	if !tokenTrusted {
		var tokenQuota int
		switch quotaType {
		case 1:
			tokenQuota = s.relayInfo.TokenQuota
		case 2, 3:
			// 对于时段限额模式，使用窗口剩余额度
			// TokenQuota 在 auth 中设置为 RemainQuota，但时段模式下我们需要窗口剩余额度
			// 这里简化处理：如果 TokenQuota > trustQuota 就信任
			tokenQuota = s.relayInfo.TokenQuota
		default:
			tokenQuota = s.relayInfo.TokenQuota
		}
		tokenTrusted = tokenQuota > trustQuota
	}
	if !tokenTrusted {
		return false
	}

	return s.relayInfo.UserQuota > trustQuota
}

func (s *BillingSession) syncRelayInfo() {
	info := s.relayInfo
	info.FinalPreConsumedQuota = s.preConsumedQuota
	info.BillingSource = s.funding.Source()
}

// decreaseTokenQuota 根据配额类型扣减 token 额度
func (s *BillingSession) decreaseTokenQuota(quota int) error {
	quotaType := s.relayInfo.TokenQuotaType
	tokenId := s.relayInfo.TokenId
	tokenKey := s.relayInfo.TokenKey

	switch quotaType {
	case 0: // 无限额度，不扣减
		return nil
	case 1: // 永久限额
		return model.DecreaseTokenQuota(tokenId, tokenKey, quota)
	case 2: // 时段限额
		return model.DecreaseWindowQuota(tokenId, tokenKey, quota)
	case 3: // 时段+周期限额
		if err := model.DecreaseWindowQuota(tokenId, tokenKey, quota); err != nil {
			return err
		}
		return model.DecreaseCycleQuota(tokenId, tokenKey, quota)
	default:
		return model.DecreaseTokenQuota(tokenId, tokenKey, quota)
	}
}

// increaseTokenQuota 根据配额类型退还 token 额度
func (s *BillingSession) increaseTokenQuota(quota int) error {
	quotaType := s.relayInfo.TokenQuotaType
	tokenId := s.relayInfo.TokenId
	tokenKey := s.relayInfo.TokenKey

	switch quotaType {
	case 0: // 无限额度，不退还
		return nil
	case 1: // 永久限额
		return model.IncreaseTokenQuota(tokenId, tokenKey, quota)
	case 2: // 时段限额
		return model.IncreaseWindowQuota(tokenId, tokenKey, quota)
	case 3: // 时段+周期限额
		if err := model.IncreaseWindowQuota(tokenId, tokenKey, quota); err != nil {
			return err
		}
		return model.IncreaseCycleQuota(tokenId, tokenKey, quota)
	default:
		return model.IncreaseTokenQuota(tokenId, tokenKey, quota)
	}
}

// increaseTokenQuotaByAmount 退还 token 额度（用于退款场景，使用独立的 tokenId/tokenKey 参数）
func (s *BillingSession) increaseTokenQuotaByAmount(tokenId int, tokenKey string, quota int) error {
	quotaType := s.relayInfo.TokenQuotaType

	switch quotaType {
	case 0:
		return nil
	case 1:
		return model.IncreaseTokenQuota(tokenId, tokenKey, quota)
	case 2:
		return model.IncreaseWindowQuota(tokenId, tokenKey, quota)
	case 3:
		if err := model.IncreaseWindowQuota(tokenId, tokenKey, quota); err != nil {
			return err
		}
		return model.IncreaseCycleQuota(tokenId, tokenKey, quota)
	default:
		return model.IncreaseTokenQuota(tokenId, tokenKey, quota)
	}
}

func NewBillingSession(c *gin.Context, relayInfo *relaycommon.RelayInfo, preConsumedQuota int) (*BillingSession, *types.NewAPIError) {
	if relayInfo == nil {
		return nil, types.NewError(
			fmt.Errorf("relayInfo is nil"),
			types.ErrorCodeInvalidRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	userQuota, err := model.GetUserQuota(relayInfo.UserId, false)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
	}
	if userQuota <= 0 {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("用户额度不足, 剩余额度: %s", logger.FormatQuota(userQuota)),
			types.ErrorCodeInsufficientUserQuota,
			http.StatusForbidden,
			types.ErrOptionWithSkipRetry(),
			types.ErrOptionWithNoRecordErrorLog(),
		)
	}
	if userQuota-preConsumedQuota < 0 {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf(
				"预扣费额度失败, 用户剩余额度: %s, 需要预扣费额度: %s",
				logger.FormatQuota(userQuota),
				logger.FormatQuota(preConsumedQuota),
			),
			types.ErrorCodeInsufficientUserQuota,
			http.StatusForbidden,
			types.ErrOptionWithSkipRetry(),
			types.ErrOptionWithNoRecordErrorLog(),
		)
	}
	relayInfo.UserQuota = userQuota

	session := &BillingSession{
		relayInfo: relayInfo,
		funding:   &WalletFunding{userId: relayInfo.UserId},
	}
	if apiErr := session.preConsume(c, preConsumedQuota); apiErr != nil {
		return nil, apiErr
	}
	return session, nil
}
