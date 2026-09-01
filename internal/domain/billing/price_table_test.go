package billing

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
)

func TestNormalizeAndValidateModelPricePlansRejectsInvalidConfiguration(t *testing.T) {
	validPlan := func() contract.ModelPricePlan {
		return testTokenPricePlan(1, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "1")
	}
	maxTokens := 100
	from := int64(20)
	until := int64(10)

	tests := []struct {
		name     string
		plans    []contract.ModelPricePlan
		contains string
	}{
		{
			name: "parent and child components",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.Components = append(plan.Components, contract.ModelPriceComponent{
					Component: contract.PriceComponentTextInput,
					Unit:      contract.PriceUnitPerMillionTokens,
					UnitPrice: "1",
				})
				return plan
			}()},
			contains: "input cannot be configured together",
		},
		{
			name: "reasoning output is not billable",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.Components = []contract.ModelPriceComponent{{
					Component: contract.PriceComponentReasoningOutput,
					Unit:      contract.PriceUnitPerMillionTokens,
					UnitPrice: "1",
				}}
				return plan
			}()},
			contains: "reasoning_output is an output split",
		},
		{
			name: "precision exceeds configured scale",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.PricePrecision = 18
				plan.Components[0].UnitPrice = "0.0000000000000000001"
				return plan
			}()},
			contains: "exceeds 18 decimal places",
		},
		{
			name: "exchange rate must be positive",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.ExchangeRate = "0"
				return plan
			}()},
			contains: "exchange_rate must be greater than zero",
		},
		{
			name: "currency must be a recognized ISO code",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.Currency = "ZZZ"
				return plan
			}()},
			contains: "currency must be a three-letter ISO code",
		},
		{
			name: "fixed multiplier must be non-negative",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.GroupMultiplierSource = contract.GroupMultiplierSourceFixed
				plan.GroupMultiplier = "-0.1"
				return plan
			}()},
			contains: "group_multiplier must be >= 0",
		},
		{
			name: "invalid time window",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.EffectiveFrom = &from
				plan.EffectiveUntil = &until
				return plan
			}()},
			contains: "effective_until must be later",
		},
		{
			name: "group scope list separator",
			plans: []contract.ModelPricePlan{func() contract.ModelPricePlan {
				plan := validPlan()
				plan.EffectiveGroup = "vip,internal"
				return plan
			}()},
			contains: "effective_group must not contain control characters or commas",
		},
		{
			name: "overlapping context tiers",
			plans: []contract.ModelPricePlan{
				func() contract.ModelPricePlan {
					plan := validPlan()
					plan.ContextMaxTokens = &maxTokens
					return plan
				}(),
				func() contract.ModelPricePlan {
					plan := validPlan()
					plan.ContextMinTokens = 50
					return plan
				}(),
			},
			contains: "overlapping model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeAndValidateModelPricePlans(tt.plans)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("NormalizeAndValidateModelPricePlans() error = %v, want substring %q", err, tt.contains)
			}
		})
	}
}

func TestNormalizeAndValidateModelPricePlansPreservesHumanReadableScopes(t *testing.T) {
	plan := testTokenPricePlan(1, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "1")
	plan.Endpoint = "responses/v2+beta"
	plan.EffectiveGroup = "VIP Plus (APAC)"
	plan.ServiceTier = "priority batch"

	plans, err := NormalizeAndValidateModelPricePlans([]contract.ModelPricePlan{plan})
	if err != nil {
		t.Fatalf("NormalizeAndValidateModelPricePlans() error = %v", err)
	}
	got := plans[0]
	if got.Endpoint != plan.Endpoint || got.EffectiveGroup != plan.EffectiveGroup || got.ServiceTier != plan.ServiceTier {
		t.Fatalf("normalized scopes = endpoint=%q group=%q tier=%q", got.Endpoint, got.EffectiveGroup, got.ServiceTier)
	}
}

func TestNormalizeAndValidateModelPricePlansNormalizesExplicitPlan(t *testing.T) {
	plan := testTokenPricePlan(99, contract.PricePlanSourceLegacy, contract.PriceComponentOutput, "1.200")
	plan.ModelName = "  model-a  "
	plan.Currency = "usd"
	plan.Components = append(plan.Components, contract.ModelPriceComponent{
		Component: contract.PriceComponentCacheRead,
		Unit:      contract.PriceUnitPerMillionTokens,
		UnitPrice: "0.5",
	})

	plans, err := NormalizeAndValidateModelPricePlans([]contract.ModelPricePlan{plan})
	if err != nil {
		t.Fatalf("NormalizeAndValidateModelPricePlans() error = %v", err)
	}
	got := plans[0]
	if got.ID != 0 || got.Source != contract.PricePlanSourceExplicit || got.ReadOnly {
		t.Fatalf("unexpected normalized metadata: %+v", got)
	}
	if got.ModelName != "model-a" || got.Currency != "USD" {
		t.Fatalf("unexpected normalized strings: model=%q currency=%q", got.ModelName, got.Currency)
	}
	if got.Components[0].Component != contract.PriceComponentCacheRead || got.Components[1].Component != contract.PriceComponentOutput {
		t.Fatalf("components were not deterministically sorted: %+v", got.Components)
	}
}

func TestResolveModelPriceComponentFallsBackAcrossExplicitAndLegacyPlans(t *testing.T) {
	legacy := testTokenPricePlan(9, contract.PricePlanSourceLegacy, contract.PriceComponentInput, "2")
	legacy.Components = append(legacy.Components, contract.ModelPriceComponent{
		Component: contract.PriceComponentOutput,
		Unit:      contract.PriceUnitPerMillionTokens,
		UnitPrice: "3",
	})
	explicit := testTokenPricePlan(1, contract.PricePlanSourceExplicit, contract.PriceComponentTextInput, "10")
	explicit.Components = append(explicit.Components, contract.ModelPriceComponent{
		Component: contract.PriceComponentTextOutput,
		Unit:      contract.PriceUnitPerMillionTokens,
		UnitPrice: "11",
	})
	query := contract.ModelPricePlanQuery{ModelName: "model-a", ContextTokens: 12, EffectiveAt: 100}

	text, ok := ResolveModelPriceComponent([]contract.ModelPricePlan{legacy, explicit}, query, contract.PriceComponentTextInput)
	if !ok || text.Component.UnitPrice != "10" || text.PlanID != 1 || text.PlanSource != contract.PricePlanSourceExplicit {
		t.Fatalf("text component = %+v (exists=%t), want explicit override", text, ok)
	}
	image, ok := ResolveModelPriceComponent([]contract.ModelPricePlan{legacy, explicit}, query, contract.PriceComponentImageInput)
	if !ok || image.Component.UnitPrice != "2" || image.PlanID != 9 || image.Component.Component != contract.PriceComponentInput {
		t.Fatalf("image component = %+v (exists=%t), want legacy parent fallback", image, ok)
	}
	output, ok := ResolveModelPriceComponent([]contract.ModelPricePlan{legacy, explicit}, query, contract.PriceComponentAudioOutput)
	if !ok || output.Component.UnitPrice != "3" || output.Component.Component != contract.PriceComponentOutput {
		t.Fatalf("output component = %+v (exists=%t), want legacy output fallback", output, ok)
	}
	reasoning, ok := ResolveModelPriceComponent([]contract.ModelPricePlan{legacy, explicit}, query, contract.PriceComponentReasoningOutput)
	if !ok || reasoning.Component.UnitPrice != "11" || reasoning.Component.Component != contract.PriceComponentTextOutput || reasoning.PlanID != 1 {
		t.Fatalf("reasoning component = %+v (exists=%t), want explicit text-output fallback", reasoning, ok)
	}
}

func TestResolveModelPricePlanHonorsScopeContextAndTimePrecedence(t *testing.T) {
	global := testTokenPricePlan(1, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "1")
	endpoint := testTokenPricePlan(2, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "2")
	endpoint.Endpoint = "chat"
	tier := testTokenPricePlan(3, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "3")
	tier.ServiceTier = "flex"
	group := testTokenPricePlan(4, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "4")
	group.EffectiveGroup = "premium"
	contextMax := 200
	contextPlan := testTokenPricePlan(5, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "5")
	contextPlan.ContextMinTokens = 100
	contextPlan.ContextMaxTokens = &contextMax
	from := int64(100)
	until := int64(200)
	timePlan := testTokenPricePlan(6, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "6")
	timePlan.EffectiveFrom = &from
	timePlan.EffectiveUntil = &until
	plans := []contract.ModelPricePlan{global, endpoint, tier, group, contextPlan, timePlan}

	selected, ok := ResolveModelPricePlan(plans, contract.ModelPricePlanQuery{
		ModelName: "model-a", Endpoint: "chat", ServiceTier: "flex", EffectiveGroup: "premium", ContextTokens: 150, EffectiveAt: 150,
	})
	if !ok || selected.ID != group.ID {
		t.Fatalf("group-specific plan = %+v (exists=%t), want id %d", selected, ok, group.ID)
	}

	selected, ok = ResolveModelPricePlan(plans, contract.ModelPricePlanQuery{
		ModelName: "model-a", Endpoint: "chat", ServiceTier: "flex", ContextTokens: 50, EffectiveAt: 150,
	})
	if !ok || selected.ID != endpoint.ID {
		t.Fatalf("endpoint plan = %+v (exists=%t), want id %d", selected, ok, endpoint.ID)
	}

	selected, ok = ResolveModelPricePlan([]contract.ModelPricePlan{global, contextPlan, timePlan}, contract.ModelPricePlanQuery{
		ModelName: "model-a", ContextTokens: 150, EffectiveAt: 150,
	})
	if !ok || selected.ID != contextPlan.ID {
		t.Fatalf("context plan = %+v (exists=%t), want id %d", selected, ok, contextPlan.ID)
	}

	selected, ok = ResolveModelPricePlan([]contract.ModelPricePlan{global, contextPlan, timePlan}, contract.ModelPricePlanQuery{
		ModelName: "model-a", ContextTokens: 50, EffectiveAt: 150,
	})
	if !ok || selected.ID != timePlan.ID {
		t.Fatalf("time plan = %+v (exists=%t), want id %d", selected, ok, timePlan.ID)
	}
}

func TestResolveModelPriceComponentStopsForFreeAndPerRequestPlans(t *testing.T) {
	legacy := testTokenPricePlan(9, contract.PricePlanSourceLegacy, contract.PriceComponentInput, "2")
	free := testTokenPricePlan(1, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "1")
	free.BillingMode = contract.BillingModeFree
	free.Components = nil
	query := contract.ModelPricePlanQuery{ModelName: "model-a", EffectiveAt: 10}

	selected, ok := ResolveModelPricePlan([]contract.ModelPricePlan{legacy, free}, query)
	if !ok || selected.BillingMode != contract.BillingModeFree {
		t.Fatalf("free plan = %+v (exists=%t)", selected, ok)
	}
	if component, ok := ResolveModelPriceComponent([]contract.ModelPricePlan{legacy, free}, query, contract.PriceComponentInput); ok || component != nil {
		t.Fatalf("free plan must not fall back to a token component: %+v", component)
	}

	perRequest := testTokenPricePlan(2, contract.PricePlanSourceExplicit, contract.PriceComponentInput, "1")
	perRequest.BillingMode = contract.BillingModePerRequest
	perRequest.Components = []contract.ModelPriceComponent{{
		Component: contract.PriceComponentRequest,
		Unit:      contract.PriceUnitPerRequest,
		UnitPrice: "0.01",
	}}
	selected, ok = ResolveModelPricePlan([]contract.ModelPricePlan{legacy, perRequest}, query)
	if !ok || selected.BillingMode != contract.BillingModePerRequest {
		t.Fatalf("per-request plan = %+v (exists=%t)", selected, ok)
	}
	if component, ok := ResolveModelPriceComponent([]contract.ModelPricePlan{legacy, perRequest}, query, contract.PriceComponentInput); ok || component != nil {
		t.Fatalf("per-request plan must not fall back to a token component: %+v", component)
	}
}

func testTokenPricePlan(id int64, source contract.PricePlanSource, component contract.PriceComponent, price string) contract.ModelPricePlan {
	return contract.ModelPricePlan{
		ID:                    id,
		ModelName:             "model-a",
		BillingMode:           contract.BillingModeToken,
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        12,
		RoundingMode:          contract.PriceRoundingHalfUp,
		GroupMultiplierSource: contract.GroupMultiplierSourceInherit,
		Components: []contract.ModelPriceComponent{{
			Component: component,
			Unit:      contract.PriceUnitPerMillionTokens,
			UnitPrice: price,
		}},
		Source: source,
	}
}
