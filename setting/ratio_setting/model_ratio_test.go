package ratio_setting

import "testing"

func TestGetModelRatioOrPriceAcceptsContextPricingOnlyModel(t *testing.T) {
	if err := UpdateContextPricingByJSONString(contextPricingTestConfig); err != nil {
		t.Fatalf("UpdateContextPricingByJSONString returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = UpdateContextPricingByJSONString("{}")
	})

	_, _, exist := GetModelRatioOrPrice("tier-test-model")
	if !exist {
		t.Fatal("context pricing only model should be treated as priced")
	}
}
