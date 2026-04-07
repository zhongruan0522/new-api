package relay

import (
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

const chatToResponsesAssistantMessageID = "msg_0"
const chatToResponsesReasoningItemID = "rs_0"

type chatToResponsesStreamConverter struct {
	id      string
	model   string
	created int64

	sentCreated        bool
	sentInProgress     bool
	sentMsgAdded       bool
	sentReasoningAdded bool
	reasoningDone      bool
	finishReason       string
	reasoningBuilder   strings.Builder
	textBuilder        strings.Builder
	toolCallsByID      map[string]*chatToResponsesToolCallState
	toolCallOrder      []string

	usage *dto.Usage
	err   error
}

type chatToResponsesToolCallState struct {
	id             string
	index          int
	name           string
	args           strings.Builder
	emittedArgsLen int
	sentAdded      bool
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
	if !c.sentInProgress {
		frame, err := c.emitInProgress()
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
		c.sentInProgress = true
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
	if delta := strings.TrimSpace(choice.Delta.GetReasoningContent()); delta != "" {
		frame, err := c.emitReasoningDelta(delta)
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
	}
	if delta := strings.TrimSpace(choice.Delta.GetContentString()); delta != "" {
		if !c.reasoningDone {
			doneFrame, err := c.emitReasoningDoneIfAny()
			if err != nil {
				return "", err
			}
			out.WriteString(doneFrame)
		}
		frame, err := c.emitAssistantTextDelta(delta)
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
	}
	if len(choice.Delta.ToolCalls) > 0 {
		if !c.reasoningDone {
			doneFrame, err := c.emitReasoningDoneIfAny()
			if err != nil {
				return "", err
			}
			out.WriteString(doneFrame)
		}
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

func (c *chatToResponsesStreamConverter) emitInProgress() (string, error) {
	resp := &dto.OpenAIResponsesResponse{
		ID:        c.ensureID(),
		Object:    "response",
		CreatedAt: int(c.ensureCreated()),
		Status:    "in_progress",
		Model:     c.model,
		Output:    make([]dto.ResponsesOutput, 0),
	}
	return encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type:     "response.in_progress",
		Response: resp,
	})
}

// emitReasoningDelta preserves chat reasoning deltas as Responses reasoning
// summary events so Responses clients can continue consuming thinking streams.
func (c *chatToResponsesStreamConverter) emitReasoningDelta(delta string) (string, error) {
	c.reasoningBuilder.WriteString(delta)

	var out strings.Builder
	if !c.sentReasoningAdded {
		added, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
			Type: "response.output_item.added",
			Item: &dto.ResponsesOutput{
				Type:   "reasoning",
				ID:     chatToResponsesReasoningItemID,
				Status: "in_progress",
				Summary: []dto.ResponsesContentPart{{
					Type: "summary_text",
				}},
			},
			ItemID: chatToResponsesReasoningItemID,
		})
		if err != nil {
			return "", err
		}
		out.WriteString(added)
		c.sentReasoningAdded = true
	}

	frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type:   "response.reasoning_summary_text.delta",
		Delta:  delta,
		ItemID: chatToResponsesReasoningItemID,
	})
	if err != nil {
		return "", err
	}
	out.WriteString(frame)
	return out.String(), nil
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
		delta := call.Function.Arguments
		if strings.TrimSpace(delta) != "" {
			state.args.WriteString(delta)
		}

		if !state.sentAdded && strings.TrimSpace(state.name) != "" {
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

		if !state.sentAdded {
			continue
		}

		pendingArgs := state.args.String()
		if len(pendingArgs) > state.emittedArgsLen {
			delta = pendingArgs[state.emittedArgsLen:]
			frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
				Type:   "response.function_call_arguments.delta",
				Delta:  delta,
				ItemID: callID,
			})
			if err != nil {
				return "", err
			}
			out.WriteString(frame)
			state.emittedArgsLen = len(pendingArgs)
		}
	}
	return out.String(), nil
}

func (c *chatToResponsesStreamConverter) emitCompleted() (string, error) {
	reasoningDoneFrame, err := c.emitReasoningDoneIfAny()
	if err != nil {
		return "", err
	}
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
	out.WriteString(reasoningDoneFrame)
	out.WriteString(messageDoneFrame)
	for _, call := range c.toolCallOrder {
		state := c.toolCallsByID[call]
		if state == nil {
			continue
		}
		if strings.TrimSpace(state.name) == "" {
			return "", fmt.Errorf("tool call %q is missing function name", state.id)
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

	eventType := "response.completed"
	if resp.Status == "incomplete" {
		eventType = "response.incomplete"
	} else if resp.Status == "failed" {
		eventType = "response.failed"
	}
	frame, err := encodeResponsesStreamEvent(dto.ResponsesStreamResponse{
		Type:     eventType,
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
