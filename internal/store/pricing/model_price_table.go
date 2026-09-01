package pricingstore

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/config/ratio"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/store/db"
)

// ModelPricePlan is the persistent header for a componentized price table.
// Component rows live separately so one scope can carry multiple independently
// priced usage components without encoding mutable configuration as JSON.
type ModelPricePlan struct {
	ID                    int64  `gorm:"primaryKey;autoIncrement"`
	ModelName             string `gorm:"type:varchar(255);not null;index:idx_model_price_plan_scope"`
	Endpoint              string `gorm:"type:varchar(128);not null;index:idx_model_price_plan_scope"`
	EffectiveGroup        string `gorm:"type:varchar(128);not null;index:idx_model_price_plan_scope"`
	ServiceTier           string `gorm:"type:varchar(128);not null;index:idx_model_price_plan_scope"`
	ContextMinTokens      int    `gorm:"not null"`
	ContextMaxTokens      *int
	EffectiveFrom         *int64                `gorm:"index"`
	EffectiveUntil        *int64                `gorm:"index"`
	BillingMode           string                `gorm:"type:varchar(32);not null"`
	Currency              string                `gorm:"type:char(3);not null"`
	ExchangeRate          string                `gorm:"type:varchar(64);not null"`
	PricePrecision        int                   `gorm:"not null"`
	RoundingMode          string                `gorm:"type:varchar(32);not null"`
	GroupMultiplierSource string                `gorm:"type:varchar(32);not null"`
	GroupMultiplier       string                `gorm:"type:varchar(64);not null"`
	CreatedAt             int64                 `gorm:"bigint;not null;index"`
	UpdatedAt             int64                 `gorm:"bigint;not null"`
	Components            []ModelPriceComponent `gorm:"foreignKey:PlanID;constraint:OnDelete:CASCADE"`
}

func (ModelPricePlan) TableName() string {
	return "model_price_plans"
}

// ModelPriceComponent is deliberately a row rather than a JSON blob. The
// unique plan/component index prevents duplicate fees for one component.
type ModelPriceComponent struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	PlanID    int64  `gorm:"not null;uniqueIndex:uk_model_price_component,priority:1;index"`
	Component string `gorm:"type:varchar(64);not null;uniqueIndex:uk_model_price_component,priority:2"`
	Unit      string `gorm:"type:varchar(32);not null"`
	UnitPrice string `gorm:"type:varchar(64);not null"`
}

func (ModelPriceComponent) TableName() string {
	return "model_price_components"
}

var modelPricePlanCache = struct {
	sync.RWMutex
	plans     []contract.ModelPricePlan
	refreshed time.Time
}{}

const modelPricePlanCacheTTL = time.Minute

func GetModelPricePlans() ([]contract.ModelPricePlan, error) {
	modelPricePlanCache.RLock()
	if time.Since(modelPricePlanCache.refreshed) < modelPricePlanCacheTTL {
		plans := cloneModelPricePlans(modelPricePlanCache.plans)
		modelPricePlanCache.RUnlock()
		return plans, nil
	}
	modelPricePlanCache.RUnlock()

	modelPricePlanCache.Lock()
	defer modelPricePlanCache.Unlock()
	if time.Since(modelPricePlanCache.refreshed) < modelPricePlanCacheTTL {
		return cloneModelPricePlans(modelPricePlanCache.plans), nil
	}

	var records []ModelPricePlan
	if err := dbstore.DB.Preload("Components").Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	plans := make([]contract.ModelPricePlan, 0, len(records))
	for _, record := range records {
		plans = append(plans, modelPricePlanToContract(record))
	}
	modelPricePlanCache.plans = plans
	modelPricePlanCache.refreshed = time.Now()
	return cloneModelPricePlans(plans), nil
}

// ReplaceModelPricePlans atomically replaces only the component-price table.
// It intentionally does not touch legacy option maps, preserving an immediate
// lossless rollback path to ModelRatio/ModelPrice configuration.
func ReplaceModelPricePlans(plans []contract.ModelPricePlan) error {
	tx := dbstore.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := tx.Where("plan_id > ?", 0).Delete(&ModelPriceComponent{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("id > ?", 0).Delete(&ModelPricePlan{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	now := common.GetTimestamp()
	for _, plan := range plans {
		record := modelPricePlanFromContract(plan, now)
		if err := tx.Omit("Components").Create(&record).Error; err != nil {
			tx.Rollback()
			return err
		}
		for _, component := range record.Components {
			component.PlanID = record.ID
			if err := tx.Create(&component).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	InvalidateModelPricePlanCache()
	return nil
}

func InvalidateModelPricePlanCache() {
	modelPricePlanCache.Lock()
	modelPricePlanCache.plans = nil
	modelPricePlanCache.refreshed = time.Time{}
	modelPricePlanCache.Unlock()
}

// GetLegacyModelPricePlans projects the still-authoritative legacy option maps
// into the new component schema without persisting or mutating them.
func GetLegacyModelPricePlans() []contract.ModelPricePlan {
	modelPrices := ratio.GetModelPriceCopy()
	modelRatios := ratio.GetModelRatioCopy()
	completionRatios := ratio.GetCompletionRatioCopy()
	cacheRatios := ratio.GetCacheRatioCopy()
	createCacheRatios := ratio.GetCreateCacheRatioCopy()
	audioRatios := ratio.GetAudioRatioCopy()
	audioCompletionRatios := ratio.GetAudioCompletionRatioCopy()
	contextPricings := ratio.GetContextPricingCopy()

	modelNames := make(map[string]struct{})
	for name := range modelPrices {
		modelNames[name] = struct{}{}
	}
	for name := range modelRatios {
		modelNames[name] = struct{}{}
	}
	for name := range completionRatios {
		modelNames[name] = struct{}{}
	}
	for name := range cacheRatios {
		modelNames[name] = struct{}{}
	}
	for name := range createCacheRatios {
		modelNames[name] = struct{}{}
	}
	for name := range audioRatios {
		modelNames[name] = struct{}{}
	}
	for name := range audioCompletionRatios {
		modelNames[name] = struct{}{}
	}
	for name := range contextPricings {
		modelNames[name] = struct{}{}
	}

	names := make([]string, 0, len(modelNames))
	for name := range modelNames {
		names = append(names, name)
	}
	sort.Strings(names)

	plans := make([]contract.ModelPricePlan, 0, len(names))
	for _, modelName := range names {
		plans = append(plans, GetLegacyModelPricePlansForModel(modelName)...)
	}
	return plans
}

// GetLegacyModelPricePlansForModel uses the same getters as the live legacy
// billing path and preserves the concrete model name in its projection. The
// model marketplace can therefore show a compatible plan for a channel model
// whose old ratio was matched through a built-in wildcard normalization rule.
func GetLegacyModelPricePlansForModel(modelName string) []contract.ModelPricePlan {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil
	}

	modelPrice, hasModelPrice := ratio.GetModelPrice(modelName, false)
	modelRatio, hasModelRatio, _ := ratio.GetModelRatio(modelName)
	cacheRatio, _ := ratio.GetCacheRatio(modelName)
	cacheCreationRatio, _ := ratio.GetCreateCacheRatio(modelName)
	contextPricing, hasContextPricing := ratio.GetContextPricingConfig(modelName)

	input := contract.LegacyPriceInput{
		ModelName:            modelName,
		HasModelRatio:        hasModelRatio,
		ModelRatio:           modelRatio,
		CompletionRatio:      ratio.GetCompletionRatio(modelName),
		CacheRatio:           cacheRatio,
		CacheCreationRatio:   cacheCreationRatio,
		CacheCreation1hRatio: cacheCreationRatio * ratio.ClaudeCacheCreation1hMultiplier,
		AudioRatio:           ratio.GetAudioRatio(modelName),
		AudioCompletionRatio: ratio.GetAudioCompletionRatio(modelName),
		QuotaPerUnit:         common.QuotaPerUnit,
	}
	if hasModelPrice {
		input.ModelPrice = &modelPrice
	}
	if hasContextPricing {
		contextPricingCopy := contextPricing
		input.ContextPricing = &contextPricingCopy
	}
	return contract.LegacyPricePlans(input)
}

func modelPricePlanFromContract(plan contract.ModelPricePlan, now int64) ModelPricePlan {
	components := make([]ModelPriceComponent, 0, len(plan.Components))
	for _, component := range plan.Components {
		components = append(components, ModelPriceComponent{
			Component: string(component.Component),
			Unit:      string(component.Unit),
			UnitPrice: component.UnitPrice,
		})
	}
	return ModelPricePlan{
		ModelName:             plan.ModelName,
		Endpoint:              plan.Endpoint,
		EffectiveGroup:        plan.EffectiveGroup,
		ServiceTier:           plan.ServiceTier,
		ContextMinTokens:      plan.ContextMinTokens,
		ContextMaxTokens:      cloneIntPointer(plan.ContextMaxTokens),
		EffectiveFrom:         cloneInt64Pointer(plan.EffectiveFrom),
		EffectiveUntil:        cloneInt64Pointer(plan.EffectiveUntil),
		BillingMode:           string(plan.BillingMode),
		Currency:              plan.Currency,
		ExchangeRate:          plan.ExchangeRate,
		PricePrecision:        plan.PricePrecision,
		RoundingMode:          string(plan.RoundingMode),
		GroupMultiplierSource: string(plan.GroupMultiplierSource),
		GroupMultiplier:       plan.GroupMultiplier,
		CreatedAt:             now,
		UpdatedAt:             now,
		Components:            components,
	}
}

func modelPricePlanToContract(plan ModelPricePlan) contract.ModelPricePlan {
	components := make([]contract.ModelPriceComponent, 0, len(plan.Components))
	for _, component := range plan.Components {
		components = append(components, contract.ModelPriceComponent{
			Component: contract.PriceComponent(component.Component),
			Unit:      contract.PriceUnit(component.Unit),
			UnitPrice: component.UnitPrice,
		})
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i].Component < components[j].Component
	})
	return contract.ModelPricePlan{
		ID:                    plan.ID,
		ModelName:             plan.ModelName,
		Endpoint:              plan.Endpoint,
		EffectiveGroup:        plan.EffectiveGroup,
		ServiceTier:           plan.ServiceTier,
		ContextMinTokens:      plan.ContextMinTokens,
		ContextMaxTokens:      cloneIntPointer(plan.ContextMaxTokens),
		EffectiveFrom:         cloneInt64Pointer(plan.EffectiveFrom),
		EffectiveUntil:        cloneInt64Pointer(plan.EffectiveUntil),
		BillingMode:           contract.BillingMode(plan.BillingMode),
		Currency:              plan.Currency,
		ExchangeRate:          plan.ExchangeRate,
		PricePrecision:        plan.PricePrecision,
		RoundingMode:          contract.PriceRoundingMode(plan.RoundingMode),
		GroupMultiplierSource: contract.GroupMultiplierSource(plan.GroupMultiplierSource),
		GroupMultiplier:       plan.GroupMultiplier,
		Components:            components,
		Source:                contract.PricePlanSourceExplicit,
		CreatedAt:             plan.CreatedAt,
		UpdatedAt:             plan.UpdatedAt,
	}
}

func cloneModelPricePlans(plans []contract.ModelPricePlan) []contract.ModelPricePlan {
	clones := make([]contract.ModelPricePlan, len(plans))
	for i, plan := range plans {
		clones[i] = plan
		clones[i].ContextMaxTokens = cloneIntPointer(plan.ContextMaxTokens)
		clones[i].EffectiveFrom = cloneInt64Pointer(plan.EffectiveFrom)
		clones[i].EffectiveUntil = cloneInt64Pointer(plan.EffectiveUntil)
		clones[i].Components = make([]contract.ModelPriceComponent, len(plan.Components))
		copy(clones[i].Components, plan.Components)
	}
	return clones
}

func CloneModelPricePlans(plans []contract.ModelPricePlan) []contract.ModelPricePlan {
	return cloneModelPricePlans(plans)
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func MustGetModelPricePlans() []contract.ModelPricePlan {
	plans, err := GetModelPricePlans()
	if err != nil {
		common.SysError(fmt.Sprintf("load model price plans: %v", err))
		return nil
	}
	return plans
}
