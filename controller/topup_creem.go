package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"one-api/common"
	"one-api/model"
	"one-api/setting"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

const (
	PaymentMethodCreem = "creem"
)

var creemAdaptor = &CreemAdaptor{}

type CreemPayRequest struct {
	ProductId     string `json:"product_id"`
	PaymentMethod string `json:"payment_method"`
}

type CreemProduct struct {
	ProductId string  `json:"productId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	Quota     int64   `json:"quota"`
}

type CreemAdaptor struct {
}

func (*CreemAdaptor) RequestPay(c *gin.Context, req *CreemPayRequest) {
	if req.PaymentMethod != PaymentMethodCreem {
		c.JSON(200, gin.H{"message": "error", "data": "不支持的支付渠道"})
		return
	}

	if req.ProductId == "" {
		c.JSON(200, gin.H{"message": "error", "data": "请选择产品"})
		return
	}

	// 解析产品列表
	var products []CreemProduct
	err := json.Unmarshal([]byte(setting.CreemProducts), &products)
	if err != nil {
		log.Println("解析Creem产品列表失败", err)
		c.JSON(200, gin.H{"message": "error", "data": "产品配置错误"})
		return
	}

	// 查找对应的产品
	var selectedProduct *CreemProduct
	for _, product := range products {
		if product.ProductId == req.ProductId {
			selectedProduct = &product
			break
		}
	}

	if selectedProduct == nil {
		c.JSON(200, gin.H{"message": "error", "data": "产品不存在"})
		return
	}

	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)

	reference := fmt.Sprintf("creem-api-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "ref_" + common.Sha1([]byte(reference))

	checkoutUrl, err := genCreemLink(referenceId, selectedProduct, user.Email, user.Username)
	if err != nil {
		log.Println("获取Creem支付链接失败", err)
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	topUp := &model.TopUp{
		UserId:     id,
		Amount:     selectedProduct.Quota,
		Money:      selectedProduct.Price,
		TradeNo:    referenceId,
		CreateTime: time.Now().Unix(),
		Status:     common.TopUpStatusPending,
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url": checkoutUrl,
		},
	})
}

func RequestCreemPay(c *gin.Context) {
	var req CreemPayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	creemAdaptor.RequestPay(c, &req)
}

type CreemWebhookData struct {
	Type string `json:"type"`
	Data struct {
		RequestId string            `json:"request_id"`
		Status    string            `json:"status"`
		Metadata  map[string]string `json:"metadata"`
	} `json:"data"`
}

func CreemWebhook(c *gin.Context) {
	// 解析 webhook 数据
	var webhookData CreemWebhookData
	if err := c.ShouldBindJSON(&webhookData); err != nil {
		log.Printf("解析Creem Webhook参数失败: %v\n", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 检查事件类型
	if webhookData.Type != "checkout.completed" {
		log.Printf("忽略Creem Webhook事件类型: %s", webhookData.Type)
		c.Status(http.StatusOK)
		return
	}

	// 获取引用ID
	referenceId := webhookData.Data.RequestId
	if referenceId == "" {
		log.Println("Creem Webhook缺少request_id字段")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 处理支付完成事件
	err := model.RechargeCreem(referenceId)
	if err != nil {
		log.Println("Creem充值处理失败:", err.Error(), referenceId)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	log.Printf("Creem充值成功: %s", referenceId)
	c.Status(http.StatusOK)
}

type CreemCheckoutRequest struct {
	ProductId string `json:"product_id"`
	RequestId string `json:"request_id"`
	Customer  struct {
		Email string `json:"email"`
	} `json:"customer"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type CreemCheckoutResponse struct {
	CheckoutUrl string `json:"checkout_url"`
	Id          string `json:"id"`
}

func genCreemLink(referenceId string, product *CreemProduct, email string, username string) (string, error) {
	if setting.CreemApiKey == "" {
		return "", fmt.Errorf("未配置Creem API密钥")
	}

	// 根据测试模式选择 API 端点
	apiUrl := "https://api.creem.io/v1/checkouts"
	if setting.CreemTestMode {
		apiUrl = "https://test-api.creem.io/v1/checkouts"
	}

	// 构建请求数据
	requestData := CreemCheckoutRequest{
		ProductId: product.ProductId,
		RequestId: referenceId,
		Customer: struct {
			Email string `json:"email"`
		}{
			Email: email,
		},
		Metadata: map[string]string{
			"username":     username,
			"reference_id": referenceId,
		},
	}

	// 序列化请求数据
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", setting.CreemApiKey)

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()
	log.Printf(" creem req host: %s, key %s req 【%s】", apiUrl, setting.CreemApiKey, jsonData)

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Creem API 返回错误状态 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var checkoutResp CreemCheckoutResponse
	err = json.Unmarshal(body, &checkoutResp)
	if err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if checkoutResp.CheckoutUrl == "" {
		return "", fmt.Errorf("Creem API 未返回支付链接")
	}

	log.Printf("Creem 支付链接创建成功: %s, 订单ID: %s", referenceId, checkoutResp.Id)
	return checkoutResp.CheckoutUrl, nil
}
