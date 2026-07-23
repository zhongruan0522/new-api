package openai

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/constant"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	relayconstant "github.com/NookMux/NookMux/relay/constant"
)

func azureInfo(baseURL, apiVersion string, relayMode int) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RequestURLPath: "/v1/responses/compact",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeAzure,
			ChannelBaseUrl: baseURL,
			ApiVersion:     apiVersion,
		},
		RelayMode: relayMode,
	}
}

// TestAzureResponsesCompactRequestURL reproduces the user-facing #149 bug: an
// Azure channel POSTing /v1/responses/compact used to fall through to the
// deployment-style URL. It must instead use the Responses path with /compact.
func TestAzureResponsesCompactRequestURL(t *testing.T) {
	info := azureInfo("https://example.openai.azure.com", "", relayconstant.RelayModeResponsesCompact)

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.Contains(got, "/openai/v1/responses/compact?api-version=preview") {
		t.Fatalf("azure compact URL = %q, want /openai/v1/responses/compact?api-version=preview", got)
	}
	if strings.Contains(got, "/openai/deployments/") {
		t.Fatalf("azure compact URL must not use deployment path: %s", got)
	}
}

// TestAzureResponsesCompactCognitiveServicesRequestURL checks the cognitiveservices
// variant, which uses the channel's api-version and the deployment-less path.
func TestAzureResponsesCompactCognitiveServicesRequestURL(t *testing.T) {
	info := azureInfo("https://example.cognitiveservices.azure.com", "2025-04-01-preview", relayconstant.RelayModeResponsesCompact)

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.Contains(got, "/openai/responses/compact?api-version=2025-04-01-preview") {
		t.Fatalf("azure cognitive compact URL = %q, want /openai/responses/compact?api-version=2025-04-01-preview", got)
	}
}

// TestAzureResponsesCompactCustomResponsesVersion checks that a channel-level
// override of the responses API version also applies to compact requests.
func TestAzureResponsesCompactCustomResponsesVersion(t *testing.T) {
	info := azureInfo("https://example.openai.azure.com", "", relayconstant.RelayModeResponsesCompact)
	info.ChannelOtherSettings.AzureResponsesVersion = "2025-03-01-preview"

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.Contains(got, "/openai/v1/responses/compact?api-version=2025-03-01-preview") {
		t.Fatalf("azure compact URL = %q, want custom responses version", got)
	}
}

// TestAzureResponsesNonCompactUnchanged guards against regressing the existing
// non-compact responses URL while extending the branch to cover compact.
func TestAzureResponsesNonCompactUnchanged(t *testing.T) {
	info := azureInfo("https://example.openai.azure.com", "", relayconstant.RelayModeResponses)

	got, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL error = %v", err)
	}
	if !strings.Contains(got, "/openai/v1/responses?api-version=preview") {
		t.Fatalf("azure responses URL = %q, want /openai/v1/responses?api-version=preview", got)
	}
	if strings.Contains(got, "/compact") {
		t.Fatalf("non-compact responses URL must not contain /compact: %s", got)
	}
}
