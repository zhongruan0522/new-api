import React from 'react';
import {
  Button,
  Tooltip,
} from '@douyinfe/semi-ui';
import {
  RefreshCw,
  Copy,
  Trash2,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

const MessageActions = ({ message, styleState, onMessageReset, onMessageCopy, onMessageDelete, isAnyMessageGenerating = false }) => {
  const { t } = useTranslation();

  const isLoading = message.status === 'loading' || message.status === 'incomplete';

  const shouldDisableActions = isAnyMessageGenerating;

  return (
    <div className="flex items-center gap-0.5">
      {!isLoading && (
        <Tooltip content={shouldDisableActions ? t('正在生成中，请稍候...') : t('重试')} position="top">
          <Button
            theme="borderless"
            type="tertiary"
            size="small"
            icon={<RefreshCw size={styleState.isMobile ? 12 : 14} />}
            onClick={() => !shouldDisableActions && onMessageReset(message)}
            disabled={shouldDisableActions}
            className={`!rounded-md ${shouldDisableActions ? '!text-gray-300 !cursor-not-allowed' : '!text-gray-400 hover:!text-blue-600 hover:!bg-blue-50'} ${styleState.isMobile ? '!w-6 !h-6' : '!w-7 !h-7'} !p-0 transition-all`}
            aria-label={t('重试')}
          />
        </Tooltip>
      )}

      {message.content && (
        <Tooltip content={t('复制')} position="top">
          <Button
            theme="borderless"
            type="tertiary"
            size="small"
            icon={<Copy size={styleState.isMobile ? 12 : 14} />}
            onClick={() => onMessageCopy(message)}
            className={`!rounded-md !text-gray-400 hover:!text-green-600 hover:!bg-green-50 ${styleState.isMobile ? '!w-6 !h-6' : '!w-7 !h-7'} !p-0 transition-all`}
            aria-label={t('复制')}
          />
        </Tooltip>
      )}

      {!isLoading && (
        <Tooltip content={shouldDisableActions ? t('正在生成中，请稍候...') : t('删除')} position="top">
          <Button
            theme="borderless"
            type="tertiary"
            size="small"
            icon={<Trash2 size={styleState.isMobile ? 12 : 14} />}
            onClick={() => !shouldDisableActions && onMessageDelete(message)}
            disabled={shouldDisableActions}
            className={`!rounded-md ${shouldDisableActions ? '!text-gray-300 !cursor-not-allowed' : '!text-gray-400 hover:!text-red-600 hover:!bg-red-50'} ${styleState.isMobile ? '!w-6 !h-6' : '!w-7 !h-7'} !p-0 transition-all`}
            aria-label={t('删除')}
          />
        </Tooltip>
      )}
    </div>
  );
};

export default MessageActions; 