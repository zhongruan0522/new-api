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

import React, { useContext, useState } from 'react';
import {
  Modal,
  Button,
  Space,
  Typography,
  Input,
  InputGroup,
} from '@douyinfe/semi-ui';
import { IconCopy, IconKey, IconClose } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../../../context/Status';
import { copy, showSuccess } from '../../../../helpers';

const { Text, Title } = Typography;

const TokenCreatedModal = ({ visible, onClose, keys, t }) => {
  const [statusState] = useContext(StatusContext);
  const [copiedIndex, setCopiedIndex] = useState(null);

  const serverAddress =
    statusState?.status?.server_address || window.location.origin;

  const baseUrl = serverAddress.replace(/\/+$/, '');

  const handleCopy = async (text, index) => {
    const ok = await copy(text);
    if (ok) {
      setCopiedIndex(index);
      showSuccess(t('已复制到剪贴板'));
      setTimeout(() => setCopiedIndex(null), 2000);
    }
  };

  const handleCopyAllKeys = async () => {
    const allKeys = keys.map((k) => `sk-${k}`).join('\n');
    const ok = await copy(allKeys);
    if (ok) {
      showSuccess(t('已复制所有密钥到剪贴板'));
    }
  };

  const configItems = [
    { label: 'OpenAI', baseUrl: `${baseUrl}/v1` },
    { label: 'Claude', baseUrl: `${baseUrl}/v1` },
    { label: 'Gemini', baseUrl: `${baseUrl}/v1beta` },
  ];

  return (
    <Modal
      title={
        <div className='flex items-center gap-2'>
          <IconKey
            style={{ color: 'var(--semi-color-success)' }}
            size='extra-large'
          />
          <span>{t('API 密钥已创建')}</span>
        </div>
      }
      visible={visible}
      onCancel={onClose}
      closeIcon={<IconClose />}
      footer={
        <div className='flex justify-between'>
          <Button type='primary' onClick={handleCopyAllKeys}>
            {keys.length > 1 ? t('复制所有密钥') : t('复制密钥')}
          </Button>
          <Button type='tertiary' onClick={onClose}>
            {t('关闭')}
          </Button>
        </div>
      }
      width={520}
    >
      <div className='space-y-4'>
        {/* 安全提示 */}
        <div
          className='p-3 rounded-lg text-sm'
          style={{
            backgroundColor: 'var(--semi-color-warning-light-default)',
            border: '1px solid var(--semi-color-warning-light-hover)',
          }}
        >
          {t('请妥善保管以下密钥，不要泄露给他人。')}
        </div>

        {/* 密钥列表 */}
        <div className='space-y-2'>
          {keys.map((key, index) => (
            <div key={index} className='flex items-center gap-2'>
              <InputGroup size='large' style={{ flex: 1 }}>
                <Input
                  value={`sk-${key}`}
                  readOnly
                  style={{
                    fontFamily: 'monospace',
                    fontSize: '13px',
                  }}
                />
                <Button
                  icon={<IconCopy />}
                  theme={copiedIndex === index ? 'solid' : 'light'}
                  type={copiedIndex === index ? 'primary' : 'tertiary'}
                  onClick={() => handleCopy(`sk-${key}`, index)}
                />
              </InputGroup>
            </div>
          ))}
        </div>

        {/* 配置信息 */}
        <div
          className='p-4 rounded-lg space-y-3'
          style={{
            backgroundColor: 'var(--semi-color-fill-0)',
          }}
        >
          <Text strong className='text-sm'>
            {t('基础配置 URL')}
          </Text>
          <div className='space-y-2'>
            {configItems.map((item, index) => (
              <div
                key={index}
                className='flex items-center justify-between p-2 rounded'
                style={{ backgroundColor: 'var(--semi-color-bg-1)' }}
              >
                <div className='flex items-center gap-2'>
                  <Text strong className='text-xs' type='secondary'>
                    {item.label}
                  </Text>
                  <Text className='text-xs' style={{ fontFamily: 'monospace' }}>
                    {item.baseUrl}
                  </Text>
                </div>
                <Button
                  size='small'
                  icon={<IconCopy size='small' />}
                  theme='light'
                  type='tertiary'
                  onClick={() => handleCopy(item.baseUrl, `config-${index}`)}
                />
              </div>
            ))}
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default TokenCreatedModal;
