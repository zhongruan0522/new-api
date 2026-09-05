package logcontroller

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/internal/config/console"
	"github.com/NookMux/NookMux/internal/store/log"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

func TestFilterHiddenUsageLogFieldsAppliesBillingDetailsVisibility(t *testing.T) {
	billingDetails := `{"schema_version":1,"tokens":{"input":{"text_input":12},"output":{"text_output":7,"reasoning_output":3},"cache":{"read_cache":4,"write_cache":5,"write_cache_5m":5}}}`
	oldFields := console.GetConsoleSetting().UsageLogFields
	oldUserDetailsEnabled := console.GetConsoleSetting().UsageLogFieldsUserEnabled
	t.Cleanup(func() {
		console.GetConsoleSetting().UsageLogFields = oldFields
		console.GetConsoleSetting().UsageLogFieldsUserEnabled = oldUserDetailsEnabled
	})

	console.GetConsoleSetting().UsageLogFields = `{"billing_details":{"admin":true,"user":true}}`

	visible := []*logstore.Log{{BillingDetails: &billingDetails}}
	filterHiddenUsageLogFields(visible)
	if visible[0].BillingDetails == nil || *visible[0].BillingDetails != billingDetails {
		t.Fatalf("visible billing_details = %v, want unchanged schema v1 payload", visible[0].BillingDetails)
	}

	console.GetConsoleSetting().UsageLogFields = `{"billing_details":{"admin":true,"user":false}}`
	hidden := []*logstore.Log{{BillingDetails: &billingDetails}}
	filterHiddenUsageLogFields(hidden)
	if hidden[0].BillingDetails != nil {
		t.Fatalf("hidden billing_details = %v, want nil", hidden[0].BillingDetails)
	}

	console.GetConsoleSetting().UsageLogFieldsUserEnabled = false
	detailsDisabled := []*logstore.Log{{BillingDetails: &billingDetails}}
	filterHiddenUsageLogFields(detailsDisabled)
	if detailsDisabled[0].BillingDetails != nil {
		t.Fatalf("billing_details with user details disabled = %v, want nil", detailsDisabled[0].BillingDetails)
	}
}

func TestFilterHiddenUsageLogFieldsAppliesPriceSnapshotVisibility(t *testing.T) {
	other := `{"billing_price_snapshot":{"source":"legacy","service_tier":"default"},"text_input":5}`
	oldFields := console.GetConsoleSetting().UsageLogFields
	oldUserDetailsEnabled := console.GetConsoleSetting().UsageLogFieldsUserEnabled
	t.Cleanup(func() {
		console.GetConsoleSetting().UsageLogFields = oldFields
		console.GetConsoleSetting().UsageLogFieldsUserEnabled = oldUserDetailsEnabled
	})

	console.GetConsoleSetting().UsageLogFields = `{"price_table":{"admin":true,"user":true}}`
	visible := []*logstore.Log{{Other: other}}
	filterHiddenUsageLogFields(visible)
	if !strings.Contains(visible[0].Other, "billing_price_snapshot") {
		t.Fatalf("visible price snapshot Other = %s, want snapshot retained", visible[0].Other)
	}

	console.GetConsoleSetting().UsageLogFields = `{"price_table":{"admin":true,"user":false}}`
	hidden := []*logstore.Log{{Other: other}}
	filterHiddenUsageLogFields(hidden)
	if strings.Contains(hidden[0].Other, "billing_price_snapshot") {
		t.Fatalf("hidden price snapshot Other = %s, want snapshot removed", hidden[0].Other)
	}
	if !strings.Contains(hidden[0].Other, `"text_input":5`) {
		t.Fatalf("hidden price snapshot Other = %s, want unrelated token fields retained", hidden[0].Other)
	}

	console.GetConsoleSetting().UsageLogFields = ""
	console.GetConsoleSetting().UsageLogFieldsUserEnabled = false
	detailsDisabled := []*logstore.Log{{Other: other}}
	filterHiddenUsageLogFields(detailsDisabled)
	if strings.Contains(detailsDisabled[0].Other, "billing_price_snapshot") {
		t.Fatalf("details disabled Other = %s, want price snapshot removed", detailsDisabled[0].Other)
	}
}

func TestBillingDetailsWireProjection(t *testing.T) {
	billingDetails := `{"schema_version":1,"tokens":{"input":{"text_input":12},"output":{},"cache":{}}}`

	encoded, err := jsonx.Marshal(logstore.Log{BillingDetails: &billingDetails})
	if err != nil {
		t.Fatalf("marshal visible billing_details: %v", err)
	}
	if !strings.Contains(string(encoded), `"billing_details":`) {
		t.Fatalf("visible wire payload %s, want billing_details projection", encoded)
	}

	encoded, err = jsonx.Marshal(logstore.Log{})
	if err != nil {
		t.Fatalf("marshal hidden billing_details: %v", err)
	}
	if strings.Contains(string(encoded), `"billing_details":`) {
		t.Fatalf("hidden wire payload %s, want omitted billing_details", encoded)
	}
}
