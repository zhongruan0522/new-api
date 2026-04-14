import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Spin, Typography, Empty, Button } from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, showError } from '../../../../helpers';

const { Title, Paragraph } = Typography;

// 套餐名称到友好显示名称的映射
const PLAN_DISPLAY_NAMES = {
  'glm-coding-plan': '智谱 GLM 官方套餐',
  'glm-coding-plan-international': '智谱 GLM 国际套餐',
  'kimi-coding-plan': 'Kimi 官方套餐',
  'minimax-coding-plan': 'MiniMax 官方套餐',
  'minimax-coding-plan-international': 'MiniMax 国际套餐',
};

const PlanQuotaModal = ({ visible, onCancel, channel, onRefresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [quotaData, setQuotaData] = useState(null);

  const fetchQuotaData = useCallback(async () => {
    if (!channel?.id) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/channel/plan/quota/${channel.id}`);
      const { success, message, data } = res.data;
      if (success) {
        setQuotaData(data);
      } else {
        showError(message || t('查询额度失败'));
      }
    } catch (error) {
      showError(error.message || t('网络错误'));
    } finally {
      setLoading(false);
    }
  }, [channel?.id, t]);

  useEffect(() => {
    if (visible && channel?.id) {
      fetchQuotaData();
    }
    if (!visible) {
      setQuotaData(null);
    }
  }, [visible, channel?.id, fetchQuotaData]);

  const planDisplayName =
    quotaData?.plan_name
      ? PLAN_DISPLAY_NAMES[quotaData.plan_name] || quotaData.plan_name
      : '';

  // 判断是否支持额度查询
  const isQuotaSupported = quotaData?.quota_supported === true;

  return (
    <Modal
      title={t('额度查询')}
      visible={visible}
      onCancel={onCancel}
      footer={
        <Button type='primary' onClick={onCancel}>
          {t('关闭')}
        </Button>
      }
      width={500}
      centered
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <Spin size='large' />
        </div>
      ) : isQuotaSupported ? (
        <div style={{ padding: '16px 0' }}>
          <Title heading={5} style={{ marginBottom: 12 }}>
            {planDisplayName}
          </Title>
          <Paragraph type='tertiary' style={{ marginBottom: 8 }}>
            {t('渠道')}：{quotaData?.channel_name || '-'}
          </Paragraph>
          <Paragraph type='tertiary'>
            {t('该套餐的额度查询功能即将上线，敬请期待。')}
          </Paragraph>
        </div>
      ) : (
        <div style={{ padding: '16px 0' }}>
          <Empty
            image={
              <IllustrationNoResult style={{ width: 120, height: 120 }} />
            }
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 120, height: 120 }} />
            }
            title={t('即将支持')}
            description={
              planDisplayName
                ? t('${plan} 的额度查询功能即将上线，敬请期待。').replace(
                    '${plan}',
                    planDisplayName,
                  )
                : t('该套餐的额度查询功能即将上线，敬请期待。')
            }
            style={{ padding: 20 }}
          />
        </div>
      )}
    </Modal>
  );
};

export default PlanQuotaModal;
