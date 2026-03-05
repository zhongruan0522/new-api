package relay

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const chatToResponsesAssistantMessageID = "msg_0"

type chatToResponsesStreamConverter struct {
	id      string
	model   string
	created int64

	sentCreated   bool
	sentMsgAdded  bool
	finishReason  string
	textBuilder   strings.Builder
	toolCallsByID map[string]*chatToResponsesToolCallState
	toolCallOrder []string

	usage *dto.Usage
	err   error
}

type chatToResponsesToolCallState struct {
	id        string
	index     int
	name      string
	args      strings.Builder
	sentAdded bool
}

func newChatToResponsesStreamConverter() *chatToResponsesStreamConverter {
	return &chatToResponsesStreamConverter{
		toolCallsByID: make(map[string]*chatToResponsesToolCallState),
	}
}

func (c *chatToResponsesStreamConverter) Err() error {
	return c.err
}

func (c *chatToResponsesStreamConverter) ConvertFrame(event string, data string, rawFrame string) (string, error) {
	if c.err != nil {
		return "", c.err
	}
	if isSSECommentFrame(rawFrame) {
		return rawFrame, nil
	}
	data = strings.TrimSpace(data)
	if data == "" {
		return "", nil
	}
	if data == "[DONE]" {
		return c.emitCompleted()
	}

	chunk, err := c.parseChatChunk(data)
	if err != nil {
		return "", err
	}
	return c.convertChunk(chunk)
}

func (c *chatToResponsesStreamConverter) emitCreated() (string, error) {
	resp := &dto.OpenAIResponsesResponse{
		ID:        c.ensureID(),
		Object:    "response",
		CreatedAt: int(c.ensureCreated()),
		Status:    "in_progress",
		Model:     c.model,
		Output:    make([]dto.ResponsesOutput, 0),
	}
	return encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type:     "response.created",
		Response: resp,
	})
}

func (c *chatToResponsesStreamConverter) convertChunk(chunk *dto.ChatCompletionsStreamResponse) (string, error) {
	c.hydrateFromChunk(chunk)

	if chunk.Usage != nil && len(chunk.Choices) == 0 {
		c.usage = chunk.Usage
		return "", nil
	}

	var out strings.Builder
	if !c.sentCreated {
		frame, err := c.emitCreated()
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
		c.sentCreated = true
	}

	for _, choice := range chunk.Choices {
		frame, err := c.convertChoice(choice)
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
	}
	return out.String(), nil
}

func (c *chatToResponsesStreamConverter) parseChatChunk(data string) (*dto.ChatCompletionsStreamResponse, error) {
	var chunk dto.ChatCompletionsStreamResponse
	if err := common.UnmarshalJsonStr(data, &chunk); err != nil {
		c.err = fmt.Errorf("unmarshal chat stream chunk failed: %w", err)
		return nil, c.err
	}
	return &chunk, nil
}

func (c *chatToResponsesStreamConverter) convertChoice(choice dto.ChatCompletionsStreamResponseChoice) (string, error) {
	c.captureFinishReason(choice)

	var out strings.Builder
	if delta := strings.TrimSpace(choice.Delta.GetContentString()); delta != "" {
		frame, err := c.emitAssistantTextDelta(delta)
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
	}
	if len(choice.Delta.ToolCalls) > 0 {
		frames, err := c.emitToolCallDeltas(choice.Delta.ToolCalls)
		if err != nil {
			return "", err
		}
		out.WriteString(frames)
	}
	return out.String(), nil
}

func (c *chatToResponsesStreamConverter) captureFinishReason(choice dto.ChatCompletionsStreamResponseChoice) {
	if choice.FinishReason == nil {
		return
	}
	if strings.TrimSpace(*choice.FinishReason) == "" {
		return
	}
	c.finishReason = *choice.FinishReason
}

func (c *chatToResponsesStreamConverter) emitAssistantTextDelta(delta string) (string, error) {
	c.textBuilder.WriteString(delta)

	var out strings.Builder
	if !c.sentMsgAdded {
		added, err := c.emitMessageAdded()
		if err != nil {
			return "", err
		}
		out.WriteString(added)
		c.sentMsgAdded = true
	}

	frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type:   "response.output_text.delta",
		Delta:  delta,
		ItemID: chatToResponsesAssistantMessageID,
	})
	if err != nil {
		return "", err
	}
	out.WriteString(frame)
	return out.String(), nil
}

func (c *chatToResponsesStreamConverter) emitMessageAdded() (string, error) {
	return encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type: "response.output_item.added",
		Item: &dto.ResponsesOutput{
			Type:   "message",
			ID:     chatToResponsesAssistantMessageID,
			Status: "in_progress",
			Role:   "assistant",
		},
		ItemID: chatToResponsesAssistantMessageID,
	})
}

func (c *chatToResponsesStreamConverter) emitToolCallDeltas(calls []dto.ToolCallResponse) (string, error) {
	var out strings.Builder
	for _, call := range calls {
		callID := strings.TrimSpace(call.ID)
		if callID == "" {
			callID = fmt.Sprintf("call_%d", len(c.toolCallsByID))
		}

		state := c.getOrCreateToolCall(callID)
		if strings.TrimSpace(call.Function.Name) != "" {
			state.name = call.Function.Name
		}

		if !state.sentAdded {
			frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
				Type: "response.output_item.added",
				Item: &dto.ResponsesOutput{
					Type:   "function_call",
					ID:     callID,
					Status: "in_progress",
					Role:   "assistant",
					CallId: callID,
					Name:   state.name,
				},
				ItemID: callID,
			})
			if err != nil {
				return "", err
			}
			out.WriteString(frame)
			state.sentAdded = true
		}

		delta := call.Function.Arguments
		if strings.TrimSpace(delta) != "" {
			state.args.WriteString(delta)
			frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
				Type:   "response.function_call_arguments.delta",
				Delta:  delta,
				ItemID: callID,
			})
			if err != nil {
				return "", err
			}
			out.WriteString(frame)
		}
	}
	return out.String(), nil
}

func (c *chatToResponsesStreamConverter) emitCompleted() (string, error) {
	messageDoneFrame, err := c.emitMessageDoneIfAny()
	if err != nil {
		return "", err
	}

	resp := &dto.OpenAIResponsesResponse{
		ID:        c.ensureID(),
		Object:    "response",
		CreatedAt: int(c.ensureCreated()),
		Status:    c.mapStatus(),
		Model:     c.model,
		Output:    c.buildOutput(),
		Usage:     c.buildUsage(),
	}

	var out strings.Builder
	out.WriteString(messageDoneFrame)
	for _, call := range c.toolCallOrder {
		state := c.toolCallsByID[call]
		if state == nil {
			continue
		}
		frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
			Type: "response.output_item.done",
			Item: &dto.ResponsesOutput{
				Type:      "function_call",
				ID:        state.id,
				Status:    "completed",
				Role:      "assistant",
				CallId:    state.id,
				Name:      state.name,
				Arguments: state.args.String(),
			},
			ItemID: state.id,
		})
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
	}

	frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type:     "response.completed",
		Response: resp,
	})
	if err != nil {
		return "", err
	}
	out.WriteString(frame)
	return out.String(), nil
}

func isSSECommentFrame(rawFrame string) bool {
	return strings.HasPrefix(strings.TrimSpace(rawFrame), ":")
}
