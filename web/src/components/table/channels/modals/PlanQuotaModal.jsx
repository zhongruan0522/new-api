import React, { useState, useEffect, useCallback, useMemo } from 'react';
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
  ButtonGroup,
} from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
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

const MODEL_COLORS = ['#d97757', '#6a9bcc', '#788c5d', '#9b6db7', '#c4a44e', '#5bb8a9'];

const getStatusColor = (status) => {
  switch (status) {
    case '充裕': return 'green';
    case '适中': return 'orange';
    case '紧张': return 'red';
    default: return 'grey';
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

const PlanVersionTag = ({ version }) => {
  if (!version) return null;
  return <Tag color={version === '新' ? 'green' : 'orange'}>{version}套餐</Tag>;
};

const LimitCard = ({ title, data, resetLabel }) => {
  if (!data) return null;
  return (
    <div style={{ padding: 12, border: '1px solid var(--semi-color-border)', borderRadius: 8, marginBottom: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Text strong>{title}</Text>
        <Tag color={getStatusColor(data.status)} size='small'>{data.status}</Tag>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 20, fontWeight: 600, color: getProgressColor(data.percentage) }}>{data.percentage}%</span>
      </div>
      <Progress percent={data.percentage} stroke={getProgressColor(data.percentage)} showInfo={false} size='small' style={{ marginTop: 4 }} />
      {resetLabel && <Text type='tertiary' size='small' style={{ marginTop: 4, display: 'block' }}>{resetLabel}</Text>}
    </div>
  );
};

const McpLimitCard = ({ data }) => {
  if (!data) return null;
  return (
    <div style={{ padding: 12, border: '1px solid var(--semi-color-border)', borderRadius: 8, marginBottom: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Text strong>MCP 工具限额</Text>
        <Tag color={getStatusColor(data.status)} size='small'>{data.status}</Tag>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <span style={{ fontSize: 20, fontWeight: 600, color: getProgressColor(data.percentage) }}>{data.percentage}%</span>
        {data.current_usage && <Text type='tertiary' size='small'>{data.current_usage}</Text>}
      </div>
      <Progress percent={data.percentage} stroke={getProgressColor(data.percentage)} showInfo={false} size='small' />
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

// 时间参数生成，与 zaicontrol 保持一致（延迟10分钟，北京时间）
const getTimeParams = (days) => {
  const now = new Date();
  const end = new Date(now.getTime() - 600000);
  let start = new Date();
  if (days === 0) {
    start.setHours(0, 0, 0, 0);
  } else {
    start.setTime(now.getTime() - days * 86400000);
  }
  // 按北京时间 (UTC+8) 格式化
  const toBJ = (d) => {
    const utc = d.getTime() + d.getTimezoneOffset() * 60000;
    return new Date(utc + 8 * 3600000);
  };
  const fmt = (d) => {
    const bj = toBJ(d);
    const z = (n) => n.toString().padStart(2, '0');
    return `${bj.getFullYear()}-${z(bj.getMonth() + 1)}-${z(bj.getDate())}+${z(bj.getHours())}:${z(bj.getMinutes())}:${z(bj.getSeconds())}`;
  };
  return `startTime=${fmt(start)}&endTime=${fmt(end)}`;
};

// 将智谱原始数据转为 VChart 长格式
const flattenUsageData = (rawData, usageType) => {
  const d = rawData?.data || {};
  const x = d.x_time || [];
  if (!x.length) return { values: [], total: 0, summary: [] };

  if (usageType === 'model') {
    const totalTokens = d.totalUsage?.totalTokensUsage || 0;
    const totalArr = (d.tokensUsage || []).map((v) => v || 0);
    const modelList = d.modelDataList || [];
    const summaryList = d.modelSummaryList || d.totalUsage?.modelSummaryList || [];

    const allFields = ['总用量', ...modelList.map((m) => m.modelName)];
    const flatValues = [];
    x.forEach((t, i) => {
      // 总用量行
      flatValues.push({ time: t, value: totalArr[i] || 0, type: '总用量' });
      // 各模型行
      modelList.forEach((m) => {
        const arr = (m.tokensUsage || []).map((v) => v || 0);
        flatValues.push({ time: t, value: arr[i] || 0, type: m.modelName });
      });
    });

    return { values: flatValues, total: totalTokens, summary: summaryList, fields: allFields };
  }

  // tool mode
  const s1 = (d.networkSearchCount || []).map((v) => v || 0);
  const s2 = (d.webReadMcpCount || []).map((v) => v || 0);
  const s3 = (d.zreadMcpCount || []).map((v) => v || 0);
  const total = s1.reduce((a, b) => a + b, 0) + s2.reduce((a, b) => a + b, 0) + s3.reduce((a, b) => a + b, 0);
  const flatValues = [];
  x.forEach((t, i) => {
    flatValues.push({ time: t, value: s1[i], type: '联网搜索' });
    flatValues.push({ time: t, value: s2[i], type: '网页读取' });
    flatValues.push({ time: t, value: s3[i], type: '开源仓库' });
  });

  return { values: flatValues, total, summary: [], fields: ['联网搜索', '网页读取', '开源仓库'] };
};

const UsageChart = ({ channelId }) => {
  const [usageType, setUsageType] = useState('model');
  const [range, setRange] = useState(7);
  const [loading, setLoading] = useState(false);
  const [rawData, setRawData] = useState(null);

  const fetchUsage = useCallback(async () => {
    if (!channelId) return;
    setLoading(true);
    try {
      const params = getTimeParams(range);
      const res = await API.get(
        `/api/channel/plan/glm/usage/${channelId}?type=${usageType}&${params}`,
        { skipErrorHandler: true },
      );
      setRawData(res.data);
    } catch {
      setRawData(null);
    } finally {
      setLoading(false);
    }
  }, [channelId, usageType, range]);

  useEffect(() => {
    fetchUsage();
  }, [fetchUsage]);

  const { values, total, summary, fields } = useMemo(
    () => flattenUsageData(rawData, usageType),
    [rawData, usageType],
  );

  const vchartSpec = useMemo(() => {
    if (!values.length) return null;

    // 给不同系列分配颜色和线型
    const colorMap = {};
    fields.forEach((f, i) => {
      if (f === '总用量') {
        colorMap[f] = '#b0aea5';
      } else {
        colorMap[f] = MODEL_COLORS[i % MODEL_COLORS.length];
      }
    });

    return {
      type: 'common',
      data: [{ id: 'usage', values }],
      series: [{
        type: 'line',
        xField: 'time',
        yField: 'value',
        seriesField: 'type',
        smooth: true,
        line: { style: { lineWidth: 2 } },
        point: { visible: false },
      }],
      axes: [
        { orient: 'bottom', type: 'band', bandField: 'time', label: { style: { fontSize: 10 }, autoRotate: true, autoRotateAngle: [-45, 45] } },
        { orient: 'left', type: 'linear', field: 'value', label: { style: { fontSize: 10 } }, grid: { visible: true, style: { lineDash: [3, 3], stroke: 'var(--semi-color-border)' } } },
      ],
      color: { type: 'ordinal', range: fields.map((f) => colorMap[f]), domain: fields },
      legends: { visible: true, position: 'top', item: { label: { style: { fontSize: 11 } } }, autoPage: true, maxRow: 1 },
      tooltip: { visible: true, mark: { content: [{ key: (d) => d.type, value: (d) => d.value?.toLocaleString() }] } },
      height: 260,
      padding: { top: 10, bottom: 5, left: 10, right: 10 },
    };
  }, [values, fields]);

  const ranges = [
    { key: 0, label: '当日' },
    { key: 7, label: '7天' },
    { key: 15, label: '15天' },
    { key: 30, label: '30天' },
  ];

  return (
    <div style={{ borderTop: '1px solid var(--semi-color-border)', paddingTop: 12, marginTop: 4 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <ButtonGroup size='small'>
          <Button onClick={() => setUsageType('model')} type={usageType === 'model' ? 'primary' : 'tertiary'}>模型</Button>
          <Button onClick={() => setUsageType('tool')} type={usageType === 'tool' ? 'primary' : 'tertiary'}>工具</Button>
        </ButtonGroup>
        <div style={{ display: 'flex', gap: 4 }}>
          {ranges.map((r) => (
            <Button key={r.key} size='small' type={range === r.key ? 'primary' : 'tertiary'} onClick={() => setRange(r.key)}>
              {r.label}
            </Button>
          ))}
        </div>
      </div>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 12, marginBottom: 8, flexWrap: 'wrap' }}>
        <div>
          <Text type='tertiary' size='small'>{usageType === 'model' ? 'Tokens总量' : '工具调用次'}</Text>
          <span style={{ fontSize: 20, fontWeight: 600, marginLeft: 8 }}>{total.toLocaleString()}</span>
        </div>
        {usageType === 'model' && summary.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {summary.map((m, i) => (
              <span key={i} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, marginRight: 8 }}>
                <span style={{ width: 8, height: 8, borderRadius: '50%', background: MODEL_COLORS[(i + 1) % MODEL_COLORS.length], display: 'inline-block' }} />
                <span style={{ fontSize: 12 }}>{m.modelName}</span>
                <span style={{ fontSize: 12, color: 'var(--semi-color-text-2)' }}>{(m.totalTokens || 0).toLocaleString()}</span>
              </span>
            ))}
          </div>
        )}
      </div>
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}><Spin /></div>
      ) : vchartSpec ? (
        <VChart spec={vchartSpec} />
      ) : (
        <div style={{ textAlign: 'center', padding: '20px 0', color: 'var(--semi-color-text-2)' }}>暂无数据</div>
      )}
      <div style={{ textAlign: 'right', marginTop: 4 }}>
        <Text type='quaternary' size='small'>数据延迟约10分钟</Text>
      </div>
    </div>
  );
};

const isGlmPlan = (planName) => planName === 'glm-coding-plan' || planName === 'glm-coding-plan-international';

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

  const planDisplayName = quotaData?.plan_name ? PLAN_DISPLAY_NAMES[quotaData.plan_name] || quotaData.plan_name : '';
  const hasRealData = quotaData && isGlmPlan(quotaData.plan_name) && quotaData.product_name;

  const formatResetTime = (timeStr) => {
    if (!timeStr) return '';
    try {
      const d = new Date(timeStr);
      if (isNaN(d.getTime())) return timeStr;
      return `${d.getMonth() + 1}月${d.getDate()}日`;
    } catch { return timeStr; }
  };

  const formatHourReset = (timeStr) => {
    if (!timeStr) return '';
    try {
      const d = new Date(timeStr);
      if (isNaN(d.getTime())) return timeStr;
      return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')} 重置`;
    } catch { return timeStr; }
  };

  return (
    <Modal
      title={t('额度查询')}
      visible={visible}
      onCancel={onCancel}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button onClick={fetchQuotaData} disabled={loading}>{t('刷新')}</Button>
          <Button type='primary' onClick={onCancel}>{t('关闭')}</Button>
        </div>
      }
      width={580}
      centered
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}><Spin size='large' /></div>
      ) : hasRealData ? (
        <div style={{ padding: '8px 0' }}>
          {/* 套餐基本信息 */}
          <div style={{ padding: 16, border: '1px solid var(--semi-color-border)', borderRadius: 8, marginBottom: 12 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <Title heading={5} style={{ margin: 0 }}>{quotaData.product_name}</Title>
              <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                <PlanVersionTag version={quotaData.plan_version} />
                <LevelTag level={quotaData.product_level} />
              </div>
            </div>
            <Descriptions row size='small' style={{ marginTop: 4 }}>
              <Descriptions.Item itemKey='生效'><Text size='small'>{quotaData.effective_date || '-'}</Text></Descriptions.Item>
              <Descriptions.Item itemKey='到期'><Text size='small'>{quotaData.expiry_date || '-'}</Text></Descriptions.Item>
            </Descriptions>
            <div style={{ marginTop: 4, borderTop: '1px solid var(--semi-color-border)', paddingTop: 8 }}>
              <Text type={quotaData.auto_renew ? 'success' : 'warning'} size='small'>
                {quotaData.auto_renew ? '✅ 自动续费' : '⚠️ 未开续费'}
              </Text>
            </div>
          </div>

          {/* 限额卡片 */}
          {quotaData.weekly_limit && (
            <LimitCard title='每周限额' data={quotaData.weekly_limit}
              resetLabel={formatResetTime(quotaData.weekly_limit.next_reset_time) ? `下次重置: ${formatResetTime(quotaData.weekly_limit.next_reset_time)}` : ''} />
          )}
          {quotaData.token_limit && (
            <LimitCard title='每5小时限额' data={quotaData.token_limit}
              resetLabel={formatHourReset(quotaData.token_limit.next_reset_time)} />
          )}
          <McpLimitCard data={quotaData.mcp_tool_limit} />

          {/* 用量图表 */}
          <UsageChart channelId={channel?.id} />
        </div>
      ) : (
        <div style={{ padding: '16px 0' }}>
          <Empty
            image={<IllustrationNoResult style={{ width: 120, height: 120 }} />}
            darkModeImage={<IllustrationNoResultDark style={{ width: 120, height: 120 }} />}
            title={t('即将支持')}
            description={
              planDisplayName
                ? t('${plan} 的额度查询功能即将上线，敬请期待。').replace('${plan}', planDisplayName)
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
