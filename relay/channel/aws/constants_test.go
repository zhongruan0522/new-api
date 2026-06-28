package aws

import "testing"

func TestClaudeOpus47ModelMapping(t *testing.T) {
	modelID := getAwsModelID("claude-opus-4-7")
	if modelID != "anthropic.claude-opus-4-7" {
		t.Fatalf("getAwsModelID = %q, want anthropic.claude-opus-4-7", modelID)
	}

	for _, regionPrefix := range []string{"us", "ap", "eu"} {
		if !awsModelCanCrossRegion(modelID, regionPrefix) {
			t.Fatalf("awsModelCanCrossRegion(%q, %q) = false, want true", modelID, regionPrefix)
		}
	}
}
