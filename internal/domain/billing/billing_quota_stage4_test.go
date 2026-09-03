package billing

import (
	"fmt"
	"testing"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/billing/contract"
	"github.com/NookMux/NookMux/internal/domain/shared"
	dbstore "github.com/NookMux/NookMux/internal/store/db"
	logstore "github.com/NookMux/NookMux/internal/store/log"
	pricingstore "github.com/NookMux/NookMux/internal/store/pricing"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupEmptyExplicitPricePlans gives CalculateUsage tests a real, empty
// component table without replacing the lossless legacy fallback.
func setupEmptyExplicitPricePlans(t *testing.T) {
	t.Helper()
	oldDB := dbstore.DB
	oldLogDB := dbstore.LOG_DB
	testDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open empty price plan sqlite: %v", err)
	}
	if err := testDB.AutoMigrate(&pricingstore.ModelPricePlan{}, &pricingstore.ModelPriceComponent{}); err != nil {
		t.Fatalf("migrate empty price plan sqlite: %v", err)
	}
	dbstore.DB = testDB
	dbstore.LOG_DB = testDB
	pricingstore.InvalidateModelPricePlanCache()
	t.Cleanup(func() {
		if sqlDB, err := testDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		dbstore.DB = oldDB
		dbstore.LOG_DB = oldLogDB
		pricingstore.InvalidateModelPricePlanCache()
	})
}

func stage4TokenPlan(id int64, components ...contract.ModelPriceComponent) contract.ModelPricePlan {
	return contract.ModelPricePlan{
		ID:                    id,
		ModelName:             "model-a",
		BillingMode:           contract.BillingModeToken,
		Currency:              "USD",
		ExchangeRate:          "1",
		PricePrecision:        12,
		RoundingMode:          contract.PriceRoundingHalfUp,
		GroupMultiplierSource: contract.GroupMultiplierSourceInherit,
		Components:            components,
		Source:                contract.PricePlanSourceExplicit,
	}
}

func stage4Component(component contract.PriceComponent, price string) contract.ModelPriceComponent {
	return contract.ModelPriceComponent{Component: component, Unit: contract.PriceUnitPerMillionTokens, UnitPrice: price}
}

func stage4Usage() *BillingUsage {
	write5m := 50
	return &BillingUsage{
		PromptAggregateTokens: 850,
		OutputTokens:          500,
		CacheReadTokens:       100,
		CacheWriteTokens:      50,
		CacheWrite5mTokens:    &write5m,
	}
}

func TestStage4ExplicitTokenPlanMatchesEquivalentLegacyQuota(t *testing.T) {
	bu := stage4Usage()
	priceData := normalizedQuotaTestPriceData()
	legacy := mustQuota(t, bu, priceData, AudioPricingAbsolute, "model-a")

	// The legacy ratio projection converts quota points/token into USD/1M:
	// ratio * 1,000,000 / QuotaPerUnit.
	quotaPerUnit := decimal.NewFromInt(500_000)
	legacyPrice := func(ratio string, multiplier string) string {
		r, err := decimal.NewFromString(ratio)
		if err != nil {
			t.Fatal(err)
		}
		m, err := decimal.NewFromString(multiplier)
		if err != nil {
			t.Fatal(err)
		}
		return r.Mul(m).Mul(decimal.NewFromInt(1_000_000)).Div(quotaPerUnit).String()
	}
	explicit := stage4TokenPlan(11,
		stage4Component(contract.PriceComponentInput, legacyPrice("2", "1")),
		stage4Component(contract.PriceComponentOutput, legacyPrice("2", "3")),
		stage4Component(contract.PriceComponentCacheRead, legacyPrice("2", "0.5")),
		stage4Component(contract.PriceComponentCacheWrite5m, legacyPrice("2", "1.25")),
		stage4Component(contract.PriceComponentCacheWrite1h, legacyPrice("2", "2")),
	)
	plans := append([]contract.ModelPricePlan{explicit}, legacyPricePlansForRelay("model-a", priceData)...)
	got, err := calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, plans, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("explicit token quota: %v", err)
	}
	if got.TokenTotal.Cmp(legacy.TokenTotal) != 0 {
		t.Fatalf("explicit quota %s != legacy quota %s", got.TokenTotal, legacy.TokenTotal)
	}
	if got.PriceSnapshot == nil || got.PriceSnapshot.Source != contract.PricePlanSourceExplicit {
		t.Fatalf("explicit snapshot = %+v", got.PriceSnapshot)
	}
	if len(got.PriceSnapshot.Components) != 4 {
		t.Fatalf("snapshot components = %+v", got.PriceSnapshot.Components)
	}
}

func TestStage4ChildComponentsDoNotDoubleChargeParents(t *testing.T) {
	bu := &BillingUsage{
		PromptAggregateTokens: 1000,
		OutputTokens:          500,
		AudioInputTokens:      intPtr(100),
		AudioOutputTokens:     intPtr(50),
	}
	priceData := normalizedQuotaTestPriceData()
	plans := []contract.ModelPricePlan{stage4TokenPlan(12,
		stage4Component(contract.PriceComponentTextInput, "8"),
		stage4Component(contract.PriceComponentTextOutput, "24"),
		stage4Component(contract.PriceComponentAudioInput, "32"),
		stage4Component(contract.PriceComponentAudioOutput, "64"),
	)}
	allPlans := append(plans, legacyPricePlansForRelay("model-a", priceData)...)
	result, err := calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, allPlans, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("split component quota: %v", err)
	}
	// TokenTotal holds 900 text input + 450 text output + 50 audio output.
	// The absolute audio input fee remains separate: 100 * 16 = 1,600.
	assertQuotaTotal(t, result, 900*4+450*12+50*32)
	if result.AudioInputQuota.Cmp(decimal.NewFromInt(1600)) != 0 {
		t.Fatalf("audio input quota = %s, want 1600", result.AudioInputQuota)
	}
	if result.AudioInputQuota.IsZero() {
		t.Fatal("absolute audio input must remain a separate fee")
	}
	if input, output := bu.InputTokens(), bu.OutputTokens; input != 1000 || output != 500 {
		t.Fatalf("billing usage mutated: input=%d output=%d", input, output)
	}
}

func TestStage4PerRequestFreeServiceTierAndContext(t *testing.T) {
	priceData := normalizedQuotaTestPriceData()
	bu := stage4Usage()

	perRequest := stage4TokenPlan(20, stage4Component(contract.PriceComponentInput, "1"))
	perRequest.BillingMode = contract.BillingModePerRequest
	perRequest.Components = []contract.ModelPriceComponent{{
		Component: contract.PriceComponentRequest,
		Unit:      contract.PriceUnitPerRequest,
		UnitPrice: "0.03",
	}}
	result, err := calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, []contract.ModelPricePlan{perRequest}, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("per-request quota: %v", err)
	}
	assertQuotaTotal(t, result, 15000)

	free := stage4TokenPlan(21)
	free.BillingMode = contract.BillingModeFree
	free.Components = nil
	result, err = calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, []contract.ModelPricePlan{free}, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("free quota: %v", err)
	}
	assertQuotaTotal(t, result, 0)

	priority := stage4TokenPlan(22,
		stage4Component(contract.PriceComponentInput, "4"),
		stage4Component(contract.PriceComponentOutput, "12"),
		stage4Component(contract.PriceComponentCacheRead, "2"),
		stage4Component(contract.PriceComponentCacheWrite5m, "5"),
		stage4Component(contract.PriceComponentCacheWrite1h, "4"),
	)
	priority.ServiceTier = "priority"
	priority.ContextMinTokens = 1000
	legacy := mustQuota(t, bu, priceData, AudioPricingAbsolute, "model-a")
	defaultPlan := stage4TokenPlan(23,
		stage4Component(contract.PriceComponentInput, "2"),
		stage4Component(contract.PriceComponentOutput, "6"),
		stage4Component(contract.PriceComponentCacheRead, "0.5"),
		stage4Component(contract.PriceComponentCacheWrite5m, "1.25"),
		stage4Component(contract.PriceComponentCacheWrite1h, "2"),
	)
	query := contract.ModelPricePlanQuery{ModelName: "model-a", ServiceTier: "priority", ContextTokens: 1400}
	result, err = calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, []contract.ModelPricePlan{priority, defaultPlan}, query)
	if err != nil {
		t.Fatalf("service-tier quota: %v", err)
	}
	if result.PriceSnapshot.ServiceTier != "priority" || result.PriceSnapshot.ContextMinTokens != 1000 {
		t.Fatalf("tier snapshot = %+v", result.PriceSnapshot)
	}
	// Priority scope is selected even when it is priced equivalently to the
	// legacy path; snapshot dimensions prove which plan won.
	if result.TokenTotal.Cmp(legacy.TokenTotal) != 0 {
		t.Logf("result lines: %+v; legacy lines: %+v", result.Lines, legacy.Lines)
		t.Fatalf("service-tier quota %s, want %s", result.TokenTotal, legacy.TokenTotal)
	}
}

func TestStage4FixedMultiplierExchangeAndFinalRounding(t *testing.T) {
	bu := &BillingUsage{PromptAggregateTokens: 1_000_000}
	priceData := normalizedQuotaTestPriceData()
	plan := stage4TokenPlan(30, stage4Component(contract.PriceComponentInput, "0.10101"))
	plan.Currency = "USD"
	plan.ExchangeRate = "1"
	plan.GroupMultiplierSource = contract.GroupMultiplierSourceFixed
	plan.GroupMultiplier = "1.1"
	plan.RoundingMode = contract.PriceRoundingFloor
	plans := append([]contract.ModelPricePlan{plan}, legacyPricePlansForRelay("model-a", priceData)...)
	result, err := calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, plans, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("fixed multiplier quota: %v", err)
	}
	// 1,000,000 * 0.10101 / 1,000,000 * 500,000 * 1.1 = 55,555.5. The
	// unrounded settlement stays exact; final rounding happens separately.
	if result.TokenTotal.Cmp(decimal.NewFromFloat(55555.5)) != 0 {
		t.Fatalf("unrounded total = %s, want 55555.5", result.TokenTotal)
	}
	if got := result.PriceSnapshot.GroupMultiplier; got != "1.1" {
		t.Fatalf("snapshot multiplier = %s, want 1.1", got)
	}
	if got := roundEntryQuota(result.TokenTotal, result); got != 55555 {
		t.Fatalf("explicit floor rounding = %d", got)
	}
}

func TestStage4FinalRoundingModes(t *testing.T) {
	half := decimal.NewFromInt(55555).Add(decimal.NewFromFloat(.5))
	oneAndHalf := decimal.NewFromInt(55556).Add(decimal.NewFromFloat(.5))
	tests := []struct {
		mode contract.PriceRoundingMode
		want int
	}{
		{contract.PriceRoundingHalfUp, 55556},
		{contract.PriceRoundingHalfEven, 55556},
		{contract.PriceRoundingFloor, 55555},
		{contract.PriceRoundingCeil, 55556},
	}
	for _, test := range tests {
		if got := RoundBillingQuota(half, test.mode); got != test.want {
			t.Fatalf("half value %s mode %s = %d, want %d", half, test.mode, got, test.want)
		}
	}
	if got := RoundBillingQuota(oneAndHalf, contract.PriceRoundingHalfEven); got != 55556 {
		t.Fatalf("banker's tie = %d, want 55556", got)
	}
}

func TestStage4CrossPlanFallbackRecordsComponentMultiplier(t *testing.T) {
	bu := stage4Usage()
	priceData := normalizedQuotaTestPriceData()
	priceData.GroupRatioInfo.GroupRatio = 3
	outer := stage4TokenPlan(40, stage4Component(contract.PriceComponentTextInput, "2"))
	outer.GroupMultiplierSource = contract.GroupMultiplierSourceFixed
	outer.GroupMultiplier = "2"
	plans := append([]contract.ModelPricePlan{outer}, legacyPricePlansForRelay("model-a", priceData)...)
	result, err := calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, plans, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("cross-plan quota: %v", err)
	}
	if result.PriceSnapshot.GroupMultiplier != "2" {
		t.Fatalf("effective multiplier = %s, want 2", result.PriceSnapshot.GroupMultiplier)
	}
	components := make(map[contract.PriceComponent]BillingPriceComponentSnapshot)
	for _, component := range result.PriceSnapshot.Components {
		components[component.Component] = component
	}
	if got := components[contract.PriceComponentTextInput].GroupMultiplier; got != "2" {
		t.Fatalf("input component multiplier = %s, want effective plan multiplier 2", got)
	}
	if got := components[contract.PriceComponentTextOutput].GroupMultiplier; got != "3" {
		t.Fatalf("fallback output multiplier = %s, want inherited multiplier 3", got)
	}
}

func TestStage4LegacyAbsoluteAudioSnapshotExplainsRemainders(t *testing.T) {
	bu := &BillingUsage{
		PromptAggregateTokens: 1000,
		OutputTokens:          500,
		AudioInputTokens:      intPtr(200),
	}
	priceData := normalizedQuotaTestPriceData()
	legacyPlan := legacyPricePlansForRelay("gemini-2.5-flash", priceData)[0]
	result, err := calculateNormalizedQuotaWithPlans(
		bu, priceData, AudioPricingAbsolute, []contract.ModelPricePlan{legacyPlan},
		contract.ModelPricePlanQuery{ModelName: "gemini-2.5-flash"},
	)
	if err != nil {
		t.Fatalf("legacy absolute audio quota: %v", err)
	}
	components := make(map[contract.PriceComponent]BillingPriceComponentSnapshot)
	for _, component := range result.PriceSnapshot.Components {
		components[component.Component] = component
	}
	if got := components[contract.PriceComponentTextInput]; got.UnitPrice != "4" {
		t.Fatalf("ordinary input snapshot = %+v", got)
	}
	if got := components[contract.PriceComponentTextOutput]; got.UnitPrice != "12" {
		t.Fatalf("ordinary output snapshot = %+v", got)
	}
	if got := components[contract.PriceComponentAudioInput]; got.UnitPrice != "1" || got.GroupMultiplier != "1" {
		t.Fatalf("audio input snapshot = %+v", got)
	}
	for _, duplicate := range []contract.PriceComponent{
		contract.PriceComponentInput,
		contract.PriceComponentAudioOutput,
	} {
		if _, ok := components[duplicate]; ok {
			t.Fatalf("unexpected %s snapshot component", duplicate)
		}
	}
}

func TestStage4SnapshotRemainsUsableAfterPriceMutation(t *testing.T) {
	bu := &BillingUsage{PromptAggregateTokens: 1_000_000, OutputTokens: 1}
	priceData := normalizedQuotaTestPriceData()
	plan := stage4TokenPlan(31,
		stage4Component(contract.PriceComponentInput, "4"),
		stage4Component(contract.PriceComponentOutput, "12"),
		stage4Component(contract.PriceComponentCacheRead, "1"),
		stage4Component(contract.PriceComponentCacheWrite5m, "2.5"),
		stage4Component(contract.PriceComponentCacheWrite1h, "4"),
	)
	result, err := calculateNormalizedQuotaWithPlans(bu, priceData, AudioPricingAbsolute, []contract.ModelPricePlan{plan}, contract.ModelPricePlanQuery{ModelName: "model-a"})
	if err != nil {
		t.Fatalf("snapshot quota: %v", err)
	}
	before := result.TokenTotal.String()
	for i := range plan.Components {
		plan.Components[i].UnitPrice = "999"
	}
	if result.TokenTotal.String() != before {
		t.Fatalf("settled decimal changed after plan mutation: %s != %s", result.TokenTotal, before)
	}
	if result.PriceSnapshot.Components[0].UnitPrice != "4" {
		t.Fatalf("snapshot price changed: %s", result.PriceSnapshot.Components[0].UnitPrice)
	}
}

func TestStage4ExplicitPlanSettlesAndPersistsSnapshot(t *testing.T) {
	setupApplyQuotaTestDB(t)
	plan := stage4TokenPlan(0,
		stage4Component(contract.PriceComponentTextInput, "4"),
		stage4Component(contract.PriceComponentTextOutput, "12"),
		stage4Component(contract.PriceComponentCacheRead, "2"),
	)
	plan.ModelName = "gpt-4o"
	plans, err := NormalizeAndValidateModelPricePlans([]contract.ModelPricePlan{plan})
	if err != nil {
		t.Fatalf("validate explicit plan: %v", err)
	}
	if err := pricingstore.ReplaceModelPricePlans(plans); err != nil {
		t.Fatalf("persist explicit plan: %v", err)
	}

	relayInfo := newApplyQuotaTestRelayInfo()
	usage := newApplyQuotaTestUsage()
	settlement, apiErr := CalculateUsage(newUsageTestContext(), relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("explicit plan CalculateUsage: %v", apiErr)
	}
	if settlement.quota != 460 {
		t.Fatalf("explicit plan quota = %d, want equivalent legacy quota 460", settlement.quota)
	}
	relayInfo.FinalPreConsumedQuota = settlement.quota
	if apiErr := ApplyQuota(newApplyQuotaTestContext(), relayInfo, settlement); apiErr != nil {
		t.Fatalf("explicit plan ApplyQuota: %v", apiErr)
	}

	var stored logstore.Log
	require.Eventually(t, func() bool {
		return dbstore.LOG_DB.Where("user_id = ?", applyQuotaTestUserId).
			Order("id DESC").First(&stored).Error == nil
	}, 2*time.Second, 10*time.Millisecond)
	other, err := common.StrToMap(stored.Other)
	if err != nil {
		t.Fatalf("parse consume-log Other: %v", err)
	}
	snapshot, ok := other["billing_price_snapshot"].(map[string]interface{})
	if !ok {
		t.Fatalf("billing price snapshot missing in %+v", other)
	}
	if snapshot["source"] != string(contract.PricePlanSourceExplicit) {
		t.Fatalf("snapshot source = %#v, want explicit", snapshot["source"])
	}
	components, ok := snapshot["components"].([]interface{})
	if !ok || len(components) != 3 {
		t.Fatalf("snapshot components = %#v, want three settled components", snapshot["components"])
	}

	// Replace the live table after settlement. The already persisted Other map
	// must continue to explain the original quota from its embedded snapshot.
	mutatedPlan := stage4TokenPlan(0,
		stage4Component(contract.PriceComponentTextInput, "999"),
		stage4Component(contract.PriceComponentTextOutput, "999"),
		stage4Component(contract.PriceComponentCacheRead, "999"),
	)
	mutatedPlan.ModelName = "gpt-4o"
	mutatedPlans, err := NormalizeAndValidateModelPricePlans([]contract.ModelPricePlan{mutatedPlan})
	if err != nil {
		t.Fatalf("validate mutated plan: %v", err)
	}
	if err := pricingstore.ReplaceModelPricePlans(mutatedPlans); err != nil {
		t.Fatalf("replace price table: %v", err)
	}
	inputComponent, ok := components[0].(map[string]interface{})
	if !ok || inputComponent["unit_price"] != "4" {
		t.Fatalf("persisted input snapshot = %#v, want original unit price 4", components[0])
	}
}

func installStage4Plans(t *testing.T, plan contract.ModelPricePlan) {
	t.Helper()
	plans, err := NormalizeAndValidateModelPricePlans([]contract.ModelPricePlan{plan})
	if err != nil {
		t.Fatalf("validate stage 4 plan: %v", err)
	}
	if err := pricingstore.ReplaceModelPricePlans(plans); err != nil {
		t.Fatalf("persist stage 4 plan: %v", err)
	}
}

func TestStage4ExplicitPerRequestProductionPath(t *testing.T) {
	setupApplyQuotaTestDB(t)
	plan := stage4TokenPlan(0)
	plan.ModelName = "gpt-4o"
	plan.BillingMode = contract.BillingModePerRequest
	plan.Components = []contract.ModelPriceComponent{{
		Component: contract.PriceComponentRequest,
		Unit:      contract.PriceUnitPerRequest,
		UnitPrice: "0.03",
	}}
	installStage4Plans(t, plan)

	settlement, apiErr := CalculateUsage(newUsageTestContext(), newApplyQuotaTestRelayInfo(), newApplyQuotaTestUsage())
	if apiErr != nil {
		t.Fatalf("per-request CalculateUsage: %v", apiErr)
	}
	if settlement.quota != 15000 {
		t.Fatalf("per-request quota = %d, want 15000", settlement.quota)
	}
}

func TestStage4ExplicitFreeProductionPath(t *testing.T) {
	setupApplyQuotaTestDB(t)
	plan := stage4TokenPlan(0)
	plan.ModelName = "gpt-4o"
	plan.BillingMode = contract.BillingModeFree
	plan.Components = nil
	installStage4Plans(t, plan)

	settlement, apiErr := CalculateUsage(newUsageTestContext(), newApplyQuotaTestRelayInfo(), newApplyQuotaTestUsage())
	if apiErr != nil {
		t.Fatalf("free CalculateUsage: %v", apiErr)
	}
	if settlement.quota != 0 {
		t.Fatalf("free quota = %d, want 0", settlement.quota)
	}
}

func TestStage4ExplicitServiceTierAndContextProductionPath(t *testing.T) {
	setupApplyQuotaTestDB(t)
	plan := stage4TokenPlan(0,
		stage4Component(contract.PriceComponentTextInput, "4"),
		stage4Component(contract.PriceComponentTextOutput, "12"),
	)
	plan.ModelName = "gpt-4o"
	plan.ServiceTier = "priority"
	plan.ContextMinTokens = 1000
	installStage4Plans(t, plan)

	relayInfo := newApplyQuotaTestRelayInfo()
	relayInfo.ServiceTierEffective = "priority"
	usage := &shared.Usage{PromptTokens: 1100, CompletionTokens: 0, TotalTokens: 1100}
	settlement, apiErr := CalculateUsage(newApplyQuotaTestContext(), relayInfo, usage)
	if apiErr != nil {
		t.Fatalf("scoped CalculateUsage: %v", apiErr)
	}
	// 1,100 ordinary input tokens at $4/1M with QuotaPerUnit=500,000.
	if settlement.quota != 2200 {
		t.Fatalf("scoped quota = %d, want 2200", settlement.quota)
	}
	if settlement.priceSnapshot == nil ||
		settlement.priceSnapshot.ServiceTier != "priority" ||
		settlement.priceSnapshot.ContextMinTokens != 1000 ||
		settlement.priceSnapshot.ContextTokens != 1100 {
		t.Fatalf("scoped snapshot = %+v", settlement.priceSnapshot)
	}
}

func intPtr(value int) *int {
	return &value
}
