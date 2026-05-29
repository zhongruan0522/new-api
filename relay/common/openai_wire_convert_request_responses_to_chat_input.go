package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

type responsesInputTypeProbe struct {
	Type string `json:"type"`
}

type responsesFunctionCallInput struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Input     string `json:"input"`
}

type responsesFunctionCallOutputInput struct {
	Type    string          `json:"type"`
	CallID  string          `json:"call_id"`
	Output  json.RawMessage `json:"output"`
	IsError *bool           `json:"tool_call_is_error,omitempty"`
}

type responsesReasoningInput struct {
	Type    string                     `json:"type"`
	Summary []dto.ResponsesContentPart `json:"summary,omitempty"`
}

func buildChatMessagesFromResponsesInput(raw json.RawMessage) ([]dto.Message, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("input is required")
	}

	switch common.GetJsonType(raw) {
	case "string":
		var s string
		if err := common.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("unmarshal input string failed: %w", err)
		}
		return []dto.Message{{Role: "user", Content: s}}, nil
	case "array":
		return buildChatMessagesFromResponsesInputArray(raw)
	default:
		return nil, fmt.Errorf("unsupported input type: %s", common.GetJsonType(raw))
	}
}

func buildChatMessagesFromResponsesInputArray(raw json.RawMessage) ([]dto.Message, error) {
	var items []json.RawMessage
	if err := common.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("unmarshal input array failed: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("input items are empty")
	}

	out := make([]dto.Message, 0, len(items))
	pendingReasoning := ""
	pendingToolCalls := make([]dto.ToolCallResponse, 0)
	flushToolCalls := func() error {
		if len(pendingToolCalls) == 0 {
			return nil
		}
		if len(out) > 0 && strings.EqualFold(out[len(out)-1].Role, "assistant") {
			if err := appendToolCallsToChatAssistantMessage(&out[len(out)-1], pendingReasoning, pendingToolCalls); err != nil {
				return err
			}
			pendingReasoning = ""
			pendingToolCalls = pendingToolCalls[:0]
			return nil
		}
		msg, err := buildChatAssistantMessage("", pendingReasoning, pendingToolCalls)
		if err != nil {
			return err
		}
		out = append(out, msg)
		pendingReasoning = ""
		pendingToolCalls = pendingToolCalls[:0]
		return nil
	}
	for i, itemRaw := range items {
		itemType, err := probeResponsesInputItemType(itemRaw)
		if err != nil {
			return nil, fmt.Errorf("input[%d]: %w", i, err)
		}
		if itemType == "" {
			itemType = openAIResponsesInputItemTypeMessage
		}
		if itemType == openAIResponsesInputItemTypeReasoning {
			reasoning, err := extractResponsesReasoningSummary(itemRaw)
			if err != nil {
				return nil, fmt.Errorf("input[%d]: %w", i, err)
			}
			if strings.TrimSpace(reasoning) != "" {
				pendingReasoning = appendReasoningSummary(pendingReasoning, reasoning)
			}
			continue
		}
		if itemType == openAIResponsesInputItemTypeFunctionCall || itemType == openAIResponsesInputItemTypeCustomToolCall {
			call, err := buildChatToolCallFromResponsesFunctionCall(itemRaw)
			if err != nil {
				return nil, fmt.Errorf("input[%d]: %w", i, err)
			}
			pendingToolCalls = append(pendingToolCalls, call)
			continue
		}
		if err := flushToolCalls(); err != nil {
			return nil, fmt.Errorf("input[%d]: %w", i, err)
		}

		msgs, err := buildChatMessagesFromResponsesInputItemByType(itemType, itemRaw)
		if err != nil {
			return nil, fmt.Errorf("input[%d]: %w", i, err)
		}
		if strings.TrimSpace(pendingReasoning) != "" {
			attachReasoningToMessages(&msgs, pendingReasoning)
			pendingReasoning = ""
		}
		out = append(out, msgs...)
	}
	if err := flushToolCalls(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(pendingReasoning) != "" {
		out = append(out, dto.Message{Role: "assistant", ReasoningContent: pendingReasoning})
	}
	return out, nil
}

func buildChatMessagesFromResponsesInputItemByType(itemType string, raw json.RawMessage) ([]dto.Message, error) {
	switch itemType {
	case openAIResponsesInputItemTypeMessage:
		return buildChatMessagesFromResponsesMessageItem(raw)
	case openAIResponsesInputItemTypeReasoning:
		reasoning, err := extractResponsesReasoningSummary(raw)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(reasoning) == "" {
			return nil, nil
		}
		return []dto.Message{{Role: "assistant", ReasoningContent: reasoning}}, nil
	case openAIResponsesInputItemTypeFunctionCall, openAIResponsesInputItemTypeCustomToolCall:
		msg, err := buildChatToolCallMessageFromResponsesFunctionCall(raw)
		if err != nil {
			return nil, err
		}
		return []dto.Message{msg}, nil
	case openAIResponsesInputItemTypeFunctionCallOutput, openAIResponsesInputItemTypeCustomToolOutput:
		msg, err := buildChatToolOutputMessageFromResponsesFunctionCallOutput(raw)
		if err != nil {
			return nil, err
		}
		return []dto.Message{msg}, nil
	default:
		return nil, fmt.Errorf("unsupported input item type: %q", itemType)
	}
}

func probeResponsesInputItemType(raw json.RawMessage) (string, error) {
	var probe responsesInputTypeProbe
	if err := common.Unmarshal(raw, &probe); err != nil {
		return "", fmt.Errorf("unmarshal input item type failed: %w", err)
	}
	return strings.TrimSpace(probe.Type), nil
}

func buildChatMessagesFromResponsesMessageItem(raw json.RawMessage) ([]dto.Message, error) {
	var item dto.Input
	if err := common.Unmarshal(raw, &item); err != nil {
		return nil, fmt.Errorf("unmarshal message item failed: %w", err)
	}

	role := strings.TrimSpace(item.Role)
	if role == "" {
		role = "user"
	}

	msg, err := buildChatMessageFromResponsesMessageContent(role, item.Content)
	if err != nil {
		return nil, err
	}
	return []dto.Message{msg}, nil
}

func buildChatMessageFromResponsesMessageContent(role string, raw json.RawMessage) (dto.Message, error) {
	switch common.GetJsonType(raw) {
	case "string":
		var s string
		if err := common.Unmarshal(raw, &s); err != nil {
			return dto.Message{}, fmt.Errorf("unmarshal content string failed: %w", err)
		}
		return dto.Message{Role: role, Content: s}, nil
	case "array":
		var parts []map[string]any
		if err := common.Unmarshal(raw, &parts); err != nil {
			return dto.Message{}, fmt.Errorf("unmarshal content parts failed: %w", err)
		}
		media, err := convertResponsesContentPartsToChat(parts)
		if err != nil {
			return dto.Message{}, err
		}
		if text, ok := collapseChatMediaToString(media); ok {
			return dto.Message{Role: role, Content: text}, nil
		}
		return dto.Message{Role: role, Content: media}, nil
	default:
		return dto.Message{}, fmt.Errorf("unsupported content type: %s", common.GetJsonType(raw))
	}
}

func buildChatToolCallMessageFromResponsesFunctionCall(raw json.RawMessage) (dto.Message, error) {
	call, err := buildChatToolCallFromResponsesFunctionCall(raw)
	if err != nil {
		return dto.Message{}, err
	}

	rawToolCalls, err := common.Marshal([]dto.ToolCallResponse{call})
	if err != nil {
		return dto.Message{}, fmt.Errorf("marshal tool_calls failed: %w", err)
	}

	return dto.Message{
		Role:      "assistant",
		Content:   nil,
		ToolCalls: rawToolCalls,
	}, nil
}

func buildChatToolCallFromResponsesFunctionCall(raw json.RawMessage) (dto.ToolCallResponse, error) {
	var item responsesFunctionCallInput
	if err := common.Unmarshal(raw, &item); err != nil {
		return dto.ToolCallResponse{}, fmt.Errorf("unmarshal function_call item failed: %w", err)
	}
	callID := strings.TrimSpace(item.CallID)
	if callID == "" {
		return dto.ToolCallResponse{}, fmt.Errorf("function_call.call_id is required")
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return dto.ToolCallResponse{}, fmt.Errorf("%s.name is required", item.Type)
	}
	if strings.TrimSpace(item.Type) == openAIResponsesInputItemTypeCustomToolCall {
		custom, err := common.Marshal(map[string]any{
			"name":  name,
			"input": item.Input,
		})
		if err != nil {
			return dto.ToolCallResponse{}, fmt.Errorf("marshal custom tool call failed: %w", err)
		}
		return dto.ToolCallResponse{
			ID:     callID,
			Type:   dto.CustomType,
			Custom: custom,
		}, nil
	}

	return dto.ToolCallResponse{
		ID:   callID,
		Type: "function",
		Function: dto.FunctionResponse{
			Name:      name,
			Arguments: item.Arguments,
		},
	}, nil
}

func buildChatToolOutputMessageFromResponsesFunctionCallOutput(raw json.RawMessage) (dto.Message, error) {
	var item responsesFunctionCallOutputInput
	if err := common.Unmarshal(raw, &item); err != nil {
		return dto.Message{}, fmt.Errorf("unmarshal function_call_output item failed: %w", err)
	}
	callID := strings.TrimSpace(item.CallID)
	if callID == "" {
		return dto.Message{}, fmt.Errorf("function_call_output.call_id is required")
	}
	output, err := responsesFunctionCallOutputToChatContent(item.Output)
	if err != nil {
		return dto.Message{}, err
	}

	return dto.Message{
		Role:            "tool",
		Content:         output,
		ToolCallId:      callID,
		ToolCallIsError: item.IsError,
	}, nil
}

func appendToolCallsToChatAssistantMessage(msg *dto.Message, reasoning string, toolCalls []dto.ToolCallResponse) error {
	if msg == nil || len(toolCalls) == 0 {
		return nil
	}
	if strings.TrimSpace(reasoning) != "" {
		msg.ReasoningContent = appendReasoningSummary(msg.ReasoningContent, reasoning)
	}
	existing := make([]dto.ToolCallResponse, 0)
	if len(msg.ToolCalls) > 0 && common.GetJsonType(msg.ToolCalls) != "null" {
		if err := common.Unmarshal(msg.ToolCalls, &existing); err != nil {
			return fmt.Errorf("unmarshal existing tool_calls failed: %w", err)
		}
	}
	existing = append(existing, toolCalls...)
	raw, err := common.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshal tool_calls failed: %w", err)
	}
	msg.ToolCalls = raw
	return nil
}

func responsesFunctionCallOutputToChatContent(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || common.GetJsonType(raw) == "null" {
		return "", nil
	}
	if common.GetJsonType(raw) == "string" {
		var output string
		if err := common.Unmarshal(raw, &output); err != nil {
			return nil, fmt.Errorf("unmarshal function_call_output.output failed: %w", err)
		}
		return output, nil
	}
	if common.GetJsonType(raw) != "array" {
		return nil, fmt.Errorf("function_call_output.output must be a string or content array, got %s", common.GetJsonType(raw))
	}

	var parts []map[string]any
	if err := common.Unmarshal(raw, &parts); err != nil {
		return nil, fmt.Errorf("unmarshal function_call_output.output failed: %w", err)
	}
	media, err := convertResponsesContentPartsToChat(parts)
	if err != nil {
		return nil, err
	}
	text, ok := collapseChatMediaToString(media)
	if !ok {
		return nil, fmt.Errorf("function_call_output.output only supports text parts for chat.completions conversion")
	}
	return text, nil
}

func convertResponsesContentPartsToChat(parts []map[string]any) ([]dto.MediaContent, error) {
	out := make([]dto.MediaContent, 0, len(parts))
	for _, part := range parts {
		typ, _ := part["type"].(string)
		typ = strings.TrimSpace(typ)
		switch typ {
		case openAIResponsesInputTypeText, openAIResponsesOutputTypeText:
			text, _ := part["text"].(string)
			out = append(out, dto.MediaContent{Type: dto.ContentTypeText, Text: text})
		case openAIResponsesInputTypeImage:
			image := extractResponsesImageURL(part["image_url"])
			if image == nil || strings.TrimSpace(image.Url) == "" {
				return nil, fmt.Errorf("invalid input_image content")
			}
			out = append(out, dto.MediaContent{Type: dto.ContentTypeImageURL, ImageUrl: image})
		case openAIResponsesInputTypeFile:
			file := extractResponsesFile(part)
			if file == nil {
				return nil, fmt.Errorf("invalid input_file content")
			}
			out = append(out, dto.MediaContent{Type: dto.ContentTypeFile, File: file})
		default:
			return nil, fmt.Errorf("unsupported responses content type: %q", typ)
		}
	}
	return out, nil
}

func extractResponsesImageURL(v any) *dto.MessageImageUrl {
	switch val := v.(type) {
	case string:
		return &dto.MessageImageUrl{Url: val, Detail: "high"}
	case map[string]any:
		u, _ := val["url"].(string)
		detail, _ := val["detail"].(string)
		if strings.TrimSpace(detail) == "" {
			detail = "high"
		}
		return &dto.MessageImageUrl{Url: u, Detail: detail}
	default:
		return nil
	}
}

// extractResponsesReasoningSummary flattens a Responses reasoning item into the
// assistant reasoning_content field used by ChatCompletions-compatible clients.
func appendReasoningSummary(existing string, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	return existing + "\n" + next
}

func extractResponsesReasoningSummary(raw json.RawMessage) (string, error) {
	var item responsesReasoningInput
	if err := common.Unmarshal(raw, &item); err != nil {
		return "", fmt.Errorf("unmarshal reasoning item failed: %w", err)
	}
	parts := make([]string, 0, len(item.Summary))
	for _, summary := range item.Summary {
		if strings.TrimSpace(summary.Text) != "" {
			parts = append(parts, summary.Text)
		}
	}
	return strings.Join(parts, "\n"), nil
}

func attachReasoningToMessages(msgs *[]dto.Message, reasoning string) {
	if msgs == nil || len(*msgs) == 0 {
		return
	}
	for i := range *msgs {
		if strings.EqualFold((*msgs)[i].Role, "assistant") {
			(*msgs)[i].ReasoningContent = reasoning
			return
		}
	}
	*msgs = append([]dto.Message{{Role: "assistant", ReasoningContent: reasoning}}, (*msgs)...)
}

func extractResponsesFile(part map[string]any) *dto.MessageFile {
	if part == nil {
		return nil
	}
	file := &dto.MessageFile{}
	if fileID, _ := part["file_id"].(string); strings.TrimSpace(fileID) != "" {
		file.FileId = fileID
		return file
	}
	if fileData, _ := part["file_data"].(string); strings.TrimSpace(fileData) != "" {
		file.FileData = fileData
		if fileName, _ := part["filename"].(string); strings.TrimSpace(fileName) != "" {
			file.FileName = fileName
		}
		return file
	}
	if fileURL, _ := part["file_url"].(string); strings.TrimSpace(fileURL) != "" {
		file.FileData = fileURL
		if fileName, _ := part["filename"].(string); strings.TrimSpace(fileName) != "" {
			file.FileName = fileName
		}
		return file
	}
	return nil
}

func collapseChatMediaToString(media []dto.MediaContent) (string, bool) {
	if len(media) == 0 {
		return "", true
	}
	var builder strings.Builder
	for _, part := range media {
		if part.Type != dto.ContentTypeText {
			return "", false
		}
		builder.WriteString(part.Text)
	}
	return builder.String(), true
}
