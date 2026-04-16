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
import { Tag, Space } from '@douyinfe/semi-ui';
import {
  renderModelTag,
  stringToColor,
  calculateModelPrice,
  calculateExtraPrices,
  getLobeHubIcon,
} from '../../../../../helpers';
import {
  renderLimitedItems,
} from '../../../../common/ui/RenderUtils';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';

// 统一额外计费标签，确保列表页与详情页展示一致。
const EXTRA_PRICE_LABELS = {
  cacheReadPrice: '缓存读取',
  cacheCreatePrice5m: '缓存创建',
  audioInputPrice: '音频输入',
  audioOutputPrice: '音频输出',
};

function renderQuotaType(type, t) {
  switch (type) {
    case 1:
      return (
        <Tag color='teal' shape='circle'>
          {t('按次计费')}
        </Tag>
      );
    case 0:
      return (
        <Tag color='violet' shape='circle'>
          {t('按量计费')}
        </Tag>
      );
    default:
      return t('未知');
  }
}

// Render vendor name
const renderVendor = (vendorName, vendorIcon, t) => {
  if (!vendorName) return '-';
  return (
    <Tag
      color='white'
      shape='circle'
      prefixIcon={getLobeHubIcon(vendorIcon || 'Layers', 14)}
    >
      {vendorName}
    </Tag>
  );
};

// Render tags list using RenderUtils
const renderTags = (text) => {
  if (!text) return '-';
  const tagsArr = text.split(',').filter((tag) => tag.trim());
  return renderLimitedItems({
    items: tagsArr,
    renderItem: (tag, idx) => (
      <Tag
        key={idx}
        color={stringToColor(tag.trim())}
        shape='circle'
        size='small'
      >
        {tag.trim()}
      </Tag>
    ),
    maxDisplay: 3,
  });
};

function renderSupportedEndpoints(endpoints) {
  if (!endpoints || endpoints.length === 0) {
    return null;
  }
  return (
    <Space wrap>
      {endpoints.map((endpoint, idx) => (
        <Tag key={endpoint} color={stringToColor(endpoint)} shape='circle'>
          {endpoint}
        </Tag>
      ))}
    </Space>
  );
}

export const getPricingTableColumns = ({
  t,
  selectedGroup,
  groupRatio,
  copyText,
  currency,
  tokenUnit,
  displayPrice,
}) => {
  const isMobile = useIsMobile();
  const priceDataCache = new WeakMap();

  const getPriceData = (record) => {
    let cache = priceDataCache.get(record);
    if (!cache) {
      cache = calculateModelPrice({
        record,
        selectedGroup,
        groupRatio,
        tokenUnit,
        displayPrice,
        currency,
      });
      priceDataCache.set(record, cache);
    }
    return cache;
  };

  const endpointColumn = {
    title: t('可用端点类型'),
    dataIndex: 'supported_endpoint_types',
    render: (text, record, index) => {
      return renderSupportedEndpoints(text);
    },
  };

  const modelNameColumn = {
    title: t('模型名称'),
    dataIndex: 'model_name',
    render: (text, record, index) => {
      return renderModelTag(text, {
        onClick: () => {
          copyText(text);
        },
      });
    },
    onFilter: (value, record) =>
      record.model_name.toLowerCase().includes(value.toLowerCase()),
  };

  const quotaColumn = {
    title: t('计费类型'),
    dataIndex: 'quota_type',
    render: (text, record, index) => {
      return renderQuotaType(parseInt(text), t);
    },
    sorter: (a, b) => a.quota_type - b.quota_type,
  };

  const tagsColumn = {
    title: t('标签'),
    dataIndex: 'tags',
    render: renderTags,
  };

  const vendorColumn = {
    title: t('供应商'),
    dataIndex: 'vendor_name',
    render: (text, record) => renderVendor(text, record.vendor_icon, t),
  };

  const baseColumns = [
    modelNameColumn,
    vendorColumn,
    tagsColumn,
    quotaColumn,
  ];

  const priceColumn = {
    title: t('模型价格'),
    dataIndex: 'model_price',
    ...(isMobile ? {} : { fixed: 'right' }),
    render: (text, record, index) => {
      const priceData = getPriceData(record);
      const extraPrices = calculateExtraPrices({
        record,
        selectedGroup,
        groupRatio,
        tokenUnit,
        displayPrice,
        currency,
      });

      if (priceData.isPerToken) {
        const extraPriceItems = [
          ['cacheReadPrice', extraPrices.cacheReadPrice],
          ['cacheCreatePrice5m', extraPrices.cacheCreatePrice5m],
          ['audioInputPrice', extraPrices.audioInputPrice],
          ['audioOutputPrice', extraPrices.audioOutputPrice],
        ];

        return (
          <div className='space-y-1'>
            <div className='text-gray-700'>
              {t('输入')} {priceData.inputPrice} / 1{priceData.unitLabel} tokens
            </div>
            <div className='text-gray-700'>
              {t('输出')} {priceData.completionPrice} / 1{priceData.unitLabel}{' '}
              tokens
            </div>
            <div className='space-y-1 mt-1 pt-1 border-t border-gray-100'>
              {extraPriceItems.map(([field, value]) => (
                <div key={field} className='text-gray-500 text-xs'>
                  {t(EXTRA_PRICE_LABELS[field])}
                  {field === 'cacheCreatePrice5m' ? ' (5m)' : ''}：
                  {value ? `${value} / 1${extraPrices.unitLabel} tokens` : t('不支持')}
                </div>
              ))}
            </div>
          </div>
        );
      } else {
        return (
          <div className='text-gray-700'>
            {t('模型价格')}：{priceData.price}
          </div>
        );
      }
    },
  };

  const columns = [...baseColumns];
  columns.push(endpointColumn);
  columns.push(priceColumn);
  return columns;
};
