// Playground 相关常量
export const DEFAULT_MESSAGES = [
  {
    role: 'user',
    id: '2',
    createAt: 1715676751919,
    content: '你好',
  },
  {
    role: 'assistant',
    id: '3',
    createAt: 1715676751919,
    content: '你好，请问有什么可以帮助您的吗？',
    reasoningContent: '',
    isReasoningExpanded: false,
  },
];

export const MESSAGE_STATUS = {
  LOADING: 'loading',
  INCOMPLETE: 'incomplete',
  COMPLETE: 'complete',
  ERROR: 'error',
};

export const MESSAGE_ROLES = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system',
};

export const DEBUG_TABS = {
  PREVIEW: 'preview',
  REQUEST: 'request',
  RESPONSE: 'response',
};

export const API_ENDPOINTS = {
  CHAT_COMPLETIONS: '/pg/chat/completions',
  USER_MODELS: '/api/user/models',
  USER_GROUPS: '/api/user/self/groups',
};

export const DEFAULT_CONFIG = {
  inputs: {
    model: 'gpt-4',
    group: '',
    temperature: 0.7,
    top_p: 1,
    max_tokens: 2048,
    frequency_penalty: 0,
    presence_penalty: 0,
    seed: null,
    stream: true,
    imageEnabled: false,
    imageUrls: [''],
  },
  parameterEnabled: {
    temperature: true,
    top_p: false,
    max_tokens: false,
    frequency_penalty: false,
    presence_penalty: false,
    seed: false,
  },
  systemPrompt: '',
  showDebugPanel: false,
};

export const THINK_TAG_REGEX = /<think>([\s\S]*?)<\/think>/g;

export const ERROR_MESSAGES = {
  NO_TEXT_CONTENT: '此消息没有可复制的文本内容',
  INVALID_MESSAGE_TYPE: '无法复制此类型的消息内容',
  COPY_FAILED: '复制失败，请手动选择文本复制',
  COPY_HTTPS_REQUIRED: '复制功能需要 HTTPS 环境，请手动复制',
  BROWSER_NOT_SUPPORTED: '浏览器不支持复制功能，请手动复制',
}; 