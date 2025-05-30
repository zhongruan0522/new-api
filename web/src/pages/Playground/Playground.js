import React, { useCallback, useContext, useEffect, useState, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User/index.js';
import {
  API,
  getUserIdFromLocalStorage,
  showError,
  getLogo,
  isMobile,
} from '../../helpers/index.js';
import {
  Layout,
  Toast,
  Modal,
} from '@douyinfe/semi-ui';
import { SSE } from 'sse';
import { StyleContext } from '../../context/Style/index.js';
import { useTranslation } from 'react-i18next';
import { stringToColor } from '../../helpers/render.js';

import SettingsPanel from '../../components/playground/SettingsPanel';
import ChatArea from '../../components/playground/ChatArea';
import DebugPanel from '../../components/playground/DebugPanel';
import MessageContent from '../../components/playground/MessageContent';
import MessageActions from '../../components/playground/MessageActions';
import FloatingButtons from '../../components/playground/FloatingButtons';

import { saveConfig, loadConfig } from '../../components/playground/configStorage';

let id = 4;
function getId() {
  return `${id++}`;
}

const generateAvatarDataUrl = (username) => {
  if (!username) {
    return 'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/docs-icon.png';
  }
  const firstLetter = username[0].toUpperCase();
  const bgColor = stringToColor(username);
  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32">
      <circle cx="16" cy="16" r="16" fill="${bgColor}" />
      <text x="50%" y="50%" dominant-baseline="central" text-anchor="middle" font-size="16" fill="#ffffff" font-family="sans-serif">${firstLetter}</text>
    </svg>
  `;
  return `data:image/svg+xml;base64,${btoa(svg)}`;
};

const Playground = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);

  const roleInfo = {
    user: {
      name: userState?.user?.username || 'User',
      avatar: generateAvatarDataUrl(userState?.user?.username),
    },
    assistant: {
      name: 'Assistant',
      avatar: getLogo(),
    },
    system: {
      name: 'System',
      avatar:
        'https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/other/logo.png',
    },
  };

  const defaultMessage = [
    {
      role: 'user',
      id: '2',
      createAt: 1715676751919,
      content: t('你好'),
    },
    {
      role: 'assistant',
      id: '3',
      createAt: 1715676751919,
      content: t('你好，请问有什么可以帮助您的吗？'),
      reasoningContent: '',
      isReasoningExpanded: false,
    },
  ];

  const savedConfig = loadConfig();

  const [inputs, setInputs] = useState(savedConfig.inputs);
  const [parameterEnabled, setParameterEnabled] = useState(savedConfig.parameterEnabled);
  const [systemPrompt, setSystemPrompt] = useState(savedConfig.systemPrompt);
  const [showDebugPanel, setShowDebugPanel] = useState(savedConfig.showDebugPanel);

  const [searchParams, setSearchParams] = useSearchParams();
  const [status, setStatus] = useState({});
  const [message, setMessage] = useState(defaultMessage);
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [showSettings, setShowSettings] = useState(false);
  const [debugData, setDebugData] = useState({
    request: null,
    response: null,
    timestamp: null
  });
  const [activeDebugTab, setActiveDebugTab] = useState('preview');
  const [styleState, styleDispatch] = useContext(StyleContext);
  const sseSourceRef = useRef(null);
  const chatRef = useRef(null);

  const saveConfigTimeoutRef = useRef(null);

  const [previewPayload, setPreviewPayload] = useState(null);

  const constructPreviewPayload = useCallback(() => {
    try {
      let systemMessage = null;
      if (systemPrompt !== '') {
        systemMessage = {
          role: 'system',
          id: '1',
          createAt: 1715676751919,
          content: systemPrompt,
        };
      }
      
      let messages = message.map((item) => {
        return {
          role: item.role,
          content: item.content,
        };
      });
      
      if (messages.length === 0 || messages.every(msg => msg.role !== 'user')) {
        const validImageUrls = inputs.imageUrls ? inputs.imageUrls.filter(url => url.trim() !== '') : [];
        
        if (inputs.imageEnabled && validImageUrls.length > 0) {
          const messageContent = [
            {
              type: 'text',
              text: '你好'
            },
            ...validImageUrls.map(url => ({
              type: 'image_url',
              image_url: {
                url: url.trim(),
              },
            })),
          ];
          
          messages.push({
            role: 'user',
            content: messageContent
          });
        } else {
          messages.push({
            role: 'user',
            content: '你好'
          });
        }
      } else {
        const lastUserMessageIndex = messages.length - 1;
        for (let i = messages.length - 1; i >= 0; i--) {
          if (messages[i].role === 'user') {
            if (inputs.imageEnabled && inputs.imageUrls) {
              const validImageUrls = inputs.imageUrls.filter(url => url.trim() !== '');
              if (validImageUrls.length > 0) {
                let textContent = '示例消息';
                
                if (typeof messages[i].content === 'string') {
                  textContent = messages[i].content;
                } else if (Array.isArray(messages[i].content)) {
                  const textPart = messages[i].content.find(item => item.type === 'text');
                  if (textPart && textPart.text) {
                    textContent = textPart.text;
                  }
                }
                
                messages[i] = {
                  ...messages[i],
                  content: [
                    {
                      type: 'text',
                      text: textContent
                    },
                    ...validImageUrls.map(url => ({
                      type: 'image_url',
                      image_url: {
                        url: url.trim(),
                      },
                    })),
                  ]
                };
              }
            }
            break;
          }
        }
      }
      
      if (systemMessage) {
        messages.unshift(systemMessage);
      }
      
      const payload = {
        messages: messages,
        stream: inputs.stream,
        model: inputs.model,
        group: inputs.group,
      };

      if (parameterEnabled.max_tokens && inputs.max_tokens > 0) {
        payload.max_tokens = parseInt(inputs.max_tokens);
      }
      if (parameterEnabled.temperature) {
        payload.temperature = inputs.temperature;
      }
      if (parameterEnabled.top_p) {
        payload.top_p = inputs.top_p;
      }
      if (parameterEnabled.frequency_penalty) {
        payload.frequency_penalty = inputs.frequency_penalty;
      }
      if (parameterEnabled.presence_penalty) {
        payload.presence_penalty = inputs.presence_penalty;
      }
      if (parameterEnabled.seed && inputs.seed !== null && inputs.seed !== '') {
        payload.seed = parseInt(inputs.seed);
      }

      return payload;
    } catch (error) {
      console.error('构造预览请求体失败:', error);
      return null;
    }
  }, [inputs, parameterEnabled, systemPrompt, message]);

  useEffect(() => {
    const newPreviewPayload = constructPreviewPayload();
    setPreviewPayload(newPreviewPayload);
    
    setDebugData(prev => ({
      ...prev,
      previewRequest: newPreviewPayload,
      previewTimestamp: new Date().toISOString()
    }));
  }, [constructPreviewPayload]);

  const debouncedSaveConfig = useCallback(() => {
    if (saveConfigTimeoutRef.current) {
      clearTimeout(saveConfigTimeoutRef.current);
    }

    saveConfigTimeoutRef.current = setTimeout(() => {
      const configToSave = {
        inputs,
        parameterEnabled,
        systemPrompt,
        showDebugPanel,
      };
      saveConfig(configToSave);
    }, 1000);
  }, [inputs, parameterEnabled, systemPrompt, showDebugPanel]);

  useEffect(() => {
    debouncedSaveConfig();

    return () => {
      if (saveConfigTimeoutRef.current) {
        clearTimeout(saveConfigTimeoutRef.current);
      }
    };
  }, [debouncedSaveConfig]);

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const handleParameterToggle = (paramName) => {
    setParameterEnabled(prev => ({
      ...prev,
      [paramName]: !prev[paramName]
    }));
  };

  const handleConfigImport = useCallback((importedConfig) => {
    if (importedConfig.inputs) {
      setInputs(prev => ({
        ...prev,
        ...importedConfig.inputs,
      }));
    }

    if (importedConfig.parameterEnabled) {
      setParameterEnabled(prev => ({
        ...prev,
        ...importedConfig.parameterEnabled,
      }));
    }

    if (importedConfig.systemPrompt) {
      setSystemPrompt(importedConfig.systemPrompt);
    }

    if (typeof importedConfig.showDebugPanel === 'boolean') {
      setShowDebugPanel(importedConfig.showDebugPanel);
    }
  }, []);

  const handleConfigReset = useCallback(() => {
    const defaultConfig = loadConfig();
    setInputs(defaultConfig.inputs);
    setParameterEnabled(defaultConfig.parameterEnabled);
    setSystemPrompt(defaultConfig.systemPrompt);
    setShowDebugPanel(defaultConfig.showDebugPanel);
  }, []);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('未登录或登录已过期，请重新登录！'));
    }
    let status = localStorage.getItem('status');
    if (status) {
      status = JSON.parse(status);
      setStatus(status);
    }
    loadModels();
    loadGroups();
  }, [searchParams, t]);

  useEffect(() => {
    const handleResize = () => {
      styleDispatch({
        type: 'set_is_mobile',
        payload: isMobile(),
      });
    };

    handleResize();

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, [styleDispatch]);

  const loadModels = async () => {
    let res = await API.get(`/api/user/models`);
    const { success, message, data } = res.data;
    if (success) {
      let localModelOptions = data.map((model) => ({
        label: model,
        value: model,
      }));
      setModels(localModelOptions);

      const hasCurrentModel = localModelOptions.some(option => option.value === inputs.model);
      if (!hasCurrentModel && localModelOptions.length > 0) {
        handleInputChange('model', localModelOptions[0].value);
      }
    } else {
      showError(t(message));
    }
  };

  const loadGroups = async () => {
    let res = await API.get(`/api/user/self/groups`);
    const { success, message, data } = res.data;
    if (success) {
      let localGroupOptions = Object.entries(data).map(([group, info]) => ({
        label: info.desc.length > 20 ? info.desc.substring(0, 20) + '...' : info.desc,
        value: group,
        ratio: info.ratio,
        fullLabel: info.desc,
      }));

      if (localGroupOptions.length === 0) {
        localGroupOptions = [
          {
            label: t('用户分组'),
            value: '',
            ratio: 1,
          },
        ];
      } else {
        const localUser = JSON.parse(localStorage.getItem('user'));
        const userGroup =
          (userState.user && userState.user.group) ||
          (localUser && localUser.group);

        if (userGroup) {
          const userGroupIndex = localGroupOptions.findIndex(
            (g) => g.value === userGroup,
          );
          if (userGroupIndex > -1) {
            const userGroupOption = localGroupOptions.splice(
              userGroupIndex,
              1,
            )[0];
            localGroupOptions.unshift(userGroupOption);
          }
        }
      }

      setGroups(localGroupOptions);

      const hasCurrentGroup = localGroupOptions.some(option => option.value === inputs.group);
      if (!hasCurrentGroup) {
        handleInputChange('group', localGroupOptions[0].value);
      }
    } else {
      showError(t(message));
    }
  };

  const getSystemMessage = () => {
    if (systemPrompt !== '') {
      return {
        role: 'system',
        id: '1',
        createAt: 1715676751919,
        content: systemPrompt,
      };
    }
  };

  let handleNonStreamRequest = async (payload) => {
    setDebugData(prev => ({
      ...prev,
      request: payload,
      timestamp: new Date().toISOString(),
      response: null
    }));
    setActiveDebugTab('request');

    try {
      const response = await fetch('/pg/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'New-Api-User': getUserIdFromLocalStorage(),
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        let errorBody = '';
        try {
          errorBody = await response.text();
        } catch (e) {
          errorBody = '无法读取错误响应体';
        }

        const errorInfo = {
          error: 'HTTP错误',
          status: response.status,
          statusText: response.statusText,
          body: errorBody,
          timestamp: new Date().toISOString()
        };

        setDebugData(prev => ({
          ...prev,
          response: JSON.stringify(errorInfo, null, 2)
        }));
        setActiveDebugTab('response');

        throw new Error(`HTTP error! status: ${response.status}, body: ${errorBody}`);
      }

      const data = await response.json();

      setDebugData(prev => ({
        ...prev,
        response: JSON.stringify(data, null, 2)
      }));
      setActiveDebugTab('response');

      if (data.choices && data.choices[0]) {
        const choice = data.choices[0];
        let content = choice.message?.content || '';
        let reasoningContent = choice.message?.reasoning_content || '';

        if (content.includes('<think>')) {
          const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
          let thoughts = [];
          let replyParts = [];
          let lastIndex = 0;
          let match;

          thinkTagRegex.lastIndex = 0;
          while ((match = thinkTagRegex.exec(content)) !== null) {
            replyParts.push(content.substring(lastIndex, match.index));
            thoughts.push(match[1]);
            lastIndex = match.index + match[0].length;
          }
          replyParts.push(content.substring(lastIndex));

          content = replyParts.join('');
          if (thoughts.length > 0) {
            if (reasoningContent) {
              reasoningContent += '\n\n---\n\n' + thoughts.join('\n\n---\n\n');
            } else {
              reasoningContent = thoughts.join('\n\n---\n\n');
            }
          }
        }

        content = content.replace(/<\/?think>/g, '').trim();

        setMessage((prevMessage) => {
          const newMessages = [...prevMessage];
          const lastMessage = newMessages[newMessages.length - 1];
          if (lastMessage && lastMessage.status === 'loading') {
            newMessages[newMessages.length - 1] = {
              ...lastMessage,
              content: content,
              reasoningContent: reasoningContent,
              status: 'complete',
              isReasoningExpanded: false
            };
          }
          return newMessages;
        });
      }
    } catch (error) {
      console.error('Non-stream request error:', error);

      const errorInfo = {
        error: '非流式请求错误',
        message: error.message,
        timestamp: new Date().toISOString(),
        stack: error.stack
      };

      if (error.message.includes('HTTP error')) {
        errorInfo.details = '服务器返回了错误状态码';
      } else if (error.message.includes('Failed to fetch')) {
        errorInfo.details = '网络连接失败或服务器无响应';
      }

      setDebugData(prev => ({
        ...prev,
        response: JSON.stringify(errorInfo, null, 2)
      }));
      setActiveDebugTab('response');

      setMessage((prevMessage) => {
        const newMessages = [...prevMessage];
        const lastMessage = newMessages[newMessages.length - 1];
        if (lastMessage && lastMessage.status === 'loading') {
          newMessages[newMessages.length - 1] = {
            ...lastMessage,
            content: t('请求发生错误: ') + error.message,
            status: 'error',
            isReasoningExpanded: false
          };
        }
        return newMessages;
      });
    }
  };

  let handleSSE = (payload) => {
    setDebugData(prev => ({
      ...prev,
      request: payload,
      timestamp: new Date().toISOString(),
      response: null
    }));
    setActiveDebugTab('request');

    let source = new SSE('/pg/chat/completions', {
      headers: {
        'Content-Type': 'application/json',
        'New-Api-User': getUserIdFromLocalStorage(),
      },
      method: 'POST',
      payload: JSON.stringify(payload),
    });

    sseSourceRef.current = source;

    let responseData = '';
    let hasReceivedFirstResponse = false;

    source.addEventListener('message', (e) => {
      if (e.data === '[DONE]') {
        source.close();
        sseSourceRef.current = null;
        setDebugData(prev => ({
          ...prev,
          response: responseData
        }));
        completeMessage();
        return;
      }

      try {
        let payload = JSON.parse(e.data);
        responseData += e.data + '\n';

        if (!hasReceivedFirstResponse) {
          setActiveDebugTab('response');
          hasReceivedFirstResponse = true;
        }

        const delta = payload.choices?.[0]?.delta;
        if (delta) {
          if (delta.reasoning_content) {
            streamMessageUpdate(delta.reasoning_content, 'reasoning');
          }
          if (delta.content) {
            streamMessageUpdate(delta.content, 'content');
          }
        }
      } catch (error) {
        console.error('Failed to parse SSE message:', error);
        const errorInfo = `解析错误: ${error.message}`;

        setDebugData(prev => ({
          ...prev,
          response: responseData + `\n\nError: ${errorInfo}`
        }));
        setActiveDebugTab('response');

        streamMessageUpdate(t('解析响应数据时发生错误'), 'content');
        completeMessage('error');
      }
    });

    source.addEventListener('error', (e) => {
      console.error('SSE Error:', e);
      const errorMessage = e.data || t('请求发生错误');

      const errorInfo = {
        error: 'SSE连接错误',
        message: errorMessage,
        status: source.status,
        readyState: source.readyState,
        timestamp: new Date().toISOString()
      };

      setDebugData(prev => ({
        ...prev,
        response: responseData + '\n\nSSE Error:\n' + JSON.stringify(errorInfo, null, 2)
      }));
      setActiveDebugTab('response');

      streamMessageUpdate(errorMessage, 'content');
      completeMessage('error');
      sseSourceRef.current = null;
      source.close();
    });

    source.addEventListener('readystatechange', (e) => {
      if (e.readyState >= 2) {
        if (source.status !== undefined && source.status !== 200) {
          const errorInfo = {
            error: 'HTTP状态错误',
            status: source.status,
            readyState: source.readyState,
            timestamp: new Date().toISOString()
          };

          setDebugData(prev => ({
            ...prev,
            response: responseData + '\n\nHTTP Error:\n' + JSON.stringify(errorInfo, null, 2)
          }));
          setActiveDebugTab('response');

          source.close();
          streamMessageUpdate(t('连接已断开'), 'content');
          completeMessage('error');
        }
      }
    });

    try {
      source.stream();
    } catch (error) {
      console.error('Failed to start SSE stream:', error);
      const errorInfo = {
        error: '启动SSE流失败',
        message: error.message,
        timestamp: new Date().toISOString()
      };

      setDebugData(prev => ({
        ...prev,
        response: 'Stream启动失败:\n' + JSON.stringify(errorInfo, null, 2)
      }));
      setActiveDebugTab('response');

      streamMessageUpdate(t('建立连接时发生错误'), 'content');
      completeMessage('error');
    }
  };

  const onMessageSend = useCallback(
    (content, attachment) => {
      console.log('attachment: ', attachment);
      setMessage((prevMessage) => {
        let messageContent;
        const validImageUrls = inputs.imageUrls.filter(url => url.trim() !== '');

        if (inputs.imageEnabled && validImageUrls.length > 0) {
          messageContent = [
            {
              type: 'text',
              text: content,
            },
            ...validImageUrls.map(url => ({
              type: 'image_url',
              image_url: {
                url: url.trim(),
              },
            })),
          ];
        } else {
          messageContent = content;
        }

        const newMessage = [
          ...prevMessage,
          {
            role: 'user',
            content: messageContent,
            createAt: Date.now(),
            id: getId(),
          },
        ];

        const getPayload = () => {
          let systemMessage = getSystemMessage();
          let messages = newMessage.map((item) => {
            return {
              role: item.role,
              content: item.content,
            };
          });
          if (systemMessage) {
            messages.unshift(systemMessage);
          }
          const payload = {
            messages: messages,
            stream: inputs.stream,
            model: inputs.model,
            group: inputs.group,
          };

          if (parameterEnabled.max_tokens && inputs.max_tokens > 0) {
            payload.max_tokens = parseInt(inputs.max_tokens);
          }
          if (parameterEnabled.temperature) {
            payload.temperature = inputs.temperature;
          }
          if (parameterEnabled.top_p) {
            payload.top_p = inputs.top_p;
          }
          if (parameterEnabled.frequency_penalty) {
            payload.frequency_penalty = inputs.frequency_penalty;
          }
          if (parameterEnabled.presence_penalty) {
            payload.presence_penalty = inputs.presence_penalty;
          }
          if (parameterEnabled.seed && inputs.seed !== null && inputs.seed !== '') {
            payload.seed = parseInt(inputs.seed);
          }

          return payload;
        };

        const payload = getPayload();

        if (inputs.stream) {
          handleSSE(payload);
        } else {
          handleNonStreamRequest(payload);
        }

        if (inputs.imageEnabled) {
          setTimeout(() => {
            handleInputChange('imageEnabled', false);
          }, 100);
        }

        newMessage.push({
          role: 'assistant',
          content: '',
          reasoningContent: '',
          isReasoningExpanded: true,
          createAt: Date.now(),
          id: getId(),
          status: 'loading',
        });
        return newMessage;
      });
    },
    [getSystemMessage, inputs, setMessage, parameterEnabled, handleInputChange],
  );

  const completeMessage = useCallback((status = 'complete') => {
    setMessage((prevMessage) => {
      const lastMessage = prevMessage[prevMessage.length - 1];
      if (lastMessage.status === 'complete' || lastMessage.status === 'error') {
        return prevMessage;
      }
      return [...prevMessage.slice(0, -1), { ...lastMessage, status: status, isReasoningExpanded: false }];
    });
  }, [setMessage]);

  const streamMessageUpdate = useCallback((textChunk, type) => {
    setMessage((prevMessage) => {
      const lastMessage = prevMessage[prevMessage.length - 1];
      let newMessage = { ...lastMessage };

      if (lastMessage.status === 'error') {
        return prevMessage;
      }

      if (lastMessage.status === 'loading' || lastMessage.status === 'incomplete') {
        if (type === 'reasoning') {
          newMessage = {
            ...newMessage,
            reasoningContent: (lastMessage.reasoningContent || '') + textChunk,
            status: 'incomplete',
          };
        } else if (type === 'content') {
          const shouldCollapseReasoning = !lastMessage.content && lastMessage.reasoningContent;
          const newContent = (lastMessage.content || '') + textChunk;

          let shouldCollapseFromThinkTag = false;
          if (lastMessage.isReasoningExpanded && newContent.includes('</think>')) {
            const thinkMatches = newContent.match(/<think>/g);
            const thinkCloseMatches = newContent.match(/<\/think>/g);
            if (thinkMatches && thinkCloseMatches && thinkCloseMatches.length >= thinkMatches.length) {
              shouldCollapseFromThinkTag = true;
            }
          }

          newMessage = {
            ...newMessage,
            content: newContent,
            status: 'incomplete',
            isReasoningExpanded: (shouldCollapseReasoning || shouldCollapseFromThinkTag) ? false : lastMessage.isReasoningExpanded,
          };
        }
      }
      return [...prevMessage.slice(0, -1), newMessage];
    });
  }, [setMessage]);

  const handleMessageCopy = useCallback((message) => {
    if (!message.content) return;

    let textToCopy;

    if (Array.isArray(message.content)) {
      const textContent = message.content.find(item => item.type === 'text');
      if (textContent && textContent.text && typeof textContent.text === 'string') {
        textToCopy = textContent.text;
      } else {
        Toast.warning({
          content: t('此消息没有可复制的文本内容'),
          duration: 2,
        });
        return;
      }
    } else if (typeof message.content === 'string') {
      textToCopy = message.content;
    } else {
      Toast.warning({
        content: t('无法复制此类型的消息内容'),
        duration: 2,
      });
      return;
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(textToCopy).then(() => {
        Toast.success({
          content: t('消息已复制到剪贴板'),
          duration: 2,
        });
      }).catch(err => {
        console.error('Clipboard API 复制失败:', err);
        fallbackCopyToClipboard(textToCopy);
      });
    } else {
      fallbackCopyToClipboard(textToCopy);
    }
  }, [t]);

  const fallbackCopyToClipboard = useCallback((text) => {
    try {
      if (!document.execCommand) {
        throw new Error('execCommand not supported');
      }

      const textArea = document.createElement('textarea');
      textArea.value = text;

      textArea.style.position = 'fixed';
      textArea.style.top = '-9999px';
      textArea.style.left = '-9999px';
      textArea.style.opacity = '0';
      textArea.style.pointerEvents = 'none';
      textArea.style.zIndex = '-1';
      textArea.setAttribute('readonly', '');

      document.body.appendChild(textArea);

      if (textArea.select) {
        textArea.select();
      }
      if (textArea.setSelectionRange) {
        textArea.setSelectionRange(0, text.length);
      }

      const successful = document.execCommand('copy');
      document.body.removeChild(textArea);

      if (successful) {
        Toast.success({
          content: t('消息已复制到剪贴板'),
          duration: 2,
        });
      } else {
        throw new Error('execCommand copy failed');
      }
    } catch (err) {
      console.error('回退复制方案也失败:', err);

      let errorMessage = t('复制失败，请手动选择文本复制');

      if (window.location.protocol === 'http:' && window.location.hostname !== 'localhost') {
        errorMessage = t('复制功能需要 HTTPS 环境，请手动复制');
      } else if (!navigator.clipboard && !document.execCommand) {
        errorMessage = t('浏览器不支持复制功能，请手动复制');
      }

      Toast.error({
        content: errorMessage,
        duration: 4,
      });
    }
  }, [t]);

  const handleMessageReset = useCallback((targetMessage) => {
    setMessage(prevMessages => {
      const messageIndex = prevMessages.findIndex(msg => msg.id === targetMessage.id);
      if (messageIndex === -1) return prevMessages;

      if (targetMessage.role === 'user') {
        const newMessages = prevMessages.slice(0, messageIndex);
        setTimeout(() => {
          let contentToSend;
          if (Array.isArray(targetMessage.content)) {
            const textContent = targetMessage.content.find(item => item.type === 'text');
            contentToSend = textContent && textContent.text ? textContent.text : '';
          } else {
            contentToSend = targetMessage.content;
          }
          onMessageSend(contentToSend);
        }, 100);
        return newMessages;
      } else if (targetMessage.role === 'assistant') {
        let userMessageIndex = messageIndex - 1;
        while (userMessageIndex >= 0 && prevMessages[userMessageIndex].role !== 'user') {
          userMessageIndex--;
        }
        if (userMessageIndex >= 0) {
          const userMessage = prevMessages[userMessageIndex];
          const newMessages = prevMessages.slice(0, userMessageIndex);
          setTimeout(() => {
            let contentToSend;
            if (Array.isArray(userMessage.content)) {
              const textContent = userMessage.content.find(item => item.type === 'text');
              contentToSend = textContent && textContent.text ? textContent.text : '';
            } else {
              contentToSend = userMessage.content;
            }
            onMessageSend(contentToSend);
          }, 100);
          return newMessages;
        }
      }
      return prevMessages;
    });
  }, [onMessageSend]);

  const handleMessageDelete = useCallback((targetMessage) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('确定要删除这条消息吗？'),
      okText: t('确定'),
      cancelText: t('取消'),
      okButtonProps: {
        type: 'danger',
      },
      onOk: () => {
        setMessage(prevMessages => {
          const messageIndex = prevMessages.findIndex(msg => msg.id === targetMessage.id);
          if (messageIndex === -1) return prevMessages;

          if (targetMessage.role === 'user' && messageIndex < prevMessages.length - 1) {
            const nextMessage = prevMessages[messageIndex + 1];
            if (nextMessage.role === 'assistant') {
              Toast.success({
                content: t('已删除消息及其回复'),
                duration: 2,
              });
              return prevMessages.filter((_, index) => index !== messageIndex && index !== messageIndex + 1);
            }
          }

          Toast.success({
            content: t('消息已删除'),
            duration: 2,
          });
          return prevMessages.filter(msg => msg.id !== targetMessage.id);
        });
      },
    });
  }, [setMessage, t]);

  const onStopGenerator = useCallback(() => {
    if (sseSourceRef.current) {
      sseSourceRef.current.close();
      sseSourceRef.current = null;
      setMessage((prevMessage) => {
        const lastMessage = prevMessage[prevMessage.length - 1];
        if (lastMessage.status === 'loading' || lastMessage.status === 'incomplete') {
          let currentContent = lastMessage.content || '';
          let currentReasoningContent = lastMessage.reasoningContent || '';

          if (currentContent.includes('<think>')) {
            const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
            let match;
            let thoughtsFromPairedTags = [];
            let replyParts = [];
            let lastIndex = 0;

            while ((match = thinkTagRegex.exec(currentContent)) !== null) {
              replyParts.push(currentContent.substring(lastIndex, match.index));
              thoughtsFromPairedTags.push(match[1]);
              lastIndex = match.index + match[0].length;
            }
            replyParts.push(currentContent.substring(lastIndex));

            if (thoughtsFromPairedTags.length > 0) {
              const pairedThoughtsStr = thoughtsFromPairedTags.join('\n\n---\n\n');
              if (currentReasoningContent) {
                currentReasoningContent += '\n\n---\n\n' + pairedThoughtsStr;
              } else {
                currentReasoningContent = pairedThoughtsStr;
              }
            }
            currentContent = replyParts.join('');
          }

          const lastOpenThinkIndex = currentContent.lastIndexOf('<think>');
          if (lastOpenThinkIndex !== -1) {
            const fragmentAfterLastOpen = currentContent.substring(lastOpenThinkIndex);
            if (!fragmentAfterLastOpen.includes('</think>')) {
              const unclosedThought = fragmentAfterLastOpen.substring('<think>'.length).trim();
              if (unclosedThought) {
                if (currentReasoningContent) {
                  currentReasoningContent += '\n\n---\n\n' + unclosedThought;
                } else {
                  currentReasoningContent = unclosedThought;
                }
              }
              currentContent = currentContent.substring(0, lastOpenThinkIndex);
            }
          }

          currentContent = currentContent.replace(/<\/?think>/g, '').trim();

          return [...prevMessage.slice(0, -1), {
            ...lastMessage,
            status: 'complete',
            reasoningContent: currentReasoningContent || null,
            content: currentContent,
            isReasoningExpanded: false
          }];
        }
        return prevMessage;
      });
    }
  }, [setMessage]);

  const toggleReasoningExpansion = (messageId) => {
    setMessage(prevMessages =>
      prevMessages.map(msg =>
        msg.id === messageId && msg.role === 'assistant'
          ? { ...msg, isReasoningExpanded: !msg.isReasoningExpanded }
          : msg
      )
    );
  };

  const renderCustomChatContent = useCallback(
    ({ message, className }) => {
      return (
        <MessageContent
          message={message}
          className={className}
          styleState={styleState}
          onToggleReasoningExpansion={toggleReasoningExpansion}
        />
      );
    },
    [styleState],
  );

  const renderChatBoxAction = useCallback((props) => {
    const { message: currentMessage } = props;

    const isAnyMessageGenerating = message.some(msg => msg.status === 'loading' || msg.status === 'incomplete');

    return (
      <MessageActions
        message={currentMessage}
        styleState={styleState}
        onMessageReset={handleMessageReset}
        onMessageCopy={handleMessageCopy}
        onMessageDelete={handleMessageDelete}
        isAnyMessageGenerating={isAnyMessageGenerating}
      />
    );
  }, [handleMessageReset, handleMessageCopy, handleMessageDelete, styleState, message]);

  return (
    <div className="h-full bg-gray-50">
      <Layout style={{ height: '100%', background: 'transparent' }} className="flex flex-col md:flex-row">
        {(showSettings || !styleState.isMobile) && (
          <Layout.Sider
            style={{
              background: 'transparent',
              borderRight: 'none',
              flexShrink: 0,
              minWidth: styleState.isMobile ? '100%' : 320,
              maxWidth: styleState.isMobile ? '100%' : 320,
              height: styleState.isMobile ? 'auto' : 'calc(100vh - 100px)',
              overflow: 'auto',
              position: styleState.isMobile ? 'fixed' : 'relative',
              zIndex: styleState.isMobile ? 1000 : 1,
              width: '100%',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
            }}
            width={styleState.isMobile ? '100%' : 320}
            className={styleState.isMobile ? 'bg-white shadow-lg' : ''}
          >
            <SettingsPanel
              inputs={inputs}
              parameterEnabled={parameterEnabled}
              models={models}
              groups={groups}
              systemPrompt={systemPrompt}
              styleState={styleState}
              showSettings={showSettings}
              showDebugPanel={showDebugPanel}
              onInputChange={handleInputChange}
              onParameterToggle={handleParameterToggle}
              onSystemPromptChange={setSystemPrompt}
              onCloseSettings={() => setShowSettings(false)}
              onConfigImport={handleConfigImport}
              onConfigReset={handleConfigReset}
            />
          </Layout.Sider>
        )}

        <Layout.Content className="relative flex-1 overflow-hidden">
          <div className="sm:px-4 overflow-hidden flex flex-col lg:flex-row gap-2 sm:gap-4 h-[calc(100vh-100px)]">
            <div className="flex-1 flex flex-col">
              <ChatArea
                chatRef={chatRef}
                message={message}
                inputs={inputs}
                styleState={styleState}
                showDebugPanel={showDebugPanel}
                roleInfo={roleInfo}
                onMessageSend={onMessageSend}
                onMessageCopy={handleMessageCopy}
                onMessageReset={handleMessageReset}
                onMessageDelete={handleMessageDelete}
                onStopGenerator={onStopGenerator}
                onClearMessages={() => setMessage([])}
                onToggleDebugPanel={() => setShowDebugPanel(!showDebugPanel)}
                renderCustomChatContent={renderCustomChatContent}
                renderChatBoxAction={renderChatBoxAction}
              />
            </div>

            {/* 调试面板 - 桌面端 */}
            {showDebugPanel && !styleState.isMobile && (
              <div className="w-96 flex-shrink-0 h-full">
                <DebugPanel
                  debugData={debugData}
                  activeDebugTab={activeDebugTab}
                  onActiveDebugTabChange={setActiveDebugTab}
                  styleState={styleState}
                />
              </div>
            )}
          </div>

          {/* 调试面板 - 移动端覆盖层 */}
          {showDebugPanel && styleState.isMobile && (
            <div
              style={{
                position: 'fixed',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                zIndex: 1000,
                backgroundColor: 'white',
                overflow: 'auto',
              }}
              className="shadow-lg"
            >
              <DebugPanel
                debugData={debugData}
                activeDebugTab={activeDebugTab}
                onActiveDebugTabChange={setActiveDebugTab}
                styleState={styleState}
                showDebugPanel={showDebugPanel}
                onCloseDebugPanel={() => setShowDebugPanel(false)}
              />
            </div>
          )}

          {/* 浮动按钮 */}
          <FloatingButtons
            styleState={styleState}
            showSettings={showSettings}
            showDebugPanel={showDebugPanel}
            onToggleSettings={() => setShowSettings(!showSettings)}
            onToggleDebugPanel={() => setShowDebugPanel(!showDebugPanel)}
          />
        </Layout.Content>
      </Layout>
    </div>
  );
};

export default Playground;
