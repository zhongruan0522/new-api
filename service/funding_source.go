package service

import (
	"fmt"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/logger"
	"github.com/zhongruan0522/new-api/model"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"gorm.io/gorm"
)

// FundingSource abstracts a pre-consume / settle / refund lifecycle for a funding source.
type FundingSource interface {
	Source() string
	PreConsume(amount int) error
	Settle(delta int) error
	Refund() error
}

type WalletFunding struct {
	userId   int
	consumed int
}

type SubscriptionFunding struct {
	userId         int
	subscriptionId int
	requestId      string
	relayInfo      *relaycommon.RelayInfo
	consumed       int
}

type SubscriptionThenWalletFunding struct {
	userId               int
	subscriptionId       int
	requestId            string
	relayInfo            *relaycommon.RelayInfo
	subscriptionConsumed int
	walletConsumed       int
}

func (w *WalletFunding) Source() string { return BillingSourceWallet }

func (w *WalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	if err := model.DecreaseUserQuota(w.userId, amount); err != nil {
		return err
	}
	w.consumed = amount
	return nil
}

func (w *WalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return model.DecreaseUserQuota(w.userId, delta)
	}
	return model.IncreaseUserQuota(w.userId, -delta, false)
}

func (w *WalletFunding) Refund() error {
	if w.consumed <= 0 {
		return nil
	}
	return model.IncreaseUserQuota(w.userId, w.consumed, false)
}

func (s *SubscriptionFunding) Source() string { return BillingSourceSubscription }

func (s *SubscriptionFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		sub, err := model.GetActiveUserSubscriptionTx(tx, s.userId, now, true)
		if err != nil {
			return err
		}
		if sub == nil || sub.Id != s.subscriptionId {
			return fmt.Errorf("当前没有可用套餐")
		}
		snapshot := sub.Snapshot(now)
		if snapshot.TotalRemaining <= 0 || snapshot.WindowRemaining < amount {
			return fmt.Errorf("套餐额度不足，当前窗口剩余额度 %s，总剩余额度 %s", logger.FormatQuota(snapshot.WindowRemaining), logger.FormatQuota(snapshot.TotalRemaining))
		}
		bill := sub.ConsumeQuota(amount)
		sub.UpdatedTime = now
		bill.ChannelId = s.getChannelId()
		bill.ModelName = s.getModelName()
		bill.RequestId = s.requestId
		bill.Event = common.SubscriptionBillEventPreConsume
		bill.Content = fmt.Sprintf("套餐预扣费 %s", logger.FormatQuota(amount))
		if err := model.UpdateUserSubscription(tx, sub, "window_used_quota", "used_total_quota", "next_reset_at", "updated_time"); err != nil {
			return err
		}
		if err := model.CreateSubscriptionBill(tx, &bill); err != nil {
			return err
		}
		s.consumed = amount
		return nil
	})
}

func (s *SubscriptionFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		sub, err := model.GetActiveUserSubscriptionTx(tx, s.userId, now, true)
		if err != nil {
			return err
		}
		if sub == nil || sub.Id != s.subscriptionId {
			return fmt.Errorf("当前没有可用套餐")
		}
		var bill model.SubscriptionBill
		if delta > 0 {
			bill = sub.ConsumeQuota(delta)
			bill.Event = common.SubscriptionBillEventSettle
			bill.Content = fmt.Sprintf("套餐补扣 %s", logger.FormatQuota(delta))
		} else {
			bill = sub.RefundQuota(-delta)
			bill.Event = common.SubscriptionBillEventRefund
			bill.Content = fmt.Sprintf("套餐返还 %s", logger.FormatQuota(-delta))
		}
		bill.ChannelId = s.getChannelId()
		bill.ModelName = s.getModelName()
		bill.RequestId = s.requestId
		sub.UpdatedTime = now
		if err := model.UpdateUserSubscription(tx, sub, "window_used_quota", "used_total_quota", "next_reset_at", "updated_time"); err != nil {
			return err
		}
		return model.CreateSubscriptionBill(tx, &bill)
	})
}

func (s *SubscriptionFunding) Refund() error {
	if s.consumed <= 0 {
		return nil
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		sub, err := model.GetActiveUserSubscriptionTx(tx, s.userId, now, true)
		if err != nil {
			return err
		}
		if sub == nil || sub.Id != s.subscriptionId {
			return fmt.Errorf("当前没有可用套餐")
		}
		bill := sub.RefundQuota(s.consumed)
		sub.UpdatedTime = now
		bill.ChannelId = s.getChannelId()
		bill.ModelName = s.getModelName()
		bill.RequestId = s.requestId
		bill.Event = common.SubscriptionBillEventRefund
		bill.Content = fmt.Sprintf("套餐请求失败返还 %s", logger.FormatQuota(s.consumed))
		if err := model.UpdateUserSubscription(tx, sub, "window_used_quota", "used_total_quota", "next_reset_at", "updated_time"); err != nil {
			return err
		}
		return model.CreateSubscriptionBill(tx, &bill)
	})
}

func (s *SubscriptionFunding) getChannelId() int {
	if s.relayInfo != nil && s.relayInfo.ChannelMeta != nil {
		return s.relayInfo.ChannelId
	}
	return 0
}

func (s *SubscriptionFunding) getModelName() string {
	if s.relayInfo != nil {
		return s.relayInfo.OriginModelName
	}
	return ""
}

func (s *SubscriptionThenWalletFunding) Source() string {
	if s.subscriptionConsumed > 0 && s.walletConsumed > 0 {
		return BillingSourceSubscriptionWallet
	}
	if s.subscriptionConsumed > 0 {
		return BillingSourceSubscription
	}
	return BillingSourceWallet
}

func (s *SubscriptionThenWalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	return s.consume(amount, common.SubscriptionBillEventPreConsume, "套餐优先扣费")
}

func (s *SubscriptionThenWalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return s.consume(delta, common.SubscriptionBillEventSettle, "套餐优先补扣")
	}
	return s.refund(-delta)
}

func (s *SubscriptionThenWalletFunding) Refund() error {
	return s.refund(s.subscriptionConsumed + s.walletConsumed)
}

func (s *SubscriptionThenWalletFunding) consume(amount int, event string, contentPrefix string) error {
	var subscriptionConsumed int
	var walletConsumed int
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		subscriptionAmount := 0
		if s.subscriptionId > 0 {
			sub, err := model.GetActiveUserSubscriptionTx(tx, s.userId, now, true)
			if err != nil {
				return err
			}
			if sub != nil && sub.Id == s.subscriptionId {
				snapshot := sub.Snapshot(now)
				subscriptionAmount = snapshot.WindowRemaining
				if subscriptionAmount > amount {
					subscriptionAmount = amount
				}
				if subscriptionAmount > 0 {
					bill := sub.ConsumeQuota(subscriptionAmount)
					sub.UpdatedTime = now
					bill.ChannelId = s.getChannelId()
					bill.ModelName = s.getModelName()
					bill.RequestId = s.requestId
					bill.Event = event
					bill.Content = fmt.Sprintf("%s %s", contentPrefix, logger.FormatQuota(subscriptionAmount))
					if err := model.UpdateUserSubscription(tx, sub, "window_used_quota", "used_total_quota", "next_reset_at", "updated_time"); err != nil {
						return err
					}
					if err := model.CreateSubscriptionBill(tx, &bill); err != nil {
						return err
					}
				}
			}
		}

		walletAmount := amount - subscriptionAmount
		if walletAmount > 0 {
			var userQuota int
			if err := tx.Model(&model.User{}).Where("id = ?", s.userId).Select("quota").Scan(&userQuota).Error; err != nil {
				return err
			}
			if userQuota < walletAmount {
				return fmt.Errorf("套餐额度不足且用户额度不足, 套餐可用额度: %s, 用户剩余额度: %s, 需要余额额度: %s",
					logger.FormatQuota(subscriptionAmount),
					logger.FormatQuota(userQuota),
					logger.FormatQuota(walletAmount),
				)
			}
			if err := model.DecreaseUserQuotaTx(tx, s.userId, walletAmount); err != nil {
				return err
			}
		}

		subscriptionConsumed = subscriptionAmount
		walletConsumed = walletAmount
		return nil
	})
	if err != nil {
		return err
	}
	s.subscriptionConsumed += subscriptionConsumed
	s.walletConsumed += walletConsumed
	return nil
}

func (s *SubscriptionThenWalletFunding) refund(amount int) error {
	if amount <= 0 {
		return nil
	}
	if amount > s.subscriptionConsumed+s.walletConsumed {
		amount = s.subscriptionConsumed + s.walletConsumed
	}

	var refundedSubscription int
	var refundedWallet int
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		walletAmount := amount
		if walletAmount > s.walletConsumed {
			walletAmount = s.walletConsumed
		}
		if walletAmount > 0 {
			if err := model.IncreaseUserQuotaTx(tx, s.userId, walletAmount); err != nil {
				return err
			}
		}

		subscriptionAmount := amount - walletAmount
		if subscriptionAmount > s.subscriptionConsumed {
			subscriptionAmount = s.subscriptionConsumed
		}
		if subscriptionAmount > 0 {
			sub, err := model.GetActiveUserSubscriptionTx(tx, s.userId, now, true)
			if err != nil {
				return err
			}
			if sub == nil || sub.Id != s.subscriptionId {
				return fmt.Errorf("当前没有可用套餐")
			}
			bill := sub.RefundQuota(subscriptionAmount)
			sub.UpdatedTime = now
			bill.ChannelId = s.getChannelId()
			bill.ModelName = s.getModelName()
			bill.RequestId = s.requestId
			bill.Event = common.SubscriptionBillEventRefund
			bill.Content = fmt.Sprintf("套餐优先扣费返还 %s", logger.FormatQuota(subscriptionAmount))
			if err := model.UpdateUserSubscription(tx, sub, "window_used_quota", "used_total_quota", "next_reset_at", "updated_time"); err != nil {
				return err
			}
			if err := model.CreateSubscriptionBill(tx, &bill); err != nil {
				return err
			}
		}

		refundedWallet = walletAmount
		refundedSubscription = subscriptionAmount
		return nil
	})
	if err != nil {
		return err
	}
	s.walletConsumed -= refundedWallet
	s.subscriptionConsumed -= refundedSubscription
	return nil
}

func (s *SubscriptionThenWalletFunding) getChannelId() int {
	if s.relayInfo != nil && s.relayInfo.ChannelMeta != nil {
		return s.relayInfo.ChannelId
	}
	return 0
}

func (s *SubscriptionThenWalletFunding) getModelName() string {
	if s.relayInfo != nil {
		return s.relayInfo.OriginModelName
	}
	return ""
}
