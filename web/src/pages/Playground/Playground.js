import React, { useCallback, useContext, useEffect, useState } from 'react';
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
} from '@douyinfe/semi-ui';
import { SSE } from 'sse';
import { IconSetting, IconSpin, IconChevronRight, IconChevronUp } from '@douyinfe/semi-icons';
import { StyleContext } from '../../context/Style/index.js';
import { useTranslation } from 'react-i18next';
import { renderGroupOption, truncateText, stringToColor } from '../../helpers/render.js';

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
  const [styleState, styleDispatch] = useContext(StyleContext);

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
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

  const commonOuterStyle = {
    border: '1px solid var(--semi-color-border)',
    borderRadius: '16px',
    margin: '0px 8px',
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

  let handleSSE = (payload) => {
    let source = new SSE('/pg/chat/completions', {
      headers: {
        'Content-Type': 'application/json',
        'New-Api-User': getUserIdFromLocalStorage(),
      },
      method: 'POST',
      payload: JSON.stringify(payload),
    });

    source.addEventListener('message', (e) => {
      if (e.data === '[DONE]') {
        source.close();
        completeMessage();
        return;
      }

      try {
        let payload = JSON.parse(e.data);
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
        streamMessageUpdate(t('解析响应数据时发生错误'), 'content');
        completeMessage('error');
      }
    });

    source.addEventListener('error', (e) => {
      console.error('SSE Error:', e);
      const errorMessage = e.data || t('请求发生错误');
      streamMessageUpdate(errorMessage, 'content');
      completeMessage('error');
      source.close();
    });

    source.addEventListener('readystatechange', (e) => {
      if (e.readyState >= 2) {
        if (source.status !== undefined && source.status !== 200) {
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
          return {
            messages: messages,
            stream: true,
            model: inputs.model,
            group: inputs.group,
            max_tokens: parseInt(inputs.max_tokens),
            temperature: inputs.temperature,
          };
        };

        handleSSE(getPayload());
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
          newMessage = {
            ...newMessage,
            content: (lastMessage.content || '') + textChunk,
            status: 'incomplete',
          };
        }
      }
      return [...prevMessage.slice(0, -1), newMessage];
    });
  }, [setMessage]);

  const SettingsToggle = () => {
    if (!styleState.isMobile) return null;
    return (
      <Button
        icon={<IconSetting />}
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
      <div
        style={{
          margin: '8px 16px',
          display: 'flex',
          flexDirection: 'row',
          alignItems: 'flex-end',
          borderRadius: 16,
          padding: 10,
          border: '1px solid var(--semi-color-border)',
        }}
        onClick={onClick}
      >
        {inputNode}
        {sendNode}
      </div>
    );
  }

  const renderInputArea = useCallback((props) => {
    return <CustomInputRender {...props} />;
  }, []);

  const renderCustomChatContent = useCallback(
    ({ message, className }) => {
      if (message.status === 'error') {
        return (
          <div className={className} style={{
            display: 'flex',
            alignItems: 'center',
            padding: '12px',
            color: 'var(--semi-color-danger)'
          }}>
            <Typography.Text type="danger">{message.content || t('请求发生错误')}</Typography.Text>
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
          const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
          let match;

          thinkTagRegex.lastIndex = 0;
          while ((match = thinkTagRegex.exec(fullContent)) !== null) {
            replyParts.push(fullContent.substring(lastIndex, match.index));
            thoughts.push(match[1]);
            lastIndex = match.index + match[0].length;
          }
          replyParts.push(fullContent.substring(lastIndex));

          currentDisplayableFinalContent = replyParts.join('').trim();

          if (thoughts.length > 0) {
            currentExtractedThinkingContent = thoughts.join('\n\n---\n\n');
            thinkingSource = '<think> tags';
          }

          if (isThinkingStatus && currentDisplayableFinalContent.includes('<think')) {
            const lastOpenThinkIndex = currentDisplayableFinalContent.lastIndexOf('<think>');
            if (lastOpenThinkIndex !== -1) {
              const fragmentAfterLastOpen = currentDisplayableFinalContent.substring(lastOpenThinkIndex);
              if (!fragmentAfterLastOpen.substring("<think>".length).includes('</think>')) {
                const unclosedThought = fragmentAfterLastOpen.substring("<think>".length);
                if (currentExtractedThinkingContent) {
                  currentExtractedThinkingContent += (currentExtractedThinkingContent ? '\n\n---\n\n' : '') + unclosedThought;
                } else {
                  currentExtractedThinkingContent = unclosedThought;
                }
                if (!thinkingSource && unclosedThought) thinkingSource = '<think> tags (streaming)';
                currentDisplayableFinalContent = currentDisplayableFinalContent.substring(0, lastOpenThinkIndex).trim();
              }
            }
          }
        }

        if (typeof currentDisplayableFinalContent === 'string' && currentDisplayableFinalContent.trim().startsWith("<think>")) {
          const startsWithCompleteThinkTagRegex = /^<think>[\s\S]*?<\/think>/;
          if (!startsWithCompleteThinkTagRegex.test(currentDisplayableFinalContent.trim())) {
            currentDisplayableFinalContent = "";
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
          <div className={className} style={{ display: 'flex', alignItems: 'center', padding: '12px' }}>
            <IconSpin spin />
            <Typography.Text type="secondary" style={{ marginLeft: '8px' }}>{t('正在思考...')}</Typography.Text>
          </div>
        );
      }

      return (
        <div className={className}>
          {message.role === 'assistant' && finalExtractedThinkingContent && (
            <div style={{
              background: 'var(--semi-color-tertiary-light-hover)',
              borderRadius: '16px',
              marginBottom: '8px',
              overflow: 'hidden',
            }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '8px 12px',
                  cursor: 'pointer',
                  height: 'auto',
                }}
                onClick={() => toggleReasoningExpansion(message.id)}
              >
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <Typography.Text strong={message.isReasoningExpanded} style={{ fontSize: '13px', color: 'var(--semi-color-text-1)' }}>{headerText}</Typography.Text>
                  {thinkingSource && (
                    <Tag size="small" color='green' shape="circle" style={{ marginLeft: '8px' }}>
                      {thinkingSource}
                    </Tag>
                  )}
                </div>
                <div>
                  {isThinkingStatus && <IconSpin spin />}
                  {!isThinkingStatus && (message.isReasoningExpanded ? <IconChevronUp size="small" /> : <IconChevronRight size="small" />)}
                </div>
              </div>
              <div
                style={{
                  maxHeight: message.isReasoningExpanded ? '160px' : '0px',
                  overflowY: message.isReasoningExpanded ? 'auto' : 'hidden',
                  overflowX: 'hidden',
                  transition: 'max-height 0.3s ease-in-out, padding 0.3s ease-in-out',
                  padding: message.isReasoningExpanded ? '0px 12px 12px 12px' : '0px 12px',
                  boxSizing: 'border-box',
                }}
              >
                <MarkdownRender raw={finalExtractedThinkingContent} />
              </div>
            </div>
          )}

          {(finalDisplayableFinalContent && finalDisplayableFinalContent.trim() !== '') && (
            <MarkdownRender raw={finalDisplayableFinalContent} />
          )}
          {!(finalExtractedThinkingContent) && !(finalDisplayableFinalContent && finalDisplayableFinalContent.trim() !== '') && message.role === 'assistant' && (
            <div></div>
          )}
        </div>
      );
    },
    [t, setMessage],
  );

  return (
    <Layout style={{ height: '100%' }}>
      {(showSettings || !styleState.isMobile) && (
        <Layout.Sider
          style={{ display: styleState.isMobile ? 'block' : 'initial' }}
        >
          <Card style={commonOuterStyle}>
            <div style={{ marginTop: 10 }}>
              <Typography.Text strong>{t('分组')}：</Typography.Text>
            </div>
            <Select
              placeholder={t('请选择分组')}
              name='group'
              required
              selection
              onChange={(value) => {
                handleInputChange('group', value);
              }}
              value={inputs.group}
              autoComplete='new-password'
              optionList={groups}
              renderOptionItem={renderGroupOption}
              style={{ width: '100%' }}
            />
            <div style={{ marginTop: 10 }}>
              <Typography.Text strong>{t('模型')}：</Typography.Text>
            </div>
            <Select
              placeholder={t('请选择模型')}
              name='model'
              required
              selection
              searchPosition='dropdown'
              filter
              onChange={(value) => {
                handleInputChange('model', value);
              }}
              value={inputs.model}
              autoComplete='new-password'
              optionList={models}
            />
            <div style={{ marginTop: 10 }}>
              <Typography.Text strong>Temperature：</Typography.Text>
            </div>
            <Slider
              step={0.1}
              min={0.1}
              max={1}
              value={inputs.temperature}
              onChange={(value) => {
                handleInputChange('temperature', value);
              }}
            />
            <div style={{ marginTop: 10 }}>
              <Typography.Text strong>MaxTokens：</Typography.Text>
            </div>
            <Input
              placeholder='MaxTokens'
              name='max_tokens'
              required
              autoComplete='new-password'
              defaultValue={0}
              value={inputs.max_tokens}
              onChange={(value) => {
                handleInputChange('max_tokens', value);
              }}
            />

            <div style={{ marginTop: 10 }}>
              <Typography.Text strong>System：</Typography.Text>
            </div>
            <TextArea
              placeholder='System Prompt'
              name='system'
              required
              autoComplete='new-password'
              autosize
              defaultValue={systemPrompt}
              onChange={(value) => {
                setSystemPrompt(value);
              }}
            />
          </Card>
        </Layout.Sider>
      )}
      <Layout.Content>
        <div style={{ height: '100%', position: 'relative' }}>
          <SettingsToggle />
          <Chat
            chatBoxRenderConfig={{
              renderChatBoxContent: renderCustomChatContent,
              renderChatBoxAction: () => {
                return <div></div>;
              },
            }}
            renderInputArea={renderInputArea}
            roleConfig={roleInfo}
            style={commonOuterStyle}
            chats={message}
            onMessageSend={onMessageSend}
            showClearContext
            onClear={() => {
              setMessage([]);
            }}
          />
        </div>
      </Layout.Content>
    </Layout>
  );
};

export default Playground;
