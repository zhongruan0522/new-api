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
import {
  Button,
  Space,
  Tag,
  Typography,
  Modal,
  Popover
} from '@douyinfe/semi-ui';
import {
  timestamp2string,
  getLobeHubIcon,
  stringToColor
} from '../../../helpers';

const { Text } = Typography;

// Render timestamp
function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

// Generic renderer for list-style tags with limit and popover
function renderLimitedItems({ items, renderItem, maxDisplay = 3 }) {
  if (!items || items.length === 0) return '-';
  const displayItems = items.slice(0, maxDisplay);
  const remainingItems = items.slice(maxDisplay);
  return (
    <Space spacing={1} wrap>
      {displayItems.map((item, idx) => renderItem(item, idx))}
      {remainingItems.length > 0 && (
        <Popover
          content={
            <div className='p-2'>
              <Space spacing={1} wrap>
                {remainingItems.map((item, idx) => renderItem(item, idx))}
              </Space>
            </div>
          }
          position='top'
        >
          <Tag size='small' shape='circle' color='grey'>
            +{remainingItems.length}
          </Tag>
        </Popover>
      )}
    </Space>
  );
}

// Render vendor column with icon
const renderVendorTag = (vendorId, vendorMap, t) => {
  if (!vendorId || !vendorMap[vendorId]) return '-';
  const v = vendorMap[vendorId];
  return (
    <Tag
      color='white'
      shape='circle'
      prefixIcon={getLobeHubIcon(v.icon || 'Layers', 14)}
    >
      {v.name}
    </Tag>
  );
};

// Render description with ellipsis
const renderDescription = (text) => {
  return (
    <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 200 }}>
      {text || '-'}
    </Text>
  );
};

// Render tags
const renderTags = (text) => {
  if (!text) return '-';
  const tagsArr = text.split(',').filter(Boolean);
  return renderLimitedItems({
    items: tagsArr,
    renderItem: (tag, idx) => (
      <Tag key={idx} size="small" shape='circle' color={stringToColor(tag)}>
        {tag}
      </Tag>
    ),
  });
};

// Render endpoints
const renderEndpoints = (text) => {
  let arr;
  try {
    arr = JSON.parse(text);
  } catch (_) { }
  if (!Array.isArray(arr)) return text || '-';
  return renderLimitedItems({
    items: arr,
    renderItem: (ep, idx) => (
      <Tag key={idx} color="blue" size="small" shape='circle'>
        {ep}
      </Tag>
    ),
  });
};

// Render bound channels
const renderBoundChannels = (channels) => {
  if (!channels || channels.length === 0) return '-';
  return renderLimitedItems({
    items: channels,
    renderItem: (c, idx) => (
      <Tag key={idx} color="purple" size="small" shape='circle'>
        {c.name}({c.type})
      </Tag>
    ),
  });
};

// Render operations column
const renderOperations = (text, record, setEditingModel, setShowEdit, manageModel, refresh, t) => {
  return (
    <Space wrap>
      {record.status === 1 ? (
        <Button
          type='danger'
          size="small"
          onClick={() => manageModel(record.id, 'disable', record)}
        >
          {t('禁用')}
        </Button>
      ) : (
        <Button
          size="small"
          onClick={() => manageModel(record.id, 'enable', record)}
        >
          {t('启用')}
        </Button>
      )}

      <Button
        type='tertiary'
        size="small"
        onClick={() => {
          setEditingModel(record);
          setShowEdit(true);
        }}
      >
        {t('编辑')}
      </Button>

      <Button
        type='danger'
        size="small"
        onClick={() => {
          Modal.confirm({
            title: t('确定是否要删除此模型？'),
            content: t('此修改将不可逆'),
            onOk: () => {
              (async () => {
                await manageModel(record.id, 'delete', record);
                await refresh();
              })();
            },
          });
        }}
      >
        {t('删除')}
      </Button>
    </Space>
  );
};

export const getModelsColumns = ({
  t,
  manageModel,
  setEditingModel,
  setShowEdit,
  refresh,
  vendorMap,
}) => {
  return [
    {
      title: t('模型名称'),
      dataIndex: 'model_name',
    },
    {
      title: t('描述'),
      dataIndex: 'description',
      render: renderDescription,
    },
    {
      title: t('供应商'),
      dataIndex: 'vendor_id',
      render: (vendorId, record) => renderVendorTag(vendorId, vendorMap, t),
    },
    {
      title: t('标签'),
      dataIndex: 'tags',
      render: renderTags,
    },
    {
      title: t('端点'),
      dataIndex: 'endpoints',
      render: renderEndpoints,
    },
    {
      title: t('已绑定渠道'),
      dataIndex: 'bound_channels',
      render: renderBoundChannels,
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_time',
      render: (text, record, index) => {
        return <div>{renderTimestamp(text)}</div>;
      },
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      render: (text, record, index) => {
        return <div>{renderTimestamp(text)}</div>;
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      render: (text, record, index) => renderOperations(
        text,
        record,
        setEditingModel,
        setShowEdit,
        manageModel,
        refresh,
        t
      ),
    },
  ];
};