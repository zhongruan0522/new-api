package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/dto"
	"github.com/zhongruan0522/new-api/logger"
	relaycommon "github.com/zhongruan0522/new-api/relay/common"
	"github.com/zhongruan0522/new-api/relay/helper"
	"github.com/zhongruan0522/new-api/service"
	"github.com/zhongruan0522/new-api/types"

	"github.com/gin-gonic/gin"
)

func OaiResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	// read response body
	var responsesResponse dto.OpenAIResponsesResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	err = common.Unmarshal(responseBody, &responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	if responsesResponse.HasImageGenerationCall() {
		c.Set("image_generation_call", true)
		c.Set("image_generation_call_quality", responsesResponse.GetQuality())
		c.Set("image_generation_call_size", responsesResponse.GetSize())
	}

	// compute usage
	usage := dto.Usage{}
	relaycommon.ApplyResponsesUsageToChatUsage(&usage, responsesResponse.Usage)

	if info != nil && info.RelayFormat == types.RelayFormatClaude {
		responseBody, err = convertResponsesBodyToClaudeBody(&responsesResponse, &usage, info)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
		}
	}

	// 写入新的 response body
	service.IOCopyBytesGracefully(c, resp, responseBody)

	if info == nil || info.ResponsesUsageInfo == nil || info.ResponsesUsageInfo.BuiltInTools == nil {
		return &usage, nil
	}
	// 解析 Output 中的内置工具调用次数（web_search_call、file_search_call 等）
	// 注意：不能遍历 responsesResponse.Tools，那是请求工具配置的回显，不是实际调用结果
	for _, output := range responsesResponse.Output {
		var toolType string
		switch output.Type {
		case dto.BuildInCallWebSearchCall:
			toolType = dto.BuildInToolWebSearchPreview
		case dto.BuildInCallFileSearchCall:
			toolType = dto.BuildInToolFileSearch
		default:
			continue
		}
		buildToolinfo, ok := info.ResponsesUsageInfo.BuiltInTools[toolType]
		if !ok || buildToolinfo == nil {
			logger.LogError(c, fmt.Sprintf("BuiltInTools not found for tool type: %v", toolType))
			continue
		}
		buildToolinfo.CallCount++
	}
	return &usage, nil
}

func OaiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, "invalid response or response body")
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	defer service.CloseResponseBodyGracefully(resp)

	var usage = &dto.Usage{}
	var responseTextBuilder strings.Builder
	var responsesToChat relaycommon.OpenAIWireStreamConverter
	if info.RelayFormat == types.RelayFormatClaude {
		responsesToChat = relaycommon.NewResponsesToChatStreamConverter(false)
	}

	helper.StreamScannerHandler(c, resp, info, func(data string) bool {

		// 检查当前数据是否包含 completed 状态和 usage 信息
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err == nil {
			switch streamResponse.Type {
			case "response.completed":
				if streamResponse.Response != nil {
					if streamResponse.Response.Usage != nil {
						relaycommon.ApplyResponsesUsageToChatUsage(usage, streamResponse.Response.Usage)
						if info.RelayFormat == types.RelayFormatClaude && info.ClaudeConvertInfo != nil {
							info.ClaudeConvertInfo.Usage = usage
						}
					}
					if streamResponse.Response.HasImageGenerationCall() {
						c.Set("image_generation_call", true)
						c.Set("image_generation_call_quality", streamResponse.Response.GetQuality())
						c.Set("image_generation_call_size", streamResponse.Response.GetSize())
					}
				}
			case "response.output_text.delta":
				// 处理输出文本
				responseTextBuilder.WriteString(streamResponse.Delta)
			case dto.ResponsesOutputTypeItemDone:
				// 内置工具调用计数
				if streamResponse.Item != nil && info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
					switch streamResponse.Item.Type {
					case dto.BuildInCallWebSearchCall:
						if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool != nil {
							webSearchTool.CallCount++
						}
					case dto.BuildInCallFileSearchCall:
						if fileSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch]; exists && fileSearchTool != nil {
							fileSearchTool.CallCount++
						}
					}
				}
			}
		} else {
			logger.LogError(c, "failed to unmarshal stream response: "+err.Error())
		}

		if info.RelayFormat == types.RelayFormatClaude {
			if err := writeResponsesStreamAsClaude(c, info, responsesToChat, data); err != nil {
				logger.LogError(c, "failed to convert responses stream to claude: "+err.Error())
				return false
			}
			return true
		}

		if streamResponse.Type != "" {
			sendResponsesStreamData(c, streamResponse, data)
		}
		return true
	})

	if usage.CompletionTokens == 0 {
		// 计算输出文本的 token 数量
		tempStr := responseTextBuilder.String()
		if len(tempStr) > 0 {
			// 非正常结束，使用输出文本的 token 数量
			completionTokens := service.CountTextToken(tempStr, info.UpstreamModelName)
			usage.CompletionTokens = completionTokens
		}
	}

	if usage.PromptTokens == 0 && usage.CompletionTokens != 0 {
		usage.PromptTokens = info.GetEstimatePromptTokens()
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return usage, nil
}

func convertResponsesBodyToClaudeBody(responsesResponse *dto.OpenAIResponsesResponse, usage *dto.Usage, info *relaycommon.RelayInfo) ([]byte, error) {
	chatResponse, err := relaycommon.ConvertResponsesResponseToChatCompletionResponse(responsesResponse)
	if err != nil {
		return nil, err
	}
	if usage != nil {
		chatResponse.Usage = *usage
	}
	claudeResp := service.ResponseOpenAI2Claude(chatResponse, info)
	return common.Marshal(claudeResp)
}

func writeResponsesStreamAsClaude(c *gin.Context, info *relaycommon.RelayInfo, converter relaycommon.OpenAIWireStreamConverter, data string) error {
	if converter == nil {
		return fmt.Errorf("responses to chat stream converter is nil")
	}
	if info.ClaudeConvertInfo == nil {
		info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{LastMessagesType: relaycommon.LastMessageTypeNone}
	}
	var streamResponse dto.ResponsesStreamResponse
	if err := common.UnmarshalJsonStr(data, &streamResponse); err == nil && streamResponse.Response != nil && streamResponse.Response.Usage != nil {
		usage := &dto.Usage{}
		relaycommon.ApplyResponsesUsageToChatUsage(usage, streamResponse.Response.Usage)
		info.ClaudeConvertInfo.Usage = usage
	}
	out, err := converter.ConvertFrame("", data, "data: "+data+"\n\n")
	if err != nil {
		return err
	}
	for _, chatData := range chatDataFrames(out) {
		if chatData == "[DONE]" {
			continue
		}
		info.SendResponseCount++
		if err := handleClaudeFormat(c, chatData, info); err != nil {
			return err
		}
	}
	return nil
}

func chatDataFrames(s string) []string {
	frames := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n\n")
	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		frame = strings.TrimSpace(frame)
		if frame == "" || strings.HasPrefix(frame, ":") {
			continue
		}
		var dataLines []string
		for _, line := range strings.Split(frame, "\n") {
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		if len(dataLines) > 0 {
			out = append(out, strings.Join(dataLines, "\n"))
		}
	}
	return out
}
