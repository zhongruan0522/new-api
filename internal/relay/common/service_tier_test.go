package common

import "testing"

func TestSetEffectiveServiceTierIgnoresEmptyEvents(t *testing.T) {
	info := &RelayInfo{}
	info.SetEffectiveServiceTier("flex")
	info.SetEffectiveServiceTier("")
	if info.ServiceTierEffective != "flex" {
		t.Fatalf("ServiceTierEffective = %q, want flex", info.ServiceTierEffective)
	}

	var nilInfo *RelayInfo
	nilInfo.SetEffectiveServiceTier("flex")
}
