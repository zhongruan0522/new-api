package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/infra/log"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	relayconstant "github.com/NookMux/NookMux/internal/relay/constant"
	"github.com/NookMux/NookMux/internal/relay/helper"

	sensitive "github.com/NookMux/NookMux/internal/domain/sensitive"
	"github.com/NookMux/NookMux/internal/httpapi"
	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
)

func OpenaiTTSHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) *shared.Usage {
	info.UsageSource = relayconstant.UsageSourceOpenAIChat
	// the status code has been judged before, if there is a body reading failure,
	// it should be regarded as a non-recoverable error, so it should not return err for external retry.
	// Analogous to nginx's load balancing, it will only retry if it can't be requested or
	// if the upstream returns a specific status code, once the upstream has already written the header,
	// the subsequent failure of the response body should be regarded as a non-recoverable error,
	// and can be terminated directly.
	defer helper.CloseResponseBodyGracefully(resp)
	usage := &shared.Usage{}
	usage.PromptTokens = info.GetEstimatePromptTokens()
	usage.TotalTokens = info.GetEstimatePromptTokens()
	for k, v := range resp.Header {
		if !helper.ShouldCopyUpstreamHeader(c, k, v) {
			continue
		}
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)

	if info.IsStream {
		upstreamUsageSeen := false
		helper.StreamScannerHandler(c, resp, info, func(data string) bool {
			if sensitive.SundaySearch(data, "usage") {
				var simpleResponse shared.SimpleResponse
				err := jsonx.Unmarshal([]byte(data), &simpleResponse)
				if err != nil {
					log.LogError(c, err.Error())
				}
				if simpleResponse.Usage.TotalTokens != 0 {
					upstreamUsageSeen = true
					usage.PromptTokens = simpleResponse.Usage.InputTokens
					usage.CompletionTokens = simpleResponse.OutputTokens
					usage.TotalTokens = simpleResponse.TotalTokens
				}
			}
			_ = helper.StringData(c, data)
			return true
		})
		if !upstreamUsageSeen {
			// 流式响应未出现上游 usage：初始估算值继续用于计费（现有行为），
			// 但不属于上游 Token 用量，billing_details 不落列。
			httpapi.SetContextKey(c, common.ContextKeyLocalCountTokens, true)
		}
	} else {
		httpapi.SetContextKey(c, common.ContextKeyLocalCountTokens, true)
		// 读取响应体到缓冲区
		bodyBytes, err := helper.ReadMediaResponseBody(resp.Body)
		if err != nil {
			log.LogError(c, fmt.Sprintf("failed to read TTS response body: %v", err))
			c.Writer.WriteHeaderNow()
			return usage
		}

		// 写入响应到客户端
		c.Writer.WriteHeaderNow()
		_, err = c.Writer.Write(bodyBytes)
		if err != nil {
			log.LogError(c, fmt.Sprintf("failed to write TTS response: %v", err))
		}

		// 计算音频时长并更新 usage
		audioFormat := "mp3" // 默认格式
		if audioReq, ok := info.Request.(*shared.AudioRequest); ok && audioReq.ResponseFormat != "" {
			audioFormat = audioReq.ResponseFormat
		}

		var duration float64
		var durationErr error

		if audioFormat == "pcm" {
			// PCM 格式没有文件头，根据 OpenAI TTS 的 PCM 参数计算时长
			// 采样率: 24000 Hz, 位深度: 16-bit (2 bytes), 声道数: 1
			const sampleRate = 24000
			const bytesPerSample = 2
			const channels = 1
			duration = float64(len(bodyBytes)) / float64(sampleRate*bytesPerSample*channels)
		} else {
			ext := "." + audioFormat
			reader := bytes.NewReader(bodyBytes)
			duration, durationErr = common.GetAudioDuration(c.Request.Context(), reader, ext)
		}

		usage.PromptTokensDetails.TextTokens = usage.PromptTokens

		if durationErr != nil {
			log.LogWarn(c, fmt.Sprintf("failed to get audio duration: %v", durationErr))
			// 如果无法获取时长，则设置保底的 CompletionTokens，根据body大小计算
			sizeInKB := float64(len(bodyBytes)) / 1000.0
			estimatedTokens := int(math.Ceil(sizeInKB)) // 粗略估算每KB约等于1 token
			usage.CompletionTokens = estimatedTokens
			usage.CompletionTokenDetails.AudioTokens = estimatedTokens
		} else if duration > 0 {
			// 计算 token: ceil(duration) / 60.0 * 1000，即每分钟 1000 tokens
			completionTokens := int(math.Round(math.Ceil(duration) / 60.0 * 1000))
			usage.CompletionTokens = completionTokens
			usage.CompletionTokenDetails.AudioTokens = completionTokens
		}
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return usage
}

func OpenaiSTTHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, responseFormat string) (*shared.NookMuxError, *shared.Usage) {
	// STT 真实 usage 按 prompt/completion 口径上报，语义为 OpenAI Chat 族。
	info.UsageSource = relayconstant.UsageSourceOpenAIChat
	defer helper.CloseResponseBodyGracefully(resp)

	responseBody, err := helper.ReadResponseBody(resp.Body)
	if err != nil {
		return shared.NewOpenAIError(err, shared.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError), nil
	}
	// 写入新的 response body
	helper.IOCopyBytesGracefully(c, resp, responseBody)

	var responseData struct {
		Usage json.RawMessage `json:"usage"`
	}
	if err := jsonx.Unmarshal(responseBody, &responseData); err == nil && len(responseData.Usage) > 0 {
		usage := &shared.Usage{}
		if err := jsonx.Unmarshal(responseData.Usage, usage); err == nil && usage.TotalTokens > 0 {
			// STT 上游 usage 用单数 input_token_details 承载输入明细（Chat/Responses
			// 用复数 input_tokens_details，shared.Usage 的 tag 匹配不到），单独补入
			// InputTokensDetails 供 billing_details 归一化读取官方缓存/文本/音频拆分。
			// 只写 InputTokensDetails 不写 PromptTokensDetails：audio_handler 按
			// PromptTokensDetails.AudioTokens 分流 PostAudioConsumeQuota，
			// 写入会改变既有计费路径与 quota（阶段 1 不切公式）。
			var sttUsageDetails struct {
				InputTokenDetails *shared.InputTokenDetails `json:"input_token_details"`
			}
			if err := jsonx.Unmarshal(responseData.Usage, &sttUsageDetails); err == nil && sttUsageDetails.InputTokenDetails != nil {
				usage.InputTokensDetails = sttUsageDetails.InputTokenDetails
			}
			if usage.PromptTokens == 0 {
				usage.PromptTokens = usage.InputTokens
			}
			if usage.CompletionTokens == 0 {
				usage.CompletionTokens = usage.OutputTokens
			}
			return nil, usage
		}
	}

	usage := &shared.Usage{}
	usage.PromptTokens = info.GetEstimatePromptTokens()
	usage.CompletionTokens = 0
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	// usage 为本地估算，不属于上游 Token 用量，billing_details 不落列。
	httpapi.SetContextKey(c, common.ContextKeyLocalCountTokens, true)
	return nil, usage
}
