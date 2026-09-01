package operation

import "testing"

func TestPricingAuditModuleDefaultsForPreExistingModuleMaps(t *testing.T) {
	previous := auditSetting
	t.Cleanup(func() {
		auditSetting = previous
	})

	auditSetting.Enabled = true
	auditSetting.Modules = `{"option":true}`
	if !IsAuditModuleEnabled("pricing") {
		t.Fatal("pricing audit module must default to enabled for an existing module map")
	}

	auditSetting.Modules = `{"pricing":false}`
	if IsAuditModuleEnabled("pricing") {
		t.Fatal("explicitly disabled pricing audit module must remain disabled")
	}
}
