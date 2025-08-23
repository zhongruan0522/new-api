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

import React, { useState, useEffect, useMemo } from 'react';
import { Card, Tag, Avatar, Typography, Tooltip } from '@douyinfe/semi-ui';
import { getLobeHubIcon } from '../../../../../helpers';
import SearchActions from './SearchActions';

const { Paragraph } = Typography;

const PricingVendorIntro = ({
  filterVendor,
  models = [],
  allModels = [],
  t,
  selectedRowKeys = [],
  copyText,
  handleChange,
  handleCompositionStart,
  handleCompositionEnd,
  isMobile = false,
  searchValue = '',
  setShowFilterModal
}) => {
  const MAX_VISIBLE_AVATARS = 8;
  // 轮播动效状态（只对全部供应商生效）
  const [currentOffset, setCurrentOffset] = useState(0);

  // 获取所有供应商信息
  const vendorInfo = useMemo(() => {
    const vendors = new Map();
    let unknownCount = 0;

    (allModels.length > 0 ? allModels : models).forEach(model => {
      if (model.vendor_name) {
        if (!vendors.has(model.vendor_name)) {
          vendors.set(model.vendor_name, {
            name: model.vendor_name,
            icon: model.vendor_icon,
            description: model.vendor_description,
            count: 0
          });
        }
        vendors.get(model.vendor_name).count++;
      } else {
        unknownCount++;
      }
    });

    const vendorList = Array.from(vendors.values()).sort((a, b) => a.name.localeCompare(b.name));

    if (unknownCount > 0) {
      vendorList.push({
        name: 'unknown',
        icon: null,
        description: t('包含来自未知或未标明供应商的AI模型，这些模型可能来自小型供应商或开源项目。'),
        count: unknownCount
      });
    }

    return vendorList;
  }, [allModels, models, t]);

  // 计算当前过滤器的模型数量
  const currentModelCount = models.length;

  // 设置轮播定时器（只对全部供应商且有足够头像时生效）
  useEffect(() => {
    if (filterVendor !== 'all' || vendorInfo.length <= 3) {
      setCurrentOffset(0); // 重置偏移
      return;
    }

    const interval = setInterval(() => {
      setCurrentOffset(prev => (prev + 1) % vendorInfo.length);
    }, 2000);

    return () => clearInterval(interval);
  }, [filterVendor, vendorInfo.length]);

  // 获取供应商描述信息（从后端数据中）
  const getVendorDescription = (vendorKey) => {
    if (vendorKey === 'all') {
      return t('查看所有可用的AI模型供应商，包括众多知名供应商的模型。');
    }
    if (vendorKey === 'unknown') {
      return t('包含来自未知或未标明供应商的AI模型，这些模型可能来自小型供应商或开源项目。');
    }
    const vendor = vendorInfo.find(v => v.name === vendorKey);
    return vendor?.description || t('该供应商提供多种AI模型，适用于不同的应用场景。');
  };

  // 统一的 Tag 样式
  const tagStyle = {
    backgroundColor: 'rgba(255,255,255,0.95)',
    color: '#1f2937',
    border: '1px solid rgba(255,255,255,0.8)',
    fontWeight: '500'
  };

  // 生成封面背景样式
  const getCoverStyle = (primaryDarkerChannel) => ({
    '--palette-primary-darkerChannel': primaryDarkerChannel,
    backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  });

  // 抽象的头部卡片渲染（用于全部供应商与具体供应商）
  const renderHeaderCard = ({ title, count, description, rightContent, primaryDarkerChannel }) => (
    <Card className="!rounded-2xl shadow-sm border-0"
      cover={
        <div
          className="relative h-32"
          style={getCoverStyle(primaryDarkerChannel)}
        >
          <div className="relative z-10 h-full flex items-center justify-between p-4">
            {/* 左侧：标题与描述 */}
            <div className="flex-1 min-w-0 mr-4">
              <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 mb-2">
                <h2 className="text-lg sm:text-xl font-bold truncate" style={{ color: 'white' }}>
                  {title}
                </h2>
                <Tag style={tagStyle} shape="circle" size="small" className="self-start sm:self-center">
                  {t('共 {{count}} 个模型', { count })}
                </Tag>
              </div>
              <Paragraph
                className="text-xs sm:text-sm leading-relaxed !mb-0"
                style={{ color: 'rgba(255,255,255,0.9)' }}
                ellipsis={{
                  rows: 2,
                  expandable: true,
                  collapsible: true,
                  collapseText: t('收起'),
                  expandText: t('展开')
                }}
              >
                {description}
              </Paragraph>
            </div>

            {/* 右侧：展示区 */}
            <div className="flex-shrink-0">
              {rightContent}
            </div>
          </div>
        </div>
      }
    >
      {/* 搜索与操作区 */}
      {renderSearchActions()}
    </Card>
  );

  // 为全部供应商创建特殊的头像组合
  const renderAllVendorsAvatar = () => {
    // 重新排列数组，让当前偏移量的头像在第一位
    const rotatedVendors = vendorInfo.length > 3 ? [
      ...vendorInfo.slice(currentOffset),
      ...vendorInfo.slice(0, currentOffset)
    ] : vendorInfo;

    // 如果没有供应商，显示占位符
    if (vendorInfo.length === 0) {
      return (
        <div className="min-w-16 h-16 rounded-2xl bg-white shadow-md flex items-center justify-center px-2">
          <Avatar size="default" color="transparent">
            AI
          </Avatar>
        </div>
      );
    }

    const visible = rotatedVendors.slice(0, MAX_VISIBLE_AVATARS);
    const rest = vendorInfo.length - visible.length;

    return (
      <div className="min-w-16 h-16 rounded-2xl bg-white/90 shadow-md backdrop-blur-sm flex items-center justify-center px-2">
        <div className="flex items-center gap-2">
          {visible.map((vendor) => (
            <Tooltip key={vendor.name} content={vendor.name === 'unknown' ? t('未知供应商') : vendor.name} position="top">
              <div
                className="w-8 h-8 rounded-full flex items-center justify-center border"
                style={{
                  background: 'linear-gradient(180deg, rgba(59,130,246,0.08), rgba(59,130,246,0.02))',
                  boxShadow: '0 1px 2px rgba(0,0,0,0.04)',
                  borderColor: 'rgba(59, 130, 246, 0.25)'
                }}
              >
                {vendor.icon ? (
                  getLobeHubIcon(vendor.icon, 18)
                ) : (
                  <Avatar size="small" style={{ backgroundColor: 'rgba(59, 130, 246, 0.08)' }}>
                    {vendor.name === 'unknown' ? '?' : vendor.name.charAt(0).toUpperCase()}
                  </Avatar>
                )}
              </div>
            </Tooltip>
          ))}
          {rest > 0 && (
            <div
              className="w-8 h-8 rounded-full bg-blue-50 text-blue-600 text-xs font-medium flex items-center justify-center"
              title={`+${rest}`}
            >
              {`+${rest}`}
            </div>
          )}
        </div>
      </div>
    );
  };

  // 为具体供应商渲染单个图标
  const renderVendorAvatar = (vendor) => (
    <div className="w-16 h-16 rounded-2xl bg-white/90 shadow-md backdrop-blur-sm flex items-center justify-center">
      {vendor.icon ?
        getLobeHubIcon(vendor.icon, 40) :
        <Avatar size="large" style={{ backgroundColor: 'rgba(16, 185, 129, 0.1)' }}>
          {vendor.name === 'unknown' ? '?' : vendor.name.charAt(0).toUpperCase()}
        </Avatar>
      }
    </div>
  );

  // 渲染搜索和操作区域
  const renderSearchActions = () => (
    <SearchActions
      selectedRowKeys={selectedRowKeys}
      copyText={copyText}
      handleChange={handleChange}
      handleCompositionStart={handleCompositionStart}
      handleCompositionEnd={handleCompositionEnd}
      isMobile={isMobile}
      searchValue={searchValue}
      setShowFilterModal={setShowFilterModal}
      t={t}
    />
  );

  // 如果是全部供应商
  if (filterVendor === 'all') {
    return renderHeaderCard({
      title: t('全部供应商'),
      count: currentModelCount,
      description: getVendorDescription('all'),
      rightContent: renderAllVendorsAvatar(),
      primaryDarkerChannel: '37 99 235'
    });
  }

  // 具体供应商
  const currentVendor = vendorInfo.find(v => v.name === filterVendor);
  if (!currentVendor) {
    return null;
  }

  const vendorDisplayName = currentVendor.name === 'unknown' ? t('未知供应商') : currentVendor.name;

  return renderHeaderCard({
    title: vendorDisplayName,
    count: currentModelCount,
    description: currentVendor.description || getVendorDescription(currentVendor.name),
    rightContent: renderVendorAvatar(currentVendor),
    primaryDarkerChannel: '16 185 129'
  });
};

export default PricingVendorIntro;