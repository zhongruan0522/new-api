import React, { useMemo } from 'react';
import { Button, Empty, Tag, Typography, Dropdown } from '@douyinfe/semi-ui';
import { IconMore } from '@douyinfe/semi-icons';
import CardTable from '../../common/ui/CardTable';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { timestamp2string } from '../../../helpers';

const { Text } = Typography;

const STATUS_MAP = {
  pending: { label: '待处理', color: 'orange' },
  processing: { label: '处理中', color: 'blue' },
  completed: { label: '已完成', color: 'green' },
};

const TYPE_MAP = {
  bug: '缺陷报告',
  feature: '功能请求',
  question: '使用咨询',
  other: '其他',
};

const TicketsTable = ({
  tickets,
  loading,
  activePage,
  pageSize,
  total,
  handlePageChange,
  handlePageSizeChange,
  onViewTicket,
  onCloseTicket,
  t,
}) => {
  const columns = useMemo(() => {
    return [
      {
        title: t('标题'),
        dataIndex: 'title',
        key: 'title',
        render: (text) => {
          return (
            <Text strong style={{ cursor: 'pointer' }}>{text}</Text>
          );
        },
      },
      {
        title: t('类型'),
        dataIndex: 'type',
        key: 'type',
        width: 120,
        render: (text) => {
          return <>{TYPE_MAP[text] || text}</>;
        },
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (text) => {
          const status = STATUS_MAP[text];
          if (!status) return <Tag>{text}</Tag>;
          return <Tag color={status.color}>{t(status.label)}</Tag>;
        },
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 180,
        render: (text, record) => {
          return <>{record?.created_at ? timestamp2string(record.created_at) : '-'}</>;
        },
      },
      {
        title: t('更新时间'),
        dataIndex: 'updated_at',
        key: 'updated_at',
        width: 180,
        render: (text, record) => {
          return <>{record?.updated_at ? timestamp2string(record.updated_at) : '-'}</>;
        },
      },
      {
        title: t('更多'),
        key: 'actions',
        width: 80,
        render: (text, record) => {
          return (
            <Dropdown
              trigger='click'
              position='bottomRight'
              render={
                <Dropdown.Menu>
                  <Dropdown.Item onClick={() => onViewTicket(record)}>
                    {t('查看详情')}
                  </Dropdown.Item>
                  <Dropdown.Item onClick={() => onCloseTicket(record)}>
                    {t('关闭工单')}
                  </Dropdown.Item>
                </Dropdown.Menu>
              }
            >
              <Button
                size='small'
                type='tertiary'
                icon={<IconMore />}
                onClick={(e) => e.stopPropagation()}
              />
            </Dropdown>
          );
        },
      },
    ];
  }, [t, onViewTicket, onCloseTicket]);

  return (
    <CardTable
      columns={columns}
      dataSource={tickets}
      rowKey='id'
      loading={loading}
      style={{ width: '100%' }}
      pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: total,
        pageSizeOpts: [10, 20, 50, 100],
        showSizeChanger: true,
        onPageSizeChange: handlePageSizeChange,
        onPageChange: handlePageChange,
      }}
      hidePagination={true}
      onRow={(record) => ({
        onClick: () => onViewTicket(record),
        style: { cursor: 'pointer' },
      })}
      empty={
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={t('暂无工单')}
          style={{ padding: 30 }}
        />
      }
      className='rounded-xl overflow-hidden'
      size='middle'
    />
  );
};

export default TicketsTable;
