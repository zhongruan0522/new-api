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

import React, { Children, isValidElement } from 'react';

/**
 * 统一的页面布局组件（参考 QuantumNous/new-api default 主题的 SectionPageLayout）
 *
 * 提供一致的页面结构：标题区域（标题 + 描述 + 操作按钮）+ 内容区域
 * 使用 compound component 模式，通过子组件插槽定义各区域内容。
 *
 * 用法：
 * <SectionPageLayout>
 *   <SectionPageLayout.Title>渠道管理</SectionPageLayout.Title>
 *   <SectionPageLayout.Description>管理所有API渠道</SectionPageLayout.Description>
 *   <SectionPageLayout.Actions><Button>添加</Button></SectionPageLayout.Actions>
 *   <SectionPageLayout.Content><Table /></SectionPageLayout.Content>
 * </SectionPageLayout>
 */

// ========== 插槽占位组件（不渲染自身，仅用于传递 children） ==========

function TitleSlot(props) {
  return null;
}
TitleSlot.displayName = 'SectionPageLayout.Title';

function DescriptionSlot(props) {
  return null;
}
DescriptionSlot.displayName = 'SectionPageLayout.Description';

function ActionsSlot(props) {
  return null;
}
ActionsSlot.displayName = 'SectionPageLayout.Actions';

function ContentSlot(props) {
  return null;
}
ContentSlot.displayName = 'SectionPageLayout.Content';

// ========== 主组件 ==========

function SectionPageLayout(props) {
  let title = null;
  let description = null;
  let actions = null;
  let content = null;

  // 遍历子组件，按类型提取各插槽内容
  Children.forEach(props.children, (node) => {
    if (!isValidElement(node)) return;
    if (node.type === TitleSlot) title = node.props.children;
    else if (node.type === DescriptionSlot) description = node.props.children;
    else if (node.type === ActionsSlot) actions = node.props.children;
    else if (node.type === ContentSlot) content = node.props.children;
  });

  const hasHeader = title || description || actions;

  return (
    <div className='section-page-layout flex h-full flex-col'>
      {/* 页面标题区域 */}
      {hasHeader && (
        <div className='section-page-layout-header shrink-0 px-3 pt-3 pb-2.5 sm:px-4 sm:pt-6 sm:pb-4'>
          <div className='flex flex-wrap items-center justify-between gap-x-3 gap-y-2 sm:gap-x-4'>
            <div className='min-w-0'>
              {title && (
                <h2 className='truncate text-base font-bold tracking-tight text-semi-color-text-0 sm:text-lg'>
                  {title}
                </h2>
              )}
              {description && (
                <p className='line-clamp-2 max-sm:text-xs sm:text-sm text-semi-color-text-2'>
                  {description}
                </p>
              )}
            </div>
            {actions && (
              <div className='flex shrink-0 flex-wrap items-center gap-2 sm:gap-x-4'>
                {actions}
              </div>
            )}
          </div>
        </div>
      )}

      {/* 页面内容区域 */}
      <div className='section-page-layout-content min-h-0 flex-1 overflow-auto px-3 pb-3 sm:px-4 sm:pb-4'>
        {content}
      </div>
    </div>
  );
}

SectionPageLayout.Title = TitleSlot;
SectionPageLayout.Description = DescriptionSlot;
SectionPageLayout.Actions = ActionsSlot;
SectionPageLayout.Content = ContentSlot;

export default SectionPageLayout;
