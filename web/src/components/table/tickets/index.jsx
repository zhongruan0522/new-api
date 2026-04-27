import React, { useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Tabs, TabPane, Input, Button, Space } from '@douyinfe/semi-ui';
import { IconPlus, IconSearch } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import TicketsTable from './TicketsTable';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

// Mock data for UI demonstration
const generateMockTickets = () => {
  const types = ['bug', 'feature', 'question', 'other'];
  const statuses = ['pending', 'processing', 'completed'];
  const titles = [
    'API调用返回500错误',
    '希望增加批量导入功能',
    '如何设置速率限制？',
    '账单显示异常',
    '模型响应速度较慢',
    '请求新增Claude 3.5模型支持',
    'Token余额未到账',
    '如何查看API使用统计？',
    '渠道连接超时问题',
    '希望支持流式输出',
    '兑换码无法使用',
    'Webhook回调失败',
  ];

  return titles.map((title, index) => {
    const now = Date.now();
    const createdOffset = Math.floor(Math.random() * 7 * 24 * 3600) * 1000;
    const updatedOffset = Math.floor(Math.random() * 2 * 24 * 3600) * 1000;
    return {
      id: index + 1,
      title,
      type: types[index % types.length],
      status: statuses[index % statuses.length],
      created_at: Math.floor((now - createdOffset) / 1000),
      updated_at: Math.floor((now - updatedOffset) / 1000),
    };
  });
};

const MOCK_TICKETS = generateMockTickets();

const STATUS_TABS = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待处理' },
  { key: 'processing', label: '处理中' },
  { key: 'completed', label: '已完成' },
];

const TicketsPage = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [activeStatus, setActiveStatus] = useState('all');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const filteredTickets = useMemo(() => {
    let result = MOCK_TICKETS;

    if (activeStatus !== 'all') {
      result = result.filter((ticket) => ticket.status === activeStatus);
    }

    if (searchKeyword.trim()) {
      const keyword = searchKeyword.trim().toLowerCase();
      result = result.filter((ticket) =>
        ticket.title.toLowerCase().includes(keyword),
      );
    }

    // Sort by updated_at descending
    result = [...result].sort((a, b) => b.updated_at - a.updated_at);

    return result;
  }, [activeStatus, searchKeyword]);

  const pagedTickets = useMemo(() => {
    const start = (activePage - 1) * pageSize;
    return filteredTickets.slice(start, start + pageSize);
  }, [filteredTickets, activePage, pageSize]);

  const handlePageChange = useCallback((page) => {
    setActivePage(page);
  }, []);

  const handlePageSizeChange = useCallback((size) => {
    setPageSize(size);
    setActivePage(1);
  }, []);

  const handleTabChange = useCallback((key) => {
    setActiveStatus(key);
    setActivePage(1);
  }, []);

  const handleSearch = useCallback((value) => {
    setSearchKeyword(value);
    setActivePage(1);
  }, []);

  // Header area: tabs on left, search + button on right
  const headerArea = (
    <div className='flex flex-col gap-3 w-full'>
      {/* Top row: tabs left, actions right */}
      <div className='flex items-center justify-between w-full flex-wrap gap-2'>
        {/* Left: Status tabs */}
        <Tabs
          activeKey={activeStatus}
          type='button'
          onChange={handleTabChange}
          size='small'
        >
          {STATUS_TABS.map((tab) => (
            <TabPane key={tab.key} itemKey={tab.key} tab={t(tab.label)} />
          ))}
        </Tabs>

        {/* Right: Search + New ticket button */}
        <Space>
          <Input
            prefix={<IconSearch />}
            placeholder={t('搜索工单...')}
            value={searchKeyword}
            onChange={handleSearch}
            showClear
            onClear={() => handleSearch('')}
            style={{ width: 200 }}
          />
          <Button theme='solid' icon={<IconPlus />}>
            {t('新建工单')}
          </Button>
        </Space>
      </div>
    </div>
  );

  return (
    <CardPro
      type='type1'
      descriptionArea={headerArea}
      paginationArea={createCardProPagination({
        currentPage: activePage,
        pageSize,
        total: filteredTickets.length,
        onPageChange: handlePageChange,
        onPageSizeChange: handlePageSizeChange,
        isMobile,
        t,
      })}
      t={t}
    >
      <TicketsTable
        tickets={pagedTickets}
        loading={false}
        activePage={activePage}
        pageSize={pageSize}
        total={filteredTickets.length}
        handlePageChange={handlePageChange}
        handlePageSizeChange={handlePageSizeChange}
        t={t}
      />
    </CardPro>
  );
};

export default TicketsPage;
