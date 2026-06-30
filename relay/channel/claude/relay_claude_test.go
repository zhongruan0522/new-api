package claude

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/types"
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
	if claudeInfo.Usage.PromptTokens != 180 {
		t.Errorf("PromptTokens = %d, want 180", claudeInfo.Usage.PromptTokens)
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
			PromptTokens: 180,
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
	if claudeInfo.Usage.PromptTokens != 180 {
		t.Errorf("PromptTokens = %d, want 180", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", claudeInfo.Usage.CompletionTokens)
	}
	if claudeInfo.Usage.TotalTokens != 380 {
		t.Errorf("TotalTokens = %d, want 380", claudeInfo.Usage.TotalTokens)
	}
	if !claudeInfo.Done {
		t.Error("expected Done = true")
	}
}

func TestFormatClaudeResponseInfo_MessageDelta_OnlyOutputTokens(t *testing.T) {
	// 模拟 Bedrock: message_start 已积累 usage
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{
			PromptTokens: 180,
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
	if claudeInfo.Usage.PromptTokens != 180 {
		t.Errorf("PromptTokens = %d, want 180", claudeInfo.Usage.PromptTokens)
	}
	if claudeInfo.Usage.CompletionTokens != 200 {
		t.Errorf("CompletionTokens = %d, want 200", claudeInfo.Usage.CompletionTokens)
	}
	if claudeInfo.Usage.TotalTokens != 380 {
		t.Errorf("TotalTokens = %d, want 380", claudeInfo.Usage.TotalTokens)
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

func TestClaudeOpenAIStreamMasksResponseModel(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName:    "alias-model",
		ResponseModelName:  "alias-model",
		RelayFormat:        types.RelayFormatOpenAI,
		ShouldIncludeUsage: true,
		ClaudeConvertInfo:  &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone},
		ChannelMeta:        &relaycommon.ChannelMeta{UpstreamModelName: "real-model"},
	}
	claudeInfo := &ClaudeResponseInfo{
		ResponseId: "msg_1",
		Created:    1710000000,
		Model:      "real-model",
		Usage:      &dto.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
		Done:       true,
	}
	messageStart, err := common.Marshal(dto.ClaudeResponse{
		Type: "message_start",
		Message: &dto.ClaudeMediaMessage{
			Id:    "msg_1",
			Model: "real-model",
			Usage: &dto.ClaudeUsage{InputTokens: 1},
		},
	})
	if err != nil {
		t.Fatalf("marshal message_start: %v", err)
	}
	if apiErr := HandleStreamResponseData(c, info, claudeInfo, string(messageStart)); apiErr != nil {
		t.Fatalf("HandleStreamResponseData error: %v", apiErr)
	}
	HandleStreamFinalResponse(c, info, claudeInfo)

	out := w.Body.String()
	if strings.Contains(out, `"model":"real-model"`) {
		t.Fatalf("claude stream output leaked upstream model: %s", out)
	}
	if !strings.Contains(out, `"model":"alias-model"`) {
		t.Fatalf("claude stream output missing alias model: %s", out)
	}
}

func TestRequestOpenAI2ClaudeMessageIgnoresUnsupportedFileContent(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					dto.MediaContent{
						Type: dto.ContentTypeText,
						Text: "see attachment",
					},
					dto.MediaContent{
						Type: dto.ContentTypeFile,
						File: &dto.MessageFile{
							FileName: "blob.bin",
							FileData: "JVBERi0xLjQK",
						},
					},
				},
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)
	require.Len(t, claudeRequest.Messages, 1)

	content, ok := claudeRequest.Messages[0].Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	require.Len(t, content, 1)
	require.Equal(t, "text", content[0].Type)
	require.NotNil(t, content[0].Text)
	require.Equal(t, "see attachment", *content[0].Text)
}

func TestRequestOpenAI2ClaudeMessageSupportsPDFFileContent(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					dto.MediaContent{
						Type: dto.ContentTypeFile,
						File: &dto.MessageFile{
							FileName: "spec.pdf",
							FileData: "JVBERi0xLjQK",
						},
					},
					dto.MediaContent{
						Type: dto.ContentTypeText,
						Text: "summarize it",
					},
				},
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)
	require.Len(t, claudeRequest.Messages, 1)

	content, ok := claudeRequest.Messages[0].Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	require.Len(t, content, 2)
	require.Equal(t, "document", content[0].Type)
	require.NotNil(t, content[0].Source)
	require.Equal(t, "base64", content[0].Source.Type)
	require.Equal(t, "application/pdf", content[0].Source.MediaType)
	require.Equal(t, "JVBERi0xLjQK", content[0].Source.Data)
	require.Equal(t, "text", content[1].Type)
	require.NotNil(t, content[1].Text)
	require.Equal(t, "summarize it", *content[1].Text)
}

func TestRequestOpenAI2ClaudeMessageConvertsTextFileContentToText(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					dto.MediaContent{
						Type: dto.ContentTypeFile,
						File: &dto.MessageFile{
							FileName: "notes.txt",
							FileData: base64.StdEncoding.EncodeToString([]byte("alpha\nbeta")),
						},
					},
				},
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)
	require.Len(t, claudeRequest.Messages, 1)

	content, ok := claudeRequest.Messages[0].Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	require.Len(t, content, 1)
	require.Equal(t, "text", content[0].Type)
	require.NotNil(t, content[0].Text)
	require.Equal(t, "alpha\nbeta", *content[0].Text)
}

// --- Issue #141: Claude 工具调用五件套回归测试（用户视角） ---

func buildAssistantToolCallMessage(t *testing.T, id, name, args string) dto.Message {
	t.Helper()
	msg := dto.Message{
		Role:    "assistant",
		Content: "",
	}
	msg.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   id,
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      name,
				Arguments: args,
			},
		},
	})
	return msg
}

func findToolUse(content any, toolUseId string) (dto.ClaudeMediaMessage, bool) {
	media, ok := content.([]dto.ClaudeMediaMessage)
	if !ok {
		return dto.ClaudeMediaMessage{}, false
	}
	for _, m := range media {
		if m.Type == "tool_use" && m.Id == toolUseId {
			return m, true
		}
	}
	return dto.ClaudeMediaMessage{}, false
}

// 空字符串的 tool arguments 仍需保留 tool_use 块，否则后续 tool_result 会引用不存在的 tool_use_id。
func TestRequestOpenAI2ClaudeMessagePreservesToolUseWithEmptyArguments(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
			buildAssistantToolCallMessage(t, "call_empty", "lookup", ""),
			{
				Role:       "tool",
				ToolCallId: "call_empty",
				Content:    "ok",
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)

	// assistant 消息必须包含对应的 tool_use 块
	var assistantFound bool
	for _, m := range claudeRequest.Messages {
		if tu, ok := findToolUse(m.Content, "call_empty"); ok {
			assistantFound = true
			require.Equal(t, "lookup", tu.Name, "tool_use name should be preserved")
			inputMap, ok := tu.Input.(map[string]any)
			require.True(t, ok, "tool_use input should be an empty object, got %T", tu.Input)
			require.Empty(t, inputMap, "tool_use input should default to {} when arguments empty")
		}
	}
	require.True(t, assistantFound, "empty-arguments tool_use block was dropped")
}

// 畸形的 tool arguments 不应丢弃整个 tool_use，避免破坏与后续 tool_result 的配对。
func TestRequestOpenAI2ClaudeMessagePreservesToolUseWithMalformedArguments(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{Role: "user", Content: "hi"},
			buildAssistantToolCallMessage(t, "call_bad", "search", "{not-json"),
			{
				Role:       "tool",
				ToolCallId: "call_bad",
				Content:    "result",
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)

	tu, ok := findToolUse(claudeRequest.Messages[1].Content, "call_bad")
	require.True(t, ok, "malformed-arguments tool_use block was dropped")
	require.Equal(t, "search", tu.Name)
	inputMap, ok := tu.Input.(map[string]any)
	require.True(t, ok, "tool_use input should default to an object on malformed args")
	require.Empty(t, inputMap)
}

// 空文本部分不应被发送给 Bedrock（会返回 400），但整条消息为空时用占位符兜底。
func TestRequestOpenAI2ClaudeMessageOmitsEmptyTextBlocks(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					dto.MediaContent{Type: dto.ContentTypeText, Text: ""},
					dto.MediaContent{Type: dto.ContentTypeText, Text: "real question"},
				},
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)
	require.Len(t, claudeRequest.Messages, 1)

	content, ok := claudeRequest.Messages[0].Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	// 空 text 部分应被跳过，仅保留非空 text
	require.Len(t, content, 1)
	require.Equal(t, "text", content[0].Type)
	require.NotNil(t, content[0].Text)
	require.Equal(t, "real question", *content[0].Text)
}

// 当 content 数组所有部分都被过滤为空时，应回退为占位符，避免 Bedrock 400。
func TestRequestOpenAI2ClaudeMessageFallsBackToPlaceholderWhenContentEmpty(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					dto.MediaContent{Type: dto.ContentTypeText, Text: "   "},
				},
			},
		},
	}

	claudeRequest, err := RequestOpenAI2ClaudeMessage(nil, nil, request)
	require.NoError(t, err)
	require.Len(t, claudeRequest.Messages, 1)

	content, ok := claudeRequest.Messages[0].Content.([]dto.ClaudeMediaMessage)
	require.True(t, ok)
	require.Len(t, content, 1, "empty content should fall back to a placeholder text block")
	require.Equal(t, "text", content[0].Type)
	require.NotNil(t, content[0].Text)
	require.Equal(t, "...", *content[0].Text)
}

// Claude 流式 content_block.index 已是 0 基，不能再 -1，否则并发工具调用 index 0 与 1 会撞键。
func TestStreamResponseClaude2OpenAIUsesClaudeZeroBasedToolIndexes(t *testing.T) {
	idx0 := 0
	idx1 := 1
	first := StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
		Type:  "content_block_start",
		Index: &idx0,
		ContentBlock: &dto.ClaudeMediaMessage{
			Type: "tool_use",
			Id:   "call_0",
			Name: "weather",
		},
	})
	require.NotNil(t, first)
	require.Len(t, first.Choices, 1)
	require.Len(t, first.Choices[0].Delta.ToolCalls, 1)
	require.NotNil(t, first.Choices[0].Delta.ToolCalls[0].Index)
	require.Equal(t, 0, *first.Choices[0].Delta.ToolCalls[0].Index)
	require.Equal(t, "call_0", first.Choices[0].Delta.ToolCalls[0].ID)

	second := StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
		Type:  "content_block_start",
		Index: &idx1,
		ContentBlock: &dto.ClaudeMediaMessage{
			Type: "tool_use",
			Id:   "call_1",
			Name: "calendar",
		},
	})
	require.NotNil(t, second)
	require.Len(t, second.Choices[0].Delta.ToolCalls, 1)
	require.NotNil(t, second.Choices[0].Delta.ToolCalls[0].Index)
	require.Equal(t, 1, *second.Choices[0].Delta.ToolCalls[0].Index, "second concurrent tool call must keep index 1, not collapse to 0")
}

// 空 partial_json 的 input_json_delta 不应解引用 nil 导致崩溃。
func TestStreamResponseClaude2OpenAINilInputJSONDeltaDoesNotPanic(t *testing.T) {
	idx := 0
	require.NotPanics(t, func() {
		resp := StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
			Type:  "content_block_delta",
			Index: &idx,
			Delta: &dto.ClaudeMediaMessage{
				Type:       "input_json_delta",
				PartialJson: nil,
			},
		})
		// 即使 partial_json 为空，也应安全返回（参数为空字符串），而不是崩溃
		if resp != nil {
			require.Len(t, resp.Choices[0].Delta.ToolCalls, 1)
			require.Equal(t, "", resp.Choices[0].Delta.ToolCalls[0].Function.Arguments)
		}
	})
}

// 并发工具调用的参数增量应保持各自 index，互不覆盖。
func TestStreamResponseClaude2OpenAIParallelToolArgumentDeltasKeepIndexes(t *testing.T) {
	idx0 := 0
	idx1 := 1
	partial0 := `{"a"`
	partial1 := `{"b"`

	a := StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
		Type:  "content_block_delta",
		Index: &idx0,
		Delta: &dto.ClaudeMediaMessage{
			Type:        "input_json_delta",
			PartialJson: &partial0,
		},
	})
	b := StreamResponseClaude2OpenAI(&dto.ClaudeResponse{
		Type:  "content_block_delta",
		Index: &idx1,
		Delta: &dto.ClaudeMediaMessage{
			Type:        "input_json_delta",
			PartialJson: &partial1,
		},
	})

	require.Equal(t, 0, *a.Choices[0].Delta.ToolCalls[0].Index)
	require.Equal(t, `{"a"`, a.Choices[0].Delta.ToolCalls[0].Function.Arguments)
	require.Equal(t, 1, *b.Choices[0].Delta.ToolCalls[0].Index)
	require.Equal(t, `{"b"`, b.Choices[0].Delta.ToolCalls[0].Function.Arguments)
}
