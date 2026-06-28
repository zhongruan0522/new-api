package zhipu_4v

import (
	"strings"

	"github.com/zhongruan0522/new-api/dto"
)

func requestOpenAI2Zhipu(request dto.GeneralOpenAIRequest) *dto.GeneralOpenAIRequest {
	messages := make([]dto.Message, 0, len(request.Messages))
	for _, message := range request.Messages {
		// 智谱只接受 system/user/assistant/tool 角色。OpenAI o 系列/gpt-5 使用 "developer"
		// 作为系统角色（Responses → Chat 转换会按模型名生成它）。此外智谱对多条 system
		// 消息会相互覆盖导致内容丢失，因此将连续的 system/developer 消息合并为一条
		// system，文本以换行拼接，避免丢内容。
		if isZhipuSystemRole(message.Role) {
			text := strings.TrimSpace(message.StringContent())
			if text == "" {
				continue
			}
			if n := len(messages); n > 0 && strings.EqualFold(messages[n-1].Role, "system") {
				appendZhipuSystemText(&messages[n-1], text)
				continue
			}
			messages = append(messages, dto.Message{Role: "system", Content: text})
			continue
		}
		if !message.IsStringContent() {
			mediaMessages := message.ParseContent()
			for j, mediaMessage := range mediaMessages {
				if mediaMessage.Type == dto.ContentTypeImageURL {
					imageUrl := mediaMessage.GetImageMedia()
					// check if base64
					if strings.HasPrefix(imageUrl.Url, "data:image/") {
						// 去除base64数据的URL前缀（如果有）
						if idx := strings.Index(imageUrl.Url, ","); idx != -1 {
							imageUrl.Url = imageUrl.Url[idx+1:]
						}
					}
					mediaMessage.ImageUrl = imageUrl
					mediaMessages[j] = mediaMessage
				}
			}
			message.SetMediaContent(mediaMessages)
		}
		messages = append(messages, dto.Message{
			Role:       message.Role,
			Content:    message.Content,
			ToolCalls:  message.ToolCalls,
			ToolCallId: message.ToolCallId,
		})
	}
	str, ok := request.Stop.(string)
	var Stop []string
	if ok {
		Stop = []string{str}
	} else {
		Stop, _ = request.Stop.([]string)
	}
	return &dto.GeneralOpenAIRequest{
		Model:       request.Model,
		Stream:      request.Stream,
		Messages:    messages,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		MaxTokens:   request.GetMaxTokens(),
		Stop:        Stop,
		Tools:       request.Tools,
		ToolChoice:  request.ToolChoice,
		THINKING:    request.THINKING,
	}
}

// isZhipuSystemRole 判断是否为需要归一化的系统类角色（system/developer）。
func isZhipuSystemRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "system", "developer":
		return true
	}
	return false
}

// appendZhipuSystemText 将文本追加到已有 system 消息，文本以换行分隔。
func appendZhipuSystemText(msg *dto.Message, addition string) {
	if addition == "" {
		return
	}
	existing := strings.TrimSpace(msg.StringContent())
	if existing == "" {
		msg.SetStringContent(addition)
		return
	}
	msg.SetStringContent(existing + "\n" + addition)
}
