package reasoning

import "testing"

func TestParseOpenAIReasoningEffortFromModelSuffix(t *testing.T) {
	effort, baseModel := ParseOpenAIReasoningEffortFromModelSuffix("gpt-5-mini-low")
	if effort != "low" || baseModel != "gpt-5-mini" {
		t.Fatalf("ParseOpenAIReasoningEffortFromModelSuffix = (%q, %q), want (low, gpt-5-mini)", effort, baseModel)
	}

	effort, baseModel = ParseOpenAIReasoningEffortFromModelSuffix("gpt-5-mini")
	if effort != "" || baseModel != "gpt-5-mini" {
		t.Fatalf("ParseOpenAIReasoningEffortFromModelSuffix without suffix = (%q, %q)", effort, baseModel)
	}
}

func TestParseDeepSeekV4ThinkingSuffix(t *testing.T) {
	tests := []struct {
		model        string
		wantBase     string
		wantThinking string
		wantEffort   string
		wantOK       bool
	}{
		{
			model:        "deepseek-v4-flash-none",
			wantBase:     "deepseek-v4-flash",
			wantThinking: "disabled",
			wantOK:       true,
		},
		{
			model:        "deepseek-v4-pro-max",
			wantBase:     "deepseek-v4-pro",
			wantThinking: "enabled",
			wantEffort:   "max",
			wantOK:       true,
		},
		{
			model:    "deepseek-chat-max",
			wantBase: "deepseek-chat-max",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			baseModel, thinkingType, effort, ok := ParseDeepSeekV4ThinkingSuffix(tt.model)
			if baseModel != tt.wantBase || thinkingType != tt.wantThinking || effort != tt.wantEffort || ok != tt.wantOK {
				t.Fatalf("ParseDeepSeekV4ThinkingSuffix = (%q, %q, %q, %v), want (%q, %q, %q, %v)", baseModel, thinkingType, effort, ok, tt.wantBase, tt.wantThinking, tt.wantEffort, tt.wantOK)
			}
		})
	}
}
