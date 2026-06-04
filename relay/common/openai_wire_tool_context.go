package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
)

// OpenAIWireToolContext carries per-request metadata needed to undo tool
// proxying after a Chat-only upstream responds.
type OpenAIWireToolContext struct {
	toolProxies map[string]OpenAIWireToolSpec
}

type OpenAIWireToolSpec struct {
	Type      string
	Name      string
	Namespace string
}

func (s OpenAIWireToolSpec) IsCustom() bool {
	return s.Type == openAIResponsesToolTypeCustom
}

func (s OpenAIWireToolSpec) IsFunction() bool {
	return s.Type == openAIResponsesToolTypeFunction
}

func (s OpenAIWireToolSpec) IsToolSearch() bool {
	return s.Type == openAIResponsesToolTypeToolSearch
}

func NewOpenAIWireToolContext() *OpenAIWireToolContext {
	return &OpenAIWireToolContext{toolProxies: make(map[string]OpenAIWireToolSpec)}
}

func (c *OpenAIWireToolContext) AddFunctionToolProxy(proxyName string, responsesName string, namespace string) {
	c.addToolProxy(proxyName, OpenAIWireToolSpec{
		Type:      openAIResponsesToolTypeFunction,
		Name:      responsesName,
		Namespace: namespace,
	})
}

func (c *OpenAIWireToolContext) AddCustomToolProxy(proxyName string, responsesName string, namespace ...string) {
	spec := OpenAIWireToolSpec{
		Type: openAIResponsesToolTypeCustom,
		Name: responsesName,
	}
	if len(namespace) > 0 {
		spec.Namespace = namespace[0]
	}
	c.addToolProxy(proxyName, spec)
}

func (c *OpenAIWireToolContext) AddToolSearchProxy(proxyName string) {
	c.addToolProxy(proxyName, OpenAIWireToolSpec{
		Type: openAIResponsesToolTypeToolSearch,
		Name: openAIResponsesToolSearchChatName,
	})
}

func (c *OpenAIWireToolContext) addToolProxy(proxyName string, spec OpenAIWireToolSpec) {
	if c == nil {
		return
	}
	proxyName = strings.TrimSpace(proxyName)
	spec.Type = strings.TrimSpace(spec.Type)
	spec.Name = strings.TrimSpace(spec.Name)
	spec.Namespace = strings.TrimSpace(spec.Namespace)
	if proxyName == "" {
		return
	}
	if spec.Type == "" {
		spec.Type = openAIResponsesToolTypeFunction
	}
	if spec.Name == "" {
		spec.Name = proxyName
	}
	if c.toolProxies == nil {
		c.toolProxies = make(map[string]OpenAIWireToolSpec)
	}
	c.toolProxies[proxyName] = spec
}

func (c *OpenAIWireToolContext) ResolveToolProxy(proxyName string) (OpenAIWireToolSpec, bool) {
	if c == nil || len(c.toolProxies) == 0 {
		return OpenAIWireToolSpec{}, false
	}
	spec, ok := c.toolProxies[strings.TrimSpace(proxyName)]
	if !ok {
		return OpenAIWireToolSpec{}, false
	}
	return spec, true
}

func (c *OpenAIWireToolContext) ResolveCustomToolProxy(proxyName string) (string, bool) {
	spec, ok := c.ResolveToolProxy(proxyName)
	if !ok || spec.Type != openAIResponsesToolTypeCustom {
		return "", false
	}
	return spec.Name, true
}

func (c *OpenAIWireToolContext) HasCustomToolProxies() bool {
	if c == nil {
		return false
	}
	for _, spec := range c.toolProxies {
		if spec.Type == openAIResponsesToolTypeCustom {
			return true
		}
	}
	return false
}

func BuildChatArgumentsForResponsesCustomToolInput(input string) (string, error) {
	raw, err := common.Marshal(map[string]any{openAIResponsesCustomInputField: input})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func ExtractResponsesCustomToolInputFromChatArguments(arguments string) (input string, complete bool) {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return "", true
	}
	if trimmed[0] != '{' && trimmed[0] != '"' {
		return arguments, true
	}

	var payload map[string]json.RawMessage
	if err := common.Unmarshal([]byte(trimmed), &payload); err == nil {
		if raw := payload[openAIResponsesCustomInputField]; len(raw) > 0 {
			var s string
			if err := common.Unmarshal(raw, &s); err == nil {
				return s, true
			}
			return string(raw), true
		}
	}

	var s string
	if err := common.Unmarshal([]byte(trimmed), &s); err == nil {
		return s, true
	}
	return arguments, false
}

func BuildResponsesToolSearchArgumentsFromChatArguments(arguments string) (any, error) {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return map[string]any{}, nil
	}
	var payload any
	if err := common.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, err
	}
	if _, ok := payload.(map[string]any); !ok {
		return nil, fmt.Errorf("tool_search arguments must be a JSON object")
	}
	return payload, nil
}

func ResponsesArgumentsToChatString(arguments any) (string, error) {
	if arguments == nil {
		return "", nil
	}
	switch v := arguments.(type) {
	case string:
		return v, nil
	case json.RawMessage:
		if len(v) == 0 {
			return "", nil
		}
		if strings.TrimSpace(string(v)) == "" {
			return "", nil
		}
		var s string
		if err := common.Unmarshal(v, &s); err == nil {
			return s, nil
		}
		return string(v), nil
	default:
		raw, err := common.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
}
