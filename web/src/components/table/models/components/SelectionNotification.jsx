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

import React, { useEffect } from 'react';
import { Notification, Button, Space } from '@douyinfe/semi-ui';

// 固定通知 ID，保持同一个实例即可避免闪烁
const NOTICE_ID = 'models-batch-actions';

/**
 * SelectionNotification 选择通知组件
 * 1. 当 selectedKeys.length > 0 时，使用固定 id 创建/更新通知
 * 2. 当 selectedKeys 清空时关闭通知
 */
const SelectionNotification = ({ selectedKeys = [], t, onDelete }) => {
  // 根据选中数量决定显示/隐藏或更新通知
  useEffect(() => {
    const selectedCount = selectedKeys.length;

    if (selectedCount > 0) {
      const content = (
        <Space>
          <span>{t('已选择 {{count}} 个模型', { count: selectedCount })}</span>
          <Button
            size="small"
            type="danger"
            theme="solid"
            onClick={onDelete}
          >
            {t('删除所选模型')}
          </Button>
        </Space>
      );

      // 使用相同 id 更新通知（若已存在则就地更新，不存在则创建）
      Notification.info({
        id: NOTICE_ID,
        title: t('批量操作'),
        content,
        duration: 0, // 不自动关闭
        position: 'bottom',
        showClose: false,
      });
    } else {
      // 取消全部勾选时关闭通知
      Notification.close(NOTICE_ID);
    }
  }, [selectedKeys, t, onDelete]);

  // 卸载时确保关闭通知
  useEffect(() => {
    return () => {
      Notification.close(NOTICE_ID);
    };
  }, []);

  return null; // 该组件不渲染可见内容
};

export default SelectionNotification;
