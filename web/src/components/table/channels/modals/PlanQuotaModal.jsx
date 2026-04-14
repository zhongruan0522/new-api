import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Spin,
  Typography,
  Empty,
  Button,
  Tag,
  Descriptions,
  Progress,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, showError } from '../../../../helpers';

const { Title, Text } = Typography;

const PLAN_DISPLAY_NAMES = {
  'glm-coding-plan': '智谱 GLM 官方套餐',
  'glm-coding-plan-international': '智谱 GLM 国际套餐',
  'kimi-coding-plan': 'Kimi 官方套餐',
  'minimax-coding-plan': 'MiniMax 官方套餐',
  'minimax-coding-plan-international': 'MiniMax 国际套餐',
};

const getStatusColor = (status) => {
  switch (status) {
    case '充裕':
      return 'green';
    case '适中':
      return 'orange';
    case '紧张':
      return 'red';
    default:
      return 'grey';
  }
};

const getProgressColor = (percentage) => {
  if (percentage >= 80) return '#ef4444';
  if (percentage >= 50) return '#f97316';
  return '#22c55e';
};

const LevelTag = ({ level }) => {
  const colorMap = { Lite: 'cyan', Pro: 'blue', Max: 'violet', Standard: 'grey' };
  return <Tag color={colorMap[level] || 'grey'}>{level}</Tag>;
};

const LimitCard = ({ title, data, resetLabel }) => {
  if (!data) return null;
  return (
    <div style={{
      padding: 12,
      border: '1px solid var(--semi-color-border)',
      borderRadius: 8,
      marginBottom: 12,
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Text strong>{title}</Text>
        <Tag color={getStatusColor(data.status)} size='small'>{data.status}</Tag>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 20, fontWeight: 600, color: getProgressColor(data.percentage) }}>
          {data.percentage}%
        </span>
      </div>
      <Progress
        percent={data.percentage}
        stroke={getProgressColor(data.percentage)}
        showInfo={false}
        size='small'
        style={{ marginTop: 4 }}
      />
      {resetLabel && <Text type='tertiary' size='small' style={{ marginTop: 4, display: 'block' }}>{resetLabel}</Text>}
    </div>
  );
};

const McpLimitCard = ({ data }) => {
  if (!data) return null;
  return (
    <div style={{
      padding: 12,
      border: '1px solid var(--semi-color-border)',
      borderRadius: 8,
      marginBottom: 12,
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Text strong>MCP 工具限额</Text>
        <Tag color={getStatusColor(data.status)} size='small'>{data.status}</Tag>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <span style={{ fontSize: 20, fontWeight: 600, color: getProgressColor(data.percentage) }}>
          {data.percentage}%
        </span>
        {data.current_usage && (
          <Text type='tertiary' size='small'>{data.current_usage}</Text>
        )}
      </div>
      <Progress
        percent={data.percentage}
        stroke={getProgressColor(data.percentage)}
        showInfo={false}
        size='small'
      />
      <Text type='tertiary' size='small' style={{ marginTop: 4, display: 'block' }}>每月1号重置</Text>
      {data.tools && data.tools.length > 0 && (
        <div style={{ marginTop: 8, borderTop: '1px solid var(--semi-color-border)', paddingTop: 8 }}>
          {data.tools.map((tool, i) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '2px 0' }}>
              <Text type='tertiary' size='small'>{tool.name}</Text>
              <Text size='small'>{tool.usage}</Text>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const isGlmPlan = (planName) => {
  return planName === 'glm-coding-plan' || planName === 'glm-coding-plan-international';
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

  // 是否从后端拿到真实的 GLM 额度数据
  const hasRealData = quotaData && isGlmPlan(quotaData.plan_name) && quotaData.product_name;

  // 格式化重置时间
  const formatResetTime = (timeStr) => {
    if (!timeStr) return '';
    try {
      const d = new Date(timeStr);
      if (isNaN(d.getTime())) return timeStr;
      return `${d.getMonth() + 1}月${d.getDate()}日`;
    } catch {
      return timeStr;
    }
  };

  const formatHourReset = (timeStr) => {
    if (!timeStr) return '';
    try {
      const d = new Date(timeStr);
      if (isNaN(d.getTime())) return timeStr;
      return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')} 重置`;
    } catch {
      return timeStr;
    }
  };

  return (
    <Modal
      title={t('额度查询')}
      visible={visible}
      onCancel={onCancel}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button onClick={fetchQuotaData} disabled={loading}>
            {t('刷新')}
          </Button>
          <Button type='primary' onClick={onCancel}>
            {t('关闭')}
          </Button>
        </div>
      }
      width={520}
      centered
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <Spin size='large' />
        </div>
      ) : hasRealData ? (
        <div style={{ padding: '8px 0' }}>
          {/* 套餐基本信息 */}
          <div style={{
            padding: 16,
            border: '1px solid var(--semi-color-border)',
            borderRadius: 8,
            marginBottom: 12,
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <Title heading={5} style={{ margin: 0 }}>{quotaData.product_name}</Title>
              <LevelTag level={quotaData.product_level} />
            </div>
            <Descriptions
              row
              size='small'
              style={{ marginTop: 4 }}
            >
              <Descriptions.Item itemKey='生效'>
                <Text size='small'>{quotaData.effective_date || '-'}</Text>
              </Descriptions.Item>
              <Descriptions.Item itemKey='到期'>
                <Text size='small'>{quotaData.expiry_date || '-'}</Text>
              </Descriptions.Item>
            </Descriptions>
            <div style={{ marginTop: 4, borderTop: '1px solid var(--semi-color-border)', paddingTop: 8 }}>
              <Text type={quotaData.auto_renew ? 'success' : 'warning'} size='small'>
                {quotaData.auto_renew ? '✅ 自动续费' : '⚠️ 未开续费'}
              </Text>
            </div>
          </div>

          {/* 限额卡片 */}
          {quotaData.weekly_limit && (
            <LimitCard
              title='每周限额'
              data={quotaData.weekly_limit}
              resetLabel={formatResetTime(quotaData.weekly_limit.next_reset_time) ? `下次重置: ${formatResetTime(quotaData.weekly_limit.next_reset_time)}` : ''}
            />
          )}
          {quotaData.token_limit && (
            <LimitCard
              title='每5小时限额'
              data={quotaData.token_limit}
              resetLabel={formatHourReset(quotaData.token_limit.next_reset_time)}
            />
          )}
          <McpLimitCard data={quotaData.mcp_tool_limit} />
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
