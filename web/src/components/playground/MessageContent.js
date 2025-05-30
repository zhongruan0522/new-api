import React from 'react';
import {
  Typography,
  TextArea,
  Button,
} from '@douyinfe/semi-ui';
import MarkdownRenderer from '../common/MarkdownRenderer';
import {
  ChevronRight,
  ChevronUp,
  Brain,
  Loader2,
  Check,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

const MessageContent = ({
  message,
  className,
  styleState,
  onToggleReasoningExpansion,
  isEditing = false,
  onEditSave,
  onEditCancel,
  editValue,
  onEditValueChange
}) => {
  const { t } = useTranslation();

  if (message.status === 'error') {
    let errorText;

    if (Array.isArray(message.content)) {
      const textContent = message.content.find(item => item.type === 'text');
      errorText = textContent && textContent.text && typeof textContent.text === 'string'
        ? textContent.text
        : t('请求发生错误');
    } else if (typeof message.content === 'string') {
      errorText = message.content;
    } else {
      errorText = t('请求发生错误');
    }

    return (
      <div className={`${className} flex items-center p-4 bg-red-50 rounded-xl`}>
        <Typography.Text type="danger" className="text-sm">
          {errorText}
        </Typography.Text>
      </div>
    );
  }

  const isThinkingStatus = message.status === 'loading' || message.status === 'incomplete';
  let currentExtractedThinkingContent = null;
  let currentDisplayableFinalContent = "";
  let thinkingSource = null;

  const getTextContent = (content) => {
    if (Array.isArray(content)) {
      const textItem = content.find(item => item.type === 'text');
      return textItem && textItem.text && typeof textItem.text === 'string' ? textItem.text : '';
    } else if (typeof content === 'string') {
      return content;
    }
    return '';
  };

  currentDisplayableFinalContent = getTextContent(message.content);

  if (message.role === 'assistant') {
    let baseContentForDisplay = getTextContent(message.content);
    let combinedThinkingContent = "";

    if (message.reasoningContent) {
      combinedThinkingContent = message.reasoningContent;
      thinkingSource = 'reasoningContent';
    }

    if (baseContentForDisplay.includes('<think>')) {
      const thinkTagRegex = /<think>([\s\S]*?)<\/think>/g;
      let match;
      let thoughtsFromPairedTags = [];
      let replyParts = [];
      let lastIndex = 0;

      while ((match = thinkTagRegex.exec(baseContentForDisplay)) !== null) {
        replyParts.push(baseContentForDisplay.substring(lastIndex, match.index));
        thoughtsFromPairedTags.push(match[1]);
        lastIndex = match.index + match[0].length;
      }
      replyParts.push(baseContentForDisplay.substring(lastIndex));

      if (thoughtsFromPairedTags.length > 0) {
        const pairedThoughtsStr = thoughtsFromPairedTags.join('\n\n---\n\n');
        if (combinedThinkingContent) {
          combinedThinkingContent += '\n\n---\n\n' + pairedThoughtsStr;
        } else {
          combinedThinkingContent = pairedThoughtsStr;
        }
        thinkingSource = thinkingSource ? thinkingSource + ' & <think> tags' : '<think> tags';
      }

      baseContentForDisplay = replyParts.join('');
    }

    if (isThinkingStatus) {
      const lastOpenThinkIndex = baseContentForDisplay.lastIndexOf('<think>');
      if (lastOpenThinkIndex !== -1) {
        const fragmentAfterLastOpen = baseContentForDisplay.substring(lastOpenThinkIndex);
        if (!fragmentAfterLastOpen.includes('</think>')) {
          const unclosedThought = fragmentAfterLastOpen.substring('<think>'.length).trim();
          if (unclosedThought) {
            if (combinedThinkingContent) {
              combinedThinkingContent += '\n\n---\n\n' + unclosedThought;
            } else {
              combinedThinkingContent = unclosedThought;
            }
            thinkingSource = thinkingSource ? thinkingSource + ' + streaming <think>' : 'streaming <think>';
          }
          baseContentForDisplay = baseContentForDisplay.substring(0, lastOpenThinkIndex);
        }
      }
    }

    currentExtractedThinkingContent = combinedThinkingContent || null;
    currentDisplayableFinalContent = baseContentForDisplay.replace(/<\/?think>/g, '').trim();
  }

  const headerText = (isThinkingStatus && !message.isThinkingComplete) ? t('思考中...') : t('思考过程');
  const finalExtractedThinkingContent = currentExtractedThinkingContent;
  const finalDisplayableFinalContent = currentDisplayableFinalContent;

  if (message.role === 'assistant' &&
    isThinkingStatus &&
    !finalExtractedThinkingContent &&
    (!finalDisplayableFinalContent || finalDisplayableFinalContent.trim() === '')) {
    return (
      <div className={`${className} flex items-center gap-2 sm:gap-4 p-4 sm:p-6 bg-gradient-to-r from-purple-50 to-indigo-50 rounded-xl sm:rounded-2xl`}>
        <div className="w-8 h-8 sm:w-10 sm:h-10 rounded-full bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center shadow-lg">
          <Loader2 className="animate-spin text-white" size={styleState.isMobile ? 16 : 20} />
        </div>
        <div className="flex flex-col">
          <Typography.Text strong className="text-gray-800 text-sm sm:text-base">
            {t('正在思考...')}
          </Typography.Text>
          <Typography.Text className="text-gray-500 text-xs sm:text-sm">
            AI 正在分析您的问题
          </Typography.Text>
        </div>
      </div>
    );
  }

  return (
    <div className={className}>
      {/* 为system角色添加特殊标识 */}
      {message.role === 'system' && (
        <div className="mb-2 sm:mb-4">
          <div className="flex items-center gap-2 p-2 sm:p-3 bg-gradient-to-r from-amber-50 to-orange-50 rounded-lg" style={{ border: '1px solid var(--semi-color-border)' }}>
            <div className="w-4 h-4 sm:w-5 sm:h-5 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center shadow-sm">
              <Typography.Text className="text-white text-xs font-bold">S</Typography.Text>
            </div>
            <Typography.Text className="text-amber-700 text-xs sm:text-sm font-medium">
              {t('系统消息')}
            </Typography.Text>
          </div>
        </div>
      )}

      {/* 渲染推理内容 */}
      {message.role === 'assistant' && finalExtractedThinkingContent && (
        <div className="rounded-xl sm:rounded-2xl mb-2 sm:mb-4 overflow-hidden shadow-sm backdrop-blur-sm">
          <div
            className="flex items-center justify-between p-3 sm:p-5 cursor-pointer hover:bg-gradient-to-r hover:from-white/20 hover:to-purple-50/30 transition-all"
            style={{
              background: 'linear-gradient(135deg, #4c1d95 0%, #6d28d9 50%, #7c3aed 100%)',
              position: 'relative'
            }}
            onClick={() => onToggleReasoningExpansion(message.id)}
          >
            <div className="absolute inset-0 overflow-hidden">
              <div className="absolute -top-10 -right-10 w-40 h-40 bg-white opacity-5 rounded-full"></div>
              <div className="absolute -bottom-8 -left-8 w-24 h-24 bg-white opacity-10 rounded-full"></div>
            </div>
            <div className="flex items-center gap-2 sm:gap-4 relative">
              <div className="w-6 h-6 sm:w-8 sm:h-8 rounded-full bg-white/20 flex items-center justify-center shadow-lg">
                <Brain style={{ color: 'white' }} size={styleState.isMobile ? 12 : 16} />
              </div>
              <div className="flex flex-col">
                <Typography.Text strong style={{ color: 'white' }} className="text-sm sm:text-base">
                  {headerText}
                </Typography.Text>
                {thinkingSource && (
                  <Typography.Text style={{ color: 'white' }} className="text-xs mt-0.5 opacity-80 hidden sm:block">
                    来源: {thinkingSource}
                  </Typography.Text>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2 sm:gap-3 relative">
              {isThinkingStatus && !message.isThinkingComplete && (
                <div className="flex items-center gap-1 sm:gap-2">
                  <Loader2 style={{ color: 'white' }} className="animate-spin" size={styleState.isMobile ? 14 : 18} />
                  <Typography.Text style={{ color: 'white' }} className="text-xs sm:text-sm font-medium opacity-90">
                    思考中
                  </Typography.Text>
                </div>
              )}
              {(!isThinkingStatus || message.isThinkingComplete) && (
                <div className="w-5 h-5 sm:w-6 sm:h-6 rounded-full bg-white/20 flex items-center justify-center">
                  {message.isReasoningExpanded ?
                    <ChevronUp size={styleState.isMobile ? 12 : 16} style={{ color: 'white' }} /> :
                    <ChevronRight size={styleState.isMobile ? 12 : 16} style={{ color: 'white' }} />
                  }
                </div>
              )}
            </div>
          </div>
          <div
            className={`transition-all duration-500 ease-out ${message.isReasoningExpanded ? 'max-h-96 opacity-100' : 'max-h-0 opacity-0'
              } overflow-hidden bg-gradient-to-br from-purple-50 via-indigo-50 to-violet-50`}
          >
            {message.isReasoningExpanded && (
              <div className="p-3 sm:p-5 pt-2 sm:pt-4">
                <div
                  className="bg-white/70 backdrop-blur-sm rounded-lg sm:rounded-xl p-2 shadow-inner overflow-x-auto overflow-y-auto thinking-content-scroll"
                  style={{
                    maxHeight: '200px',
                    minHeight: '100px',
                    scrollbarWidth: 'thin',
                    scrollbarColor: 'rgba(0, 0, 0, 0.3) transparent'
                  }}
                >
                  <div className="prose prose-xs sm:prose-sm prose-purple max-w-none text-xs sm:text-sm">
                    <MarkdownRenderer
                      content={finalExtractedThinkingContent}
                      className=""
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 渲染消息内容 */}
      {isEditing ? (
        /* 编辑模式 */
        <div className="space-y-3">
          <TextArea
            value={editValue}
            onChange={(value) => onEditValueChange(value)}
            placeholder={t('请输入消息内容...')}
            autosize={{ minRows: 3, maxRows: 12 }}
            style={{
              resize: 'vertical',
              fontSize: styleState.isMobile ? '14px' : '15px',
              lineHeight: '1.6',
            }}
            className="!border-blue-200 focus:!border-blue-400 !bg-blue-50/50"
          />
          <div className="flex items-center gap-2 w-full">
            <Button
              size="small"
              type="danger"
              theme="light"
              icon={<X size={14} />}
              onClick={onEditCancel}
              className="flex-1"
            >
              {t('取消')}
            </Button>
            <Button
              size="small"
              type="warning"
              theme="solid"
              icon={<Check size={14} />}
              onClick={onEditSave}
              disabled={!editValue || editValue.trim() === ''}
              className="flex-1"
            >
              {t('保存')}
            </Button>
          </div>
        </div>
      ) : (
        /* 正常显示模式 */
        (() => {
          if (Array.isArray(message.content)) {
            const textContent = message.content.find(item => item.type === 'text');
            const imageContents = message.content.filter(item => item.type === 'image_url');

            return (
              <div>
                {/* 显示图片 */}
                {imageContents.length > 0 && (
                  <div className="mb-3 space-y-2">
                    {imageContents.map((imgItem, index) => (
                      <div key={index} className="max-w-sm">
                        <img
                          src={imgItem.image_url.url}
                          alt={`用户上传的图片 ${index + 1}`}
                          className="rounded-lg max-w-full h-auto shadow-sm border"
                          style={{ maxHeight: '300px' }}
                          onError={(e) => {
                            e.target.style.display = 'none';
                            e.target.nextSibling.style.display = 'block';
                          }}
                        />
                        <div
                          className="text-red-500 text-sm p-2 bg-red-50 rounded-lg border border-red-200"
                          style={{ display: 'none' }}
                        >
                          图片加载失败: {imgItem.image_url.url}
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* 显示文本内容 */}
                {textContent && textContent.text && typeof textContent.text === 'string' && textContent.text.trim() !== '' && (
                  <div className={`prose prose-xs sm:prose-sm prose-gray max-w-none overflow-x-auto text-xs sm:text-sm ${message.role === 'user' ? 'user-message' : ''}`}>
                    <MarkdownRenderer
                      content={textContent.text}
                      className={message.role === 'user' ? 'user-message' : ''}
                    />
                  </div>
                )}
              </div>
            );
          }

          if (typeof message.content === 'string') {
            if (message.role === 'assistant') {
              if (finalDisplayableFinalContent && finalDisplayableFinalContent.trim() !== '') {
                return (
                  <div className="prose prose-xs sm:prose-sm prose-gray max-w-none overflow-x-auto text-xs sm:text-sm">
                    <MarkdownRenderer
                      content={finalDisplayableFinalContent}
                      className=""
                    />
                  </div>
                );
              }
            } else {
              return (
                <div className={`prose prose-xs sm:prose-sm prose-gray max-w-none overflow-x-auto text-xs sm:text-sm ${message.role === 'user' ? 'user-message' : ''}`}>
                  <MarkdownRenderer
                    content={message.content}
                    className={message.role === 'user' ? 'user-message' : ''}
                  />
                </div>
              );
            }
          }

          return null;
        })()
      )}
    </div>
  );
};

export default MessageContent; 