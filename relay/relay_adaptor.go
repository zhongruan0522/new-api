package relay

import (
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
	"github.com/zhongruan0522/new-api/relay/channel/vertex"
	"github.com/zhongruan0522/new-api/relay/channel/xiaomi"
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
	case constant.APITypeXiaomi:
		return &xiaomi.Adaptor{}
	}
	return nil
}
