import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Modal,
  Spin,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';

const initValues = {
  id: 0,
  name: '',
  plan_id: 0,
  count: 1,
  expired_time: 0,
};

const SubscriptionRedemptionPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [plans, setPlans] = useState([]);
  const [items, setItems] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [createdKeys, setCreatedKeys] = useState([]);
  const formApiRef = useRef(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const [listRes, plansRes] = await Promise.all([
        API.get('/api/subscription_redemption/', {
          params: { p: 1, page_size: 100 },
        }),
        API.get('/api/subscription_plan/', {
          params: { p: 1, page_size: 100 },
        }),
      ]);
      if (!listRes.data.success) {
        showError(listRes.data.message);
        return;
      }
      if (!plansRes.data.success) {
        showError(plansRes.data.message);
        return;
      }
      setItems(listRes.data.data.items || []);
      setPlans((plansRes.data.data.items || []).map((item) => ({
        label: item.name,
        value: item.id,
      })));
    } catch (error) {
      showError(error.message || t('加载套餐兑换码失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const submit = async () => {
    const values = formApiRef.current?.getValues();
    if (!values) return;
    try {
      const res = await API.post('/api/subscription_redemption/', values);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setCreatedKeys(res.data.data || []);
      showSuccess(t('套餐兑换码创建成功'));
      setModalVisible(false);
      loadData();
    } catch (error) {
      showError(error.message || t('创建套餐兑换码失败'));
    }
  };

  const columns = [
    { title: t('名称'), dataIndex: 'name' },
    { title: t('套餐'), dataIndex: 'plan_name' },
    { title: t('兑换码'), dataIndex: 'key' },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (value) => {
        if (value === 1) return <Tag color='green'>{t('可用')}</Tag>;
        if (value === 3) return <Tag color='orange'>{t('已使用')}</Tag>;
        return <Tag>{t('禁用')}</Tag>;
      },
    },
    {
      title: t('过期时间'),
      dataIndex: 'expired_time',
      render: (value) => (value ? timestamp2string(value) : t('永不过期')),
    },
  ];

  return (
    <div className='mt-[60px]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('套餐兑换码')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('批量创建套餐兑换码，并将套餐作为兑换结果发放给用户')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <Button theme='solid' onClick={() => setModalVisible(true)}>
            {t('新增套餐兑换码')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <Card>
            <Spin spinning={loading}>
              <Table rowKey='id' columns={columns} dataSource={items} pagination={false} />
              {createdKeys.length > 0 && (
                <div className='mt-4 whitespace-pre-wrap break-all text-sm'>
                  {createdKeys.join('\n')}
                </div>
              )}
            </Spin>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Modal
        visible={modalVisible}
        title={t('新增套餐兑换码')}
        onOk={submit}
        onCancel={() => setModalVisible(false)}
      >
        <Form initValues={initValues} getFormApi={(api) => (formApiRef.current = api)}>
          <Form.Input field='name' label={t('名称')} />
          <Form.Select field='plan_id' label={t('套餐')} optionList={plans} />
          <Form.InputNumber field='count' label={t('数量')} min={1} max={100} style={{ width: '100%' }} />
          <Form.InputNumber field='expired_time' label={t('过期时间戳')} style={{ width: '100%' }} extraText={t('留空或填 0 表示永不过期')} />
        </Form>
      </Modal>
    </div>
  );
};

export default SubscriptionRedemptionPage;
