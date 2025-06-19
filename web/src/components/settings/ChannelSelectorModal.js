import React, { useState } from 'react';
import {
  Modal,
  Transfer,
  Input,
  Space,
  Checkbox,
  Avatar,
  Highlight,
} from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';

const CHANNEL_STATUS_CONFIG = {
  1: { color: 'green', text: '启用' },
  2: { color: 'red', text: '禁用' },
  3: { color: 'amber', text: '自禁' },
  default: { color: 'grey', text: '未知' }
};

const getChannelStatusConfig = (status) => {
  return CHANNEL_STATUS_CONFIG[status] || CHANNEL_STATUS_CONFIG.default;
};

export default function ChannelSelectorModal({
  t,
  visible,
  onCancel,
  onOk,
  allChannels = [],
  selectedChannelIds = [],
  setSelectedChannelIds,
  channelEndpoints,
  updateChannelEndpoint,
}) {
  const [searchText, setSearchText] = useState('');

  const ChannelInfo = ({ item, showEndpoint = false, isSelected = false }) => {
    const channelId = item.key || item.value;
    const currentEndpoint = channelEndpoints[channelId];
    const baseUrl = item._originalData?.base_url || '';
    const status = item._originalData?.status || 0;
    const statusConfig = getChannelStatusConfig(status);

    return (
      <>
        <Avatar color={statusConfig.color} size="small">
          {statusConfig.text}
        </Avatar>
        <div className="info">
          <div className="name">
            {isSelected ? (
              item.label
            ) : (
              <Highlight sourceString={item.label} searchWords={[searchText]} />
            )}
          </div>
          <div className="email" style={showEndpoint ? { display: 'flex', alignItems: 'center', gap: '4px' } : {}}>
            <span className="text-xs text-gray-500 truncate max-w-[200px]" title={baseUrl}>
              {isSelected ? (
                baseUrl
              ) : (
                <Highlight sourceString={baseUrl} searchWords={[searchText]} />
              )}
            </span>
            {showEndpoint && (
              <Input
                size="small"
                value={currentEndpoint}
                onChange={(value) => updateChannelEndpoint(channelId, value)}
                placeholder="/api/ratio_config"
                className="flex-1 text-xs"
                style={{ fontSize: '12px' }}
              />
            )}
            {isSelected && !showEndpoint && (
              <span className="text-xs text-gray-700 font-mono bg-gray-100 px-2 py-1 rounded ml-2">
                {currentEndpoint}
              </span>
            )}
          </div>
        </div>
      </>
    );
  };

  const renderSourceItem = (item) => {
    return (
      <div className="components-transfer-source-item" key={item.key}>
        <Checkbox
          onChange={item.onChange}
          checked={item.checked}
          style={{ height: 52, alignItems: 'center' }}
        >
          <ChannelInfo item={item} showEndpoint={true} />
        </Checkbox>
      </div>
    );
  };

  const renderSelectedItem = (item) => {
    return (
      <div className="components-transfer-selected-item" key={item.key}>
        <ChannelInfo item={item} isSelected={true} />
        <IconClose style={{ cursor: 'pointer' }} onClick={item.onRemove} />
      </div>
    );
  };

  const channelFilter = (input, item) => {
    const searchLower = input.toLowerCase();
    return item.label.toLowerCase().includes(searchLower) ||
      (item._originalData?.base_url || '').toLowerCase().includes(searchLower);
  };

  return (
    <Modal
      visible={visible}
      onCancel={onCancel}
      onOk={onOk}
      title={<span className="text-lg font-semibold">{t('选择同步渠道')}</span>}
      width={1000}
    >
      <Space vertical style={{ width: '100%' }}>
        <Transfer
          style={{ width: '100%' }}
          dataSource={allChannels}
          value={selectedChannelIds}
          onChange={setSelectedChannelIds}
          renderSourceItem={renderSourceItem}
          renderSelectedItem={renderSelectedItem}
          filter={channelFilter}
          inputProps={{ placeholder: t('搜索渠道名称或地址') }}
          onSearch={setSearchText}
          emptyContent={{
            left: t('暂无渠道'),
            right: t('暂无选择'),
            search: t('无搜索结果'),
          }}
        />
      </Space>
    </Modal>
  );
} 