import React, { useState, useRef, useEffect } from 'react';
import { Modal, TextArea, Button, Tag, Typography, Spin } from '@douyinfe/semi-ui';
import { IconSend } from '@douyinfe/semi-icons';
import { timestamp2string } from '../../../../helpers';

const { Text } = Typography;

const STATUS_MAP = {
  pending: { label: '待处理', color: 'orange' },
  processing: { label: '处理中', color: 'blue' },
  completed: { label: '已完成', color: 'green' },
};

const TYPE_MAP = {
  bug: '缺陷报告',
  feature: '功能请求',
  question: '使用咨询',
  other: '其他',
};

const TicketDetailModal = ({
  visible,
  onCancel,
  ticket,
  messages = [],
  loading = false,
  sending = false,
  onSend,
  t,
}) => {
  const [replyText, setReplyText] = useState('');
  const messagesEndRef = useRef(null);

  // Auto-scroll to bottom when messages change or modal opens
  useEffect(() => {
    if (visible && messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [visible, messages]);

  useEffect(() => {
    if (!visible) {
      setReplyText('');
    }
  }, [visible]);

  const handleSend = async () => {
    if (!replyText.trim() || sending || typeof onSend !== 'function') return;
    const submitted = await onSend(replyText.trim());
    if (submitted) {
      setReplyText('');
    }
  };

  const statusInfo = STATUS_MAP[ticket?.status];
  const statusLabel = statusInfo ? t(statusInfo.label) : ticket?.status;

  return (
    <Modal
      visible={visible}
      onCancel={onCancel}
      footer={null}
      closeOnEsc
      style={{
        width: '50vw',
        maxWidth: '50vw',
        // Semi Modal uses inner content, we control height via bodyStyle
      }}
      bodyStyle={{
        height: '80vh',
        maxHeight: '80vh',
        display: 'flex',
        flexDirection: 'column',
        padding: 0,
        overflow: 'hidden',
      }}
      title={
        <div className='flex items-center gap-3'>
          <span>{ticket?.title || t('工单详情')}</span>
          {statusInfo && (
            <Tag color={statusInfo.color} size='small'>
              {statusLabel}
            </Tag>
          )}
          {ticket?.type && (
            <Tag color='grey' size='small'>
              {TYPE_MAP[ticket.type] || ticket.type}
            </Tag>
          )}
        </div>
      }
    >
      {/* Messages area */}
      <div
        className='flex-1 overflow-y-auto'
        style={{ padding: '16px 24px' }}
      >
        <Spin spinning={loading}>
          {messages.length === 0 && !loading ? (
            <div className='py-6 text-center'>
              <Text type='tertiary'>{t('暂无消息')}</Text>
            </div>
          ) : null}
          {messages.map((msg, index) => {
            if (msg.type === 'status') {
              return (
                <div key={index} className='flex justify-center my-3'>
                  <Text type='tertiary' size='small'>
                    {msg.username} {t('将状态更改为')}
                    <Tag
                      color={STATUS_MAP[msg.value]?.color || 'grey'}
                      size='small'
                      style={{ marginLeft: 4, marginRight: 4 }}
                    >
                      {t(STATUS_MAP[msg.value]?.label || msg.value)}
                    </Tag>
                    {timestamp2string(msg.time)}
                  </Text>
                </div>
              );
            }

            const isAdmin = msg.role === 'admin';
            return (
              <div key={index} className='mb-4'>
                <div className='flex items-baseline gap-2 mb-1'>
                  <Text
                    strong
                    size='small'
                    style={{ color: isAdmin ? 'var(--semi-color-primary)' : undefined }}
                  >
                    {msg.username}
                  </Text>
                  <Text type='quaternary' size='small'>
                    {timestamp2string(msg.time)}
                  </Text>
                </div>
                <div
                  className='rounded-lg px-3 py-2'
                  style={{
                    backgroundColor: isAdmin
                      ? 'var(--semi-color-primary-light-default)'
                      : 'var(--semi-color-fill-0)',
                    color: 'var(--semi-color-text-0)',
                    lineHeight: 1.6,
                    fontSize: 14,
                    display: 'inline-block',
                    maxWidth: '100%',
                    wordBreak: 'break-word',
                  }}
                >
                  {msg.content}
                </div>
              </div>
            );
          })}
        </Spin>
        <div ref={messagesEndRef} />
      </div>

      {/* Reply area */}
      <div
        className='flex items-end gap-2'
        style={{
          padding: '12px 24px 16px',
          borderTop: '1px solid var(--semi-color-border)',
          backgroundColor: 'var(--semi-color-bg-1)',
        }}
      >
        <TextArea
          value={replyText}
          onChange={setReplyText}
          placeholder={t('输入回复内容...')}
          autosize
          maxRows={4}
          style={{ flex: 1 }}
          disabled={loading || sending || !ticket?.id}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              void handleSend();
            }
          }}
        />
        <Button
          theme='solid'
          icon={<IconSend />}
          onClick={() => void handleSend()}
          disabled={!replyText.trim() || loading || !ticket?.id}
          loading={sending}
        >
          {t('发送')}
        </Button>
      </div>
    </Modal>
  );
};

export default TicketDetailModal;
