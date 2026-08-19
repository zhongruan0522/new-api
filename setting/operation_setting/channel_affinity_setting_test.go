package operation_setting

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/pkg/jsonx"
)

func TestChannelAffinityRuleSerializesSkipRetryFalse(t *testing.T) {
	rule := ChannelAffinityRule{
		Name:       "user editable rule",
		ModelRegex: []string{"^claude-.*$"},
		KeySources: []ChannelAffinityKeySource{
			{Type: "gjson", Path: "metadata.user_id"},
		},
		SkipRetryOnFailure: false,
		IncludeUsingGroup:  true,
		IncludeRuleName:    true,
	}

	data, err := jsonx.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal channel affinity rule: %v", err)
	}

	if !strings.Contains(string(data), `"skip_retry_on_failure":false`) {
		t.Fatalf("serialized rule should keep explicit false skip_retry_on_failure, got %s", data)
	}
}
