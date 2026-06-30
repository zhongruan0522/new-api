package dto

import (
	"strings"
	"testing"

	"github.com/zhongruan0522/new-api/common"
)

func TestOpenAIResponsesResponseInstructionsPreservesObject(t *testing.T) {
	raw := []byte(`{"id":"resp_123","instructions":{"role":"system","content":"be concise"}}`)

	var resp OpenAIResponsesResponse
	if err := common.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if string(resp.Instructions) != `{"role":"system","content":"be concise"}` {
		t.Fatalf("instructions raw = %s, want original object", resp.Instructions)
	}

	out, err := common.Marshal(resp)
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
	if err := common.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if string(resp.Instructions) != `null` {
		t.Fatalf("instructions raw = %s, want null", resp.Instructions)
	}

	out, err := common.Marshal(resp)
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
	if err := common.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if string(resp.Instructions) != `"follow the policy"` {
		t.Fatalf("instructions raw = %s, want string", resp.Instructions)
	}

	out, err := common.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response error = %v", err)
	}
	if !strings.Contains(string(out), `"instructions":"follow the policy"`) {
		t.Fatalf("marshalled response lost string instructions shape: %s", out)
	}
}
