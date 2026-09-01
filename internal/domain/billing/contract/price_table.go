package contract

import "strconv"

// BillingMode describes how a price plan is charged. Token plans contain
// component prices, per-request plans contain exactly one request component,
// and free plans intentionally contain no component prices.
type BillingMode string

const (
	BillingModeToken      BillingMode = "token"
	BillingModePerRequest BillingMode = "per_request"
	BillingModeFree       BillingMode = "free"
)

// PriceComponent identifies a billable usage component. Input and output are
// parent components for legacy-compatible pricing; they cannot coexist with
// their more specific children in one plan.
type PriceComponent string

const (
	PriceComponentInput         PriceComponent = "input"
	PriceComponentTextInput     PriceComponent = "text_input"
	PriceComponentImageInput    PriceComponent = "image_input"
	PriceComponentAudioInput    PriceComponent = "audio_input"
	PriceComponentVideoInput    PriceComponent = "video_input"
	PriceComponentDocumentInput PriceComponent = "document_input"

	PriceComponentOutput      PriceComponent = "output"
	PriceComponentTextOutput  PriceComponent = "text_output"
	PriceComponentAudioOutput PriceComponent = "audio_output"
	PriceComponentImageOutput PriceComponent = "image_output"
	// Reasoning is an output usage split. It intentionally has no price-table
	// entry because it always reuses the applicable output price.
	PriceComponentReasoningOutput PriceComponent = "reasoning_output"

	PriceComponentCacheRead    PriceComponent = "cache_read"
	PriceComponentCacheWrite5m PriceComponent = "cache_write_5m"
	PriceComponentCacheWrite1h PriceComponent = "cache_write_1h"
	PriceComponentRequest      PriceComponent = "request"
)

var TokenPriceComponents = []PriceComponent{
	PriceComponentInput,
	PriceComponentTextInput,
	PriceComponentImageInput,
	PriceComponentAudioInput,
	PriceComponentVideoInput,
	PriceComponentDocumentInput,
	PriceComponentOutput,
	PriceComponentTextOutput,
	PriceComponentAudioOutput,
	PriceComponentImageOutput,
	PriceComponentCacheRead,
	PriceComponentCacheWrite5m,
	PriceComponentCacheWrite1h,
}

var InputChildPriceComponents = []PriceComponent{
	PriceComponentTextInput,
	PriceComponentImageInput,
	PriceComponentAudioInput,
	PriceComponentVideoInput,
	PriceComponentDocumentInput,
}

var OutputChildPriceComponents = []PriceComponent{
	PriceComponentTextOutput,
	PriceComponentAudioOutput,
	PriceComponentImageOutput,
}

type PriceUnit string

const (
	PriceUnitPerMillionTokens PriceUnit = "per_1m_tokens"
	PriceUnitPerRequest       PriceUnit = "per_request"
)

type PriceRoundingMode string

const (
	PriceRoundingHalfUp   PriceRoundingMode = "half_up"
	PriceRoundingHalfEven PriceRoundingMode = "half_even"
	PriceRoundingFloor    PriceRoundingMode = "floor"
	PriceRoundingCeil     PriceRoundingMode = "ceil"
)

type GroupMultiplierSource string

const (
	GroupMultiplierSourceInherit GroupMultiplierSource = "inherit_group_ratio"
	GroupMultiplierSourceFixed   GroupMultiplierSource = "fixed"
)

type PricePlanSource string

const (
	PricePlanSourceExplicit PricePlanSource = "explicit"
	PricePlanSourceLegacy   PricePlanSource = "legacy"
)

// ModelPriceComponent stores an exact decimal price as text. Keeping the
// literal avoids cross-database floating point drift and leaves stage 4 free
// to apply the configured rounding rule at settlement time.
type ModelPriceComponent struct {
	Component PriceComponent `json:"component"`
	Unit      PriceUnit      `json:"unit"`
	UnitPrice string         `json:"unit_price"`
}

// ModelPricePlan scopes a component-price set by model, endpoint, effective
// group, service tier, context range, and time range. Empty endpoint/group/
// tier fields are deliberate fallbacks, not missing values.
type ModelPricePlan struct {
	ID                    int64                 `json:"id,omitempty"`
	ModelName             string                `json:"model_name"`
	Endpoint              string                `json:"endpoint,omitempty"`
	EffectiveGroup        string                `json:"effective_group,omitempty"`
	ServiceTier           string                `json:"service_tier,omitempty"`
	ContextMinTokens      int                   `json:"context_min_tokens"`
	ContextMaxTokens      *int                  `json:"context_max_tokens,omitempty"`
	EffectiveFrom         *int64                `json:"effective_from,omitempty"`
	EffectiveUntil        *int64                `json:"effective_until,omitempty"`
	BillingMode           BillingMode           `json:"billing_mode"`
	Currency              string                `json:"currency"`
	ExchangeRate          string                `json:"exchange_rate"`
	PricePrecision        int                   `json:"price_precision"`
	RoundingMode          PriceRoundingMode     `json:"rounding_mode"`
	GroupMultiplierSource GroupMultiplierSource `json:"group_multiplier_source"`
	GroupMultiplier       string                `json:"group_multiplier,omitempty"`
	Components            []ModelPriceComponent `json:"components"`
	Source                PricePlanSource       `json:"source,omitempty"`
	ReadOnly              bool                  `json:"read_only,omitempty"`
	CreatedAt             int64                 `json:"created_at,omitempty"`
	UpdatedAt             int64                 `json:"updated_at,omitempty"`
}

type ModelPriceTableConfiguration struct {
	Plans       []ModelPricePlan `json:"plans"`
	LegacyPlans []ModelPricePlan `json:"legacy_plans"`
}

// ModelPricePlanQuery identifies the request dimensions used to select a
// component-price plan. EffectiveAt is a Unix timestamp supplied by the
// caller, which keeps resolution deterministic and makes time-window tests
// independent of the wall clock.
type ModelPricePlanQuery struct {
	ModelName      string
	Endpoint       string
	EffectiveGroup string
	ServiceTier    string
	ContextTokens  int
	EffectiveAt    int64
}

// ResolvedModelPriceComponent records the exact component and plan selected
// for a request dimension. It is intentionally separate from billing so stage
// 4 can consume the same resolver without changing stage 3 persistence.
type ResolvedModelPriceComponent struct {
	PlanID      int64               `json:"plan_id,omitempty"`
	Component   ModelPriceComponent `json:"component"`
	PlanSource  PricePlanSource     `json:"plan_source"`
	BillingMode BillingMode         `json:"billing_mode"`
}

// LegacyPriceInput is the lossless bridge from the existing ratio maps into
// component-price plans. The values remain in their legacy stores; this is a
// read-only projection and never migrates or rewrites old configuration.
type LegacyPriceInput struct {
	ModelName            string
	ModelPrice           *float64
	HasModelRatio        bool
	ModelRatio           float64
	CompletionRatio      float64
	CacheRatio           float64
	CacheCreationRatio   float64
	CacheCreation1hRatio float64
	AudioRatio           float64
	AudioCompletionRatio float64
	// QuotaPerUnit is the current quota-points-per-USD setting. Legacy token
	// ratios are quota points per token, so its value is required to project an
	// equivalent per-million-token price instead of assuming the default.
	QuotaPerUnit   float64
	ContextPricing *ContextPricingConfig
}

func LegacyPricePlans(input LegacyPriceInput) []ModelPricePlan {
	if input.ModelName == "" {
		return nil
	}
	quotaPerUnit := input.QuotaPerUnit
	if !(quotaPerUnit > 0) {
		// Keep the leaf contract backward compatible for callers that predate
		// the explicit field. Store-backed callers always provide the live value.
		quotaPerUnit = legacyDefaultQuotaPerUnit
	}
	if input.ContextPricing != nil && input.ContextPricing.Enabled && len(input.ContextPricing.Tiers) > 0 {
		plans := make([]ModelPricePlan, 0, len(input.ContextPricing.Tiers))
		for _, tier := range input.ContextPricing.Tiers {
			plans = append(plans, legacyTokenPlan(
				input.ModelName,
				tier.MinTokens,
				tier.MaxTokens,
				tier.ModelRatio,
				tier.CompletionRatio,
				tier.CacheRatio,
				tier.CreateCacheRatio,
				tier.CreateCacheRatio*1.6,
				tier.AudioRatio,
				tier.AudioCompletionRatio,
				quotaPerUnit,
			))
		}
		return plans
	}
	if input.ModelPrice != nil {
		if *input.ModelPrice == 0 {
			return []ModelPricePlan{legacyFreePlan(input.ModelName, 0, nil)}
		}
		return []ModelPricePlan{legacyPerRequestPlan(input.ModelName, *input.ModelPrice)}
	}
	if !input.HasModelRatio {
		return nil
	}
	return []ModelPricePlan{legacyTokenPlan(
		input.ModelName,
		0,
		nil,
		input.ModelRatio,
		input.CompletionRatio,
		input.CacheRatio,
		input.CacheCreationRatio,
		input.CacheCreation1hRatio,
		input.AudioRatio,
		input.AudioCompletionRatio,
		quotaPerUnit,
	)}
}

const (
	legacyTokensPerMillion    = 1_000_000.0
	legacyDefaultQuotaPerUnit = 500_000.0
)

func legacyBasePlan(modelName string, minTokens int, maxTokens *int) ModelPricePlan {
	return ModelPricePlan{
		ModelName:             modelName,
		ContextMinTokens:      minTokens,
		ContextMaxTokens:      maxTokens,
		Components:            make([]ModelPriceComponent, 0),
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        18,
		RoundingMode:          PriceRoundingHalfUp,
		GroupMultiplierSource: GroupMultiplierSourceInherit,
		Source:                PricePlanSourceLegacy,
		ReadOnly:              true,
	}
}

func legacyFreePlan(modelName string, minTokens int, maxTokens *int) ModelPricePlan {
	plan := legacyBasePlan(modelName, minTokens, maxTokens)
	plan.BillingMode = BillingModeFree
	return plan
}

func legacyPerRequestPlan(modelName string, modelPrice float64) ModelPricePlan {
	plan := legacyBasePlan(modelName, 0, nil)
	plan.BillingMode = BillingModePerRequest
	plan.Components = []ModelPriceComponent{{
		Component: PriceComponentRequest,
		Unit:      PriceUnitPerRequest,
		UnitPrice: legacyPriceString(modelPrice),
	}}
	return plan
}

func legacyTokenPlan(
	modelName string,
	minTokens int,
	maxTokens *int,
	modelRatio float64,
	completionRatio float64,
	cacheRatio float64,
	cacheCreation5mRatio float64,
	cacheCreation1hRatio float64,
	audioRatio float64,
	audioCompletionRatio float64,
	quotaPerUnit float64,
) ModelPricePlan {
	if modelRatio == 0 {
		return legacyFreePlan(modelName, minTokens, maxTokens)
	}
	plan := legacyBasePlan(modelName, minTokens, maxTokens)
	plan.BillingMode = BillingModeToken
	plan.Components = append(
		legacyInputComponents(modelRatio, audioRatio, quotaPerUnit),
		legacyOutputComponents(modelRatio, completionRatio, audioRatio, audioCompletionRatio, quotaPerUnit)...,
	)
	plan.Components = append(plan.Components,
		legacyTokenComponent(PriceComponentCacheRead, modelRatio*cacheRatio, quotaPerUnit),
		legacyTokenComponent(PriceComponentCacheWrite5m, modelRatio*cacheCreation5mRatio, quotaPerUnit),
		legacyTokenComponent(PriceComponentCacheWrite1h, modelRatio*cacheCreation1hRatio, quotaPerUnit),
	)
	return plan
}

// Legacy ratio pricing has one generic input/output price plus optional audio
// overrides. A component plan may not contain a parent and a child together,
// so an audio override expands the generic side into its full child set. This
// preserves the old prices without producing a configuration that would charge
// the same tokens twice when stage 4 consumes the table.
func legacyInputComponents(modelRatio, audioRatio, quotaPerUnit float64) []ModelPriceComponent {
	basePrice := legacyTokenUnitPrice(modelRatio, quotaPerUnit)
	audioPrice := legacyTokenUnitPrice(modelRatio*audioRatio, quotaPerUnit)
	if basePrice == audioPrice {
		return []ModelPriceComponent{legacyTokenComponent(PriceComponentInput, modelRatio, quotaPerUnit)}
	}
	components := make([]ModelPriceComponent, 0, len(InputChildPriceComponents))
	for _, component := range InputChildPriceComponents {
		ratio := modelRatio
		if component == PriceComponentAudioInput {
			ratio = modelRatio * audioRatio
		}
		components = append(components, legacyTokenComponent(component, ratio, quotaPerUnit))
	}
	return components
}

func legacyOutputComponents(modelRatio, completionRatio, audioRatio, audioCompletionRatio, quotaPerUnit float64) []ModelPriceComponent {
	baseRatio := modelRatio * completionRatio
	audioRatioValue := modelRatio * audioRatio * audioCompletionRatio
	if legacyTokenUnitPrice(baseRatio, quotaPerUnit) == legacyTokenUnitPrice(audioRatioValue, quotaPerUnit) {
		return []ModelPriceComponent{legacyTokenComponent(PriceComponentOutput, baseRatio, quotaPerUnit)}
	}
	components := make([]ModelPriceComponent, 0, len(OutputChildPriceComponents))
	for _, component := range OutputChildPriceComponents {
		ratio := baseRatio
		if component == PriceComponentAudioOutput {
			ratio = audioRatioValue
		}
		components = append(components, legacyTokenComponent(component, ratio, quotaPerUnit))
	}
	return components
}

func legacyTokenComponent(component PriceComponent, ratio, quotaPerUnit float64) ModelPriceComponent {
	return ModelPriceComponent{
		Component: component,
		Unit:      PriceUnitPerMillionTokens,
		UnitPrice: legacyTokenUnitPrice(ratio, quotaPerUnit),
	}
}

func legacyTokenUnitPrice(ratio, quotaPerUnit float64) string {
	// A legacy ratio is quota points per token. Convert it through the live
	// quota exchange before displaying the component as USD per million tokens.
	return legacyPriceString(ratio * legacyTokensPerMillion / quotaPerUnit)
}

func legacyPriceString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
