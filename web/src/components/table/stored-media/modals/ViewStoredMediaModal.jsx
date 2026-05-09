/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Modal, Button, Tag, Typography } from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { IconCopy, IconClose } from '@douyinfe/semi-icons';
import { copy } from '../../../../helpers';

const { Text } = Typography;

const ViewStoredMediaModal = ({ visible, onCancel, data, t }) => {
  const isMobile = useIsMobile();
  const mediaType = data?.media_type;
  const url = data?.url || '';

  const handleCopyUrl = async () => {
    if (url) {
      await copy(url);
    }
  };

  if (isMobile) {
    return (
      <Modal
        visible={visible}
        onCancel={onCancel}
        footer={null}
        closeOnEsc
        style={{ maxWidth: '95vw' }}
        bodyStyle={{ padding: '12px 16px' }}
        title={
          <div className='flex items-center justify-between w-full'>
            <div className='flex items-center gap-2'>
              <span className='text-base font-medium'>{t('多模态文件')}</span>
              {mediaType && (
                <Tag color='white' shape='circle' size='small'>
                  {mediaType}
                </Tag>
              )}
            </div>
            <Button
              icon={<IconClose />}
              theme='borderless'
              type='tertiary'
              size='small'
              onClick={onCancel}
            />
          </div>
        }
      >
        <div className='flex flex-col gap-3'>
          {/* 直接显示图片 */}
          {url && mediaType === 'image' && (
            <div className='flex justify-center'>
              <img
                src={url}
                alt='stored'
                style={{ maxWidth: '100%', maxHeight: 240, borderRadius: 8 }}
              />
            </div>
          )}
          {url && mediaType === 'video' && (
            <div className='flex justify-center'>
              <video
                src={url}
                controls
                style={{ maxWidth: '100%', maxHeight: 240, borderRadius: 8 }}
              />
            </div>
          )}

          {/* 简化URL复制 */}
          {url && (
            <Button
              icon={<IconCopy />}
              theme='outline'
              size='small'
              onClick={handleCopyUrl}
              block
            >
              {t('复制链接')}
            </Button>
          )}

          {/* 说明信息 */}
          <div className='flex items-center justify-center gap-2 text-xs text-semi-color-text-2'>
            <Text type='secondary' size='small'>
              {t('ID')}: {data?.id || '-'}
            </Text>
            <Text type='secondary' size='small'>
              {data?.created_at ? timestamp2string(data.created_at) : '-'}
            </Text>
          </div>
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      title={t('多模态文件')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size='large'
    >
      <div className='flex flex-col gap-3'>
        <div className='flex flex-wrap gap-2 items-center'>
          <Text type='secondary'>
            {t('ID')}: {data?.id || '-'}
          </Text>
          <Text type='secondary'>
            {t('时间')}: {data?.created_at ? timestamp2string(data.created_at) : '-'}
          </Text>
          {mediaType && (
            <Tag color='white' shape='circle'>
              {mediaType}
            </Tag>
          )}
        </div>

        {url && (
          <div className='flex flex-col gap-2'>
            <Text type='secondary'>{t('转换后URL')}</Text>
            <Input value={url} readOnly />
          </div>
        )}

        {/* Preview */}
        {url && mediaType === 'image' && (
          <div className='flex justify-center'>
            <img
              src={url}
              alt='stored'
              style={{ maxWidth: '100%', maxHeight: 320, borderRadius: 12 }}
            />
          </div>
        )}
        {url && mediaType === 'video' && (
          <div className='flex justify-center'>
            <video
              src={url}
              controls
              style={{ maxWidth: '100%', maxHeight: 320, borderRadius: 12 }}
            />
          </div>
        )}
      </div>
    </Modal>
  );
};

export default ViewStoredMediaModal;

