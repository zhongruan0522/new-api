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

import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Table, Card, Skeleton, Pagination, Empty, Button, Collapsible } from '@douyinfe/semi-ui';
import { IconChevronDown, IconChevronUp } from '@douyinfe/semi-icons';
import PropTypes from 'prop-types';
import { useIsMobile } from '../../../hooks/common/useIsMobile';

/**
 * CardTable 响应式表格组件
 * 
 * 在桌面端渲染 Semi-UI 的 Table 组件，在移动端则将每一行数据渲染成 Card 形式。
 * 该组件与 Table 组件的大部分 API 保持一致，只需将原 Table 换成 CardTable 即可。
 */
const CardTable = ({ columns = [], dataSource = [], loading = false, rowKey = 'key', ...tableProps }) => {
  const isMobile = useIsMobile();
  const { t } = useTranslation();

  // Skeleton 显示控制，确保至少展示 500ms 动效
  const [showSkeleton, setShowSkeleton] = useState(loading);
  const loadingStartRef = useRef(Date.now());

  useEffect(() => {
    if (loading) {
      loadingStartRef.current = Date.now();
      setShowSkeleton(true);
    } else {
      const elapsed = Date.now() - loadingStartRef.current;
      const remaining = Math.max(0, 500 - elapsed);
      if (remaining === 0) {
        setShowSkeleton(false);
      } else {
        const timer = setTimeout(() => setShowSkeleton(false), remaining);
        return () => clearTimeout(timer);
      }
    }
  }, [loading]);

  // 解析行主键
  const getRowKey = (record, index) => {
    if (typeof rowKey === 'function') return rowKey(record);
    return record[rowKey] !== undefined ? record[rowKey] : index;
  };

  // 如果不是移动端，直接渲染原 Table
  if (!isMobile) {
    return (
      <Table
        columns={columns}
        dataSource={dataSource}
        loading={loading}
        rowKey={rowKey}
        {...tableProps}
      />
    );
  }

  // 加载中占位：根据列信息动态模拟真实布局
  if (showSkeleton) {
    const visibleCols = columns.filter((col) => {
      if (tableProps?.visibleColumns && col.key) {
        return tableProps.visibleColumns[col.key];
      }
      return true;
    });

    const renderSkeletonCard = (key) => {
      const placeholder = (
        <div className="p-2">
          {visibleCols.map((col, idx) => {
            if (!col.title) {
              return (
                <div key={idx} className="mt-2 flex justify-end">
                  <Skeleton.Title active style={{ width: 100, height: 24 }} />
                </div>
              );
            }

            return (
              <div key={idx} className="flex justify-between items-center py-1 border-b last:border-b-0 border-dashed" style={{ borderColor: 'var(--semi-color-border)' }}>
                <Skeleton.Title active style={{ width: 80, height: 14 }} />
                <Skeleton.Title
                  active
                  style={{
                    width: `${50 + (idx % 3) * 10}%`,
                    maxWidth: 180,
                    height: 14,
                  }}
                />
              </div>
            );
          })}
        </div>
      );

      return (
        <Card key={key} className="!rounded-2xl shadow-sm">
          <Skeleton loading={true} active placeholder={placeholder}></Skeleton>
        </Card>
      );
    };

    return (
      <div className="flex flex-col gap-2">
        {[1, 2, 3].map((i) => renderSkeletonCard(i))}
      </div>
    );
  }

  // 渲染移动端卡片
  const isEmpty = !showSkeleton && (!dataSource || dataSource.length === 0);

  // 移动端行卡片组件（含可折叠详情）
  const MobileRowCard = ({ record, index }) => {
    const [showDetails, setShowDetails] = useState(false);
    const rowKeyVal = getRowKey(record, index);

    const hasDetails =
      tableProps.expandedRowRender &&
      (!tableProps.rowExpandable || tableProps.rowExpandable(record));

    return (
      <Card key={rowKeyVal} className="!rounded-2xl shadow-sm">
        {columns.map((col, colIdx) => {
          // 忽略隐藏列
          if (tableProps?.visibleColumns && !tableProps.visibleColumns[col.key]) {
            return null;
          }

          const title = col.title;
          const cellContent = col.render
            ? col.render(record[col.dataIndex], record, index)
            : record[col.dataIndex];

          // 空标题列（通常为操作按钮）单独渲染
          if (!title) {
            return (
              <div key={col.key || colIdx} className="mt-2 flex justify-end">
                {cellContent}
              </div>
            );
          }

          return (
            <div
              key={col.key || colIdx}
              className="flex justify-between items-start py-1 border-b last:border-b-0 border-dashed"
              style={{ borderColor: 'var(--semi-color-border)' }}
            >
              <span className="font-medium text-gray-600 mr-2 whitespace-nowrap select-none">
                {title}
              </span>
              <div className="flex-1 break-all flex justify-end items-center gap-1">
                {cellContent !== undefined && cellContent !== null ? cellContent : '-'}
              </div>
            </div>
          );
        })}

        {hasDetails && (
          <>
            <Button
              theme='borderless'
              size='small'
              className='w-full flex justify-center mt-2'
              icon={showDetails ? <IconChevronUp /> : <IconChevronDown />}
              onClick={(e) => {
                e.stopPropagation();
                setShowDetails(!showDetails);
              }}
            >
              {showDetails ? t('收起') : t('详情')}
            </Button>
            <Collapsible isOpen={showDetails} keepDOM>
              <div className="pt-2">
                {tableProps.expandedRowRender(record, index)}
              </div>
            </Collapsible>
          </>
        )}
      </Card>
    );
  };

  if (isEmpty) {
    // 若传入 empty 属性则使用之，否则使用默认 Empty
    if (tableProps.empty) return tableProps.empty;
    return (
      <div className="flex justify-center p-4">
        <Empty description="No Data" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {dataSource.map((record, index) => (
        <MobileRowCard key={getRowKey(record, index)} record={record} index={index} />
      ))}
      {/* 分页组件 */}
      {tableProps.pagination && dataSource.length > 0 && (
        <div className="mt-2 flex justify-center">
          <Pagination {...tableProps.pagination} />
        </div>
      )}
    </div>
  );
};

CardTable.propTypes = {
  columns: PropTypes.array.isRequired,
  dataSource: PropTypes.array,
  loading: PropTypes.bool,
  rowKey: PropTypes.oneOfType([PropTypes.string, PropTypes.func]),
};

export default CardTable; 