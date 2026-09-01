package billingcontroller

import (
	"mime"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/audit"
	"github.com/NookMux/NookMux/internal/domain/billing"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/internal/store/audit"
	"github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/gin-gonic/gin"
)

type updateModelPriceTableRequest struct {
	// A pointer distinguishes an omitted/null field from an explicit empty
	// array. Administrators may intentionally clear explicit plans with [], but
	// a malformed {} request must never erase the complete price table.
	Plans *[]contract.ModelPricePlan `json:"plans"`
}

// GetModelPriceTableConfiguration returns both the editable component table
// and a read-only projection of legacy ratio settings. Keeping the two sources
// separate makes the fallback behavior inspectable without mutating legacy
// options or implying that they have already been migrated.
func GetModelPriceTableConfiguration(c *gin.Context) {
	plans, err := pricingstore.GetModelPricePlans()
	if err != nil {
		common.SysError("load component price table: " + err.Error())
		httpapi.ApiErrorI18n(c, i18n.MsgPricingPriceTableReadFailed)
		return
	}
	httpapi.ApiSuccess(c, contract.ModelPriceTableConfiguration{
		Plans:       plans,
		LegacyPlans: pricingstore.GetLegacyModelPricePlans(),
	})
}

// UpdateModelPriceTableConfiguration replaces only explicit component plans.
// Legacy ratio maps remain untouched, which preserves rollback and supplies
// component-level fallback until every explicit plan is complete.
func UpdateModelPriceTableConfiguration(c *gin.Context) {
	contentType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || contentType != gin.MIMEJSON {
		httpapi.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}

	var request updateModelPriceTableRequest
	if err := httpapi.UnmarshalBodyReusable(c, &request); err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}
	if request.Plans == nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPricingPriceTableInvalid, map[string]any{"Reason": "plans is required"})
		return
	}
	normalizedPlans, err := billing.NormalizeAndValidateModelPricePlans(*request.Plans)
	if err != nil {
		httpapi.ApiErrorI18n(c, i18n.MsgPricingPriceTableInvalid, map[string]any{"Reason": err.Error()})
		return
	}

	before, err := pricingstore.GetModelPricePlans()
	if err != nil {
		common.SysError("load component price table before update: " + err.Error())
		httpapi.ApiErrorI18n(c, i18n.MsgPricingPriceTableReadFailed)
		return
	}
	if err := pricingstore.ReplaceModelPricePlans(normalizedPlans); err != nil {
		common.SysError("replace component price table: " + err.Error())
		httpapi.ApiErrorI18n(c, i18n.MsgPricingPriceTableSaveFailed)
		return
	}
	after := normalizedPlans
	if persisted, err := pricingstore.GetModelPricePlans(); err != nil {
		common.SysError("load persisted component price table for audit: " + err.Error())
	} else {
		after = persisted
	}

	audit.RecordAudit(c, auditstore.AuditModulePricing, auditstore.AuditActionUpdate, "更新组件化模型价格表", before, after)
	// ReplaceModelPricePlans invalidates its own table cache. RefreshPricing
	// rebuilds the marketplace cache after the write has committed.
	if err := pricingstore.RefreshPricing(); err != nil {
		common.SysError("refresh pricing after component price table update: " + err.Error())
		httpapi.ApiErrorI18n(c, i18n.MsgPricingPriceTableRefreshFailed)
		return
	}
	httpapi.ApiSuccessI18n(c, i18n.MsgPricingPriceTableSaved, contract.ModelPriceTableConfiguration{
		Plans:       after,
		LegacyPlans: pricingstore.GetLegacyModelPricePlans(),
	})
}
