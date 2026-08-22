package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/gin-gonic/gin"
)

// 上游在 HTTP 200 的 SSE 流中返回错误帧（如 429 限流被部分网关转成 200 + error 帧）。
// 流处理器若不识别错误帧，会在计费阶段因 totalTokens=0 被误记为
// 「status_code=502, 上游没有返回计费信息，无法扣费（可能是上游超时）」，
// 而客户端实际拿到的是真实上游错误。这里验证错误帧被原样暴露。
func newStreamErrorTestContext(t *testing.T, path string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c, w
}

func newSSEStreamResponse(frames ...string) *http.Response {
	body := ""
	for _, frame := range frames {
		body += "data: " + frame + "\n\n"
	}
	body += "data: [DONE]\n\n"
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestOaiStreamHandlerReturnsUpstreamErrorFrame(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/chat/completions")
	info := testRelayInfo("alias-model", "real-model")
	info.RelayMode = 1 // relayconstant.RelayModeChatCompletions

	resp := newSSEStreamResponse(`{"id":"chatcmpl_err","object":"chat.completion.chunk","created":1710000000,"model":"real-model","error":{"message":"Rate limit exceeded for model","type":"rate_limit_error","code":"429"}}`)

	usage, apiErr := OaiStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "error frame must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Rate limit exceeded")
	// 错误帧 code 中的真实状态码被还原，日志不再统一误报 502。
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Equal(t, shared.ErrorCode("429"), apiErr.GetErrorCode())
}

func TestOaiResponsesStreamHandlerReturnsUpstreamFailedEvent(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/responses")
	info := testRelayInfo("alias-model", "real-model")

	resp := newSSEStreamResponse(`{"type":"response.failed","response":{"id":"resp_err","object":"response","created_at":1710000000,"status":"failed","model":"real-model","error":{"message":"Rate limit exceeded for responses","type":"rate_limit_error","code":"429"}}}`)

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "response.failed event must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Rate limit exceeded")
	// 错误载荷 code 中的真实状态码被还原，日志不再统一误报 502。
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Equal(t, shared.ErrorCode("429"), apiErr.GetErrorCode())
}

// 官方 Responses 流式规范定义了顶层 type:"error" 事件（message/code 平铺，
// 无嵌套 error 对象）。此前 ResponsesStreamResponse 无顶层字段承载，事件被
// 静默丢弃后落入「502 上游没有返回计费信息」兜底。
// 参考 https://developers.openai.com/api/reference/resources/responses/streaming-events
func TestOaiResponsesStreamHandlerReturnsTopLevelErrorEvent(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/responses")
	info := testRelayInfo("alias-model", "real-model")

	resp := newSSEStreamResponse(`{"type":"error","code":"credit_balance_exhausted","message":"Credit balance exhausted"}`)

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "type:error event must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Credit balance exhausted")
	// 官方非数字计费 code 还原为 429，日志不再统一误报 502。
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

// 回归保护（Chat 流）：内容携带 error 关键词不得被错误帧检测误报。
// Chat 流用 strings.Contains(data, `"error"`) 预筛——正常内容中的
// "error" 若带引号，经上游 JSON 序列化必然转义为 \"error\"，子串不匹配；
// 若上游发出未转义裸引号（无效 JSON），unmarshal 失败也不会进错误分支。
func TestOaiStreamHandlerErrorKeywordInContentIsNotError(t *testing.T) {
	cases := []struct {
		name  string
		frame string
	}{
		{"delta contains error word", `{"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Fix the error and retry. Error handling matters."},"finish_reason":null}]}`},
		{"delta is literal escaped error json", `{"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"{\"error\":{\"message\":\"Credit balance exhausted\",\"code\":\"insufficient_quota\"}}"},"finish_reason":null}]}`},
		{"function arguments reference error codes", `{"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"retry\":\"rate_limit_error\"}"}}]},"finish_reason":null}]}`},
		{"model name contains error", `{"id":"c1","object":"chat.completion.chunk","model":"my-error-fix-model","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`},
		{"normal usage final frame", `{"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newStreamErrorTestContext(t, "/v1/chat/completions")
			info := testRelayInfo("alias-model", "real-model")
			info.RelayMode = 1 // relayconstant.RelayModeChatCompletions
			resp := newSSEStreamResponse(tc.frame)

			usage, apiErr := OaiStreamHandler(c, info, resp)
			assert.Nil(t, apiErr, "normal content mentioning error must not be misdetected")
			require.NotNil(t, usage)
		})
	}
}

// 上游发出未转义裸 "error" 引号（无效 JSON）时，Contains 预筛命中但
// unmarshal 失败，同样不误报。
func TestOaiStreamHandlerMalformedJSONWithRawErrorWordIsNotError(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/chat/completions")
	info := testRelayInfo("alias-model", "real-model")
	info.RelayMode = 1

	resp := newSSEStreamResponse(`{"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"say "error" out loud"},"finish_reason":null}]}`)

	usage, apiErr := OaiStreamHandler(c, info, resp)
	assert.Nil(t, apiErr, "malformed JSON with raw error word must not be misdetected")
	require.NotNil(t, usage)
}

// 非流式（Chat）：content 含转义 error JSON 字符串不得误报。
func TestOpenaiHandlerErrorKeywordInContentIsNotError(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/chat/completions")
	info := testRelayInfo("alias-model", "real-model")
	info.RelayMode = 1

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"c1","object":"chat.completion","model":"real-model","choices":[{"index":0,"message":{"role":"assistant","content":"The API returned {\"error\":{\"message\":\"Credit balance exhausted\"}} which means quota is exhausted"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)),
	}

	usage, apiErr := OpenaiHandler(c, info, resp)
	assert.Nil(t, apiErr, "content with escaped error json must not be misdetected")
	require.NotNil(t, usage)
	assert.Equal(t, 15, usage.TotalTokens)
}

// 部分网关把 429/5xx 转成 HTTP 200 + 裸 {"error":{...}} 帧下发（无 type 字段）。
func TestOaiResponsesStreamHandlerReturnsBareErrorFrame(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/responses")
	info := testRelayInfo("alias-model", "real-model")

	resp := newSSEStreamResponse(`{"error":{"message":"Organization spend limit reached","type":"insufficient_quota","code":"organization_spend_limit_exceeded"}}`)

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "bare error frame must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Organization spend limit reached")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

// response.failed 事件的错误对象可能位于事件顶层而非 response 内。
func TestOaiResponsesStreamHandlerReturnsFailedEventWithTopLevelError(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/responses")
	info := testRelayInfo("alias-model", "real-model")

	resp := newSSEStreamResponse(`{"type":"response.failed","error":{"message":"Credit balance exhausted","type":"insufficient_quota","code":"credit_balance_exhausted"}}`)

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "failed event with top-level error must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Credit balance exhausted")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

// response.failed 事件可能只有 status:"failed" 而无 error 对象。
func TestOaiResponsesStreamHandlerReturnsFailedEventWithoutErrorObject(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/responses")
	info := testRelayInfo("alias-model", "real-model")

	resp := newSSEStreamResponse(`{"type":"response.failed","response":{"id":"resp_err","object":"response","status":"failed"}}`)

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.NotNil(t, apiErr, "failed event without error object must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
}

// 回归保护：正常 Responses 流（delta + completed 带 usage）不受统一错误检测影响。
func TestOaiResponsesStreamHandlerNormalUsageStillWorks(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/responses")
	info := testRelayInfo("alias-model", "real-model")

	resp := newSSEStreamResponse(
		`{"type":"response.output_text.delta","delta":"hello"}`,
		`{"type":"response.completed","response":{"id":"resp_ok","object":"response","status":"completed","usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`,
	)

	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	assert.Nil(t, apiErr, "normal responses stream should not be misdetected as error")
	require.NotNil(t, usage)
	assert.Equal(t, 15, usage.TotalTokens)
}

// 回归保护：正常带 usage 的流不应被错误识别影响。
func TestOaiStreamHandlerNormalUsageStillWorks(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/chat/completions")
	info := testRelayInfo("alias-model", "real-model")
	info.RelayMode = 1

	resp := newSSEStreamResponse(
		`{"id":"chatcmpl_ok","object":"chat.completion.chunk","created":1710000000,"model":"real-model","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`,
		`{"id":"chatcmpl_ok","object":"chat.completion.chunk","created":1710000000,"model":"real-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
	)

	usage, apiErr := OaiStreamHandler(c, info, resp)
	assert.Nil(t, apiErr, "normal stream should not be misdetected as error")
	require.NotNil(t, usage)
	assert.Equal(t, 15, usage.TotalTokens)
}

// 回归保护：正常输出内容携带 error 关键词不得被统一错误检测误报。
// 错误识别基于 JSON key（顶层 error 字段 / type:"error" 事件的平铺
// message+code / response.failed 的 status），文本值只落在 delta/text
// 等字段，不会触发检测。
func TestOaiResponsesStreamHandlerErrorKeywordInContentIsNotError(t *testing.T) {
	cases := []struct {
		name  string
		frame string
	}{
		{"delta contains error word", `{"type":"response.output_text.delta","delta":"Error handling is important. Fix the error and retry."}`},
		{"delta is literal 429 message", `{"type":"response.output_text.delta","delta":"Credit balance exhausted. Organization spend limit reached."}`},
		{"delta is literal error json string", `{"type":"response.output_text.delta","delta":"{\"error\":{\"message\":\"Credit balance exhausted\",\"code\":\"insufficient_quota\"}}"}`},
		{"refusal delta mentions error", `{"type":"response.refusal.delta","delta":"I cannot help with that error"}`},
		{"function arguments reference error codes", `{"type":"response.function_call_arguments.delta","delta":"{\"retry_on\":\"rate_limit_error\",\"code\":\"insufficient_quota\"}"}`},
		{"completed message item", `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"error text here"}]}}`},
		{"incomplete with usage", `{"type":"response.incomplete","response":{"id":"r1","status":"incomplete","usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`},
		{"annotation url contains error", `{"type":"response.output_text.annotation.added","annotation":{"type":"url_citation","url":"https://example.com/error-docs"}}`},
		{"reasoning summary mentions error", `{"type":"response.reasoning_summary_text.delta","delta":"thinking about the error message"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newStreamErrorTestContext(t, "/v1/responses")
			info := testRelayInfo("alias-model", "real-model")
			resp := newSSEStreamResponse(tc.frame)

			usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
			assert.Nil(t, apiErr, "normal content mentioning error must not be misdetected")
			require.NotNil(t, usage)
		})
	}
}

// 部分上游错误载荷只有 message 没有 type（OpenAI 规范中 type 可选），
// 旧判断 oaiError.Type != "" 会漏判并掉进空 usage 502 日志。
func TestOpenaiHandlerReturnsUpstreamErrorBodyWithoutType(t *testing.T) {
	c, _ := newStreamErrorTestContext(t, "/v1/chat/completions")
	info := testRelayInfo("alias-model", "real-model")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Model is overloaded","code":"503"}}`)),
	}

	usage, apiErr := OpenaiHandler(c, info, resp)
	require.NotNil(t, apiErr, "message-only error body must surface instead of falling into empty-usage 502 billing log")
	assert.Nil(t, usage)
	assert.Contains(t, apiErr.Error(), "Model is overloaded")
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
}

// 上游真实 HTTP 状态非 2xx 时不会被转换，状态码原样保留；被网关转成 200 时
// 优先从错误 code 还原真实状态码，无法还原才回退 502（保持可重试语义）。
func TestUpstreamErrorStatusCode(t *testing.T) {
	// 直接传 HTTP 状态码（数字 code 与字符串 code）
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusTooManyRequests, nil))
	assert.Equal(t, http.StatusInternalServerError, upstreamErrorStatusCode(http.StatusInternalServerError, nil))
	// HTTP 200 + 数字 code（Gemini 风格 {"error":{"code":429}}）
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: float64(429)}))
	// HTTP 200 + 字符串 code（OpenAI 风格 {"error":{"code":"429"}}）
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "429"}))
	// HTTP 200 + 无法还原的 code → 502 兜底
	assert.Equal(t, http.StatusBadGateway, upstreamErrorStatusCode(http.StatusOK, nil))
	assert.Equal(t, http.StatusBadGateway, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "some_unknown_code"}))
	// HTTP 200 + OpenAI 官方非数字计费 code（billing/quota 429、服务端 5xx）
	// 参考 https://developers.openai.com/api/docs/guides/error-codes
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "insufficient_quota"}))
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "credit_balance_exhausted"}))
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "organization_spend_limit_exceeded"}))
	assert.Equal(t, http.StatusTooManyRequests, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "rate_limit_exceeded"}))
	assert.Equal(t, http.StatusInternalServerError, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "server_error"}))
	assert.Equal(t, http.StatusServiceUnavailable, upstreamErrorStatusCode(http.StatusOK, &shared.OpenAIError{Code: "model_overloaded"}))
}
