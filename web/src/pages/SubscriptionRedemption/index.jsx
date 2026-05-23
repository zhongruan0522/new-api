import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Pagination,
  Popover,
  Progress,
  Space,
  Spin,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import {
  API,
  copy,
  downloadTextAsFile,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { ITEMS_PER_PAGE, REDEMPTION_STATUS } from '../../constants';

const BATCH_SIZE = 100;

const emptyEditing = { id: undefined };

const getStatusTag = (record, t) => {
  const expired =
    record.status === REDEMPTION_STATUS.UNUSED &&
    record.expired_time !== 0 &&
    record.expired_time < Math.floor(Date.now() / 1000);
  if (expired) return <Tag color='orange'>{t('已过期')}</Tag>;
  if (record.status === REDEMPTION_STATUS.UNUSED) {
    return <Tag color='green'>{t('未使用')}</Tag>;
  }
  if (record.status === REDEMPTION_STATUS.USED) {
    return <Tag color='grey'>{t('已使用')}</Tag>;
  }
  return <Tag color='red'>{t('已禁用')}</Tag>;
};

const normalizeSubmitValues = (values) => {
  const next = { ...values };
  next.plan_id = Number(next.plan_id || 0);
  next.count = Number(next.count || 0);
  if (!next.expired_time) {
    next.expired_time = 0;
  } else if (next.expired_time instanceof Date) {
    next.expired_time = Math.floor(next.expired_time.getTime() / 1000);
  }
  return next;
};

const SubscriptionRedemptionPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [plans, setPlans] = useState([]);
  const [items, setItems] = useState([]);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [selectedRows, setSelectedRows] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editing, setEditing] = useState(emptyEditing);
  const [batchProgress, setBatchProgress] = useState(null);
  const formApiRef = useRef(null);

  const isEdit = editing.id !== undefined;

  const planOptions = useMemo(
    () => plans.map((item) => ({ label: item.name, value: item.id })),
    [plans],
  );

  const getInitValues = () => ({
    name: '',
    plan_id: planOptions[0]?.value,
    count: 1,
    expired_time: null,
  });

  const loadPlans = async () => {
    const res = await API.get('/api/subscription_plan/', {
      params: { p: 1, page_size: 100 },
    });
    if (!res.data.success) {
      throw new Error(res.data.message);
    }
    setPlans(res.data.data.items || []);
  };

  const loadRedemptions = async (page = activePage, size = pageSize) => {
    setLoading(true);
    try {
      const url = keyword
        ? '/api/subscription_redemption/search'
        : '/api/subscription_redemption/';
      const res = await API.get(url, {
        params: { keyword, p: page, page_size: size },
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setItems(res.data.data.items || []);
      setTotal(res.data.data.total || 0);
      setActivePage(res.data.data.page || page);
    } catch (error) {
      showError(error.message || t('加载套餐兑换码失败'));
    } finally {
      setLoading(false);
    }
  };

  const refresh = async (page = activePage) => {
    await loadRedemptions(page, pageSize);
  };

  useEffect(() => {
    setLoading(true);
    Promise.all([loadPlans(), loadRedemptions(1, pageSize)])
      .catch((error) => showError(error.message || t('加载套餐兑换码失败')))
      .finally(() => setLoading(false));
  }, []);

  const openCreate = () => {
    setEditing(emptyEditing);
    setModalVisible(true);
    setTimeout(() => formApiRef.current?.setValues(getInitValues()), 0);
  };

  const openEdit = (record) => {
    setEditing(record);
    setModalVisible(true);
    setTimeout(() => {
      formApiRef.current?.setValues({
        ...record,
        expired_time: record.expired_time
          ? new Date(record.expired_time * 1000)
          : null,
      });
    }, 0);
  };

  const promptDownload = (keys, name, partial = false) => {
    Modal.confirm({
      title: partial ? t('部分兑换码创建成功') : t('兑换码创建成功'),
      content: partial
        ? t('部分批次创建失败，已成功创建 {{count}} 个兑换码，是否下载？', {
            count: keys.length,
          })
        : t('兑换码创建成功，是否下载兑换码？', { count: keys.length }),
      onOk: () => downloadTextAsFile(keys.join('\n'), `${name}.txt`),
    });
  };

  const submit = async () => {
    const rawValues = formApiRef.current?.getValues();
    if (!rawValues) return;
    const values = normalizeSubmitValues(rawValues);
    if (!values.name) {
      showError(t('请输入名称'));
      return;
    }
    if (!values.plan_id) {
      showError(t('请选择套餐'));
      return;
    }
    if (!isEdit && values.count <= 0) {
      showError(t('生成数量必须大于0'));
      return;
    }
    setSubmitting(true);
    if (isEdit) {
      try {
        const res = await API.put('/api/subscription_redemption/', {
          ...values,
          id: editing.id,
        });
        if (!res.data.success) {
          showError(res.data.message);
          return;
        }
        showSuccess(t('套餐兑换码更新成功'));
        setModalVisible(false);
        await refresh();
      } catch (error) {
        showError(error.message || t('套餐兑换码更新失败'));
      } finally {
        setSubmitting(false);
      }
      return;
    }

    const totalCount = values.count;
    const batchCount = Math.ceil(totalCount / BATCH_SIZE);
    let allKeys = [];
    try {
      for (let i = 0; i < batchCount; i++) {
        const count =
          i === batchCount - 1 ? totalCount - i * BATCH_SIZE : BATCH_SIZE;
        setBatchProgress({ current: i + 1, total: batchCount });
        const res = await API.post('/api/subscription_redemption/', {
          ...values,
          count,
        });
        if (!res.data.success) {
          showError(res.data.message);
          if (allKeys.length) promptDownload(allKeys, values.name, true);
          return;
        }
        allKeys = allKeys.concat(res.data.data || []);
      }
      showSuccess(t('套餐兑换码创建成功'));
      setModalVisible(false);
      await refresh(1);
      if (allKeys.length) promptDownload(allKeys, values.name);
    } catch (error) {
      showError(error.message || t('创建套餐兑换码失败'));
      if (allKeys.length) promptDownload(allKeys, values.name, true);
    } finally {
      setSubmitting(false);
      setBatchProgress(null);
    }
  };

  const updateStatus = async (record, status) => {
    setLoading(true);
    try {
      const res = await API.put('/api/subscription_redemption/?status_only=true', {
        id: record.id,
        status,
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess(t('操作成功完成！'));
      await refresh();
    } catch (error) {
      showError(error.message || t('操作失败'));
    } finally {
      setLoading(false);
    }
  };

  const deleteOne = (record) => {
    Modal.confirm({
      title: t('确定是否要删除此兑换码？'),
      content: record.name,
      onOk: async () => {
        setLoading(true);
        try {
          const res = await API.delete(`/api/subscription_redemption/${record.id}/`);
          if (!res.data.success) {
            showError(res.data.message);
            return;
          }
          showSuccess(t('删除成功'));
          await refresh(items.length === 1 && activePage > 1 ? activePage - 1 : activePage);
        } catch (error) {
          showError(error.message || t('删除失败'));
        } finally {
          setLoading(false);
        }
      },
    });
  };

  const clearInvalid = () => {
    Modal.confirm({
      title: t('确定清除所有失效兑换码？'),
      content: t('将删除已使用、已禁用及过期的兑换码，此操作不可撤销。'),
      onOk: async () => {
        setLoading(true);
        try {
          const res = await API.delete('/api/subscription_redemption/invalid');
          if (!res.data.success) {
            showError(res.data.message);
            return;
          }
          showSuccess(t('已删除 {{count}} 条失效兑换码', { count: res.data.data }));
          await refresh(1);
        } catch (error) {
          showError(error.message || t('删除失败'));
        } finally {
          setLoading(false);
        }
      },
    });
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制到剪贴板！'));
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  const copySelected = async () => {
    if (!selectedRows.length) {
      showError(t('请至少选择一个兑换码！'));
      return;
    }
    await copyText(selectedRows.map((item) => `${item.name}    ${item.key}`).join('\n'));
  };

  const columns = [
    { title: t('ID'), dataIndex: 'id', width: 80 },
    { title: t('名称'), dataIndex: 'name' },
    { title: t('套餐'), dataIndex: 'plan_name' },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (_, record) => getStatusTag(record, t),
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_time',
      render: (value) => timestamp2string(value),
    },
    {
      title: t('过期时间'),
      dataIndex: 'expired_time',
      render: (value) => (value ? timestamp2string(value) : t('永不过期')),
    },
    {
      title: t('兑换人ID'),
      dataIndex: 'used_user_id',
      render: (value) => (value ? value : t('无')),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      fixed: 'right',
      width: 250,
      render: (_, record) => (
        <Space>
          <Popover content={record.key} style={{ padding: 20 }} position='top'>
            <Button size='small' type='tertiary'>
              {t('查看')}
            </Button>
          </Popover>
          <Button size='small' onClick={() => copyText(record.key)}>
            {t('复制')}
          </Button>
          <Button
            size='small'
            type='tertiary'
            disabled={record.status !== REDEMPTION_STATUS.UNUSED}
            onClick={() => openEdit(record)}
          >
            {t('编辑')}
          </Button>
          {record.status === REDEMPTION_STATUS.UNUSED ? (
            <Button
              size='small'
              type='warning'
              theme='light'
              onClick={() => updateStatus(record, REDEMPTION_STATUS.DISABLED)}
            >
              {t('禁用')}
            </Button>
          ) : (
            <Button
              size='small'
              type='secondary'
              theme='light'
              disabled={record.status === REDEMPTION_STATUS.USED}
              onClick={() => updateStatus(record, REDEMPTION_STATUS.UNUSED)}
            >
              {t('启用')}
            </Button>
          )}
          <Button size='small' type='danger' theme='light' onClick={() => deleteOne(record)}>
            {t('删除')}
          </Button>
        </Space>
      ),
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
          <Space wrap>
            <Button theme='solid' onClick={openCreate}>
              {t('新增套餐兑换码')}
            </Button>
            <Button onClick={copySelected}>{t('复制所选兑换码到剪贴板')}</Button>
            <Button type='danger' onClick={clearInvalid}>
              {t('清除失效兑换码')}
            </Button>
          </Space>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <Card>
            <div className='flex flex-col md:flex-row gap-2 md:items-center md:justify-between mb-3'>
              <Input
                value={keyword}
                onChange={setKeyword}
                placeholder={t('搜索名称、兑换码或套餐')}
                showClear
                style={{ maxWidth: 360 }}
                onEnterPress={() => {
                  setActivePage(1);
                  loadRedemptions(1, pageSize);
                }}
              />
              <Button
                loading={loading}
                onClick={() => {
                  setActivePage(1);
                  loadRedemptions(1, pageSize);
                }}
              >
                {t('搜索')}
              </Button>
            </div>
            <Spin spinning={loading}>
              <Table
                rowKey='id'
                columns={columns}
                dataSource={items}
                pagination={false}
                scroll={{ x: 'max-content' }}
                rowSelection={{
                  onChange: (_, rows) => setSelectedRows(rows),
                }}
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
                      loadRedemptions(page, pageSize);
                    }}
                    onPageSizeChange={(size) => {
                      setPageSize(size);
                      setActivePage(1);
                      loadRedemptions(1, size);
                    }}
                  />
                </div>
              )}
            </Spin>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Modal
        visible={modalVisible}
        title={t(isEdit ? '编辑套餐兑换码' : '新增套餐兑换码')}
        onOk={submit}
        onCancel={() => setModalVisible(false)}
        confirmLoading={submitting}
      >
        <Form initValues={getInitValues()} getFormApi={(api) => (formApiRef.current = api)}>
          <Form.Input
            field='name'
            label={t('名称')}
            rules={[{ required: true, message: t('请输入名称') }]}
          />
          <Form.Select
            field='plan_id'
            label={t('套餐')}
            optionList={planOptions}
            rules={[{ required: true, message: t('请选择套餐') }]}
          />
          {!isEdit && (
            <Form.InputNumber
              field='count'
              label={t('生成数量')}
              min={1}
              rules={[{ required: true, message: t('请输入生成数量') }]}
              style={{ width: '100%' }}
              extraText={t('单次上限 {{max}} 个，超出将自动分批创建', {
                max: BATCH_SIZE,
              })}
            />
          )}
          <Form.DatePicker
            field='expired_time'
            label={t('过期时间')}
            type='dateTime'
            style={{ width: '100%' }}
            placeholder={t('选择过期时间（可选，留空为永久）')}
          />
          {batchProgress && (
            <Progress
              percent={Math.round((batchProgress.current / batchProgress.total) * 100)}
              showInfo
            />
          )}
        </Form>
      </Modal>
    </div>
  );
};

export default SubscriptionRedemptionPage;
