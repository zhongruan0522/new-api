package minimax

import (
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NookMux/NookMux/dto"
	"github.com/NookMux/NookMux/pkg/jsonx"
	relaycommon "github.com/NookMux/NookMux/relay/common"
	"github.com/NookMux/NookMux/relay/constant"
	"github.com/NookMux/NookMux/types"
	"github.com/gin-gonic/gin"

	"time"
)

// TestHandleTTSResponse_BillingFields 验证 MiniMax TTS 的 usage 字段构造：
// 按产品需求，usage_characters 同时映射到【输入 Token】和【音频输出 Token】。
// 回归保护：
//   - 修复前 AudioTokens 始终为 0，永远走文本倍率分支，audio_ratio 配置失效
//   - 必须同时填充 PromptTokensDetails.TextTokens 和 CompletionTokenDetails.AudioTokens
func TestHandleTTSResponse_BillingFields(t *testing.T) {
	// 构造一个非空 hex 音频响应
	audioBytes := []byte{0x01, 0x02, 0x03, 0x04}
	hexAudio := hex.EncodeToString(audioBytes)

	respBody := MiniMaxTTSResponse{
		Data:      MiniMaxTTSData{Audio: hexAudio, Status: 2},
		ExtraInfo: MiniMaxExtraInfo{UsageCharacters: 42},
		BaseResp:  MiniMaxBaseResp{StatusCode: 0},
	}
	body, _ := jsonx.Marshal(respBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(""))

	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeAudioSpeech,
		StartTime: time.Now(),
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Header:     make(http.Header),
	}

	usageAny, apiErr := handleTTSResponse(c, resp, info)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}

	usage, ok := usageAny.(*dto.Usage)
	if !ok {
		t.Fatalf("expected *dto.Usage, got %T", usageAny)
	}

	// 按产品需求：usage_characters 同时映射到输入和音频输出
	if got := usage.PromptTokens; got != 42 {
		t.Errorf("PromptTokens = %d, want 42 (usage_characters)", got)
	}
	if got := usage.PromptTokensDetails.TextTokens; got != 42 {
		t.Errorf("PromptTokensDetails.TextTokens = %d, want 42 (usage_characters)", got)
	}
	if got := usage.CompletionTokens; got != 42 {
		t.Errorf("CompletionTokens = %d, want 42 (usage_characters)", got)
	}
	// 关键：AudioTokens 必须非 0，否则 audio_handler.go:70 不会走音频倍率分支
	if got := usage.CompletionTokenDetails.AudioTokens; got != 42 {
		t.Errorf("CompletionTokenDetails.AudioTokens = %d, want 42 (usage_characters)", got)
	}
	if got := usage.TotalTokens; got != 84 {
		t.Errorf("TotalTokens = %d, want 84 (usage_characters * 2)", got)
	}
}

// TestHandleTTSResponse_ZeroUsage 验证 usage_characters=0 时 TotalTokens=0，
// 让 PostAudioConsumeQuota 走 NewEmptyUsageRetryError 重试/退款路径，
// 不要把估算的 prompt 当作真实 usage 静默计费。
func TestHandleTTSResponse_ZeroUsage(t *testing.T) {
	audioBytes := []byte{0x01, 0x02}
	hexAudio := hex.EncodeToString(audioBytes)

	respBody := MiniMaxTTSResponse{
		Data:      MiniMaxTTSData{Audio: hexAudio, Status: 2},
		ExtraInfo: MiniMaxExtraInfo{UsageCharacters: 0},
		BaseResp:  MiniMaxBaseResp{StatusCode: 0},
	}
	body, _ := jsonx.Marshal(respBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(""))

	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeAudioSpeech,
		StartTime: time.Now(),
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Header:     make(http.Header),
	}

	usageAny, apiErr := handleTTSResponse(c, resp, info)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}

	usage, ok := usageAny.(*dto.Usage)
	if !ok {
		t.Fatalf("expected *dto.Usage, got %T", usageAny)
	}

	if usage.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0 (must trigger empty-usage retry path)", usage.TotalTokens)
	}
	if usage.CompletionTokenDetails.AudioTokens != 0 {
		t.Errorf("AudioTokens = %d, want 0", usage.CompletionTokenDetails.AudioTokens)
	}
}

func TestHandleTTSResponse_UsesRequestedContentType(t *testing.T) {
	audioBytes := []byte{0x52, 0x49, 0x46, 0x46}
	hexAudio := hex.EncodeToString(audioBytes)

	respBody := MiniMaxTTSResponse{
		Data:      MiniMaxTTSData{Audio: hexAudio, Status: 2},
		ExtraInfo: MiniMaxExtraInfo{UsageCharacters: 1},
		BaseResp:  MiniMaxBaseResp{StatusCode: 0},
	}
	body, _ := jsonx.Marshal(respBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(""))
	c.Set("minimax_audio_format", "wav")

	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeAudioSpeech,
		StartTime: time.Now(),
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Header:     make(http.Header),
	}

	_, apiErr := handleTTSResponse(c, resp, info)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if got := w.Header().Get("Content-Type"); got != "audio/wav" {
		t.Fatalf("Content-Type = %q, want audio/wav", got)
	}
	if got := w.Body.Bytes(); string(got) != string(audioBytes) {
		t.Fatalf("body = %v, want %v", got, audioBytes)
	}
}

// TestHandleTTSResponse_ErrorStatus 验证上游业务错误 (status_code != 0) 被正确暴露
func TestHandleTTSResponse_ErrorStatus(t *testing.T) {
	respBody := MiniMaxTTSResponse{
		BaseResp: MiniMaxBaseResp{StatusCode: 1001, StatusMsg: "invalid voice"},
	}
	body, _ := jsonx.Marshal(respBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(""))

	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeAudioSpeech,
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Header:     make(http.Header),
	}

	_, apiErr := handleTTSResponse(c, resp, info)
	if apiErr == nil {
		t.Fatal("expected error for non-zero base_resp.status_code, got nil")
	}
	if !strings.Contains(apiErr.Error(), "1001") {
		t.Errorf("error should contain status code 1001, got: %v", apiErr)
	}
	if strings.Contains(strings.ToLower(apiErr.Error()), "minimax") {
		t.Errorf("error should not expose provider name, got: %v", apiErr)
	}
}

func TestHandleTTSResponse_ErrorStatusSanitizesProviderName(t *testing.T) {
	respBody := MiniMaxTTSResponse{
		BaseResp: MiniMaxBaseResp{StatusCode: 1001, StatusMsg: "MiniMax invalid voice"},
	}
	body, _ := jsonx.Marshal(respBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(""))

	info := &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeAudioSpeech,
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Header:     make(http.Header),
	}

	_, apiErr := handleTTSResponse(c, resp, info)
	if apiErr == nil {
		t.Fatal("expected error for non-zero base_resp.status_code, got nil")
	}
	if strings.Contains(strings.ToLower(apiErr.Error()), "minimax") {
		t.Errorf("error should not expose provider name, got: %v", apiErr)
	}
	if !strings.Contains(apiErr.Error(), "upstream invalid voice") {
		t.Errorf("error should keep sanitized upstream message, got: %v", apiErr)
	}
}

// 编译期类型检查：确保 types 包仍被引用（避免 import 残留告警）
var _ = types.ErrorCodeBadResponse
