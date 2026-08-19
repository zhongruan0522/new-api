package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

const (
	openAIResponsesOutputTypeMessage        = "message"
	openAIResponsesOutputTypeReasoning      = "reasoning"
	openAIResponsesOutputTypeFunctionCall   = "function_call"
	openAIResponsesOutputTypeCustomToolCall = "custom_tool_call"
	openAIResponsesOutputTypeToolSearchCall = "tool_search_call"

	openAIResponsesOutputContentTypeText = "output_text"
)

func ConvertResponsesResponseToChatCompletionResponse(responsesResp *shared.OpenAIResponsesResponse) (*shared.OpenAITextResponse, error) {
	if responsesResp == nil {
		return nil, fmt.Errorf("responses response is nil")
	}

	content, reasoning, toolCalls, err := extractChatMessageFromResponsesOutput(responsesResp.Output)
	if err != nil {
		return nil, err
	}

	finishReason := mapResponsesStatusToChatFinishReason(responsesResp.Status, len(toolCalls) > 0)
	assistantMsg, err := buildChatAssistantMessage(content, reasoning, toolCalls)
	if err != nil {
		return nil, err
	}

	out := &shared.OpenAITextResponse{
		Id:      responsesResp.ID,
		Object:  "chat.completion",
		Model:   responsesResp.Model,
		Created: coerceCreatedAtFromResponses(responsesResp.CreatedAt),
		Choices: []shared.OpenAITextResponseChoice{{Index: 0, Message: assistantMsg, FinishReason: finishReason}},
	}
	applyResponsesUsageToChat(out, responsesResp.Usage)
	return out, nil
}

func ConvertChatCompletionResponseToResponsesResponse(chatResp *shared.OpenAITextResponse) (*shared.OpenAIResponsesResponse, error) {
	return ConvertChatCompletionResponseToResponsesResponseWithToolContext(chatResp, nil)
}

func ConvertChatCompletionResponseToResponsesResponseWithToolContext(chatResp *shared.OpenAITextResponse, toolContext *OpenAIWireToolContext) (*shared.OpenAIResponsesResponse, error) {
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
	output, err := buildResponsesOutputFromChat(choice.Message, assistantText, choice.Message.ToolCalls, toolContext)
	if err != nil {
		return nil, err
	}

	out := &shared.OpenAIResponsesResponse{
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

func getSingleChatChoice(choices []shared.OpenAITextResponseChoice) (shared.OpenAITextResponseChoice, error) {
	if len(choices) == 0 {
		return shared.OpenAITextResponseChoice{}, fmt.Errorf("chat completion response has no choices")
	}
	if len(choices) > 1 {
		return shared.OpenAITextResponseChoice{}, fmt.Errorf("responses api conversion does not support multiple choices: %d", len(choices))
	}
	return choices[0], nil
}

// extractChatMessageFromResponsesOutput merges Responses output items into a
// single Chat assistant message while preserving reasoning and tool calls.
func extractChatMessageFromResponsesOutput(output []shared.ResponsesOutput) (content string, reasoning string, toolCalls []shared.ToolCallResponse, err error) {
	var builder strings.Builder
	var reasoningBuilder strings.Builder
	var calls []shared.ToolCallResponse
	for _, item := range output {
		itemType := strings.TrimSpace(item.Type)
		switch itemType {
		case openAIResponsesOutputTypeReasoning:
			for _, part := range item.Summary {
				if strings.TrimSpace(part.Text) == "" {
					continue
				}
				if reasoningBuilder.Len() > 0 {
					reasoningBuilder.WriteString("\n")
				}
				reasoningBuilder.WriteString(part.Text)
			}
		case openAIResponsesOutputTypeMessage:
			for _, part := range item.Content {
				if strings.TrimSpace(part.Type) != openAIResponsesOutputContentTypeText {
					return "", "", nil, fmt.Errorf("unsupported responses message content type: %q", part.Type)
				}
				builder.WriteString(part.Text)
			}
		case openAIResponsesOutputTypeFunctionCall, openAIResponsesOutputTypeCustomToolCall, openAIResponsesOutputTypeToolSearchCall:
			callID := strings.TrimSpace(item.CallId)
			if callID == "" {
				callID = strings.TrimSpace(item.ID)
			}
			if callID == "" {
				callID = fmt.Sprintf("call_%d", len(calls))
			}
			if itemType == openAIResponsesOutputTypeCustomToolCall {
				custom, marshalErr := jsonx.Marshal(map[string]any{"name": item.Name, "input": item.Input})
				if marshalErr != nil {
					return "", "", nil, fmt.Errorf("marshal custom tool call failed: %w", marshalErr)
				}
				calls = append(calls, shared.ToolCallResponse{ID: callID, Type: shared.CustomType, Custom: custom})
				continue
			}
			name := item.Name
			if itemType == openAIResponsesOutputTypeToolSearchCall {
				name = openAIResponsesToolSearchChatName
			} else if strings.TrimSpace(item.Namespace) != "" {
				name = flattenOpenAIResponsesNamespaceToolName(item.Namespace, item.Name)
			}
			arguments, argErr := ResponsesArgumentsToChatString(item.Arguments)
			if argErr != nil {
				return "", "", nil, fmt.Errorf("marshal %s.arguments failed: %w", itemType, argErr)
			}
			calls = append(calls, shared.ToolCallResponse{
				ID:   callID,
				Type: "function",
				Function: shared.FunctionResponse{
					Name:      name,
					Arguments: arguments,
				},
			})
		default:
			return "", "", nil, fmt.Errorf("unsupported responses output item type: %q", itemType)
		}
	}
	return builder.String(), reasoningBuilder.String(), calls, nil
}

func mapResponsesStatusToChatFinishReason(status string, sawToolCalls bool) string {
	if strings.EqualFold(strings.TrimSpace(status), "failed") {
		return "error"
	}
	if strings.EqualFold(strings.TrimSpace(status), "incomplete") {
		return "length"
	}
	if sawToolCalls {
		return "tool_calls"
	}
	return "stop"
}

func buildChatAssistantMessage(content string, reasoning string, toolCalls []shared.ToolCallResponse) (shared.Message, error) {
	msg := shared.Message{Role: "assistant", Content: content}
	if strings.TrimSpace(content) == "" {
		msg.Content = nil
	}
	if reasoning != "" {
		msg.SetReasoningContent(reasoning)
	}
	if len(toolCalls) == 0 {
		return msg, nil
	}
	raw, err := jsonx.Marshal(toolCalls)
	if err != nil {
		return shared.Message{}, fmt.Errorf("marshal tool_calls failed: %w", err)
	}
	msg.ToolCalls = raw
	return msg, nil
}

func applyResponsesUsageToChat(out *shared.OpenAITextResponse, usage *shared.Usage) {
	if out == nil {
		return
	}
	ApplyResponsesUsageToChatUsage(&out.Usage, usage)
}

func coerceCreatedAtFromResponses(createdAt int) any {
	if createdAt != 0 {
		return createdAt
	}
	return time.Now().Unix()
}

func extractChatMessageTextOnly(msg shared.Message) (string, error) {
	if msg.Content == nil {
		return "", nil
	}
	if msg.IsStringContent() {
		return msg.StringContent(), nil
	}

	parts := msg.ParseContent()
	var builder strings.Builder
	for _, part := range parts {
		if strings.TrimSpace(part.Type) != shared.ContentTypeText {
			return "", fmt.Errorf("chat response content only supports %q, got %q", shared.ContentTypeText, part.Type)
		}
		builder.WriteString(part.Text)
	}
	return strings.TrimSpace(builder.String()), nil
}

func buildResponsesOutputFromChat(msg shared.Message, text string, rawToolCalls json.RawMessage, toolContext *OpenAIWireToolContext) ([]shared.ResponsesOutput, error) {
	output := make([]shared.ResponsesOutput, 0, 2)
	if reasoning := normalizeChatResponseReasoning(msg); reasoning != "" {
		output = append(output, shared.ResponsesOutput{
			Type:   openAIResponsesOutputTypeReasoning,
			ID:     "rs_0",
			Status: "completed",
			Summary: []shared.ResponsesContentPart{{
				Type: openAIResponsesSummaryTextType,
				Text: reasoning,
			}},
		})
	}
	if strings.TrimSpace(text) != "" {
		output = append(output, shared.ResponsesOutput{
			Type:   openAIResponsesOutputTypeMessage,
			ID:     "msg_0",
			Status: "completed",
			Role:   "assistant",
			Content: []shared.ResponsesOutputContent{
				{Type: openAIResponsesOutputContentTypeText, Text: text},
			},
		})
	}

	toolOutputs, err := convertChatToolCallsToResponsesOutput(rawToolCalls, toolContext)
	if err != nil {
		return nil, err
	}
	output = append(output, toolOutputs...)
	if len(output) == 0 {
		output = append(output, shared.ResponsesOutput{
			Type:   openAIResponsesOutputTypeMessage,
			ID:     "msg_0",
			Status: "completed",
			Role:   "assistant",
			Content: []shared.ResponsesOutputContent{{
				Type: openAIResponsesOutputContentTypeText,
				Text: "",
			}},
		})
	}
	return output, nil
}

func mapChatFinishReasonToResponsesStatus(finishReason string) string {
	if strings.EqualFold(strings.TrimSpace(finishReason), "error") {
		return "failed"
	}
	if strings.EqualFold(strings.TrimSpace(finishReason), "length") {
		return "incomplete"
	}
	return "completed"
}

func normalizeChatResponseReasoning(msg shared.Message) string {
	return strings.TrimSpace(msg.GetReasoningContent())
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

func mapChatUsageToResponses(u shared.Usage) *shared.Usage {
	return MapChatUsageToResponsesUsage(u)
}

func convertChatToolCallsToResponsesOutput(raw json.RawMessage, toolContext *OpenAIWireToolContext) ([]shared.ResponsesOutput, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var calls []shared.ToolCallResponse
	if err := jsonx.Unmarshal(raw, &calls); err != nil {
		return nil, fmt.Errorf("unmarshal tool_calls failed: %w", err)
	}
	if len(calls) == 0 {
		return nil, nil
	}

	out := make([]shared.ResponsesOutput, 0, len(calls))
	for i, call := range calls {
		callID := strings.TrimSpace(call.ID)
		if callID == "" {
			callID = fmt.Sprintf("call_%d", i)
		}
		if strings.EqualFold(common.Interface2String(call.Type), shared.CustomType) {
			customName, customInput, err := parseChatCustomToolCall(call.Custom)
			if err != nil {
				return nil, fmt.Errorf("tool_calls[%d]: %w", i, err)
			}
			out = append(out, shared.ResponsesOutput{
				Type:   openAIResponsesOutputTypeCustomToolCall,
				ID:     callID,
				Status: "completed",
				Role:   "assistant",
				CallId: callID,
				Name:   customName,
				Input:  customInput,
			})
			continue
		}
		if strings.TrimSpace(call.Function.Name) == "" {
			return nil, fmt.Errorf("tool_calls[%d].function.name is required", i)
		}
		item, err := buildResponsesToolOutputFromChatCall(callID, call.Function.Name, call.Function.Arguments, toolContext)
		if err != nil {
			return nil, fmt.Errorf("tool_calls[%d]: %w", i, err)
		}
		out = append(out, item)
	}

	return out, nil
}

func buildResponsesToolOutputFromChatCall(callID string, chatName string, chatArguments string, toolContext *OpenAIWireToolContext) (shared.ResponsesOutput, error) {
	spec, ok := toolContext.ResolveToolProxy(chatName)
	if ok {
		switch spec.Type {
		case openAIResponsesToolTypeCustom:
			input, complete := ExtractResponsesCustomToolInputFromChatArguments(chatArguments)
			if !complete {
				return shared.ResponsesOutput{}, fmt.Errorf("custom tool proxy %q arguments must contain a complete %q string", chatName, openAIResponsesCustomInputField)
			}
			return shared.ResponsesOutput{
				Type:      openAIResponsesOutputTypeCustomToolCall,
				ID:        callID,
				Status:    "completed",
				Role:      "assistant",
				CallId:    callID,
				Name:      spec.Name,
				Namespace: spec.Namespace,
				Input:     input,
			}, nil
		case openAIResponsesToolTypeToolSearch:
			arguments, err := BuildResponsesToolSearchArgumentsFromChatArguments(chatArguments)
			if err != nil {
				return shared.ResponsesOutput{}, fmt.Errorf("parse tool_search arguments failed: %w", err)
			}
			return shared.ResponsesOutput{
				Type:      openAIResponsesOutputTypeToolSearchCall,
				ID:        callID,
				Status:    "completed",
				CallId:    callID,
				Execution: "client",
				Arguments: arguments,
			}, nil
		case openAIResponsesToolTypeFunction:
			return shared.ResponsesOutput{
				Type:      openAIResponsesOutputTypeFunctionCall,
				ID:        callID,
				Status:    "completed",
				Role:      "assistant",
				CallId:    callID,
				Name:      spec.Name,
				Namespace: spec.Namespace,
				Arguments: chatArguments,
			}, nil
		}
	}
	return shared.ResponsesOutput{
		Type:      openAIResponsesOutputTypeFunctionCall,
		ID:        callID,
		Status:    "completed",
		Role:      "assistant",
		CallId:    callID,
		Name:      chatName,
		Arguments: chatArguments,
	}, nil
}

func parseChatCustomToolCall(raw json.RawMessage) (name string, input string, err error) {
	if len(raw) == 0 {
		return "", "", fmt.Errorf("custom is required")
	}
	var custom map[string]any
	if err := jsonx.Unmarshal(raw, &custom); err != nil {
		return "", "", fmt.Errorf("unmarshal custom failed: %w", err)
	}
	name = strings.TrimSpace(common.Interface2String(custom["name"]))
	if name == "" {
		return "", "", fmt.Errorf("custom.name is required")
	}
	return name, common.Interface2String(custom["input"]), nil
}
