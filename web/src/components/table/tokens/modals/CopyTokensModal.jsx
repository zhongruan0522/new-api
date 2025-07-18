import React from 'react';
import { Modal, Button, Space } from '@douyinfe/semi-ui';

const CopyTokensModal = ({ visible, onCancel, selectedKeys, copyText, t }) => {
  // Handle copy with name and key format
  const handleCopyWithName = async () => {
    let content = '';
    for (let i = 0; i < selectedKeys.length; i++) {
      content += selectedKeys[i].name + '    sk-' + selectedKeys[i].key + '\n';
    }
    await copyText(content);
    onCancel();
  };

  // Handle copy with key only format
  const handleCopyKeyOnly = async () => {
    let content = '';
    for (let i = 0; i < selectedKeys.length; i++) {
      content += 'sk-' + selectedKeys[i].key + '\n';
    }
    await copyText(content);
    onCancel();
  };

  return (
    <Modal
      title={t('复制令牌')}
      icon={null}
      visible={visible}
      onCancel={onCancel}
      footer={
        <Space>
          <Button
            type='tertiary'
            onClick={handleCopyWithName}
          >
            {t('名称+密钥')}
          </Button>
          <Button
            onClick={handleCopyKeyOnly}
          >
            {t('仅密钥')}
          </Button>
        </Space>
      }
    >
      {t('请选择你的复制方式')}
    </Modal>
  );
};

export default CopyTokensModal; 