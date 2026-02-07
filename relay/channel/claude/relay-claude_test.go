package claude

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestFormatClaudeResponseInfo_MessageStart(t *testing.T) {
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{},
	}
	claudeResponse := &dto.ClaudeResponse{
		Type: "message_start",
		Message: &dto.ClaudeMediaMessage{
			Id:    "msg_123",
			Model: "claude-3-5-sonnet",
			Usage: &dto.ClaudeUsage{
				InputTokens:              100,
				OutputTokens:             1,
				CacheCreationInputTokens: 50,
				CacheReadInputTokens:     30,
			},
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	if claudeInfo.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.PromptTokensDetails.CachedTokens != 30 {
		t.Errorf("CachedTokens = %d, want 30", claudeInfo.Usage.PromptTokensDetails.CachedTokens)
	}
	if claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens != 50 {
		t.Errorf("CachedCreationTokens = %d, want 50", claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens)
	}
	if claudeInfo.ResponseId != "msg_123" {
		t.Errorf("ResponseId = %s, want msg_123", claudeInfo.ResponseId)
	}
	if claudeInfo.Model != "claude-3-5-sonnet" {
		t.Errorf("Model = %s, want claude-3-5-sonnet", claudeInfo.Model)
	}
}

func TestFormatClaudeResponseInfo_MessageDelta_FullUsage(t *testing.T) {
	// message_start 先积累 usage
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{
			PromptTokens: 100,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         30,
				CachedCreationTokens: 50,
			},
			CompletionTokens: 1,
		},
	}

	// message_delta 带完整 usage（原生 Anthropic 场景）
	claudeResponse := &dto.ClaudeResponse{
		Type: "message_delta",
		Usage: &dto.ClaudeUsage{
			InputTokens:              100,
			OutputTokens:             200,
			CacheCreationInputTokens: 50,
			CacheReadInputTokens:     30,
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	if claudeInfo.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", claudeInfo.Usage.CompletionTokens)
	}
	if claudeInfo.Usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", claudeInfo.Usage.TotalTokens)
	}
	if !claudeInfo.Done {
		t.Error("expected Done = true")
	}
}

func TestFormatClaudeResponseInfo_MessageDelta_OnlyOutputTokens(t *testing.T) {
	// 模拟 Bedrock: message_start 已积累 usage
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{
			PromptTokens: 100,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:         30,
				CachedCreationTokens: 50,
			},
			CompletionTokens:            1,
			ClaudeCacheCreation5mTokens: 10,
			ClaudeCacheCreation1hTokens: 20,
		},
	}

	// Bedrock 的 message_delta 只有 output_tokens，缺少 input_tokens 和 cache 字段
	claudeResponse := &dto.ClaudeResponse{
		Type: "message_delta",
		Usage: &dto.ClaudeUsage{
			OutputTokens: 200,
			// InputTokens, CacheCreationInputTokens, CacheReadInputTokens 都是 0
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	// PromptTokens 应保持 message_start 的值（因为 message_delta 的 InputTokens=0，不更新）
	if claudeInfo.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", claudeInfo.Usage.CompletionTokens)
	}
	if claudeInfo.Usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", claudeInfo.Usage.TotalTokens)
	}
	// cache 字段应保持 message_start 的值
	if claudeInfo.Usage.PromptTokensDetails.CachedTokens != 30 {
		t.Errorf("CachedTokens = %d, want 30", claudeInfo.Usage.PromptTokensDetails.CachedTokens)
	}
	if claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens != 50 {
		t.Errorf("CachedCreationTokens = %d, want 50", claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens)
	}
	if claudeInfo.Usage.ClaudeCacheCreation5mTokens != 10 {
		t.Errorf("ClaudeCacheCreation5mTokens = %d, want 10", claudeInfo.Usage.ClaudeCacheCreation5mTokens)
	}
	if claudeInfo.Usage.ClaudeCacheCreation1hTokens != 20 {
		t.Errorf("ClaudeCacheCreation1hTokens = %d, want 20", claudeInfo.Usage.ClaudeCacheCreation1hTokens)
	}
	if !claudeInfo.Done {
		t.Error("expected Done = true")
	}
}

func TestFormatClaudeResponseInfo_NilClaudeInfo(t *testing.T) {
	claudeResponse := &dto.ClaudeResponse{Type: "message_start"}
	ok := FormatClaudeResponseInfo(claudeResponse, nil, nil)
	if ok {
		t.Error("expected false for nil claudeInfo")
	}
}

func TestFormatClaudeResponseInfo_ContentBlockDelta(t *testing.T) {
	text := "hello"
	claudeInfo := &ClaudeResponseInfo{
		Usage:        &dto.Usage{},
		ResponseText: strings.Builder{},
	}
	claudeResponse := &dto.ClaudeResponse{
		Type: "content_block_delta",
		Delta: &dto.ClaudeMediaMessage{
			Text: &text,
		},
	}

	ok := FormatClaudeResponseInfo(claudeResponse, nil, claudeInfo)
	if !ok {
		t.Fatal("expected true")
	}
	if claudeInfo.ResponseText.String() != "hello" {
		t.Errorf("ResponseText = %q, want %q", claudeInfo.ResponseText.String(), "hello")
	}
}

// TestEnrichMessageDeltaUsage 测试 message_delta 事件的 usage 补全逻辑
// 这是修复 issue #2881 的核心逻辑：当上游（如 Bedrock）的 message_delta 缺少
// input_tokens 和 cache 相关字段时，用 claudeInfo 中积累的数据补全
func TestEnrichMessageDeltaUsage(t *testing.T) {
	tests := []struct {
		name          string
		claudeInfo    *ClaudeResponseInfo
		deltaUsage    *dto.ClaudeUsage
		wantInput     int
		wantCacheRead int
		wantCacheCreate int
		wantOutput    int
		want5m        int
		want1h        int
	}{
		{
			name: "Bedrock: delta 只有 output_tokens，从 claudeInfo 补全其他字段",
			claudeInfo: &ClaudeResponseInfo{
				Usage: &dto.Usage{
					PromptTokens: 100,
					PromptTokensDetails: dto.InputTokenDetails{
						CachedTokens:         30,
						CachedCreationTokens: 50,
					},
					ClaudeCacheCreation5mTokens: 10,
					ClaudeCacheCreation1hTokens: 20,
				},
			},
			deltaUsage:      &dto.ClaudeUsage{OutputTokens: 200},
			wantInput:       100,
			wantCacheRead:   30,
			wantCacheCreate: 50,
			wantOutput:      200,
			want5m:          10,
			want1h:          20,
		},
		{
			name: "原生 Anthropic: delta 已包含所有字段，不覆盖",
			claudeInfo: &ClaudeResponseInfo{
				Usage: &dto.Usage{
					PromptTokens: 100,
					PromptTokensDetails: dto.InputTokenDetails{
						CachedTokens:         30,
						CachedCreationTokens: 50,
					},
				},
			},
			deltaUsage: &dto.ClaudeUsage{
				InputTokens:              100,
				OutputTokens:             200,
				CacheReadInputTokens:     30,
				CacheCreationInputTokens: 50,
			},
			wantInput:       100,
			wantCacheRead:   30,
			wantCacheCreate: 50,
			wantOutput:      200,
		},
		{
			name: "delta usage 为 nil，创建并补全",
			claudeInfo: &ClaudeResponseInfo{
				Usage: &dto.Usage{
					PromptTokens: 80,
					PromptTokensDetails: dto.InputTokenDetails{
						CachedTokens:         20,
						CachedCreationTokens: 40,
					},
				},
			},
			deltaUsage:      nil,
			wantInput:       80,
			wantCacheRead:   20,
			wantCacheCreate: 40,
			wantOutput:      0,
		},
		{
			name: "没有 cache 数据，不补全",
			claudeInfo: &ClaudeResponseInfo{
				Usage: &dto.Usage{
					PromptTokens: 100,
				},
			},
			deltaUsage:      &dto.ClaudeUsage{OutputTokens: 50},
			wantInput:       100,
			wantCacheRead:   0,
			wantCacheCreate: 0,
			wantOutput:      50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claudeResponse := &dto.ClaudeResponse{
				Type:  "message_delta",
				Usage: tt.deltaUsage,
			}

			// 模拟 HandleStreamResponseData 中 Claude 格式的补全逻辑
			enrichMessageDeltaUsage(claudeResponse, tt.claudeInfo)

			if claudeResponse.Usage == nil {
				t.Fatal("Usage should not be nil after enrichment")
			}
			if claudeResponse.Usage.InputTokens != tt.wantInput {
				t.Errorf("InputTokens = %d, want %d", claudeResponse.Usage.InputTokens, tt.wantInput)
			}
			if claudeResponse.Usage.CacheReadInputTokens != tt.wantCacheRead {
				t.Errorf("CacheReadInputTokens = %d, want %d", claudeResponse.Usage.CacheReadInputTokens, tt.wantCacheRead)
			}
			if claudeResponse.Usage.CacheCreationInputTokens != tt.wantCacheCreate {
				t.Errorf("CacheCreationInputTokens = %d, want %d", claudeResponse.Usage.CacheCreationInputTokens, tt.wantCacheCreate)
			}
			if claudeResponse.Usage.OutputTokens != tt.wantOutput {
				t.Errorf("OutputTokens = %d, want %d", claudeResponse.Usage.OutputTokens, tt.wantOutput)
			}
			if tt.want5m > 0 || tt.want1h > 0 {
				if claudeResponse.Usage.CacheCreation == nil {
					t.Fatal("CacheCreation should not be nil")
				}
				if claudeResponse.Usage.CacheCreation.Ephemeral5mInputTokens != tt.want5m {
					t.Errorf("Ephemeral5mInputTokens = %d, want %d", claudeResponse.Usage.CacheCreation.Ephemeral5mInputTokens, tt.want5m)
				}
				if claudeResponse.Usage.CacheCreation.Ephemeral1hInputTokens != tt.want1h {
					t.Errorf("Ephemeral1hInputTokens = %d, want %d", claudeResponse.Usage.CacheCreation.Ephemeral1hInputTokens, tt.want1h)
				}
			}
		})
	}
}
