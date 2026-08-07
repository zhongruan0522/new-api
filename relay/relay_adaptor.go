package relay

import (
	"github.com/NookMux/NookMux/constant"
	"github.com/NookMux/NookMux/relay/channel"
	"github.com/NookMux/NookMux/relay/channel/aws"
	"github.com/NookMux/NookMux/relay/channel/bytedance"
	"github.com/NookMux/NookMux/relay/channel/claude"
	"github.com/NookMux/NookMux/relay/channel/deepseek"
	"github.com/NookMux/NookMux/relay/channel/gemini"
	"github.com/NookMux/NookMux/relay/channel/minimax"
	"github.com/NookMux/NookMux/relay/channel/moonshot"
	"github.com/NookMux/NookMux/relay/channel/ollama"
	"github.com/NookMux/NookMux/relay/channel/openai"
	"github.com/NookMux/NookMux/relay/channel/siliconflow"
	"github.com/NookMux/NookMux/relay/channel/vertex"
	"github.com/NookMux/NookMux/relay/channel/xiaomi"
	"github.com/NookMux/NookMux/relay/channel/zhipu_4v"
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
	case constant.APITypeByteDance:
		return &bytedance.Adaptor{}
	}
	return nil
}
