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
import { Card, Skeleton } from '@douyinfe/semi-ui';

const PricingVendorIntroSkeleton = ({
  isAllVendors = false
}) => {
  // 统一的封面样式函数
  const getCoverStyle = (primaryDarkerChannel) => ({
    '--palette-primary-darkerChannel': primaryDarkerChannel,
    backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  });

  // 快速生成骨架矩形
  const rect = (style = {}, key) => (
    <div key={key} className="animate-pulse" style={style} />
  );

  const placeholder = (
    <Card className="!rounded-2xl shadow-sm border-0"
      cover={
        <div
          className="relative h-32"
          style={getCoverStyle(isAllVendors ? '37 99 235' : '16 185 129')}
        >
          <div className="relative z-10 h-full flex items-center justify-between p-4">
            {/* 左侧：标题和描述骨架 */}
            <div className="flex-1 min-w-0 mr-4">
              <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 mb-2">
                {rect({
                  width: isAllVendors ? 120 : 100,
                  height: 24,
                  backgroundColor: 'rgba(255, 255, 255, 0.25)',
                  borderRadius: 8,
                  backdropFilter: 'blur(4px)'
                })}
                {rect({
                  width: 80,
                  height: 20,
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  borderRadius: 9999,
                  backdropFilter: 'blur(4px)',
                  border: '1px solid rgba(255,255,255,0.3)'
                })}
              </div>
              <div className="space-y-2">
                {rect({
                  width: '100%',
                  height: 14,
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  borderRadius: 4,
                  backdropFilter: 'blur(4px)'
                })}
                {rect({
                  width: '75%',
                  height: 14,
                  backgroundColor: 'rgba(255, 255, 255, 0.15)',
                  borderRadius: 4,
                  backdropFilter: 'blur(4px)'
                })}
              </div>
            </div>

            {/* 右侧：供应商图标骨架 */}
            <div className="flex-shrink-0 min-w-16 h-16 rounded-2xl bg-white/90 shadow-md backdrop-blur-sm flex items-center justify-center px-2">
              {isAllVendors ? (
                <div className="flex items-center gap-2">
                  {Array.from({ length: 4 }).map((_, index) => (
                    rect({
                      width: 32,
                      height: 32,
                      backgroundColor: 'rgba(59, 130, 246, 0.1)',
                      borderRadius: 9999,
                      border: '1px solid rgba(59, 130, 246, 0.2)'
                    }, index)
                  ))}
                </div>
              ) : (
                rect({
                  width: 40,
                  height: 40,
                  backgroundColor: 'rgba(16, 185, 129, 0.1)',
                  borderRadius: 12,
                  border: '1px solid rgba(16, 185, 129, 0.2)'
                })
              )}
            </div>
          </div>
        </div>
      }
    >
      {/* 搜索和操作区域骨架 */}
      <div className="flex items-center gap-4 w-full">
        {/* 搜索框骨架 */}
        <div className="flex-1">
          {rect({
            width: '100%',
            height: 32,
            backgroundColor: 'rgba(156, 163, 175, 0.1)',
            borderRadius: 8,
            border: '1px solid rgba(156, 163, 175, 0.2)'
          })}
        </div>

        {/* 操作按钮骨架 */}
        {rect({
          width: 80,
          height: 32,
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          borderRadius: 8,
          border: '1px solid rgba(59, 130, 246, 0.2)'
        })}
      </div>
    </Card>
  );

  return (
    <Skeleton loading={true} active placeholder={placeholder}></Skeleton>
  );
};

export default PricingVendorIntroSkeleton;