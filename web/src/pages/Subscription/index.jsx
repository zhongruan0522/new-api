import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconGift } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import {
  API,
  renderQuota,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';

const { Text, Title } = Typography;

const submitExternalForm = (url, params) => {
  const form = document.createElement('form');
  form.action = url;
  form.method = 'POST';
  form.target = '_blank';
  Object.entries(params).forEach(([key, value]) => {
    const input = document.createElement('input');
    input.type = 'hidden';
    input.name = key;
    input.value = value;
    form.appendChild(input);
  });
  document.body.appendChild(form);
  form.submit();
  document.body.removeChild(form);
};

const unitLabelMap = {
  hour: '小时',
  day: '天',
  month: '月',
};

const formatCycle = (count, unit) => `${count}${unitLabelMap[unit] || unit}`;

const SubscriptionPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState(null);
  const [orders, setOrders] = useState([]);
  const [selectedMethod, setSelectedMethod] = useState('stripe');
  const [submitting, setSubmitting] = useState('');
  const [redemptionCode, setRedemptionCode] = useState('');

  const loadData = async () => {
    setLoading(true);
    try {
      const [summaryRes, ordersRes] = await Promise.all([
        API.get('/api/user/subscription'),
        API.get('/api/user/subscription/orders', {
          params: { p: 1, page_size: 20 },
        }),
      ]);
      if (!summaryRes.data.success) {
        showError(summaryRes.data.message);
        return;
      }
      if (!ordersRes.data.success) {
        showError(ordersRes.data.message);
        return;
      }
      setSummary(summaryRes.data.data);
      setOrders(ordersRes.data.data.items || []);
      const methods = summaryRes.data.data.pay_methods || [];
      if (methods.length > 0) {
        setSelectedMethod((current) =>
          methods.some((item) => item.type === current)
            ? current
            : methods[0].type,
        );
      }
    } catch (error) {
      showError(error.message || t('加载订阅信息失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const paymentOptions = useMemo(
    () =>
      (summary?.pay_methods || []).map((method) => ({
        label: method.name,
        value: method.type,
      })),
    [summary],
  );

  const handleBalanceAction = async (planId, action) => {
    setSubmitting(`${action}-${planId}-balance`);
    try {
      const res = await API.post('/api/user/subscription/balance', {
        plan_id: planId,
        action,
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess(t('套餐操作成功'));
      await loadData();
    } catch (error) {
      showError(error.message || t('套餐操作失败'));
    } finally {
      setSubmitting('');
    }
  };

  const handleCashAction = async (planId, action) => {
    if (!selectedMethod) {
      showError(t('请选择支付方式'));
      return;
    }
    setSubmitting(`${action}-${planId}-${selectedMethod}`);
    try {
      const payload = {
        plan_id: planId,
        action,
        payment_method: selectedMethod,
      };
      const url =
        selectedMethod === 'stripe'
          ? '/api/user/subscription/stripe/pay'
          : '/api/user/subscription/pay';
      const res = await API.post(url, payload);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      const data = res.data.data;
      if (selectedMethod === 'stripe') {
        window.open(data.pay_link, '_blank');
      } else {
        submitExternalForm(data.url, data.data || {});
      }
      showSuccess(t('已拉起支付，请在完成后刷新页面'));
    } catch (error) {
      showError(error.message || t('拉起支付失败'));
    } finally {
      setSubmitting('');
    }
  };

  const redeemSubscription = async () => {
    const key = redemptionCode.trim();
    if (!key) {
      showError(t('请输入兑换码'));
      return;
    }
    setSubmitting('redeem');
    try {
      const res = await API.post('/api/user/subscription/redeem', { key });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess(t('套餐兑换成功'));
      setRedemptionCode('');
      await loadData();
    } catch (error) {
      showError(error.message || t('套餐兑换失败'));
    } finally {
      setSubmitting('');
    }
  };

  const orderColumns = [
    { title: t('订单号'), dataIndex: 'trade_no' },
    { title: t('操作'), dataIndex: 'action' },
    { title: t('支付来源'), dataIndex: 'payment_source' },
    { title: t('支付方式'), dataIndex: 'payment_method' },
    {
      title: t('金额'),
      dataIndex: 'amount',
      render: (value) => `$${Number(value || 0).toFixed(2)}`,
    },
    { title: t('状态'), dataIndex: 'status' },
    {
      title: t('创建时间'),
      dataIndex: 'create_time',
      render: (value) => timestamp2string(value),
    },
  ];

  return (
    <div className='mt-[60px]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('订阅管理')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('购买、续订或升级套餐，并查看当前套餐额度与订单记录')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Content>
          <Spin spinning={loading}>
            <div className='grid grid-cols-1 xl:grid-cols-[1.2fr_0.8fr] gap-4'>
              <Card>
                <div className='mb-4 flex items-center justify-between'>
                  <Title heading={5}>{t('套餐订阅')}</Title>
                  {summary?.payment_mode !== 'balance' && paymentOptions.length > 0 && (
                    <Form>
                      <Form.Select
                        field='payment_method'
                        label={t('支付方式')}
                        optionList={paymentOptions}
                        value={selectedMethod}
                        onChange={setSelectedMethod}
                      />
                    </Form>
                  )}
                </div>
                <div className='max-h-[520px] overflow-y-auto pr-1 space-y-3'>
                  {(summary?.plans || []).map((plan) => {
                    const currentPlanId = summary?.active?.plan_id || summary?.active?.planId;
                    const action = currentPlanId
                      ? currentPlanId === plan.id
                        ? 'renew'
                        : 'upgrade'
                      : 'purchase';
                    return (
                      <Card key={plan.id} className='!border !border-semi-color-border'>
                        <div className='flex items-start justify-between gap-3'>
                          <div>
                            <Title heading={6}>{plan.name}</Title>
                            <Text type='secondary'>{plan.description || t('未填写描述')}</Text>
                            <div className='mt-2 flex flex-wrap gap-2'>
                              <Tag>{t('周期')}: {formatCycle(plan.duration_count, plan.duration_unit)}</Tag>
                              <Tag>{t('总额度')}: {renderQuota(plan.total_quota)}</Tag>
                              <Tag>{t('刷新额度')}: {renderQuota(plan.reset_quota)}</Tag>
                              <Tag>{t('刷新时间')}: {formatCycle(plan.reset_interval_count, plan.reset_interval_unit)}</Tag>
                            </div>
                          </div>
                          <div className='text-right'>
                            <div className='text-xl font-semibold'>${Number(plan.price || 0).toFixed(2)}</div>
                            <div className='mt-2 flex flex-col gap-2'>
                              {(summary?.payment_mode === 'balance' || summary?.payment_mode === 'both') && (
                                <Button
                                  theme='solid'
                                  loading={submitting === `${action}-${plan.id}-balance`}
                                  onClick={() => handleBalanceAction(plan.id, action)}
                                >
                                  {action === 'purchase'
                                    ? t('余额购买')
                                    : action === 'renew'
                                      ? t('余额续订')
                                      : t('余额升级')}
                                </Button>
                              )}
                              {(summary?.payment_mode === 'cash' || summary?.payment_mode === 'both') && (
                                <Button
                                  type='primary'
                                  theme='light'
                                  loading={submitting === `${action}-${plan.id}-${selectedMethod}`}
                                  onClick={() => handleCashAction(plan.id, action)}
                                >
                                  {action === 'purchase'
                                    ? t('现金购买')
                                    : action === 'renew'
                                      ? t('现金续订')
                                      : t('现金升级')}
                                </Button>
                              )}
                            </div>
                          </div>
                        </div>
                      </Card>
                    );
                  })}
                  {!summary?.plans?.length && <Empty description={t('暂无可购买套餐')} />}
                </div>
              </Card>

              <div className='space-y-4'>
                <Card>
                  <Title heading={5}>{t('套餐兑换码')}</Title>
                  <Input
                    value={redemptionCode}
                    onChange={setRedemptionCode}
                    onEnterPress={redeemSubscription}
                    placeholder={t('请输入兑换码')}
                    prefix={<IconGift />}
                    showClear
                    suffix={
                      <Button
                        theme='solid'
                        type='primary'
                        loading={submitting === 'redeem'}
                        onClick={redeemSubscription}
                      >
                        {t('兑换套餐')}
                      </Button>
                    }
                  />
                </Card>

                <Card>
                  <Title heading={5}>{t('当前套餐')}</Title>
                  {summary?.active ? (
                    <div className='space-y-2'>
                      <div className='text-lg font-semibold'>{summary.active.plan_name}</div>
                      <Text type='secondary'>
                        {t('到期时间')}: {timestamp2string(summary.active.expires_at)}
                      </Text>
                      <div>{t('总已用额度')}: {renderQuota(summary.active.used_total_quota)}</div>
                      <div>{t('窗口已用额度')}: {renderQuota(summary.active.window_used_quota)}</div>
                      <div>{t('下次刷新')}: {timestamp2string(summary.active.next_reset_at)}</div>
                      <Button type='danger' theme='light'>
                        {t('退订请提交工单')}
                      </Button>
                    </div>
                  ) : (
                    <Empty description={t('当前没有生效套餐')} />
                  )}
                </Card>

                <Card>
                  <Title heading={5}>{t('待生效套餐')}</Title>
                  {summary?.pending ? (
                    <div className='space-y-2'>
                      <div className='text-lg font-semibold'>{summary.pending.plan_name}</div>
                      <Text type='secondary'>
                        {t('开始时间')}: {timestamp2string(summary.pending.starts_at)}
                      </Text>
                    </div>
                  ) : (
                    <Empty description={t('暂无待生效套餐')} />
                  )}
                </Card>
              </div>
            </div>

            <Card className='mt-4'>
              <Title heading={5}>{t('套餐订单')}</Title>
              <Table rowKey='id' pagination={false} columns={orderColumns} dataSource={orders} />
            </Card>
          </Spin>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </div>
  );
};

export default SubscriptionPage;
