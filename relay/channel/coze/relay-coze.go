package coze

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"one-api/dto"
	"one-api/relay/common"
	relaycommon "one-api/relay/common"
	"one-api/service"

	"github.com/gin-gonic/gin"
)

func convertCozeChatRequest(request dto.GeneralOpenAIRequest) *CozeRequest {
	var messages []CozeEnterMessage
	// 将 request的messages的role为user的content转换为CozeMessage
	for _, message := range request.Messages {
		if message.Role == "user" {
			messages = append(messages, CozeEnterMessage{
				Role:    "user",
				Content: message.Content,
				// TODO: support more content type
				ContentType: "text",
			})
		}
	}
	cozeRequest := &CozeRequest{
		// TODO: model to botid
		BotId:    "1",
		Messages: messages,
	}
	return cozeRequest
}

func cozeChatHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return service.OpenAIErrorWrapperLocal(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	// convert coze response to openai response
	var response dto.TextResponse
	var cozeResponse CozeChatResponse
	err = json.Unmarshal(responseBody, &cozeResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	response.Model = info.UpstreamModelName
	// TODO: 处理 cozeResponse
	return nil, nil
}

func checkIfChatComplete(a *Adaptor, c *gin.Context, info *relaycommon.RelayInfo) (error, bool) {
	requestURL := fmt.Sprintf("%s/v3/chat/retrieve", info.BaseUrl)

	requestURL = requestURL + "?conversation_id=" + c.GetString("coze_conversation_id") + "&chat_id=" + c.GetString("coze_chat_id")
	// 将 conversationId和chatId作为参数发送get请求
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return err, false
	}
	err = a.SetupRequestHeader(c, &req.Header, info)
	if err != nil {
		return err, false
	}

	resp, err := doRequest(req, info) // 调用 doRequest
	if err != nil {
		return err, false
	}
	if resp == nil { // 确保在 doRequest 失败时 resp 不为 nil 导致 panic
		return fmt.Errorf("resp is nil"), false
	}
	defer resp.Body.Close() // 确保响应体被关闭

	// 解析 resp 到 CozeChatResponse
	var cozeResponse CozeChatResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err), false
	}
	err = json.Unmarshal(responseBody, &cozeResponse)
	if err != nil {
		return fmt.Errorf("unmarshal response body failed: %w", err), false
	}
	if cozeResponse.Data.Status == "completed" {
		// 在上下文设置 usage
		c.Set("coze_token_count", cozeResponse.Data.Usage.TokenCount)
		c.Set("coze_output_count", cozeResponse.Data.Usage.OutputCount)
		c.Set("coze_input_count", cozeResponse.Data.Usage.InputCount)
		return nil, true
	} else if cozeResponse.Data.Status == "failed" || cozeResponse.Data.Status == "canceled" || cozeResponse.Data.Status == "requires_action" {
		return fmt.Errorf("chat status: %s", cozeResponse.Data.Status), false
	} else {
		return nil, false
	}
}

func doRequest(req *http.Request, info *common.RelayInfo) (*http.Response, error) {
	var client *http.Client
	var err error // 声明 err 变量
	if proxyURL, ok := info.ChannelSetting["proxy"]; ok {
		client, err = service.NewProxyHttpClient(proxyURL.(string))
		if err != nil {
			return nil, fmt.Errorf("new proxy http client failed: %w", err)
		}
	} else {
		client = service.GetHttpClient()
	}
	resp, err := client.Do(req)
	if err != nil { // 增加对 client.Do(req) 返回错误的检查
		return nil, fmt.Errorf("client.Do failed: %w", err)
	}
	_ = resp.Body.Close()
	return resp, nil
}
