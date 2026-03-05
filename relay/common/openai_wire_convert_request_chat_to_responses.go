package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
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
	return &dto.OpenAIResponsesRequest{
		Model:           chatReq.Model,
		Input:           input,
		MaxOutputTokens: chatReq.GetMaxTokens(),
		Stream:          chatReq.Stream,
		Temperature:     chatReq.Temperature,
		User:            chatReq.User,
		Metadata:        chatReq.Metadata,
		Store:           chatReq.Store,
	}
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
	if chatReq.ResponseFormat == nil || strings.TrimSpace(chatReq.ResponseFormat.Type) == "" {
		return nil
	}
	raw, err := common.Marshal(map[string]any{"format": chatReq.ResponseFormat})
	if err != nil {
		return fmt.Errorf("marshal text.format failed: %w", err)
	}
	out.Text = raw
	return nil
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
