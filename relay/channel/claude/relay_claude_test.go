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
