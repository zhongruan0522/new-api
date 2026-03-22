package relay

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
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
	output := make([]dto.ResponsesOutput, 0, 1+len(c.toolCallsByID))
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
		output = append(output, dto.ResponsesOutput{
			Type:      "function_call",
			ID:        state.id,
			Status:    "completed",
			Role:      "assistant",
			CallId:    state.id,
			Name:      state.name,
			Arguments: state.args.String(),
		})
	}
	return output
}

func (c *chatToResponsesStreamConverter) buildUsage() *dto.Usage {
	if c.usage == nil {
		return nil
	}
	u := &dto.Usage{
		InputTokens:  c.usage.PromptTokens,
		OutputTokens: c.usage.CompletionTokens,
		TotalTokens:  c.usage.TotalTokens,
		InputTokensDetails: &dto.InputTokenDetails{
			CachedTokens: c.usage.PromptTokensDetails.CachedTokens,
		},
	}
	if u.TotalTokens == 0 {
		u.TotalTokens = u.InputTokens + u.OutputTokens
	}
	return u
}

func (c *chatToResponsesStreamConverter) mapStatus() string {
	switch strings.TrimSpace(strings.ToLower(c.finishReason)) {
	case "length":
		return "incomplete"
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
