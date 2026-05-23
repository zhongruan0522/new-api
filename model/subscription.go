package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"gorm.io/gorm"
)

var ErrNoActiveSubscription = errors.New("no active subscription")

// SubscriptionPlan defines a sellable user subscription package.
type SubscriptionPlan struct {
	Id                 int     `json:"id" gorm:"primaryKey"`
	Name               string  `json:"name" gorm:"uniqueIndex;type:varchar(64);not null"`
	Description        string  `json:"description" gorm:"type:text"`
	DurationCount      int     `json:"duration_count" gorm:"not null"`
	DurationUnit       string  `json:"duration_unit" gorm:"type:varchar(16);not null"`
	Price              float64 `json:"price" gorm:"not null"`
	TotalQuota         int     `json:"total_quota" gorm:"not null"`
	ResetQuota         int     `json:"reset_quota" gorm:"not null"`
	ResetIntervalCount int     `json:"reset_interval_count" gorm:"not null"`
	ResetIntervalUnit  string  `json:"reset_interval_unit" gorm:"type:varchar(16);not null"`
	Enabled            bool    `json:"enabled" gorm:"default:true;index"`
	CreatedTime        int64   `json:"created_time" gorm:"not null"`
	UpdatedTime        int64   `json:"updated_time" gorm:"not null"`
}

// UserSubscription stores a user-owned subscription lifecycle and quota state.
type UserSubscription struct {
	Id                 int     `json:"id" gorm:"primaryKey"`
	UserId             int     `json:"user_id" gorm:"index;not null"`
	PlanId             int     `json:"plan_id" gorm:"index;not null"`
	PlanName           string  `json:"plan_name" gorm:"type:varchar(64);not null"`
	Price              float64 `json:"price" gorm:"not null"`
	Status             string  `json:"status" gorm:"type:varchar(16);index;not null"`
	StartsAt           int64   `json:"starts_at" gorm:"index;not null"`
	ExpiresAt          int64   `json:"expires_at" gorm:"index;not null"`
	NextResetAt        int64   `json:"next_reset_at" gorm:"not null"`
	DurationCount      int     `json:"duration_count" gorm:"not null"`
	DurationUnit       string  `json:"duration_unit" gorm:"type:varchar(16);not null"`
	TotalQuota         int     `json:"total_quota" gorm:"not null"`
	ResetQuota         int     `json:"reset_quota" gorm:"not null"`
	ResetIntervalCount int     `json:"reset_interval_count" gorm:"not null"`
	ResetIntervalUnit  string  `json:"reset_interval_unit" gorm:"type:varchar(16);not null"`
	WindowUsedQuota    int     `json:"window_used_quota" gorm:"not null"`
	UsedTotalQuota     int     `json:"used_total_quota" gorm:"not null"`
	CreatedTime        int64   `json:"created_time" gorm:"not null"`
	UpdatedTime        int64   `json:"updated_time" gorm:"not null"`
}

// SubscriptionOrder tracks purchase / renew / upgrade payments independent from wallet top-up orders.
type SubscriptionOrder struct {
	Id             int     `json:"id" gorm:"primaryKey"`
	UserId         int     `json:"user_id" gorm:"index;not null"`
	PlanId         int     `json:"plan_id" gorm:"index;not null"`
	SubscriptionId int     `json:"subscription_id" gorm:"index"`
	TradeNo        string  `json:"trade_no" gorm:"uniqueIndex;type:varchar(255);not null"`
	Action         string  `json:"action" gorm:"type:varchar(16);not null"`
	PaymentMethod  string  `json:"payment_method" gorm:"type:varchar(64);not null"`
	PaymentSource  string  `json:"payment_source" gorm:"type:varchar(32);not null"`
	Amount         float64 `json:"amount" gorm:"not null"`
	TargetPlanId   int     `json:"target_plan_id" gorm:"index"`
	Status         string  `json:"status" gorm:"type:varchar(16);index;not null"`
	Meta           string  `json:"meta" gorm:"type:text"`
	CreateTime     int64   `json:"create_time" gorm:"not null"`
	CompleteTime   int64   `json:"complete_time"`
}

// SubscriptionBill stores auditable quota changes for a subscription.
type SubscriptionBill struct {
	Id               int    `json:"id" gorm:"primaryKey"`
	UserId           int    `json:"user_id" gorm:"index;not null"`
	SubscriptionId   int    `json:"subscription_id" gorm:"index;not null"`
	OrderId          int    `json:"order_id" gorm:"index"`
	RequestId        string `json:"request_id" gorm:"type:varchar(128);index"`
	ModelName        string `json:"model_name" gorm:"type:varchar(255)"`
	ChannelId        int    `json:"channel_id" gorm:"index"`
	Event            string `json:"event" gorm:"type:varchar(32);index;not null"`
	Quota            int    `json:"quota" gorm:"not null"`
	BeforeWindowUsed int    `json:"before_window_used" gorm:"not null"`
	AfterWindowUsed  int    `json:"after_window_used" gorm:"not null"`
	BeforeTotalUsed  int    `json:"before_total_used" gorm:"not null"`
	AfterTotalUsed   int    `json:"after_total_used" gorm:"not null"`
	Content          string `json:"content" gorm:"type:text"`
	CreatedTime      int64  `json:"created_time" gorm:"index;not null"`
}

// SubscriptionRedemption defines a subscription exchange code.
type SubscriptionRedemption struct {
	Id           int            `json:"id" gorm:"primaryKey"`
	UserId       int            `json:"user_id" gorm:"index;not null"`
	PlanId       int            `json:"plan_id" gorm:"index;not null"`
	PlanName     string         `json:"plan_name" gorm:"type:varchar(64);not null"`
	Key          string         `json:"key" gorm:"uniqueIndex;type:varchar(64);not null"`
	Status       int            `json:"status" gorm:"index;default:1"`
	Name         string         `json:"name" gorm:"index"`
	CreatedTime  int64          `json:"created_time" gorm:"index;not null"`
	RedeemedTime int64          `json:"redeemed_time"`
	Count        int            `json:"count" gorm:"-:all"`
	UsedUserId   int            `json:"used_user_id" gorm:"index"`
	ExpiredTime  int64          `json:"expired_time" gorm:"index"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// SubscriptionQuotaSnapshot exposes current subscription quota state for API responses.
type SubscriptionQuotaSnapshot struct {
	NominalResetQuota int   `json:"nominal_reset_quota"`
	EffectiveResetCap int   `json:"effective_reset_cap"`
	WindowRemaining   int   `json:"window_remaining"`
	TotalRemaining    int   `json:"total_remaining"`
	NextResetAt       int64 `json:"next_reset_at"`
}

func normalizeSubscriptionUnit(unit string) string {
	return strings.ToLower(strings.TrimSpace(unit))
}

func isValidSubscriptionUnit(unit string) bool {
	switch normalizeSubscriptionUnit(unit) {
	case "hour", "day", "month":
		return true
	default:
		return false
	}
}

func addSubscriptionDuration(base time.Time, count int, unit string) time.Time {
	switch normalizeSubscriptionUnit(unit) {
	case "hour":
		return base.Add(time.Duration(count) * time.Hour)
	case "day":
		return base.AddDate(0, 0, count)
	case "month":
		return base.AddDate(0, count, 0)
	default:
		panic("invalid subscription unit")
	}
}

// Validate validates the persisted subscription plan contract.
func (plan *SubscriptionPlan) Validate() error {
	if strings.TrimSpace(plan.Name) == "" {
		return fmt.Errorf("套餐名称不能为空")
	}
	if plan.DurationCount <= 0 {
		return fmt.Errorf("套餐周期必须大于 0")
	}
	if !isValidSubscriptionUnit(plan.DurationUnit) {
		return fmt.Errorf("套餐周期单位不支持")
	}
	if plan.Price < 0 {
		return fmt.Errorf("套餐价格不能小于 0")
	}
	if plan.TotalQuota <= 0 {
		return fmt.Errorf("套餐总额度必须大于 0")
	}
	if plan.ResetQuota <= 0 {
		return fmt.Errorf("套餐窗口额度必须大于 0")
	}
	if plan.ResetIntervalCount <= 0 {
		return fmt.Errorf("套餐刷新时间必须大于 0")
	}
	if !isValidSubscriptionUnit(plan.ResetIntervalUnit) {
		return fmt.Errorf("套餐刷新单位不支持")
	}
	return nil
}

func (sub *UserSubscription) refreshWindow(now int64) bool {
	if sub.NextResetAt == 0 {
		sub.NextResetAt = addSubscriptionDuration(time.Unix(sub.StartsAt, 0), sub.ResetIntervalCount, sub.ResetIntervalUnit).Unix()
		return true
	}
	if now < sub.NextResetAt {
		return false
	}
	base := time.Unix(sub.NextResetAt, 0)
	for now >= sub.NextResetAt {
		base = addSubscriptionDuration(base, sub.ResetIntervalCount, sub.ResetIntervalUnit)
		sub.NextResetAt = base.Unix()
	}
	sub.WindowUsedQuota = 0
	return true
}

func (sub *UserSubscription) Snapshot(now int64) SubscriptionQuotaSnapshot {
	_ = sub.refreshWindow(now)
	totalRemaining := sub.TotalQuota - sub.UsedTotalQuota
	if totalRemaining < 0 {
		totalRemaining = 0
	}
	effectiveResetCap := sub.ResetQuota
	if effectiveResetCap > totalRemaining {
		effectiveResetCap = totalRemaining
	}
	windowRemaining := effectiveResetCap - sub.WindowUsedQuota
	if windowRemaining < 0 {
		windowRemaining = 0
	}
	return SubscriptionQuotaSnapshot{
		NominalResetQuota: sub.ResetQuota,
		EffectiveResetCap: effectiveResetCap,
		WindowRemaining:   windowRemaining,
		TotalRemaining:    totalRemaining,
		NextResetAt:       sub.NextResetAt,
	}
}

func (sub *UserSubscription) ConsumeQuota(quota int) SubscriptionBill {
	beforeWindow := sub.WindowUsedQuota
	beforeTotal := sub.UsedTotalQuota
	sub.WindowUsedQuota += quota
	sub.UsedTotalQuota += quota
	return SubscriptionBill{
		UserId:           sub.UserId,
		SubscriptionId:   sub.Id,
		Quota:            quota,
		BeforeWindowUsed: beforeWindow,
		AfterWindowUsed:  sub.WindowUsedQuota,
		BeforeTotalUsed:  beforeTotal,
		AfterTotalUsed:   sub.UsedTotalQuota,
		CreatedTime:      common.GetTimestamp(),
	}
}

func (sub *UserSubscription) RefundQuota(quota int) SubscriptionBill {
	beforeWindow := sub.WindowUsedQuota
	beforeTotal := sub.UsedTotalQuota
	sub.WindowUsedQuota -= quota
	if sub.WindowUsedQuota < 0 {
		sub.WindowUsedQuota = 0
	}
	sub.UsedTotalQuota -= quota
	if sub.UsedTotalQuota < 0 {
		sub.UsedTotalQuota = 0
	}
	return SubscriptionBill{
		UserId:           sub.UserId,
		SubscriptionId:   sub.Id,
		Quota:            quota,
		BeforeWindowUsed: beforeWindow,
		AfterWindowUsed:  sub.WindowUsedQuota,
		BeforeTotalUsed:  beforeTotal,
		AfterTotalUsed:   sub.UsedTotalQuota,
		CreatedTime:      common.GetTimestamp(),
	}
}

func CreateSubscriptionPlan(plan *SubscriptionPlan) error {
	now := common.GetTimestamp()
	plan.CreatedTime = now
	plan.UpdatedTime = now
	return DB.Create(plan).Error
}

func UpdateSubscriptionPlan(plan *SubscriptionPlan) error {
	plan.UpdatedTime = common.GetTimestamp()
	return DB.Model(plan).Select(
		"name",
		"description",
		"duration_count",
		"duration_unit",
		"price",
		"total_quota",
		"reset_quota",
		"reset_interval_count",
		"reset_interval_unit",
		"enabled",
		"updated_time",
	).Updates(plan).Error
}

func GetSubscriptionPlanById(id int) (*SubscriptionPlan, error) {
	var plan SubscriptionPlan
	err := DB.Where("id = ?", id).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func GetEnabledSubscriptionPlans() ([]*SubscriptionPlan, error) {
	var plans []*SubscriptionPlan
	err := DB.Where("enabled = ?", true).Order("id asc").Find(&plans).Error
	return plans, err
}

func GetAllSubscriptionPlans(pageInfo *common.PageInfo) ([]*SubscriptionPlan, int64, error) {
	var plans []*SubscriptionPlan
	var total int64
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	if err := tx.Model(&SubscriptionPlan{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&plans).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return plans, total, nil
}

func GetSubscriptionOrderByTradeNo(tradeNo string) (*SubscriptionOrder, error) {
	var order SubscriptionOrder
	err := DB.Where("trade_no = ?", tradeNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func CreateSubscriptionOrder(order *SubscriptionOrder) error {
	order.CreateTime = common.GetTimestamp()
	return DB.Create(order).Error
}

func ListUserSubscriptionOrders(userId int, pageInfo *common.PageInfo) ([]*SubscriptionOrder, int64, error) {
	var orders []*SubscriptionOrder
	var total int64
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	if err := tx.Model(&SubscriptionOrder{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Where("user_id = ?", userId).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func GetUserSubscriptions(userId int) ([]*UserSubscription, error) {
	var subs []*UserSubscription
	err := DB.Where("user_id = ?", userId).Order("starts_at desc, id desc").Find(&subs).Error
	return subs, err
}

func getNextPendingSubscriptionTx(tx *gorm.DB, userId int, now int64, forUpdate bool) (*UserSubscription, error) {
	var pending UserSubscription
	query := tx.Where("user_id = ? AND status = ? AND starts_at <= ?", userId, common.SubscriptionStatusPending, now).Order("starts_at asc, id asc")
	if forUpdate && !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	err := query.First(&pending).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pending, nil
}

func GetActiveUserSubscriptionTx(tx *gorm.DB, userId int, now int64, forUpdate bool) (*UserSubscription, error) {
	var sub UserSubscription
	query := tx.Where("user_id = ? AND status = ?", userId, common.SubscriptionStatusActive).Order("id desc")
	if forUpdate && !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	err := query.First(&sub).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		if now >= sub.ExpiresAt {
			sub.Status = common.SubscriptionStatusExpired
			sub.UpdatedTime = now
			if err = tx.Model(&sub).Select("status", "updated_time").Updates(&sub).Error; err != nil {
				return nil, err
			}
		} else {
			if sub.refreshWindow(now) {
				sub.UpdatedTime = now
				if err = tx.Model(&sub).Select("window_used_quota", "next_reset_at", "updated_time").Updates(&sub).Error; err != nil {
					return nil, err
				}
			}
			return &sub, nil
		}
	}

	pending, err := getNextPendingSubscriptionTx(tx, userId, now, forUpdate)
	if err != nil {
		return nil, err
	}
	if pending == nil {
		return nil, nil
	}
	pending.Status = common.SubscriptionStatusActive
	if pending.NextResetAt == 0 {
		pending.NextResetAt = addSubscriptionDuration(time.Unix(pending.StartsAt, 0), pending.ResetIntervalCount, pending.ResetIntervalUnit).Unix()
	}
	pending.UpdatedTime = now
	if err = tx.Model(pending).Select("status", "next_reset_at", "updated_time").Updates(pending).Error; err != nil {
		return nil, err
	}
	return pending, nil
}

func GetUserActiveSubscription(userId int) (*UserSubscription, error) {
	var result *UserSubscription
	err := DB.Transaction(func(tx *gorm.DB) error {
		sub, err := GetActiveUserSubscriptionTx(tx, userId, common.GetTimestamp(), false)
		if err != nil {
			return err
		}
		result = sub
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNoActiveSubscription
	}
	return result, nil
}

func CreateUserSubscription(tx *gorm.DB, sub *UserSubscription) error {
	if tx == nil {
		tx = DB
	}
	return tx.Create(sub).Error
}

func UpdateUserSubscription(tx *gorm.DB, sub *UserSubscription, fields ...string) error {
	if tx == nil {
		tx = DB
	}
	if len(fields) == 0 {
		return tx.Model(sub).Updates(sub).Error
	}
	return tx.Model(sub).Select(fields).Updates(sub).Error
}

func CreateSubscriptionBill(tx *gorm.DB, bill *SubscriptionBill) error {
	if tx == nil {
		tx = DB
	}
	return tx.Create(bill).Error
}

func CreateSubscriptionRedemption(redemption *SubscriptionRedemption) error {
	return DB.Create(redemption).Error
}

func GetSubscriptionRedemptionById(id int) (*SubscriptionRedemption, error) {
	var redemption SubscriptionRedemption
	err := DB.Where("id = ?", id).First(&redemption).Error
	if err != nil {
		return nil, err
	}
	return &redemption, nil
}

func GetAllSubscriptionRedemptions(startIdx int, num int) ([]*SubscriptionRedemption, int64, error) {
	var redemptions []*SubscriptionRedemption
	var total int64
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	if err := tx.Model(&SubscriptionRedemption{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return redemptions, total, nil
}

func SearchSubscriptionRedemptions(keyword string, startIdx int, num int) ([]*SubscriptionRedemption, int64, error) {
	var redemptions []*SubscriptionRedemption
	var total int64
	query := DB.Model(&SubscriptionRedemption{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR "+commonKeyCol+" LIKE ? OR plan_name LIKE ?", like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error; err != nil {
		return nil, 0, err
	}
	return redemptions, total, nil
}

func UpdateSubscriptionRedemption(redemption *SubscriptionRedemption) error {
	return DB.Model(redemption).Select("name", "status", "expired_time", "plan_id", "plan_name").Updates(redemption).Error
}

func DeleteSubscriptionRedemptionById(id int) error {
	return DB.Delete(&SubscriptionRedemption{}, "id = ?", id).Error
}

func DeleteInvalidSubscriptionRedemptions() (int64, error) {
	result := DB.Where("status != ? OR (expired_time != 0 AND expired_time < ?)", common.RedemptionCodeStatusEnabled, common.GetTimestamp()).Delete(&SubscriptionRedemption{})
	return result.RowsAffected, result.Error
}

func ConsumeSubscriptionRedemption(key string, userId int) (*SubscriptionRedemption, error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	var redemption SubscriptionRedemption
	err := DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Set("gorm:query_option", "FOR UPDATE").Where(commonKeyCol+" = ?", key)
		if err := query.First(&redemption).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("无效的兑换码")
			}
			return err
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		redemption.RedeemedTime = common.GetTimestamp()
		return tx.Model(&redemption).Select("status", "used_user_id", "redeemed_time").Updates(&redemption).Error
	})
	if err != nil {
		return nil, err
	}
	return &redemption, nil
}

func ConsumeSubscriptionRedemptionTx(tx *gorm.DB, key string, userId int) (*SubscriptionRedemption, error) {
	if key == "" {
		return nil, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return nil, errors.New("无效的 user id")
	}
	var redemption SubscriptionRedemption
	query := tx.Where(commonKeyCol+" = ?", key)
	if !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	if err := query.First(&redemption).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的兑换码")
		}
		return nil, err
	}
	if redemption.Status != common.RedemptionCodeStatusEnabled {
		return nil, errors.New("该兑换码已被使用")
	}
	if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
		return nil, errors.New("该兑换码已过期")
	}
	redemption.Status = common.RedemptionCodeStatusUsed
	redemption.UsedUserId = userId
	redemption.RedeemedTime = common.GetTimestamp()
	if err := tx.Model(&redemption).Select("status", "used_user_id", "redeemed_time").Updates(&redemption).Error; err != nil {
		return nil, err
	}
	return &redemption, nil
}

func GetSubscriptionBillsBySubscriptionId(subscriptionId int, limit int) ([]*SubscriptionBill, error) {
	var bills []*SubscriptionBill
	query := DB.Where("subscription_id = ?", subscriptionId).Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&bills).Error
	return bills, err
}

func CloneSubscriptionFromPlan(userId int, plan *SubscriptionPlan, status string, startsAt int64, usedTotalQuota int) *UserSubscription {
	starts := time.Unix(startsAt, 0)
	return &UserSubscription{
		UserId:             userId,
		PlanId:             plan.Id,
		PlanName:           plan.Name,
		Price:              plan.Price,
		Status:             status,
		StartsAt:           startsAt,
		ExpiresAt:          addSubscriptionDuration(starts, plan.DurationCount, plan.DurationUnit).Unix(),
		NextResetAt:        addSubscriptionDuration(starts, plan.ResetIntervalCount, plan.ResetIntervalUnit).Unix(),
		DurationCount:      plan.DurationCount,
		DurationUnit:       normalizeSubscriptionUnit(plan.DurationUnit),
		TotalQuota:         plan.TotalQuota,
		ResetQuota:         plan.ResetQuota,
		ResetIntervalCount: plan.ResetIntervalCount,
		ResetIntervalUnit:  normalizeSubscriptionUnit(plan.ResetIntervalUnit),
		UsedTotalQuota:     usedTotalQuota,
		CreatedTime:        common.GetTimestamp(),
		UpdatedTime:        common.GetTimestamp(),
	}
}

func CalculateSubscriptionUpgradeAmount(current *UserSubscription, target *SubscriptionPlan, now int64) float64 {
	if current == nil {
		return target.Price
	}
	if now >= current.ExpiresAt {
		return target.Price
	}
	totalDuration := current.ExpiresAt - current.StartsAt
	if totalDuration <= 0 {
		return target.Price
	}
	remaining := float64(current.ExpiresAt-now) / float64(totalDuration)
	remainingValue := current.Price * math.Max(remaining, 0)
	amount := target.Price - remainingValue
	if amount < 0 {
		return 0
	}
	return amount
}

func SortSubscriptionsForDisplay(subs []*UserSubscription) {
	sort.Slice(subs, func(i, j int) bool {
		if subs[i].Status == subs[j].Status {
			if subs[i].StartsAt == subs[j].StartsAt {
				return subs[i].Id > subs[j].Id
			}
			return subs[i].StartsAt > subs[j].StartsAt
		}
		leftActive := subs[i].Status == common.SubscriptionStatusActive
		rightActive := subs[j].Status == common.SubscriptionStatusActive
		if leftActive != rightActive {
			return leftActive
		}
		leftPending := subs[i].Status == common.SubscriptionStatusPending
		rightPending := subs[j].Status == common.SubscriptionStatusPending
		if leftPending != rightPending {
			return leftPending
		}
		return subs[i].Id > subs[j].Id
	})
}
