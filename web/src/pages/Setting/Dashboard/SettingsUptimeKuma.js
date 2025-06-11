import React, { useEffect, useState, useRef, useMemo, useCallback } from 'react';
import {
  Form,
  Button,
  Typography,
  Row,
  Col,
} from '@douyinfe/semi-ui';
import {
  Save,
  Activity
} from 'lucide-react';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const SettingsUptimeKuma = ({ options, refresh }) => {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);

  const initValues = useMemo(() => ({
    uptimeKumaUrl: options?.UptimeKumaUrl || '',
    uptimeKumaSlug: options?.UptimeKumaSlug || ''
  }), [options?.UptimeKumaUrl, options?.UptimeKumaSlug]);

  useEffect(() => {
    if (formApiRef.current) {
      formApiRef.current.setValues(initValues, { isOverride: true });
    }
  }, [initValues]);

  const handleSave = async () => {
    const api = formApiRef.current;
    if (!api) {
      showError(t('表单未初始化'));
      return;
    }

    try {
      setLoading(true);
      const { uptimeKumaUrl, uptimeKumaSlug } = await api.validate();

      const trimmedUrl = (uptimeKumaUrl || '').trim();
      const trimmedSlug = (uptimeKumaSlug || '').trim();

      if (trimmedUrl === options?.UptimeKumaUrl && trimmedSlug === options?.UptimeKumaSlug) {
        showSuccess(t('无需保存，配置未变动'));
        return;
      }

      const [urlRes, slugRes] = await Promise.all([
        trimmedUrl === options?.UptimeKumaUrl ? Promise.resolve({ data: { success: true } }) : API.put('/api/option/', {
          key: 'UptimeKumaUrl',
          value: trimmedUrl
        }),
        trimmedSlug === options?.UptimeKumaSlug ? Promise.resolve({ data: { success: true } }) : API.put('/api/option/', {
          key: 'UptimeKumaSlug',
          value: trimmedSlug
        })
      ]);

      if (!urlRes.data.success) throw new Error(urlRes.data.message || t('URL 保存失败'));
      if (!slugRes.data.success) throw new Error(slugRes.data.message || t('Slug 保存失败'));

      showSuccess(t('Uptime Kuma 设置保存成功'));
      refresh?.();
    } catch (err) {
      console.error(err);
      showError(err.message || t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  const isValidUrl = useCallback((string) => {
    try {
      new URL(string);
      return true;
    } catch (_) {
      return false;
    }
  }, []);

  const renderHeader = () => (
    <div className="flex flex-col w-full">
      <div className="flex flex-col md:flex-row md:justify-between md:items-center gap-4 mb-2">
        <div className="flex items-center text-blue-500">
          <Activity size={16} className="mr-2" />
          <Text>
            {t('配置')}&nbsp;
            <a
              href="https://github.com/louislam/uptime-kuma"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 hover:underline"
            >
              Uptime&nbsp;Kuma
            </a>
            &nbsp;{t('服务监控地址，用于展示服务状态信息')}
          </Text>
        </div>

        <div className="flex gap-2">
          <Button
            icon={<Save size={14} />}
            theme='solid'
            type='primary'
            onClick={handleSave}
            loading={loading}
            className="!rounded-full"
          >
            {t('保存设置')}
          </Button>
        </div>
      </div>
    </div>
  );

  return (
    <Form.Section text={renderHeader()}>
      <Form
        layout="vertical"
        autoScrollToError
        initValues={initValues}
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
      >
        <Row gutter={[24, 24]}>
          <Col xs={24} md={12}>
            <Form.Input
              showClear
              field="uptimeKumaUrl"
              label={{ text: t("Uptime Kuma 服务地址") }}
              placeholder={t("请输入 Uptime Kuma 服务地址")}
              style={{ fontFamily: 'monospace' }}
              helpText={t("请输入 Uptime Kuma 服务的完整地址，例如：https://uptime.example.com")}
              rules={[
                {
                  validator: (_, value) => {
                    const url = (value || '').trim();

                    if (url && !isValidUrl(url)) {
                      return Promise.reject(t('请输入有效的 URL 地址'));
                    }

                    return Promise.resolve();
                  }
                }
              ]}
            />
          </Col>

          <Col xs={24} md={12}>
            <Form.Input
              showClear
              field="uptimeKumaSlug"
              label={{ text: t("状态页面 Slug") }}
              placeholder={t("请输入状态页面 Slug")}
              style={{ fontFamily: 'monospace' }}
              helpText={t("请输入状态页面的 slug 标识符，例如：my-status")}
              rules={[
                {
                  validator: (_, value) => {
                    const slug = (value || '').trim();

                    if (slug && !/^[a-zA-Z0-9_-]+$/.test(slug)) {
                      return Promise.reject(t('Slug 只能包含字母、数字、下划线和连字符'));
                    }

                    return Promise.resolve();
                  }
                }
              ]}
            />
          </Col>
        </Row>
      </Form>
    </Form.Section>
  );
};

export default SettingsUptimeKuma; 