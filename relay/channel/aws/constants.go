package aws

import "strings"

var awsModelIDMap = map[string]string{
	"claude-3-haiku-20240307":  "anthropic.claude-3-haiku-20240307-v1:0",
	"claude-opus-4-20250514":   "anthropic.claude-opus-4-20250514-v1:0",
	"claude-sonnet-4-20250514": "anthropic.claude-sonnet-4-20250514-v1:0",
}

var awsModelCanCrossRegionMap = map[string]map[string]bool{
	"anthropic.claude-3-sonnet-20240229-v1:0": {
		"us": true,
		"eu": true,
		"ap": true,
	},
	"anthropic.claude-3-opus-20240229-v1:0": {
		"us": true,
	},
	"anthropic.claude-3-haiku-20240307-v1:0": {
		"us": true,
		"eu": true,
		"ap": true,
	},
	"anthropic.claude-3-5-sonnet-20240620-v1:0": {
		"us": true,
		"eu": true,
		"ap": true,
	},
	"anthropic.claude-3-5-sonnet-20241022-v2:0": {
		"us": true,
		"ap": true,
	},
	"anthropic.claude-3-5-haiku-20241022-v1:0": {
		"us": true,
	},
	"anthropic.claude-3-7-sonnet-20250219-v1:0": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	"anthropic.claude-sonnet-4-20250514-v1:0": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	"anthropic.claude-opus-4-20250514-v1:0": {
		"us": true,
	},
	"anthropic.claude-opus-4-1-20250805-v1:0": {
		"us": true,
	},
	"anthropic.claude-sonnet-4-5-20250929-v1:0": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	"anthropic.claude-opus-4-5-20251101-v1:0": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	"anthropic.claude-opus-4-6-v1": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	"anthropic.claude-opus-4-7": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	"anthropic.claude-haiku-4-5-20251001-v1:0": {
		"us": true,
		"ap": true,
		"eu": true,
	},
	// Nova models - all support three major regions
	"amazon.nova-micro-v1:0": {
		"us":   true,
		"eu":   true,
		"apac": true,
	},
	"amazon.nova-lite-v1:0": {
		"us":   true,
		"eu":   true,
		"apac": true,
	},
	"amazon.nova-pro-v1:0": {
		"us":   true,
		"eu":   true,
		"apac": true,
	},
	"amazon.nova-premier-v1:0": {
		"us": true,
	},
	"amazon.nova-canvas-v1:0": {
		"us":   true,
		"eu":   true,
		"apac": true,
	},
	"amazon.nova-reel-v1:0": {
		"us":   true,
		"eu":   true,
		"apac": true,
	},
	"amazon.nova-reel-v1:1": {
		"us": true,
	},
	"amazon.nova-sonic-v1:0": {
		"us":   true,
		"eu":   true,
		"apac": true,
	},
}

var awsRegionCrossModelPrefixMap = map[string]string{
	"us": "us",
	"eu": "eu",
	"ap": "apac",
}

var ChannelName = "aws"

// 判断是否为Nova模型
func isNovaModel(modelId string) bool {
	return strings.Contains(modelId, "nova-")
}
