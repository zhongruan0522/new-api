import React, { useContext, useEffect, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Layout, Toast, Modal } from '@douyinfe/semi-ui';

// Context
import { UserContext } from '../../context/User/index.js';
import { StyleContext } from '../../context/Style/index.js';

// Utils and hooks
import { API, showError, getLogo, isMobile } from '../../helpers/index.js';
import { stringToColor } from '../../helpers/render.js';
import { usePlaygroundState } from '../../hooks/usePlaygroundState.js';
import { useMessageActions } from '../../hooks/useMessageActions.js';
import { useApiRequest } from '../../hooks/useApiRequest.js';

// Constants and utils
import {
  DEFAULT_MESSAGES,
  MESSAGE_ROLES,
  API_ENDPOINTS
} from '../../utils/constants.js';
import {
  buildMessageContent,
  createMessage,
  createLoadingAssistantMessage,
  getTextContent
} from '../../utils/messageUtils.js';
import {
  buildApiPayload,
  processModelsData,
  processGroupsData
} from '../../utils/apiUtils.js';

// Components
import SettingsPanel from '../../components/playground/SettingsPanel.js';
import ChatArea from '../../components/playground/ChatArea.js';
import DebugPanel from '../../components/playground/DebugPanel.js';
import MessageContent from '../../components/playground/MessageContent.js';
import MessageActions from '../../components/playground/MessageActions.js';
import FloatingButtons from '../../components/playground/FloatingButtons.js';

// 生成头像
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
  const [userState] = useContext(UserContext);
  const [styleState, styleDispatch] = useContext(StyleContext);
  const [searchParams] = useSearchParams();

  // 使用自定义hooks
  const state = usePlaygroundState();
  const {
    inputs,
    parameterEnabled,
    systemPrompt,
    showDebugPanel,
    showSettings,
    models,
    groups,
    status,
    message,
    debugData,
    activeDebugTab,
    previewPayload,
    editingMessageId,
    editValue,
    sseSourceRef,
    chatRef,
    handleInputChange,
    handleParameterToggle,
    debouncedSaveConfig,
    handleConfigImport,
    handleConfigReset,
    setShowSettings,
    setModels,
    setGroups,
    setStatus,
    setMessage,
    setDebugData,
    setActiveDebugTab,
    setPreviewPayload,
    setEditingMessageId,
    setEditValue,
    setSystemPrompt,
    setShowDebugPanel,
  } = state;

  // API 请求相关
  const { sendRequest, onStopGenerator } = useApiRequest(
    setMessage,
    setDebugData,
    setActiveDebugTab,
    sseSourceRef
  );

  // 角色信息
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
      avatar: getLogo(),
    },
  };

  // 消息操作
  const messageActions = useMessageActions(message, setMessage, onMessageSend);

  // 构建预览请求体
  const constructPreviewPayload = useCallback(() => {
    try {
      const systemMessage = systemPrompt !== '' ? createMessage(
        MESSAGE_ROLES.SYSTEM,
        systemPrompt,
        { id: '1', createAt: 1715676751919 }
      ) : null;

      let messages = [...message];

      // 如果没有用户消息，添加默认消息
      if (messages.length === 0 || messages.every(msg => msg.role !== MESSAGE_ROLES.USER)) {
        const validImageUrls = inputs.imageUrls ? inputs.imageUrls.filter(url => url.trim() !== '') : [];
        const content = buildMessageContent('你好', validImageUrls, inputs.imageEnabled);
        messages = [createMessage(MESSAGE_ROLES.USER, content)];
      } else {
        // 处理最后一个用户消息的图片
        const lastUserMessageIndex = messages.length - 1;
        for (let i = messages.length - 1; i >= 0; i--) {
          if (messages[i].role === MESSAGE_ROLES.USER) {
            if (inputs.imageEnabled && inputs.imageUrls) {
              const validImageUrls = inputs.imageUrls.filter(url => url.trim() !== '');
              if (validImageUrls.length > 0) {
                const textContent = getTextContent(messages[i]) || '示例消息';
                const content = buildMessageContent(textContent, validImageUrls, true);
                messages[i] = { ...messages[i], content };
              }
            }
            break;
          }
        }
      }

      return buildApiPayload(messages, systemMessage, inputs, parameterEnabled);
    } catch (error) {
      console.error('构造预览请求体失败:', error);
      return null;
    }
  }, [inputs, parameterEnabled, systemPrompt, message]);

  // 发送消息
  function onMessageSend(content, attachment) {
    console.log('attachment: ', attachment);

    const validImageUrls = inputs.imageUrls.filter(url => url.trim() !== '');
    const messageContent = buildMessageContent(content, validImageUrls, inputs.imageEnabled);

    const userMessage = createMessage(MESSAGE_ROLES.USER, messageContent);
    const loadingMessage = createLoadingAssistantMessage();

    setMessage(prevMessage => {
      const newMessages = [...prevMessage, userMessage];

      const systemMessage = systemPrompt !== '' ? createMessage(
        MESSAGE_ROLES.SYSTEM,
        systemPrompt,
        { id: '1', createAt: 1715676751919 }
      ) : null;

      const payload = buildApiPayload(newMessages, systemMessage, inputs, parameterEnabled);
      sendRequest(payload, inputs.stream);

      // 禁用图片模式
      if (inputs.imageEnabled) {
        setTimeout(() => {
          handleInputChange('imageEnabled', false);
        }, 100);
      }

      return [...newMessages, loadingMessage];
    });
  }

  // 加载模型和分组
  const loadModels = async () => {
    try {
      const res = await API.get(API_ENDPOINTS.USER_MODELS);
      const { success, message, data } = res.data;

      if (success) {
        const { modelOptions, selectedModel } = processModelsData(data, inputs.model);
        setModels(modelOptions);

        if (selectedModel !== inputs.model) {
          handleInputChange('model', selectedModel);
        }
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(t('加载模型失败'));
    }
  };

  const loadGroups = async () => {
    try {
      const res = await API.get(API_ENDPOINTS.USER_GROUPS);
      const { success, message, data } = res.data;

      if (success) {
        const userGroup = userState?.user?.group || JSON.parse(localStorage.getItem('user'))?.group;
        const groupOptions = processGroupsData(data, userGroup);
        setGroups(groupOptions);

        const hasCurrentGroup = groupOptions.some(option => option.value === inputs.group);
        if (!hasCurrentGroup) {
          handleInputChange('group', groupOptions[0]?.value || '');
        }
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(t('加载分组失败'));
    }
  };

  // 编辑消息相关
  const handleMessageEdit = useCallback((targetMessage) => {
    const editableContent = getTextContent(targetMessage);
    setEditingMessageId(targetMessage.id);
    setEditValue(editableContent);
  }, [setEditingMessageId, setEditValue]);

  const handleEditSave = useCallback(() => {
    if (!editingMessageId || !editValue.trim()) return;

    setMessage(prevMessages => {
      const messageIndex = prevMessages.findIndex(msg => msg.id === editingMessageId);
      if (messageIndex === -1) return prevMessages;

      const targetMessage = prevMessages[messageIndex];
      let newContent;

      if (Array.isArray(targetMessage.content)) {
        newContent = targetMessage.content.map(item =>
          item.type === 'text' ? { ...item, text: editValue.trim() } : item
        );
      } else {
        newContent = editValue.trim();
      }

      const updatedMessages = prevMessages.map(msg =>
        msg.id === editingMessageId ? { ...msg, content: newContent } : msg
      );

      // 处理用户消息编辑后的重新生成
      if (targetMessage.role === MESSAGE_ROLES.USER) {
        const hasSubsequentAssistantReply = messageIndex < prevMessages.length - 1 &&
          prevMessages[messageIndex + 1].role === MESSAGE_ROLES.ASSISTANT;

        if (hasSubsequentAssistantReply) {
          Modal.confirm({
            title: t('消息已编辑'),
            content: t('检测到该消息后有AI回复，是否删除后续回复并重新生成？'),
            okText: t('重新生成'),
            cancelText: t('仅保存'),
            onOk: () => {
              const messagesUntilUser = updatedMessages.slice(0, messageIndex + 1);
              setMessage(messagesUntilUser);

              setTimeout(() => {
                const systemMessage = systemPrompt !== '' ? createMessage(
                  MESSAGE_ROLES.SYSTEM,
                  systemPrompt,
                  { id: '1', createAt: 1715676751919 }
                ) : null;

                const payload = buildApiPayload(messagesUntilUser, systemMessage, inputs, parameterEnabled);

                setMessage(prevMsg => [...prevMsg, createLoadingAssistantMessage()]);
                sendRequest(payload, inputs.stream);
              }, 100);
            },
            onCancel: () => setMessage(updatedMessages)
          });
          return prevMessages;
        }
      }

      return updatedMessages;
    });

    setEditingMessageId(null);
    setEditValue('');
    Toast.success({ content: t('消息已更新'), duration: 2 });
  }, [editingMessageId, editValue, t, systemPrompt, inputs, parameterEnabled, sendRequest, setMessage, setEditingMessageId, setEditValue]);

  const handleEditCancel = useCallback(() => {
    setEditingMessageId(null);
    setEditValue('');
  }, [setEditingMessageId, setEditValue]);

  // 切换推理展开状态
  const toggleReasoningExpansion = (messageId) => {
    setMessage(prevMessages =>
      prevMessages.map(msg =>
        msg.id === messageId && msg.role === MESSAGE_ROLES.ASSISTANT
          ? { ...msg, isReasoningExpanded: !msg.isReasoningExpanded }
          : msg
      )
    );
  };

  // 渲染函数
  const renderCustomChatContent = useCallback(
    ({ message, className }) => {
      const isCurrentlyEditing = editingMessageId === message.id;

      return (
        <MessageContent
          message={message}
          className={className}
          styleState={styleState}
          onToggleReasoningExpansion={toggleReasoningExpansion}
          isEditing={isCurrentlyEditing}
          onEditSave={handleEditSave}
          onEditCancel={handleEditCancel}
          editValue={editValue}
          onEditValueChange={setEditValue}
        />
      );
    },
    [styleState, editingMessageId, editValue, handleEditSave, handleEditCancel, setEditValue],
  );

  const renderChatBoxAction = useCallback((props) => {
    const { message: currentMessage } = props;
    const isAnyMessageGenerating = message.some(msg =>
      msg.status === 'loading' || msg.status === 'incomplete'
    );
    const isCurrentlyEditing = editingMessageId === currentMessage.id;

    return (
      <MessageActions
        message={currentMessage}
        styleState={styleState}
        onMessageReset={messageActions.handleMessageReset}
        onMessageCopy={messageActions.handleMessageCopy}
        onMessageDelete={messageActions.handleMessageDelete}
        onRoleToggle={messageActions.handleRoleToggle}
        onMessageEdit={handleMessageEdit}
        isAnyMessageGenerating={isAnyMessageGenerating}
        isEditing={isCurrentlyEditing}
      />
    );
  }, [messageActions, styleState, message, editingMessageId, handleMessageEdit]);

  // Effects
  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('未登录或登录已过期，请重新登录！'));
    }

    const savedStatus = localStorage.getItem('status');
    if (savedStatus) {
      setStatus(JSON.parse(savedStatus));
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
    return () => window.removeEventListener('resize', handleResize);
  }, [styleDispatch]);

  useEffect(() => {
    const newPreviewPayload = constructPreviewPayload();
    setPreviewPayload(newPreviewPayload);
    setDebugData(prev => ({
      ...prev,
      previewRequest: newPreviewPayload,
      previewTimestamp: new Date().toISOString()
    }));
  }, [constructPreviewPayload, setPreviewPayload, setDebugData]);

  // 监听配置变化并自动保存
  useEffect(() => {
    debouncedSaveConfig();
  }, [inputs, parameterEnabled, systemPrompt, showDebugPanel]);

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
                onMessageCopy={messageActions.handleMessageCopy}
                onMessageReset={messageActions.handleMessageReset}
                onMessageDelete={messageActions.handleMessageDelete}
                onRoleToggle={messageActions.handleRoleToggle}
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
