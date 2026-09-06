package billing

import (
	"fmt"
	"math"

	"github.com/NookMux/NookMux/pkg/jsonx"
)

// billing_details JSON（PRD 第 4 章）：
//   - 顶层只允许 schema_version 与 tokens；tokens 下只允许 input/output/cache 三组；
//   - 扩展字段必须先升级 schema 版本，未知字段/未知版本在读取端显式报错；
//   - 上游未返回的可选拆分写 null 或省略，不能用 0 伪装"官方返回了零"；
//   - 全部 token 值必须是非负整数；
//   - write_cache_5m + write_cache_1h <= write_cache，差值为未分档写入；
//   - 序列化为 canonical JSON：结构体字段顺序固定、snake_case、无调试注释，
//     不写 quota/单价/倍率/分组/供应商/service tier/错误信息/请求元数据。
//
// 核心总量（普通输入、输出总量、raw 输入总量）不进该 JSON，继续由
// Log.PromptTokens / Log.CompletionTokens 兼容列承载（PRD 1.2.4、4.1.3）。

type BillingDetailsPayload struct {
	SchemaVersion int                 `json:"schema_version"`
	Tokens        BillingTokensDetail `json:"tokens"`
}

type BillingTokensDetail struct {
	Input  BillingInputTokens  `json:"input"`
	Output BillingOutputTokens `json:"output"`
	Cache  BillingCacheTokens  `json:"cache"`
}

type BillingInputTokens struct {
	TextInput     *int `json:"text_input,omitempty"`
	ImageInput    *int `json:"image_input,omitempty"`
	AudioInput    *int `json:"audio_input,omitempty"`
	VideoInput    *int `json:"video_input,omitempty"`
	DocumentInput *int `json:"document_input,omitempty"`
}

type BillingOutputTokens struct {
	TextOutput         *int `json:"text_output,omitempty"`
	AudioOutput        *int `json:"audio_output,omitempty"`
	ImageOutput        *int `json:"image_output,omitempty"`
	ReasoningOutput    *int `json:"reasoning_output,omitempty"`
	AcceptedPrediction *int `json:"accepted_prediction,omitempty"`
	RejectedPrediction *int `json:"rejected_prediction,omitempty"`
}

type BillingCacheTokens struct {
	ReadCache    *int `json:"read_cache,omitempty"`
	WriteCache   *int `json:"write_cache,omitempty"`
	WriteCache5m *int `json:"write_cache_5m,omitempty"`
	WriteCache1h *int `json:"write_cache_1h,omitempty"`
}

// SerializeBillingUsage 把 BillingUsage 序列化为 schema v1 canonical JSON。
// 入参必须先经 finalizeBillingUsage 校验（负数已在构建期显式失败）；
// 只有官方明确返回（含 PRD 缓存写入转换规则产出的 5m 分档）的拆分才写入，
// 官方显式 0 会被保留；三个分组对象始终存在。
func SerializeBillingUsage(bu *BillingUsage) (string, error) {
	if bu == nil {
		return "", fmt.Errorf("billing usage is nil")
	}
	payload := BillingDetailsPayload{
		SchemaVersion: BillingDetailsSchemaVersion,
		Tokens: BillingTokensDetail{
			Input: BillingInputTokens{
				TextInput:     positiveInt(bu.TextInputTokens),
				ImageInput:    positiveInt(bu.ImageInputTokens),
				AudioInput:    positiveInt(bu.AudioInputTokens),
				VideoInput:    positiveInt(bu.VideoInputTokens),
				DocumentInput: positiveInt(bu.DocumentInputTokens),
			},
			Output: BillingOutputTokens{
				TextOutput:         positiveInt(bu.TextOutputTokens),
				AudioOutput:        positiveInt(bu.AudioOutputTokens),
				ImageOutput:        positiveInt(bu.ImageOutputTokens),
				ReasoningOutput:    positiveInt(bu.ReasoningTokens),
				AcceptedPrediction: positiveInt(bu.AcceptedPredictionTokens),
				RejectedPrediction: positiveInt(bu.RejectedPredictionTokens),
			},
			Cache: BillingCacheTokens{
				ReadCache:    optionalCacheInt(bu.CacheReadTokens, bu.CacheReadPresent),
				WriteCache:   optionalCacheInt(bu.CacheWriteTokens, bu.CacheWritePresent),
				WriteCache5m: positiveInt(bu.CacheWrite5mTokens),
				WriteCache1h: positiveInt(bu.CacheWrite1hTokens),
			},
		},
	}
	encoded, err := jsonx.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func positiveInt(value *int) *int {
	if value == nil {
		return nil
	}
	return value
}

func optionalCacheInt(value int, present bool) *int {
	if !present && value <= 0 {
		return nil
	}
	return &value
}

// ParseBillingDetailsJSON 是读取端唯一入口：历史日志（NULL/空串）不调用本函数，
// 由调用方先按"billing_details 是否存在"判断新旧格式（PRD 4.3.5）。
// 损坏 JSON、未知版本、未知字段、负数、非整数与分档大于总量都显式报错，
// 不静默裁剪、不猜测相似字段。
func ParseBillingDetailsJSON(raw string) (*BillingDetailsPayload, error) {
	if raw == "" {
		return nil, fmt.Errorf("billing_details is empty")
	}
	var payload BillingDetailsPayload
	if err := jsonx.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("billing_details is not valid JSON: %w", err)
	}
	if payload.SchemaVersion != BillingDetailsSchemaVersion {
		return nil, fmt.Errorf("unknown billing_details schema version: %d", payload.SchemaVersion)
	}
	if err := validateBillingDetailsKeys(raw); err != nil {
		return nil, err
	}
	if err := validateBillingDetailsPayload(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// validateBillingDetailsKeys 拒绝未知字段：schema v1 的字段集合是封闭的。
func validateBillingDetailsKeys(raw string) error {
	var probe map[string]any
	if err := jsonx.Unmarshal([]byte(raw), &probe); err != nil {
		return fmt.Errorf("billing_details is not valid JSON: %w", err)
	}
	if err := rejectUnknownKeys(probe, "root", map[string]bool{"schema_version": true, "tokens": true}); err != nil {
		return err
	}
	tokens, ok := probe["tokens"].(map[string]any)
	if !ok {
		return fmt.Errorf("billing_details.tokens must be an object")
	}
	if err := rejectUnknownKeys(tokens, "tokens", map[string]bool{"input": true, "output": true, "cache": true}); err != nil {
		return err
	}
	groupFields := map[string]map[string]bool{
		"input": {
			"text_input": true, "image_input": true, "audio_input": true,
			"video_input": true, "document_input": true,
		},
		"output": {
			"text_output": true, "audio_output": true, "image_output": true,
			"reasoning_output": true, "accepted_prediction": true, "rejected_prediction": true,
		},
		"cache": {
			"read_cache": true, "write_cache": true,
			"write_cache_5m": true, "write_cache_1h": true,
		},
	}
	for group, fields := range groupFields {
		value, present := tokens[group]
		if !present {
			return fmt.Errorf("billing_details.tokens.%s is required", group)
		}
		groupMap, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("billing_details.tokens.%s must be an object", group)
		}
		if err := rejectUnknownKeys(groupMap, "tokens."+group, fields); err != nil {
			return err
		}
	}
	return nil
}

func rejectUnknownKeys(values map[string]any, path string, allowed map[string]bool) error {
	for key := range values {
		if !allowed[key] {
			return fmt.Errorf("unknown billing_details field: %s.%s", path, key)
		}
	}
	return nil
}

func validateBillingDetailsPayload(payload *BillingDetailsPayload) error {
	nonNegative := map[string]*int{
		"input.text_input":           payload.Tokens.Input.TextInput,
		"input.image_input":          payload.Tokens.Input.ImageInput,
		"input.audio_input":          payload.Tokens.Input.AudioInput,
		"input.video_input":          payload.Tokens.Input.VideoInput,
		"input.document_input":       payload.Tokens.Input.DocumentInput,
		"output.text_output":         payload.Tokens.Output.TextOutput,
		"output.audio_output":        payload.Tokens.Output.AudioOutput,
		"output.image_output":        payload.Tokens.Output.ImageOutput,
		"output.reasoning_output":    payload.Tokens.Output.ReasoningOutput,
		"output.accepted_prediction": payload.Tokens.Output.AcceptedPrediction,
		"output.rejected_prediction": payload.Tokens.Output.RejectedPrediction,
		"cache.read_cache":           payload.Tokens.Cache.ReadCache,
		"cache.write_cache":          payload.Tokens.Cache.WriteCache,
		"cache.write_cache_5m":       payload.Tokens.Cache.WriteCache5m,
		"cache.write_cache_1h":       payload.Tokens.Cache.WriteCache1h,
	}
	for name, value := range nonNegative {
		if value == nil {
			continue
		}
		if *value < 0 {
			return fmt.Errorf("negative token count %s=%d", name, *value)
		}
	}
	writeCache := intValue(payload.Tokens.Cache.WriteCache)
	var tiered int
	if payload.Tokens.Cache.WriteCache5m != nil {
		tiered = *payload.Tokens.Cache.WriteCache5m
	}
	if payload.Tokens.Cache.WriteCache1h != nil {
		if *payload.Tokens.Cache.WriteCache1h > 0 && tiered > math.MaxInt-*payload.Tokens.Cache.WriteCache1h {
			return fmt.Errorf("cache write tiers overflow")
		}
		tiered += *payload.Tokens.Cache.WriteCache1h
	}
	if tiered < 0 {
		return fmt.Errorf("cache write tiers (%d) exceed write_cache total (%d)", tiered, writeCache)
	}
	if tiered > writeCache {
		return fmt.Errorf("cache write tiers (%d) exceed write_cache total (%d)", tiered, writeCache)
	}
	return nil
}
