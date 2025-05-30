import { formatMessageForAPI } from './messageUtils';

// 构建API请求载荷
export const buildApiPayload = (messages, systemMessage, inputs, parameterEnabled) => {
  const formattedMessages = messages.map(formatMessageForAPI);

  if (systemMessage) {
    formattedMessages.unshift(formatMessageForAPI(systemMessage));
  }

  const payload = {
    messages: formattedMessages,
    stream: inputs.stream,
    model: inputs.model,
    group: inputs.group,
  };

  // 添加可选参数
  const optionalParams = [
    'max_tokens', 'temperature', 'top_p',
    'frequency_penalty', 'presence_penalty', 'seed'
  ];

  optionalParams.forEach(param => {
    if (parameterEnabled[param]) {
      if (param === 'max_tokens' && inputs[param] > 0) {
        payload[param] = parseInt(inputs[param]);
      } else if (param === 'seed' && inputs[param] !== null && inputs[param] !== '') {
        payload[param] = parseInt(inputs[param]);
      } else if (param !== 'max_tokens' && param !== 'seed') {
        payload[param] = inputs[param];
      }
    }
  });

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