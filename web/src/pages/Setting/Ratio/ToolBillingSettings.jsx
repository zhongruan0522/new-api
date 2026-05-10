import React, { useEffect, useState, useCallback } from 'react';
import {
  Table,
  Button,
  Input,
  Modal,
  Form,
  Space,
  Select,
  Switch,
  Tag,
  Popconfirm,
  InputNumber,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlus, IconDelete, IconEdit, IconSave } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const TOOL_TYPES = [
  { value: 'web_search', label: 'Web Search' },
  { value: 'image_generation', label: 'Image Generation' },
];

const BILLING_MODES = [
  { value: 'per_call', label: '按次计费 (USD/次)' },
];

const PROVIDERS = [
  { value: '', label: '全部' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'claude', label: 'Claude' },
  { value: 'gemini', label: 'Gemini' },
];

const QUALITIES = [
  { value: '', label: '全部' },
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
];

const SIZES = [
  { value: '', label: '全部' },
  { value: '1024x1024', label: '1024x1024' },
  { value: '1024x1536', label: '1024x1536' },
  { value: '1536x1024', label: '1536x1024' },
];

const TOOL_TYPE_COLORS = {
  web_search: 'blue',
  image_generation: 'purple',
};

const ToolBillingSettings = () => {
  const { t } = useTranslation();
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState(null);
  const [editForm, setEditForm] = useState({
    id: '',
    name: '',
    tool_type: 'web_search',
    billing_mode: 'per_call',
    price: 0,
    model_filter: '',
    quality: '',
    size: '',
    provider: '',
    enabled: true,
  });

  const fetchRules = useCallback(async () => {
    try {
      setLoading(true);
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data;
      if (success) {
        const rulesOption = data.find(
          (item) => item.key === 'tool_billing_setting.rules'
        );
        if (rulesOption) {
          try {
            const parsed = JSON.parse(rulesOption.value);
            setRules(Array.isArray(parsed) ? parsed : []);
          } catch {
            setRules([]);
          }
        } else {
          setRules([]);
        }
      } else {
        showError(message);
      }
    } catch {
      showError('获取工具计费规则失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const saveRules = async (newRules) => {
    try {
      setSaving(true);
      const res = await API.put('/api/option/', {
        key: 'tool_billing_setting.rules',
        value: JSON.stringify(newRules),
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess('工具计费规则已保存');
        setRules(newRules);
      } else {
        showError(message);
      }
    } catch {
      showError('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleSave = () => {
    saveRules(rules);
  };

  const openAddModal = () => {
    setEditingRule(null);
    setEditForm({
      id: '',
      name: '',
      tool_type: 'web_search',
      billing_mode: 'per_call',
      price: 0,
      model_filter: '',
      quality: '',
      size: '',
      provider: '',
      enabled: true,
    });
    setModalVisible(true);
  };

  const openEditModal = (record, index) => {
    setEditingRule(index);
    setEditForm({ ...record });
    setModalVisible(true);
  };

  const handleModalOk = () => {
    if (!editForm.id.trim()) {
      showError('规则 ID 不能为空');
      return;
    }
    if (!editForm.name.trim()) {
      showError('规则名称不能为空');
      return;
    }

    const newRules = [...rules];
    const rule = { ...editForm };

    if (editingRule !== null) {
      newRules[editingRule] = rule;
    } else {
      // Check duplicate ID
      if (newRules.some((r) => r.id === rule.id)) {
        showError('规则 ID 已存在');
        return;
      }
      newRules.push(rule);
    }

    setRules(newRules);
    setModalVisible(false);
  };

  const handleDelete = (index) => {
    const newRules = rules.filter((_, i) => i !== index);
    setRules(newRules);
  };

  const handleToggleEnabled = (index, enabled) => {
    const newRules = [...rules];
    newRules[index] = { ...newRules[index], enabled };
    setRules(newRules);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 200,
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: t('名称'),
      dataIndex: 'name',
      width: 200,
    },
    {
      title: t('工具类型'),
      dataIndex: 'tool_type',
      width: 120,
      render: (text) => (
        <Tag color={TOOL_TYPE_COLORS[text] || 'grey'}>
          {text === 'web_search' ? 'Web Search' : 'Image Gen'}
        </Tag>
      ),
    },
    {
      title: t('计费模式'),
      dataIndex: 'billing_mode',
      width: 120,
      render: (text) =>
        text === 'per_call' ? '按次' : text,
    },
    {
      title: t('价格 (USD)'),
      dataIndex: 'price',
      width: 120,
      render: (text, record) => {
        return `$${text}/次`;
      },
    },
    {
      title: t('供应商'),
      dataIndex: 'provider',
      width: 90,
      render: (text) => text || '全部',
    },
    {
      title: t('模型过滤'),
      dataIndex: 'model_filter',
      width: 160,
      render: (text) => text || '全部',
    },
    {
      title: t('质量'),
      dataIndex: 'quality',
      width: 80,
      render: (text) => text || '-',
    },
    {
      title: t('尺寸'),
      dataIndex: 'size',
      width: 110,
      render: (text) => text || '-',
    },
    {
      title: t('启用'),
      dataIndex: 'enabled',
      width: 80,
      render: (text, record, index) => (
        <Switch
          checked={text}
          onChange={(val) => handleToggleEnabled(index, val)}
          size='small'
        />
      ),
    },
    {
      title: t('操作'),
      width: 120,
      render: (_, record, index) => (
        <Space>
          <Button
            icon={<IconEdit />}
            theme='borderless'
            size='small'
            onClick={() => openEditModal(record, index)}
          />
          <Popconfirm
            title={t('确认删除此规则？')}
            onConfirm={() => handleDelete(index)}
          >
            <Button
              icon={<IconDelete />}
              theme='borderless'
              size='small'
              type='danger'
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const isImageGeneration = editForm.tool_type === 'image_generation';

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          marginBottom: 16,
        }}
      >
        <Space>
          <Button icon={<IconPlus />} theme='solid' onClick={openAddModal}>
            {t('添加规则')}
          </Button>
        </Space>
        <Button
          icon={<IconSave />}
          theme='solid'
          type='primary'
          loading={saving}
          onClick={handleSave}
        >
          {t('保存')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={rules}
        rowKey='id'
        loading={loading}
        pagination={false}
        size='small'
      />

      <Modal
        title={editingRule !== null ? t('编辑规则') : t('添加规则')}
        visible={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        okText={t('确定')}
        cancelText={t('取消')}
        width={600}
      >
        <Form labelPosition='left' labelWidth={120}>
          <Form.Input
            field='id'
            label='ID'
            value={editForm.id}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, id: val }))
            }
            placeholder='如: web_search_openai'
            disabled={editingRule !== null}
          />
          <Form.Input
            field='name'
            label={t('名称')}
            value={editForm.name}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, name: val }))
            }
            placeholder='如: OpenAI Web Search'
          />
          <Form.Select
            field='tool_type'
            label={t('工具类型')}
            value={editForm.tool_type}
            onChange={(val) =>
              setEditForm((prev) => ({
                ...prev,
                tool_type: val,
                quality: val === 'web_search' ? '' : prev.quality,
                size: val === 'web_search' ? '' : prev.size,
              }))
            }
            optionList={TOOL_TYPES}
          />
          <Form.Select
            field='billing_mode'
            label={t('计费模式')}
            value={editForm.billing_mode}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, billing_mode: val }))
            }
            optionList={BILLING_MODES}
          />
          <Form.InputNumber
            field='price'
            label={t('价格 (USD)')}
            value={editForm.price}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, price: val }))
            }
            min={0}
            step={0.001}
            precision={6}
          />
          <Form.Select
            field='provider'
            label={t('供应商')}
            value={editForm.provider}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, provider: val }))
            }
            optionList={PROVIDERS}
          />
          <Form.Input
            field='model_filter'
            label={t('模型过滤')}
            value={editForm.model_filter}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, model_filter: val }))
            }
            placeholder='如: gpt-4o*,gpt-4.1* (留空=全部)'
          />
          {isImageGeneration && (
            <>
              <Form.Select
                field='quality'
                label={t('质量')}
                value={editForm.quality}
                onChange={(val) =>
                  setEditForm((prev) => ({ ...prev, quality: val }))
                }
                optionList={QUALITIES}
              />
              <Form.Select
                field='size'
                label={t('尺寸')}
                value={editForm.size}
                onChange={(val) =>
                  setEditForm((prev) => ({ ...prev, size: val }))
                }
                optionList={SIZES}
              />
            </>
          )}
          <Form.Switch
            field='enabled'
            label={t('启用')}
            checked={editForm.enabled}
            onChange={(val) =>
              setEditForm((prev) => ({ ...prev, enabled: val }))
            }
          />
        </Form>
      </Modal>
    </div>
  );
};

export default ToolBillingSettings;
