import { THINK_TAG_REGEX, MESSAGE_ROLES } from '../constants/playground.constants';

// 生成唯一ID
let messageId = 4;
export const generateMessageId = () => `${messageId++}`;

// 提取消息中的文本内容
export const getTextContent = (message) => {
  if (!message || !message.content) return '';

  if (Array.isArray(message.content)) {
    const textContent = message.content.find(item => item.type === 'text');
    return textContent?.text || '';
  }
  return typeof message.content === 'string' ? message.content : '';
};

// 处理 think 标签
export const processThinkTags = (content, reasoningContent = '') => {
  if (!content || !content.includes('<think>')) {
    return { content, reasoningContent };
  }

  const thoughts = [];
  const replyParts = [];
  let lastIndex = 0;
  let match;

  THINK_TAG_REGEX.lastIndex = 0;
  while ((match = THINK_TAG_REGEX.exec(content)) !== null) {
    replyParts.push(content.substring(lastIndex, match.index));
    thoughts.push(match[1]);
    lastIndex = match.index + match[0].length;
  }
  replyParts.push(content.substring(lastIndex));

  const processedContent = replyParts.join('').replace(/<\/?think>/g, '').trim();
  const thoughtsStr = thoughts.join('\n\n---\n\n');
  const processedReasoningContent = reasoningContent && thoughtsStr
    ? `${reasoningContent}\n\n---\n\n${thoughtsStr}`
    : reasoningContent || thoughtsStr;

  return {
    content: processedContent,
    reasoningContent: processedReasoningContent
  };
};

// 处理未完成的 think 标签
export const processIncompleteThinkTags = (content, reasoningContent = '') => {
  if (!content) return { content: '', reasoningContent };

  const lastOpenThinkIndex = content.lastIndexOf('<think>');
  if (lastOpenThinkIndex === -1) {
    return processThinkTags(content, reasoningContent);
  }

  const fragmentAfterLastOpen = content.substring(lastOpenThinkIndex);
  if (!fragmentAfterLastOpen.includes('</think>')) {
    const unclosedThought = fragmentAfterLastOpen.substring('<think>'.length).trim();
    const cleanContent = content.substring(0, lastOpenThinkIndex);
    const processedReasoningContent = unclosedThought
      ? reasoningContent ? `${reasoningContent}\n\n---\n\n${unclosedThought}` : unclosedThought
      : reasoningContent;

    return processThinkTags(cleanContent, processedReasoningContent);
  }

  return processThinkTags(content, reasoningContent);
};

// 构建消息内容（包含图片）
export const buildMessageContent = (textContent, imageUrls = [], imageEnabled = false) => {
  if (!textContent && (!imageUrls || imageUrls.length === 0)) {
    return '';
  }

  const validImageUrls = imageUrls.filter(url => url && url.trim() !== '');

  if (imageEnabled && validImageUrls.length > 0) {
    return [
      { type: 'text', text: textContent || '' },
      ...validImageUrls.map(url => ({
        type: 'image_url',
        image_url: { url: url.trim() }
      }))
    ];
  }

  return textContent || '';
};

// 创建新消息
export const createMessage = (role, content, options = {}) => ({
  role,
  content,
  createAt: Date.now(),
  id: generateMessageId(),
  ...options
});

// 创建加载中的助手消息
export const createLoadingAssistantMessage = () => createMessage(
  MESSAGE_ROLES.ASSISTANT,
  '',
  {
    reasoningContent: '',
    isReasoningExpanded: true,
    isThinkingComplete: false,
    hasAutoCollapsed: false,
    status: 'loading'
  }
);

// 检查消息是否包含图片
export const hasImageContent = (message) => {
  return message &&
    Array.isArray(message.content) &&
    message.content.some(item => item.type === 'image_url');
};

// 格式化消息用于API请求
export const formatMessageForAPI = (message) => {
  if (!message) return null;

  return {
    role: message.role,
    content: message.content
  };
};

// 验证消息是否有效
export const isValidMessage = (message) => {
  return message &&
    message.role &&
    (message.content || message.content === '');
};

// 获取最后一条用户消息
export const getLastUserMessage = (messages) => {
  if (!Array.isArray(messages)) return null;

  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === MESSAGE_ROLES.USER) {
      return messages[i];
    }
  }
  return null;
};

// 获取最后一条助手消息
export const getLastAssistantMessage = (messages) => {
  if (!Array.isArray(messages)) return null;

  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === MESSAGE_ROLES.ASSISTANT) {
      return messages[i];
    }
  }
  return null;
};

// 构建API请求负载（从apiUtils移动过来）
export const buildApiPayload = (messages, systemPrompt, inputs, parameterEnabled) => {
  const processedMessages = messages
    .filter(isValidMessage)
    .map(formatMessageForAPI)
    .filter(Boolean);

  // 如果有系统提示，插入到消息开头
  if (systemPrompt && systemPrompt.trim()) {
    processedMessages.unshift({
      role: MESSAGE_ROLES.SYSTEM,
      content: systemPrompt.trim()
    });
  }

  const payload = {
    model: inputs.model,
    messages: processedMessages,
    stream: inputs.stream,
  };

  // 添加启用的参数
  const parameterMappings = {
    temperature: 'temperature',
    top_p: 'top_p',
    max_tokens: 'max_tokens',
    frequency_penalty: 'frequency_penalty',
    presence_penalty: 'presence_penalty',
    seed: 'seed'
  };

  Object.entries(parameterMappings).forEach(([key, param]) => {
    if (parameterEnabled[key] && inputs[param] !== undefined && inputs[param] !== null) {
      payload[param] = inputs[param];
    }
  });

  return payload;
}; 