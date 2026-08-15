package controller

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/NookMux/NookMux/constant"
	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/model"
)

func newChannelTestValidateContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c
}

func TestBuildChannelTestPromptLength(t *testing.T) {
	for i := 0; i < 50; i++ {
		prompt := buildChannelTestPrompt("openai", "gpt-4o-mini", false)
		if !prompt.requiresTextAnswer {
			t.Fatalf("expected text answer validation for chat endpoint")
		}
		length := len([]rune(prompt.prompt))
		if length < 50 || length > 100 {
			t.Fatalf("prompt length %d out of range [50,100]: %q", length, prompt.prompt)
		}
		if !strings.Contains(prompt.prompt, "final integer") {
			t.Fatalf("prompt missing final-integer instruction: %q", prompt.prompt)
		}
	}
}

func TestMatchesChannelTestExpectedAnswer(t *testing.T) {
	if !matchesChannelTestExpectedAnswer("2", 2) {
		t.Fatal("expected plain integer to match")
	}
	if !matchesChannelTestExpectedAnswer(" 2\n", 2) {
		t.Fatal("expected trimmed integer to match")
	}
	if matchesChannelTestExpectedAnswer("2 is the answer", 2) {
		t.Fatal("expected multi-token text to fail")
	}
	if matchesChannelTestExpectedAnswer("3", 2) {
		t.Fatal("expected wrong integer to fail")
	}
}

func TestExtractChannelTestAITextFromChatResponse(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"content":"2"}}]}`)
	text := extractChannelTestAIText(body)
	if text != "2" {
		t.Fatalf("unexpected extracted text: %q", text)
	}
}

func TestExtractChannelTestAITextFromStreamResponse(t *testing.T) {
	body := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"2\"}}]}\n\ndata: [DONE]\n\n")
	text := extractChannelTestAIText(body)
	if text != "2" {
		t.Fatalf("unexpected extracted stream text: %q", text)
	}
}

func TestChannelTestEmbeddingModelClassification(t *testing.T) {
	modelName := "text-embed-3-small"
	if supportsChannelTestTool("", modelName) {
		t.Fatal("expected auto-detected embed model to be incompatible with tool tests")
	}
	if requiresChannelTestTextAnswer("", modelName) {
		t.Fatal("expected auto-detected embed model to skip text answer validation")
	}
	request := buildTestRequest(modelName, "", nil, false, channelTestPrompt{})
	if _, ok := request.(*dto.EmbeddingRequest); !ok {
		t.Fatalf("expected embed model to build embedding request, got %T", request)
	}
}

func TestResponseHasChannelTestToolCall(t *testing.T) {
	withTool := []byte(`{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"report_result","arguments":"{\"value\":42}"}}]}}]}`)
	if !responseHasChannelTestToolCall(withTool) {
		t.Fatal("expected tool call to be detected")
	}
	withoutTool := []byte(`{"choices":[{"message":{"content":"42"}}]}`)
	if responseHasChannelTestToolCall(withoutTool) {
		t.Fatal("did not expect tool call detection for plain text")
	}
}

func TestValidateChannelTestResponseToolUnsupported(t *testing.T) {
	err := validateChannelTestResponse(
		newChannelTestValidateContext(),
		[]byte(`{"choices":[{"message":{"content":"hello"}}]}`),
		channelTestPrompt{isTool: true},
	)
	if err == nil {
		t.Fatal("expected tool validation failure")
	}
	if !strings.Contains(err.Error(), channelTestToolNotSupported) {
		t.Fatalf("expected tool unsupported message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "hello") {
		t.Fatalf("expected original AI text in error, got %q", err.Error())
	}
}

func TestValidateChannelTestResponseArithmeticUsesOriginalText(t *testing.T) {
	err := validateChannelTestResponse(
		newChannelTestValidateContext(),
		[]byte(`{"choices":[{"message":{"content":"I think it is four"}}]}`),
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 2},
	)
	if err == nil {
		t.Fatal("expected arithmetic validation failure")
	}
	if err.Error() != "I think it is four" {
		t.Fatalf("expected original AI text as error message, got %q", err.Error())
	}
}

func TestSupportsChannelTestToolForChannel(t *testing.T) {
	cases := []struct {
		name         string
		channelType  int
		endpointType string
		modelName    string
		want         bool
	}{
		{
			name:         "openai chat allowed",
			channelType:  constant.ChannelTypeOpenAI,
			endpointType: string(constant.EndpointTypeOpenAI),
			modelName:    "gpt-4o-mini",
			want:         true,
		},
		{
			name:         "openai responses allowed",
			channelType:  constant.ChannelTypeOpenAI,
			endpointType: string(constant.EndpointTypeOpenAIResponse),
			modelName:    "gpt-4o-mini",
			want:         true,
		},
		{
			name:         "claude chat allowed",
			channelType:  constant.ChannelTypeAnthropic,
			endpointType: string(constant.EndpointTypeAnthropic),
			modelName:    "claude-3-5-sonnet",
			want:         true,
		},
		{
			name:         "claude responses allowed via chat conversion",
			channelType:  constant.ChannelTypeAnthropic,
			endpointType: string(constant.EndpointTypeOpenAIResponse),
			modelName:    "claude-3-5-sonnet",
			want:         true,
		},
		{
			name:         "gemini chat allowed",
			channelType:  constant.ChannelTypeGemini,
			endpointType: string(constant.EndpointTypeGemini),
			modelName:    "gemini-2.0-flash",
			want:         true,
		},
		{
			name:         "gemini responses rejected (conversion not implemented)",
			channelType:  constant.ChannelTypeGemini,
			endpointType: string(constant.EndpointTypeOpenAIResponse),
			modelName:    "gemini-2.0-flash",
			want:         false,
		},
		{
			name:         "vertex responses rejected",
			channelType:  constant.ChannelTypeVertexAi,
			endpointType: string(constant.EndpointTypeOpenAIResponse),
			modelName:    "claude-3-5-sonnet@vertex",
			want:         false,
		},
		{
			name:         "aws chat allowed for claude models",
			channelType:  constant.ChannelTypeAws,
			endpointType: string(constant.EndpointTypeOpenAI),
			modelName:    "claude-3-5-sonnet",
			want:         true,
		},
		{
			name:         "aws nova chat rejected (tools dropped)",
			channelType:  constant.ChannelTypeAws,
			endpointType: string(constant.EndpointTypeOpenAI),
			modelName:    "amazon-nova-pro",
			want:         false,
		},
		{
			name:         "embed endpoint rejected",
			channelType:  constant.ChannelTypeOpenAI,
			endpointType: string(constant.EndpointTypeEmbeddings),
			modelName:    "text-embed-3-small",
			want:         false,
		},
		{
			name:         "auto detect codex on openai allowed",
			channelType:  constant.ChannelTypeOpenAI,
			endpointType: "",
			modelName:    "codex-mini-latest",
			want:         true,
		},
		{
			name:         "auto detect compact model rejected",
			channelType:  constant.ChannelTypeOpenAI,
			endpointType: "",
			modelName:    "gpt-4o-mini-openai-compact",
			want:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			channel := &model.Channel{Type: tc.channelType}
			got := supportsChannelTestToolForChannel(channel, tc.endpointType, tc.modelName)
			if got != tc.want {
				t.Fatalf("supportsChannelTestToolForChannel(%d, %q, %q) = %v, want %v",
					tc.channelType, tc.endpointType, tc.modelName, got, tc.want)
			}
		})
	}
}

func TestApplyChannelTestToolsToChatRequestUsesCompatibleChoice(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{}
	applyChannelTestToolsToChatRequest(req, channelTestPrompt{isTool: true})

	if len(req.Tools) != 1 {
		t.Fatalf("expected exactly one tool, got %d", len(req.Tools))
	}
	if req.Tools[0].Function.Name != channelTestToolName {
		t.Fatalf("expected tool name %q, got %q", channelTestToolName, req.Tools[0].Function.Name)
	}
	choice, ok := req.ToolChoice.(string)
	if !ok {
		t.Fatalf("expected tool_choice to be string for maximum upstream compatibility, got %T", req.ToolChoice)
	}
	if choice != "required" {
		t.Fatalf("expected tool_choice = \"required\", got %q", choice)
	}
}

func TestApplyChannelTestToolsToChatRequestLeavesRequestUntouchedWhenNotTool(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{}
	applyChannelTestToolsToChatRequest(req, channelTestPrompt{isTool: false})

	if req.Tools != nil {
		t.Fatalf("expected tools to remain nil for non-tool test, got %+v", req.Tools)
	}
	if req.ToolChoice != nil {
		t.Fatalf("expected tool_choice to remain nil for non-tool test, got %+v", req.ToolChoice)
	}
}

func TestApplyChannelTestToolsToResponsesRequestShape(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	applyChannelTestToolsToResponsesRequest(req, channelTestPrompt{isTool: true})

	if len(req.Tools) == 0 {
		t.Fatal("expected tools to be set on responses request")
	}
	var tools []map[string]any
	if err := json.Unmarshal(req.Tools, &tools); err != nil {
		t.Fatalf("failed to unmarshal responses tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected exactly one responses tool, got %d", len(tools))
	}
	if name, _ := tools[0]["name"].(string); name != channelTestToolName {
		t.Fatalf("expected responses tool name %q, got %v", channelTestToolName, tools[0]["name"])
	}
	if toolType, _ := tools[0]["type"].(string); toolType != "function" {
		t.Fatalf("expected responses tool type \"function\", got %v", tools[0]["type"])
	}

	var choice string
	if err := json.Unmarshal(req.ToolChoice, &choice); err != nil {
		t.Fatalf("expected responses tool_choice to be a JSON string for compatibility, body=%s err=%v", string(req.ToolChoice), err)
	}
	if choice != "required" {
		t.Fatalf("expected responses tool_choice = \"required\", got %q", choice)
	}
}

func TestApplyChannelTestToolsToResponsesRequestLeavesRequestUntouchedWhenNotTool(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	applyChannelTestToolsToResponsesRequest(req, channelTestPrompt{isTool: false})

	if len(req.Tools) != 0 {
		t.Fatalf("expected tools to remain empty for non-tool responses test, got %s", string(req.Tools))
	}
	if len(req.ToolChoice) != 0 {
		t.Fatalf("expected tool_choice to remain empty for non-tool responses test, got %s", string(req.ToolChoice))
	}
}

func TestBuildTestRequestToolChatStreamOptions(t *testing.T) {
	// 工具测试 + 流式: Chat 请求需要保留 stream + stream_options, 让 adaptor 决定是否清理。
	streamReq, ok := buildTestRequest(
		"gpt-4o-mini",
		string(constant.EndpointTypeOpenAI),
		nil,
		true,
		channelTestPrompt{isTool: true},
	).(*dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected general chat request for openai endpoint tool test, got %T", streamReq)
	}
	if !streamReq.Stream {
		t.Fatal("expected stream=true to be preserved on tool test request")
	}
	if streamReq.StreamOptions == nil || !streamReq.StreamOptions.IncludeUsage {
		t.Fatal("expected stream_options.include_usage to be set for chat tool stream test")
	}
	if len(streamReq.Tools) != 1 {
		t.Fatalf("expected tool test request to carry one tool, got %d", len(streamReq.Tools))
	}
	if choice, ok := streamReq.ToolChoice.(string); !ok || choice != "required" {
		t.Fatalf("expected tool_choice string \"required\", got %+v", streamReq.ToolChoice)
	}
}

func TestBuildTestRequestToolResponsesShape(t *testing.T) {
	req, ok := buildTestRequest(
		"gpt-4o-mini",
		string(constant.EndpointTypeOpenAIResponse),
		nil,
		false,
		channelTestPrompt{isTool: true},
	).(*dto.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("expected responses request for responses endpoint tool test, got %T", req)
	}
	if len(req.Tools) == 0 {
		t.Fatal("expected tools to be set on responses tool test request")
	}
	if len(req.ToolChoice) == 0 {
		t.Fatal("expected tool_choice to be set on responses tool test request")
	}
}

func TestResponseHasChannelTestToolCallRequiredChoicePath(t *testing.T) {
	// 即便上游严格按 "required" 调用唯一工具,响应里 tool_calls.function.name 仍应为 report_result,
	// 这里覆盖非流式和 SSE 流式两种响应路径。
	nonStream := []byte(`{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"report_result","arguments":"{\"value\":42}"}}]}}]}`)
	if !responseHasChannelTestToolCall(nonStream) {
		t.Fatal("expected non-stream required-choice tool call to be detected")
	}

	stream := []byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"name\":\"report_result\",\"arguments\":\"{\\\"value\\\":42}\"}}]}}]}\n\ndata: [DONE]\n\n")
	if !responseHasChannelTestToolCall(stream) {
		t.Fatal("expected stream required-choice tool call to be detected")
	}
}

func TestIsChannelTestOSeriesModel(t *testing.T) {
	positive := []string{"o1", "o3", "o4", "o5", "o1-mini", "o3-pro", "o4-mini-high", "O3-MINI"}
	negative := []string{"", "gpt-4o", "gpt-4o-mini", "openai", "ollama", "o", "oa1", "o100", "o1n"}
	for _, name := range positive {
		if !isChannelTestOSeriesModel(name) {
			t.Fatalf("expected %q to be detected as o-series reasoning model", name)
		}
	}
	for _, name := range negative {
		if isChannelTestOSeriesModel(name) {
			t.Fatalf("did not expect %q to be detected as o-series reasoning model", name)
		}
	}
}

func TestIsChannelTestReasoningModel(t *testing.T) {
	positive := []string{
		"deepseek-r1", "deepseek-reasoner", "qwq-32b",
		"gpt-oss:thinking", "gemini-2.5-flash-thinking",
		"o1-mini", "o3", "qwen-r1",
	}
	negative := []string{
		"", "gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet",
		"claude-3-7-sonnet-thinking", "glm-5.2",
	}
	for _, name := range positive {
		if !isChannelTestReasoningModel(name) {
			t.Fatalf("expected %q to be detected as reasoning model", name)
		}
	}
	for _, name := range negative {
		if isChannelTestReasoningModel(name) {
			t.Fatalf("did not expect %q to be detected as reasoning model", name)
		}
	}
}

func TestResolveChannelTestTokenBudget(t *testing.T) {
	cases := []struct {
		name             string
		modelName        string
		endpointType     string
		isTool           bool
		wantMaxTokens    uint
		wantCompletion   uint
	}{
		{
			name:           "plain chat model gets baseline headroom",
			modelName:      "gpt-4o-mini",
			endpointType:   string(constant.EndpointTypeOpenAI),
			wantMaxTokens:  channelTestPlainMaxTokens,
			wantCompletion: 0,
		},
		{
			name:           "reasoning model gets larger budget",
			modelName:      "deepseek-r1",
			endpointType:   string(constant.EndpointTypeOpenAI),
			wantMaxTokens:  channelTestReasoningMaxTokens,
			wantCompletion: 0,
		},
		{
			name:           "o-series uses max_completion_tokens",
			modelName:      "o3-mini",
			endpointType:   string(constant.EndpointTypeOpenAI),
			wantMaxTokens:  0,
			wantCompletion: channelTestReasoningMaxTokens,
		},
		{
			name:           "gemini endpoint keeps generous baseline",
			modelName:      "gemini-2.0-flash",
			endpointType:   string(constant.EndpointTypeGemini),
			wantMaxTokens:  channelTestGeminiMaxTokens,
			wantCompletion: 0,
		},
		{
			name:           "plain tool test keeps tool budget",
			modelName:      "gpt-4o-mini",
			endpointType:   string(constant.EndpointTypeOpenAI),
			isTool:         true,
			wantMaxTokens:  channelTestPlainToolMaxTokens,
			wantCompletion: 0,
		},
		{
			name:           "reasoning tool test gets larger tool budget",
			modelName:      "deepseek-r1",
			endpointType:   string(constant.EndpointTypeOpenAI),
			isTool:         true,
			wantMaxTokens:  channelTestReasoningToolMaxTokens,
			wantCompletion: 0,
		},
		{
			name:           "auto-detect plain model baseline",
			modelName:      "glm-5.2",
			endpointType:   "",
			wantMaxTokens:  channelTestPlainMaxTokens,
			wantCompletion: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			maxTokens, maxCompletion := resolveChannelTestTokenBudget(tc.modelName, tc.endpointType, tc.isTool)
			if maxTokens != tc.wantMaxTokens {
				t.Fatalf("maxTokens = %d, want %d", maxTokens, tc.wantMaxTokens)
			}
			if maxCompletion != tc.wantCompletion {
				t.Fatalf("maxCompletionTokens = %d, want %d", maxCompletion, tc.wantCompletion)
			}
		})
	}
}

func TestBuildTestRequestExplicitOpenAIGivesReasoningModelEnoughTokens(t *testing.T) {
	// Regression: selecting the OpenAI-Chat endpoint used to hardcode
	// max_tokens=32, truncating reasoning models mid-thought and producing an
	// empty content body. Reasoning models must now receive a larger budget
	// even through the explicit endpoint path.
	req, ok := buildTestRequest(
		"deepseek-r1",
		string(constant.EndpointTypeOpenAI),
		nil,
		false,
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 10},
	).(*dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected GeneralOpenAIRequest, got %T", req)
	}
	if req.MaxTokens < channelTestReasoningMaxTokens {
		t.Fatalf("reasoning model via explicit OpenAI endpoint got max_tokens=%d, want >= %d",
			req.MaxTokens, channelTestReasoningMaxTokens)
	}
}

func TestBuildTestRequestExplicitOpenAIOSeriesUsesCompletionTokens(t *testing.T) {
	req, ok := buildTestRequest(
		"o3-mini",
		string(constant.EndpointTypeOpenAI),
		nil,
		false,
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 10},
	).(*dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected GeneralOpenAIRequest, got %T", req)
	}
	if req.MaxCompletionTokens == 0 {
		t.Fatal("o-series via explicit OpenAI endpoint should set max_completion_tokens, got 0")
	}
}

func TestBuildTestRequestExplicitOpenAIPlainModelHasBaselineHeadroom(t *testing.T) {
	// Non-reasoning models that are not detectable by name (like glm-5.2)
	// must receive more than the old 32-token cap so undetectable reasoning
	// variants have room to finish simple arithmetic.
	req, ok := buildTestRequest(
		"glm-5.2",
		string(constant.EndpointTypeOpenAI),
		nil,
		false,
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 10},
	).(*dto.GeneralOpenAIRequest)
	if !ok {
		t.Fatalf("expected GeneralOpenAIRequest, got %T", req)
	}
	if req.MaxTokens <= 32 {
		t.Fatalf("plain model via explicit OpenAI endpoint got max_tokens=%d, want > 32", req.MaxTokens)
	}
}

func TestExtractChannelTestReasoningTextFromChatResponse(t *testing.T) {
	// Mirrors the user-reported glm-5.2 shape: empty content, reasoning in
	// reasoning_content, finish_reason=length.
	body := []byte(`{"choices":[{"finish_reason":"length","index":0,"message":{"content":"","reasoning_content":"step 1: compute 50/5","role":"assistant"}}]}`)
	text := extractChannelTestReasoningText(body)
	if text != "step 1: compute 50/5" {
		t.Fatalf("unexpected reasoning text: %q", text)
	}
}

func TestExtractChannelTestReasoningTextFromStreamResponse(t *testing.T) {
	body := []byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"think\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"reasoning_content\":\"ing\"}}]}\n\ndata: [DONE]\n\n")
	text := extractChannelTestReasoningText(body)
	if text != "thinking" {
		t.Fatalf("unexpected accumulated reasoning text: %q", text)
	}
}

func TestExtractChannelTestReasoningTextFromReasoningField(t *testing.T) {
	// Some upstreams use "reasoning" instead of "reasoning_content".
	body := []byte(`{"choices":[{"message":{"content":"","reasoning":"deduce the answer"}}]}`)
	text := extractChannelTestReasoningText(body)
	if text != "deduce the answer" {
		t.Fatalf("unexpected reasoning field text: %q", text)
	}
}

func TestValidateChannelTestResponseReasoningOnly(t *testing.T) {
	// When content is empty but reasoning exists, the error must explain the
	// situation instead of the generic "empty model response".
	body := []byte(`{"choices":[{"finish_reason":"length","index":0,"message":{"content":"","reasoning_content":"I am thinking...","role":"assistant"}}]}`)
	err := validateChannelTestResponse(
		newChannelTestValidateContext(),
		body,
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 10},
	)
	if err == nil {
		t.Fatal("expected validation failure for reasoning-only response")
	}
	if strings.Contains(err.Error(), "empty") || strings.Contains(err.Error(), "空") {
		t.Fatalf("should not report generic empty response for reasoning-only body, got %q", err.Error())
	}
}

func TestValidateChannelTestResponseTrulyEmpty(t *testing.T) {
	// No content and no reasoning at all should still produce the generic
	// empty-response error.
	body := []byte(`{"choices":[{"finish_reason":"stop","index":0,"message":{"content":"","role":"assistant"}}]}`)
	err := validateChannelTestResponse(
		newChannelTestValidateContext(),
		body,
		channelTestPrompt{requiresTextAnswer: true, expectedAnswer: 10},
	)
	if err == nil {
		t.Fatal("expected validation failure for truly empty response")
	}
}
