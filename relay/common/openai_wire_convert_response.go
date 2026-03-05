package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	openAIResponsesOutputTypeMessage      = "message"
	openAIResponsesOutputTypeFunctionCall = "function_call"

	openAIResponsesOutputContentTypeText = "output_text"
)

func ConvertResponsesResponseToChatCompletionResponse(responsesResp *dto.OpenAIResponsesResponse) (*dto.OpenAITextResponse, error) {
	if responsesResp == nil {
		return nil, fmt.Errorf("responses response is nil")
	}

	content, toolCalls, err := extractChatMessageFromResponsesOutput(responsesResp.Output)
	if err != nil {
		return nil, err
	}

	finishReason := mapResponsesStatusToChatFinishReason(responsesResp.Status, len(toolCalls) > 0)
	assistantMsg, err := buildChatAssistantMessage(content, toolCalls)
	if err != nil {
		return nil, err
	}

	out := &dto.OpenAITextResponse{
		Id:      responsesResp.ID,
		Object:  "chat.completion",
		Model:   responsesResp.Model,
		Created: coerceCreatedAtFromResponses(responsesResp.CreatedAt),
		Choices: []dto.OpenAITextResponseChoice{{Index: 0, Message: assistantMsg, FinishReason: finishReason}},
	}
	applyResponsesUsageToChat(out, responsesResp.Usage)
	return out, nil
}

func ConvertChatCompletionResponseToResponsesResponse(chatResp *dto.OpenAITextResponse) (*dto.OpenAIResponsesResponse, error) {
	if chatResp == nil {
		return nil, fmt.Errorf("chat completion response is nil")
	}
	choice, err := getSingleChatChoice(chatResp.Choices)
	if err != nil {
		return nil, err
	}

	assistantText, err := extractChatMessageTextOnly(choice.Message)
	if err != nil {
		return nil, err
	}
	output, err := buildResponsesOutputFromChat(assistantText, choice.Message.ToolCalls)
	if err != nil {
		return nil, err
	}

	out := &dto.OpenAIResponsesResponse{
		ID:        chatResp.Id,
		Object:    "response",
		CreatedAt: coerceCreatedAtFromChat(chatResp.Created),
		Status:    mapChatFinishReasonToResponsesStatus(choice.FinishReason),
		Model:     chatResp.Model,
		Output:    output,
		Usage:     mapChatUsageToResponses(chatResp.Usage),
	}
	return out, nil
}

func getSingleChatChoice(choices []dto.OpenAITextResponseChoice) (dto.OpenAITextResponseChoice, error) {
	if len(choices) == 0 {
		return dto.OpenAITextResponseChoice{}, fmt.Errorf("chat completion response has no choices")
	}
	if len(choices) > 1 {
		return dto.OpenAITextResponseChoice{}, fmt.Errorf("responses api conversion does not support multiple choices: %d", len(choices))
	}
	return choices[0], nil
}

func extractChatMessageFromResponsesOutput(output []dto.ResponsesOutput) (content string, toolCalls []dto.ToolCallResponse, err error) {
	var builder strings.Builder
	var calls []dto.ToolCallResponse
	for _, item := range output {
		itemType := strings.TrimSpace(item.Type)
		switch itemType {
		case openAIResponsesOutputTypeMessage:
			for _, part := range item.Content {
				if strings.TrimSpace(part.Type) != openAIResponsesOutputContentTypeText {
					return "", nil, fmt.Errorf("unsupported responses message content type: %q", part.Type)
				}
				builder.WriteString(part.Text)
			}
		case openAIResponsesOutputTypeFunctionCall:
			callID := strings.TrimSpace(item.CallId)
			if callID == "" {
				callID = strings.TrimSpace(item.ID)
			}
			if callID == "" {
				callID = fmt.Sprintf("call_%d", len(calls))
			}
			calls = append(calls, dto.ToolCallResponse{
				ID:   callID,
				Type: "function",
				Function: dto.FunctionResponse{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		default:
			return "", nil, fmt.Errorf("unsupported responses output item type: %q", itemType)
		}
	}
	return builder.String(), calls, nil
}

func mapResponsesStatusToChatFinishReason(status string, sawToolCalls bool) string {
	if strings.EqualFold(strings.TrimSpace(status), "incomplete") {
		return "length"
	}
	if sawToolCalls {
		return "tool_calls"
	}
	return "stop"
}

func buildChatAssistantMessage(content string, toolCalls []dto.ToolCallResponse) (dto.Message, error) {
	msg := dto.Message{Role: "assistant", Content: content}
	if len(toolCalls) == 0 {
		return msg, nil
	}
	raw, err := common.Marshal(toolCalls)
	if err != nil {
		return dto.Message{}, fmt.Errorf("marshal tool_calls failed: %w", err)
	}
	msg.ToolCalls = raw
	return msg, nil
}

func applyResponsesUsageToChat(out *dto.OpenAITextResponse, usage *dto.Usage) {
	if out == nil || usage == nil {
		return
	}
	out.Usage.PromptTokens = usage.InputTokens
	out.Usage.CompletionTokens = usage.OutputTokens
	out.Usage.TotalTokens = usage.TotalTokens
	if out.Usage.TotalTokens == 0 {
		out.Usage.TotalTokens = out.Usage.PromptTokens + out.Usage.CompletionTokens
	}
	if usage.InputTokensDetails != nil {
		out.Usage.PromptTokensDetails.CachedTokens = usage.InputTokensDetails.CachedTokens
	}
}

func coerceCreatedAtFromResponses(createdAt int) any {
	if createdAt != 0 {
		return createdAt
	}
	return time.Now().Unix()
}

func extractChatMessageTextOnly(msg dto.Message) (string, error) {
	if msg.Content == nil {
		return "", nil
	}
	if msg.IsStringContent() {
		return msg.StringContent(), nil
	}

	parts := msg.ParseContent()
	var builder strings.Builder
	for _, part := range parts {
		if strings.TrimSpace(part.Type) != dto.ContentTypeText {
			return "", fmt.Errorf("chat response content only supports %q, got %q", dto.ContentTypeText, part.Type)
		}
		builder.WriteString(part.Text)
	}
	return strings.TrimSpace(builder.String()), nil
}

func buildResponsesOutputFromChat(text string, rawToolCalls json.RawMessage) ([]dto.ResponsesOutput, error) {
	output := make([]dto.ResponsesOutput, 0, 1)
	if strings.TrimSpace(text) != "" {
		output = append(output, dto.ResponsesOutput{
			Type:   openAIResponsesOutputTypeMessage,
			ID:     "msg_0",
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{Type: openAIResponsesOutputContentTypeText, Text: text},
			},
		})
	}

	toolOutputs, err := convertChatToolCallsToResponsesOutput(rawToolCalls)
	if err != nil {
		return nil, err
	}
	return append(output, toolOutputs...), nil
}

func mapChatFinishReasonToResponsesStatus(finishReason string) string {
	if strings.EqualFold(strings.TrimSpace(finishReason), "length") {
		return "incomplete"
	}
	return "completed"
}

func coerceCreatedAtFromChat(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return int(time.Now().Unix())
	}
}

func mapChatUsageToResponses(u dto.Usage) *dto.Usage {
	usage := &dto.Usage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
		InputTokensDetails: &dto.InputTokenDetails{
			CachedTokens: u.PromptTokensDetails.CachedTokens,
		},
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	return usage
}

func convertChatToolCallsToResponsesOutput(raw json.RawMessage) ([]dto.ResponsesOutput, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var calls []dto.ToolCallResponse
	if err := common.Unmarshal(raw, &calls); err != nil {
		return nil, fmt.Errorf("unmarshal tool_calls failed: %w", err)
	}
	if len(calls) == 0 {
		return nil, nil
	}

	out := make([]dto.ResponsesOutput, 0, len(calls))
	for i, call := range calls {
		callID := strings.TrimSpace(call.ID)
		if callID == "" {
			callID = fmt.Sprintf("call_%d", i)
		}
		if strings.TrimSpace(call.Function.Name) == "" {
			return nil, fmt.Errorf("tool_calls[%d].function.name is required", i)
		}
		out = append(out, dto.ResponsesOutput{
			Type:      openAIResponsesOutputTypeFunctionCall,
			ID:        callID,
			Status:    "completed",
			Role:      "assistant",
			CallId:    callID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}

	return out, nil
}
