import { THINK_TAG_REGEX, MESSAGE_ROLES } from './constants';

// 生成唯一ID
let messageId = 4;
export const generateMessageId = () => `${messageId++}`;

// 提取消息中的文本内容
export const getTextContent = (message) => {
  if (Array.isArray(message.content)) {
    const textContent = message.content.find(item => item.type === 'text');
    return textContent?.text || '';
  }
  return typeof message.content === 'string' ? message.content : '';
};

// 处理 think 标签
export const processThinkTags = (content, reasoningContent = '') => {
  if (!content.includes('<think>')) {
    return { content, reasoningContent };
  }

  let thoughts = [];
  let replyParts = [];
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

  let processedReasoningContent = reasoningContent;
  if (thoughts.length > 0) {
    const thoughtsStr = thoughts.join('\n\n---\n\n');
    processedReasoningContent = reasoningContent
      ? `${reasoningContent}\n\n---\n\n${thoughtsStr}`
      : thoughtsStr;
  }

  return {
    content: processedContent,
    reasoningContent: processedReasoningContent
  };
};

// 处理未完成的 think 标签
export const processIncompleteThinkTags = (content, reasoningContent = '') => {
  const lastOpenThinkIndex = content.lastIndexOf('<think>');
  if (lastOpenThinkIndex === -1) {
    return processThinkTags(content, reasoningContent);
  }

  const fragmentAfterLastOpen = content.substring(lastOpenThinkIndex);
  if (!fragmentAfterLastOpen.includes('</think>')) {
    const unclosedThought = fragmentAfterLastOpen.substring('<think>'.length).trim();
    const cleanContent = content.substring(0, lastOpenThinkIndex);

    let processedReasoningContent = reasoningContent;
    if (unclosedThought) {
      processedReasoningContent = reasoningContent
        ? `${reasoningContent}\n\n---\n\n${unclosedThought}`
        : unclosedThought;
    }

    return processThinkTags(cleanContent, processedReasoningContent);
  }

  return processThinkTags(content, reasoningContent);
};

// 构建消息内容（包含图片）
export const buildMessageContent = (textContent, imageUrls = [], imageEnabled = false) => {
  const validImageUrls = imageUrls.filter(url => url.trim() !== '');

  if (imageEnabled && validImageUrls.length > 0) {
    return [
      { type: 'text', text: textContent },
      ...validImageUrls.map(url => ({
        type: 'image_url',
        image_url: { url: url.trim() }
      }))
    ];
  }

  return textContent;
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
    status: 'loading'
  }
);

// 检查消息是否包含图片
export const hasImageContent = (message) => {
  return Array.isArray(message.content) &&
    message.content.some(item => item.type === 'image_url');
};

// 格式化消息用于API请求
export const formatMessageForAPI = (message) => ({
  role: message.role,
  content: message.content
}); 