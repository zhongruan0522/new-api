package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/relay/channel/openrouter"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/relay/reasonmap"
)

func claudeThinkingBudgetToReasoningEffort(budget int) string {
	switch {
	case budget <= 0:
		return ""
	case budget <= 1280:
		return "low"
	case budget <= 2048:
		return "medium"
	default:
		return "high"
	}
}

func claudeWebSearchMaxUsesToContextSize(maxUses int) string {
	switch {
	case maxUses <= 0:
		return ""
	case maxUses <= 1:
		return "low"
	case maxUses <= 5:
		return "medium"
	default:
		return "high"
	}
}

func extractClaudeOutputConfigEffort(outputConfig json.RawMessage) string {
	if len(outputConfig) == 0 {
		return ""
	}

	var config dto.ClaudeOutputConfig
	if err := common.Unmarshal(outputConfig, &config); err != nil {
		return ""
	}
	if config.Effort == "max" {
		return "xhigh"
	}
	return config.Effort
}

func extractClaudeReasoningEffort(claudeRequest dto.ClaudeRequest) string {
	if effort := extractClaudeOutputConfigEffort(claudeRequest.OutputConfig); effort != "" {
		return effort
	}
	if claudeRequest.Thinking == nil {
		return ""
	}
	if claudeRequest.Thinking.Type == "adaptive" {
		return "high"
	}
	return claudeThinkingBudgetToReasoningEffort(claudeRequest.Thinking.GetBudgetTokens())
}

func buildOpenAIWebSearchOptions(tool *dto.ClaudeWebSearchTool) *dto.WebSearchOptions {
	if tool == nil {
		return nil
	}

	options := &dto.WebSearchOptions{
		SearchContextSize: claudeWebSearchMaxUsesToContextSize(tool.MaxUses),
	}
	if tool.UserLocation != nil {
		payload, err := common.Marshal(map[string]any{
			"approximate": tool.UserLocation,
		})
		if err == nil {
			options.UserLocation = payload
		}
	}
	return options
}

func appendOpenAIToolMessage(openAIMessages *[]dto.Message, claudeRequest dto.ClaudeRequest, mediaMsg dto.ClaudeMediaMessage) {
	toolName := mediaMsg.Name
	if toolName == "" {
		toolName = claudeRequest.SearchToolNameByToolCallId(mediaMsg.ToolUseId)
	}

	oaiToolMessage := dto.Message{
		Role:            "tool",
		Name:            &toolName,
		ToolCallId:      mediaMsg.ToolUseId,
		ToolCallIsError: mediaMsg.IsError,
	}
	if mediaMsg.IsStringContent() {
		oaiToolMessage.SetStringContent(mediaMsg.GetStringContent())
	} else {
		mediaContents := mediaMsg.ParseMediaContent()
		if len(mediaContents) == 0 {
			oaiToolMessage.SetStringContent("")
		} else {
			encodeJSON, _ := common.Marshal(mediaContents)
			oaiToolMessage.SetStringContent(string(encodeJSON))
		}
	}
	*openAIMessages = append(*openAIMessages, oaiToolMessage)
}

func ClaudeToOpenAIRequest(claudeRequest dto.ClaudeRequest, info *relaycommon.RelayInfo) (*dto.GeneralOpenAIRequest, error) {
	openAIRequest := dto.GeneralOpenAIRequest{
		Model:       claudeRequest.Model,
		MaxTokens:   claudeRequest.MaxTokens,
		Temperature: claudeRequest.Temperature,
		TopP:        claudeRequest.TopP,
		Stream:      claudeRequest.Stream,
	}

	isOpenRouter := info.ChannelType == constant.ChannelTypeOpenRouter
	openAIRequest.ReasoningEffort = extractClaudeReasoningEffort(claudeRequest)

	if claudeRequest.Thinking != nil && claudeRequest.Thinking.Type == "enabled" {
		if isOpenRouter {
			reasoning := openrouter.RequestReasoning{
				MaxTokens: claudeRequest.Thinking.GetBudgetTokens(),
			}
			reasoningJSON, err := common.Marshal(reasoning)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal reasoning: %w", err)
			}
			openAIRequest.Reasoning = reasoningJSON
		} else {
			thinkingSuffix := "-thinking"
			if strings.HasSuffix(info.OriginModelName, thinkingSuffix) &&
				!strings.HasSuffix(openAIRequest.Model, thinkingSuffix) {
				openAIRequest.Model = openAIRequest.Model + thinkingSuffix
			}
		}
	}
	if claudeRequest.Thinking != nil && claudeRequest.Thinking.Type == "adaptive" && openAIRequest.ReasoningEffort == "" {
		// Claude adaptive thinking has no direct Chat Completions equivalent, so we
		// expose it as high effort to preserve the intent when routing through OpenAI-style channels.
		openAIRequest.ReasoningEffort = "high"
	}

	// Convert stop sequences
	if len(claudeRequest.StopSequences) == 1 {
		openAIRequest.Stop = claudeRequest.StopSequences[0]
	} else if len(claudeRequest.StopSequences) > 1 {
		openAIRequest.Stop = claudeRequest.StopSequences
	}

	// Convert tools
	openAITools := make([]dto.ToolCallRequest, 0)
	normalTools, webSearchTools := dto.ProcessTools(claudeRequest.GetTools())
	for _, claudeTool := range normalTools {
		openAITool := dto.ToolCallRequest{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:        claudeTool.Name,
				Description: claudeTool.Description,
				Parameters:  claudeTool.InputSchema,
			},
		}
		openAITools = append(openAITools, openAITool)
	}
	openAIRequest.Tools = openAITools
	if len(webSearchTools) > 0 {
		openAIRequest.WebSearchOptions = buildOpenAIWebSearchOptions(webSearchTools[0])
	}

	// Convert messages
	openAIMessages := make([]dto.Message, 0)

	// Add system message if present
	if claudeRequest.System != nil {
		if claudeRequest.IsStringSystem() && claudeRequest.GetStringSystem() != "" {
			openAIMessage := dto.Message{
				Role: "system",
			}
			openAIMessage.SetStringContent(claudeRequest.GetStringSystem())
			openAIMessages = append(openAIMessages, openAIMessage)
		} else {
			systems := claudeRequest.ParseSystem()
			if len(systems) > 0 {
				openAIMessage := dto.Message{
					Role: "system",
				}
				isOpenRouterClaude := isOpenRouter && strings.HasPrefix(info.UpstreamModelName, "anthropic/claude")
				if isOpenRouterClaude {
					systemMediaMessages := make([]dto.MediaContent, 0, len(systems))
					for _, system := range systems {
						message := dto.MediaContent{
							Type:         "text",
							Text:         system.GetText(),
							CacheControl: system.CacheControl,
						}
						systemMediaMessages = append(systemMediaMessages, message)
					}
					openAIMessage.SetMediaContent(systemMediaMessages)
				} else {
					systemStr := ""
					for _, system := range systems {
						if system.Text != nil {
							systemStr += *system.Text
						}
					}
					openAIMessage.SetStringContent(systemStr)
				}
				openAIMessages = append(openAIMessages, openAIMessage)
			}
		}
	}
	for _, claudeMessage := range claudeRequest.Messages {
		openAIMessage := dto.Message{
			Role: claudeMessage.Role,
		}

		if claudeMessage.IsStringContent() {
			openAIMessage.SetStringContent(claudeMessage.GetStringContent())
		} else {
			content, err := claudeMessage.ParseContent()
			if err != nil {
				return nil, err
			}
			contents := content
			var toolCalls []dto.ToolCallRequest
			mediaMessages := make([]dto.MediaContent, 0, len(contents))
			var reasoningBuilder strings.Builder
			var reasoningSignature string
			var redactedReasoningBuilder strings.Builder

			for _, mediaMsg := range contents {
				switch mediaMsg.Type {
				case "thinking":
					if mediaMsg.Thinking != nil {
						reasoningBuilder.WriteString(*mediaMsg.Thinking)
					}
					if mediaMsg.Signature != "" {
						reasoningSignature = mediaMsg.Signature
					}
				case "redacted_thinking":
					redactedReasoningBuilder.WriteString(mediaMsg.Data)
				case "text":
					message := dto.MediaContent{
						Type:         "text",
						Text:         mediaMsg.GetText(),
						CacheControl: mediaMsg.CacheControl,
					}
					mediaMessages = append(mediaMessages, message)
				case "image":
					if mediaMsg.Source == nil {
						continue
					}
					imageURL := mediaMsg.Source.Url
					if imageURL == "" {
						// Claude base64 images are converted back into OpenAI data URLs.
						imageURL = fmt.Sprintf("data:%s;base64,%s", mediaMsg.Source.MediaType, mediaMsg.Source.Data)
					}
					mediaMessage := dto.MediaContent{
						Type:     "image_url",
						ImageUrl: &dto.MessageImageUrl{Url: imageURL},
					}
					mediaMessages = append(mediaMessages, mediaMessage)
				case "tool_use":
					toolCall := dto.ToolCallRequest{
						ID:   mediaMsg.Id,
						Type: "function",
						Function: dto.FunctionRequest{
							Name:      mediaMsg.Name,
							Arguments: toJSONString(mediaMsg.Input),
						},
					}
					toolCalls = append(toolCalls, toolCall)
				case "tool_result":
					appendOpenAIToolMessage(&openAIMessages, claudeRequest, mediaMsg)
				}
			}

			if reasoningBuilder.Len() > 0 {
				openAIMessage.ReasoningContent = reasoningBuilder.String()
			}
			if reasoningSignature != "" {
				openAIMessage.ReasoningSignature = reasoningSignature
			}
			if redactedReasoningBuilder.Len() > 0 {
				openAIMessage.RedactedReasoningContent = redactedReasoningBuilder.String()
			}

			if len(toolCalls) > 0 {
				openAIMessage.SetToolCalls(toolCalls)
			}

			if len(mediaMessages) > 0 && len(toolCalls) == 0 {
				openAIMessage.SetMediaContent(mediaMessages)
			}
		}
		if len(openAIMessage.ParseContent()) > 0 || len(openAIMessage.ToolCalls) > 0 || openAIMessage.ReasoningContent != "" || openAIMessage.ReasoningSignature != "" || openAIMessage.RedactedReasoningContent != "" {
			openAIMessages = append(openAIMessages, openAIMessage)
		}
	}

	openAIRequest.Messages = openAIMessages

	return &openAIRequest, nil
}

func generateStopBlock(index int) *dto.ClaudeResponse {
	return &dto.ClaudeResponse{
		Type:  "content_block_stop",
		Index: common.GetPointer[int](index),
	}
}

func StreamResponseOpenAI2Claude(openAIResponse *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) []*dto.ClaudeResponse {
	if info.ClaudeConvertInfo.Done {
		return nil
	}

	var claudeResponses []*dto.ClaudeResponse
	// stopOpenBlocks emits the required content_block_stop event(s) for the currently open block(s)
	// according to Anthropic's SSE streaming state machine:
	// content_block_start -> content_block_delta* -> content_block_stop (per index).
	//
	// For text/thinking, there is at most one open block at info.ClaudeConvertInfo.Index.
	// For tools, OpenAI tool_calls can stream multiple parallel tool_use blocks (indexed from 0),
	// so we may have multiple open blocks and must stop each one explicitly.
	stopOpenBlocks := func() {
		switch info.ClaudeConvertInfo.LastMessagesType {
		case relaycommon.LastMessageTypeText, relaycommon.LastMessageTypeThinking:
			claudeResponses = append(claudeResponses, generateStopBlock(info.ClaudeConvertInfo.Index))
		case relaycommon.LastMessageTypeTools:
			base := info.ClaudeConvertInfo.ToolCallBaseIndex
			for offset := 0; offset <= info.ClaudeConvertInfo.ToolCallMaxIndexOffset; offset++ {
				claudeResponses = append(claudeResponses, generateStopBlock(base+offset))
			}
		}
	}
	// stopOpenBlocksAndAdvance closes the currently open block(s) and advances the content block index
	// to the next available slot for subsequent content_block_start events.
	//
	// This prevents invalid streams where a content_block_delta (e.g. thinking_delta) is emitted for an
	// index whose active content_block type is different (the typical cause of "Mismatched content block type").
	stopOpenBlocksAndAdvance := func() {
		if info.ClaudeConvertInfo.LastMessagesType == relaycommon.LastMessageTypeNone {
			return
		}
		stopOpenBlocks()
		switch info.ClaudeConvertInfo.LastMessagesType {
		case relaycommon.LastMessageTypeTools:
			info.ClaudeConvertInfo.Index = info.ClaudeConvertInfo.ToolCallBaseIndex + info.ClaudeConvertInfo.ToolCallMaxIndexOffset + 1
			info.ClaudeConvertInfo.ToolCallBaseIndex = 0
			info.ClaudeConvertInfo.ToolCallMaxIndexOffset = 0
		default:
			info.ClaudeConvertInfo.Index++
		}
		info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeNone
	}
	flushPendingThinkingSignature := func(closeImmediately bool) {
		if info.ClaudeConvertInfo.PendingReasoningSignature == "" {
			return
		}
		if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeThinking {
			stopOpenBlocksAndAdvance()
			idx := info.ClaudeConvertInfo.Index
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Index: &idx,
				Type:  "content_block_start",
				ContentBlock: &dto.ClaudeMediaMessage{
					Type:     "thinking",
					Thinking: common.GetPointer[string](""),
				},
			})
			info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeThinking
		}
		idx := info.ClaudeConvertInfo.Index
		signature := info.ClaudeConvertInfo.PendingReasoningSignature
		claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
			Index: &idx,
			Type:  "content_block_delta",
			Delta: &dto.ClaudeMediaMessage{
				Type:      "signature_delta",
				Signature: signature,
			},
		})
		info.ClaudeConvertInfo.PendingReasoningSignature = ""
		if closeImmediately {
			stopOpenBlocksAndAdvance()
		}
	}
	if info.SendResponseCount == 1 {
		msg := &dto.ClaudeMediaMessage{
			Id:    openAIResponse.Id,
			Model: openAIResponse.Model,
			Type:  "message",
			Role:  "assistant",
			Usage: &dto.ClaudeUsage{
				InputTokens:  info.GetEstimatePromptTokens(),
				OutputTokens: 0,
			},
		}
		msg.SetContent(make([]any, 0))
		claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
			Type:    "message_start",
			Message: msg,
		})
		//claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
		//	Type: "ping",
		//})
		if openAIResponse.IsToolCall() {
			info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeTools
			info.ClaudeConvertInfo.ToolCallBaseIndex = 0
			info.ClaudeConvertInfo.ToolCallMaxIndexOffset = 0
			var toolCall dto.ToolCallResponse
			if len(openAIResponse.Choices) > 0 && len(openAIResponse.Choices[0].Delta.ToolCalls) > 0 {
				toolCall = openAIResponse.Choices[0].Delta.ToolCalls[0]
			} else {
				first := openAIResponse.GetFirstToolCall()
				if first != nil {
					toolCall = *first
				} else {
					toolCall = dto.ToolCallResponse{}
				}
			}
			resp := &dto.ClaudeResponse{
				Type: "content_block_start",
				ContentBlock: &dto.ClaudeMediaMessage{
					Id:    toolCall.ID,
					Type:  "tool_use",
					Name:  toolCall.Function.Name,
					Input: map[string]interface{}{},
				},
			}
			resp.SetIndex(0)
			claudeResponses = append(claudeResponses, resp)
			// 首块包含工具 delta，则追加 input_json_delta
			if toolCall.Function.Arguments != "" {
				idx := 0
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx,
					Type:  "content_block_delta",
					Delta: &dto.ClaudeMediaMessage{
						Type:        "input_json_delta",
						PartialJson: &toolCall.Function.Arguments,
					},
				})
			}
		} else {

		}
		// 判断首个响应是否存在内容（非标准的 OpenAI 响应）
		if len(openAIResponse.Choices) > 0 {
			reasoning := openAIResponse.Choices[0].Delta.GetReasoningContent()
			reasoningSignature := ""
			if openAIResponse.Choices[0].Delta.ReasoningSignature != nil {
				reasoningSignature = *openAIResponse.Choices[0].Delta.ReasoningSignature
			}
			redactedReasoning := ""
			if openAIResponse.Choices[0].Delta.RedactedReasoningContent != nil {
				redactedReasoning = *openAIResponse.Choices[0].Delta.RedactedReasoningContent
			}
			content := openAIResponse.Choices[0].Delta.GetContentString()

			if reasoning != "" {
				if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeThinking {
					stopOpenBlocksAndAdvance()
				}
				idx := info.ClaudeConvertInfo.Index
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx,
					Type:  "content_block_start",
					ContentBlock: &dto.ClaudeMediaMessage{
						Type:     "thinking",
						Thinking: common.GetPointer[string](""),
					},
				})
				idx2 := idx
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Index: &idx2,
					Type:  "content_block_delta",
					Delta: &dto.ClaudeMediaMessage{
						Type:     "thinking_delta",
						Thinking: &reasoning,
					},
				})
				if reasoningSignature != "" {
					info.ClaudeConvertInfo.PendingReasoningSignature = reasoningSignature
					flushPendingThinkingSignature(false)
				}
				info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeThinking
			} else {
				if reasoningSignature != "" {
					info.ClaudeConvertInfo.PendingReasoningSignature = reasoningSignature
				}
				if redactedReasoning != "" {
					flushPendingThinkingSignature(true)
					stopOpenBlocksAndAdvance()
					idx := info.ClaudeConvertInfo.Index
					claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
						Index: &idx,
						Type:  "content_block_start",
						ContentBlock: &dto.ClaudeMediaMessage{
							Type: "redacted_thinking",
							Data: redactedReasoning,
						},
					})
					claudeResponses = append(claudeResponses, generateStopBlock(idx))
					info.ClaudeConvertInfo.Index++
				}
				if content != "" {
					if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeText {
						stopOpenBlocksAndAdvance()
					}
					idx := info.ClaudeConvertInfo.Index
					claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
						Index: &idx,
						Type:  "content_block_start",
						ContentBlock: &dto.ClaudeMediaMessage{
							Type: "text",
							Text: common.GetPointer[string](""),
						},
					})
					idx2 := idx
					claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
						Index: &idx2,
						Type:  "content_block_delta",
						Delta: &dto.ClaudeMediaMessage{
							Type: "text_delta",
							Text: common.GetPointer[string](content),
						},
					})
					info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeText
				}
			}
		}

		// 如果首块就带 finish_reason，需要立即发送停止块
		if len(openAIResponse.Choices) > 0 && openAIResponse.Choices[0].FinishReason != nil && *openAIResponse.Choices[0].FinishReason != "" {
			info.FinishReason = *openAIResponse.Choices[0].FinishReason
			flushPendingThinkingSignature(true)
			stopOpenBlocks()
			oaiUsage := openAIResponse.Usage
			if oaiUsage == nil {
				oaiUsage = info.ClaudeConvertInfo.Usage
			}
			if oaiUsage != nil {
				claudeUsage := dto.OpenAIUsageToClaudeUsage(oaiUsage)
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Type:  "message_delta",
					Usage: claudeUsage,
					Delta: &dto.ClaudeMediaMessage{
						StopReason: common.GetPointer[string](stopReasonOpenAI2Claude(info.FinishReason)),
					},
				})
			}
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type: "message_stop",
			})
			info.ClaudeConvertInfo.Done = true
		}
		return claudeResponses
	}

	if len(openAIResponse.Choices) == 0 {
		// no choices
		// 可能为非标准的 OpenAI 响应，判断是否已经完成
		if info.ClaudeConvertInfo.Done {
			flushPendingThinkingSignature(true)
			stopOpenBlocks()
			oaiUsage := info.ClaudeConvertInfo.Usage
			if oaiUsage != nil {
				claudeUsage := dto.OpenAIUsageToClaudeUsage(oaiUsage)
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Type:  "message_delta",
					Usage: claudeUsage,
					Delta: &dto.ClaudeMediaMessage{
						StopReason: common.GetPointer[string](stopReasonOpenAI2Claude(info.FinishReason)),
					},
				})
			}
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type: "message_stop",
			})
		}
		return claudeResponses
	} else {
		chosenChoice := openAIResponse.Choices[0]
		doneChunk := chosenChoice.FinishReason != nil && *chosenChoice.FinishReason != ""
		if doneChunk {
			info.FinishReason = *chosenChoice.FinishReason
		}
		if chosenChoice.Delta.ReasoningSignature != nil && *chosenChoice.Delta.ReasoningSignature != "" && chosenChoice.Delta.GetReasoningContent() == "" {
			info.ClaudeConvertInfo.PendingReasoningSignature = *chosenChoice.Delta.ReasoningSignature
		}

		var claudeResponse dto.ClaudeResponse
		var isEmpty bool
		claudeResponse.Type = "content_block_delta"
		if len(chosenChoice.Delta.ToolCalls) > 0 {
			flushPendingThinkingSignature(true)
			toolCalls := chosenChoice.Delta.ToolCalls
			if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeTools {
				stopOpenBlocksAndAdvance()
				info.ClaudeConvertInfo.ToolCallBaseIndex = info.ClaudeConvertInfo.Index
				info.ClaudeConvertInfo.ToolCallMaxIndexOffset = 0
			}
			info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeTools
			base := info.ClaudeConvertInfo.ToolCallBaseIndex
			maxOffset := info.ClaudeConvertInfo.ToolCallMaxIndexOffset

			for i, toolCall := range toolCalls {
				offset := 0
				if toolCall.Index != nil {
					offset = *toolCall.Index
				} else {
					offset = i
				}
				if offset > maxOffset {
					maxOffset = offset
				}
				blockIndex := base + offset

				idx := blockIndex
				if toolCall.Function.Name != "" {
					claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
						Index: &idx,
						Type:  "content_block_start",
						ContentBlock: &dto.ClaudeMediaMessage{
							Id:    toolCall.ID,
							Type:  "tool_use",
							Name:  toolCall.Function.Name,
							Input: map[string]interface{}{},
						},
					})
				}

				if len(toolCall.Function.Arguments) > 0 {
					claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
						Index: &idx,
						Type:  "content_block_delta",
						Delta: &dto.ClaudeMediaMessage{
							Type:        "input_json_delta",
							PartialJson: &toolCall.Function.Arguments,
						},
					})
				}
			}
			info.ClaudeConvertInfo.ToolCallMaxIndexOffset = maxOffset
			info.ClaudeConvertInfo.Index = base + maxOffset
		} else {
			reasoning := chosenChoice.Delta.GetReasoningContent()
			redactedReasoning := ""
			if chosenChoice.Delta.RedactedReasoningContent != nil {
				redactedReasoning = *chosenChoice.Delta.RedactedReasoningContent
			}
			textContent := chosenChoice.Delta.GetContentString()
			if reasoning != "" || redactedReasoning != "" || textContent != "" {
				if reasoning != "" {
					if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeThinking {
						stopOpenBlocksAndAdvance()
						idx := info.ClaudeConvertInfo.Index
						claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
							Index: &idx,
							Type:  "content_block_start",
							ContentBlock: &dto.ClaudeMediaMessage{
								Type:     "thinking",
								Thinking: common.GetPointer[string](""),
							},
						})
					}
					info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeThinking
					claudeResponse.Delta = &dto.ClaudeMediaMessage{
						Type:     "thinking_delta",
						Thinking: &reasoning,
					}
					if chosenChoice.Delta.ReasoningSignature != nil && *chosenChoice.Delta.ReasoningSignature != "" {
						info.ClaudeConvertInfo.PendingReasoningSignature = *chosenChoice.Delta.ReasoningSignature
					}
				} else if redactedReasoning != "" {
					flushPendingThinkingSignature(true)
					stopOpenBlocksAndAdvance()
					idx := info.ClaudeConvertInfo.Index
					claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
						Index: &idx,
						Type:  "content_block_start",
						ContentBlock: &dto.ClaudeMediaMessage{
							Type: "redacted_thinking",
							Data: redactedReasoning,
						},
					})
					claudeResponses = append(claudeResponses, generateStopBlock(idx))
					info.ClaudeConvertInfo.Index++
				} else {
					flushPendingThinkingSignature(true)
					if info.ClaudeConvertInfo.LastMessagesType != relaycommon.LastMessageTypeText {
						stopOpenBlocksAndAdvance()
						idx := info.ClaudeConvertInfo.Index
						claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
							Index: &idx,
							Type:  "content_block_start",
							ContentBlock: &dto.ClaudeMediaMessage{
								Type: "text",
								Text: common.GetPointer[string](""),
							},
						})
					}
					info.ClaudeConvertInfo.LastMessagesType = relaycommon.LastMessageTypeText
					claudeResponse.Delta = &dto.ClaudeMediaMessage{
						Type: "text_delta",
						Text: common.GetPointer[string](textContent),
					}
				}
			} else {
				isEmpty = true
			}
		}

		claudeResponse.Index = common.GetPointer[int](info.ClaudeConvertInfo.Index)
		if !isEmpty && claudeResponse.Delta != nil {
			claudeResponses = append(claudeResponses, &claudeResponse)
			if claudeResponse.Delta.Type == "thinking_delta" {
				flushPendingThinkingSignature(false)
			}
		}

		if doneChunk || info.ClaudeConvertInfo.Done {
			flushPendingThinkingSignature(true)
			stopOpenBlocks()
			oaiUsage := openAIResponse.Usage
			if oaiUsage == nil {
				oaiUsage = info.ClaudeConvertInfo.Usage
			}
			if oaiUsage != nil {
				claudeUsage := dto.OpenAIUsageToClaudeUsage(oaiUsage)
				claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
					Type:  "message_delta",
					Usage: claudeUsage,
					Delta: &dto.ClaudeMediaMessage{
						StopReason: common.GetPointer[string](stopReasonOpenAI2Claude(info.FinishReason)),
					},
				})
			}
			claudeResponses = append(claudeResponses, &dto.ClaudeResponse{
				Type: "message_stop",
			})
			info.ClaudeConvertInfo.Done = true
			return claudeResponses
		}
	}

	return claudeResponses
}

func ResponseOpenAI2Claude(openAIResponse *dto.OpenAITextResponse, info *relaycommon.RelayInfo) *dto.ClaudeResponse {
	var stopReason string
	contents := make([]dto.ClaudeMediaMessage, 0)
	claudeResponse := &dto.ClaudeResponse{
		Id:    openAIResponse.Id,
		Type:  "message",
		Role:  "assistant",
		Model: openAIResponse.Model,
	}
	for _, choice := range openAIResponse.Choices {
		stopReason = stopReasonOpenAI2Claude(choice.FinishReason)

		reasoningText := choice.Message.ReasoningContent
		if reasoningText == "" {
			reasoningText = choice.Message.Reasoning
		}
		if reasoningText != "" || choice.Message.ReasoningSignature != "" {
			claudeContent := dto.ClaudeMediaMessage{Type: "thinking", Signature: choice.Message.ReasoningSignature}
			claudeContent.Thinking = common.GetPointer[string](reasoningText)
			contents = append(contents, claudeContent)
		}
		if choice.Message.RedactedReasoningContent != "" {
			contents = append(contents, dto.ClaudeMediaMessage{
				Type: "redacted_thinking",
				Data: choice.Message.RedactedReasoningContent,
			})
		}

		for _, toolUse := range choice.Message.ParseToolCalls() {
			claudeContent := dto.ClaudeMediaMessage{Type: "tool_use", Id: toolUse.ID, Name: toolUse.Function.Name}
			var mapParams map[string]interface{}
			if err := common.Unmarshal([]byte(toolUse.Function.Arguments), &mapParams); err == nil {
				claudeContent.Input = mapParams
			} else {
				claudeContent.Input = toolUse.Function.Arguments
			}
			contents = append(contents, claudeContent)
		}

		if choice.Message.IsStringContent() {
			text := choice.Message.StringContent()
			if text != "" {
				claudeContent := dto.ClaudeMediaMessage{Type: "text"}
				claudeContent.SetText(text)
				contents = append(contents, claudeContent)
			}
			continue
		}

		for _, media := range choice.Message.ParseContent() {
			switch media.Type {
			case "text":
				claudeContent := dto.ClaudeMediaMessage{Type: "text"}
				claudeContent.SetText(media.Text)
				contents = append(contents, claudeContent)
			case "image_url":
				imageURL := media.GetImageMedia()
				if imageURL == nil || imageURL.Url == "" {
					continue
				}
				claudeContent := dto.ClaudeMediaMessage{Type: "image", Source: &dto.ClaudeMessageSource{}}
				if strings.HasPrefix(imageURL.Url, "data:") {
					mimeType := imageURL.MimeType
					base64Data := imageURL.Url
					if strings.Contains(imageURL.Url, ",") {
						parts := strings.SplitN(imageURL.Url, ",", 2)
						base64Data = parts[1]
						if mimeType == "" && strings.HasPrefix(parts[0], "data:") {
							mimeType = strings.TrimPrefix(strings.TrimSuffix(parts[0], ";base64"), "data:")
						}
					}
					claudeContent.Source.Type = "base64"
					claudeContent.Source.MediaType = mimeType
					claudeContent.Source.Data = base64Data
				} else {
					claudeContent.Source.Type = "url"
					claudeContent.Source.Url = imageURL.Url
				}
				contents = append(contents, claudeContent)
			}
		}
	}
	claudeResponse.Content = contents
	claudeResponse.StopReason = stopReason
	claudeResponse.Usage = dto.OpenAIUsageToClaudeUsage(&openAIResponse.Usage)

	return claudeResponse
}

func stopReasonOpenAI2Claude(reason string) string {
	return reasonmap.OpenAIFinishReasonToClaudeStopReason(reason)
}

func toJSONString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func GeminiToOpenAIRequest(geminiRequest *dto.GeminiChatRequest, info *relaycommon.RelayInfo) (*dto.GeneralOpenAIRequest, error) {
	openaiRequest := &dto.GeneralOpenAIRequest{
		Model:  info.UpstreamModelName,
		Stream: info.IsStream,
	}

	// 转换 messages
	var messages []dto.Message
	for _, content := range geminiRequest.Contents {
		message := dto.Message{
			Role: convertGeminiRoleToOpenAI(content.Role),
		}

		// 处理 parts
		var mediaContents []dto.MediaContent
		var toolCalls []dto.ToolCallRequest
		for _, part := range content.Parts {
			if part.Text != "" {
				mediaContent := dto.MediaContent{
					Type: "text",
					Text: part.Text,
				}
				mediaContents = append(mediaContents, mediaContent)
			} else if part.InlineData != nil {
				mediaContent := dto.MediaContent{
					Type: "image_url",
					ImageUrl: &dto.MessageImageUrl{
						Url:      fmt.Sprintf("data:%s;base64,%s", part.InlineData.MimeType, part.InlineData.Data),
						Detail:   "auto",
						MimeType: part.InlineData.MimeType,
					},
				}
				mediaContents = append(mediaContents, mediaContent)
			} else if part.FileData != nil {
				mediaContent := dto.MediaContent{
					Type: "image_url",
					ImageUrl: &dto.MessageImageUrl{
						Url:      part.FileData.FileUri,
						Detail:   "auto",
						MimeType: part.FileData.MimeType,
					},
				}
				mediaContents = append(mediaContents, mediaContent)
			} else if part.FunctionCall != nil {
				// 处理 Gemini 的工具调用
				toolCall := dto.ToolCallRequest{
					ID:   fmt.Sprintf("call_%d", len(toolCalls)+1), // 生成唯一ID
					Type: "function",
					Function: dto.FunctionRequest{
						Name:      part.FunctionCall.FunctionName,
						Arguments: toJSONString(part.FunctionCall.Arguments),
					},
				}
				toolCalls = append(toolCalls, toolCall)
			} else if part.FunctionResponse != nil {
				// 处理 Gemini 的工具响应，创建单独的 tool 消息
				toolMessage := dto.Message{
					Role:       "tool",
					ToolCallId: fmt.Sprintf("call_%d", len(toolCalls)), // 使用对应的调用ID
				}
				toolMessage.SetStringContent(toJSONString(part.FunctionResponse.Response))
				messages = append(messages, toolMessage)
			}
		}

		// 设置消息内容
		if len(toolCalls) > 0 {
			// 如果有工具调用，设置工具调用
			message.SetToolCalls(toolCalls)
		} else if len(mediaContents) == 1 && mediaContents[0].Type == "text" {
			// 如果只有一个文本内容，直接设置字符串
			message.Content = mediaContents[0].Text
		} else if len(mediaContents) > 0 {
			// 如果有多个内容或包含媒体，设置为数组
			message.SetMediaContent(mediaContents)
		}

		// 只有当消息有内容或工具调用时才添加
		if len(message.ParseContent()) > 0 || len(message.ToolCalls) > 0 {
			messages = append(messages, message)
		}
	}

	openaiRequest.Messages = messages

	if geminiRequest.GenerationConfig.Temperature != nil {
		openaiRequest.Temperature = geminiRequest.GenerationConfig.Temperature
	}
	if geminiRequest.GenerationConfig.TopP > 0 {
		openaiRequest.TopP = geminiRequest.GenerationConfig.TopP
	}
	if geminiRequest.GenerationConfig.TopK > 0 {
		openaiRequest.TopK = int(geminiRequest.GenerationConfig.TopK)
	}
	if geminiRequest.GenerationConfig.MaxOutputTokens > 0 {
		openaiRequest.MaxTokens = geminiRequest.GenerationConfig.MaxOutputTokens
	}
	// gemini stop sequences 最多 5 个，openai stop 最多 4 个
	if len(geminiRequest.GenerationConfig.StopSequences) > 0 {
		openaiRequest.Stop = geminiRequest.GenerationConfig.StopSequences[:4]
	}
	if geminiRequest.GenerationConfig.CandidateCount > 0 {
		openaiRequest.N = geminiRequest.GenerationConfig.CandidateCount
	}

	// 转换工具调用
	if len(geminiRequest.GetTools()) > 0 {
		var tools []dto.ToolCallRequest
		for _, tool := range geminiRequest.GetTools() {
			if tool.FunctionDeclarations != nil {
				functionDeclarations, err := common.Any2Type[[]dto.FunctionRequest](tool.FunctionDeclarations)
				if err != nil {
					common.SysError(fmt.Sprintf("failed to parse gemini function declarations: %v (type=%T)", err, tool.FunctionDeclarations))
					continue
				}
				for _, function := range functionDeclarations {
					openAITool := dto.ToolCallRequest{
						Type: "function",
						Function: dto.FunctionRequest{
							Name:        function.Name,
							Description: function.Description,
							Parameters:  function.Parameters,
						},
					}
					tools = append(tools, openAITool)
				}
			}
		}
		if len(tools) > 0 {
			openaiRequest.Tools = tools
		}
	}

	// gemini system instructions
	if geminiRequest.SystemInstructions != nil {
		// 将系统指令作为第一条消息插入
		systemMessage := dto.Message{
			Role:    "system",
			Content: extractTextFromGeminiParts(geminiRequest.SystemInstructions.Parts),
		}
		openaiRequest.Messages = append([]dto.Message{systemMessage}, openaiRequest.Messages...)
	}

	return openaiRequest, nil
}

func convertGeminiRoleToOpenAI(geminiRole string) string {
	switch geminiRole {
	case "user":
		return "user"
	case "model":
		return "assistant"
	case "function":
		return "function"
	default:
		return "user"
	}
}

func extractTextFromGeminiParts(parts []dto.GeminiPart) string {
	var texts []string
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// ResponseOpenAI2Gemini 将 OpenAI 响应转换为 Gemini 格式
func ResponseOpenAI2Gemini(openAIResponse *dto.OpenAITextResponse, info *relaycommon.RelayInfo) *dto.GeminiChatResponse {
	geminiResponse := &dto.GeminiChatResponse{
		Candidates:    make([]dto.GeminiChatCandidate, 0, len(openAIResponse.Choices)),
		UsageMetadata: OpenAIUsageToGeminiUsage(openAIResponse.Usage),
	}

	for _, choice := range openAIResponse.Choices {
		candidate := dto.GeminiChatCandidate{
			Index:         int64(choice.Index),
			SafetyRatings: []dto.GeminiChatSafetyRating{},
		}

		// 设置结束原因
		var finishReason string
		switch choice.FinishReason {
		case "stop":
			finishReason = "STOP"
		case "length":
			finishReason = "MAX_TOKENS"
		case "content_filter":
			finishReason = "SAFETY"
		case "tool_calls":
			finishReason = "STOP"
		default:
			finishReason = "STOP"
		}
		candidate.FinishReason = &finishReason

		// 转换消息内容
		content := dto.GeminiChatContent{
			Role:  "model",
			Parts: make([]dto.GeminiPart, 0),
		}

		reasoningContent := choice.Message.ReasoningContent
		if reasoningContent == "" {
			reasoningContent = choice.Message.Reasoning
		}
		if reasoningContent != "" {
			content.Parts = append(content.Parts, dto.GeminiPart{
				Text:    reasoningContent,
				Thought: true,
			})
		}

		if textContent := choice.Message.StringContent(); textContent != "" {
			content.Parts = append(content.Parts, dto.GeminiPart{Text: textContent})
		}

		for _, toolCall := range choice.Message.ParseToolCalls() {
			content.Parts = append(content.Parts, dto.GeminiPart{
				FunctionCall: &dto.FunctionCall{
					FunctionName: toolCall.Function.Name,
					Arguments:    parseOpenAIFunctionArguments(toolCall.Function.Arguments),
				},
			})
		}

		candidate.Content = content
		geminiResponse.Candidates = append(geminiResponse.Candidates, candidate)
	}

	return geminiResponse
}

// StreamResponseOpenAI2Gemini 将 OpenAI 流式响应转换为 Gemini 格式
func StreamResponseOpenAI2Gemini(openAIResponse *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) *dto.GeminiChatResponse {
	// 检查是否有实际内容或结束标志
	hasContent := false
	hasFinishReason := false
	for _, choice := range openAIResponse.Choices {
		if len(choice.Delta.GetContentString()) > 0 || len(choice.Delta.GetReasoningContent()) > 0 || (choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0) {
			hasContent = true
		}
		if choice.FinishReason != nil {
			hasFinishReason = true
		}
	}

	// 如果没有实际内容且没有结束标志，跳过。主要针对 openai 流响应开头的空数据
	if !hasContent && !hasFinishReason {
		return nil
	}

	geminiResponse := &dto.GeminiChatResponse{
		Candidates: make([]dto.GeminiChatCandidate, 0, len(openAIResponse.Choices)),
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     info.GetEstimatePromptTokens(),
			CandidatesTokenCount: 0, // 流式响应中可能没有完整的 usage 信息
			TotalTokenCount:      info.GetEstimatePromptTokens(),
		},
	}

	if openAIResponse.Usage != nil {
		if usageMetadata := OpenAIUsageToGeminiUsage(*openAIResponse.Usage); HasGeminiUsageMetadata(usageMetadata) {
			geminiResponse.UsageMetadata = usageMetadata
		}
	}

	for _, choice := range openAIResponse.Choices {
		candidate := dto.GeminiChatCandidate{
			Index:         int64(choice.Index),
			SafetyRatings: []dto.GeminiChatSafetyRating{},
		}

		// 设置结束原因
		if choice.FinishReason != nil {
			var finishReason string
			switch *choice.FinishReason {
			case "stop":
				finishReason = "STOP"
			case "length":
				finishReason = "MAX_TOKENS"
			case "content_filter":
				finishReason = "SAFETY"
			case "tool_calls":
				finishReason = "STOP"
			default:
				finishReason = "STOP"
			}
			candidate.FinishReason = &finishReason
		}

		// 转换消息内容
		content := dto.GeminiChatContent{
			Role:  "model",
			Parts: make([]dto.GeminiPart, 0),
		}

		if reasoningContent := choice.Delta.GetReasoningContent(); reasoningContent != "" {
			content.Parts = append(content.Parts, dto.GeminiPart{
				Text:    reasoningContent,
				Thought: true,
			})
		}

		if textContent := choice.Delta.GetContentString(); textContent != "" {
			content.Parts = append(content.Parts, dto.GeminiPart{Text: textContent})
		}

		for _, toolCall := range choice.Delta.ToolCalls {
			content.Parts = append(content.Parts, dto.GeminiPart{
				FunctionCall: &dto.FunctionCall{
					FunctionName: toolCall.Function.Name,
					Arguments:    parseOpenAIFunctionArguments(toolCall.Function.Arguments),
				},
			})
		}

		candidate.Content = content
		geminiResponse.Candidates = append(geminiResponse.Candidates, candidate)
	}

	return geminiResponse
}

// parseOpenAIFunctionArguments keeps Gemini function-call args structured when the OpenAI payload is valid JSON.
func parseOpenAIFunctionArguments(arguments string) any {
	if strings.TrimSpace(arguments) == "" {
		return map[string]interface{}{}
	}

	var args any
	if err := common.Unmarshal([]byte(arguments), &args); err == nil {
		return args
	}
	return map[string]interface{}{"arguments": arguments}
}
