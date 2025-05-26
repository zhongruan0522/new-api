package dto

import "encoding/json"

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	AvgLogprobs  float64       `json:"avgLogprobs"`
}

type GeminiTokenDetails struct {
	Modality   string `json:"modality"`
	TokenCount int    `json:"tokenCount"`
}

type GeminiUsageMetadata struct {
	PromptTokenCount        int                  `json:"promptTokenCount"`
	CandidatesTokenCount    int                  `json:"candidatesTokenCount"`
	TotalTokenCount         int                  `json:"totalTokenCount"`
	PromptTokensDetails     []GeminiTokenDetails `json:"promptTokensDetails"`
	CandidatesTokensDetails []GeminiTokenDetails `json:"candidatesTokensDetails"`
}

type GeminiTextGenerationResponse struct {
	Candidates    []GeminiCandidate   `json:"candidates"`
	UsageMetadata GeminiUsageMetadata `json:"usageMetadata"`
	ModelVersion  string              `json:"modelVersion"`
	ResponseID    string              `json:"responseId"`
}

type GeminiGenerationConfig struct {
	StopSequences              []string         `json:"stopSequences,omitempty"`
	ResponseMimeType           string           `json:"responseMimeType,omitempty"`
	ResponseSchema             *json.RawMessage `json:"responseSchema,omitempty"`
	ResponseModalities         *json.RawMessage `json:"responseModalities,omitempty"`
	CandidateCount             int              `json:"candidateCount,omitempty"`
	MaxOutputTokens            int              `json:"maxOutputTokens,omitempty"`
	Temperature                float64          `json:"temperature,omitempty"`
	TopP                       float64          `json:"topP,omitempty"`
	TopK                       int              `json:"topK,omitempty"`
	Seed                       int              `json:"seed,omitempty"`
	PresencePenalty            float64          `json:"presencePenalty,omitempty"`
	FrequencyPenalty           float64          `json:"frequencyPenalty,omitempty"`
	ResponseLogprobs           bool             `json:"responseLogprobs,omitempty"`
	LogProbs                   int              `json:"logProbs,omitempty"`
	EnableEnhancedCivicAnswers bool             `json:"enableEnhancedCivicAnswers,omitempty"`
	SpeechConfig               *json.RawMessage `json:"speechConfig,omitempty"`
	ThinkingConfig             *json.RawMessage `json:"thinkingConfig,omitempty"`
	MediaResolution            *json.RawMessage `json:"mediaResolution,omitempty"`
}

type GeminiTextGenerationRequest struct {
	Contents          []GeminiContent        `json:"contents"`
	Tools             *json.RawMessage       `json:"tools,omitempty"`
	ToolConfig        *json.RawMessage       `json:"toolConfig,omitempty"`
	SafetySettings    *json.RawMessage       `json:"safetySettings,omitempty"`
	SystemInstruction *json.RawMessage       `json:"systemInstruction,omitempty"`
	GenerationConfig  GeminiGenerationConfig `json:"generationConfig,omitempty"`
	CachedContent     *json.RawMessage       `json:"cachedContent,omitempty"`
}
