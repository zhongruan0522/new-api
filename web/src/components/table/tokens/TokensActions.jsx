import React from 'react';
import { Button, Modal, Space } from '@douyinfe/semi-ui';
import { showError } from '../../../helpers';

const TokensActions = ({
  selectedKeys,
  setEditingToken,
  setShowEdit,
  batchCopyTokens,
  batchDeleteTokens,
  copyText,
  t,
}) => {
  // Handle copy selected tokens with options
  const handleCopySelectedTokens = () => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个令牌！'));
      return;
    }

    Modal.info({
      title: t('复制令牌'),
      icon: null,
      content: t('请选择你的复制方式'),
      footer: (
        <Space>
          <Button
            type='tertiary'
            onClick={async () => {
              let content = '';
              for (let i = 0; i < selectedKeys.length; i++) {
                content +=
                  selectedKeys[i].name + '    sk-' + selectedKeys[i].key + '\n';
              }
              await copyText(content);
              Modal.destroyAll();
            }}
          >
            {t('名称+密钥')}
          </Button>
          <Button
            onClick={async () => {
              let content = '';
              for (let i = 0; i < selectedKeys.length; i++) {
                content += 'sk-' + selectedKeys[i].key + '\n';
              }
              await copyText(content);
              Modal.destroyAll();
            }}
          >
            {t('仅密钥')}
          </Button>
        </Space>
      ),
    });
  };

  // Handle delete selected tokens with confirmation
  const handleDeleteSelectedTokens = () => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个令牌！'));
      return;
    }

    Modal.confirm({
      title: t('批量删除令牌'),
      content: (
        <div>
          {t('确定要删除所选的 {{count}} 个令牌吗？', { count: selectedKeys.length })}
        </div>
      ),
      onOk: () => batchDeleteTokens(),
    });
  };

  return (
    <div className="flex flex-wrap gap-2 w-full md:w-auto order-2 md:order-1">
      <Button
        type="primary"
        className="flex-1 md:flex-initial"
        onClick={() => {
          setEditingToken({
            id: undefined,
          });
          setShowEdit(true);
        }}
        size="small"
      >
        {t('添加令牌')}
      </Button>

      <Button
        type='tertiary'
        className="flex-1 md:flex-initial"
        onClick={handleCopySelectedTokens}
        size="small"
      >
        {t('复制所选令牌')}
      </Button>

      <Button
        type='danger'
        className="w-full md:w-auto"
        onClick={handleDeleteSelectedTokens}
        size="small"
      >
        {t('删除所选令牌')}
      </Button>
    </div>
  );
};

export default TokensActions; 