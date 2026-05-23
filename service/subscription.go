package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SubscriptionSummary struct {
	Plans         []*model.SubscriptionPlan `json:"plans"`
	Active        *model.UserSubscription   `json:"active,omitempty"`
	Pending       *model.UserSubscription   `json:"pending,omitempty"`
	Subscriptions []*model.UserSubscription `json:"subscriptions"`
	PaymentMode   string                    `json:"payment_mode"`
	PayMethods    []map[string]string       `json:"pay_methods"`
}

func getPendingSubscriptionTx(tx *gorm.DB, userId int, forUpdate bool) (*model.UserSubscription, error) {
	var sub model.UserSubscription
	query := tx.Where("user_id = ? AND status = ?", userId, common.SubscriptionStatusPending).Order("starts_at asc, id asc")
	if forUpdate && !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	err := query.First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func cancelPendingSubscriptionsTx(tx *gorm.DB, userId int) error {
	return tx.Model(&model.UserSubscription{}).
		Where("user_id = ? AND status = ?", userId, common.SubscriptionStatusPending).
		Updates(map[string]any{
			"status":       common.SubscriptionStatusCancelled,
			"updated_time": common.GetTimestamp(),
		}).Error
}

func BuildSubscriptionSummary(userId int) (*SubscriptionSummary, error) {
	plans, err := model.GetEnabledSubscriptionPlans()
	if err != nil {
		return nil, err
	}
	var subs []*model.UserSubscription
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		_, activeErr := model.GetActiveUserSubscriptionTx(tx, userId, common.GetTimestamp(), false)
		if activeErr != nil && activeErr != model.ErrNoActiveSubscription {
			return activeErr
		}
		return tx.Where("user_id = ?", userId).Order("starts_at desc, id desc").Find(&subs).Error
	})
	if err != nil {
		return nil, err
	}
	model.SortSubscriptionsForDisplay(subs)
	summary := &SubscriptionSummary{
		Plans: plans,
		Subscriptions: subs,
		PaymentMode: operation_setting.NormalizeSubscriptionPaymentMode(operation_setting.GetSubscriptionSetting().PaymentMode),
		PayMethods:  GetSubscriptionCashPaymentMethods(),
	}
	for _, sub := range subs {
		switch sub.Status {
		case common.SubscriptionStatusActive:
			if summary.Active == nil {
				summary.Active = sub
			}
		case common.SubscriptionStatusPending:
			if summary.Pending == nil {
				summary.Pending = sub
			}
		}
	}
	return summary, nil
}

func GetSubscriptionCashPaymentMethods() []map[string]string {
	methods := make([]map[string]string, 0, len(operation_setting.PayMethods)+1)
	for _, method := range operation_setting.PayMethods {
		methods = append(methods, method)
	}
	if setting.StripeApiSecret != "" && setting.StripeWebhookSecret != "" && setting.StripePriceId != "" {
		methods = append(methods, map[string]string{
			"name":  "Stripe",
			"type":  "stripe",
			"color": "rgba(var(--semi-purple-5), 1)",
		})
	}
	return methods
}

func getSubscriptionPlanForOrder(planId int) (*model.SubscriptionPlan, error) {
	plan, err := model.GetSubscriptionPlanById(planId)
	if err != nil {
		return nil, err
	}
	if !plan.Enabled {
		return nil, fmt.Errorf("套餐未启用")
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	return plan, nil
}

func getSubscriptionAmountQuota(amount float64) int {
	return int(math.Round(amount * common.QuotaPerUnit))
}

func validateOrderAction(action string, allowAssign bool) error {
	switch action {
	case common.SubscriptionOrderActionPurchase,
		common.SubscriptionOrderActionRenew,
		common.SubscriptionOrderActionUpgrade:
		return nil
	case common.SubscriptionOrderActionAssign:
		if allowAssign {
			return nil
		}
		return fmt.Errorf("不支持的套餐操作")
	default:
		return fmt.Errorf("不支持的套餐操作")
	}
}

func prepareSubscriptionOrderTx(tx *gorm.DB, userId int, plan *model.SubscriptionPlan, action string, allowAssign bool) (float64, *model.UserSubscription, *model.UserSubscription, error) {
	if err := validateOrderAction(action, allowAssign); err != nil {
		return 0, nil, nil, err
	}
	now := common.GetTimestamp()
	active, err := model.GetActiveUserSubscriptionTx(tx, userId, now, true)
	if err != nil && err != model.ErrNoActiveSubscription {
		return 0, nil, nil, err
	}
	pending, err := getPendingSubscriptionTx(tx, userId, true)
	if err != nil {
		return 0, nil, nil, err
	}
	amount := plan.Price
	switch action {
	case common.SubscriptionOrderActionPurchase:
		if active != nil {
			return 0, nil, nil, fmt.Errorf("已有生效套餐，请使用续订或升级")
		}
		if pending != nil {
			return 0, nil, nil, fmt.Errorf("已有待生效套餐")
		}
	case common.SubscriptionOrderActionRenew:
		if active == nil {
			return 0, nil, nil, fmt.Errorf("当前没有可续订的套餐")
		}
		if pending != nil {
			return 0, nil, nil, fmt.Errorf("已存在待生效套餐")
		}
	case common.SubscriptionOrderActionUpgrade:
		if active == nil {
			return 0, nil, nil, fmt.Errorf("当前没有可升级的套餐")
		}
		if pending != nil {
			return 0, nil, nil, fmt.Errorf("已存在待生效套餐，请先处理后再升级")
		}
		if plan.Id == active.PlanId || plan.Price <= active.Price {
			return 0, nil, nil, fmt.Errorf("升级目标必须高于当前套餐")
		}
		amount = model.CalculateSubscriptionUpgradeAmount(active, plan, now)
	case common.SubscriptionOrderActionAssign:
		if pending != nil {
			return 0, nil, nil, fmt.Errorf("已存在待生效套餐")
		}
		amount = 0
	}
	return amount, active, pending, nil
}

func CalculateSubscriptionOrderAmount(userId int, planId int, action string) (float64, error) {
	plan, err := getSubscriptionPlanForOrder(planId)
	if err != nil {
		return 0, err
	}
	if err := validateOrderAction(action, false); err != nil {
		return 0, err
	}
	var amount float64
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		computed, _, _, prepErr := prepareSubscriptionOrderTx(tx, userId, plan, action, false)
		if prepErr != nil {
			return prepErr
		}
		amount = computed
		return nil
	})
	return amount, err
}

func createSubscriptionOrderTx(tx *gorm.DB, userId int, plan *model.SubscriptionPlan, action string, paymentSource string, paymentMethod string, tradeNo string, allowAssign bool, autoComplete bool) (*model.SubscriptionOrder, error) {
	amount, active, _, err := prepareSubscriptionOrderTx(tx, userId, plan, action, allowAssign)
	if err != nil {
		return nil, err
	}
	if !autoComplete && paymentSource == common.SubscriptionPaymentModeCash {
		var pendingOrder model.SubscriptionOrder
		query := tx.Where("user_id = ? AND status = ? AND payment_source = ?", userId, common.TopUpStatusPending, common.SubscriptionPaymentModeCash)
		if !common.UsingSQLite {
			query = query.Set("gorm:query_option", "FOR UPDATE")
		}
		if err := query.Order("id desc").First(&pendingOrder).Error; err == nil {
			return nil, fmt.Errorf("存在未完成的套餐支付订单，请先完成或等待其过期")
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	now := common.GetTimestamp()
	status := common.TopUpStatusPending
	if autoComplete {
		status = common.TopUpStatusSuccess
	}
	order := &model.SubscriptionOrder{
		UserId:        userId,
		PlanId:        plan.Id,
		TradeNo:       tradeNo,
		Action:        action,
		PaymentMethod: paymentMethod,
		PaymentSource: paymentSource,
		Amount:        amount,
		Status:        status,
		CreateTime:    now,
	}
	if active != nil {
		order.TargetPlanId = active.PlanId
		order.SubscriptionId = active.Id
	}
	if err := tx.Create(order).Error; err != nil {
		return nil, err
	}
	if paymentSource == common.SubscriptionPaymentModeBalance {
		quotaCost := getSubscriptionAmountQuota(amount)
		result := tx.Model(&model.User{}).
			Where("id = ? AND quota >= ?", userId, quotaCost).
			Update("quota", gorm.Expr("quota - ?", quotaCost))
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return nil, fmt.Errorf("余额不足，无法购买套餐")
		}
	}
	if autoComplete {
		if err := completeSubscriptionOrderTx(tx, order); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func CreateSubscriptionOrder(userId int, planId int, action string, paymentSource string, paymentMethod string) (*model.SubscriptionOrder, error) {
	plan, err := getSubscriptionPlanForOrder(planId)
	if err != nil {
		return nil, err
	}
	if err := validateOrderAction(action, false); err != nil {
		return nil, err
	}
	var order *model.SubscriptionOrder
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		created, err := createSubscriptionOrderTx(tx, userId, plan, action, paymentSource, paymentMethod, BuildSubscriptionTradeNo(userId), false, true)
		order = created
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func CreatePendingSubscriptionOrder(userId int, planId int, action string, paymentSource string, paymentMethod string, tradeNo string) (*model.SubscriptionOrder, error) {
	plan, err := getSubscriptionPlanForOrder(planId)
	if err != nil {
		return nil, err
	}
	if err := validateOrderAction(action, false); err != nil {
		return nil, err
	}
	var order *model.SubscriptionOrder
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		created, createErr := createSubscriptionOrderTx(tx, userId, plan, action, paymentSource, paymentMethod, tradeNo, false, false)
		order = created
		return createErr
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func completeSubscriptionOrderTx(tx *gorm.DB, order *model.SubscriptionOrder) error {
	now := common.GetTimestamp()
	if order.Status != common.TopUpStatusPending && order.Status != common.TopUpStatusSuccess {
		return fmt.Errorf("套餐订单状态错误")
	}
	plan, err := model.GetSubscriptionPlanById(order.PlanId)
	if err != nil {
		return err
	}
	active, err := model.GetActiveUserSubscriptionTx(tx, order.UserId, now, true)
	if err != nil && err != model.ErrNoActiveSubscription {
		return err
	}
	pending, err := getPendingSubscriptionTx(tx, order.UserId, true)
	if err != nil {
		return err
	}
	var resultSubscription *model.UserSubscription
	switch order.Action {
	case common.SubscriptionOrderActionPurchase:
		if active != nil {
			return fmt.Errorf("已有生效套餐，请使用续订或升级")
		}
		if pending != nil {
			return fmt.Errorf("已有待生效套餐")
		}
		resultSubscription = model.CloneSubscriptionFromPlan(order.UserId, plan, common.SubscriptionStatusActive, now, 0)
		if err := model.CreateUserSubscription(tx, resultSubscription); err != nil {
			return err
		}
	case common.SubscriptionOrderActionRenew:
		if active == nil {
			return fmt.Errorf("当前没有可续订的套餐")
		}
		if pending != nil {
			return fmt.Errorf("已存在待生效套餐")
		}
		resultSubscription = model.CloneSubscriptionFromPlan(order.UserId, plan, common.SubscriptionStatusPending, active.ExpiresAt, 0)
		if err := model.CreateUserSubscription(tx, resultSubscription); err != nil {
			return err
		}
	case common.SubscriptionOrderActionUpgrade:
		if active == nil {
			return fmt.Errorf("当前没有可升级的套餐")
		}
		if err := cancelPendingSubscriptionsTx(tx, order.UserId); err != nil {
			return err
		}
		active.Status = common.SubscriptionStatusReplaced
		active.UpdatedTime = now
		if err := model.UpdateUserSubscription(tx, active, "status", "updated_time"); err != nil {
			return err
		}
		resultSubscription = model.CloneSubscriptionFromPlan(order.UserId, plan, common.SubscriptionStatusActive, now, active.UsedTotalQuota)
		if err := model.CreateUserSubscription(tx, resultSubscription); err != nil {
			return err
		}
	case common.SubscriptionOrderActionAssign:
		if pending != nil {
			return fmt.Errorf("已存在待生效套餐")
		}
		status := common.SubscriptionStatusActive
		startsAt := now
		if active != nil {
			status = common.SubscriptionStatusPending
			startsAt = active.ExpiresAt
		}
		resultSubscription = model.CloneSubscriptionFromPlan(order.UserId, plan, status, startsAt, 0)
		if err := model.CreateUserSubscription(tx, resultSubscription); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的套餐操作")
	}
	order.SubscriptionId = resultSubscription.Id
	order.Status = common.TopUpStatusSuccess
	order.CompleteTime = now
	if err := tx.Model(order).Select("subscription_id", "status", "complete_time").Updates(order).Error; err != nil {
		return err
	}
	content := fmt.Sprintf("套餐订单完成：%s -> %s", order.Action, resultSubscription.PlanName)
	if err := model.CreateSubscriptionBill(tx, &model.SubscriptionBill{
		UserId:           order.UserId,
		SubscriptionId:   resultSubscription.Id,
		OrderId:          order.Id,
		Event:            common.SubscriptionBillEventAssign,
		Quota:            0,
		BeforeWindowUsed: resultSubscription.WindowUsedQuota,
		AfterWindowUsed:  resultSubscription.WindowUsedQuota,
		BeforeTotalUsed:  resultSubscription.UsedTotalQuota,
		AfterTotalUsed:   resultSubscription.UsedTotalQuota,
		Content:          content,
		CreatedTime:      now,
	}); err != nil {
		return err
	}
	model.RecordLog(order.UserId, model.LogTypeManage, content)
	return nil
}

func CompleteSubscriptionOrder(tradeNo string) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var order model.SubscriptionOrder
		query := tx.Where("trade_no = ?", tradeNo)
		if !common.UsingSQLite {
			query = query.Set("gorm:query_option", "FOR UPDATE")
		}
		if err := query.First(&order).Error; err != nil {
			return err
		}
		if order.Status == common.TopUpStatusSuccess {
			return nil
		}
		if order.Status != common.TopUpStatusPending {
			return fmt.Errorf("套餐订单状态错误")
		}
		return completeSubscriptionOrderTx(tx, &order)
	})
}

func MarkSubscriptionOrderStatus(tradeNo string, status string) error {
	return model.DB.Model(&model.SubscriptionOrder{}).
		Where("trade_no = ? AND status = ?", tradeNo, common.TopUpStatusPending).
		Updates(map[string]any{"status": status}).Error
}

func AssignSubscriptionPlanToUser(userId int, planId int) error {
	plan, err := getSubscriptionPlanForOrder(planId)
	if err != nil {
		return err
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		_, createErr := createSubscriptionOrderTx(tx, userId, plan, common.SubscriptionOrderActionAssign, common.SubscriptionPaymentModeBalance, common.SubscriptionPaymentModeBalance, BuildSubscriptionTradeNo(userId), true, true)
		return createErr
	})
	return err
}

func ChangeUserSubscriptionPlan(userId int, planId int) error {
	plan, err := getSubscriptionPlanForOrder(planId)
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	return model.DB.Transaction(func(tx *gorm.DB) error {
		active, err := model.GetActiveUserSubscriptionTx(tx, userId, now, true)
		if err != nil && err != model.ErrNoActiveSubscription {
			return err
		}
		if active == nil {
			return fmt.Errorf("当前没有可变更的套餐")
		}
		if err := cancelPendingSubscriptionsTx(tx, userId); err != nil {
			return err
		}
		active.Status = common.SubscriptionStatusReplaced
		active.UpdatedTime = now
		if err := model.UpdateUserSubscription(tx, active, "status", "updated_time"); err != nil {
			return err
		}
		nextSubscription := model.CloneSubscriptionFromPlan(userId, plan, common.SubscriptionStatusActive, now, active.UsedTotalQuota)
		if err := model.CreateUserSubscription(tx, nextSubscription); err != nil {
			return err
		}
		order := &model.SubscriptionOrder{
			UserId:         userId,
			PlanId:         plan.Id,
			SubscriptionId: nextSubscription.Id,
			TradeNo:        BuildSubscriptionTradeNo(userId),
			Action:         common.SubscriptionOrderActionChange,
			PaymentMethod:  "admin",
			PaymentSource:  "admin",
			Amount:         0,
			TargetPlanId:   active.PlanId,
			Status:         common.TopUpStatusSuccess,
			CreateTime:     now,
			CompleteTime:   now,
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		content := fmt.Sprintf("管理员变更套餐：%s -> %s", active.PlanName, nextSubscription.PlanName)
		if err := model.CreateSubscriptionBill(tx, &model.SubscriptionBill{
			UserId:           userId,
			SubscriptionId:   nextSubscription.Id,
			OrderId:          order.Id,
			Event:            common.SubscriptionBillEventChange,
			Quota:            0,
			BeforeWindowUsed: nextSubscription.WindowUsedQuota,
			AfterWindowUsed:  nextSubscription.WindowUsedQuota,
			BeforeTotalUsed:  nextSubscription.UsedTotalQuota,
			AfterTotalUsed:   nextSubscription.UsedTotalQuota,
			Content:          content,
			CreatedTime:      now,
		}); err != nil {
			return err
		}
		model.RecordLog(userId, model.LogTypeManage, content)
		return nil
	})
}

func RemoveUserSubscription(userId int, subscriptionId int) error {
	now := common.GetTimestamp()
	return model.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&model.UserSubscription{}).Where("user_id = ?", userId)
		if subscriptionId > 0 {
			query = query.Where("id = ?", subscriptionId)
		} else {
			query = query.Where("status IN ?", []string{common.SubscriptionStatusActive, common.SubscriptionStatusPending})
		}
		result := query.Updates(map[string]any{
			"status":       common.SubscriptionStatusCancelled,
			"updated_time": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("未找到可移除的套餐")
		}
		return nil
	})
}

func RedeemSubscriptionPlan(key string, userId int) (*model.SubscriptionRedemption, error) {
	var redemption *model.SubscriptionRedemption
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		current, consumeErr := model.ConsumeSubscriptionRedemptionTx(tx, key, userId)
		if consumeErr != nil {
			return consumeErr
		}
		redemption = current
		plan, planErr := getSubscriptionPlanForOrder(current.PlanId)
		if planErr != nil {
			return planErr
		}
		_, assignErr := createSubscriptionOrderTx(tx, userId, plan, common.SubscriptionOrderActionAssign, common.SubscriptionPaymentModeBalance, common.SubscriptionPaymentModeBalance, BuildSubscriptionTradeNo(userId), true, true)
		return assignErr
	})
	if err != nil {
		return nil, err
	}
	model.RecordLog(userId, model.LogTypeManage, fmt.Sprintf("通过套餐兑换码领取套餐 %s", redemption.PlanName))
	return redemption, nil
}

func BuildSubscriptionTradeNo(userId int) string {
	return fmt.Sprintf("SUB%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
}

func ShouldUseSubscription(paymentMode string) bool {
	return strings.TrimSpace(paymentMode) == common.SubscriptionPaymentModeBalance
}

func WriteActiveSubscriptionContext(c *gin.Context, userId int) error {
	sub, err := model.GetUserActiveSubscription(userId)
	if err != nil {
		if err == model.ErrNoActiveSubscription {
			common.SetContextKey(c, constant.ContextKeySubscriptionActive, false)
			common.SetContextKey(c, constant.ContextKeySubscriptionId, 0)
			common.SetContextKey(c, constant.ContextKeySubscriptionPlanId, 0)
			common.SetContextKey(c, constant.ContextKeySubscriptionBillingRatio, 1.0)
			return nil
		}
		return err
	}
	common.SetContextKey(c, constant.ContextKeySubscriptionActive, true)
	common.SetContextKey(c, constant.ContextKeySubscriptionId, sub.Id)
	common.SetContextKey(c, constant.ContextKeySubscriptionPlanId, sub.PlanId)
	common.SetContextKey(c, constant.ContextKeySubscriptionBillingRatio, 1.0)
	return nil
}
