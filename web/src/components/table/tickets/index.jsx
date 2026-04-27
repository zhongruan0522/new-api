import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Tabs,
  TabPane,
  Input,
  Button,
  Space,
  Modal,
  Select,
} from '@douyinfe/semi-ui';
import { IconPlus, IconSearch } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import TicketsTable from './TicketsTable';
import TicketDetailModal from './modals/TicketDetailModal';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';
import { API, isAdmin, showError, showSuccess } from '../../../helpers';

const STATUS_TABS = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待处理' },
  { key: 'processing', label: '处理中' },
  { key: 'completed', label: '已完成' },
];

const TICKET_TYPE_OPTIONS = [
  { value: 'bug', label: '缺陷报告' },
  { value: 'feature', label: '功能请求' },
  { value: 'question', label: '使用咨询' },
  { value: 'other', label: '其他' },
];

const defaultCreateForm = {
  title: '',
  type: 'question',
  content: '',
};

const TicketsPage = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const isAdminUser = useMemo(() => isAdmin(), []);

  const [activeStatus, setActiveStatus] = useState('all');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [tickets, setTickets] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [replyLoading, setReplyLoading] = useState(false);
  const [createVisible, setCreateVisible] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [activeTicket, setActiveTicket] = useState(null);
  const [createForm, setCreateForm] = useState(defaultCreateForm);

  const ticketListEndpoint = isAdminUser ? '/api/ticket/admin' : '/api/ticket';

  const fetchTickets = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get(ticketListEndpoint, {
        params: {
          p: activePage,
          page_size: pageSize,
          status: activeStatus,
          keyword: searchKeyword.trim() || undefined,
        },
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('获取工单失败'));
        return;
      }
      setTickets(data?.items || []);
      setTotal(data?.total || 0);
    } catch (error) {
      showError(error.response?.data?.message || t('获取工单失败'));
    } finally {
      setLoading(false);
    }
  }, [ticketListEndpoint, activePage, pageSize, activeStatus, searchKeyword, t]);

  const loadTicketDetail = useCallback(
    async (ticketId) => {
      setDetailLoading(true);
      try {
        const res = await API.get(`/api/ticket/${ticketId}`);
        const { success, message, data } = res.data;
        if (!success) {
          showError(message || t('获取工单详情失败'));
          return false;
        }
        setActiveTicket(data);
        return true;
      } catch (error) {
        showError(error.response?.data?.message || t('获取工单详情失败'));
        return false;
      } finally {
        setDetailLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

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

  const handleViewTicket = useCallback(
    (ticket) => {
      setActiveTicket(ticket);
      setDetailVisible(true);
      void loadTicketDetail(ticket.id);
    },
    [loadTicketDetail],
  );

  const handleCloseDetail = useCallback(() => {
    setDetailVisible(false);
    setActiveTicket(null);
  }, []);

  const handleReplyTicket = useCallback(
    async (content) => {
      if (!activeTicket?.id) {
        return false;
      }
      setReplyLoading(true);
      try {
        const res = await API.post(`/api/ticket/${activeTicket.id}/reply`, {
          content,
        });
        const { success, message } = res.data;
        if (!success) {
          showError(message || t('发送回复失败'));
          return false;
        }
        showSuccess(t('回复已发送'));
        await Promise.all([fetchTickets(), loadTicketDetail(activeTicket.id)]);
        return true;
      } catch (error) {
        showError(error.response?.data?.message || t('发送回复失败'));
        return false;
      } finally {
        setReplyLoading(false);
      }
    },
    [activeTicket, fetchTickets, loadTicketDetail, t],
  );

  const handleCloseTicket = useCallback(
    (ticket) => {
      Modal.confirm({
        title: t('确认关闭工单'),
        content: t('关闭后工单状态将变为已完成，是否继续？'),
        okText: t('关闭工单'),
        cancelText: t('取消'),
        okButtonProps: {
          type: 'danger',
        },
        onOk: async () => {
          try {
            const res = await API.post(`/api/ticket/${ticket.id}/close`);
            const { success, message } = res.data;
            if (!success) {
              showError(message || t('关闭工单失败'));
              return;
            }
            showSuccess(t('工单已关闭'));
            await fetchTickets();
            if (detailVisible && activeTicket?.id === ticket.id) {
              await loadTicketDetail(ticket.id);
            }
          } catch (error) {
            showError(error.response?.data?.message || t('关闭工单失败'));
          }
        },
      });
    },
    [activeTicket, detailVisible, fetchTickets, loadTicketDetail, t],
  );

  const handleOpenCreate = useCallback(() => {
    setCreateForm(defaultCreateForm);
    setCreateVisible(true);
  }, []);

  const handleCreateTicket = useCallback(async () => {
    if (!createForm.title.trim()) {
      showError(t('请输入工单标题'));
      return;
    }
    if (!createForm.content.trim()) {
      showError(t('请输入工单内容'));
      return;
    }

    setCreateLoading(true);
    try {
      const res = await API.post('/api/ticket', {
        title: createForm.title,
        type: createForm.type,
        content: createForm.content,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('创建工单失败'));
        return;
      }

      const shouldRefreshDirectly =
        activeStatus === 'all' &&
        activePage === 1 &&
        searchKeyword.trim() === '';

      setCreateVisible(false);
      setCreateForm(defaultCreateForm);
      setActiveStatus('all');
      setSearchKeyword('');
      setActivePage(1);
      showSuccess(t('工单创建成功'));

      if (shouldRefreshDirectly) {
        await fetchTickets();
      }

      if (data?.id) {
        setActiveTicket(data);
        setDetailVisible(true);
      }
    } catch (error) {
      showError(error.response?.data?.message || t('创建工单失败'));
    } finally {
      setCreateLoading(false);
    }
  }, [
    createForm,
    activeStatus,
    activePage,
    searchKeyword,
    fetchTickets,
    t,
  ]);

  const headerArea = (
    <div className='flex flex-col gap-3 w-full'>
      <div className='flex items-center justify-between w-full flex-wrap gap-2'>
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
          <Button theme='solid' icon={<IconPlus />} onClick={handleOpenCreate}>
            {t('新建工单')}
          </Button>
        </Space>
      </div>
    </div>
  );

  return (
    <>
      <Modal
        title={t('新建工单')}
        visible={createVisible}
        onOk={handleCreateTicket}
        onCancel={() => setCreateVisible(false)}
        confirmLoading={createLoading}
        okText={t('提交')}
        cancelText={t('取消')}
        size={isMobile ? 'full-width' : 'small'}
      >
        <div className='flex flex-col gap-3'>
          <Input
            value={createForm.title}
            onChange={(value) =>
              setCreateForm((prev) => ({ ...prev, title: value }))
            }
            placeholder={t('请输入工单标题')}
            showClear
          />
          <Select
            value={createForm.type}
            onChange={(value) =>
              setCreateForm((prev) => ({ ...prev, type: value }))
            }
          >
            {TICKET_TYPE_OPTIONS.map((option) => (
              <Select.Option key={option.value} value={option.value}>
                {t(option.label)}
              </Select.Option>
            ))}
          </Select>
          <Input.TextArea
            value={createForm.content}
            onChange={(value) =>
              setCreateForm((prev) => ({ ...prev, content: value }))
            }
            placeholder={t('请描述您遇到的问题或需求')}
            autosize={{ minRows: 6, maxRows: 10 }}
          />
        </div>
      </Modal>

      <TicketDetailModal
        visible={detailVisible}
        onCancel={handleCloseDetail}
        ticket={activeTicket}
        messages={activeTicket?.messages || []}
        loading={detailLoading}
        sending={replyLoading}
        onSend={handleReplyTicket}
        t={t}
      />

      <CardPro
        type='type1'
        descriptionArea={headerArea}
        paginationArea={createCardProPagination({
          currentPage: activePage,
          pageSize,
          total,
          onPageChange: handlePageChange,
          onPageSizeChange: handlePageSizeChange,
          isMobile,
          t,
        })}
        t={t}
      >
        <TicketsTable
          tickets={tickets}
          loading={loading}
          activePage={activePage}
          pageSize={pageSize}
          total={total}
          handlePageChange={handlePageChange}
          handlePageSizeChange={handlePageSizeChange}
          onViewTicket={handleViewTicket}
          onCloseTicket={handleCloseTicket}
          t={t}
        />
      </CardPro>
    </>
  );
};

export default TicketsPage;
