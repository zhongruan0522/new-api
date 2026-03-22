package relay

import (
	"strconv"

	"github.com/zhongruan0522/new-api/constant"
	"github.com/zhongruan0522/new-api/relay/channel"
	"github.com/zhongruan0522/new-api/relay/channel/ali"
	"github.com/zhongruan0522/new-api/relay/channel/aws"
	"github.com/zhongruan0522/new-api/relay/channel/claude"
	"github.com/zhongruan0522/new-api/relay/channel/cloudflare"
	"github.com/zhongruan0522/new-api/relay/channel/cohere"
	"github.com/zhongruan0522/new-api/relay/channel/deepseek"
	"github.com/zhongruan0522/new-api/relay/channel/dify"
	"github.com/zhongruan0522/new-api/relay/channel/gemini"
	"github.com/zhongruan0522/new-api/relay/channel/jina"
	"github.com/zhongruan0522/new-api/relay/channel/minimax"
	"github.com/zhongruan0522/new-api/relay/channel/mistral"
	"github.com/zhongruan0522/new-api/relay/channel/moonshot"
	"github.com/zhongruan0522/new-api/relay/channel/ollama"
	"github.com/zhongruan0522/new-api/relay/channel/openai"
	"github.com/zhongruan0522/new-api/relay/channel/palm"
	"github.com/zhongruan0522/new-api/relay/channel/siliconflow"
	taskali "github.com/zhongruan0522/new-api/relay/channel/task/ali"
	taskdoubao "github.com/zhongruan0522/new-api/relay/channel/task/doubao"
	taskGemini "github.com/zhongruan0522/new-api/relay/channel/task/gemini"
	"github.com/zhongruan0522/new-api/relay/channel/task/hailuo"
	"github.com/zhongruan0522/new-api/relay/channel/task/suno"
	taskvertex "github.com/zhongruan0522/new-api/relay/channel/task/vertex"
	"github.com/zhongruan0522/new-api/relay/channel/tencent"
	"github.com/zhongruan0522/new-api/relay/channel/vertex"
	"github.com/zhongruan0522/new-api/relay/channel/volcengine"
	"github.com/zhongruan0522/new-api/relay/channel/xai"
	"github.com/zhongruan0522/new-api/relay/channel/xunfei"
	"github.com/zhongruan0522/new-api/relay/channel/zhipu_4v"
	"github.com/gin-gonic/gin"
)

func GetAdaptor(apiType int) channel.Adaptor {
	switch apiType {
	case constant.APITypeAli:
		return &ali.Adaptor{}
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypePaLM:
		return &palm.Adaptor{}
	case constant.APITypeTencent:
		return &tencent.Adaptor{}
	case constant.APITypeXunfei:
		return &xunfei.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeCohere:
		return &cohere.Adaptor{}
	case constant.APITypeDify:
		return &dify.Adaptor{}
	case constant.APITypeJina:
		return &jina.Adaptor{}
	case constant.APITypeCloudflare:
		return &cloudflare.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeMistral:
		return &mistral.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeVolcEngine:
		return &volcengine.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeXai:
		return &xai.Adaptor{}
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
		case constant.ChannelTypeAli:
			return &taskali.TaskAdaptor{}
		case constant.ChannelTypeVertexAi:
			return &taskvertex.TaskAdaptor{}
		case constant.ChannelTypeVolcEngine:
			return &taskdoubao.TaskAdaptor{}
		case constant.ChannelTypeGemini:
			return &taskGemini.TaskAdaptor{}
		case constant.ChannelTypeMiniMax:
			return &hailuo.TaskAdaptor{}
		}
	}
	return nil
}
