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
import { Modal, Input, Tag, Typography } from '@douyinfe/semi-ui';
import { timestamp2string } from '../../../../helpers';

const { Text } = Typography;

const ViewStoredMediaModal = ({ visible, onCancel, data, t }) => {
  const mediaType = data?.media_type;
  const url = data?.url || '';
  const basePreview = data?.base_preview || '';
  const truncated = !!data?.base_truncated;

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
          {truncated && (
            <Tag color='orange' shape='circle'>
              {t('已截断')}
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

        <div className='flex flex-col gap-2'>
          <Text type='secondary'>{t('原Base')}</Text>
          <Input.TextArea
            value={basePreview || ''}
            readOnly
            autosize={{ minRows: 4, maxRows: 10 }}
          />
        </div>
      </div>
    </Modal>
  );
};

export default ViewStoredMediaModal;

