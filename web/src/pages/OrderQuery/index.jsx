import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  DatePicker,
  Empty,
  Input,
  Pagination,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import {
  API,
  getQuotaPerUnit,
  renderQuota,
  showError,
  timestamp2string,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Text } = Typography;

const orderTypeOptions = [
  { value: 'topup', label: '充值' },
  { value: 'subscription', label: '套餐' },
];

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待支付' },
  { value: 'success', label: '成功' },
  { value: 'expired', label: '已过期' },
  { value: 'failed', label: '失败' },
];

const getTimestamp = (value) => {
  if (!value) return 0;
  if (value instanceof Date) return Math.floor(value.getTime() / 1000);
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 0;
  return Math.floor(date.getTime() / 1000);
};

const formatMoney = (value) => `$${Number(value || 0).toFixed(2)}`;
const getSubscriptionCashAmount = (record) =>
  record.payment_source === 'cash' ? record.amount : 0;
const getSubscriptionBalanceAmount = (record) =>
  record.payment_source === 'balance' ? record.amount : 0;

const OrderQueryPage = () => {
  const { t } = useTranslation();
  const [orderType, setOrderType] = useState('topup');
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [dateRange, setDateRange] = useState([]);
  const [tradeNo, setTradeNo] = useState('');
  const [status, setStatus] = useState('');
  const [paymentMethod, setPaymentMethod] = useState('');

  const buildParams = (page, size) => {
    const [start, end] = Array.isArray(dateRange) ? dateRange : [];
    return {
      p: page,
      page_size: size,
      trade_no: tradeNo.trim(),
      status,
      payment_method: paymentMethod.trim(),
      start_time: getTimestamp(start),
      end_time: getTimestamp(end),
    };
  };

  const loadOrders = async (page = activePage, size = pageSize) => {
    setLoading(true);
    try {
      const endpoint =
        orderType === 'subscription'
          ? '/api/user/subscription/orders'
          : '/api/user/topup/self';
      const res = await API.get(endpoint, { params: buildParams(page, size) });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setItems(res.data.data.items || []);
      setTotal(res.data.data.total || 0);
      setActivePage(res.data.data.page || page);
    } catch (error) {
      showError(error.message || t('加载订单失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setActivePage(1);
    loadOrders(1, pageSize);
  }, [orderType]);

  const resetFilters = () => {
    setDateRange([]);
    setTradeNo('');
    setStatus('');
    setPaymentMethod('');
    setActivePage(1);
    setTimeout(() => loadOrders(1, pageSize), 0);
  };

  const renderStatus = (value) => {
    const colorMap = {
      success: 'green',
      pending: 'orange',
      expired: 'grey',
      failed: 'red',
    };
    return <Tag color={colorMap[value] || 'blue'}>{t(value || '-')}</Tag>;
  };

  const topupColumns = [
    { title: t('订单号'), dataIndex: 'trade_no', render: (text) => <Text copyable>{text}</Text> },
    { title: t('支付方式'), dataIndex: 'payment_method' },
    { title: t('充值额度'), dataIndex: 'amount', render: (value) => renderQuota(Number(value || 0) * getQuotaPerUnit()) },
    { title: t('支付金额'), dataIndex: 'money', render: formatMoney },
    { title: t('状态'), dataIndex: 'status', render: renderStatus },
    { title: t('创建时间'), dataIndex: 'create_time', render: timestamp2string },
  ];

  const subscriptionColumns = [
    { title: t('订单号'), dataIndex: 'trade_no', render: (text) => <Text copyable>{text}</Text> },
    { title: t('支付方式'), dataIndex: 'payment_method' },
    { title: t('套餐名称'), dataIndex: 'plan_name' },
    {
      title: t('套餐实际支付金额'),
      render: (_, record) => formatMoney(getSubscriptionCashAmount(record)),
    },
    {
      title: t('套餐余额扣款金额'),
      render: (_, record) => formatMoney(getSubscriptionBalanceAmount(record)),
    },
    { title: t('状态'), dataIndex: 'status', render: renderStatus },
    { title: t('创建时间'), dataIndex: 'create_time', render: timestamp2string },
  ];

  const columns = useMemo(
    () => (orderType === 'subscription' ? subscriptionColumns : topupColumns),
    [orderType],
  );

  return (
    <div className='mt-[60px]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('订单查询')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('查询充值和套餐订单记录')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Content>
          <Card>
            <div className='mb-4'>
              <Space wrap>
                {orderTypeOptions.map((option) => (
                  <Button
                    key={option.value}
                    theme={orderType === option.value ? 'solid' : 'light'}
                    type='primary'
                    onClick={() => setOrderType(option.value)}
                  >
                    {t(option.label)}
                  </Button>
                ))}
              </Space>
            </div>

            <div className='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-2 mb-4'>
              <DatePicker
                type='dateTimeRange'
                value={dateRange}
                onChange={setDateRange}
                placeholder={[t('开始时间'), t('结束时间')]}
                showClear
                style={{ width: '100%' }}
              />
              <Input
                value={tradeNo}
                onChange={setTradeNo}
                prefix={<IconSearch />}
                placeholder={t('准确订单号')}
                showClear
              />
              <Select
                value={status}
                onChange={setStatus}
                optionList={statusOptions.map((item) => ({
                  value: item.value,
                  label: t(item.label),
                }))}
                style={{ width: '100%' }}
              />
              <Input
                value={paymentMethod}
                onChange={setPaymentMethod}
                placeholder={t('支付方式')}
                showClear
              />
              <Space>
                <Button
                  theme='solid'
                  type='primary'
                  loading={loading}
                  onClick={() => {
                    setActivePage(1);
                    loadOrders(1, pageSize);
                  }}
                >
                  {t('查询')}
                </Button>
                <Button onClick={resetFilters}>{t('重置')}</Button>
              </Space>
            </div>

            <Spin spinning={loading}>
              <Table
                rowKey='id'
                columns={columns}
                dataSource={items}
                pagination={false}
                scroll={{ x: 'max-content' }}
                empty={<Empty description={t('搜索无结果')} />}
              />
              {total > 0 && (
                <div className='mt-4 flex justify-end'>
                  <Pagination
                    currentPage={activePage}
                    pageSize={pageSize}
                    total={total}
                    pageSizeOpts={[10, 20, 50, 100]}
                    showSizeChanger
                    onPageChange={(page) => {
                      setActivePage(page);
                      loadOrders(page, pageSize);
                    }}
                    onPageSizeChange={(size) => {
                      setPageSize(size);
                      setActivePage(1);
                      loadOrders(1, size);
                    }}
                  />
                </div>
              )}
            </Spin>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </div>
  );
};

export default OrderQueryPage;
