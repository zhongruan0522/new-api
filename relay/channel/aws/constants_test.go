package aws

import "testing"

func TestClaudeOpus4ModelMapping(t *testing.T) {
	modelID := getAwsModelID("claude-opus-4-20250514")
	if modelID != "anthropic.claude-opus-4-20250514-v1:0" {
		t.Fatalf("getAwsModelID = %q, want anthropic.claude-opus-4-20250514-v1:0", modelID)
	}

	for _, regionPrefix := range []string{"us"} {
		if !awsModelCanCrossRegion(modelID, regionPrefix) {
			t.Fatalf("awsModelCanCrossRegion(%q, %q) = false, want true", modelID, regionPrefix)
		}
	}
}
