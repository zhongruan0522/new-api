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
}

type responsesFunctionCallOutputInput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output any    `json:"output"`
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
				pendingReasoning = reasoning
			}
			continue
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
	case openAIResponsesInputItemTypeFunctionCall:
		msg, err := buildChatToolCallMessageFromResponsesFunctionCall(raw)
		if err != nil {
			return nil, err
		}
		return []dto.Message{msg}, nil
	case openAIResponsesInputItemTypeFunctionCallOutput:
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
	var item responsesFunctionCallInput
	if err := common.Unmarshal(raw, &item); err != nil {
		return dto.Message{}, fmt.Errorf("unmarshal function_call item failed: %w", err)
	}
	callID := strings.TrimSpace(item.CallID)
	if callID == "" {
		return dto.Message{}, fmt.Errorf("function_call.call_id is required")
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return dto.Message{}, fmt.Errorf("function_call.name is required")
	}

	rawToolCalls, err := common.Marshal([]dto.ToolCallResponse{
		{
			ID:   callID,
			Type: "function",
			Function: dto.FunctionResponse{
				Name:      name,
				Arguments: item.Arguments,
			},
		},
	})
	if err != nil {
		return dto.Message{}, fmt.Errorf("marshal tool_calls failed: %w", err)
	}

	return dto.Message{
		Role:      "assistant",
		Content:   nil,
		ToolCalls: rawToolCalls,
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
	output, ok := item.Output.(string)
	if !ok {
		return dto.Message{}, fmt.Errorf("function_call_output.output must be a string")
	}

	return dto.Message{
		Role:       "tool",
		Content:    output,
		ToolCallId: callID,
	}, nil
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
