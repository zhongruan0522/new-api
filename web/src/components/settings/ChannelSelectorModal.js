import React from 'react';
import {
  Modal,
  Transfer,
  Input,
  Space,
  Checkbox,
} from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';

/**
 * ChannelSelectorModal
 * 负责选择同步渠道、测试与批量测试等 UI，纯展示组件。
 * 业务状态与动作通过 props 注入，保持可复用与可测试。
 */
export default function ChannelSelectorModal({
  t,
  visible,
  onCancel,
  onOk,
  // 渠道选择
  allChannels = [],
  selectedChannelIds = [],
  setSelectedChannelIds,
  // 渠道端点
  channelEndpoints,
  updateChannelEndpoint,
}) {
  // Transfer 自定义渲染
  const renderSourceItem = (item) => {
    const channelId = item.key || item.value;
    const currentEndpoint = channelEndpoints[channelId];
    const baseUrl = item._originalData?.base_url || '';

    return (
      <div key={item.key} style={{ padding: 8 }}>
        <div className="flex flex-col gap-2 w-full">
          <div className="flex items-center w-full">
            <Checkbox checked={item.checked} onChange={item.onChange}>
              <span className="font-medium">{item.label}</span>
            </Checkbox>
          </div>
          <div className="flex items-center gap-1 ml-4">
            <span className="text-xs text-gray-500 truncate max-w-[120px]" title={baseUrl}>
              {baseUrl}
            </span>
            <Input
              size="small"
              value={currentEndpoint}
              onChange={(value) => updateChannelEndpoint(channelId, value)}
              placeholder="/api/ratio_config"
              className="flex-1 text-xs"
              style={{ fontSize: '12px' }}
            />
          </div>
        </div>
      </div>
    );
  };

  const renderSelectedItem = (item) => {
    const channelId = item.key || item.value;
    const currentEndpoint = channelEndpoints[channelId];
    const baseUrl = item._originalData?.base_url || '';

    return (
      <div key={item.key} style={{ padding: 6 }}>
        <div className="flex flex-col gap-2 w-full">
          <div className="flex items-center w-full">
            <span className="font-medium">{item.label}</span>
            <IconClose style={{ cursor: 'pointer' }} onClick={item.onRemove} className="ml-auto" />
          </div>
          <div className="flex items-center gap-1 ml-4">
            <span
              className="text-xs text-gray-500 truncate max-w-[120px]"
              title={baseUrl}
            >
              {baseUrl}
            </span>
            <span className="text-xs text-gray-700 font-mono bg-gray-100 px-2 py-1 rounded flex-1">
              {currentEndpoint}
            </span>
          </div>
        </div>
      </div>
    );
  };

  const channelFilter = (input, item) => item.label.toLowerCase().includes(input.toLowerCase());

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