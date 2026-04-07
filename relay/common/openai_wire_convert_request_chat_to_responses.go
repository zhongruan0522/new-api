package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

func ConvertChatCompletionsRequestToResponsesRequest(chatReq *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error) {
	if chatReq == nil {
		return nil, fmt.Errorf("chat request is nil")
	}
	if chatReq.N > 1 {
		return nil, fmt.Errorf("responses api does not support n=%d", chatReq.N)
	}

	instructions, remaining, err := splitChatInstructions(chatReq.Messages)
	if err != nil {
		return nil, err
	}
	input, err := buildResponsesInputFromChatMessages(remaining)
	if err != nil {
		return nil, err
	}

	out := newResponsesRequestFromChat(chatReq, input)
	if err := applyChatToResponsesInstructions(out, instructions); err != nil {
		return nil, err
	}
	if err := applyChatToResponsesTools(out, chatReq); err != nil {
		return nil, err
	}
	applyChatToResponsesSampling(out, chatReq)
	if err := applyChatToResponsesTextFormat(out, chatReq); err != nil {
		return nil, err
	}
	return out, nil
}

func newResponsesRequestFromChat(chatReq *dto.GeneralOpenAIRequest, input json.RawMessage) *dto.OpenAIResponsesRequest {
	out := &dto.OpenAIResponsesRequest{
		Model:            chatReq.Model,
		Input:            input,
		MaxOutputTokens:  chatReq.GetMaxTokens(),
		Stream:           chatReq.Stream,
		Temperature:      chatReq.Temperature,
		TopLogprobs:      chatReq.TopLogProbs,
		SafetyIdentifier: chatReq.SafetyIdentifier,
		ServiceTier:      "",
		User:             chatReq.User,
		Metadata:         chatReq.Metadata,
		Store:            chatReq.Store,
	}
	if strings.TrimSpace(chatReq.PromptCacheKey) != "" {
		if raw, err := common.Marshal(chatReq.PromptCacheKey); err == nil {
			out.PromptCacheKey = raw
		}
	}
	if len(chatReq.PromptCacheRetention) > 0 {
		out.PromptCacheRetention = chatReq.PromptCacheRetention
	}
	return out
}

func applyChatToResponsesInstructions(out *dto.OpenAIResponsesRequest, instructions string) error {
	if strings.TrimSpace(instructions) == "" {
		return nil
	}
	raw, err := common.Marshal(instructions)
	if err != nil {
		return fmt.Errorf("marshal instructions failed: %w", err)
	}
	out.Instructions = raw
	return nil
}

func applyChatToResponsesTools(out *dto.OpenAIResponsesRequest, chatReq *dto.GeneralOpenAIRequest) error {
	if chatReq.ToolChoice != nil {
		raw, err := convertChatToolChoiceToResponsesRaw(chatReq.ToolChoice)
		if err != nil {
			return err
		}
		out.ToolChoice = raw
	}

	if len(chatReq.Tools) > 0 {
		raw, err := convertChatToolsToResponsesRaw(chatReq.Tools)
		if err != nil {
			return err
		}
		out.Tools = raw
	}

	if chatReq.ParallelTooCalls != nil {
		raw, err := common.Marshal(*chatReq.ParallelTooCalls)
		if err != nil {
			return fmt.Errorf("marshal parallel_tool_calls failed: %w", err)
		}
		out.ParallelToolCalls = raw
	}

	return nil
}

func applyChatToResponsesSampling(out *dto.OpenAIResponsesRequest, chatReq *dto.GeneralOpenAIRequest) {
	if chatReq.TopP > 0 {
		topP := chatReq.TopP
		out.TopP = &topP
	}
	if strings.TrimSpace(chatReq.ReasoningEffort) != "" {
		out.Reasoning = &dto.Reasoning{Effort: chatReq.ReasoningEffort}
	}
}

func applyChatToResponsesTextFormat(out *dto.OpenAIResponsesRequest, chatReq *dto.GeneralOpenAIRequest) error {
	payload := make(map[string]any)
	if chatReq.ResponseFormat != nil && strings.TrimSpace(chatReq.ResponseFormat.Type) != "" {
		format, err := buildResponsesTextFormatFromChat(chatReq.ResponseFormat)
		if err != nil {
			return err
		}
		payload["format"] = format
	}
	if len(chatReq.Verbosity) > 0 {
		var verbosity any
		if err := common.Unmarshal(chatReq.Verbosity, &verbosity); err != nil {
			return fmt.Errorf("unmarshal verbosity failed: %w", err)
		}
		payload["verbosity"] = verbosity
	}
	if len(payload) == 0 {
		return nil
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal text.format failed: %w", err)
	}
	out.Text = raw
	return nil
}

// buildResponsesTextFormatFromChat flattens Chat json_schema into Responses text.format.
func buildResponsesTextFormatFromChat(format *dto.ResponseFormat) (map[string]any, error) {
	if format == nil || strings.TrimSpace(format.Type) == "" {
		return nil, nil
	}
	out := map[string]any{"type": format.Type}
	if !strings.EqualFold(strings.TrimSpace(format.Type), "json_schema") {
		return out, nil
	}

	var schema dto.FormatJsonSchema
	if len(format.JsonSchema) == 0 {
		return nil, fmt.Errorf("response_format.json_schema is required when type=json_schema")
	}
	if err := common.Unmarshal(format.JsonSchema, &schema); err != nil {
		return nil, fmt.Errorf("unmarshal response_format.json_schema failed: %w", err)
	}
	if strings.TrimSpace(schema.Name) == "" {
		return nil, fmt.Errorf("response_format.json_schema.name is required")
	}
	out["name"] = schema.Name
	if strings.TrimSpace(schema.Description) != "" {
		out["description"] = schema.Description
	}
	if schema.Schema != nil {
		out["schema"] = schema.Schema
	}
	if len(schema.Strict) > 0 {
		var strict any
		if err := common.Unmarshal(schema.Strict, &strict); err != nil {
			return nil, fmt.Errorf("unmarshal response_format.json_schema.strict failed: %w", err)
		}
		out["strict"] = strict
	}
	return out, nil
}

func splitChatInstructions(messages []dto.Message) (instructions string, remaining []dto.Message, err error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	var parts []string
	out := make([]dto.Message, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(strings.ToLower(msg.Role))
		if role != "system" && role != "developer" {
			out = append(out, msg)
			continue
		}
		text, err := chatMessageTextOnly(msg)
		if err != nil {
			return "", nil, err
		}
		if strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "\n"), out, nil
}

func chatMessageTextOnly(message dto.Message) (string, error) {
	if message.Content == nil {
		return "", nil
	}
	if message.IsStringContent() {
		return message.StringContent(), nil
	}

	content := message.ParseContent()
	if len(content) == 0 {
		return "", nil
	}

	var builder strings.Builder
	for _, part := range content {
		if part.Type == dto.ContentTypeText {
			builder.WriteString(part.Text)
			continue
		}
		return "", fmt.Errorf("instructions only supports text content, got %q", part.Type)
	}
	return builder.String(), nil
}
