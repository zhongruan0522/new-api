package minimax

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/dto"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/types"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
)

var minimaxNamePattern = regexp.MustCompile(`(?i)minimax`)

type MiniMaxTTSRequest struct {
	Model             string             `json:"model"`
	Text              string             `json:"text"`
	Stream            bool               `json:"stream,omitempty"`
	StreamOptions     *StreamOptions     `json:"stream_options,omitempty"`
	VoiceSetting      VoiceSetting       `json:"voice_setting"`
	PronunciationDict *PronunciationDict `json:"pronunciation_dict,omitempty"`
	AudioSetting      *AudioSetting      `json:"audio_setting,omitempty"`
	TimbreWeights     []TimbreWeight     `json:"timbre_weights,omitempty"`
	LanguageBoost     string             `json:"language_boost,omitempty"`
	VoiceModify       *VoiceModify       `json:"voice_modify,omitempty"`
	SubtitleEnable    bool               `json:"subtitle_enable,omitempty"`
	OutputFormat      string             `json:"output_format,omitempty"`
	AigcWatermark     bool               `json:"aigc_watermark,omitempty"`
}

type StreamOptions struct {
	ExcludeAggregatedAudio bool `json:"exclude_aggregated_audio,omitempty"`
}

type VoiceSetting struct {
	VoiceID           string  `json:"voice_id"`
	Speed             float64 `json:"speed,omitempty"`
	Vol               float64 `json:"vol,omitempty"`
	Pitch             int     `json:"pitch,omitempty"`
	Emotion           string  `json:"emotion,omitempty"`
	TextNormalization bool    `json:"text_normalization,omitempty"`
	LatexRead         bool    `json:"latex_read,omitempty"`
}

type PronunciationDict struct {
	Tone []string `json:"tone,omitempty"`
}

type AudioSetting struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
	Channel    int    `json:"channel,omitempty"`
	ForceCbr   bool   `json:"force_cbr,omitempty"`
}

type TimbreWeight struct {
	VoiceID string `json:"voice_id"`
	Weight  int    `json:"weight"`
}

type VoiceModify struct {
	Pitch        int    `json:"pitch,omitempty"`
	Intensity    int    `json:"intensity,omitempty"`
	Timbre       int    `json:"timbre,omitempty"`
	SoundEffects string `json:"sound_effects,omitempty"`
}

type MiniMaxTTSResponse struct {
	Data      MiniMaxTTSData   `json:"data"`
	ExtraInfo MiniMaxExtraInfo `json:"extra_info"`
	TraceID   string           `json:"trace_id"`
	BaseResp  MiniMaxBaseResp  `json:"base_resp"`
}

type MiniMaxTTSData struct {
	Audio  string `json:"audio"`
	Status int    `json:"status"`
}

type MiniMaxExtraInfo struct {
	UsageCharacters int64 `json:"usage_characters"`
}

type MiniMaxBaseResp struct {
	StatusCode int64  `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func getContentTypeByFormat(format string) string {
	contentTypeMap := map[string]string{
		"mp3":  "audio/mpeg",
		"wav":  "audio/wav",
		"flac": "audio/flac",
		"aac":  "audio/aac",
		"opus": "audio/ogg",
		"pcm":  "audio/pcm",
	}
	if ct, ok := contentTypeMap[format]; ok {
		return ct
	}
	return "audio/mpeg" // default to mp3
}

func sanitizeTTSProviderName(message string, info *relaycommon.RelayInfo) string {
	if message == "" || info == nil {
		return minimaxNamePattern.ReplaceAllString(message, "upstream")
	}
	if strings.Contains(strings.ToLower(info.OriginModelName), "minimax") {
		return message
	}
	if info.ChannelMeta != nil &&
		strings.Contains(strings.ToLower(info.UpstreamModelName), "minimax") {
		return message
	}
	return minimaxNamePattern.ReplaceAllString(message, "upstream")
}

func handleTTSResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NookMuxError) {
	defer resp.Body.Close()
	body, readErr := common.ReadMediaResponseBody(resp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("failed to read upstream response: %w", readErr),
			types.ErrorCodeReadResponseBodyFailed,
			http.StatusInternalServerError,
		)
	}

	// Parse response
	var minimaxResp MiniMaxTTSResponse
	if unmarshalErr := jsonx.Unmarshal(body, &minimaxResp); unmarshalErr != nil {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("failed to parse TTS response: %w", unmarshalErr),
			types.ErrorCodeBadResponseBody,
			http.StatusInternalServerError,
		)
	}

	// Check base_resp status code
	if minimaxResp.BaseResp.StatusCode != 0 {
		statusMsg := sanitizeTTSProviderName(minimaxResp.BaseResp.StatusMsg, info)
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("TTS upstream error: %d - %s", minimaxResp.BaseResp.StatusCode, statusMsg),
			types.ErrorCodeBadResponse,
			http.StatusBadRequest,
		)
	}

	// Check if we have audio data
	if minimaxResp.Data.Audio == "" {
		return nil, types.NewErrorWithStatusCode(
			fmt.Errorf("no audio data in TTS response"),
			types.ErrorCodeBadResponse,
			http.StatusBadRequest,
		)
	}

	if strings.HasPrefix(minimaxResp.Data.Audio, "http") {
		c.Redirect(http.StatusFound, minimaxResp.Data.Audio)
	} else {
		// Handle hex-encoded audio data
		audioData, decodeErr := hex.DecodeString(minimaxResp.Data.Audio)
		if decodeErr != nil {
			return nil, types.NewErrorWithStatusCode(
				fmt.Errorf("failed to decode hex audio data: %w", decodeErr),
				types.ErrorCodeBadResponse,
				http.StatusInternalServerError,
			)
		}

		audioFormat := c.GetString("minimax_audio_format")
		if audioFormat == "" {
			if audioReq, ok := info.Request.(*dto.AudioRequest); ok {
				if normalizedFormat, formatErr := normalizeMiniMaxTTSAudioFormat(audioReq.ResponseFormat); formatErr == nil {
					audioFormat = normalizedFormat
				}
			}
		}
		contentType := getContentTypeByFormat(audioFormat)

		c.Data(http.StatusOK, contentType, audioData)
	}

	// MiniMax TTS 按 usage_characters（合成语音消耗的字符数）计费。
	// 按产品需求：usage_characters 同时映射到【输入 Token】和【音频输出 Token】，
	// 触发 audio_handler.go:70 的音频倍率分支 (PostAudioConsumeQuota)，
	// 让 calculateAudioQuota 同时算输入文本成本和音频输出成本。
	usageCharacters := int(minimaxResp.ExtraInfo.UsageCharacters)
	usage = &dto.Usage{
		PromptTokens:     usageCharacters,
		CompletionTokens: usageCharacters,
		TotalTokens:      usageCharacters * 2,
	}
	usage.(*dto.Usage).PromptTokensDetails.TextTokens = usageCharacters
	usage.(*dto.Usage).CompletionTokenDetails.AudioTokens = usageCharacters

	return usage, nil
}
