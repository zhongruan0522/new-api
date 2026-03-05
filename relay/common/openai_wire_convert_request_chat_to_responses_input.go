package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type chatToolCall struct {
	ID        string
	Name      string
	Arguments string
}

func buildResponsesInputFromChatMessages(messages []dto.Message) (json.RawMessage, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("responses input is empty after stripping instructions")
	}

	items := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		add, err := buildResponsesInputItemsFromChatMessage(msg)
		if err != nil {
			return nil, err
		}
		items = append(items, add...)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("responses input items are empty")
	}

	raw, err := common.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal responses input failed: %w", err)
	}
	return raw, nil
}

func buildResponsesInputItemsFromChatMessage(msg dto.Message) ([]map[string]any, error) {
	role := strings.TrimSpace(msg.Role)
	if role == "" {
		return nil, fmt.Errorf("chat message role is required")
	}
	roleLower := strings.ToLower(role)

	if roleLower == "tool" {
		return buildResponsesFunctionCallOutputItemFromToolMessage(msg)
	}
	if strings.TrimSpace(msg.ToolCallId) != "" {
		return nil, fmt.Errorf("tool_call_id is only supported for role \"tool\"")
	}

	toolCalls, err := parseChatMessageToolCalls(msg.ToolCalls)
	if err != nil {
		return nil, err
	}
	if len(toolCalls) > 0 && roleLower != "assistant" {
		return nil, fmt.Errorf("tool_calls is only supported for role \"assistant\"")
	}

	items := make([]map[string]any, 0, 1+len(toolCalls))
	item, ok, err := buildResponsesMessageItemFromChatMessage(role, msg, len(toolCalls) > 0)
	if err != nil {
		return nil, err
	}
	if ok {
		items = append(items, item)
	}
	if len(toolCalls) > 0 {
		items = append(items, buildResponsesFunctionCallInputItems(toolCalls)...)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("empty input after converting message (role=%q)", role)
	}
	return items, nil
}

func buildResponsesFunctionCallOutputItemFromToolMessage(msg dto.Message) ([]map[string]any, error) {
	callID := strings.TrimSpace(msg.ToolCallId)
	if callID == "" {
		return nil, fmt.Errorf("tool message requires tool_call_id")
	}
	output, err := chatMessageTextOnly(msg)
	if err != nil {
		return nil, err
	}
	return []map[string]any{
		{
			"type":    openAIResponsesInputItemTypeFunctionCallOutput,
			"call_id": callID,
			"output":  output,
		},
	}, nil
}

func buildResponsesMessageItemFromChatMessage(role string, msg dto.Message, allowSkipEmpty bool) (map[string]any, bool, error) {
	content, err := buildResponsesContentFromChatMessage(msg)
	if err != nil {
		return nil, false, err
	}
	if allowSkipEmpty && responsesMessageContentIsEmpty(content) {
		return nil, false, nil
	}

	return map[string]any{
		"type":    openAIResponsesInputItemTypeMessage,
		"role":    role,
		"content": content,
	}, true, nil
}

func responsesMessageContentIsEmpty(content any) bool {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []map[string]any:
		return len(v) == 0
	default:
		return true
	}
}

func buildResponsesContentFromChatMessage(msg dto.Message) (any, error) {
	if msg.Content == nil {
		return "", nil
	}
	if msg.IsStringContent() {
		return msg.StringContent(), nil
	}

	parts := msg.ParseContent()
	out := make([]map[string]any, 0, len(parts))
	for _, part := range parts {
		switch part.Type {
		case dto.ContentTypeText:
			out = append(out, map[string]any{
				"type": openAIResponsesInputTypeText,
				"text": part.Text,
			})
		case dto.ContentTypeImageURL:
			image := part.GetImageMedia()
			if image == nil || strings.TrimSpace(image.Url) == "" {
				return nil, fmt.Errorf("invalid image_url content")
			}
			imageObj := map[string]any{"url": image.Url}
			if strings.TrimSpace(image.Detail) != "" {
				imageObj["detail"] = image.Detail
			}
			out = append(out, map[string]any{
				"type":      openAIResponsesInputTypeImage,
				"image_url": imageObj,
			})
		default:
			return nil, fmt.Errorf("unsupported chat content type for responses conversion: %q", part.Type)
		}
	}
	return out, nil
}

func parseChatMessageToolCalls(raw json.RawMessage) ([]chatToolCall, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	rawType := common.GetJsonType(raw)
	if rawType == "null" {
		return nil, nil
	}
	if rawType != "array" {
		return nil, fmt.Errorf("tool_calls must be an array, got %s", rawType)
	}

	var items []map[string]any
	if err := common.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("unmarshal tool_calls failed: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	calls := make([]chatToolCall, 0, len(items))
	for i, item := range items {
		callID := strings.TrimSpace(common.Interface2String(item["id"]))
		if callID == "" {
			callID = fmt.Sprintf("call_%d", i)
		}
		fn, _ := item["function"].(map[string]any)
		name := strings.TrimSpace(common.Interface2String(fn["name"]))
		if name == "" {
			return nil, fmt.Errorf("tool_calls[%d].function.name is required", i)
		}
		args := common.Interface2String(fn["arguments"])
		calls = append(calls, chatToolCall{ID: callID, Name: name, Arguments: args})
	}

	return calls, nil
}

func buildResponsesFunctionCallInputItems(calls []chatToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		out = append(out, map[string]any{
			"type":      openAIResponsesInputItemTypeFunctionCall,
			"call_id":   call.ID,
			"name":      call.Name,
			"arguments": call.Arguments,
		})
	}
	return out
}
