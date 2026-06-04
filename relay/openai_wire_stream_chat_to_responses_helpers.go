package relay

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
)

func (c *chatToResponsesStreamConverter) hydrateFromChunk(chunk *dto.ChatCompletionsStreamResponse) {
	if chunk == nil {
		return
	}
	if c.id == "" && strings.TrimSpace(chunk.Id) != "" {
		c.id = chunk.Id
	}
	if c.model == "" && strings.TrimSpace(chunk.Model) != "" {
		c.model = chunk.Model
	}
	if c.created == 0 && chunk.Created != 0 {
		c.created = chunk.Created
	}
	if chunk.Usage != nil {
		c.usage = chunk.Usage
	}
}

func (c *chatToResponsesStreamConverter) buildOutput() ([]dto.ResponsesOutput, error) {
	output := make([]dto.ResponsesOutput, 0, 2+len(c.toolCallsByID))
	reasoning := strings.TrimSpace(c.reasoningBuilder.String())
	if reasoning != "" {
		output = append(output, dto.ResponsesOutput{
			Type:   "reasoning",
			ID:     chatToResponsesReasoningItemID,
			Status: "completed",
			Summary: []dto.ResponsesContentPart{{
				Type: "summary_text",
				Text: reasoning,
			}},
		})
	}
	text := strings.TrimSpace(c.textBuilder.String())
	if text != "" {
		output = append(output, dto.ResponsesOutput{
			Type:   "message",
			ID:     chatToResponsesAssistantMessageID,
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{Type: "output_text", Text: text},
			},
		})
	}
	for _, callID := range c.toolCallOrder {
		state := c.toolCallsByID[callID]
		if state == nil {
			continue
		}
		item, err := state.responsesOutputItem("completed")
		if err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, nil
}

func (s *chatToResponsesToolCallState) responsesArguments() (string, error) {
	if s == nil {
		return "", nil
	}
	raw := s.args.String()
	if s.customProxy {
		input, complete := relaycommon.ExtractResponsesCustomToolInputFromChatArguments(raw)
		if !complete {
			return "", fmt.Errorf("custom tool proxy %q arguments must contain a complete %q string", s.name, "input")
		}
		return input, nil
	}
	return raw, nil
}

func (s *chatToResponsesToolCallState) applyToolSpec(spec relaycommon.OpenAIWireToolSpec) {
	if s == nil {
		return
	}
	s.toolSpec = spec
	s.hasToolSpec = true
	s.name = spec.Name
	s.namespace = spec.Namespace
	s.customProxy = spec.IsCustom()
	if spec.IsCustom() {
		s.toolType = dto.CustomType
		return
	}
	s.toolType = "function"
}

func (s *chatToResponsesToolCallState) isCustomTool() bool {
	if s == nil {
		return false
	}
	return s.toolType == dto.CustomType || (s.hasToolSpec && s.toolSpec.IsCustom())
}

func (s *chatToResponsesToolCallState) isToolSearch() bool {
	return s != nil && s.hasToolSpec && s.toolSpec.IsToolSearch()
}

func (s *chatToResponsesToolCallState) responsesOutputItem(status string) (dto.ResponsesOutput, error) {
	if s == nil {
		return dto.ResponsesOutput{}, fmt.Errorf("tool call state is nil")
	}
	if s.isToolSearch() {
		item := dto.ResponsesOutput{
			Type:      "tool_search_call",
			ID:        s.id,
			Status:    status,
			CallId:    s.id,
			Execution: "client",
		}
		if status == "completed" {
			arguments, err := relaycommon.BuildResponsesToolSearchArgumentsFromChatArguments(s.args.String())
			if err != nil {
				return dto.ResponsesOutput{}, fmt.Errorf("parse tool_search arguments failed: %w", err)
			}
			item.Arguments = arguments
		}
		return item, nil
	}

	item := dto.ResponsesOutput{
		Type:      "function_call",
		ID:        s.id,
		Status:    status,
		Role:      "assistant",
		CallId:    s.id,
		Name:      s.name,
		Namespace: s.namespace,
	}
	if s.isCustomTool() {
		item.Type = "custom_tool_call"
		if status == "completed" {
			input, err := s.responsesArguments()
			if err != nil {
				return dto.ResponsesOutput{}, err
			}
			item.Input = input
		}
		return item, nil
	}
	if status == "completed" {
		arguments, err := s.responsesArguments()
		if err != nil {
			return dto.ResponsesOutput{}, err
		}
		item.Arguments = arguments
	}
	return item, nil
}

func (c *chatToResponsesStreamConverter) buildUsage() *dto.Usage {
	if c.usage == nil {
		return nil
	}
	return relaycommon.MapChatUsageToResponsesUsage(*c.usage)
}

func (c *chatToResponsesStreamConverter) mapStatus() string {
	switch strings.TrimSpace(strings.ToLower(c.finishReason)) {
	case "length":
		return "incomplete"
	case "error":
		return "failed"
	default:
		return "completed"
	}
}

func (c *chatToResponsesStreamConverter) ensureID() string {
	if c.id == "" {
		c.id = "resp-" + common.GetRandomString(12)
	}
	return c.id
}

func (c *chatToResponsesStreamConverter) ensureCreated() int64 {
	if c.created == 0 {
		c.created = time.Now().Unix()
	}
	return c.created
}

func (c *chatToResponsesStreamConverter) getOrCreateToolCall(callID string) *chatToResponsesToolCallState {
	if existing, ok := c.toolCallsByID[callID]; ok {
		return existing
	}
	state := &chatToResponsesToolCallState{
		id:    callID,
		index: len(c.toolCallsByID),
	}
	c.toolCallsByID[callID] = state
	c.toolCallOrder = append(c.toolCallOrder, callID)
	return state
}

func (c *chatToResponsesStreamConverter) resolveToolCallID(call dto.ToolCallResponse) (string, bool) {
	callID := strings.TrimSpace(call.ID)
	if call.Index != nil {
		index := *call.Index
		if callID != "" {
			if mapped := c.toolCallIDByIndex[index]; mapped != "" && mapped != callID {
				c.rekeyToolCallState(mapped, callID)
			}
			c.toolCallIDByIndex[index] = callID
			return callID, true
		}
		if mapped := c.toolCallIDByIndex[index]; mapped != "" {
			return mapped, false
		}
		callID = fmt.Sprintf("call_%d", index)
		c.toolCallIDByIndex[index] = callID
		return callID, false
	}
	if callID != "" {
		return callID, true
	}
	return fmt.Sprintf("call_%d", len(c.toolCallsByID)), false
}

func (c *chatToResponsesStreamConverter) rekeyToolCallState(from string, to string) {
	if from == "" || to == "" || from == to {
		return
	}
	state := c.toolCallsByID[from]
	if state == nil {
		return
	}
	if existing := c.toolCallsByID[to]; existing != nil {
		existing.hasStableID = true
		if existing.toolType == "" {
			existing.toolType = state.toolType
		}
		if existing.name == "" {
			existing.name = state.name
		}
		if existing.namespace == "" {
			existing.namespace = state.namespace
		}
		if !existing.hasToolSpec && state.hasToolSpec {
			existing.toolSpec = state.toolSpec
			existing.hasToolSpec = true
		}
		existing.customProxy = existing.customProxy || state.customProxy
		var args strings.Builder
		args.WriteString(state.args.String())
		args.WriteString(existing.args.String())
		existing.args = args
		if existing.emittedArgsLen < state.emittedArgsLen {
			existing.emittedArgsLen = state.emittedArgsLen
		}
		existing.sentAdded = existing.sentAdded || state.sentAdded
		delete(c.toolCallsByID, from)
		c.removeToolCallOrder(from)
		return
	}
	state.id = to
	state.hasStableID = true
	c.toolCallsByID[to] = state
	delete(c.toolCallsByID, from)
	for i, callID := range c.toolCallOrder {
		if callID == from {
			c.toolCallOrder[i] = to
			return
		}
	}
}

func (c *chatToResponsesStreamConverter) removeToolCallOrder(callID string) {
	for i, current := range c.toolCallOrder {
		if current == callID {
			c.toolCallOrder = append(c.toolCallOrder[:i], c.toolCallOrder[i+1:]...)
			return
		}
	}
}

func (c *chatToResponsesStreamConverter) emitMessageDoneIfAny() (string, error) {
	text := strings.TrimSpace(c.textBuilder.String())
	if text == "" {
		return "", nil
	}
	if !c.sentMsgAdded {
		return "", nil
	}
	return encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type: "response.output_item.done",
		Item: &dto.ResponsesOutput{
			Type:   "message",
			ID:     chatToResponsesAssistantMessageID,
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{Type: "output_text", Text: text},
			},
		},
		ItemID: chatToResponsesAssistantMessageID,
	})
}

// emitReasoningDoneIfAny closes the reasoning item once text/tool output starts
// or the chat stream finishes.
func (c *chatToResponsesStreamConverter) emitReasoningDoneIfAny() (string, error) {
	reasoning := strings.TrimSpace(c.reasoningBuilder.String())
	if reasoning == "" || !c.sentReasoningAdded || c.reasoningDone {
		return "", nil
	}
	c.reasoningDone = true
	return encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type: "response.output_item.done",
		Item: &dto.ResponsesOutput{
			Type:   "reasoning",
			ID:     chatToResponsesReasoningItemID,
			Status: "completed",
			Summary: []dto.ResponsesContentPart{{
				Type: "summary_text",
				Text: reasoning,
			}},
		},
		ItemID: chatToResponsesReasoningItemID,
	})
}

func encodeResponsesStreamEvent(stream dto.ResponsesStreamResponse) (string, error) {
	if strings.TrimSpace(stream.Type) == "" {
		return "", nil
	}
	raw, err := common.Marshal(stream)
	if err != nil {
		return "", fmt.Errorf("marshal responses stream event failed: %w", err)
	}
	return "event: " + stream.Type + "\n" + "data: " + string(raw) + "\n\n", nil
}
