package common

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

const openAIResponsesChatToolNameMaxLen = 64

type openAIResponsesFunctionTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Format      json.RawMessage `json:"format,omitempty"`
	Tools       json.RawMessage `json:"tools,omitempty"`
	Children    json.RawMessage `json:"children,omitempty"`
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
	if toolType != openAIResponsesToolTypeFunction &&
		toolType != openAIResponsesToolTypeCustom &&
		toolType != openAIResponsesToolTypeToolSearch {
		return nil, fmt.Errorf("tool_choice.type %q is not supported for chat.completions conversion", toolType)
	}
	if toolType == openAIResponsesToolTypeToolSearch {
		return map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": openAIResponsesToolSearchChatName,
			},
		}, nil
	}

	name, _ := obj["name"].(string)
	if strings.TrimSpace(name) == "" {
		name = getToolChoiceName(obj, toolType)
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("tool_choice.name is required")
	}
	if namespace, _ := obj["namespace"].(string); strings.TrimSpace(namespace) != "" {
		name = flattenOpenAIResponsesNamespaceToolName(namespace, name)
	}

	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": name,
		},
	}, nil
}

func convertResponsesToolsRawToChatTools(raw json.RawMessage) ([]dto.ToolCallRequest, error) {
	return convertResponsesToolsRawToChatToolsWithToolContext(raw, nil)
}

func convertResponsesToolsRawToChatToolsWithToolContext(raw json.RawMessage, toolContext *OpenAIWireToolContext) ([]dto.ToolCallRequest, error) {
	var tools []openAIResponsesFunctionTool
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, fmt.Errorf("unmarshal tools failed: %w", err)
	}

	out := make([]dto.ToolCallRequest, 0, len(tools))
	for i, tool := range tools {
		items, err := convertOneResponsesToolToChatTools(i, tool, "", toolContext)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func collectChatToolsFromResponsesToolSearchOutputs(raw json.RawMessage) ([]dto.ToolCallRequest, error) {
	return collectChatToolsFromResponsesToolSearchOutputsWithToolContext(raw, nil)
}

func collectChatToolsFromResponsesToolSearchOutputsWithToolContext(raw json.RawMessage, toolContext *OpenAIWireToolContext) ([]dto.ToolCallRequest, error) {
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("unmarshal input for tool_search tools failed: %w", err)
	}
	out := make([]dto.ToolCallRequest, 0)
	if err := collectChatToolsFromResponsesValue(value, &out, toolContext); err != nil {
		return nil, err
	}
	return out, nil
}

func collectChatToolsFromResponsesValue(value any, out *[]dto.ToolCallRequest, toolContext *OpenAIWireToolContext) error {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if err := collectChatToolsFromResponsesValue(item, out, toolContext); err != nil {
				return err
			}
		}
	case map[string]any:
		if typ, _ := v["type"].(string); strings.TrimSpace(typ) == openAIResponsesInputItemTypeToolSearchOutput {
			if toolsAny, ok := v["tools"]; ok {
				raw, err := common.Marshal(toolsAny)
				if err != nil {
					return fmt.Errorf("marshal tool_search_output.tools failed: %w", err)
				}
				tools, err := convertResponsesToolsRawToChatToolsWithToolContext(raw, toolContext)
				if err != nil {
					return fmt.Errorf("convert tool_search_output.tools failed: %w", err)
				}
				*out = append(*out, tools...)
			}
		}
		for _, child := range v {
			if err := collectChatToolsFromResponsesValue(child, out, toolContext); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendUniqueChatTools(base []dto.ToolCallRequest, extra []dto.ToolCallRequest) []dto.ToolCallRequest {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]dto.ToolCallRequest, 0, len(base)+len(extra))
	for _, tool := range base {
		name := chatToolRequestName(tool)
		if name != "" {
			seen[name] = struct{}{}
		}
		out = append(out, tool)
	}
	for _, tool := range extra {
		name := chatToolRequestName(tool)
		if name != "" {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
		}
		out = append(out, tool)
	}
	return out
}

func chatToolRequestName(tool dto.ToolCallRequest) string {
	if strings.TrimSpace(tool.Function.Name) != "" {
		return strings.TrimSpace(tool.Function.Name)
	}
	if len(tool.Custom) == 0 {
		return ""
	}
	var custom map[string]any
	if err := common.Unmarshal(tool.Custom, &custom); err != nil {
		return ""
	}
	return strings.TrimSpace(common.Interface2String(custom["name"]))
}

func convertOneResponsesToolToChatTools(index int, tool openAIResponsesFunctionTool, namespace string, toolContext *OpenAIWireToolContext) ([]dto.ToolCallRequest, error) {
	toolType := strings.ToLower(strings.TrimSpace(tool.Type))
	if toolType == "" {
		return nil, fmt.Errorf("tools[%d].type is required", index)
	}
	switch toolType {
	case openAIResponsesToolTypeFunction:
		item, err := convertOneResponsesFunctionToolToChatTool(index, tool, namespace, toolContext)
		if err != nil {
			return nil, err
		}
		return []dto.ToolCallRequest{item}, nil
	case openAIResponsesToolTypeCustom:
		item, err := convertOneResponsesCustomToolToChatTool(index, tool, namespace, toolContext)
		if err != nil {
			return nil, err
		}
		return []dto.ToolCallRequest{item}, nil
	case openAIResponsesToolTypeToolSearch:
		if toolContext != nil {
			toolContext.AddToolSearchProxy(openAIResponsesToolSearchChatName)
		}
		return []dto.ToolCallRequest{newResponsesToolSearchChatTool()}, nil
	case openAIResponsesToolTypeNamespace:
		return convertOneResponsesNamespaceToolToChatTools(index, tool, toolContext)
	default:
		return nil, fmt.Errorf("tools[%d].type %q is not supported for chat.completions conversion", index, tool.Type)
	}
}

func convertOneResponsesFunctionToolToChatTool(index int, tool openAIResponsesFunctionTool, namespace string, toolContext *OpenAIWireToolContext) (dto.ToolCallRequest, error) {
	originalName := strings.TrimSpace(tool.Name)
	name := strings.TrimSpace(tool.Name)
	if name == "" {
		return dto.ToolCallRequest{}, fmt.Errorf("tools[%d].name is required", index)
	}
	if strings.TrimSpace(namespace) != "" {
		name = flattenOpenAIResponsesNamespaceToolName(namespace, name)
	}
	if toolContext != nil {
		toolContext.AddFunctionToolProxy(name, originalName, namespace)
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

func convertOneResponsesCustomToolToChatTool(index int, tool openAIResponsesFunctionTool, namespace string, toolContext *OpenAIWireToolContext) (dto.ToolCallRequest, error) {
	originalName := strings.TrimSpace(tool.Name)
	name := originalName
	if originalName == "" {
		return dto.ToolCallRequest{}, fmt.Errorf("tools[%d].name is required", index)
	}
	if strings.TrimSpace(namespace) != "" {
		name = flattenOpenAIResponsesNamespaceToolName(namespace, name)
	}
	if toolContext != nil {
		toolContext.AddCustomToolProxy(name, originalName, namespace)
	}
	return dto.ToolCallRequest{
		Type: "function",
		Function: dto.FunctionRequest{
			Name:        name,
			Description: buildResponsesCustomToolChatDescription(tool),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					openAIResponsesCustomInputField: map[string]any{
						"type":        "string",
						"description": "Raw string input for the original custom tool. Preserve formatting exactly.",
					},
				},
				"required": []string{openAIResponsesCustomInputField},
			},
		},
	}, nil
}

func convertOneResponsesNamespaceToolToChatTools(index int, tool openAIResponsesFunctionTool, toolContext *OpenAIWireToolContext) ([]dto.ToolCallRequest, error) {
	namespace := strings.TrimSpace(tool.Name)
	if namespace == "" {
		return nil, fmt.Errorf("tools[%d].name is required for namespace tool", index)
	}
	childrenRaw := tool.Tools
	if len(childrenRaw) == 0 {
		childrenRaw = tool.Children
	}
	if len(childrenRaw) == 0 {
		return nil, fmt.Errorf("tools[%d].tools is required for namespace tool", index)
	}
	var children []openAIResponsesFunctionTool
	if err := common.Unmarshal(childrenRaw, &children); err != nil {
		return nil, fmt.Errorf("unmarshal tools[%d].tools failed: %w", index, err)
	}
	out := make([]dto.ToolCallRequest, 0, len(children))
	for childIndex, child := range children {
		if strings.EqualFold(strings.TrimSpace(child.Type), openAIResponsesToolTypeNamespace) {
			return nil, fmt.Errorf("tools[%d].tools[%d].type %q is not supported for chat.completions conversion", index, childIndex, child.Type)
		}
		items, err := convertOneResponsesToolToChatTools(childIndex, child, namespace, toolContext)
		if err != nil {
			return nil, fmt.Errorf("tools[%d].tools[%d]: %w", index, childIndex, err)
		}
		out = append(out, items...)
	}
	return out, nil
}

func newResponsesToolSearchChatTool() dto.ToolCallRequest {
	return dto.ToolCallRequest{
		Type: "function",
		Function: dto.FunctionRequest{
			Name:        openAIResponsesToolSearchChatName,
			Description: "Search and load Codex tools, plugins, connectors, and MCP namespaces for the current task.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query for tools or connectors to load.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of tool groups to return.",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func buildResponsesCustomToolChatDescription(tool openAIResponsesFunctionTool) string {
	description := strings.TrimSpace(tool.Description)
	if len(tool.Format) == 0 {
		return description
	}
	var format any
	if err := common.Unmarshal(tool.Format, &format); err != nil {
		return description
	}
	formatRaw, err := common.Marshal(format)
	if err != nil {
		return description
	}
	if description != "" {
		description += "\n\n"
	}
	return description + "Original custom tool format:\n" + string(formatRaw)
}

func flattenOpenAIResponsesNamespaceToolName(namespace string, name string) string {
	fullName := strings.TrimSpace(namespace) + "__" + strings.TrimSpace(name)
	if len(fullName) <= openAIResponsesChatToolNameMaxLen {
		return fullName
	}
	sum := sha256.Sum256([]byte(fullName))
	suffix := fmt.Sprintf("__%x", sum[:4])
	prefixLen := openAIResponsesChatToolNameMaxLen - len(suffix)
	if prefixLen < 0 {
		prefixLen = 0
	}
	return trimStringByBytes(fullName, prefixLen) + suffix
}

func trimStringByBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	var out strings.Builder
	for _, r := range s {
		if out.Len()+len(string(r)) > maxBytes {
			break
		}
		out.WriteRune(r)
	}
	return out.String()
}
