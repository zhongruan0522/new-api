package relay

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/relay/channel"
	"github.com/zhongruan0522/new-api/relay/channel/aws"
	"github.com/zhongruan0522/new-api/relay/channel/claude"
	"github.com/zhongruan0522/new-api/relay/channel/deepseek"
	"github.com/zhongruan0522/new-api/relay/channel/gemini"
	"github.com/zhongruan0522/new-api/relay/channel/minimax"
	"github.com/zhongruan0522/new-api/relay/channel/moonshot"
	"github.com/zhongruan0522/new-api/relay/channel/ollama"
	"github.com/zhongruan0522/new-api/relay/channel/openai"
	"github.com/zhongruan0522/new-api/relay/channel/siliconflow"
	taskGemini "github.com/zhongruan0522/new-api/relay/channel/task/gemini"
	"github.com/zhongruan0522/new-api/relay/channel/task/hailuo"
	"github.com/zhongruan0522/new-api/relay/channel/task/suno"
	taskvertex "github.com/zhongruan0522/new-api/relay/channel/task/vertex"
	"github.com/zhongruan0522/new-api/relay/channel/vertex"
	"github.com/zhongruan0522/new-api/relay/channel/zhipu_4v"
)

func GetAdaptor(apiType int) channel.Adaptor {
	switch apiType {
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeMoonshot:
		return &moonshot.Adaptor{}
	case constant.APITypeMiniMax:
		return &minimax.Adaptor{}
	}
	return nil
}

func GetTaskPlatform(c *gin.Context) constant.TaskPlatform {
	channelType := c.GetInt("channel_type")
	if channelType > 0 {
		return constant.TaskPlatform(strconv.Itoa(channelType))
	}
	return constant.TaskPlatform(c.GetString("platform"))
}

func GetTaskAdaptor(platform constant.TaskPlatform) channel.TaskAdaptor {
	switch platform {
	case constant.TaskPlatformSuno:
		return &suno.TaskAdaptor{}
	}
	if channelType, err := strconv.ParseInt(string(platform), 10, 64); err == nil {
		switch channelType {
		case constant.ChannelTypeVertexAi:
			return &taskvertex.TaskAdaptor{}
		case constant.ChannelTypeGemini:
			return &taskGemini.TaskAdaptor{}
		case constant.ChannelTypeMiniMax:
			return &hailuo.TaskAdaptor{}
		}
	}
	return nil
}
