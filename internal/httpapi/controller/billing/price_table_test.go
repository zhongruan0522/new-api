package billingcontroller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/operation"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/httpapi/middleware"
	"github.com/NookMux/NookMux/internal/infra/redis"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/channel"
	"github.com/NookMux/NookMux/internal/store/db"
	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/NookMux/NookMux/internal/store/user"
	"github.com/NookMux/NookMux/internal/store/vendor_meta"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModelPriceTableConfigurationRequiresAdminAndAuditsSuccessfulWrites(t *testing.T) {
	setupModelPriceTableControllerTestDB(t)
	gin.SetMode(gin.TestMode)

	adminToken := "admin-price-table-token"
	admin := createModelPriceTableUser(t, 1, "price-admin", common.RoleAdminUser, adminToken)
	userToken := "user-price-table-token"
	regularUser := createModelPriceTableUser(t, 2, "price-user", common.RoleCommonUser, userToken)

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("price-table-test-session"))))
	router.GET("/api/pricing/configuration", middleware.AdminAuth(), GetModelPriceTableConfiguration)
	router.PUT("/api/pricing/configuration", middleware.AdminAuth(), UpdateModelPriceTableConfiguration)

	validPlan := testControllerPricePlan("model-admin")
	requestBody, err := jsonx.Marshal(map[string]any{"plans": []contract.ModelPricePlan{validPlan}})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	regularResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", userToken, regularUser.Id, requestBody)
	if priceTableResponseSuccess(t, regularResponse) {
		t.Fatalf("ordinary user unexpectedly updated the price table: %s", regularResponse.Body.String())
	}
	regularGetResponse := performPriceTableRequest(router, http.MethodGet, "/api/pricing/configuration", userToken, regularUser.Id, nil)
	if priceTableResponseSuccess(t, regularGetResponse) {
		t.Fatalf("ordinary user unexpectedly read the price table: %s", regularGetResponse.Body.String())
	}
	plans, err := pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after ordinary user request: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("ordinary user request changed price plans: %+v", plans)
	}

	adminResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", adminToken, admin.Id, requestBody)
	if adminResponse.Code != http.StatusOK || !priceTableResponseSuccess(t, adminResponse) {
		t.Fatalf("admin update failed (%d): %s", adminResponse.Code, adminResponse.Body.String())
	}
	var savedResponse struct {
		Data contract.ModelPriceTableConfiguration `json:"data"`
	}
	if err := jsonx.Unmarshal(adminResponse.Body.Bytes(), &savedResponse); err != nil {
		t.Fatalf("decode saved price table response: %v", err)
	}
	if savedResponse.Data.LegacyPlans == nil {
		t.Fatal("saved price table response must include legacy_plans rather than null")
	}
	plans, err = pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after admin update: %v", err)
	}
	if len(plans) != 1 || plans[0].ModelName != "model-admin" {
		t.Fatalf("admin update did not persist expected plan: %+v", plans)
	}
	waitForPricingAudit(t, admin.Username)

	missingPlansResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", adminToken, admin.Id, []byte(`{}`))
	if priceTableResponseSuccess(t, missingPlansResponse) {
		t.Fatalf("omitted plans unexpectedly cleared configuration: %s", missingPlansResponse.Body.String())
	}
	plans, err = pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after omitted plans request: %v", err)
	}
	if len(plans) != 1 || plans[0].ModelName != "model-admin" {
		t.Fatalf("omitted plans request changed persisted configuration: %+v", plans)
	}

	nullPlansResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", adminToken, admin.Id, []byte(`{"plans":null}`))
	if priceTableResponseSuccess(t, nullPlansResponse) {
		t.Fatalf("null plans unexpectedly cleared configuration: %s", nullPlansResponse.Body.String())
	}
	plans, err = pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after null plans request: %v", err)
	}
	if len(plans) != 1 || plans[0].ModelName != "model-admin" {
		t.Fatalf("null plans request changed persisted configuration: %+v", plans)
	}

	malformedResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", adminToken, admin.Id, []byte(`{"plans":`))
	if priceTableResponseSuccess(t, malformedResponse) {
		t.Fatalf("malformed request unexpectedly changed configuration: %s", malformedResponse.Body.String())
	}
	plans, err = pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after malformed request: %v", err)
	}
	if len(plans) != 1 || plans[0].ModelName != "model-admin" {
		t.Fatalf("malformed request changed persisted configuration: %+v", plans)
	}

	invalidContentTypeRequest := httptest.NewRequest(http.MethodPut, "/api/pricing/configuration", bytes.NewReader(requestBody))
	invalidContentTypeRequest.Header.Set("Content-Type", "application/jsonp")
	invalidContentTypeRequest.Header.Set("Authorization", "Bearer "+adminToken)
	invalidContentTypeRequest.Header.Set("New-Api-User", strconv.Itoa(admin.Id))
	invalidContentTypeResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidContentTypeResponse, invalidContentTypeRequest)
	if priceTableResponseSuccess(t, invalidContentTypeResponse) {
		t.Fatalf("invalid content type unexpectedly changed configuration: %s", invalidContentTypeResponse.Body.String())
	}
	plans, err = pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after invalid content type: %v", err)
	}
	if len(plans) != 1 || plans[0].ModelName != "model-admin" {
		t.Fatalf("invalid content type changed persisted configuration: %+v", plans)
	}

	clearResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", adminToken, admin.Id, []byte(`{"plans":[]}`))
	if clearResponse.Code != http.StatusOK || !priceTableResponseSuccess(t, clearResponse) {
		t.Fatalf("explicit empty plans did not clear configuration (%d): %s", clearResponse.Code, clearResponse.Body.String())
	}
	plans, err = pricingstore.GetModelPricePlans()
	if err != nil {
		t.Fatalf("read plans after explicit clear: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("explicit empty plans did not clear persisted configuration: %+v", plans)
	}

	getResponse := performPriceTableRequest(router, http.MethodGet, "/api/pricing/configuration", adminToken, admin.Id, nil)
	if getResponse.Code != http.StatusOK || !priceTableResponseSuccess(t, getResponse) {
		t.Fatalf("admin configuration read failed (%d): %s", getResponse.Code, getResponse.Body.String())
	}

	freePlan := testControllerPricePlan("free-model")
	freePlan.BillingMode = contract.BillingModeFree
	freePlan.Components = []contract.ModelPriceComponent{}
	freeRequestBody, err := jsonx.Marshal(map[string]any{"plans": []contract.ModelPricePlan{freePlan}})
	if err != nil {
		t.Fatalf("marshal free price plan request: %v", err)
	}
	freeUpdateResponse := performPriceTableRequest(router, http.MethodPut, "/api/pricing/configuration", adminToken, admin.Id, freeRequestBody)
	if freeUpdateResponse.Code != http.StatusOK || !priceTableResponseSuccess(t, freeUpdateResponse) {
		t.Fatalf("free price plan update failed (%d): %s", freeUpdateResponse.Code, freeUpdateResponse.Body.String())
	}
	var freeUpdateConfiguration struct {
		Data contract.ModelPriceTableConfiguration `json:"data"`
	}
	if err := jsonx.Unmarshal(freeUpdateResponse.Body.Bytes(), &freeUpdateConfiguration); err != nil {
		t.Fatalf("decode free price plan update response: %v", err)
	}
	if len(freeUpdateConfiguration.Data.Plans) != 1 || freeUpdateConfiguration.Data.Plans[0].Components == nil {
		t.Fatalf("free price plan update must return an empty components array: %+v", freeUpdateConfiguration.Data.Plans)
	}
	freeGetResponse := performPriceTableRequest(router, http.MethodGet, "/api/pricing/configuration", adminToken, admin.Id, nil)
	if freeGetResponse.Code != http.StatusOK || !priceTableResponseSuccess(t, freeGetResponse) {
		t.Fatalf("free price plan read failed (%d): %s", freeGetResponse.Code, freeGetResponse.Body.String())
	}
	var freeReadResponse struct {
		Data contract.ModelPriceTableConfiguration `json:"data"`
	}
	if err := jsonx.Unmarshal(freeGetResponse.Body.Bytes(), &freeReadResponse); err != nil {
		t.Fatalf("decode free price plan response: %v", err)
	}
	if len(freeReadResponse.Data.Plans) != 1 || freeReadResponse.Data.Plans[0].Components == nil {
		t.Fatalf("free price plan must return an empty components array: %+v", freeReadResponse.Data.Plans)
	}
}

func setupModelPriceTableControllerTestDB(t *testing.T) {
	t.Helper()

	oldDB := dbstore.DB
	oldLogDB := dbstore.LOG_DB
	oldRedisEnabled := redis.RedisEnabled
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldAuditSetting := *operation.GetAuditSetting()

	dbHandle, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := dbHandle.AutoMigrate(
		&userstore.User{},
		&auditstore.AuditLog{},
		&pricingstore.ModelPricePlan{},
		&pricingstore.ModelPriceComponent{},
		&channelstore.Ability{},
		&channelstore.Channel{},
		&vendormetastore.Model{},
		&vendormetastore.Vendor{},
	); err != nil {
		t.Fatalf("migrate sqlite schema: %v", err)
	}
	dbstore.DB = dbHandle
	dbstore.LOG_DB = dbHandle
	redis.RedisEnabled = false
	common.MemoryCacheEnabled = false
	pricingstore.InvalidateModelPricePlanCache()
	*operation.GetAuditSetting() = operation.AuditSetting{
		Enabled: true,
		// Mirrors persisted installations from before the pricing module existed.
		// The write below must still produce a pricing audit record.
		Modules:    `{"option":true}`,
		RecordIp:   false,
		RecordDiff: true,
	}

	t.Cleanup(func() {
		pricingstore.InvalidateModelPricePlanCache()
		*operation.GetAuditSetting() = oldAuditSetting
		if sqlDB, err := dbHandle.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
		dbstore.LOG_DB = oldLogDB
		redis.RedisEnabled = oldRedisEnabled
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func createModelPriceTableUser(t *testing.T, id int, username string, role int, accessToken string) userstore.User {
	t.Helper()
	user := userstore.User{
		Id:          id,
		Username:    username,
		Password:    "password123",
		Role:        role,
		Status:      common.UserStatusEnabled,
		DisplayName: username,
		AccessToken: &accessToken,
		Group:       "default",
		AffCode:     "price-table-" + username,
	}
	if err := dbstore.DB.Create(&user).Error; err != nil {
		t.Fatalf("create %s: %v", username, err)
	}
	return user
}

func performPriceTableRequest(router http.Handler, method, path, accessToken string, userID int, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", gin.MIMEJSON)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("New-Api-User", strconv.Itoa(userID))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func priceTableResponseSuccess(t *testing.T, response *httptest.ResponseRecorder) bool {
	t.Helper()
	var body struct {
		Success bool `json:"success"`
	}
	if err := jsonx.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body.String())
	}
	return body.Success
}

func waitForPricingAudit(t *testing.T, username string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		if err := dbstore.DB.Model(&auditstore.AuditLog{}).
			Where("module = ? AND action_type = ? AND username = ?", auditstore.AuditModulePricing, auditstore.AuditActionUpdate, username).
			Count(&count).Error; err != nil {
			t.Fatalf("query pricing audit: %v", err)
		}
		if count == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for pricing audit record")
}

func testControllerPricePlan(modelName string) contract.ModelPricePlan {
	return contract.ModelPricePlan{
		ModelName:             modelName,
		BillingMode:           contract.BillingModeToken,
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        12,
		RoundingMode:          contract.PriceRoundingHalfUp,
		GroupMultiplierSource: contract.GroupMultiplierSourceInherit,
		Components: []contract.ModelPriceComponent{{
			Component: contract.PriceComponentInput,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: "1.25",
		}},
	}
}
