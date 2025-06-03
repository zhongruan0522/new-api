import { formatMessageForAPI } from './messageUtils';

// 构建API请求载荷
export const buildApiPayload = (messages, systemPrompt, inputs, parameterEnabled) => {
  const processedMessages = messages.map(formatMessageForAPI);

  // 如果有系统提示，插入到消息开头
  if (systemPrompt && systemPrompt.trim()) {
    processedMessages.unshift({
      role: 'system',
      content: systemPrompt.trim()
    });
  }

  const payload = {
    model: inputs.model,
    messages: processedMessages,
    stream: inputs.stream,
  };

  // 添加启用的参数
  if (parameterEnabled.temperature && inputs.temperature !== undefined) {
    payload.temperature = inputs.temperature;
  }
  if (parameterEnabled.top_p && inputs.top_p !== undefined) {
    payload.top_p = inputs.top_p;
  }
  if (parameterEnabled.max_tokens && inputs.max_tokens !== undefined) {
    payload.max_tokens = inputs.max_tokens;
  }
  if (parameterEnabled.frequency_penalty && inputs.frequency_penalty !== undefined) {
    payload.frequency_penalty = inputs.frequency_penalty;
  }
  if (parameterEnabled.presence_penalty && inputs.presence_penalty !== undefined) {
    payload.presence_penalty = inputs.presence_penalty;
  }
  if (parameterEnabled.seed && inputs.seed !== undefined && inputs.seed !== null) {
    payload.seed = inputs.seed;
  }

  return payload;
};

// 处理API错误响应
export const handleApiError = (error, response = null) => {
  const errorInfo = {
    error: error.message || '未知错误',
    timestamp: new Date().toISOString(),
    stack: error.stack
  };

  if (response) {
    errorInfo.status = response.status;
    errorInfo.statusText = response.statusText;
  }

  if (error.message.includes('HTTP error')) {
    errorInfo.details = '服务器返回了错误状态码';
  } else if (error.message.includes('Failed to fetch')) {
    errorInfo.details = '网络连接失败或服务器无响应';
  }

  return errorInfo;
};

// 处理模型数据
export const processModelsData = (data, currentModel) => {
  const modelOptions = data.map(model => ({
    label: model,
    value: model,
  }));

  const hasCurrentModel = modelOptions.some(option => option.value === currentModel);
  const selectedModel = hasCurrentModel && modelOptions.length > 0
    ? currentModel
    : modelOptions[0]?.value;

  return { modelOptions, selectedModel };
};

// 处理分组数据
export const processGroupsData = (data, userGroup) => {
  let groupOptions = Object.entries(data).map(([group, info]) => ({
    label: info.desc.length > 20 ? info.desc.substring(0, 20) + '...' : info.desc,
    value: group,
    ratio: info.ratio,
    fullLabel: info.desc,
  }));

  if (groupOptions.length === 0) {
    groupOptions = [{
      label: '用户分组',
      value: '',
      ratio: 1,
    }];
  } else if (userGroup) {
    const userGroupIndex = groupOptions.findIndex(g => g.value === userGroup);
    if (userGroupIndex > -1) {
      const userGroupOption = groupOptions.splice(userGroupIndex, 1)[0];
      groupOptions.unshift(userGroupOption);
    }
  }

  return groupOptions;
}; 