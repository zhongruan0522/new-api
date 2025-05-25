import {
    OpenAI,
    Claude,
    Gemini,
    Moonshot,
    Zhipu,
    Qwen,
    DeepSeek,
    Minimax,
    Wenxin,
    Spark,
    Midjourney,
    Hunyuan,
    Cohere,
    Cloudflare,
    Ai360,
    Yi,
    Jina,
    Mistral,
    XAI,
    Ollama,
    Doubao,
} from '@lobehub/icons';

export const MODEL_CATEGORIES = (t) => ({
    all: {
        label: t('全部模型'),
        icon: null,
        filter: () => true
    },
    openai: {
        label: 'OpenAI',
        icon: <OpenAI />,
        filter: (model) => model.model_name.toLowerCase().includes('gpt') ||
            model.model_name.toLowerCase().includes('dall-e') ||
            model.model_name.toLowerCase().includes('whisper') ||
            model.model_name.toLowerCase().includes('tts') ||
            model.model_name.toLowerCase().includes('text-') ||
            model.model_name.toLowerCase().includes('babbage') ||
            model.model_name.toLowerCase().includes('davinci') ||
            model.model_name.toLowerCase().includes('curie') ||
            model.model_name.toLowerCase().includes('ada')
    },
    anthropic: {
        label: 'Anthropic',
        icon: <Claude.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('claude')
    },
    gemini: {
        label: 'Gemini',
        icon: <Gemini.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('gemini')
    },
    moonshot: {
        label: 'Moonshot',
        icon: <Moonshot />,
        filter: (model) => model.model_name.toLowerCase().includes('moonshot')
    },
    zhipu: {
        label: t('智谱'),
        icon: <Zhipu.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('chatglm') ||
            model.model_name.toLowerCase().includes('glm-')
    },
    qwen: {
        label: t('通义千问'),
        icon: <Qwen.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('qwen')
    },
    deepseek: {
        label: 'DeepSeek',
        icon: <DeepSeek.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('deepseek')
    },
    minimax: {
        label: 'MiniMax',
        icon: <Minimax.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('abab')
    },
    baidu: {
        label: t('文心一言'),
        icon: <Wenxin.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('ernie')
    },
    xunfei: {
        label: t('讯飞星火'),
        icon: <Spark.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('spark')
    },
    midjourney: {
        label: 'Midjourney',
        icon: <Midjourney />,
        filter: (model) => model.model_name.toLowerCase().includes('mj_')
    },
    tencent: {
        label: t('腾讯混元'),
        icon: <Hunyuan.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('hunyuan')
    },
    cohere: {
        label: 'Cohere',
        icon: <Cohere.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('command')
    },
    cloudflare: {
        label: 'Cloudflare',
        icon: <Cloudflare.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('@cf/')
    },
    ai360: {
        label: t('360智脑'),
        icon: <Ai360.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('360')
    },
    yi: {
        label: t('零一万物'),
        icon: <Yi.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('yi')
    },
    jina: {
        label: 'Jina',
        icon: <Jina />,
        filter: (model) => model.model_name.toLowerCase().includes('jina')
    },
    mistral: {
        label: 'Mistral AI',
        icon: <Mistral.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('mistral')
    },
    xai: {
        label: 'xAI',
        icon: <XAI />,
        filter: (model) => model.model_name.toLowerCase().includes('grok')
    },
    llama: {
        label: 'Llama',
        icon: <Ollama />,
        filter: (model) => model.model_name.toLowerCase().includes('llama')
    },
    doubao: {
        label: t('豆包'),
        icon: <Doubao.Color />,
        filter: (model) => model.model_name.toLowerCase().includes('doubao')
    }
}); 