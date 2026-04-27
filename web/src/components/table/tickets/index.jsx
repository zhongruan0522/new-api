import React, { useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Tabs, TabPane, Input, Button, Space } from '@douyinfe/semi-ui';
import { IconPlus, IconSearch } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import TicketsTable from './TicketsTable';
import TicketDetailModal from './modals/TicketDetailModal';
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

// Mock messages per ticket
const generateMockMessages = (ticketId) => {
  const now = Date.now();
  const baseTime = Math.floor((now - 5 * 24 * 3600 * 1000) / 1000);

  const conversations = {
    1: [
      { type: 'message', role: 'user', username: '张三', content: '我在调用 /v1/chat/completions 接口时，频繁收到 500 Internal Server Error 错误。请求的模型是 gpt-4o，token 数量不大，大概 2000 左右。', time: baseTime },
      { type: 'message', role: 'user', username: '张三', content: '错误信息大概是：{"error":{"message":"Internal server error","type":"server_error"}}，已经持续了大概半小时了。', time: baseTime + 60 },
      { type: 'status', role: 'admin', username: '管理员', value: 'processing', time: baseTime + 300 },
      { type: 'message', role: 'admin', username: '管理员', content: '您好，感谢反馈。我们已经注意到这个问题，正在排查上游渠道的状态。请稍候。', time: baseTime + 360 },
      { type: 'message', role: 'admin', username: '管理员', content: '经排查，是上游 OpenAI 节点出现短暂故障，我们已经自动切换到备用节点。请您再试一下，应该恢复正常了。', time: baseTime + 1800 },
      { type: 'status', role: 'admin', username: '管理员', value: 'completed', time: baseTime + 1810 },
    ],
    2: [
      { type: 'message', role: 'user', username: '李四', content: '希望能增加批量导入令牌的功能，目前只能一个个手动创建，当需要创建大量令牌时效率很低。', time: baseTime + 100 },
      { type: 'message', role: 'user', username: '李四', content: '最好支持 CSV 或 JSON 格式导入，可以一次导入几百个令牌并设置相同的权限和额度。', time: baseTime + 120 },
      { type: 'status', role: 'admin', username: '管理员', value: 'processing', time: baseTime + 600 },
      { type: 'message', role: 'admin', username: '管理员', content: '感谢您的建议！这个功能已经在我们的开发计划中了，预计下个版本会支持 CSV 批量导入。', time: baseTime + 900 },
    ],
    3: [
      { type: 'message', role: 'user', username: '王五', content: '请问如何给不同的用户组设置不同的速率限制？我想给 VIP 用户更高的 RPM 限制。', time: baseTime + 200 },
      { type: 'status', role: 'admin', username: '管理员', value: 'processing', time: baseTime + 100 },
      { type: 'message', role: 'admin', username: '管理员', content: '您可以在「系统设置」->「速率限制」中，按用户分组配置不同的 RPM/TPM 限制。具体路径：控制台 -> 管理员 -> 系统设置 -> 速率限制选项卡。', time: baseTime + 200 },
      { type: 'message', role: 'user', username: '王五', content: '找到了，谢谢！', time: baseTime + 400 },
      { type: 'status', role: 'admin', username: '管理员', value: 'completed', time: baseTime + 500 },
    ],
  };

  // Return ticket-specific messages or a generic conversation
  if (conversations[ticketId]) return conversations[ticketId];

  return [
    { type: 'message', role: 'user', username: '用户', content: '提交了一个工单：' + (MOCK_TICKETS.find(t => t.id === ticketId)?.title || ''), time: baseTime + ticketId * 10 },
    { type: 'status', role: 'admin', username: '管理员', value: 'processing', time: baseTime + ticketId * 10 + 60 },
    { type: 'message', role: 'admin', username: '管理员', content: '已收到您的工单，我们正在处理中，请耐心等待。', time: baseTime + ticketId * 10 + 120 },
  ];
};

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
  const [detailVisible, setDetailVisible] = useState(false);
  const [activeTicket, setActiveTicket] = useState(null);

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

  const handleViewTicket = useCallback((ticket) => {
    setActiveTicket(ticket);
    setDetailVisible(true);
  }, []);

  const handleCloseDetail = useCallback(() => {
    setDetailVisible(false);
    setActiveTicket(null);
  }, []);

  const activeTicketMessages = useMemo(() => {
    if (!activeTicket) return [];
    return generateMockMessages(activeTicket.id);
  }, [activeTicket]);

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
    <>
      <TicketDetailModal
        visible={detailVisible}
        onCancel={handleCloseDetail}
        ticket={activeTicket}
        messages={activeTicketMessages}
        t={t}
      />

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
          onViewTicket={handleViewTicket}
          t={t}
        />
      </CardPro>
    </>
  );
};

export default TicketsPage;
