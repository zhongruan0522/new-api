package shared

import (
	"strings"
	"testing"

	"github.com/NookMux/NookMux/pkg/jsonx"
)

func TestOpenAIResponsesResponseInstructionsPreservesObject(t *testing.T) {
	raw := []byte(`{"id":"resp_123","instructions":{"role":"system","content":"be concise"}}`)

	var resp OpenAIResponsesResponse
	if err := jsonx.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if string(resp.Instructions) != `{"role":"system","content":"be concise"}` {
		t.Fatalf("instructions raw = %s, want original object", resp.Instructions)
	}

	out, err := jsonx.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response error = %v", err)
	}
	if !strings.Contains(string(out), `"instructions":{"role":"system","content":"be concise"}`) {
		t.Fatalf("marshalled response lost object instructions shape: %s", out)
	}
}

func TestOpenAIResponsesResponseInstructionsPreservesNull(t *testing.T) {
	raw := []byte(`{"id":"resp_123","instructions":null}`)

	var resp OpenAIResponsesResponse
	if err := jsonx.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if string(resp.Instructions) != `null` {
		t.Fatalf("instructions raw = %s, want null", resp.Instructions)
	}

	out, err := jsonx.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response error = %v", err)
	}
	if !strings.Contains(string(out), `"instructions":null`) {
		t.Fatalf("marshalled response lost null instructions shape: %s", out)
	}
}

func TestOpenAIResponsesResponseInstructionsStillAcceptsString(t *testing.T) {
	raw := []byte(`{"id":"resp_123","instructions":"follow the policy"}`)

	var resp OpenAIResponsesResponse
	if err := jsonx.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if string(resp.Instructions) != `"follow the policy"` {
		t.Fatalf("instructions raw = %s, want string", resp.Instructions)
	}

	out, err := jsonx.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response error = %v", err)
	}
	if !strings.Contains(string(out), `"instructions":"follow the policy"`) {
		t.Fatalf("marshalled response lost string instructions shape: %s", out)
	}
}

// TestUsageDetailsParseCacheWriteAndPredictionTokens 验证计费 PRD 3.1 的
// 官方来源解析：prompt_tokens_details.cache_write_tokens、
// input_tokens_details.cache_write_tokens 与 completion_tokens_details 的
// accepted/rejected prediction。CachedCreationTokens 保持 json:"-"，
// 序列化行为不变。
func TestUsageDetailsParseCacheWriteAndPredictionTokens(t *testing.T) {
	raw := []byte(`{
		"prompt_tokens": 200,
		"completion_tokens": 100,
		"prompt_tokens_details": {"cached_tokens": 30, "cache_write_tokens": 20},
		"completion_tokens_details": {
			"reasoning_tokens": 10,
			"accepted_prediction_tokens": 12,
			"rejected_prediction_tokens": 3
		}
	}`)
	var usage Usage
	if err := jsonx.Unmarshal(raw, &usage); err != nil {
		t.Fatalf("unmarshal usage: %v", err)
	}
	if usage.PromptTokensDetails.CachedCreationTokens != 20 {
		t.Fatalf("cache_write_tokens = %d, want 20", usage.PromptTokensDetails.CachedCreationTokens)
	}
	if usage.CompletionTokenDetails.AcceptedPredictionTokens != 12 || usage.CompletionTokenDetails.RejectedPredictionTokens != 3 {
		t.Fatalf("prediction tokens = %+v", usage.CompletionTokenDetails)
	}

	responsesRaw := []byte(`{
		"input_tokens": 200,
		"output_tokens": 100,
		"input_tokens_details": {"cached_tokens": 30, "cache_write_tokens": 20},
		"output_tokens_details": {"reasoning_tokens": 10}
	}`)
	var responsesUsage Usage
	if err := jsonx.Unmarshal(responsesRaw, &responsesUsage); err != nil {
		t.Fatalf("unmarshal responses usage: %v", err)
	}
	if responsesUsage.InputTokensDetails == nil || responsesUsage.InputTokensDetails.CachedCreationTokens != 20 {
		t.Fatalf("responses cache_write_tokens = %+v", responsesUsage.InputTokensDetails)
	}

	// 序列化不回写 cache_write_tokens（json:"-"）；prediction 为零时省略，
	// 客户端可见序列化行为不受新增字段影响。
	encoded, err := jsonx.Marshal(&usage)
	if err != nil {
		t.Fatalf("marshal usage: %v", err)
	}
	if strings.Contains(string(encoded), "cache_write_tokens") {
		t.Fatalf("cache_write_tokens must not leak into serialized usage: %s", encoded)
	}
	var zeroPredictionUsage Usage
	if err := jsonx.Unmarshal([]byte(`{"prompt_tokens":1}`), &zeroPredictionUsage); err != nil {
		t.Fatalf("unmarshal zero usage: %v", err)
	}
	zeroEncoded, err := jsonx.Marshal(&zeroPredictionUsage)
	if err != nil {
		t.Fatalf("marshal zero usage: %v", err)
	}
	if strings.Contains(string(zeroEncoded), "accepted_prediction_tokens") {
		t.Fatalf("zero-value predictions must be omitted: %s", zeroEncoded)
	}

	// null 明细保持零值，不报错。
	var nullUsage Usage
	if err := jsonx.Unmarshal([]byte(`{"prompt_tokens_details": null}`), &nullUsage); err != nil {
		t.Fatalf("unmarshal null details: %v", err)
	}
	if nullUsage.PromptTokensDetails != (InputTokenDetails{}) {
		t.Fatalf("null details should stay zero value, got %+v", nullUsage.PromptTokensDetails)
	}
}
