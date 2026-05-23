import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Modal,
  Pagination,
  Space,
  Spin,
  Switch,
  Table,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import { API, renderQuota, showError, showSuccess, timestamp2string } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const initPlan = {
  id: 0,
  name: '',
  description: '',
  duration_count: 1,
  duration_unit: 'month',
  price: 0,
  total_quota: 0,
  reset_quota: 0,
  reset_interval_count: 1,
  reset_interval_unit: 'day',
  enabled: true,
};

const unitOptions = [
  { label: '小时', value: 'hour' },
  { label: '天', value: 'day' },
  { label: '月', value: 'month' },
];

const SubscriptionPlanPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [plans, setPlans] = useState([]);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingPlan, setEditingPlan] = useState(initPlan);
  const formApiRef = useRef(null);

  const loadPlans = async (page = activePage, size = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get('/api/subscription_plan/', {
        params: { p: page, page_size: size },
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setPlans(res.data.data.items || []);
      setTotal(res.data.data.total || 0);
      setActivePage(res.data.data.page || page);
    } catch (error) {
      showError(error.message || t('加载套餐失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPlans(1, pageSize);
  }, []);

  const openCreate = () => {
    setEditingPlan(initPlan);
    setModalVisible(true);
    setTimeout(() => formApiRef.current?.setValues(initPlan), 0);
  };

  const openEdit = (record) => {
    setEditingPlan(record);
    setModalVisible(true);
    setTimeout(() => formApiRef.current?.setValues(record), 0);
  };

  const savePlan = async () => {
    const values = formApiRef.current?.getValues();
    if (!values) return;
    try {
      const url = values.id ? '/api/subscription_plan/' : '/api/subscription_plan/';
      const method = values.id ? API.put : API.post;
      const res = await method(url, values);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess(t(values.id ? '套餐更新成功' : '套餐创建成功'));
      setModalVisible(false);
      loadPlans(values.id ? activePage : 1, pageSize);
    } catch (error) {
      showError(error.message || t('保存套餐失败'));
    }
  };

  const togglePlan = async (record, enabled) => {
    try {
      const res = await API.put('/api/subscription_plan/', {
        ...record,
        enabled,
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      loadPlans(activePage, pageSize);
    } catch (error) {
      showError(error.message || t('更新套餐状态失败'));
    }
  };

  const columns = [
    { title: t('套餐名称'), dataIndex: 'name' },
    { title: t('价格'), dataIndex: 'price', render: (value) => `$${Number(value || 0).toFixed(2)}` },
    { title: t('总额度'), dataIndex: 'total_quota', render: (value) => renderQuota(value) },
    { title: t('窗口额度'), dataIndex: 'reset_quota', render: (value) => renderQuota(value) },
    {
      title: t('刷新时间'),
      render: (_, record) => `${record.reset_interval_count}${record.reset_interval_unit}`,
    },
    {
      title: t('启用'),
      dataIndex: 'enabled',
      render: (value, record) => (
        <Switch checked={value} onChange={(checked) => togglePlan(record, checked)} />
      ),
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      render: (value) => timestamp2string(value),
    },
    {
      title: t('操作'),
      render: (_, record) => (
        <Button size='small' onClick={() => openEdit(record)}>
          {t('编辑')}
        </Button>
      ),
    },
  ];

  return (
    <div className='mt-[60px]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('套餐管理')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('管理可售卖的用户套餐，包括价格、额度、周期与刷新规则')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <Button theme='solid' onClick={openCreate}>{t('新增套餐')}</Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <Card>
            <Spin spinning={loading}>
              <Table rowKey='id' pagination={false} columns={columns} dataSource={plans} />
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
                      loadPlans(page, pageSize);
                    }}
                    onPageSizeChange={(size) => {
                      setPageSize(size);
                      setActivePage(1);
                      loadPlans(1, size);
                    }}
                  />
                </div>
              )}
            </Spin>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Modal visible={modalVisible} title={t(editingPlan.id ? '编辑套餐' : '新增套餐')} onOk={savePlan} onCancel={() => setModalVisible(false)}>
        <Form initValues={editingPlan} getFormApi={(api) => (formApiRef.current = api)}>
          <Form.Input field='id' noLabel hidden />
          <Form.Input field='name' label={t('套餐名称')} />
          <Form.TextArea field='description' label={t('描述')} />
          <Form.InputNumber field='price' label={t('价格')} precision={2} style={{ width: '100%' }} />
          <Form.InputNumber field='total_quota' label={t('总额度')} style={{ width: '100%' }} />
          <Form.InputNumber field='reset_quota' label={t('窗口额度')} style={{ width: '100%' }} />
          <Space align='start' style={{ width: '100%' }}>
            <Form.InputNumber field='duration_count' label={t('周期数')} style={{ width: 120 }} />
            <Form.Select field='duration_unit' label={t('周期单位')} optionList={unitOptions} style={{ width: 160 }} />
          </Space>
          <Space align='start' style={{ width: '100%' }}>
            <Form.InputNumber field='reset_interval_count' label={t('刷新数')} style={{ width: 120 }} />
            <Form.Select field='reset_interval_unit' label={t('刷新单位')} optionList={unitOptions} style={{ width: 160 }} />
          </Space>
          <Form.Switch field='enabled' label={t('启用')} />
        </Form>
      </Modal>
    </div>
  );
};

export default SubscriptionPlanPage;
