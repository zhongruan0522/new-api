package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/setting"
	"github.com/zhongruan0522/new-api/setting/operation_setting"
	"github.com/zhongruan0522/new-api/setting/system_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"gorm.io/gorm"
)

type SubscriptionPlanAmountRequest struct {
	PlanId int    `json:"plan_id"`
	Action string `json:"action"`
}

type SubscriptionCashPayRequest struct {
	PlanId        int    `json:"plan_id"`
	Action        string `json:"action"`
	PaymentMethod string `json:"payment_method"`
	SuccessURL    string `json:"success_url,omitempty"`
	CancelURL     string `json:"cancel_url,omitempty"`
}

type SubscriptionBalancePurchaseRequest struct {
	PlanId int    `json:"plan_id"`
	Action string `json:"action"`
}

type SubscriptionAssignRequest struct {
	PlanId int `json:"plan_id"`
}

func GetSubscriptionSummary(c *gin.Context) {
	userId := c.GetInt("id")
	summary, err := service.BuildSubscriptionSummary(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetUserSubscriptionSummary(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	summary, err := service.BuildSubscriptionSummary(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetSubscriptionOrders(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	filter := getOrderFilter(c)
	orders, total, err := model.QueryUserSubscriptionOrders(userId, filter, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orders)
	common.ApiSuccess(c, pageInfo)
}

func GetSubscriptionAmount(c *gin.Context) {
	var req SubscriptionPlanAmountRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	amount, err := service.CalculateSubscriptionOrderAmount(c.GetInt("id"), req.PlanId, req.Action)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"amount": amount})
}

func PurchaseSubscriptionWithBalance(c *gin.Context) {
	mode := operation_setting.NormalizeSubscriptionPaymentMode(operation_setting.GetSubscriptionSetting().PaymentMode)
	if mode == common.SubscriptionPaymentModeCash {
		common.ApiErrorMsg(c, "当前未开启余额支付")
		return
	}
	var req SubscriptionBalancePurchaseRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	order, err := service.CreateSubscriptionOrder(c.GetInt("id"), req.PlanId, req.Action, common.SubscriptionPaymentModeBalance, common.SubscriptionPaymentModeBalance)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func RequestSubscriptionEpay(c *gin.Context) {
	if operation_setting.NormalizeSubscriptionPaymentMode(operation_setting.GetSubscriptionSetting().PaymentMode) == common.SubscriptionPaymentModeBalance {
		common.ApiErrorMsg(c, "当前未开启现金支付")
		return
	}
	var req SubscriptionCashPayRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}
	amount, err := service.CalculateSubscriptionOrderAmount(c.GetInt("id"), req.PlanId, req.Action)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tradeNo := service.BuildSubscriptionTradeNo(c.GetInt("id"))
	client := GetEpayClient()
	if client == nil {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}
	callBackAddress := service.GetCallbackAddress()
	returnURL, _ := url.Parse(system_setting.ServerAddress + "/console/subscription")
	notifyURL, _ := url.Parse(callBackAddress + "/api/user/epay/notify")
	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("SUB-%d", req.PlanId),
		Money:          fmt.Sprintf("%.2f", amount),
		Device:         epay.PC,
		NotifyUrl:      notifyURL,
		ReturnUrl:      returnURL,
	})
	if err != nil {
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	order, err := service.CreatePendingSubscriptionOrder(c.GetInt("id"), req.PlanId, req.Action, common.SubscriptionPaymentModeCash, req.PaymentMethod, tradeNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"trade_no": order.TradeNo,
		"data":     params,
		"url":      uri,
		"amount":   order.Amount,
	})
}

func RequestSubscriptionStripePay(c *gin.Context) {
	if operation_setting.NormalizeSubscriptionPaymentMode(operation_setting.GetSubscriptionSetting().PaymentMode) == common.SubscriptionPaymentModeBalance {
		common.ApiErrorMsg(c, "当前未开启现金支付")
		return
	}
	var req SubscriptionCashPayRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.PaymentMethod != PaymentMethodStripe {
		common.ApiErrorMsg(c, "不支持的支付渠道")
		return
	}
	if req.SuccessURL != "" && common.ValidateRedirectURL(req.SuccessURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "支付成功重定向URL不在可信任域名列表中"})
		return
	}
	if req.CancelURL != "" && common.ValidateRedirectURL(req.CancelURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "支付取消重定向URL不在可信任域名列表中"})
		return
	}
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	amount, err := service.CalculateSubscriptionOrderAmount(c.GetInt("id"), req.PlanId, req.Action)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tradeNo := service.BuildSubscriptionTradeNo(c.GetInt("id"))
	payLink, err := genStripeCustomAmountLink(tradeNo, user.StripeCustomer, user.Email, amount, req.SuccessURL, req.CancelURL)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	order, err := service.CreatePendingSubscriptionOrder(c.GetInt("id"), req.PlanId, req.Action, common.SubscriptionPaymentModeCash, req.PaymentMethod, tradeNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"trade_no": order.TradeNo,
		"pay_link": payLink,
		"amount":   order.Amount,
	})
}

func RedeemSubscription(c *gin.Context) {
	var req struct {
		Key string `json:"key"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := service.RedeemSubscriptionPlan(req.Key, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, redemption)
}

func GetSubscriptionPlans(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	plans, total, err := model.GetAllSubscriptionPlans(pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(plans)
	common.ApiSuccess(c, pageInfo)
}

func CreateSubscriptionPlan(c *gin.Context) {
	var plan model.SubscriptionPlan
	if err := common.DecodeJson(c.Request.Body, &plan); err != nil {
		common.ApiError(c, err)
		return
	}
	plan.DurationUnit = strings.ToLower(plan.DurationUnit)
	plan.ResetIntervalUnit = strings.ToLower(plan.ResetIntervalUnit)
	if err := plan.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CreateSubscriptionPlan(&plan); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func UpdateSubscriptionPlan(c *gin.Context) {
	var plan model.SubscriptionPlan
	if err := common.DecodeJson(c.Request.Body, &plan); err != nil {
		common.ApiError(c, err)
		return
	}
	if plan.Id == 0 {
		common.ApiErrorMsg(c, "套餐 ID 不能为空")
		return
	}
	plan.DurationUnit = strings.ToLower(plan.DurationUnit)
	plan.ResetIntervalUnit = strings.ToLower(plan.ResetIntervalUnit)
	if err := plan.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateSubscriptionPlan(&plan); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func AssignSubscriptionToUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req SubscriptionAssignRequest
	if err = common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err = service.AssignSubscriptionPlanToUser(userId, req.PlanId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ChangeSubscriptionForUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req SubscriptionAssignRequest
	if err = common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err = service.ChangeUserSubscriptionPlan(userId, req.PlanId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func RemoveSubscriptionFromUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req struct {
		SubscriptionId int `json:"subscription_id"`
	}
	if err = common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err = service.RemoveUserSubscription(userId, req.SubscriptionId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetAllSubscriptionRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllSubscriptionRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
}

func SearchSubscriptionRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchSubscriptionRedemptions(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
}

func GetSubscriptionRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetSubscriptionRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, redemption)
}

func AddSubscriptionRedemption(c *gin.Context) {
	redemption := model.SubscriptionRedemption{}
	if err := common.DecodeJson(c.Request.Body, &redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		common.ApiErrorMsg(c, "兑换码名称长度必须在 1-20 个字符之间")
		return
	}
	if redemption.Count <= 0 || redemption.Count > 100 {
		common.ApiErrorMsg(c, "单次最多创建 100 个套餐兑换码")
		return
	}
	if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
		common.ApiErrorMsg(c, "兑换码过期时间无效")
		return
	}
	plan, err := model.GetSubscriptionPlanById(redemption.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	keys := make([]string, 0, redemption.Count)
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < redemption.Count; i++ {
			current := model.SubscriptionRedemption{
				UserId:      c.GetInt("id"),
				PlanId:      plan.Id,
				PlanName:    plan.Name,
				Key:         common.GetUUID(),
				Status:      common.RedemptionCodeStatusEnabled,
				Name:        redemption.Name,
				CreatedTime: common.GetTimestamp(),
				ExpiredTime: redemption.ExpiredTime,
			}
			if err := tx.Create(&current).Error; err != nil {
				return err
			}
			keys = append(keys, current.Key)
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, keys)
}

func UpdateSubscriptionRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.SubscriptionRedemption{}
	if err := common.DecodeJson(c.Request.Body, &redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	clean, err := model.GetSubscriptionRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			common.ApiErrorMsg(c, "兑换码过期时间无效")
			return
		}
		plan, err := model.GetSubscriptionPlanById(redemption.PlanId)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		clean.Name = redemption.Name
		clean.PlanId = plan.Id
		clean.PlanName = plan.Name
		clean.ExpiredTime = redemption.ExpiredTime
	}
	if statusOnly != "" {
		clean.Status = redemption.Status
	}
	if err := model.UpdateSubscriptionRedemption(clean); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, clean)
}

func DeleteSubscriptionRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteSubscriptionRedemptionById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteInvalidSubscriptionRedemptions(c *gin.Context) {
	rows, err := model.DeleteInvalidSubscriptionRedemptions()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

func genStripeCustomAmountLink(referenceId string, customerId string, email string, amount float64, successURL string, cancelURL string) (string, error) {
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		return "", fmt.Errorf("无效的Stripe API密钥")
	}
	stripe.Key = setting.StripeApiSecret
	if successURL == "" {
		successURL = system_setting.ServerAddress + "/console/subscription"
	}
	if cancelURL == "" {
		cancelURL = system_setting.ServerAddress + "/console/subscription"
	}
	unitAmount := int64(amount * 100)
	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(referenceId),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String("usd"),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String("New API 订阅套餐"),
				},
				UnitAmount: stripe.Int64(unitAmount),
			},
			Quantity: stripe.Int64(1),
		}},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
	}
	if customerId == "" {
		if email != "" {
			params.CustomerEmail = stripe.String(email)
		}
		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
	} else {
		params.Customer = stripe.String(customerId)
	}
	result, err := session.New(params)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
