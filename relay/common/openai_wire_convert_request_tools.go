package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

type openAIResponsesFunctionTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Format      json.RawMessage `json:"format,omitempty"`
}

func convertChatToolChoiceToResponsesRaw(choice any) (json.RawMessage, error) {
	switch v := choice.(type) {
	case string:
		raw, err := common.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal tool_choice failed: %w", err)
		}
		return raw, nil
	case map[string]any:
		return convertChatToolChoiceObjectToResponsesRaw(v)
	default:
		return nil, fmt.Errorf("tool_choice must be string or object, got %T", choice)
	}
}

func convertChatToolChoiceObjectToResponsesRaw(obj map[string]any) (json.RawMessage, error) {
	toolType, ok := obj["type"].(string)
	if !ok || strings.TrimSpace(toolType) == "" {
		return nil, fmt.Errorf("tool_choice.type is required")
	}
	toolType = strings.ToLower(strings.TrimSpace(toolType))
	if toolType != "function" && toolType != dto.CustomType {
		return nil, fmt.Errorf("tool_choice.type %q is not supported for responses conversion", toolType)
	}

	name, _ := obj["name"].(string)
	if strings.TrimSpace(name) == "" {
		name = getToolChoiceName(obj, toolType)
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("tool_choice.%s.name is required", toolType)
	}

	raw, err := common.Marshal(map[string]any{"type": toolType, "name": name})
	if err != nil {
		return nil, fmt.Errorf("marshal tool_choice failed: %w", err)
	}
	return raw, nil
}

func getToolChoiceName(obj map[string]any, toolType string) string {
	tool, ok := obj[toolType].(map[string]any)
	if !ok {
		return ""
	}
	name, _ := tool["name"].(string)
	return strings.TrimSpace(name)
}

func convertChatToolsToResponsesRaw(tools []dto.ToolCallRequest) (json.RawMessage, error) {
	out := make([]openAIResponsesFunctionTool, 0, len(tools))
	for i, tool := range tools {
		item, err := convertOneChatToolToResponsesTool(i, tool)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	raw, err := common.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal tools failed: %w", err)
	}
	return raw, nil
}

func convertOneChatToolToResponsesTool(index int, tool dto.ToolCallRequest) (openAIResponsesFunctionTool, error) {
	toolType := strings.ToLower(strings.TrimSpace(tool.Type))
	if toolType == "" {
		return openAIResponsesFunctionTool{}, fmt.Errorf("tools[%d].type is required", index)
	}
	if toolType != "function" && toolType != dto.CustomType {
		return openAIResponsesFunctionTool{}, fmt.Errorf("tools[%d].type %q is not supported for responses conversion", index, tool.Type)
	}
	if toolType == dto.CustomType {
		return convertOneChatCustomToolToResponsesTool(index, tool)
	}
	name := strings.TrimSpace(tool.Function.Name)
	if name == "" {
		return openAIResponsesFunctionTool{}, fmt.Errorf("tools[%d].function.name is required", index)
	}

	var params json.RawMessage
	if tool.Function.Parameters != nil {
		raw, err := common.Marshal(tool.Function.Parameters)
		if err != nil {
			return openAIResponsesFunctionTool{}, fmt.Errorf("marshal tools[%d].function.parameters failed: %w", index, err)
		}
		params = raw
	}

	return openAIResponsesFunctionTool{
		Type:        "function",
		Name:        name,
		Description: tool.Function.Description,
		Parameters:  params,
	}, nil
}

func convertOneChatCustomToolToResponsesTool(index int, tool dto.ToolCallRequest) (openAIResponsesFunctionTool, error) {
	var custom map[string]json.RawMessage
	if len(tool.Custom) == 0 {
		return openAIResponsesFunctionTool{}, fmt.Errorf("tools[%d].custom is required", index)
	}
	if err := common.Unmarshal(tool.Custom, &custom); err != nil {
		return openAIResponsesFunctionTool{}, fmt.Errorf("unmarshal tools[%d].custom failed: %w", index, err)
	}

	var name string
	if raw := custom["name"]; len(raw) > 0 {
		if err := common.Unmarshal(raw, &name); err != nil {
			return openAIResponsesFunctionTool{}, fmt.Errorf("unmarshal tools[%d].custom.name failed: %w", index, err)
		}
	}
	if strings.TrimSpace(name) == "" {
		return openAIResponsesFunctionTool{}, fmt.Errorf("tools[%d].custom.name is required", index)
	}

	var description string
	if raw := custom["description"]; len(raw) > 0 {
		if err := common.Unmarshal(raw, &description); err != nil {
			return openAIResponsesFunctionTool{}, fmt.Errorf("unmarshal tools[%d].custom.description failed: %w", index, err)
		}
	}

	return openAIResponsesFunctionTool{
		Type:        dto.CustomType,
		Name:        name,
		Description: description,
		Format:      custom["format"],
	}, nil
}

func convertResponsesToolChoiceToChatAny(raw json.RawMessage) (any, error) {
	var v any
	if err := common.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("unmarshal tool_choice failed: %w", err)
	}

	switch t := v.(type) {
	case string:
		return t, nil
	case map[string]any:
		return convertResponsesToolChoiceObjectToChatAny(t)
	default:
		return nil, fmt.Errorf("tool_choice must be string or object, got %T", v)
	}
}

func convertResponsesToolChoiceObjectToChatAny(obj map[string]any) (any, error) {
	toolType, ok := obj["type"].(string)
	if !ok || strings.TrimSpace(toolType) == "" {
		return nil, fmt.Errorf("tool_choice.type is required")
	}
	toolType = strings.ToLower(strings.TrimSpace(toolType))
	if toolType != "function" && toolType != dto.CustomType {
		return nil, fmt.Errorf("tool_choice.type %q is not supported for chat.completions conversion", toolType)
	}

	name, _ := obj["name"].(string)
	if strings.TrimSpace(name) == "" {
		name = getToolChoiceName(obj, toolType)
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("tool_choice.name is required")
	}
	if toolType == dto.CustomType {
		return map[string]any{
			"type": dto.CustomType,
			"custom": map[string]any{
				"name": name,
			},
		}, nil
	}

	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": name,
		},
	}, nil
}

func convertResponsesToolsRawToChatTools(raw json.RawMessage) ([]dto.ToolCallRequest, error) {
	var tools []openAIResponsesFunctionTool
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, fmt.Errorf("unmarshal tools failed: %w", err)
	}

	out := make([]dto.ToolCallRequest, 0, len(tools))
	for i, tool := range tools {
		item, err := convertOneResponsesToolToChatTool(i, tool)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func convertOneResponsesToolToChatTool(index int, tool openAIResponsesFunctionTool) (dto.ToolCallRequest, error) {
	toolType := strings.ToLower(strings.TrimSpace(tool.Type))
	if toolType == "" {
		return dto.ToolCallRequest{}, fmt.Errorf("tools[%d].type is required", index)
	}
	if toolType != "function" && toolType != dto.CustomType {
		return dto.ToolCallRequest{}, fmt.Errorf("tools[%d].type %q is not supported for chat.completions conversion", index, tool.Type)
	}
	if toolType == dto.CustomType {
		return convertOneResponsesCustomToolToChatTool(index, tool)
	}
	name := strings.TrimSpace(tool.Name)
	if name == "" {
		return dto.ToolCallRequest{}, fmt.Errorf("tools[%d].name is required", index)
	}

	var params any
	if len(tool.Parameters) > 0 {
		if err := common.Unmarshal(tool.Parameters, &params); err != nil {
			return dto.ToolCallRequest{}, fmt.Errorf("unmarshal tools[%d].parameters failed: %w", index, err)
		}
	}

	return dto.ToolCallRequest{
		Type: "function",
		Function: dto.FunctionRequest{
			Name:        name,
			Description: tool.Description,
			Parameters:  params,
		},
	}, nil
}

func convertOneResponsesCustomToolToChatTool(index int, tool openAIResponsesFunctionTool) (dto.ToolCallRequest, error) {
	name := strings.TrimSpace(tool.Name)
	if name == "" {
		return dto.ToolCallRequest{}, fmt.Errorf("tools[%d].name is required", index)
	}
	custom := map[string]any{"name": name}
	if strings.TrimSpace(tool.Description) != "" {
		custom["description"] = tool.Description
	}
	if len(tool.Format) > 0 {
		var format any
		if err := common.Unmarshal(tool.Format, &format); err != nil {
			return dto.ToolCallRequest{}, fmt.Errorf("unmarshal tools[%d].format failed: %w", index, err)
		}
		custom["format"] = format
	}
	raw, err := common.Marshal(custom)
	if err != nil {
		return dto.ToolCallRequest{}, fmt.Errorf("marshal tools[%d].custom failed: %w", index, err)
	}
	return dto.ToolCallRequest{Type: dto.CustomType, Custom: raw}, nil
}
