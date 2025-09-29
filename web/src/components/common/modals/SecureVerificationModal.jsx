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
import { useTranslation } from 'react-i18next';
import { Modal, Button, Input, Typography, Tabs, TabPane, Card } from '@douyinfe/semi-ui';

/**
 * 通用安全验证模态框组件
 * 配合 useSecureVerification Hook 使用
 * @param {Object} props
 * @param {boolean} props.visible - 是否显示模态框
 * @param {Object} props.verificationMethods - 可用的验证方式
 * @param {Object} props.verificationState - 当前验证状态
 * @param {Function} props.onVerify - 验证回调
 * @param {Function} props.onCancel - 取消回调
 * @param {Function} props.onCodeChange - 验证码变化回调
 * @param {Function} props.onMethodSwitch - 验证方式切换回调
 * @param {string} props.title - 模态框标题
 * @param {string} props.description - 验证描述文本
 */
const SecureVerificationModal = ({
  visible,
  verificationMethods,
  verificationState,
  onVerify,
  onCancel,
  onCodeChange,
  onMethodSwitch,
  title,
  description,
}) => {
  const { t } = useTranslation();

  const { has2FA, hasPasskey, passkeySupported } = verificationMethods;
  const { method, loading, code } = verificationState;

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && code.trim() && !loading && method === '2fa') {
      onVerify(method, code);
    }
  };

  // 如果用户没有启用任何验证方式
  if (visible && !has2FA && !hasPasskey) {
    return (
      <Modal
        title={title || t('安全验证')}
        visible={visible}
        onCancel={onCancel}
        footer={
          <Button onClick={onCancel}>{t('确定')}</Button>
        }
        width={500}
        style={{ maxWidth: '90vw' }}
      >
        <div className='text-center py-6'>
          <div className='mb-4'>
            <svg
              className='w-16 h-16 text-yellow-500 mx-auto mb-4'
              fill='currentColor'
              viewBox='0 0 20 20'
            >
              <path
                fillRule='evenodd'
                d='M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z'
                clipRule='evenodd'
              />
            </svg>
          </div>
          <Typography.Title heading={4} className='mb-2'>
            {t('需要安全验证')}
          </Typography.Title>
          <Typography.Text type='tertiary'>
            {t('您需要先启用两步验证或 Passkey 才能查看敏感信息。')}
          </Typography.Text>
          <br />
          <Typography.Text type='tertiary'>
            {t('请前往个人设置 → 安全设置进行配置。')}
          </Typography.Text>
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      title={
        <div className='flex items-center'>
          <div className='w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center mr-3'>
            <svg
              className='w-4 h-4 text-blue-600 dark:text-blue-400'
              fill='currentColor'
              viewBox='0 0 20 20'
            >
              <path
                fillRule='evenodd'
                d='M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z'
                clipRule='evenodd'
              />
            </svg>
          </div>
          {title || t('安全验证')}
        </div>
      }
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={600}
      style={{ maxWidth: '90vw' }}
    >
      <div className='space-y-6'>
        {/* 安全提示 */}
        <div className='bg-blue-50 dark:bg-blue-900 rounded-lg p-4'>
          <div className='flex items-start'>
            <svg
              className='w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5 mr-3 flex-shrink-0'
              fill='currentColor'
              viewBox='0 0 20 20'
            >
              <path
                fillRule='evenodd'
                d='M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z'
                clipRule='evenodd'
              />
            </svg>
            <div>
              <Typography.Text
                strong
                className='text-blue-800 dark:text-blue-200'
              >
                {t('安全验证')}
              </Typography.Text>
              <Typography.Text className='block text-blue-700 dark:text-blue-300 text-sm mt-1'>
                {description || t('为了保护账户安全，请选择一种方式进行验证。')}
              </Typography.Text>
            </div>
          </div>
        </div>

        {/* 验证方式选择 */}
        <Tabs activeKey={method} onChange={onMethodSwitch} type='card'>
          {has2FA && (
            <TabPane
              tab={
                <div className='flex items-center space-x-2'>
                  <svg className='w-4 h-4' fill='currentColor' viewBox='0 0 20 20'>
                    <path d='M10 12a2 2 0 100-4 2 2 0 000 4z' />
                    <path
                      fillRule='evenodd'
                      d='M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z'
                      clipRule='evenodd'
                    />
                  </svg>
                  <span>{t('两步验证')}</span>
                </div>
              }
              itemKey='2fa'
            >
              <Card className='border-0 shadow-none bg-transparent'>
                <div className='space-y-4'>
                  <div>
                    <Typography.Text strong className='block mb-2'>
                      {t('验证码')}
                    </Typography.Text>
                    <Input
                      placeholder={t('请输入认证器验证码或备用码')}
                      value={code}
                      onChange={onCodeChange}
                      size='large'
                      maxLength={8}
                      onKeyDown={handleKeyDown}
                      autoFocus={method === '2fa'}
                    />
                    <Typography.Text type='tertiary' size='small' className='mt-2 block'>
                      {t('支持6位TOTP验证码或8位备用码')}
                    </Typography.Text>
                  </div>
                  <div className='flex justify-end space-x-3'>
                    <Button onClick={onCancel}>{t('取消')}</Button>
                    <Button
                      type='primary'
                      loading={loading}
                      disabled={!code.trim() || loading}
                      onClick={() => onVerify(method, code)}
                    >
                      {t('验证')}
                    </Button>
                  </div>
                </div>
              </Card>
            </TabPane>
          )}

          {hasPasskey && passkeySupported && (
            <TabPane
              tab={
                <div className='flex items-center space-x-2'>
                  <svg className='w-4 h-4' fill='currentColor' viewBox='0 0 20 20'>
                    <path
                      fillRule='evenodd'
                      d='M18 8a6 6 0 01-7.743 5.743L10 14l-1 1-1 1H6v2H2v-4l4.257-4.257A6 6 0 1118 8zm-6-4a1 1 0 100 2 2 2 0 012 2 1 1 0 102 0 4 4 0 00-4-4z'
                      clipRule='evenodd'
                    />
                  </svg>
                  <span>{t('Passkey')}</span>
                </div>
              }
              itemKey='passkey'
            >
              <Card className='border-0 shadow-none bg-transparent'>
                <div className='space-y-4'>
                  <div className='text-center py-4'>
                    <div className='mb-4'>
                      <svg
                        className='w-16 h-16 text-blue-500 mx-auto'
                        fill='currentColor'
                        viewBox='0 0 20 20'
                      >
                        <path
                          fillRule='evenodd'
                          d='M18 8a6 6 0 01-7.743 5.743L10 14l-1 1-1 1H6v2H2v-4l4.257-4.257A6 6 0 1118 8zm-6-4a1 1 0 100 2 2 2 0 012 2 1 1 0 102 0 4 4 0 00-4-4z'
                          clipRule='evenodd'
                        />
                      </svg>
                    </div>
                    <Typography.Text strong className='block mb-2'>
                      {t('使用 Passkey 验证')}
                    </Typography.Text>
                    <Typography.Text type='tertiary' className='block mb-4'>
                      {t('点击下方按钮，使用您的生物特征或安全密钥进行验证')}
                    </Typography.Text>
                  </div>
                  <div className='flex justify-end space-x-3'>
                    <Button onClick={onCancel}>{t('取消')}</Button>
                    <Button
                      type='primary'
                      loading={loading}
                      disabled={loading}
                      onClick={() => onVerify(method)}
                    >
                      {loading ? t('验证中...') : t('验证 Passkey')}
                    </Button>
                  </div>
                </div>
              </Card>
            </TabPane>
          )}
        </Tabs>
      </div>
    </Modal>
  );
};

export default SecureVerificationModal;