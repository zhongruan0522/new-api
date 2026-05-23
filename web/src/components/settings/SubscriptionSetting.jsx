import React, { useEffect, useRef, useState } from 'react';
import { Button, Form, Spin } from '@douyinfe/semi-ui';
import { API, showError, showSuccess, showWarning, verifyJSON } from '../../helpers';
import { useTranslation } from 'react-i18next';

const defaultInputs = {
  'subscription_setting.payment_mode': 'both',
  'subscription_setting.model_ratios': '{}',
};

const SubscriptionSetting = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(defaultInputs);
  const [initialInputs, setInitialInputs] = useState(defaultInputs);
  const formApiRef = useRef(null);

  const loadOptions = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/');
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      const nextInputs = { ...defaultInputs };
      res.data.data.forEach((item) => {
        if (Object.prototype.hasOwnProperty.call(nextInputs, item.key)) {
          nextInputs[item.key] = item.value || defaultInputs[item.key];
        }
      });
      setInputs(nextInputs);
      setInitialInputs(nextInputs);
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

  const handleSubmit = async () => {
    if (inputs['subscription_setting.model_ratios']) {
      if (!verifyJSON(inputs['subscription_setting.model_ratios'])) {
        showError(t('套餐模型倍率必须是合法 JSON'));
        return;
      }
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

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
        onValueChange={(values) => setInputs(values)}
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
          <Form.TextArea
            field='subscription_setting.model_ratios'
            label={t('套餐模型消耗倍率')}
            placeholder={t('请填写模型到套餐倍率的 JSON，例如 {"gpt-4o":1.2}')}
            autosize
            extraText={t('当前实现为 JSON 配置；后端已支持读取此配置进行套餐扣费与模型广场展示')}
          />
          <Button onClick={handleSubmit}>{t('保存套餐设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
};

export default SubscriptionSetting;
