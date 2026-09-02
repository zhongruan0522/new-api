package billing

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

const (
	maxPricePlanModelNameLength = 255
	maxPricePlanScopeLength     = 128
	// Legacy ratio maps are float64 values. Their shortest lossless decimal
	// representation can require up to 17 fractional digits after conversion to
	// a per-million-token price, so 18 places keeps compatibility projections
	// transferable into an explicit component plan without rounding them first.
	maxPricePlanPrecision      = 18
	maxUnboundedPricePlanRange = int64(^uint64(0) >> 1)
)

var maxPricePlanValue = decimal.NewFromInt(1000000000000)

// NormalizeAndValidateModelPricePlans is the server-side authority for price
// table configuration. The UI may provide early feedback, but no request can
// bypass these unit, decimal, scope, time-range, and subset checks.
func NormalizeAndValidateModelPricePlans(plans []contract.ModelPricePlan) ([]contract.ModelPricePlan, error) {
	normalized := make([]contract.ModelPricePlan, len(plans))
	for i := range plans {
		plan, err := normalizeAndValidateModelPricePlan(plans[i])
		if err != nil {
			return nil, fmt.Errorf("price plan %d: %w", i+1, err)
		}
		normalized[i] = plan
	}
	if err := validateModelPricePlanOverlaps(normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

// ResolveModelPricePlan selects the highest-precedence active price plan for a
// request. Explicit component plans always outrank compatibility projections;
// within one source, a group-specific plan wins over endpoint, service-tier,
// context, and time fallbacks. Equal candidates retain their caller order,
// which is stable for persisted plans and makes the resolver deterministic.
func ResolveModelPricePlan(plans []contract.ModelPricePlan, query contract.ModelPricePlanQuery) (*contract.ModelPricePlan, bool) {
	candidates := matchingModelPricePlans(plans, query, false)
	if len(candidates) == 0 {
		return nil, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return modelPricePlanPrecedes(candidates[i], candidates[j])
	})
	selected := cloneModelPricePlan(candidates[0])
	return &selected, true
}

// ResolveModelPriceComponent resolves one token component after the effective
// plan has established the billing mode. A token plan may deliberately omit a
// component, in which case lower-precedence plans (including the legacy
// projection) remain eligible. Parent input/output prices apply to their
// children only when that plan does not provide a more specific component.
func ResolveModelPriceComponent(plans []contract.ModelPricePlan, query contract.ModelPricePlanQuery, component contract.PriceComponent) (*contract.ResolvedModelPriceComponent, bool) {
	effectivePlan, ok := ResolveModelPricePlan(plans, query)
	if !ok || effectivePlan.BillingMode != contract.BillingModeToken {
		return nil, false
	}

	candidates := matchingModelPricePlans(plans, query, true)
	sort.SliceStable(candidates, func(i, j int) bool {
		return modelPricePlanPrecedes(candidates[i], candidates[j])
	})
	for _, plan := range candidates {
		// A split-component override means consumers must settle individual
		// modalities. A parent may still be supplied by a higher-precedence
		// plan, but once a split plan outranks every remaining parent-only
		// fallback, returning that parent would let stage 4 combine it with a
		// child price and charge the same tokens twice.
		if component == contract.PriceComponentInput && planHasChildComponent(plan, contract.InputChildPriceComponents) {
			return nil, false
		}
		if component == contract.PriceComponentOutput && planHasChildComponent(plan, contract.OutputChildPriceComponents) {
			return nil, false
		}
		if price, found := findPlanComponent(plan, component); found {
			return &contract.ResolvedModelPriceComponent{
				PlanID:      plan.ID,
				Component:   price,
				PlanSource:  normalizedPricePlanSource(plan.Source),
				BillingMode: plan.BillingMode,
				Plan:        cloneModelPricePlan(plan),
			}, true
		}
	}
	return nil, false
}

func matchingModelPricePlans(plans []contract.ModelPricePlan, query contract.ModelPricePlanQuery, tokenOnly bool) []contract.ModelPricePlan {
	candidates := make([]contract.ModelPricePlan, 0, len(plans))
	for _, plan := range plans {
		if tokenOnly && plan.BillingMode != contract.BillingModeToken {
			continue
		}
		if modelPricePlanMatches(plan, query) {
			candidates = append(candidates, plan)
		}
	}
	return candidates
}

func modelPricePlanMatches(plan contract.ModelPricePlan, query contract.ModelPricePlanQuery) bool {
	if plan.ModelName != query.ModelName {
		return false
	}
	if plan.Endpoint != "" && plan.Endpoint != query.Endpoint {
		return false
	}
	if plan.EffectiveGroup != "" && plan.EffectiveGroup != query.EffectiveGroup {
		return false
	}
	if plan.ServiceTier != "" && plan.ServiceTier != query.ServiceTier {
		return false
	}
	if query.ContextTokens < plan.ContextMinTokens {
		return false
	}
	if plan.ContextMaxTokens != nil && query.ContextTokens >= *plan.ContextMaxTokens {
		return false
	}
	if plan.EffectiveFrom != nil && query.EffectiveAt < *plan.EffectiveFrom {
		return false
	}
	if plan.EffectiveUntil != nil && query.EffectiveAt >= *plan.EffectiveUntil {
		return false
	}
	return true
}

func modelPricePlanPrecedes(left, right contract.ModelPricePlan) bool {
	leftSource := pricePlanSourcePriority(normalizedPricePlanSource(left.Source))
	rightSource := pricePlanSourcePriority(normalizedPricePlanSource(right.Source))
	if leftSource != rightSource {
		return leftSource > rightSource
	}

	// Group scope is the most user-specific override. The remaining dimensions
	// follow in descending request specificity, then narrower ranges win.
	for _, compare := range []int{
		boolToInt(left.EffectiveGroup != "") - boolToInt(right.EffectiveGroup != ""),
		boolToInt(left.Endpoint != "") - boolToInt(right.Endpoint != ""),
		boolToInt(left.ServiceTier != "") - boolToInt(right.ServiceTier != ""),
		boolToInt(hasContextScope(left)) - boolToInt(hasContextScope(right)),
		boolToInt(hasEffectiveTimeScope(left)) - boolToInt(hasEffectiveTimeScope(right)),
	} {
		if compare != 0 {
			return compare > 0
		}
	}
	if contextRangeWidth(left) != contextRangeWidth(right) {
		return contextRangeWidth(left) < contextRangeWidth(right)
	}
	if effectiveRangeWidth(left) != effectiveRangeWidth(right) {
		return effectiveRangeWidth(left) < effectiveRangeWidth(right)
	}
	return false
}

func findPlanComponent(plan contract.ModelPricePlan, requested contract.PriceComponent) (contract.ModelPriceComponent, bool) {
	for _, component := range plan.Components {
		if component.Component == requested {
			return component, true
		}
	}
	// Reasoning is an output usage split, not a billable component of its own.
	// It therefore follows the ordinary text-output price when one is explicit,
	// and otherwise falls through to the generic output parent below.
	if requested == contract.PriceComponentReasoningOutput {
		for _, component := range plan.Components {
			if component.Component == contract.PriceComponentTextOutput {
				return component, true
			}
		}
	}
	parent, hasParent := priceComponentParent(requested)
	if requested == contract.PriceComponentReasoningOutput {
		parent, hasParent = contract.PriceComponentOutput, true
	}
	if !hasParent {
		return contract.ModelPriceComponent{}, false
	}
	for _, component := range plan.Components {
		if component.Component == parent {
			return component, true
		}
	}
	return contract.ModelPriceComponent{}, false
}

func planHasChildComponent(plan contract.ModelPricePlan, children []contract.PriceComponent) bool {
	for _, configured := range plan.Components {
		for _, child := range children {
			if configured.Component == child {
				return true
			}
		}
	}
	return false
}

func priceComponentParent(component contract.PriceComponent) (contract.PriceComponent, bool) {
	for _, child := range contract.InputChildPriceComponents {
		if component == child {
			return contract.PriceComponentInput, true
		}
	}
	for _, child := range contract.OutputChildPriceComponents {
		if component == child {
			return contract.PriceComponentOutput, true
		}
	}
	return "", false
}

func normalizedPricePlanSource(source contract.PricePlanSource) contract.PricePlanSource {
	if source == contract.PricePlanSourceLegacy {
		return contract.PricePlanSourceLegacy
	}
	return contract.PricePlanSourceExplicit
}

func pricePlanSourcePriority(source contract.PricePlanSource) int {
	if source == contract.PricePlanSourceLegacy {
		return 0
	}
	return 1
}

func hasContextScope(plan contract.ModelPricePlan) bool {
	return plan.ContextMinTokens != 0 || plan.ContextMaxTokens != nil
}

func hasEffectiveTimeScope(plan contract.ModelPricePlan) bool {
	return plan.EffectiveFrom != nil || plan.EffectiveUntil != nil
}

func contextRangeWidth(plan contract.ModelPricePlan) int64 {
	if plan.ContextMaxTokens == nil {
		return maxUnboundedPricePlanRange
	}
	return int64(*plan.ContextMaxTokens - plan.ContextMinTokens)
}

func effectiveRangeWidth(plan contract.ModelPricePlan) int64 {
	if plan.EffectiveUntil == nil {
		return maxUnboundedPricePlanRange
	}
	start := int64(0)
	if plan.EffectiveFrom != nil {
		start = *plan.EffectiveFrom
	}
	return *plan.EffectiveUntil - start
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func cloneModelPricePlan(plan contract.ModelPricePlan) contract.ModelPricePlan {
	cloned := plan
	if plan.ContextMaxTokens != nil {
		value := *plan.ContextMaxTokens
		cloned.ContextMaxTokens = &value
	}
	if plan.EffectiveFrom != nil {
		value := *plan.EffectiveFrom
		cloned.EffectiveFrom = &value
	}
	if plan.EffectiveUntil != nil {
		value := *plan.EffectiveUntil
		cloned.EffectiveUntil = &value
	}
	cloned.Components = append([]contract.ModelPriceComponent(nil), plan.Components...)
	return cloned
}

func normalizeAndValidateModelPricePlan(plan contract.ModelPricePlan) (contract.ModelPricePlan, error) {
	plan.ModelName = strings.TrimSpace(plan.ModelName)
	if plan.ModelName == "" {
		return contract.ModelPricePlan{}, fmt.Errorf("model_name is required")
	}
	if len(plan.ModelName) > maxPricePlanModelNameLength {
		return contract.ModelPricePlan{}, fmt.Errorf("model_name exceeds %d characters", maxPricePlanModelNameLength)
	}

	var err error
	if plan.Endpoint, err = normalizePricePlanScope("endpoint", plan.Endpoint); err != nil {
		return contract.ModelPricePlan{}, err
	}
	if plan.EffectiveGroup, err = normalizePricePlanScope("effective_group", plan.EffectiveGroup); err != nil {
		return contract.ModelPricePlan{}, err
	}
	if plan.ServiceTier, err = normalizePricePlanScope("service_tier", plan.ServiceTier); err != nil {
		return contract.ModelPricePlan{}, err
	}

	if plan.ContextMinTokens < 0 {
		return contract.ModelPricePlan{}, fmt.Errorf("context_min_tokens must be >= 0")
	}
	if plan.ContextMaxTokens != nil && *plan.ContextMaxTokens <= plan.ContextMinTokens {
		return contract.ModelPricePlan{}, fmt.Errorf("context_max_tokens must be greater than context_min_tokens")
	}
	if plan.EffectiveFrom != nil && *plan.EffectiveFrom < 0 {
		return contract.ModelPricePlan{}, fmt.Errorf("effective_from must be >= 0")
	}
	if plan.EffectiveUntil != nil && *plan.EffectiveUntil < 0 {
		return contract.ModelPricePlan{}, fmt.Errorf("effective_until must be >= 0")
	}
	if plan.EffectiveFrom != nil && plan.EffectiveUntil != nil && *plan.EffectiveUntil <= *plan.EffectiveFrom {
		return contract.ModelPricePlan{}, fmt.Errorf("effective_until must be later than effective_from")
	}

	plan.Currency = strings.ToUpper(strings.TrimSpace(plan.Currency))
	if _, err := currency.ParseISO(plan.Currency); err != nil {
		return contract.ModelPricePlan{}, fmt.Errorf("currency must be a three-letter ISO code")
	}
	if plan.PricePrecision < 0 || plan.PricePrecision > maxPricePlanPrecision {
		return contract.ModelPricePlan{}, fmt.Errorf("price_precision must be between 0 and %d", maxPricePlanPrecision)
	}
	if !validRoundingMode(plan.RoundingMode) {
		return contract.ModelPricePlan{}, fmt.Errorf("rounding_mode is invalid")
	}
	if err := validatePositiveDecimal("exchange_rate", plan.ExchangeRate, maxPricePlanPrecision); err != nil {
		return contract.ModelPricePlan{}, err
	}
	plan.ExchangeRate = strings.TrimSpace(plan.ExchangeRate)

	if !validGroupMultiplierSource(plan.GroupMultiplierSource) {
		return contract.ModelPricePlan{}, fmt.Errorf("group_multiplier_source is invalid")
	}
	plan.GroupMultiplier = strings.TrimSpace(plan.GroupMultiplier)
	switch plan.GroupMultiplierSource {
	case contract.GroupMultiplierSourceInherit:
		if plan.GroupMultiplier != "" {
			return contract.ModelPricePlan{}, fmt.Errorf("group_multiplier must be empty when group_multiplier_source inherits the group ratio")
		}
	case contract.GroupMultiplierSourceFixed:
		if err := validateNonNegativeDecimal("group_multiplier", plan.GroupMultiplier, maxPricePlanPrecision); err != nil {
			return contract.ModelPricePlan{}, err
		}
	}

	if err := validatePricePlanComponents(&plan); err != nil {
		return contract.ModelPricePlan{}, err
	}
	plan.Source = contract.PricePlanSourceExplicit
	plan.ReadOnly = false
	plan.ID = 0
	plan.CreatedAt = 0
	plan.UpdatedAt = 0
	return plan, nil
}

func normalizePricePlanScope(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > maxPricePlanScopeLength {
		return "", fmt.Errorf("%s exceeds %d characters", name, maxPricePlanScopeLength)
	}
	// Existing channel and user-group configuration accepts human-readable names
	// rather than a restricted ASCII identifier. Preserve those valid names here,
	// while rejecting control characters and commas, which are channel-group list
	// separators and would make an effective group ambiguous.
	if strings.ContainsRune(value, ',') || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%s must not contain control characters or commas", name)
	}
	return value, nil
}

func validatePricePlanComponents(plan *contract.ModelPricePlan) error {
	switch plan.BillingMode {
	case contract.BillingModeFree:
		if len(plan.Components) != 0 {
			return fmt.Errorf("free plans cannot contain components")
		}
		// Keep successful write responses consistent with persisted reads. A free
		// plan intentionally has no components, but its JSON contract is [] rather
		// than null so consumers can safely treat components as a collection.
		plan.Components = []contract.ModelPriceComponent{}
		return nil
	case contract.BillingModePerRequest:
		if len(plan.Components) != 1 || plan.Components[0].Component != contract.PriceComponentRequest {
			return fmt.Errorf("per_request plans require exactly one request component")
		}
		if plan.Components[0].Unit != contract.PriceUnitPerRequest {
			return fmt.Errorf("request component must use per_request")
		}
		if err := validatePositiveDecimal("request unit_price", plan.Components[0].UnitPrice, plan.PricePrecision); err != nil {
			return err
		}
		plan.Components[0].UnitPrice = strings.TrimSpace(plan.Components[0].UnitPrice)
		return nil
	case contract.BillingModeToken:
		if len(plan.Components) == 0 {
			return fmt.Errorf("token plans require at least one component")
		}
	default:
		return fmt.Errorf("billing_mode is invalid")
	}

	seen := make(map[contract.PriceComponent]struct{}, len(plan.Components))
	for i := range plan.Components {
		component := &plan.Components[i]
		if component.Component == contract.PriceComponentReasoningOutput {
			return fmt.Errorf("reasoning_output is an output split, not a billable component")
		}
		if !isTokenPriceComponent(component.Component) {
			return fmt.Errorf("component %q is invalid for token pricing", component.Component)
		}
		if component.Unit != contract.PriceUnitPerMillionTokens {
			return fmt.Errorf("component %q must use per_1m_tokens", component.Component)
		}
		if _, exists := seen[component.Component]; exists {
			return fmt.Errorf("component %q is configured more than once", component.Component)
		}
		seen[component.Component] = struct{}{}
		if err := validateNonNegativeDecimal(fmt.Sprintf("%s unit_price", component.Component), component.UnitPrice, plan.PricePrecision); err != nil {
			return err
		}
		component.UnitPrice = strings.TrimSpace(component.UnitPrice)
	}
	if parentAndChildConfigured(seen, contract.PriceComponentInput, contract.InputChildPriceComponents) {
		return fmt.Errorf("input cannot be configured together with input child components")
	}
	if parentAndChildConfigured(seen, contract.PriceComponentOutput, contract.OutputChildPriceComponents) {
		return fmt.Errorf("output cannot be configured together with output child components")
	}
	sort.Slice(plan.Components, func(i, j int) bool {
		return plan.Components[i].Component < plan.Components[j].Component
	})
	return nil
}

func isTokenPriceComponent(component contract.PriceComponent) bool {
	for _, candidate := range contract.TokenPriceComponents {
		if component == candidate {
			return true
		}
	}
	return false
}

func parentAndChildConfigured(seen map[contract.PriceComponent]struct{}, parent contract.PriceComponent, children []contract.PriceComponent) bool {
	if _, exists := seen[parent]; !exists {
		return false
	}
	for _, child := range children {
		if _, exists := seen[child]; exists {
			return true
		}
	}
	return false
}

func validRoundingMode(mode contract.PriceRoundingMode) bool {
	switch mode {
	case contract.PriceRoundingHalfUp, contract.PriceRoundingHalfEven, contract.PriceRoundingFloor, contract.PriceRoundingCeil:
		return true
	default:
		return false
	}
}

func validGroupMultiplierSource(source contract.GroupMultiplierSource) bool {
	return source == contract.GroupMultiplierSourceInherit || source == contract.GroupMultiplierSourceFixed
}

func validatePositiveDecimal(name, value string, maxScale int) error {
	decimalValue, err := parsePricePlanDecimal(name, value, maxScale)
	if err != nil {
		return err
	}
	if !decimalValue.GreaterThan(decimal.Zero) {
		return fmt.Errorf("%s must be greater than zero", name)
	}
	return nil
}

func validateNonNegativeDecimal(name, value string, maxScale int) error {
	decimalValue, err := parsePricePlanDecimal(name, value, maxScale)
	if err != nil {
		return err
	}
	if decimalValue.IsNegative() {
		return fmt.Errorf("%s must be >= 0", name)
	}
	return nil
}

func parsePricePlanDecimal(name, value string, maxScale int) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return decimal.Zero, fmt.Errorf("%s is required", name)
	}
	decimalValue, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%s must be a decimal", name)
	}
	if decimalValue.Exponent() < -int32(maxScale) {
		return decimal.Zero, fmt.Errorf("%s exceeds %d decimal places", name, maxScale)
	}
	if decimalValue.Abs().GreaterThan(maxPricePlanValue) {
		return decimal.Zero, fmt.Errorf("%s exceeds the supported range", name)
	}
	return decimalValue, nil
}

func validateModelPricePlanOverlaps(plans []contract.ModelPricePlan) error {
	for left := 0; left < len(plans); left++ {
		for right := left + 1; right < len(plans); right++ {
			if !samePricePlanScope(plans[left], plans[right]) {
				continue
			}
			if contextRangesOverlap(plans[left], plans[right]) && effectiveRangesOverlap(plans[left], plans[right]) {
				return fmt.Errorf("price plans %d and %d have overlapping model, endpoint, group, service tier, context, and effective-time scopes", left+1, right+1)
			}
		}
	}
	return nil
}

func samePricePlanScope(left, right contract.ModelPricePlan) bool {
	return left.ModelName == right.ModelName &&
		left.Endpoint == right.Endpoint &&
		left.EffectiveGroup == right.EffectiveGroup &&
		left.ServiceTier == right.ServiceTier
}

func contextRangesOverlap(left, right contract.ModelPricePlan) bool {
	return rangesOverlap(int64(left.ContextMinTokens), intPointerToInt64(left.ContextMaxTokens), int64(right.ContextMinTokens), intPointerToInt64(right.ContextMaxTokens))
}

func effectiveRangesOverlap(left, right contract.ModelPricePlan) bool {
	return rangesOverlap(int64PointerValue(left.EffectiveFrom), left.EffectiveUntil, int64PointerValue(right.EffectiveFrom), right.EffectiveUntil)
}

func rangesOverlap(leftStart int64, leftEnd *int64, rightStart int64, rightEnd *int64) bool {
	if leftEnd != nil && *leftEnd <= rightStart {
		return false
	}
	if rightEnd != nil && *rightEnd <= leftStart {
		return false
	}
	return true
}

func intPointerToInt64(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func int64PointerValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
