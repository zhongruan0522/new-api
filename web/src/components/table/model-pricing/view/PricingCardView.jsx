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

import React, { useState, useRef, useEffect } from 'react';
import { Card, Tag, Tooltip, Checkbox, Empty, Pagination, Button, Skeleton } from '@douyinfe/semi-ui';
import { IconHelpCircle, IconCopy } from '@douyinfe/semi-icons';
import { IllustrationNoResult, IllustrationNoResultDark } from '@douyinfe/semi-illustrations';
import { stringToColor, getModelCategories, calculateModelPrice, formatPriceInfo } from '../../../../helpers';

const PricingCardView = ({
  filteredModels,
  loading,
  rowSelection,
  pageSize,
  setPageSize,
  currentPage,
  setCurrentPage,
  selectedGroup,
  groupRatio,
  copyText,
  setModalImageUrl,
  setIsModalOpenurl,
  currency,
  tokenUnit,
  setTokenUnit,
  displayPrice,
  showRatio,
  t
}) => {
  const [showSkeleton, setShowSkeleton] = useState(loading);
  const [skeletonCount] = useState(10);
  const loadingStartRef = useRef(Date.now());

  useEffect(() => {
    if (loading) {
      loadingStartRef.current = Date.now();
      setShowSkeleton(true);
    } else {
      const elapsed = Date.now() - loadingStartRef.current;
      const remaining = Math.max(0, 1000 - elapsed);
      if (remaining === 0) {
        setShowSkeleton(false);
      } else {
        const timer = setTimeout(() => setShowSkeleton(false), remaining);
        return () => clearTimeout(timer);
      }
    }
  }, [loading]);

  // 计算当前页面要显示的数据
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedModels = filteredModels.slice(startIndex, endIndex);

  // 渲染骨架屏卡片
  const renderSkeletonCards = () => {
    const placeholder = (
      <div className="p-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: skeletonCount }).map((_, index) => (
            <Card
              key={index}
              className="!rounded-2xl border border-gray-200"
              bodyStyle={{ padding: '24px' }}
            >
              {/* 头部：图标 + 模型名称 + 操作按钮 */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center space-x-3 flex-1 min-w-0">
                  {/* 模型图标骨架 */}
                  <div className="w-12 h-12 rounded-2xl flex items-center justify-center relative shadow-sm">
                    <Skeleton.Avatar
                      size="large"
                      style={{ width: 48, height: 48, borderRadius: 16 }}
                    />
                  </div>
                  {/* 模型名称骨架 */}
                  <div className="flex-1 min-w-0">
                    <Skeleton.Title
                      style={{
                        width: `${120 + (index % 3) * 30}px`,
                        height: 20,
                        marginBottom: 0
                      }}
                    />
                  </div>
                </div>

                <div className="flex items-center space-x-2 ml-3">
                  {/* 操作按钮骨架 */}
                  <Skeleton.Button size="small" style={{ width: 32, height: 32 }} />
                  {rowSelection && (
                    <Skeleton.Button size="small" style={{ width: 16, height: 16 }} />
                  )}
                </div>
              </div>

              {/* 价格信息骨架 */}
              <div className="mb-3">
                <Skeleton.Title
                  style={{
                    width: `${180 + (index % 4) * 20}px`,
                    height: 16,
                    marginBottom: 0
                  }}
                />
              </div>

              {/* 模型描述骨架 */}
              <div className="mb-4">
                <Skeleton.Paragraph
                  rows={2}
                  style={{ marginBottom: 0 }}
                  title={false}
                />
              </div>

              {/* 标签区域骨架 */}
              <div className="flex flex-wrap gap-2 mb-4">
                {Array.from({ length: 3 + (index % 2) }).map((_, tagIndex) => (
                  <Skeleton.Button
                    key={tagIndex}
                    size="small"
                    style={{
                      width: `${40 + (tagIndex % 3) * 15}px`,
                      height: 24,
                      borderRadius: 12
                    }}
                  />
                ))}
              </div>

              {/* 倍率信息骨架（可选） */}
              {showRatio && (
                <div className="mt-4 pt-3 border-t border-gray-100">
                  <div className="flex items-center space-x-1 mb-2">
                    <Skeleton.Title
                      style={{ width: 60, height: 12, marginBottom: 0 }}
                    />
                    <Skeleton.Button size="small" style={{ width: 14, height: 14 }} />
                  </div>
                  <div className="grid grid-cols-3 gap-2">
                    {Array.from({ length: 3 }).map((_, ratioIndex) => (
                      <Skeleton.Title
                        key={ratioIndex}
                        style={{ width: '100%', height: 12, marginBottom: 0 }}
                      />
                    ))}
                  </div>
                </div>
              )}
            </Card>
          ))}
        </div>

        {/* 分页骨架 */}
        <div className="flex justify-center mt-6 pt-4 border-t pricing-pagination-divider">
          <Skeleton.Button style={{ width: 300, height: 32 }} />
        </div>
      </div>
    );

    return (
      <Skeleton loading={true} active placeholder={placeholder}></Skeleton>
    );
  };

  // 获取模型图标
  const getModelIcon = (modelName) => {
    const categories = getModelCategories(t);
    let icon = null;

    // 遍历分类，找到匹配的模型图标
    for (const [key, category] of Object.entries(categories)) {
      if (key !== 'all' && category.filter({ model_name: modelName })) {
        icon = category.icon;
        break;
      }
    }

    // 如果找到了匹配的图标，返回包装后的图标
    if (icon) {
      return (
        <div className="w-12 h-12 rounded-2xl flex items-center justify-center relative shadow-sm">
          <div className="w-8 h-8 flex items-center justify-center">
            {React.cloneElement(icon, { size: 32 })}
          </div>
        </div>
      );
    }

    // 默认图标（如果没有匹配到任何分类）
    return (
      <div className="w-12 h-12 rounded-2xl flex items-center justify-center relative shadow-sm">
        {/* 默认的螺旋图案 */}
        <svg width="24" height="24" viewBox="0 0 24 24" className="text-gray-600">
          <path
            d="M12 2C17.5 2 22 6.5 22 12S17.5 22 12 22 2 17.5 2 12 6.5 2 12 2M12 4C7.6 4 4 7.6 4 12S7.6 20 12 20 20 16.4 20 12 16.4 4 12 4M12 6C15.3 6 18 8.7 18 12S15.3 18 12 18 6 15.3 6 12 8.7 6 12 6M12 8C10.9 8 10 8.9 10 10S10.9 12 12 12 14 11.1 14 10 13.1 8 12 8Z"
            fill="currentColor"
            opacity="0.6"
          />
        </svg>
      </div>
    );
  };

  // 获取模型描述
  const getModelDescription = (modelName) => {
    // 根据模型名称返回描述，这里可以扩展
    if (modelName.includes('gpt-3.5-turbo')) {
      return t('该模型目前指向gpt-35-turbo-0125模型，综合能力强，过去使用最广泛的文本模型。');
    }
    if (modelName.includes('gpt-4')) {
      return t('更强大的GPT-4模型，具有更好的推理能力和更准确的输出。');
    }
    if (modelName.includes('claude')) {
      return t('Anthropic开发的Claude模型，以安全性和有用性著称。');
    }
    return t('高性能AI模型，适用于各种文本生成和理解任务。');
  };

  // 渲染价格信息
  const renderPriceInfo = (record) => {
    const priceData = calculateModelPrice({
      record,
      selectedGroup,
      groupRatio,
      tokenUnit,
      displayPrice,
      currency,
      precision: 4
    });
    return formatPriceInfo(priceData, t);
  };

  // 渲染标签
  const renderTags = (record) => {
    const tags = [];

    // 计费类型标签
    if (record.quota_type === 1) {
      tags.push(
        <Tag shape='circle' key="billing" color='teal' size='small'>
          {t('按次计费')}
        </Tag>
      );
    } else {
      tags.push(
        <Tag shape='circle' key="billing" color='violet' size='small'>
          {t('按量计费')}
        </Tag>
      );
    }

    // 热度标签（示例）
    if (record.model_name.includes('gpt-3.5-turbo') || record.model_name.includes('gpt-4')) {
      tags.push(
        <Tag shape='circle' key="hot" color='red' size='small'>
          {t('热')}
        </Tag>
      );
    }

    // 端点类型标签
    if (record.supported_endpoint_types && record.supported_endpoint_types.length > 0) {
      record.supported_endpoint_types.slice(0, 2).forEach((endpoint, index) => {
        tags.push(
          <Tag shape='circle' key={`endpoint-${index}`} color={stringToColor(endpoint)} size='small'>
            {endpoint}
          </Tag>
        );
      });
    }

    // 上下文长度标签（示例）
    if (record.model_name.includes('16k')) {
      tags.push(<Tag shape='circle' key="context" color='blue' size='small'>16K</Tag>);
    } else if (record.model_name.includes('32k')) {
      tags.push(<Tag shape='circle' key="context" color='blue' size='small'>32K</Tag>);
    } else {
      tags.push(<Tag shape='circle' key="context" color='blue' size='small'>4K</Tag>);
    }

    return tags;
  };

  // 显示骨架屏
  if (showSkeleton) {
    return renderSkeletonCards();
  }

  if (!filteredModels || filteredModels.length === 0) {
    return (
      <div className="flex justify-center items-center py-20">
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={<IllustrationNoResultDark style={{ width: 150, height: 150 }} />}
          description={t('搜索无结果')}
        />
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {paginatedModels.map((model, index) => {
          const isSelected = rowSelection?.selectedRowKeys?.includes(model.id);

          return (
            <Card
              key={model.id || index}
              className={`!rounded-2xl transition-all duration-200 hover:shadow-lg border ${isSelected
                ? 'border-blue-500 bg-blue-50'
                : 'border-gray-200 hover:border-gray-300'
                }`}
              bodyStyle={{ padding: '24px' }}
            >
              {/* 头部：图标 + 模型名称 + 操作按钮 */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center space-x-3 flex-1 min-w-0">
                  {getModelIcon(model.model_name)}
                  <div className="flex-1 min-w-0">
                    <h3 className="text-xl font-bold text-gray-900 truncate">
                      {model.model_name}
                    </h3>
                  </div>
                </div>

                <div className="flex items-center space-x-2 ml-3">
                  {/* 复制按钮 */}
                  <Button
                    size="small"
                    type="tertiary"
                    icon={<IconCopy />}
                    onClick={() => copyText(model.model_name)}
                  />

                  {/* 选择框 */}
                  {rowSelection && (
                    <Checkbox
                      checked={isSelected}
                      onChange={(checked) => {
                        if (checked) {
                          rowSelection.onSelect(model, true);
                        } else {
                          rowSelection.onSelect(model, false);
                        }
                      }}
                    />
                  )}
                </div>
              </div>

              {/* 价格信息 */}
              <div className="mb-3">
                <div className="text-gray-700 text-base font-medium">
                  {renderPriceInfo(model)}
                </div>
              </div>

              {/* 模型描述 */}
              <div className="mb-4">
                <p className="text-gray-500 text-sm leading-relaxed">
                  {getModelDescription(model.model_name)}
                </p>
              </div>

              {/* 标签区域 */}
              <div className="flex flex-wrap gap-2">
                {renderTags(model)}
              </div>

              {/* 倍率信息（可选） */}
              {showRatio && (
                <div className="mt-4 pt-3 border-t border-gray-100">
                  <div className="flex items-center space-x-1 mb-2">
                    <span className="text-xs font-medium text-gray-700">{t('倍率信息')}</span>
                    <Tooltip content={t('倍率是为了方便换算不同价格的模型')}>
                      <IconHelpCircle
                        className="text-blue-500 cursor-pointer"
                        size="small"
                        onClick={() => {
                          setModalImageUrl('/ratio.png');
                          setIsModalOpenurl(true);
                        }}
                      />
                    </Tooltip>
                  </div>
                  <div className="grid grid-cols-3 gap-2 text-xs text-gray-600">
                    <div>
                      {t('模型')}: {model.quota_type === 0 ? model.model_ratio : t('无')}
                    </div>
                    <div>
                      {t('补全')}: {model.quota_type === 0 ? parseFloat(model.completion_ratio.toFixed(3)) : t('无')}
                    </div>
                    <div>
                      {t('分组')}: {groupRatio[selectedGroup]}
                    </div>
                  </div>
                </div>
              )}
            </Card>
          );
        })}
      </div>

      {/* 分页 */}
      {filteredModels.length > 0 && (
        <div className="flex justify-center mt-6 pt-4 border-t pricing-pagination-divider">
          <Pagination
            currentPage={currentPage}
            pageSize={pageSize}
            total={filteredModels.length}
            showSizeChanger={true}
            pageSizeOptions={[10, 20, 50, 100]}
            onPageChange={(page) => setCurrentPage(page)}
            onPageSizeChange={(size) => {
              setPageSize(size);
              setCurrentPage(1);
            }}
          />
        </div>
      )}
    </div>
  );
};

export default PricingCardView; 