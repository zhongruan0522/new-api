import React, { useCallback, useContext, useEffect, useState, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User/index.js';
import {
  API,
  getUserIdFromLocalStorage,
  showError,
  getLogo,
} from '../../helpers/index.js';
import {
  Card,
  Chat,
  Input,
  Layout,
  Select,
  Slider,
  TextArea,
  Typography,
  Button,
  MarkdownRender,
  Tag,
  Tabs,
  TabPane,
  Toast,
  Tooltip,
  Modal,
} from '@douyinfe/semi-ui';
import { SSE } from 'sse';
import {
  Settings,
  Sparkles,
  ChevronRight,
  ChevronUp,
  Brain,
  Zap,
  MessageSquare,
  SlidersHorizontal,
  Hash,
  Thermometer,
  Type,
  Users,
  Loader2,
  Target,
  Repeat,
  Ban,
  Shuffle,
  ToggleLeft,
  Code,
  Eye,
  EyeOff,
  FileText,
  Clock,
  Check,
  X,
  Copy,
  RefreshCw,
  Trash2,
} from 'lucide-react';
import { StyleContext } from '../../context/Style/index.js';
import { useTranslation } from 'react-i18next';
import { renderGroupOption, truncateText, stringToColor } from '../../helpers/render.js';
import { IconSend } from '@douyinfe/semi-icons';

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

  const defaultModel = 'deepseek-r1';
  const [inputs, setInputs] = useState({
    model: defaultModel,
    group: '',
    max_tokens: 0,
    temperature: 0,
    top_p: 1,
    frequency_penalty: 0,
    presence_penalty: 0,
    seed: null,
    stream: true,
  });
  const [parameterEnabled, setParameterEnabled] = useState({
    max_tokens: true,
    temperature: true,
    top_p: false,
    frequency_penalty: false,
    presence_penalty: false,
    seed: false,
  });
  const [searchParams, setSearchParams] = useSearchParams();
  const [status, setStatus] = useState({});
  const [systemPrompt, setSystemPrompt] = useState(
    'You are a helpful assistant. You can help me by answering my questions. You can also ask me questions.',
  );
  const [message, setMessage] = useState(defaultMessage);
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [showSettings, setShowSettings] = useState(true);
  const [showDebugPanel, setShowDebugPanel] = useState(true);
  const [debugData, setDebugData] = useState({
    request: null,
    response: null,
    timestamp: null
  });
  const [activeDebugTab, setActiveDebugTab] = useState('request');
  const [styleState, styleDispatch] = useContext(StyleContext);
  const sseSourceRef = useRef(null);

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const handleParameterToggle = (paramName) => {
    setParameterEnabled(prev => ({
      ...prev,
      [paramName]: !prev[paramName]
    }));
  };

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

  const loadModels = async () => {
    let res = await API.get(`/api/user/models`);
    const { success, message, data } = res.data;
    if (success) {
      let localModelOptions = data.map((model) => ({
        label: model,
        value: model,
      }));
      setModels(localModelOptions);
      // if default model is not in the list, set the first one as default
      const hasDefault = localModelOptions.some(option => option.value === defaultModel);
      if (!hasDefault && localModelOptions.length > 0) {
        setInputs((inputs) => ({ ...inputs, model: localModelOptions[0].value }));
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
        label: truncateText(info.desc, '50%'),
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
      handleInputChange('group', localGroupOptions[0].value);
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
    // 记录请求数据并自动切换到请求体标签
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
        // 尝试读取错误响应体
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

        // 记录HTTP错误到调试数据
        setDebugData(prev => ({
          ...prev,
          response: JSON.stringify(errorInfo, null, 2)
        }));
        setActiveDebugTab('response');

        throw new Error(`HTTP error! status: ${response.status}, body: ${errorBody}`);
      }

      const data = await response.json();

      // 记录响应数据并自动切换到响应标签
      setDebugData(prev => ({
        ...prev,
        response: JSON.stringify(data, null, 2)
      }));
      setActiveDebugTab('response');

      // 处理响应数据
      if (data.choices && data.choices[0]) {
        const choice = data.choices[0];
        let content = choice.message?.content || '';
        let reasoningContent = choice.message?.reasoning_content || '';

        // 处理 <think> 标签格式的思维链
        if (content.includes('<think>')) {
          const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
          let thoughts = [];
          let replyParts = [];
          let lastIndex = 0;
          let match;

          while ((match = thinkTagRegex.exec(content)) !== null) {
            replyParts.push(content.substring(lastIndex, match.index));
            thoughts.push(match[1]);
            lastIndex = match.index + match[0].length;
          }
          replyParts.push(content.substring(lastIndex));

          content = replyParts.join('').trim();
          if (thoughts.length > 0) {
            reasoningContent = thoughts.join('\n\n---\n\n');
          }
        }

        // 更新消息
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

      // 构建详细的错误信息
      const errorInfo = {
        error: '非流式请求错误',
        message: error.message,
        timestamp: new Date().toISOString(),
        stack: error.stack
      };

      // 如果是 fetch 错误，尝试获取更多信息
      if (error.message.includes('HTTP error')) {
        errorInfo.details = '服务器返回了错误状态码';
      } else if (error.message.includes('Failed to fetch')) {
        errorInfo.details = '网络连接失败或服务器无响应';
      }

      // 记录详细的错误响应并切换到响应标签
      setDebugData(prev => ({
        ...prev,
        response: JSON.stringify(errorInfo, null, 2)
      }));
      setActiveDebugTab('response');

      // 更新消息为错误状态
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
    // 记录请求数据并自动切换到请求体标签
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

    // 保存 source 引用以便后续停止生成
    sseSourceRef.current = source;

    let responseData = '';
    let hasReceivedFirstResponse = false;

    source.addEventListener('message', (e) => {
      if (e.data === '[DONE]') {
        source.close();
        sseSourceRef.current = null;
        // 记录完整响应
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

        // 收到第一个响应时自动切换到响应标签
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

        // 记录错误到调试数据
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

      // 记录错误信息到调试数据
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

          // 记录状态错误到调试数据
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

      // 记录启动错误到调试数据
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
        const newMessage = [
          ...prevMessage,
          {
            role: 'user',
            content: content,
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

          // 只添加启用的参数
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
    [getSystemMessage, inputs, setMessage],
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

      // 如果消息已经是错误状态，保持错误状态
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
          // 当开始接收 content 时，说明思考部分已经完成，应该折叠思考面板
          const shouldCollapseReasoning = !lastMessage.content && lastMessage.reasoningContent && lastMessage.isReasoningExpanded;

          const newContent = (lastMessage.content || '') + textChunk;

          // 检测 </think> 标签的完成
          let shouldCollapseFromThinkTag = false;
          if (lastMessage.isReasoningExpanded && newContent.includes('</think>')) {
            // 检查是否有完整的 <think>...</think> 对
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

  // 处理消息复制
  const handleMessageCopy = useCallback((message) => {
    if (!message.content) return;

    // 现代浏览器的 Clipboard API
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(message.content).then(() => {
        Toast.success({
          content: t('消息已复制到剪贴板'),
          duration: 2,
        });
      }).catch(err => {
        console.error('Clipboard API 复制失败:', err);
        // 如果 Clipboard API 失败，尝试回退方案
        fallbackCopyToClipboard(message.content);
      });
    } else {
      // 回退方案：使用传统的 document.execCommand
      fallbackCopyToClipboard(message.content);
    }
  }, [t]);

  // 回退复制方案
  const fallbackCopyToClipboard = useCallback((text) => {
    try {
      // 检查是否支持 execCommand
      if (!document.execCommand) {
        throw new Error('execCommand not supported');
      }

      // 创建一个临时的 textarea 元素
      const textArea = document.createElement('textarea');
      textArea.value = text;

      // 设置样式使其不可见但可选中
      textArea.style.position = 'fixed';
      textArea.style.top = '-9999px';
      textArea.style.left = '-9999px';
      textArea.style.opacity = '0';
      textArea.style.pointerEvents = 'none';
      textArea.style.zIndex = '-1';
      textArea.setAttribute('readonly', '');

      document.body.appendChild(textArea);

      // 选中文本
      if (textArea.select) {
        textArea.select();
      }
      if (textArea.setSelectionRange) {
        textArea.setSelectionRange(0, text.length);
      }

      // 使用 execCommand 复制
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

      // 提供更详细的错误信息
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

  // 处理消息重试
  const handleMessageReset = useCallback((targetMessage) => {
    setMessage(prevMessages => {
      // 找到要重试的消息的索引
      const messageIndex = prevMessages.findIndex(msg => msg.id === targetMessage.id);
      if (messageIndex === -1) return prevMessages;

      // 如果是用户消息，重新发送
      if (targetMessage.role === 'user') {
        // 删除该消息及其后面的所有消息
        const newMessages = prevMessages.slice(0, messageIndex);
        // 重新发送消息
        setTimeout(() => {
          onMessageSend(targetMessage.content);
        }, 100);
        return newMessages;
      } else if (targetMessage.role === 'assistant') {
        // 如果是助手消息，找到它前面最近的用户消息并重试
        let userMessageIndex = messageIndex - 1;
        while (userMessageIndex >= 0 && prevMessages[userMessageIndex].role !== 'user') {
          userMessageIndex--;
        }
        if (userMessageIndex >= 0) {
          const userMessage = prevMessages[userMessageIndex];
          // 删除用户消息之后的所有消息
          const newMessages = prevMessages.slice(0, userMessageIndex);
          // 重新发送用户消息
          setTimeout(() => {
            onMessageSend(userMessage.content);
          }, 100);
          return newMessages;
        }
      }
      return prevMessages;
    });
  }, [onMessageSend]);

  // 处理消息删除
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
          // 找到要删除的消息索引
          const messageIndex = prevMessages.findIndex(msg => msg.id === targetMessage.id);
          if (messageIndex === -1) return prevMessages;

          // 如果是用户消息，同时删除后面紧跟的助手回复
          if (targetMessage.role === 'user' && messageIndex < prevMessages.length - 1) {
            const nextMessage = prevMessages[messageIndex + 1];
            if (nextMessage.role === 'assistant') {
              // 删除用户消息和助手回复
              Toast.success({
                content: t('已删除消息及其回复'),
                duration: 2,
              });
              return prevMessages.filter((_, index) => index !== messageIndex && index !== messageIndex + 1);
            }
          }

          // 否则只删除当前消息
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
          let content = lastMessage.content || '';
          let reasoningContent = lastMessage.reasoningContent || '';

          // 处理 <think> 标签格式的思维链
          if (content.includes('<think>')) {
            const thinkTagRegex = /<think>([\s\S]*?)(?:<\/think>|$)/g;
            let thoughts = [];
            let replyParts = [];
            let lastIndex = 0;
            let match;

            while ((match = thinkTagRegex.exec(content)) !== null) {
              replyParts.push(content.substring(lastIndex, match.index));
              thoughts.push(match[1]);
              lastIndex = match.index + match[0].length;
            }
            replyParts.push(content.substring(lastIndex));

            // 更新内容和思维链
            content = replyParts.join('').trim();
            if (thoughts.length > 0) {
              reasoningContent = thoughts.join('\n\n---\n\n');
            }
          }

          return [...prevMessage.slice(0, -1), {
            ...lastMessage,
            status: 'complete',
            reasoningContent: reasoningContent,
            content: content,
            isReasoningExpanded: false  // 停止时折叠思维链面板
          }];
        }
        return prevMessage;
      });
    }
  }, [setMessage]);

  const DebugToggle = () => {
    return (
      <Button
        icon={showDebugPanel ? <EyeOff size={14} /> : <Eye size={14} />}
        onClick={() => setShowDebugPanel(!showDebugPanel)}
        theme="borderless"
        type="tertiary"
        size="small"
        className="!rounded-lg !text-gray-600 hover:!text-purple-600 hover:!bg-purple-50"
      >
        {showDebugPanel ? t('隐藏调试') : t('显示调试')}
      </Button>
    );
  };

  const SettingsToggle = () => {
    if (!styleState.isMobile) return null;
    return (
      <Button
        icon={<Settings size={16} />}
        style={{
          position: 'absolute',
          left: showSettings ? -10 : -20,
          top: '50%',
          transform: 'translateY(-50%)',
          zIndex: 1000,
          width: 40,
          height: 40,
          borderRadius: '0 20px 20px 0',
          padding: 0,
          boxShadow: '2px 0 8px rgba(0, 0, 0, 0.15)',
        }}
        onClick={() => setShowSettings(!showSettings)}
        theme='solid'
        type='primary'
      />
    );
  };

  function CustomInputRender(props) {
    const { detailProps } = props;
    const { clearContextNode, uploadNode, inputNode, sendNode, onClick } =
      detailProps;

    return (
      <div className="p-4">
        <div
          className="flex items-end gap-3 p-4 bg-gray-50 rounded-2xl shadow-sm hover:shadow-md transition-shadow"
          style={{ border: '1px solid var(--semi-color-border)' }}
          onClick={onClick}
        >
          <div className="flex-1">
            {inputNode}
          </div>
          <Button
            theme="solid"
            type="primary"
            className="!rounded-lg !bg-purple-500 hover:!bg-purple-600 flex-shrink-0"
            icon={<IconSend />}
          >
            {t('发送')}
          </Button>
        </div>
      </div>
    );
  }

  const renderInputArea = useCallback((props) => {
    return <CustomInputRender {...props} />;
  }, []);

  // 自定义操作按钮渲染
  const renderChatBoxAction = useCallback((props) => {
    const { message } = props;

    // 对于正在加载或未完成的消息，只显示部分按钮
    const isLoading = message.status === 'loading' || message.status === 'incomplete';

    return (
      <div className="flex items-center gap-0.5">
        {/* 重试按钮 - 只在消息完成或出错时显示 */}
        {!isLoading && (
          <Tooltip content={t('重试')} position="top">
            <Button
              theme="borderless"
              type="tertiary"
              size="small"
              icon={<RefreshCw size={14} />}
              onClick={() => handleMessageReset(message)}
              className="!rounded-md !text-gray-400 hover:!text-blue-600 hover:!bg-blue-50 !w-7 !h-7 !p-0 transition-all"
              aria-label={t('重试')}
            />
          </Tooltip>
        )}

        {/* 复制按钮 - 只在有内容时显示 */}
        {message.content && (
          <Tooltip content={t('复制')} position="top">
            <Button
              theme="borderless"
              type="tertiary"
              size="small"
              icon={<Copy size={14} />}
              onClick={() => handleMessageCopy(message)}
              className="!rounded-md !text-gray-400 hover:!text-green-600 hover:!bg-green-50 !w-7 !h-7 !p-0 transition-all"
              aria-label={t('复制')}
            />
          </Tooltip>
        )}

        {/* 删除按钮 - 只在消息完成或出错时显示，AI输出时隐藏 */}
        {!isLoading && (
          <Tooltip content={t('删除')} position="top">
            <Button
              theme="borderless"
              type="tertiary"
              size="small"
              icon={<Trash2 size={14} />}
              onClick={() => handleMessageDelete(message)}
              className="!rounded-md !text-gray-400 hover:!text-red-600 hover:!bg-red-50 !w-7 !h-7 !p-0 transition-all"
              aria-label={t('删除')}
            />
          </Tooltip>
        )}
      </div>
    );
  }, [handleMessageReset, handleMessageCopy, handleMessageDelete, t]);

  const renderCustomChatContent = useCallback(
    ({ message, className }) => {
      if (message.status === 'error') {
        return (
          <div className={`${className} flex items-center p-4 bg-red-50 rounded-xl`}>
            <Typography.Text type="danger" className="text-sm">
              {message.content || t('请求发生错误')}
            </Typography.Text>
          </div>
        );
      }

      const toggleReasoningExpansion = (messageId) => {
        setMessage(prevMessages =>
          prevMessages.map(msg =>
            msg.id === messageId && msg.role === 'assistant'
              ? { ...msg, isReasoningExpanded: !msg.isReasoningExpanded }
              : msg
          )
        );
      };

      const isThinkingStatus = message.status === 'loading' || message.status === 'incomplete';
      let currentExtractedThinkingContent = null;
      let currentDisplayableFinalContent = message.content || "";
      let thinkingSource = null;

      if (message.role === 'assistant') {
        if (message.reasoningContent) {
          currentExtractedThinkingContent = message.reasoningContent;
          thinkingSource = 'reasoningContent';
        } else if (message.content && message.content.includes('<think')) {
          const fullContent = message.content;
          let thoughts = [];
          let replyParts = [];
          let lastIndex = 0;

          // 使用更安全的正则表达式，只匹配完整的 think 标签对
          const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
          let match;

          thinkTagRegex.lastIndex = 0;
          while ((match = thinkTagRegex.exec(fullContent)) !== null) {
            replyParts.push(fullContent.substring(lastIndex, match.index));
            thoughts.push(match[1]);
            lastIndex = match.index + match[0].length;
          }
          replyParts.push(fullContent.substring(lastIndex));

          // 处理剩余的内容，移除未闭合的 think 标签
          let finalContent = replyParts.join('');

          // 如果还有未闭合的 <think> 标签，将其内容提取到思考区域
          if (isThinkingStatus) {
            const lastOpenThinkIndex = finalContent.lastIndexOf('<think>');
            if (lastOpenThinkIndex !== -1) {
              const fragmentAfterLastOpen = finalContent.substring(lastOpenThinkIndex);
              // 检查是否有对应的闭合标签
              if (!fragmentAfterLastOpen.includes('</think>')) {
                // 提取未闭合的思考内容
                const unclosedThought = fragmentAfterLastOpen.substring('<think>'.length);
                if (unclosedThought.trim()) {
                  if (currentExtractedThinkingContent) {
                    currentExtractedThinkingContent += '\n\n---\n\n' + unclosedThought;
                  } else {
                    currentExtractedThinkingContent = unclosedThought;
                  }
                  if (!thinkingSource) thinkingSource = '<think> tags (streaming)';
                }
                // 移除未闭合的 think 标签部分
                finalContent = finalContent.substring(0, lastOpenThinkIndex);
              }
            }
          }

          currentDisplayableFinalContent = finalContent.trim();

          if (thoughts.length > 0) {
            if (currentExtractedThinkingContent) {
              currentExtractedThinkingContent = thoughts.join('\n\n---\n\n') + '\n\n---\n\n' + currentExtractedThinkingContent;
            } else {
              currentExtractedThinkingContent = thoughts.join('\n\n---\n\n');
            }
            thinkingSource = '<think> tags';
          }
        }

        // 清理任何剩余的不完整 think 标签
        if (typeof currentDisplayableFinalContent === 'string') {
          // 移除任何孤立的 <think> 开始标签
          currentDisplayableFinalContent = currentDisplayableFinalContent.replace(/<think>\s*$/g, '');
          // 如果内容以 <think> 开始但没有完整的标签对，清空内容
          if (currentDisplayableFinalContent.trim().startsWith("<think>")) {
            const startsWithCompleteThinkTagRegex = /^<think>[\s\S]*?<\/think>/;
            if (!startsWithCompleteThinkTagRegex.test(currentDisplayableFinalContent.trim())) {
              currentDisplayableFinalContent = "";
            }
          }
        }
      }

      const headerText = isThinkingStatus ? t('思考中...') : t('思考过程');
      const finalExtractedThinkingContent = currentExtractedThinkingContent;
      const finalDisplayableFinalContent = currentDisplayableFinalContent;

      if (message.role === 'assistant' &&
        isThinkingStatus &&
        !finalExtractedThinkingContent &&
        (!finalDisplayableFinalContent || finalDisplayableFinalContent.trim() === '')) {
        return (
          <div className={`${className} flex items-center gap-4 p-6 bg-gradient-to-r from-purple-50 to-indigo-50 rounded-2xl`}>
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center shadow-lg">
              <Loader2 className="animate-spin text-white" size={20} />
            </div>
            <div className="flex flex-col">
              <Typography.Text strong className="text-gray-800 text-base">
                {t('正在思考...')}
              </Typography.Text>
              <Typography.Text className="text-gray-500 text-sm">
                AI 正在分析您的问题
              </Typography.Text>
            </div>
          </div>
        );
      }

      return (
        <div className={className}>
          {message.role === 'assistant' && finalExtractedThinkingContent && (
            <div className="bg-gradient-to-br from-indigo-50 via-purple-50 to-pink-50 rounded-2xl mb-4 overflow-hidden shadow-sm backdrop-blur-sm">
              <div
                className="flex items-center justify-between p-5 cursor-pointer hover:bg-gradient-to-r hover:from-white/40 hover:to-purple-50/60 transition-all"
                onClick={() => toggleReasoningExpansion(message.id)}
              >
                <div className="flex items-center gap-4">
                  <div className="w-8 h-8 rounded-full bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center shadow-lg">
                    <Brain className="text-white" size={16} />
                  </div>
                  <div className="flex flex-col">
                    <Typography.Text strong className="text-gray-800 text-base">
                      {headerText}
                    </Typography.Text>
                    {thinkingSource && (
                      <Typography.Text className="text-gray-500 text-xs mt-0.5">
                        来源: {thinkingSource}
                      </Typography.Text>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {isThinkingStatus && (
                    <div className="flex items-center gap-2">
                      <Loader2 className="animate-spin text-purple-500" size={18} />
                      <Typography.Text className="text-purple-600 text-sm font-medium">
                        思考中
                      </Typography.Text>
                    </div>
                  )}
                  {!isThinkingStatus && (
                    <div className="w-6 h-6 rounded-full bg-purple-100 flex items-center justify-center">
                      {message.isReasoningExpanded ?
                        <ChevronUp size={16} className="text-purple-600" /> :
                        <ChevronRight size={16} className="text-purple-600" />
                      }
                    </div>
                  )}
                </div>
              </div>
              <div
                className={`transition-all duration-500 ease-out ${message.isReasoningExpanded ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
                  } overflow-hidden`}
              >
                {message.isReasoningExpanded && (
                  <div className="p-5 pt-4">
                    <div className="bg-white/70 backdrop-blur-sm rounded-xl p-4 shadow-inner overflow-x-auto max-h-50 overflow-y-auto">
                      <div className="prose prose-sm prose-purple max-w-none">
                        <MarkdownRender raw={finalExtractedThinkingContent} />
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {(finalDisplayableFinalContent && finalDisplayableFinalContent.trim() !== '') && (
            <div className="prose prose-sm prose-gray max-w-none overflow-x-auto">
              <MarkdownRender raw={finalDisplayableFinalContent} />
            </div>
          )}
        </div>
      );
    },
    [t, setMessage],
  );

  return (
    <div className="min-h-screen bg-gray-50">
      <Layout style={{ height: '100vh', background: 'transparent' }}>
        {(showSettings || !styleState.isMobile) && (
          <Layout.Sider
            style={{
              background: 'transparent',
              borderRight: 'none',
              flexShrink: 0,
              minWidth: 320,
              maxWidth: 320,
              height: 'calc(100vh - 100px)',
            }}
            width={320}
          >
            <Card className="!rounded-2xl h-full flex flex-col" bodyStyle={{ padding: '24px', height: '100%', display: 'flex', flexDirection: 'column' }}>
              <div className="flex items-center justify-between mb-6 flex-shrink-0">
                <div className="flex items-center">
                  <div className="w-10 h-10 rounded-full bg-gradient-to-r from-purple-500 to-blue-500 flex items-center justify-center mr-3">
                    <SlidersHorizontal size={20} className="text-white" />
                  </div>
                  <Typography.Title heading={5} className="mb-0">
                    {t('模型设置')}
                  </Typography.Title>
                </div>
                <DebugToggle />
              </div>

              <div className="space-y-6 overflow-y-auto flex-1 pr-2 model-settings-scroll">
                {/* 分组选择 */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Users size={16} className="text-gray-500" />
                    <Typography.Text strong className="text-sm">
                      {t('分组')}
                    </Typography.Text>
                  </div>
                  <Select
                    placeholder={t('请选择分组')}
                    name='group'
                    required
                    selection
                    onChange={(value) => handleInputChange('group', value)}
                    value={inputs.group}
                    autoComplete='new-password'
                    optionList={groups}
                    renderOptionItem={renderGroupOption}
                    style={{ width: '100%' }}
                    className="!rounded-lg"
                  />
                </div>

                {/* 模型选择 */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Sparkles size={16} className="text-gray-500" />
                    <Typography.Text strong className="text-sm">
                      {t('模型')}
                    </Typography.Text>
                  </div>
                  <Select
                    placeholder={t('请选择模型')}
                    name='model'
                    required
                    selection
                    searchPosition='dropdown'
                    filter
                    onChange={(value) => handleInputChange('model', value)}
                    value={inputs.model}
                    autoComplete='new-password'
                    optionList={models}
                    className="!rounded-lg"
                  />
                </div>

                {/* Temperature */}
                <div className={`transition-opacity duration-200 ${!parameterEnabled.temperature ? 'opacity-50' : ''}`}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Thermometer size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        Temperature
                      </Typography.Text>
                      <Tag size="small" className="!rounded-full">
                        {inputs.temperature}
                      </Tag>
                    </div>
                    <Button
                      theme={parameterEnabled.temperature ? 'solid' : 'borderless'}
                      type={parameterEnabled.temperature ? 'primary' : 'tertiary'}
                      size="small"
                      icon={parameterEnabled.temperature ? <Check size={10} /> : <X size={10} />}
                      onClick={() => handleParameterToggle('temperature')}
                      className="!rounded-full !w-6 !h-6 !p-0 !min-w-0"
                    />
                  </div>
                  <Typography.Text className="text-xs text-gray-500 mb-2">
                    控制输出的随机性和创造性
                  </Typography.Text>
                  <Slider
                    step={0.1}
                    min={0.1}
                    max={1}
                    value={inputs.temperature}
                    onChange={(value) => handleInputChange('temperature', value)}
                    className="mt-2"
                    disabled={!parameterEnabled.temperature}
                  />
                </div>

                {/* Top P */}
                <div className={`transition-opacity duration-200 ${!parameterEnabled.top_p ? 'opacity-50' : ''}`}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Target size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        Top P
                      </Typography.Text>
                      <Tag size="small" className="!rounded-full">
                        {inputs.top_p}
                      </Tag>
                    </div>
                    <Button
                      theme={parameterEnabled.top_p ? 'solid' : 'borderless'}
                      type={parameterEnabled.top_p ? 'primary' : 'tertiary'}
                      size="small"
                      icon={parameterEnabled.top_p ? <Check size={10} /> : <X size={10} />}
                      onClick={() => handleParameterToggle('top_p')}
                      className="!rounded-full !w-6 !h-6 !p-0 !min-w-0"
                    />
                  </div>
                  <Typography.Text className="text-xs text-gray-500 mb-2">
                    核采样，控制词汇选择的多样性
                  </Typography.Text>
                  <Slider
                    step={0.1}
                    min={0.1}
                    max={1}
                    value={inputs.top_p}
                    onChange={(value) => handleInputChange('top_p', value)}
                    className="mt-2"
                    disabled={!parameterEnabled.top_p}
                  />
                </div>

                {/* Frequency Penalty */}
                <div className={`transition-opacity duration-200 ${!parameterEnabled.frequency_penalty ? 'opacity-50' : ''}`}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Repeat size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        Frequency Penalty
                      </Typography.Text>
                      <Tag size="small" className="!rounded-full">
                        {inputs.frequency_penalty}
                      </Tag>
                    </div>
                    <Button
                      theme={parameterEnabled.frequency_penalty ? 'solid' : 'borderless'}
                      type={parameterEnabled.frequency_penalty ? 'primary' : 'tertiary'}
                      size="small"
                      icon={parameterEnabled.frequency_penalty ? <Check size={10} /> : <X size={10} />}
                      onClick={() => handleParameterToggle('frequency_penalty')}
                      className="!rounded-full !w-6 !h-6 !p-0 !min-w-0"
                    />
                  </div>
                  <Typography.Text className="text-xs text-gray-500 mb-2">
                    频率惩罚，减少重复词汇的出现
                  </Typography.Text>
                  <Slider
                    step={0.1}
                    min={-2}
                    max={2}
                    value={inputs.frequency_penalty}
                    onChange={(value) => handleInputChange('frequency_penalty', value)}
                    className="mt-2"
                    disabled={!parameterEnabled.frequency_penalty}
                  />
                </div>

                {/* Presence Penalty */}
                <div className={`transition-opacity duration-200 ${!parameterEnabled.presence_penalty ? 'opacity-50' : ''}`}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Ban size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        Presence Penalty
                      </Typography.Text>
                      <Tag size="small" className="!rounded-full">
                        {inputs.presence_penalty}
                      </Tag>
                    </div>
                    <Button
                      theme={parameterEnabled.presence_penalty ? 'solid' : 'borderless'}
                      type={parameterEnabled.presence_penalty ? 'primary' : 'tertiary'}
                      size="small"
                      icon={parameterEnabled.presence_penalty ? <Check size={10} /> : <X size={10} />}
                      onClick={() => handleParameterToggle('presence_penalty')}
                      className="!rounded-full !w-6 !h-6 !p-0 !min-w-0"
                    />
                  </div>
                  <Typography.Text className="text-xs text-gray-500 mb-2">
                    存在惩罚，鼓励讨论新话题
                  </Typography.Text>
                  <Slider
                    step={0.1}
                    min={-2}
                    max={2}
                    value={inputs.presence_penalty}
                    onChange={(value) => handleInputChange('presence_penalty', value)}
                    className="mt-2"
                    disabled={!parameterEnabled.presence_penalty}
                  />
                </div>

                {/* MaxTokens */}
                <div className={`transition-opacity duration-200 ${!parameterEnabled.max_tokens ? 'opacity-50' : ''}`}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Hash size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        Max Tokens
                      </Typography.Text>
                    </div>
                    <Button
                      theme={parameterEnabled.max_tokens ? 'solid' : 'borderless'}
                      type={parameterEnabled.max_tokens ? 'primary' : 'tertiary'}
                      size="small"
                      icon={parameterEnabled.max_tokens ? <Check size={10} /> : <X size={10} />}
                      onClick={() => handleParameterToggle('max_tokens')}
                      className="!rounded-full !w-6 !h-6 !p-0 !min-w-0"
                    />
                  </div>
                  <Input
                    placeholder='MaxTokens'
                    name='max_tokens'
                    required
                    autoComplete='new-password'
                    defaultValue={0}
                    value={inputs.max_tokens}
                    onChange={(value) => handleInputChange('max_tokens', value)}
                    className="!rounded-lg"
                    disabled={!parameterEnabled.max_tokens}
                  />
                </div>

                {/* Seed */}
                <div className={`transition-opacity duration-200 ${!parameterEnabled.seed ? 'opacity-50' : ''}`}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Shuffle size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        Seed
                      </Typography.Text>
                      <Typography.Text className="text-xs text-gray-400">
                        (可选，用于复现结果)
                      </Typography.Text>
                    </div>
                    <Button
                      theme={parameterEnabled.seed ? 'solid' : 'borderless'}
                      type={parameterEnabled.seed ? 'primary' : 'tertiary'}
                      size="small"
                      icon={parameterEnabled.seed ? <Check size={10} /> : <X size={10} />}
                      onClick={() => handleParameterToggle('seed')}
                      className="!rounded-full !w-6 !h-6 !p-0 !min-w-0"
                    />
                  </div>
                  <Input
                    placeholder='随机种子 (留空为随机)'
                    name='seed'
                    autoComplete='new-password'
                    value={inputs.seed || ''}
                    onChange={(value) => handleInputChange('seed', value === '' ? null : value)}
                    className="!rounded-lg"
                    disabled={!parameterEnabled.seed}
                  />
                </div>

                {/* Stream Toggle */}
                <div>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <ToggleLeft size={16} className="text-gray-500" />
                      <Typography.Text strong className="text-sm">
                        流式输出
                      </Typography.Text>
                    </div>
                    <Button
                      theme={inputs.stream ? 'solid' : 'borderless'}
                      type={inputs.stream ? 'primary' : 'tertiary'}
                      size="small"
                      onClick={() => handleInputChange('stream', !inputs.stream)}
                      className="!rounded-full"
                    >
                      {inputs.stream ? '开启' : '关闭'}
                    </Button>
                  </div>
                </div>

                {/* System Prompt */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Type size={16} className="text-gray-500" />
                    <Typography.Text strong className="text-sm">
                      System Prompt
                    </Typography.Text>
                  </div>
                  <TextArea
                    placeholder='System Prompt'
                    name='system'
                    required
                    autoComplete='new-password'
                    autosize
                    defaultValue={systemPrompt}
                    onChange={(value) => setSystemPrompt(value)}
                    className="!rounded-lg"
                    maxHeight={200}
                  />
                </div>
              </div>
            </Card>
          </Layout.Sider>
        )}

        <Layout.Content className="relative flex-1 overflow-hidden">
          <div className="px-4 overflow-hidden flex gap-4" style={{ height: 'calc(100vh - 100px)' }}>
            <div className="flex-1 flex flex-col">
              <SettingsToggle />
              <Card
                className="!rounded-2xl h-full"
                bodyStyle={{ padding: 0, height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
              >
                {/* 聊天头部 */}
                <div className="px-6 py-4 bg-gradient-to-r from-purple-500 to-blue-500 rounded-t-2xl">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-white/20 backdrop-blur flex items-center justify-center">
                      <MessageSquare size={20} className="text-white" />
                    </div>
                    <div>
                      <Typography.Title heading={5} className="!text-white mb-0">
                        {t('AI 对话')}
                      </Typography.Title>
                      <Typography.Text className="!text-white/80 text-sm">
                        {inputs.model || t('选择模型开始对话')}
                      </Typography.Text>
                    </div>
                  </div>
                </div>

                {/* 聊天内容区域 */}
                <div className="flex-1 overflow-hidden">
                  <Chat
                    chatBoxRenderConfig={{
                      renderChatBoxContent: renderCustomChatContent,
                      renderChatBoxAction: renderChatBoxAction,
                    }}
                    renderInputArea={renderInputArea}
                    roleConfig={roleInfo}
                    style={{
                      height: '100%',
                      maxWidth: '100%',
                      overflow: 'hidden'
                    }}
                    chats={message}
                    onMessageSend={onMessageSend}
                    onMessageCopy={handleMessageCopy}
                    onMessageReset={handleMessageReset}
                    onMessageDelete={handleMessageDelete}
                    showClearContext
                    showStopGenerate
                    onStopGenerator={onStopGenerator}
                    onClear={() => setMessage([])}
                    className="h-full"
                    placeholder={t('请输入您的问题...')}
                  />
                </div>
              </Card>
            </div>

            {/* 调试面板 */}
            {showDebugPanel && (
              <div className="w-96 flex-shrink-0">
                <Card className="!rounded-2xl h-full flex flex-col" bodyStyle={{ padding: '24px', height: '100%', display: 'flex', flexDirection: 'column' }}>
                  <div className="flex items-center mb-6 flex-shrink-0">
                    <div className="w-10 h-10 rounded-full bg-gradient-to-r from-green-500 to-blue-500 flex items-center justify-center mr-3">
                      <Code size={20} className="text-white" />
                    </div>
                    <Typography.Title heading={5} className="mb-0">
                      {t('调试信息')}
                    </Typography.Title>
                  </div>

                  <div className="flex-1 overflow-hidden debug-panel">
                    <Tabs
                      type="line"
                      className="h-full"
                      style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
                      activeKey={activeDebugTab}
                      onChange={setActiveDebugTab}
                    >
                      <TabPane tab={
                        <div className="flex items-center gap-2">
                          <FileText size={16} />
                          {t('请求体')}
                        </div>
                      } itemKey="request">
                        <div className="h-full overflow-y-auto bg-gray-50 rounded-lg p-4 model-settings-scroll">
                          {debugData.request ? (
                            <pre className="debug-code text-gray-700 whitespace-pre-wrap break-words">
                              {JSON.stringify(debugData.request, null, 2)}
                            </pre>
                          ) : (
                            <Typography.Text type="secondary" className="text-sm">
                              {t('暂无请求数据')}
                            </Typography.Text>
                          )}
                        </div>
                      </TabPane>

                      <TabPane tab={
                        <div className="flex items-center gap-2">
                          <Zap size={16} />
                          {t('响应内容')}
                        </div>
                      } itemKey="response">
                        <div className="h-full overflow-y-auto bg-gray-50 rounded-lg p-4 model-settings-scroll">
                          {debugData.response ? (
                            <pre className="debug-code text-gray-700 whitespace-pre-wrap break-words">
                              {debugData.response}
                            </pre>
                          ) : (
                            <Typography.Text type="secondary" className="text-sm">
                              {t('暂无响应数据')}
                            </Typography.Text>
                          )}
                        </div>
                      </TabPane>
                    </Tabs>
                  </div>

                  {debugData.timestamp && (
                    <div className="flex items-center gap-2 mt-4 pt-4 flex-shrink-0">
                      <Clock size={14} className="text-gray-500" />
                      <Typography.Text className="text-xs text-gray-500">
                        {t('最后更新')}: {new Date(debugData.timestamp).toLocaleString()}
                      </Typography.Text>
                    </div>
                  )}
                </Card>
              </div>
            )}
          </div>
        </Layout.Content>
      </Layout>
    </div>
  );
};

export default Playground;
