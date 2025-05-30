import { useCallback } from 'react';
import { Toast, Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { getTextContent } from '../utils/messageUtils';
import { ERROR_MESSAGES } from '../utils/constants';

export const useMessageActions = (message, setMessage, onMessageSend) => {
  const { t } = useTranslation();

  // 复制消息
  const handleMessageCopy = useCallback((targetMessage) => {
    const textToCopy = getTextContent(targetMessage);

    if (!textToCopy) {
      Toast.warning({
        content: t(ERROR_MESSAGES.NO_TEXT_CONTENT),
        duration: 2,
      });
      return;
    }

    const copyToClipboard = async (text) => {
      if (navigator.clipboard?.writeText) {
        try {
          await navigator.clipboard.writeText(text);
          Toast.success({
            content: t('消息已复制到剪贴板'),
            duration: 2,
          });
        } catch (err) {
          console.error('Clipboard API 复制失败:', err);
          fallbackCopy(text);
        }
      } else {
        fallbackCopy(text);
      }
    };

    const fallbackCopy = (text) => {
      try {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.cssText = `
          position: fixed;
          top: -9999px;
          left: -9999px;
          opacity: 0;
          pointer-events: none;
          z-index: -1;
        `;
        textArea.setAttribute('readonly', '');

        document.body.appendChild(textArea);
        textArea.select();
        textArea.setSelectionRange(0, text.length);

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

        let errorMessage = t(ERROR_MESSAGES.COPY_FAILED);
        if (window.location.protocol === 'http:' && window.location.hostname !== 'localhost') {
          errorMessage = t(ERROR_MESSAGES.COPY_HTTPS_REQUIRED);
        } else if (!navigator.clipboard && !document.execCommand) {
          errorMessage = t(ERROR_MESSAGES.BROWSER_NOT_SUPPORTED);
        }

        Toast.error({
          content: errorMessage,
          duration: 4,
        });
      }
    };

    copyToClipboard(textToCopy);
  }, [t]);

  // 重新生成消息
  const handleMessageReset = useCallback((targetMessage) => {
    setMessage(prevMessages => {
      const messageIndex = prevMessages.findIndex(msg => msg.id === targetMessage.id);
      if (messageIndex === -1) return prevMessages;

      if (targetMessage.role === 'user') {
        const newMessages = prevMessages.slice(0, messageIndex);
        const contentToSend = getTextContent(targetMessage);

        setTimeout(() => {
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
          const contentToSend = getTextContent(userMessage);

          setTimeout(() => {
            onMessageSend(contentToSend);
          }, 100);

          return newMessages;
        }
      }

      return prevMessages;
    });
  }, [setMessage, onMessageSend]);

  // 删除消息
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
              return prevMessages.filter((_, index) =>
                index !== messageIndex && index !== messageIndex + 1
              );
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

  // 切换角色
  const handleRoleToggle = useCallback((targetMessage) => {
    setMessage(prevMessages => {
      return prevMessages.map(msg => {
        if (msg.id === targetMessage.id &&
          (msg.role === 'assistant' || msg.role === 'system')) {
          const newRole = msg.role === 'assistant' ? 'system' : 'assistant';
          Toast.success({
            content: t(`已切换为${newRole === 'system' ? 'System' : 'Assistant'}角色`),
            duration: 2,
          });
          return { ...msg, role: newRole };
        }
        return msg;
      });
    });
  }, [setMessage, t]);

  return {
    handleMessageCopy,
    handleMessageReset,
    handleMessageDelete,
    handleRoleToggle,
  };
}; 