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

import React, { useMemo } from 'react';
import { Button, Empty, Modal, Space, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import CardTable from '../../common/ui/CardTable';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { timestamp2string } from '../../../helpers';

const { Text, Paragraph } = Typography;

const StoredMediaTable = ({
  items,
  loading,
  rowSelection,
  viewLoadingKey,
  openViewModal,
  deleteOne,
  copyText,
  t,
}) => {
  const columns = useMemo(() => {
    return [
      {
        title: t('ID'),
        dataIndex: 'id',
        key: 'id',
        width: 220,
        render: (text, record) => {
          const id = record?.id || '';
          return (
            <Tooltip content={id} position='top'>
              <Paragraph
                style={{ marginBottom: 0, maxWidth: 260 }}
                ellipsis={{ rows: 1, showTooltip: false }}
                copyable={{ content: id }}
              >
                {id}
              </Paragraph>
            </Tooltip>
          );
        },
      },
      {
        title: t('时间'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 170,
        render: (text, record) => {
          return <>{record?.created_at ? timestamp2string(record.created_at) : '-'}</>;
        },
      },
      {
        title: t('原Base'),
        dataIndex: 'base_preview',
        key: 'base_preview',
        width: 260,
        render: (text, record) => {
          if (!record?.base_preview) {
            return <Text type='tertiary'>{t('未加载')}</Text>;
          }

          const preview = record.base_preview;
          const display = preview.length > 80 ? preview.slice(0, 80) + '…' : preview;

          return (
            <div className='flex items-center gap-2'>
              <Tooltip content={preview} position='top'>
                <Text>{display}</Text>
              </Tooltip>
              {record.base_truncated && (
                <Tag color='orange' size='small'>
                  {t('已截断')}
                </Tag>
              )}
            </div>
          );
        },
      },
      {
        title: t('转换后URL'),
        dataIndex: 'url',
        key: 'url',
        width: 360,
        render: (text, record) => {
          const url = record?.url || '';
          if (!url) return <Text type='tertiary'>-</Text>;
          return (
            <Paragraph
              style={{ marginBottom: 0, maxWidth: 520 }}
              ellipsis={{ rows: 1, showTooltip: true }}
              copyable={{ content: url }}
            >
              {url}
            </Paragraph>
          );
        },
      },
      {
        title: t('操作'),
        key: 'actions',
        width: 220,
        render: (text, record) => {
          const viewing = viewLoadingKey && record?.key === viewLoadingKey;

          return (
            <Space>
              <Button
                size='small'
                type='tertiary'
                loading={viewing}
                onClick={(e) => {
                  e.stopPropagation();
                  openViewModal(record);
                }}
              >
                {t('查看')}
              </Button>
              <Button
                size='small'
                type='tertiary'
                onClick={(e) => {
                  e.stopPropagation();
                  copyText(record?.url || '');
                }}
              >
                {t('复制URL')}
              </Button>
              <Button
                size='small'
                type='danger'
                onClick={(e) => {
                  e.stopPropagation();
                  Modal.confirm({
                    title: t('删除'),
                    content: t('确定要删除该文件吗？'),
                    onOk: () => deleteOne(record),
                    centered: true,
                  });
                }}
              >
                {t('删除')}
              </Button>
            </Space>
          );
        },
      },
    ];
  }, [t, viewLoadingKey, openViewModal, deleteOne, copyText]);

  return (
    <CardTable
      columns={columns}
      dataSource={items}
      rowKey='key'
      loading={loading}
      rowSelection={rowSelection}
      scroll={{ x: 'max-content' }}
      className='rounded-xl overflow-hidden'
      size='middle'
      empty={
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={t('搜索无结果')}
          style={{ padding: 30 }}
        />
      }
      hidePagination={true}
    />
  );
};

export default StoredMediaTable;

