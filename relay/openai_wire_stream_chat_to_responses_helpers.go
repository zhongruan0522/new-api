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

func (c *chatToResponsesStreamConverter) buildOutput() []dto.ResponsesOutput {
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
		item := dto.ResponsesOutput{
			Type:   "function_call",
			ID:     state.id,
			Status: "completed",
			Role:   "assistant",
			CallId: state.id,
			Name:   state.name,
		}
		if state.toolType == dto.CustomType {
			item.Type = "custom_tool_call"
			item.Input = state.args.String()
		} else {
			item.Arguments = state.args.String()
		}
		output = append(output, item)
	}
	return output
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
