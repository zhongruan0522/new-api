package relay

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
)

type responsesToChatStreamConverter struct {
	includeUsage bool

	id      string
	model   string
	created int64

	sentRole     bool
	sawToolCalls bool
	usage        *dto.Usage
	status       string

	toolCallIndexByID        map[string]int
	toolCallNameByID         map[string]string
	toolCallStartedByID      map[string]bool
	toolCallBufferedArgsByID map[string]string

	err error
}

func newResponsesToChatStreamConverter(includeUsage bool) *responsesToChatStreamConverter {
	return &responsesToChatStreamConverter{
		includeUsage:             includeUsage,
		toolCallIndexByID:        make(map[string]int),
		toolCallNameByID:         make(map[string]string),
		toolCallStartedByID:      make(map[string]bool),
		toolCallBufferedArgsByID: make(map[string]string),
		sentRole:                 false,
		sawToolCalls:             false,
		created:                  0,
	}
}

func (c *responsesToChatStreamConverter) Err() error {
	return c.err
}

func (c *responsesToChatStreamConverter) ConvertFrame(event string, data string, rawFrame string) (string, error) {
	if c.err != nil {
		return "", c.err
	}
	if strings.HasPrefix(strings.TrimSpace(rawFrame), ":") {
		return rawFrame, nil
	}
	if strings.TrimSpace(data) == "" {
		return "", nil
	}

	var stream dto.ResponsesStreamResponse
	if err := common.UnmarshalJsonStr(data, &stream); err != nil {
		c.err = fmt.Errorf("unmarshal responses stream frame failed: %w", err)
		return "", c.err
	}
	if stream.Type != "" {
		event = stream.Type
	}

	c.hydrateFromResponse(stream.Response)

	switch event {
	case "response.created":
		return "", nil
	case "response.output_text.delta":
		return c.emitTextDelta(stream.Delta)
	case "response.reasoning_summary_text.delta":
		return c.emitReasoningDelta(stream.Delta)
	case "response.output_item.added":
		c.captureToolCallMeta(stream)
		return c.emitToolCallAdded(stream)
	case "response.function_call_arguments.delta":
		return c.emitToolCallDelta(stream)
	case "response.output_item.done":
		c.captureToolCallMeta(stream)
		return c.emitToolCallAdded(stream)
	case "response.incomplete":
		c.hydrateFromResponse(stream.Response)
		c.status = "incomplete"
		return c.emitFinal()
	case "response.failed":
		c.hydrateFromResponse(stream.Response)
		c.status = "failed"
		return c.emitFinal()
	case "response.completed":
		c.hydrateFromResponse(stream.Response)
		return c.emitFinal()
	default:
		return "", nil
	}
}

func (c *responsesToChatStreamConverter) captureToolCallMeta(stream dto.ResponsesStreamResponse) {
	if stream.Item == nil {
		return
	}
	if strings.TrimSpace(stream.Item.Type) != "function_call" {
		return
	}
	callID := strings.TrimSpace(stream.Item.CallId)
	if callID == "" {
		callID = strings.TrimSpace(stream.Item.ID)
	}
	if callID == "" {
		callID = strings.TrimSpace(stream.ItemID)
	}
	if callID == "" {
		return
	}
	if strings.TrimSpace(stream.Item.Name) != "" {
		c.toolCallNameByID[callID] = stream.Item.Name
	}
	c.sawToolCalls = true
}

func (c *responsesToChatStreamConverter) hydrateFromResponse(resp *dto.OpenAIResponsesResponse) {
	if resp == nil {
		return
	}
	if c.id == "" && strings.TrimSpace(resp.ID) != "" {
		c.id = resp.ID
	}
	if c.model == "" && strings.TrimSpace(resp.Model) != "" {
		c.model = resp.Model
	}
	if c.created == 0 && resp.CreatedAt != 0 {
		c.created = int64(resp.CreatedAt)
	}
	if resp.Usage != nil {
		u := &dto.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
		if u.TotalTokens == 0 {
			u.TotalTokens = u.PromptTokens + u.CompletionTokens
		}
		if resp.Usage.InputTokensDetails != nil {
			u.PromptTokensDetails.CachedTokens = resp.Usage.InputTokensDetails.CachedTokens
		}
		c.usage = u
	}
	if strings.TrimSpace(resp.Status) != "" {
		c.status = resp.Status
	}
	if len(resp.Output) > 0 {
		for _, item := range resp.Output {
			if strings.TrimSpace(item.Type) == "function_call" {
				c.sawToolCalls = true
				break
			}
		}
	}
}

func (c *responsesToChatStreamConverter) emitTextDelta(delta string) (string, error) {
	if strings.TrimSpace(delta) == "" {
		return "", nil
	}

	chunk := c.newChatChunk()
	choice := dto.ChatCompletionsStreamResponseChoice{
		Index: 0,
		Delta: dto.ChatCompletionsStreamResponseChoiceDelta{},
	}
	if !c.sentRole {
		choice.Delta.Role = "assistant"
		c.sentRole = true
	}
	choice.Delta.Content = &delta
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}

	return encodeChatSSEChunk(chunk)
}

// emitReasoningDelta maps Responses reasoning summaries onto chat chunk
// reasoning_content so downstream clients keep intermediate thinking text.
func (c *responsesToChatStreamConverter) emitReasoningDelta(delta string) (string, error) {
	if strings.TrimSpace(delta) == "" {
		return "", nil
	}

	chunk := c.newChatChunk()
	choice := dto.ChatCompletionsStreamResponseChoice{
		Index: 0,
		Delta: dto.ChatCompletionsStreamResponseChoiceDelta{},
	}
	if !c.sentRole {
		choice.Delta.Role = "assistant"
		c.sentRole = true
	}
	choice.Delta.SetReasoningContent(delta)
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}
	return encodeChatSSEChunk(chunk)
}

func (c *responsesToChatStreamConverter) emitToolCallAdded(stream dto.ResponsesStreamResponse) (string, error) {
	callID, name, ok := c.getToolCallMeta(stream)
	if !ok {
		return "", nil
	}
	if strings.TrimSpace(name) == "" {
		return "", nil
	}
	if c.toolCallStartedByID[callID] {
		return "", nil
	}
	c.toolCallStartedByID[callID] = true
	c.sawToolCalls = true
	idx := c.getToolCallIndex(callID)

	chunk := c.newChatChunk()
	choice := dto.ChatCompletionsStreamResponseChoice{
		Index: 0,
		Delta: dto.ChatCompletionsStreamResponseChoiceDelta{},
	}
	if !c.sentRole {
		choice.Delta.Role = "assistant"
		c.sentRole = true
	}
	choice.Delta.ToolCalls = []dto.ToolCallResponse{{
		Index: common.GetPointer(idx),
		ID:    callID,
		Type:  "function",
		Function: dto.FunctionResponse{
			Name: name,
		},
	}}
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}
	frame, err := encodeChatSSEChunk(chunk)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(frame)
	if buffered := c.toolCallBufferedArgsByID[callID]; strings.TrimSpace(buffered) != "" {
		deltaFrame, err := c.emitStartedToolCallArguments(callID, idx, name, buffered)
		if err != nil {
			return "", err
		}
		builder.WriteString(deltaFrame)
		delete(c.toolCallBufferedArgsByID, callID)
	}
	return builder.String(), nil
}

func (c *responsesToChatStreamConverter) emitToolCallDelta(stream dto.ResponsesStreamResponse) (string, error) {
	callID, name, ok := c.getToolCallMeta(stream)
	if !ok {
		return "", nil
	}
	delta := stream.Delta
	if strings.TrimSpace(delta) == "" {
		return "", nil
	}
	c.sawToolCalls = true

	if !c.toolCallStartedByID[callID] {
		c.toolCallBufferedArgsByID[callID] += delta
		if strings.TrimSpace(name) == "" {
			return "", nil
		}
		return c.emitToolCallAdded(stream)
	}

	idx := c.getToolCallIndex(callID)
	frame, err := c.emitStartedToolCallArguments(callID, idx, name, delta)
	if err != nil {
		return "", err
	}
	return frame, nil
}

func (c *responsesToChatStreamConverter) emitFinal() (string, error) {
	finishReason := c.finishReason()

	stop := c.newChatChunk()
	stop.Choices = []dto.ChatCompletionsStreamResponseChoice{
		{
			Index:        0,
			FinishReason: &finishReason,
		},
	}

	var builder strings.Builder
	stopFrame, err := encodeChatSSEChunk(stop)
	if err != nil {
		return "", err
	}
	builder.WriteString(stopFrame)

	if c.includeUsage && c.usage != nil {
		usageChunk := c.newChatChunk()
		usageChunk.Choices = make([]dto.ChatCompletionsStreamResponseChoice, 0)
		usageChunk.Usage = c.usage
		frame, err := encodeChatSSEChunk(usageChunk)
		if err != nil {
			return "", err
		}
		builder.WriteString(frame)
	}

	builder.WriteString("data: [DONE]\n\n")
	return builder.String(), nil
}

func (c *responsesToChatStreamConverter) finishReason() string {
	switch strings.ToLower(strings.TrimSpace(c.status)) {
	case "failed":
		return "error"
	case "incomplete":
		return "length"
	default:
		if c.sawToolCalls {
			return "tool_calls"
		}
		return "stop"
	}
}

func (c *responsesToChatStreamConverter) newChatChunk() *dto.ChatCompletionsStreamResponse {
	if c.id == "" {
		c.id = "chatcmpl-" + common.GetRandomString(12)
	}
	if c.created == 0 {
		c.created = time.Now().Unix()
	}
	return &dto.ChatCompletionsStreamResponse{
		Id:      c.id,
		Object:  "chat.completion.chunk",
		Created: c.created,
		Model:   c.model,
	}
}

func (c *responsesToChatStreamConverter) getToolCallIndex(callID string) int {
	if idx, ok := c.toolCallIndexByID[callID]; ok {
		return idx
	}
	idx := len(c.toolCallIndexByID)
	c.toolCallIndexByID[callID] = idx
	return idx
}

func (c *responsesToChatStreamConverter) emitStartedToolCallArguments(callID string, idx int, name string, delta string) (string, error) {
	if strings.TrimSpace(delta) == "" {
		return "", nil
	}
	chunk := c.newChatChunk()
	choice := dto.ChatCompletionsStreamResponseChoice{
		Index: 0,
		Delta: dto.ChatCompletionsStreamResponseChoiceDelta{},
	}
	if !c.sentRole {
		choice.Delta.Role = "assistant"
		c.sentRole = true
	}
	choice.Delta.ToolCalls = []dto.ToolCallResponse{{
		Index: common.GetPointer(idx),
		Function: dto.FunctionResponse{
			Name:      name,
			Arguments: delta,
		},
	}}
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}
	return encodeChatSSEChunk(chunk)
}

func (c *responsesToChatStreamConverter) getToolCallMeta(stream dto.ResponsesStreamResponse) (string, string, bool) {
	callID := strings.TrimSpace(stream.ItemID)
	if callID == "" && stream.Item != nil {
		callID = strings.TrimSpace(stream.Item.CallId)
		if callID == "" {
			callID = strings.TrimSpace(stream.Item.ID)
		}
	}
	if callID == "" {
		return "", "", false
	}
	name := c.toolCallNameByID[callID]
	if stream.Item != nil && strings.TrimSpace(stream.Item.Name) != "" {
		name = stream.Item.Name
		c.toolCallNameByID[callID] = name
	}
	return callID, name, true
}

func encodeChatSSEChunk(chunk *dto.ChatCompletionsStreamResponse) (string, error) {
	if chunk == nil {
		return "", nil
	}
	raw, err := common.Marshal(chunk)
	if err != nil {
		return "", fmt.Errorf("marshal chat chunk failed: %w", err)
	}
	return "data: " + string(raw) + "\n\n", nil
}
