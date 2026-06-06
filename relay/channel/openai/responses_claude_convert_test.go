package openai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	relayconstant "github.com/zhongruan0522/new-api/relay/constant"
	"github.com/zhongruan0522/new-api/types"
)

func TestConvertClaudeRequestToResponsesUpstreamUsesSharedRules(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RelayFormat:                types.RelayFormatClaude,
		RelayMode:                  relayconstant.RelayModeChatCompletions,
		RequestConversionChain:     []types.RelayFormat{types.RelayFormatClaude},
		ClaudeConvertInfo:          &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone},
		ShouldIncludeUsage:         false,
		IsStream:                   true,
		RequestURLPath:             "/v1/messages",
		OpenAIResponsesToolContext: nil,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting:       dto.ChannelSettings{OpenAIWireAPI: dto.OpenAIWireAPIResponses},
			SupportStreamOptions: true,
			UpstreamModelName:    "gpt-5",
		},
	}
	request := &dto.ClaudeRequest{
		Model:     "gpt-5",
		MaxTokens: 256,
		Stream:    true,
		Tools: []any{dto.Tool{
			Name:        "weather",
			Description: "Get weather",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{"type": "string"},
				},
			},
		}},
		Messages: []dto.ClaudeMessage{{
			Role:    "user",
			Content: "weather in Shanghai?",
		}},
	}

	convertedAny, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, request)
	if err != nil {
		t.Fatalf("ConvertClaudeRequest error = %v", err)
	}
	converted, ok := convertedAny.(*dto.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("converted type = %T, want *dto.OpenAIResponsesRequest", convertedAny)
	}
	if info.RelayMode != relayconstant.RelayModeResponses || info.RequestURLPath != "/v1/responses" {
		t.Fatalf("upstream mode/path = %d/%q, want responses /v1/responses", info.RelayMode, info.RequestURLPath)
	}
	if !converted.Stream || converted.MaxOutputTokens != 256 {
		t.Fatalf("converted stream/max_output_tokens = %t/%d, want true/256", converted.Stream, converted.MaxOutputTokens)
	}
	if len(converted.Tools) == 0 {
		t.Fatal("converted responses tools are empty")
	}
	wantChain := []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI, types.RelayFormatOpenAIResponses}
	if len(info.RequestConversionChain) != len(wantChain) {
		t.Fatalf("RequestConversionChain = %#v, want %#v", info.RequestConversionChain, wantChain)
	}
	for i := range wantChain {
		if info.RequestConversionChain[i] != wantChain[i] {
			t.Fatalf("RequestConversionChain = %#v, want %#v", info.RequestConversionChain, wantChain)
		}
	}
}

func TestConvertResponsesBodyToClaudeBodyPreservesTextToolAndUsage(t *testing.T) {
	body, err := convertResponsesBodyToClaudeBody(&dto.OpenAIResponsesResponse{
		ID:        "resp_1",
		Model:     "gpt-5",
		CreatedAt: 1700000000,
		Status:    "completed",
		Output: []dto.ResponsesOutput{
			{
				Type:   "message",
				ID:     "msg_1",
				Status: "completed",
				Role:   "assistant",
				Content: []dto.ResponsesOutputContent{{
					Type: "output_text",
					Text: "hello",
				}},
			},
			{
				Type:      "function_call",
				ID:        "fc_1",
				Status:    "completed",
				CallId:    "call_weather",
				Name:      "weather",
				Arguments: `{"city":"Shanghai"}`,
			},
		},
		Usage: &dto.Usage{
			InputTokens:  10,
			OutputTokens: 4,
			TotalTokens:  14,
		},
	}, &dto.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14}, &relaycommon.RelayInfo{})
	if err != nil {
		t.Fatalf("convertResponsesBodyToClaudeBody error = %v", err)
	}

	var got dto.ClaudeResponse
	if err := common.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal Claude response error = %v", err)
	}
	if got.Type != "message" || got.StopReason != "tool_use" {
		t.Fatalf("Claude response type/stop = %q/%q, want message/tool_use", got.Type, got.StopReason)
	}
	if got.Usage == nil || got.Usage.InputTokens != 10 || got.Usage.OutputTokens != 4 {
		t.Fatalf("usage = %+v, want input=10 output=4", got.Usage)
	}
	if len(got.Content) != 2 {
		t.Fatalf("content len = %d, want 2: %+v", len(got.Content), got.Content)
	}
	var sawText, sawTool bool
	for _, block := range got.Content {
		switch block.Type {
		case "text":
			sawText = block.GetText() == "hello"
		case "tool_use":
			sawTool = block.Id == "call_weather" && block.Name == "weather"
		}
	}
	if !sawText {
		t.Fatalf("content = %+v, want text block hello", got.Content)
	}
	if !sawTool {
		t.Fatalf("content = %+v, want tool_use weather/call_weather", got.Content)
	}
}

func TestWriteResponsesStreamAsClaudeEmitsClaudeEvents(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-5"},
	}
	converter := relaycommon.NewResponsesToChatStreamConverter(false)

	textEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type:  "response.output_text.delta",
		Delta: "hello",
		Response: &dto.OpenAIResponsesResponse{
			ID:        "resp_1",
			Model:     "gpt-5",
			CreatedAt: 1700000000,
		},
	})
	if err != nil {
		t.Fatalf("marshal text event error = %v", err)
	}
	if err := writeResponsesStreamAsClaude(c, info, converter, string(textEvent)); err != nil {
		t.Fatalf("write text event error = %v", err)
	}

	completedEvent, err := common.Marshal(dto.ResponsesStreamResponse{
		Type: "response.completed",
		Response: &dto.OpenAIResponsesResponse{
			ID:        "resp_1",
			Model:     "gpt-5",
			CreatedAt: 1700000000,
			Status:    "completed",
			Usage: &dto.Usage{
				InputTokens:  10,
				OutputTokens: 4,
				TotalTokens:  14,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal completed event error = %v", err)
	}
	if err := writeResponsesStreamAsClaude(c, info, converter, string(completedEvent)); err != nil {
		t.Fatalf("write completed event error = %v", err)
	}

	out := recorder.Body.String()
	for _, want := range []string{
		"event: message_start",
		"event: content_block_start",
		`"type":"text_delta"`,
		`"text":"hello"`,
		"event: message_delta",
		"event: message_stop",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stream output missing %q:\n%s", want, out)
		}
	}
	if info.ClaudeConvertInfo == nil {
		t.Fatal("ClaudeConvertInfo was not initialized")
	}
}
