import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Form,
  InputNumber,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, showWarning } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const MODEL_RATIOS_KEY = 'subscription_setting.model_ratios';

const defaultInputs = {
  'subscription_setting.payment_mode': 'both',
  [MODEL_RATIOS_KEY]: '{}',
};

const parseModelRatios = (value) => {
  try {
    const parsed = JSON.parse(value || '{}');
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {};
    }
    return Object.fromEntries(
      Object.entries(parsed)
        .map(([model, ratio]) => [model, Number(ratio)])
        .filter(
          ([model, ratio]) => model && Number.isFinite(ratio) && ratio > 0,
        ),
    );
  } catch {
    return {};
  }
};

const stringifyModelRatios = (ratios) => {
  const sorted = {};
  Object.keys(ratios)
    .sort((a, b) => a.localeCompare(b))
    .forEach((key) => {
      sorted[key] = ratios[key];
    });
  return JSON.stringify(sorted, null, 2);
};

const formatRatio = (value) => {
  const ratio = Number(value || 1);
  if (!Number.isFinite(ratio)) {
    return '1x';
  }
  return `${Number(ratio.toFixed(4))}x`;
};

const SubscriptionSetting = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(defaultInputs);
  const [initialInputs, setInitialInputs] = useState(defaultInputs);
  const [models, setModels] = useState([]);
  const formApiRef = useRef(null);

  const modelRatios = useMemo(
    () => parseModelRatios(inputs[MODEL_RATIOS_KEY]),
    [inputs],
  );

  const subscriptionModels = useMemo(
    () =>
      (Array.isArray(models) ? models : [])
        .filter((model) => model?.subscription_supported)
        .map((model) => ({
          key: model.model_name,
          modelName: model.model_name,
          ratio:
            modelRatios[model.model_name] ||
            Number(model.subscription_model_ratio || 1),
          groups: Array.isArray(model.enable_groups)
            ? model.enable_groups.filter(Boolean)
            : [],
        }))
        .sort((a, b) => a.modelName.localeCompare(b.modelName)),
    [models, modelRatios],
  );

  const loadOptions = async () => {
    setLoading(true);
    try {
      const [optionRes, pricingRes] = await Promise.all([
        API.get('/api/option/'),
        API.get('/api/pricing'),
      ]);
      if (!optionRes.data.success) {
        showError(optionRes.data.message);
        return;
      }
      if (!pricingRes.data.success) {
        showError(pricingRes.data.message);
        return;
      }
      const nextInputs = { ...defaultInputs };
      optionRes.data.data.forEach((item) => {
        if (Object.prototype.hasOwnProperty.call(nextInputs, item.key)) {
          nextInputs[item.key] = item.value || defaultInputs[item.key];
        }
      });
      setInputs(nextInputs);
      setInitialInputs(nextInputs);
      setModels(pricingRes.data.data || []);
      formApiRef.current?.setValues(nextInputs);
    } catch (error) {
      showError(error.message || t('加载套餐设置失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadOptions();
  }, []);

  const updateModelRatio = (modelName, value) => {
    const ratio = Number(value);
    if (!Number.isFinite(ratio) || ratio <= 0) {
      return;
    }
    const nextRatios = {
      ...parseModelRatios(inputs[MODEL_RATIOS_KEY]),
      [modelName]: ratio,
    };
    setInputs((prev) => ({
      ...prev,
      [MODEL_RATIOS_KEY]: stringifyModelRatios(nextRatios),
    }));
  };

  const validateModelRatios = () => {
    const ratios = parseModelRatios(inputs[MODEL_RATIOS_KEY]);
    for (const [model, ratio] of Object.entries(ratios)) {
      if (!model || !Number.isFinite(ratio) || ratio <= 0) {
        showError(t('套餐模型倍率必须大于 0'));
        return false;
      }
    }
    return true;
  };

  const handleSubmit = async () => {
    if (!validateModelRatios()) {
      return;
    }
    const updates = Object.keys(inputs).filter(
      (key) => inputs[key] !== initialInputs[key],
    );
    if (!updates.length) {
      showWarning(t('你似乎并没有修改什么'));
      return;
    }
    setLoading(true);
    try {
      const results = await Promise.all(
        updates.map((key) =>
          API.put('/api/option/', {
            key,
            value: inputs[key],
          }),
        ),
      );
      const failed = results.find((item) => !item.data.success);
      if (failed) {
        showError(failed.data.message);
        return;
      }
      showSuccess(t('套餐设置已保存'));
      await loadOptions();
    } catch (error) {
      showError(error.message || t('保存套餐设置失败'));
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'modelName',
      render: (text) => <Text copyable={{ content: text }}>{text}</Text>,
    },
    {
      title: t('套餐倍率'),
      dataIndex: 'ratio',
      width: 180,
      render: (value, record) => (
        <InputNumber
          value={value}
          min={0.01}
          step={0.1}
          precision={4}
          style={{ width: 140 }}
          onChange={(nextValue) =>
            updateModelRatio(record.modelName, nextValue)
          }
        />
      ),
    },
    {
      title: t('当前倍率'),
      dataIndex: 'ratio',
      width: 120,
      render: (value) => formatRatio(value),
    },
    {
      title: t('可用分组'),
      dataIndex: 'groups',
      render: (groups) => (
        <div className='flex flex-wrap gap-1'>
          {groups.map((group) => (
            <Tag key={group} color='white' size='small' shape='circle'>
              {group}
              {t('分组')}
            </Tag>
          ))}
        </div>
      ),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
        onValueChange={(values) =>
          setInputs((prev) => ({
            ...prev,
            ...values,
          }))
        }
      >
        <Form.Section text={t('套餐设置')}>
          <Form.Select
            field='subscription_setting.payment_mode'
            label={t('支付方式')}
            optionList={[
              { label: t('余额支付'), value: 'balance' },
              { label: t('现金支付'), value: 'cash' },
              { label: t('余额&现金都可'), value: 'both' },
            ]}
            style={{ width: '100%' }}
          />
          <div className='mb-4'>
            <div className='mb-2 font-medium'>{t('支持套餐模型')}</div>
            <Table
              dataSource={subscriptionModels}
              columns={columns}
              pagination={
                subscriptionModels.length > 20 ? { pageSize: 20 } : false
              }
              size='small'
              empty={
                <div style={{ textAlign: 'center', padding: 20 }}>
                  {t('暂无支持套餐的模型')}
                </div>
              }
            />
          </div>
          <Button onClick={handleSubmit}>{t('保存套餐设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
};

export default SubscriptionSetting;
