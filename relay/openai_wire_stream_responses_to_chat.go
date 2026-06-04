package relay

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
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
	toolCallTypeByID         map[string]string
	toolCallNameByID         map[string]string
	toolCallIDByItemID       map[string]string
	toolCallHasStableIDByID  map[string]bool
	toolCallStartedByID      map[string]bool
	toolCallBufferedArgsByID map[string]string
	toolCallArgsByID         map[string]string

	err error
}

func newResponsesToChatStreamConverter(includeUsage bool) *responsesToChatStreamConverter {
	return &responsesToChatStreamConverter{
		includeUsage:             includeUsage,
		toolCallIndexByID:        make(map[string]int),
		toolCallTypeByID:         make(map[string]string),
		toolCallNameByID:         make(map[string]string),
		toolCallIDByItemID:       make(map[string]string),
		toolCallHasStableIDByID:  make(map[string]bool),
		toolCallStartedByID:      make(map[string]bool),
		toolCallBufferedArgsByID: make(map[string]string),
		toolCallArgsByID:         make(map[string]string),
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
	case "response.function_call_arguments.done":
		return c.emitToolCallDone(stream)
	case "response.custom_tool_call_input.delta":
		return c.emitToolCallDelta(stream)
	case "response.custom_tool_call_input.done":
		return c.emitToolCallDone(stream)
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
	c.rememberToolCallMeta(*stream.Item, stream.ItemID)
}

func (c *responsesToChatStreamConverter) rememberToolCallMeta(item dto.ResponsesOutput, eventItemID string) {
	itemType := strings.TrimSpace(item.Type)
	if itemType != "function_call" && itemType != "custom_tool_call" {
		return
	}
	callID := strings.TrimSpace(item.CallId)
	hasStableID := callID != ""
	if callID == "" {
		callID = strings.TrimSpace(item.ID)
	}
	if callID == "" {
		callID = strings.TrimSpace(eventItemID)
	}
	if callID == "" {
		return
	}
	if itemID := strings.TrimSpace(item.ID); itemID != "" {
		c.toolCallIDByItemID[itemID] = callID
		c.rekeyToolCallState(itemID, callID)
	}
	if itemID := strings.TrimSpace(eventItemID); itemID != "" {
		c.toolCallIDByItemID[itemID] = callID
		c.rekeyToolCallState(itemID, callID)
	}
	if strings.TrimSpace(item.Name) != "" {
		c.toolCallNameByID[callID] = item.Name
	}
	c.toolCallTypeByID[callID] = itemType
	if hasStableID {
		c.toolCallHasStableIDByID[callID] = true
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
		u := &dto.Usage{}
		relaycommon.ApplyResponsesUsageToChatUsage(u, resp.Usage)
		c.usage = u
	}
	if strings.TrimSpace(resp.Status) != "" {
		c.status = resp.Status
	}
	if len(resp.Output) > 0 {
		for _, item := range resp.Output {
			if strings.TrimSpace(item.Type) == "function_call" || strings.TrimSpace(item.Type) == "custom_tool_call" {
				c.rememberToolCallMeta(item, "")
				c.sawToolCalls = true
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
	return c.emitToolCallAddedByID(callID, name, false)
}

func (c *responsesToChatStreamConverter) emitToolCallAddedByID(callID string, name string, allowUnstableID bool) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil
	}
	if !allowUnstableID && !c.toolCallHasStableIDByID[callID] {
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
	toolCall, err := c.newChatToolCallAdded(callID, idx, name)
	if err != nil {
		return "", err
	}
	choice.Delta.ToolCalls = []dto.ToolCallResponse{toolCall}
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}
	frame, err := encodeChatSSEChunk(chunk)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(frame)
	if buffered := c.toolCallBufferedArgsByID[callID]; buffered != "" {
		deltaFrame, err := c.emitStartedToolCallArguments(callID, idx, name, buffered)
		if err != nil {
			return "", err
		}
		builder.WriteString(deltaFrame)
		delete(c.toolCallBufferedArgsByID, callID)
	}
	return builder.String(), nil
}

func (c *responsesToChatStreamConverter) newChatToolCallAdded(callID string, idx int, name string) (dto.ToolCallResponse, error) {
	if c.toolCallTypeByID[callID] == "custom_tool_call" {
		custom, err := common.Marshal(map[string]any{"name": name})
		if err != nil {
			return dto.ToolCallResponse{}, fmt.Errorf("marshal custom tool call failed: %w", err)
		}
		return dto.ToolCallResponse{
			Index:  common.GetPointer(idx),
			ID:     callID,
			Type:   dto.CustomType,
			Custom: custom,
		}, nil
	}
	return dto.ToolCallResponse{
		Index: common.GetPointer(idx),
		ID:    callID,
		Type:  "function",
		Function: dto.FunctionResponse{
			Name: name,
		},
	}, nil
}

func (c *responsesToChatStreamConverter) emitToolCallDelta(stream dto.ResponsesStreamResponse) (string, error) {
	callID, name, ok := c.getToolCallMeta(stream)
	if !ok {
		return "", nil
	}
	delta := stream.Delta
	if delta == "" {
		return "", nil
	}
	c.sawToolCalls = true
	c.toolCallArgsByID[callID] += delta

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

func (c *responsesToChatStreamConverter) emitToolCallDone(stream dto.ResponsesStreamResponse) (string, error) {
	callID, name, ok := c.getToolCallMeta(stream)
	if !ok {
		return "", nil
	}
	arguments, err := relaycommon.ResponsesArgumentsToChatString(stream.Arguments)
	if err != nil {
		return "", fmt.Errorf("marshal function_call arguments failed: %w", err)
	}
	if c.toolCallTypeByID[callID] == "custom_tool_call" {
		arguments = stream.Input
	}
	if arguments == "" && stream.Item != nil {
		if c.toolCallTypeByID[callID] == "custom_tool_call" {
			arguments = stream.Item.Input
		} else {
			arguments, err = relaycommon.ResponsesArgumentsToChatString(stream.Item.Arguments)
			if err != nil {
				return "", fmt.Errorf("marshal function_call item arguments failed: %w", err)
			}
		}
	}
	if arguments == "" {
		return "", nil
	}

	if !c.toolCallStartedByID[callID] {
		c.toolCallBufferedArgsByID[callID] = arguments
		if strings.TrimSpace(name) == "" {
			return "", nil
		}
		return c.emitToolCallAdded(stream)
	}

	emitted := c.toolCallArgsByID[callID]
	remaining := arguments
	if strings.HasPrefix(arguments, emitted) {
		remaining = arguments[len(emitted):]
	}
	if remaining == "" {
		return "", nil
	}
	c.toolCallArgsByID[callID] += remaining
	idx := c.getToolCallIndex(callID)
	return c.emitStartedToolCallArguments(callID, idx, name, remaining)
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
	pendingToolCalls, err := c.emitPendingToolCalls()
	if err != nil {
		return "", err
	}
	builder.WriteString(pendingToolCalls)
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

func (c *responsesToChatStreamConverter) emitPendingToolCalls() (string, error) {
	callIDs := make([]string, 0, len(c.toolCallNameByID))
	for callID, name := range c.toolCallNameByID {
		if c.toolCallStartedByID[callID] || strings.TrimSpace(name) == "" {
			continue
		}
		callIDs = append(callIDs, callID)
	}
	sort.Strings(callIDs)

	var out strings.Builder
	for _, callID := range callIDs {
		frame, err := c.emitToolCallAddedByID(callID, c.toolCallNameByID[callID], true)
		if err != nil {
			return "", err
		}
		out.WriteString(frame)
	}
	return out.String(), nil
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

func (c *responsesToChatStreamConverter) rekeyToolCallState(from string, to string) {
	if from == "" || to == "" || from == to {
		return
	}
	if value, ok := c.toolCallBufferedArgsByID[from]; ok {
		c.toolCallBufferedArgsByID[to] = value + c.toolCallBufferedArgsByID[to]
		delete(c.toolCallBufferedArgsByID, from)
	}
	if value, ok := c.toolCallArgsByID[from]; ok {
		c.toolCallArgsByID[to] = value + c.toolCallArgsByID[to]
		delete(c.toolCallArgsByID, from)
	}
	if c.toolCallStartedByID[from] {
		c.toolCallStartedByID[to] = true
		delete(c.toolCallStartedByID, from)
	}
	if c.toolCallHasStableIDByID[from] {
		c.toolCallHasStableIDByID[to] = true
		delete(c.toolCallHasStableIDByID, from)
	}
	if _, ok := c.toolCallIndexByID[to]; !ok {
		if value, oldOK := c.toolCallIndexByID[from]; oldOK {
			c.toolCallIndexByID[to] = value
		}
	}
	delete(c.toolCallIndexByID, from)
	if c.toolCallNameByID[to] == "" {
		c.toolCallNameByID[to] = c.toolCallNameByID[from]
	}
	delete(c.toolCallNameByID, from)
	if c.toolCallTypeByID[to] == "" {
		c.toolCallTypeByID[to] = c.toolCallTypeByID[from]
	}
	delete(c.toolCallTypeByID, from)
}

func (c *responsesToChatStreamConverter) emitStartedToolCallArguments(callID string, idx int, name string, delta string) (string, error) {
	if delta == "" {
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
	toolCall := dto.ToolCallResponse{
		Index: common.GetPointer(idx),
		Function: dto.FunctionResponse{
			Name:      name,
			Arguments: delta,
		},
	}
	if c.toolCallTypeByID[callID] == "custom_tool_call" {
		custom, err := common.Marshal(map[string]any{"input": delta})
		if err != nil {
			return "", fmt.Errorf("marshal custom tool call input delta failed: %w", err)
		}
		toolCall = dto.ToolCallResponse{
			Index:  common.GetPointer(idx),
			Type:   dto.CustomType,
			Custom: custom,
		}
	}
	choice.Delta.ToolCalls = []dto.ToolCallResponse{toolCall}
	chunk.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}
	return encodeChatSSEChunk(chunk)
}

func (c *responsesToChatStreamConverter) getToolCallMeta(stream dto.ResponsesStreamResponse) (string, string, bool) {
	if stream.Item != nil {
		c.rememberToolCallMeta(*stream.Item, stream.ItemID)
	}
	itemID := strings.TrimSpace(stream.ItemID)
	callID := c.toolCallIDByItemID[itemID]
	itemType := ""
	if stream.Item != nil {
		if currentType := strings.TrimSpace(stream.Item.Type); currentType == "function_call" || currentType == "custom_tool_call" {
			itemType = currentType
		}
		if strings.TrimSpace(stream.Item.CallId) != "" {
			callID = strings.TrimSpace(stream.Item.CallId)
		} else if callID == "" {
			callID = c.toolCallIDByItemID[strings.TrimSpace(stream.Item.ID)]
		}
		if callID == "" {
			callID = strings.TrimSpace(stream.Item.ID)
		}
	}
	if callID == "" {
		callID = itemID
	}
	if callID == "" {
		return "", "", false
	}
	if strings.HasPrefix(stream.Type, "response.custom_tool_call_input.") {
		itemType = "custom_tool_call"
	} else if strings.HasPrefix(stream.Type, "response.function_call_arguments.") {
		itemType = "function_call"
	}
	if itemType != "" {
		c.toolCallTypeByID[callID] = itemType
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
