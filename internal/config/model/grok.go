package model

import "github.com/NookMux/NookMux/internal/config/manager"

// GrokSettings defines Grok model configuration.
type GrokSettings struct {
	ViolationDeductionEnabled bool    `json:"violation_deduction_enabled"`
	ViolationDeductionAmount  float64 `json:"violation_deduction_amount"`
}

var defaultGrokSettings = GrokSettings{
	ViolationDeductionEnabled: true,
	ViolationDeductionAmount:  0.05,
}

var grokSettings = defaultGrokSettings

func init() {
	manager.GlobalConfig.Register("grok", &grokSettings)
}

func GetGrokSettings() *GrokSettings {
	return &grokSettings
}
