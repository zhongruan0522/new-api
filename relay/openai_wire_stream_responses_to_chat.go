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

	toolCallIndexByID map[string]int
	toolCallNameByID  map[string]string

	err error
}

func newResponsesToChatStreamConverter(includeUsage bool) *responsesToChatStreamConverter {
	return &responsesToChatStreamConverter{
		includeUsage:      includeUsage,
		toolCallIndexByID: make(map[string]int),
		toolCallNameByID:  make(map[string]string),
		sentRole:          false,
		sawToolCalls:      false,
		created:           0,
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
	case "response.output_text.delta":
		return c.emitTextDelta(stream.Delta)
	case "response.output_item.added":
		c.captureToolCallMeta(stream)
		return "", nil
	case "response.function_call_arguments.delta":
		return c.emitToolCallDelta(stream)
	case "response.output_item.done":
		c.captureToolCallMeta(stream)
		return "", nil
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

func (c *responsesToChatStreamConverter) emitToolCallDelta(stream dto.ResponsesStreamResponse) (string, error) {
	callID := strings.TrimSpace(stream.ItemID)
	if callID == "" {
		return "", nil
	}
	delta := stream.Delta
	if strings.TrimSpace(delta) == "" {
		return "", nil
	}
	c.sawToolCalls = true

	idx := c.getToolCallIndex(callID)
	name := c.toolCallNameByID[callID]

	chunk := c.newChatChunk()
	choice := dto.ChatCompletionsStreamResponseChoice{
		Index: 0,
		Delta: dto.ChatCompletionsStreamResponseChoiceDelta{},
	}
	if !c.sentRole {
		choice.Delta.Role = "assistant"
		c.sentRole = true
	}
	choice.Delta.ToolCalls = []dto.ToolCallResponse{
		{
			Index: common.GetPointer(idx),
			ID:    callID,
			Type:  "function",
			Function: dto.FunctionResponse{
				Name:      name,
				Arguments: delta,
			},
		},
	}
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}

	return encodeChatSSEChunk(chunk)
}

func (c *responsesToChatStreamConverter) emitFinal() (string, error) {
	finishReason := "stop"
	if c.sawToolCalls {
		finishReason = "tool_calls"
	}

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
