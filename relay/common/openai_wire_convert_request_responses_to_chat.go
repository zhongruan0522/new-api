package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

func ConvertResponsesRequestToChatCompletionsRequest(responsesReq *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	if responsesReq == nil {
		return nil, fmt.Errorf("responses request is nil")
	}
	if strings.TrimSpace(responsesReq.PreviousResponseID) != "" {
		return nil, fmt.Errorf("previous_response_id is not supported by chat.completions conversion")
	}

	systemRole := (&dto.GeneralOpenAIRequest{Model: responsesReq.Model}).GetSystemRoleName()
	systemMsg, err := buildChatSystemMessageFromInstructions(systemRole, responsesReq.Instructions)
	if err != nil {
		return nil, err
	}
	userMsgs, err := buildChatMessagesFromResponsesInput(responsesReq.Input)
	if err != nil {
		return nil, err
	}

	messages := make([]dto.Message, 0, len(userMsgs)+1)
	if systemMsg != nil {
		messages = append(messages, *systemMsg)
	}
	messages = append(messages, userMsgs...)

	out := newChatRequestFromResponses(responsesReq, messages)
	if err := applyResponsesToChatTools(out, responsesReq); err != nil {
		return nil, err
	}
	if err := applyResponsesToChatTextFormat(out, responsesReq.Text); err != nil {
		return nil, err
	}
	return out, nil
}

func newChatRequestFromResponses(responsesReq *dto.OpenAIResponsesRequest, messages []dto.Message) *dto.GeneralOpenAIRequest {
	out := &dto.GeneralOpenAIRequest{
		Model:            responsesReq.Model,
		Messages:         messages,
		Stream:           responsesReq.Stream,
		MaxTokens:        responsesReq.MaxOutputTokens,
		Temperature:      responsesReq.Temperature,
		TopLogProbs:      responsesReq.TopLogprobs,
		SafetyIdentifier: responsesReq.SafetyIdentifier,
		User:             responsesReq.User,
		Metadata:         responsesReq.Metadata,
		Store:            responsesReq.Store,
	}
	if responsesReq.TopP != nil {
		out.TopP = *responsesReq.TopP
	}
	if responsesReq.Reasoning != nil && strings.TrimSpace(responsesReq.Reasoning.Effort) != "" {
		out.ReasoningEffort = responsesReq.Reasoning.Effort
	}
	if len(responsesReq.PromptCacheKey) > 0 && common.GetJsonType(responsesReq.PromptCacheKey) == "string" {
		var promptCacheKey string
		if err := common.Unmarshal(responsesReq.PromptCacheKey, &promptCacheKey); err == nil {
			out.PromptCacheKey = promptCacheKey
		}
	}
	if len(responsesReq.PromptCacheRetention) > 0 {
		out.PromptCacheRetention = responsesReq.PromptCacheRetention
	}
	return out
}

func buildChatSystemMessageFromInstructions(systemRole string, raw json.RawMessage) (*dto.Message, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if common.GetJsonType(raw) != "string" {
		return nil, fmt.Errorf("instructions must be a string, got %s", common.GetJsonType(raw))
	}

	var s string
	if err := common.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("unmarshal instructions failed: %w", err)
	}
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	return &dto.Message{Role: systemRole, Content: s}, nil
}

func applyResponsesToChatTools(out *dto.GeneralOpenAIRequest, responsesReq *dto.OpenAIResponsesRequest) error {
	if len(responsesReq.ToolChoice) > 0 {
		toolChoice, err := convertResponsesToolChoiceToChatAny(responsesReq.ToolChoice)
		if err != nil {
			return err
		}
		out.ToolChoice = toolChoice
	}

	if len(responsesReq.Tools) > 0 {
		tools, err := convertResponsesToolsRawToChatTools(responsesReq.Tools)
		if err != nil {
			return err
		}
		out.Tools = tools
	}

	if len(responsesReq.ParallelToolCalls) > 0 {
		var enabled bool
		if err := common.Unmarshal(responsesReq.ParallelToolCalls, &enabled); err != nil {
			return fmt.Errorf("unmarshal parallel_tool_calls failed: %w", err)
		}
		out.ParallelTooCalls = &enabled
	}

	return nil
}

func applyResponsesToChatTextFormat(out *dto.GeneralOpenAIRequest, raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]json.RawMessage
	if err := common.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("unmarshal text failed: %w", err)
	}
	formatRaw, ok := payload["format"]
	if !ok || len(formatRaw) == 0 {
		if verbosityRaw, ok := payload["verbosity"]; ok && len(verbosityRaw) > 0 {
			out.Verbosity = verbosityRaw
		}
		return nil
	}
	format, err := buildChatResponseFormatFromResponses(formatRaw)
	if err != nil {
		return err
	}
	if format == nil || strings.TrimSpace(format.Type) == "" {
		if verbosityRaw, ok := payload["verbosity"]; ok && len(verbosityRaw) > 0 {
			out.Verbosity = verbosityRaw
		}
		return nil
	}
	out.ResponseFormat = format
	if verbosityRaw, ok := payload["verbosity"]; ok && len(verbosityRaw) > 0 {
		out.Verbosity = verbosityRaw
	}
	return nil
}

// buildChatResponseFormatFromResponses nests flattened Responses text.format back
// into Chat response_format for OpenAI-compatible upstreams.
func buildChatResponseFormatFromResponses(raw json.RawMessage) (*dto.ResponseFormat, error) {
	var formatMap map[string]json.RawMessage
	if err := common.Unmarshal(raw, &formatMap); err != nil {
		return nil, fmt.Errorf("unmarshal text.format failed: %w", err)
	}

	var formatType string
	if err := common.Unmarshal(formatMap["type"], &formatType); err != nil {
		return nil, fmt.Errorf("unmarshal text.format.type failed: %w", err)
	}
	formatType = strings.TrimSpace(formatType)
	if formatType == "" {
		return nil, nil
	}
	if !strings.EqualFold(formatType, "json_schema") {
		return &dto.ResponseFormat{Type: formatType}, nil
	}

	jsonSchema := dto.FormatJsonSchema{}
	if nameRaw, ok := formatMap["name"]; ok && len(nameRaw) > 0 {
		if err := common.Unmarshal(nameRaw, &jsonSchema.Name); err != nil {
			return nil, fmt.Errorf("unmarshal text.format.name failed: %w", err)
		}
	}
	if descRaw, ok := formatMap["description"]; ok && len(descRaw) > 0 {
		if err := common.Unmarshal(descRaw, &jsonSchema.Description); err != nil {
			return nil, fmt.Errorf("unmarshal text.format.description failed: %w", err)
		}
	}
	if schemaRaw, ok := formatMap["schema"]; ok && len(schemaRaw) > 0 {
		var schema any
		if err := common.Unmarshal(schemaRaw, &schema); err != nil {
			return nil, fmt.Errorf("unmarshal text.format.schema failed: %w", err)
		}
		jsonSchema.Schema = schema
	}
	if strictRaw, ok := formatMap["strict"]; ok && len(strictRaw) > 0 {
		jsonSchema.Strict = strictRaw
	}
	jsonSchemaRaw, err := common.Marshal(jsonSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal response_format.json_schema failed: %w", err)
	}
	return &dto.ResponseFormat{Type: "json_schema", JsonSchema: jsonSchemaRaw}, nil
}
