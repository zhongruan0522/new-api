package contract

import (
	"strings"
	"testing"
)

func TestLegacyTokenPricePlanExpandsAudioOverrideWithoutParentChildConflict(t *testing.T) {
	plans := LegacyPricePlans(LegacyPriceInput{
		ModelName:            "legacy-audio",
		HasModelRatio:        true,
		ModelRatio:           2,
		CompletionRatio:      3,
		CacheRatio:           0.5,
		CacheCreationRatio:   1.25,
		CacheCreation1hRatio: 2,
		AudioRatio:           4,
		AudioCompletionRatio: 2,
	})
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	plan := plans[0]
	if plan.BillingMode != BillingModeToken || !plan.ReadOnly || plan.Source != PricePlanSourceLegacy {
		t.Fatalf("unexpected legacy plan metadata: %+v", plan)
	}
	if hasPriceComponent(plan, PriceComponentInput) || hasPriceComponent(plan, PriceComponentOutput) {
		t.Fatalf("audio override must expand parent components: %+v", plan.Components)
	}

	wantPrices := map[PriceComponent]string{
		PriceComponentTextInput:     "4",
		PriceComponentImageInput:    "4",
		PriceComponentAudioInput:    "16",
		PriceComponentVideoInput:    "4",
		PriceComponentDocumentInput: "4",
		PriceComponentTextOutput:    "12",
		PriceComponentImageOutput:   "12",
		PriceComponentAudioOutput:   "32",
		PriceComponentCacheRead:     "2",
		PriceComponentCacheWrite5m:  "5",
		PriceComponentCacheWrite1h:  "8",
	}
	for component, want := range wantPrices {
		got, ok := priceComponentValue(plan, component)
		if !ok || got != want {
			t.Fatalf("component %s price = %q (exists=%t), want %q", component, got, ok, want)
		}
	}
	if len(plan.Components) != len(wantPrices) {
		t.Fatalf("got %d components, want %d: %+v", len(plan.Components), len(wantPrices), plan.Components)
	}
}

func TestLegacyTokenPricePlanKeepsParentsWhenPricesAreEquivalent(t *testing.T) {
	plans := LegacyPricePlans(LegacyPriceInput{
		ModelName:            "legacy-uniform",
		HasModelRatio:        true,
		ModelRatio:           2,
		CompletionRatio:      3,
		CacheRatio:           1,
		CacheCreationRatio:   1.25,
		CacheCreation1hRatio: 2,
		AudioRatio:           1,
		AudioCompletionRatio: 3,
	})
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	plan := plans[0]
	if got, ok := priceComponentValue(plan, PriceComponentInput); !ok || got != "4" {
		t.Fatalf("input price = %q (exists=%t), want 4", got, ok)
	}
	if got, ok := priceComponentValue(plan, PriceComponentOutput); !ok || got != "12" {
		t.Fatalf("output price = %q (exists=%t), want 12", got, ok)
	}
	for _, component := range append(append([]PriceComponent{}, InputChildPriceComponents...), OutputChildPriceComponents...) {
		if hasPriceComponent(plan, component) {
			t.Fatalf("uniform legacy pricing must use parent components, found %s", component)
		}
	}
}

func TestLegacyPricePlansPreservePerRequestFreeAndContextTiers(t *testing.T) {
	price := 0.75
	perRequest := LegacyPricePlans(LegacyPriceInput{ModelName: "per-request", ModelPrice: &price})
	if len(perRequest) != 1 || perRequest[0].BillingMode != BillingModePerRequest {
		t.Fatalf("per-request projection = %+v", perRequest)
	}
	if got, ok := priceComponentValue(perRequest[0], PriceComponentRequest); !ok || got != "0.75" {
		t.Fatalf("per-request price = %q (exists=%t), want 0.75", got, ok)
	}

	free := LegacyPricePlans(LegacyPriceInput{ModelName: "free", HasModelRatio: true, ModelRatio: 0})
	if len(free) != 1 || free[0].BillingMode != BillingModeFree || len(free[0].Components) != 0 {
		t.Fatalf("free projection = %+v", free)
	}
	if free[0].Components == nil {
		t.Fatal("free projection must serialize components as an empty array, not null")
	}

	maxTokens := 100
	tiered := LegacyPricePlans(LegacyPriceInput{
		ModelName: "tiered",
		ContextPricing: &ContextPricingConfig{
			Enabled: true,
			Tiers: []ContextPricingTier{
				{MinTokens: 0, MaxTokens: &maxTokens, ModelRatio: 1, CompletionRatio: 2, CacheRatio: 0.5, CreateCacheRatio: 1.25, AudioRatio: 1, AudioCompletionRatio: 2},
				{MinTokens: maxTokens, ModelRatio: 3, CompletionRatio: 4, CacheRatio: 0.5, CreateCacheRatio: 1.25, AudioRatio: 1, AudioCompletionRatio: 4},
			},
		},
	})
	if len(tiered) != 2 || tiered[0].ContextMinTokens != 0 || tiered[0].ContextMaxTokens == nil || *tiered[0].ContextMaxTokens != maxTokens || tiered[1].ContextMinTokens != maxTokens {
		t.Fatalf("context-tier projection = %+v", tiered)
	}
	if got, ok := priceComponentValue(tiered[1], PriceComponentInput); !ok || got != "6" {
		t.Fatalf("second tier input price = %q (exists=%t), want 6", got, ok)
	}
}

func TestLegacyTokenPricePlanUsesConfiguredQuotaPerUnit(t *testing.T) {
	plans := LegacyPricePlans(LegacyPriceInput{
		ModelName:            "custom-quota-unit",
		HasModelRatio:        true,
		ModelRatio:           1,
		CompletionRatio:      1,
		CacheRatio:           1,
		CacheCreationRatio:   1.25,
		CacheCreation1hRatio: 2,
		AudioRatio:           1,
		AudioCompletionRatio: 1,
		QuotaPerUnit:         250_000,
	})
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	if got, ok := priceComponentValue(plans[0], PriceComponentInput); !ok || got != "4" {
		t.Fatalf("input price = %q (exists=%t), want 4 for QuotaPerUnit=250000", got, ok)
	}
}

func TestLegacyTokenPricePlanKeepsLosslessFloatConversionWithinDeclaredPrecision(t *testing.T) {
	plans := LegacyPricePlans(LegacyPriceInput{
		ModelName:            "recurring-decimal",
		HasModelRatio:        true,
		ModelRatio:           4.0 / 3.0,
		CompletionRatio:      1,
		CacheRatio:           1,
		CacheCreationRatio:   1,
		AudioRatio:           1,
		AudioCompletionRatio: 1,
	})
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	plan := plans[0]
	if plan.PricePrecision != 18 {
		t.Fatalf("legacy precision = %d, want 18", plan.PricePrecision)
	}
	price, ok := priceComponentValue(plan, PriceComponentInput)
	if !ok {
		t.Fatal("legacy input component is missing")
	}
	if price != "2.6666666666666665" {
		t.Fatalf("legacy input price = %q, want shortest lossless conversion", price)
	}
	parts := strings.Split(price, ".")
	if len(parts) == 2 && len(parts[1]) > plan.PricePrecision {
		t.Fatalf("legacy input price %q exceeds declared precision %d", price, plan.PricePrecision)
	}
}

func hasPriceComponent(plan ModelPricePlan, component PriceComponent) bool {
	_, ok := priceComponentValue(plan, component)
	return ok
}

func priceComponentValue(plan ModelPricePlan, component PriceComponent) (string, bool) {
	for _, price := range plan.Components {
		if price.Component == component {
			return price.UnitPrice, true
		}
	}
	return "", false
}
