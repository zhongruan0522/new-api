package minimax

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type MiniMaxTTSRequest struct {
	Model           string  `json:"model"`
	Text            string  `json:"text"`
	VoiceID         string  `json:"voice_id"`
	Speed           float64 `json:"speed,omitempty"`
	Vol             float64 `json:"vol,omitempty"`
	Pitch           int     `json:"pitch,omitempty"`
	AudioSampleRate int     `json:"audio_sample_rate,omitempty"`
	OutputFormat    string  `json:"output_format,omitempty"`
}

type MiniMaxTTSResponse struct {
	Created int              `json:"created"`
	Data    []MiniMaxTTSData `json:"data"`
	ID      string           `json:"id"`
	Model   string           `json:"model"`
	Object  string           `json:"object"`
	Usage   MiniMaxTTSUsage  `json:"usage"`
}

type MiniMaxTTSData struct {
	Index        int    `json:"index"`
	Audio        string `json:"audio"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type MiniMaxTTSUsage struct {
	TotalTokens int `json:"total_tokens"`
}

type MiniMaxTTSErrorResponse struct {
	Error MiniMaxTTSError `json:"error"`
}

type MiniMaxTTSError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// OpenAI voice to MiniMax voice_id mapping
var openAIToMiniMaxVoiceMap = map[string]string{
	"alloy":   "male-qn-qingse",
	"echo":    "male-qn-jingying",
	"fable":   "female-shaonv",
	"onyx":    "male-qn-badao",
	"nova":    "female-shaonv-jingpin",
	"shimmer": "female-yujie",
	// Add some standard MiniMax voice IDs
	"voice-1": "male-qn-qingse",
	"voice-2": "female-shaonv",
}

// OpenAI response format to MiniMax output format mapping
var responseFormatToOutputFormatMap = map[string]string{
	"mp3":  "mp3",
	"opus": "mp3",
	"aac":  "aac",
	"flac": "flac",
	"wav":  "wav",
	"pcm":  "pcm",
}

// TTS model mapping - MiniMax uses speech-01 or speech-01-turbo
var modelToTTSModelMap = map[string]string{
	"speech-01":       "speech-01",
	"speech-01-turbo": "speech-01-turbo",
	"tts-1":           "speech-01-turbo",
	"tts-1-hd":        "speech-01",
}

func mapVoiceType(openAIVoice string) string {
	if voice, ok := openAIToMiniMaxVoiceMap[openAIVoice]; ok {
		return voice
	}
	return "female-shaonv" // default voice
}

func mapOutputFormat(responseFormat string) string {
	if format, ok := responseFormatToOutputFormatMap[responseFormat]; ok {
		return format
	}
	return "mp3" // default format
}

func getTTSModel(modelName string) string {
	if ttsModel, ok := modelToTTSModelMap[modelName]; ok {
		return ttsModel
	}
	return "speech-01-turbo" // default model
}

func getContentTypeByFormat(format string) string {
	contentTypeMap := map[string]string{
		"mp3":  "audio/mpeg",
		"wav":  "audio/wav",
		"flac": "audio/flac",
		"aac":  "audio/aac",
		"pcm":  "audio/pcm",
	}
	if ct, ok := contentTypeMap[format]; ok {
		return ct
	}
	return "audio/mpeg" // default to mp3
}

func handleTTSResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(
			errors.New("failed to read minimax response"),
			types.ErrorCodeReadResponseBodyFailed,
			http.StatusInternalServerError,
		)
	}
	defer resp.Body.Close()

	// First try to parse as error response
	var errorResp MiniMaxTTSErrorResponse
	if unmarshalErr := json.Unmarshal(body, &errorResp); unmarshalErr == nil && errorResp.Error.Code != "" {
		return nil, types.NewErrorWithStatusCode(
			errors.New(errorResp.Error.Message),
			types.ErrorCodeBadResponse,
			http.StatusBadRequest,
		)
	}

	// Parse as successful response
	var minimaxResp MiniMaxTTSResponse
	if unmarshalErr := json.Unmarshal(body, &minimaxResp); unmarshalErr != nil {
		return nil, types.NewErrorWithStatusCode(
			errors.New("failed to parse minimax response"),
			types.ErrorCodeBadResponseBody,
			http.StatusInternalServerError,
		)
	}

	// Check if we have audio data
	if len(minimaxResp.Data) == 0 || minimaxResp.Data[0].Audio == "" {
		return nil, types.NewErrorWithStatusCode(
			errors.New("no audio data in response"),
			types.ErrorCodeBadResponse,
			http.StatusBadRequest,
		)
	}

	// Decode base64 audio data
	audioData, decodeErr := base64.StdEncoding.DecodeString(minimaxResp.Data[0].Audio)
	if decodeErr != nil {
		return nil, types.NewErrorWithStatusCode(
			errors.New("failed to decode audio data"),
			types.ErrorCodeBadResponseBody,
			http.StatusInternalServerError,
		)
	}

	// Get output format from context or default to mp3
	outputFormat := c.GetString("response_format")
	if outputFormat == "" {
		outputFormat = "mp3"
	}

	contentType := getContentTypeByFormat(outputFormat)
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, audioData)

	usage = &dto.Usage{
		PromptTokens:     info.PromptTokens,
		CompletionTokens: 0,
		TotalTokens:      minimaxResp.Usage.TotalTokens,
	}

	return usage, nil
}

func handleChatCompletionResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewErrorWithStatusCode(
			errors.New("failed to read minimax response"),
			types.ErrorCodeReadResponseBodyFailed,
			http.StatusInternalServerError,
		)
	}
	defer resp.Body.Close()

	// Set response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Data(resp.StatusCode, "application/json", body)
	return nil, nil
}
