import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Form,
  Modal,
  Popconfirm,
  Switch,
  Table,
  Tag,
  Space,
  InputNumber,
  Typography,
  Tooltip,
} from '@douyinfe/semi-ui';
import { ArrowUp, ArrowDown, Plus, Trash2, Pencil } from 'lucide-react';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import { API, isRoot, showError, showSuccess } from '../../helpers';

const WEEKDAY_OPTIONS = [
  { value: 0, label: '周日' },
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
];

const DynamicRatio = () => {
  const { t } = useTranslation();
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(false);
  const [globalEnabled, setGlobalEnabled] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState(null);
  const [groups, setGroups] = useState([]);
  const [formApi, setFormApi] = useState(null);
  const canEdit = isRoot();

  const loadRules = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/dynamic_ratio/rules');
      const { success, message, data } = res.data;
      if (success) {
        setRules(data || []);
      } else {
        showError(message);
      }
    } catch {
      showError(t('加载规则失败'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const loadGlobalEnabled = useCallback(async () => {
    try {
      const res = await API.get('/api/dynamic_ratio/status');
      const { success, data, message } = res.data;
      if (success) {
        setGlobalEnabled(Boolean(data?.enabled));
      } else {
        showError(message);
      }
    } catch {
      showError(t('加载全局开关失败'));
    }
  }, [t]);

  const loadGroups = useCallback(async () => {
    try {
      const res = await API.get('/api/group/');
      const { success, data, message } = res.data;
      if (success && data) {
        const groupList = data.map((group) => ({
          value: group,
          label: group,
        }));
        setGroups(groupList);
      } else if (!success) {
        showError(message);
      }
    } catch {
      showError(t('加载分组失败'));
    }
  }, [t]);

  useEffect(() => {
    loadRules();
    loadGlobalEnabled();
    loadGroups();
  }, [loadRules, loadGlobalEnabled, loadGroups]);

  const handleToggleGlobal = async (checked) => {
    try {
      const res = await API.put('/api/dynamic_ratio/enabled', {
        enabled: checked,
      });
      const { success, message } = res.data;
      if (success) {
        setGlobalEnabled(checked);
        showSuccess(t(checked ? '已启用动态倍率' : '已禁用动态倍率'));
      } else {
        showError(message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const handleAdd = () => {
    setEditingRule(null);
    setModalVisible(true);
  };

  const handleEdit = (record) => {
    setEditingRule(record);
    setModalVisible(true);
  };

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/dynamic_ratio/rules/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('删除成功'));
        loadRules();
      } else {
        showError(message);
      }
    } catch {
      showError(t('删除失败'));
    }
  };

  const handleMoveUp = async (index) => {
    if (index <= 0) return;
    const newRules = [...rules];
    const reordered = newRules.map((r, i) => {
      if (i === index - 1)
        return { ...newRules[index], priority: newRules[index - 1].priority };
      if (i === index)
        return { ...newRules[index - 1], priority: newRules[index].priority };
      return r;
    });
    await saveReorder(reordered);
  };

  const handleMoveDown = async (index) => {
    if (index >= rules.length - 1) return;
    const newRules = [...rules];
    const reordered = newRules.map((r, i) => {
      if (i === index)
        return { ...newRules[index + 1], priority: newRules[index].priority };
      if (i === index + 1)
        return { ...newRules[index], priority: newRules[index + 1].priority };
      return r;
    });
    await saveReorder(reordered);
  };

  const saveReorder = async (reordered) => {
    try {
      const res = await API.put('/api/dynamic_ratio/rules/reorder', {
        ids: reordered.map((rule) => rule.id),
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('排序已更新'));
        loadRules();
      } else {
        showError(message);
      }
    } catch {
      showError(t('排序失败'));
    }
  };

  const handleModalOk = async () => {
    if (!formApi) return;
    const values = formApi.getValues();

    if (!values.group) {
      showError(t('请选择分组'));
      return;
    }
    if (!values.ratio || values.ratio <= 0) {
      showError(t('倍率值必须大于0'));
      return;
    }
    if (
      (values.start_time && !values.end_time) ||
      (!values.start_time && values.end_time)
    ) {
      showError(t('开始时间和结束时间必须同时填写'));
      return;
    }

    const payload = {
      group: values.group,
      concurrency: values.concurrency || null,
      weekdays:
        Array.isArray(values.weekdays) && values.weekdays.length > 0
          ? JSON.stringify(values.weekdays)
          : '',
      start_time: values.start_time || '',
      end_time: values.end_time || '',
      ratio: values.ratio,
      enable: values.enable !== false,
      priority: values.priority || 0,
    };

    try {
      let res;
      if (editingRule) {
        res = await API.put('/api/dynamic_ratio/rules', {
          id: editingRule.id,
          ...payload,
        });
      } else {
        res = await API.post('/api/dynamic_ratio/rules', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(t(editingRule ? '更新成功' : '创建成功'));
        setModalVisible(false);
        loadRules();
      } else {
        showError(message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const formatWeekdays = (weekdaysStr) => {
    if (!weekdaysStr) return t('每天');
    try {
      const arr = JSON.parse(weekdaysStr);
      if (!Array.isArray(arr) || arr.length === 0) return t('每天');
      return arr
        .map((d) => WEEKDAY_OPTIONS.find((o) => o.value === d)?.label || d)
        .join(', ');
    } catch {
      return weekdaysStr;
    }
  };

  const columns = [
    {
      title: t('启用'),
      dataIndex: 'enable',
      width: 70,
      render: (text, record) => (
        <Switch
          checked={record.enable !== false}
          disabled={!canEdit}
          onChange={async (checked) => {
            try {
              const res = await API.put('/api/dynamic_ratio/rules', {
                ...record,
                enable: checked,
              });
              const { success, message } = res.data;
              if (success) {
                showSuccess(t(checked ? '已启用' : '已禁用'));
                loadRules();
              } else {
                showError(message);
              }
            } catch {
              showError(t('操作失败'));
            }
          }}
          size='small'
        />
      ),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      width: 120,
      render: (text) => <Tag>{text}</Tag>,
    },
    {
      title: t('并发阈值'),
      dataIndex: 'concurrency',
      width: 100,
      render: (text) =>
        text ? (
          <Tag color='blue'>{text}</Tag>
        ) : (
          <Tag color='grey'>{t('不限')}</Tag>
        ),
    },
    {
      title: t('适用星期'),
      dataIndex: 'weekdays',
      width: 150,
      render: (text) => (
        <Typography.Text ellipsis={{ showTooltip: true }}>
          {formatWeekdays(text)}
        </Typography.Text>
      ),
    },
    {
      title: t('时间段'),
      width: 140,
      render: (_, record) => {
        if (!record.start_time && !record.end_time)
          return <Tag color='grey'>{t('不限')}</Tag>;
        return `${record.start_time || '00:00'} - ${record.end_time || '23:59'}`;
      },
    },
    {
      title: t('倍率值'),
      dataIndex: 'ratio',
      width: 80,
      render: (text) => {
        const color = text > 3 ? 'red' : text > 1.5 ? 'orange' : 'blue';
        return <Tag color={color}>{text}x</Tag>;
      },
    },
    {
      title: t('优先级'),
      dataIndex: 'priority',
      width: 70,
      render: (text) => text ?? 0,
    },
    {
      title: t('操作'),
      width: 140,
      render: (_, record, index) => (
        <Space>
          <Tooltip content={t('上移')}>
            <Button
              icon={<ArrowUp size={14} />}
              size='small'
              disabled={!canEdit || index === 0}
              onClick={() => handleMoveUp(index)}
            />
          </Tooltip>
          <Tooltip content={t('下移')}>
            <Button
              icon={<ArrowDown size={14} />}
              size='small'
              disabled={!canEdit || index === rules.length - 1}
              onClick={() => handleMoveDown(index)}
            />
          </Tooltip>
          <Button
            icon={<Pencil size={14} />}
            size='small'
            disabled={!canEdit}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title={t('确认删除此规则？')}
            disabled={!canEdit}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button
              icon={<Trash2 size={14} />}
              size='small'
              type='danger'
              disabled={!canEdit}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className='mt-[60px]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('动态倍率')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('根据并发、时间段等条件自动调整倍率，引导用户错峰使用')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <Space>
            <span
              style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}
            >
              {t('全局开关')}：
              <Switch
                checked={globalEnabled}
                onChange={handleToggleGlobal}
                disabled={!canEdit}
              />
            </span>
            <Button
              theme='solid'
              icon={<Plus size={14} />}
              onClick={handleAdd}
              disabled={!canEdit}
            >
              {t('新增规则')}
            </Button>
          </Space>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <Table
            columns={columns}
            dataSource={rules}
            loading={loading}
            rowKey='id'
            pagination={false}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Modal
        title={editingRule ? t('编辑规则') : t('新增规则')}
        visible={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText={t('确定')}
        cancelText={t('取消')}
        okButtonProps={{ disabled: !canEdit }}
        width={520}
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          initValues={
            editingRule
              ? {
                  group: editingRule.group,
                  concurrency: editingRule.concurrency,
                  weekdays: editingRule.weekdays
                    ? (() => {
                        try {
                          return JSON.parse(editingRule.weekdays);
                        } catch {
                          return [];
                        }
                      })()
                    : [],
                  start_time: editingRule.start_time || '',
                  end_time: editingRule.end_time || '',
                  ratio: editingRule.ratio,
                  enable: editingRule.enable !== false,
                  priority: editingRule.priority || 0,
                }
              : {
                  group: '',
                  concurrency: null,
                  weekdays: [],
                  start_time: '',
                  end_time: '',
                  ratio: 1.5,
                  enable: true,
                  priority: 0,
                }
          }
        >
          <Form.Select
            field='group'
            label={t('分组')}
            optionList={groups}
            placeholder={t('请选择分组')}
            filter
            required
            style={{ width: '100%' }}
          />
          <Form.InputNumber
            field='concurrency'
            label={t('并发阈值')}
            placeholder={t('留空表示不限')}
            min={1}
            style={{ width: '100%' }}
          />
          <Form.Select
            field='weekdays'
            label={t('适用星期')}
            multiple
            optionList={WEEKDAY_OPTIONS}
            placeholder={t('留空表示每天')}
            style={{ width: '100%' }}
          />
          <div style={{ display: 'flex', gap: 8 }}>
            <Form.Input
              field='start_time'
              label={t('开始时间')}
              placeholder='HH:MM'
              style={{ flex: 1 }}
            />
            <Form.Input
              field='end_time'
              label={t('结束时间')}
              placeholder='HH:MM'
              style={{ flex: 1 }}
            />
          </div>
          <Form.InputNumber
            field='ratio'
            label={t('倍率值')}
            min={0.01}
            step={0.1}
            required
            style={{ width: '100%' }}
          />
          <Form.InputNumber
            field='priority'
            label={t('优先级')}
            placeholder={t('越小越优先')}
            style={{ width: '100%' }}
          />
          <Form.Switch field='enable' label={t('启用')} />
        </Form>
      </Modal>
    </div>
  );
};

export default DynamicRatio;
