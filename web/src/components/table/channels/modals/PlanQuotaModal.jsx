import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
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
  'ollama-coding-plan': 'Ollama 官方套餐',
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

const formatCompactNumber = (num) => {
  if (num == null || isNaN(num)) return '0';
  const abs = Math.abs(num);
  const sign = num < 0 ? '-' : '';
  if (abs >= 1_000_000_000) return sign + (abs / 1_000_000_000).toFixed(1).replace(/\.0$/, '') + 'B';
  if (abs >= 1_000_000) return sign + (abs / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M';
  if (abs >= 1_000) return sign + (abs / 1_000).toFixed(1).replace(/\.0$/, '') + 'K';
  return sign + abs.toString();
};

const LimitCard = ({ title, data, resetLabel }) => {
  if (!data) return null;
  return (
    <div className="plan-quota-card">
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
    <div className="plan-quota-card">
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

const getTimeParams = (days) => {
  const now = new Date();
  const end = new Date(now.getTime() - 600000);
  let start = new Date();
  if (days === 0) {
    start.setHours(0, 0, 0, 0);
  } else {
    start.setTime(now.getTime() - days * 86400000);
  }
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

const formatTimeLabel = (timeStr) => {
  if (!timeStr) return '';
  // 尝试解析 "2026-01-15 10:00" 或 "2026-01-15" 格式
  const match = timeStr.match(/^(\d{4})-(\d{2})-(\d{2})\s*(.*)?$/);
  if (match) {
    const date = `${match[2]}-${match[3]}`;
    const time = match[4] || '';
    if (time) {
      // 提取 HH:MM
      const timeMatch = time.match(/^(\d{2}):(\d{2})/);
      if (timeMatch) {
        return `${date} ${timeMatch[1]}:${timeMatch[2]}`;
      }
    }
    return date;
  }
  return timeStr;
};

// 均匀采样时间标签，保留最多 maxLabels 个
const sampleTimeLabels = (times, maxLabels = 4) => {
  if (!times || times.length <= maxLabels) return times;
  const step = (times.length - 1) / (maxLabels - 1);
  const result = [];
  for (let i = 0; i < maxLabels; i++) {
    const idx = Math.round(i * step);
    result.push(times[idx]);
  }
  return result;
};

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
      flatValues.push({ time: t, value: totalArr[i] || 0, type: '总用量' });
      modelList.forEach((m) => {
        const arr = (m.tokensUsage || []).map((v) => v || 0);
        flatValues.push({ time: t, value: arr[i] || 0, type: m.modelName });
      });
    });

    return { values: flatValues, total: totalTokens, summary: summaryList, fields: allFields, times: x };
  }

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

  return { values: flatValues, total, summary: [], fields: ['联网搜索', '网页读取', '开源仓库'], times: x };
};

const buildYAxisFormatter = (maxValue) => {
  if (maxValue >= 1_000_000) return (v) => formatCompactNumber(v);
  if (maxValue >= 1_000) return (v) => formatCompactNumber(v);
  return (v) => v?.toString();
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

  const { values, total, summary, fields, times } = useMemo(
    () => flattenUsageData(rawData, usageType),
    [rawData, usageType],
  );

  const vchartSpec = useMemo(() => {
    if (!values.length) return null;

    const colorMap = {};
    fields.forEach((f, i) => {
      if (f === '总用量') {
        colorMap[f] = '#b0aea5';
      } else {
        colorMap[f] = MODEL_COLORS[i % MODEL_COLORS.length];
      }
    });

    // 计算最大值用于格式化
    const maxVal = Math.max(...values.map((v) => v.value || 0));
    const tickCount = 5;
    const tickStep = maxVal > 0 ? maxVal / (tickCount - 1) : 1;

    // 生成均匀采样的时间标签索引
    const sampledLabels = sampleTimeLabels(times, 4);
    const allTimes = times || [];

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
        {
          orient: 'bottom',
          type: 'band',
          bandField: 'time',
          label: {
            style: { fontSize: 11 },
            autoRotate: false,
            formatter: (v) => {
              if (sampledLabels.includes(v)) return formatTimeLabel(v);
              return '';
            },
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          type: 'linear',
          field: 'value',
          tick: { count: tickCount },
          label: {
            style: { fontSize: 10 },
            formatter: (v) => formatCompactNumber(v),
          },
          grid: { visible: true, style: { lineDash: [3, 3], stroke: 'var(--semi-color-border)' } },
        },
      ],
      color: { type: 'ordinal', range: fields.map((f) => colorMap[f]), domain: fields },
      legends: { visible: true, position: 'top', item: { label: { style: { fontSize: 11 } } }, autoPage: true, maxRow: 1 },
      tooltip: { visible: true, mark: { content: [{ key: (d) => d.type, value: (d) => (d.value ?? 0).toLocaleString() }] } },
      height: 240,
      padding: { top: 10, bottom: 5, left: 10, right: 10 },
    };
  }, [values, fields, times]);

  const ranges = [
    { key: 0, label: '当日' },
    { key: 7, label: '7天' },
    { key: 15, label: '15天' },
    { key: 30, label: '30天' },
  ];

  return (
    <div className="plan-quota-section">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 8 }}>
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
          <span style={{ fontSize: 20, fontWeight: 600, marginLeft: 8 }}>{formatCompactNumber(total)}</span>
        </div>
        {usageType === 'model' && summary.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
            {summary.map((m, i) => (
              <span key={i} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, marginRight: 8 }}>
                <span style={{ width: 8, height: 8, borderRadius: '50%', background: MODEL_COLORS[(i + 1) % MODEL_COLORS.length], display: 'inline-block' }} />
                <span style={{ fontSize: 12 }}>{m.modelName}</span>
                <span style={{ fontSize: 12, color: 'var(--semi-color-text-2)' }}>{formatCompactNumber(m.totalTokens || 0)}</span>
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

// 系统健康度图表
const PerformanceChart = ({ channelId }) => {
  const [range, setRange] = useState(7);
  const [loading, setLoading] = useState(false);
  const [rawData, setRawData] = useState(null);

  const fetchPerf = useCallback(async () => {
    if (!channelId) return;
    setLoading(true);
    try {
      const params = getTimeParams(range);
      const res = await API.get(
        `/api/channel/plan/glm/usage/${channelId}?type=performance&${params}`,
        { skipErrorHandler: true },
      );
      setRawData(res.data);
    } catch {
      setRawData(null);
    } finally {
      setLoading(false);
    }
  }, [channelId, range]);

  useEffect(() => {
    fetchPerf();
  }, [fetchPerf]);

  const { perfValues, avgSpeed, avgRate, perfTimes } = useMemo(() => {
    const d = rawData?.data || {};
    const x = d.x_time || [];
    if (!x.length) return { perfValues: [], avgSpeed: '--', avgRate: '--', perfTimes: [] };

    const liteSpeed = (d.liteDecodeSpeed || []).map((v) => (v ? parseFloat(v.toFixed(2)) : 0));
    const proMaxSpeed = (d.proMaxDecodeSpeed || []).map((v) => (v ? parseFloat(v.toFixed(2)) : 0));
    const liteRate = (d.liteSuccessRate || []).map((v) => (v ? parseFloat((v * 100).toFixed(2)) : 0));
    const proMaxRate = (d.proMaxSuccessRate || []).map((v) => (v ? parseFloat((v * 100).toFixed(2)) : 0));

    const flatValues = [];
    x.forEach((t, i) => {
      flatValues.push({ time: t, value: liteSpeed[i], type: 'Lite速度' });
      flatValues.push({ time: t, value: proMaxSpeed[i], type: 'Pro/Max速度' });
      flatValues.push({ time: t, value: liteRate[i], type: 'Lite成功率' });
      flatValues.push({ time: t, value: proMaxRate[i], type: 'Pro/Max成功率' });
    });

    const avg = (arr) => (arr.length ? (arr.reduce((a, b) => a + b, 0) / arr.length).toFixed(1) : 0);
    const speed = ((parseFloat(avg(liteSpeed)) + parseFloat(avg(proMaxSpeed))) / 2).toFixed(1);
    const rate = ((parseFloat(avg(liteRate)) + parseFloat(avg(proMaxRate))) / 2).toFixed(1);

    return { perfValues: flatValues, avgSpeed: speed, avgRate: rate, perfTimes: x };
  }, [rawData]);

  const vchartSpec = useMemo(() => {
    if (!perfValues.length) return null;

    const sampledLabels = sampleTimeLabels(perfTimes, 4);

    const colorMap = {
      'Lite速度': '#d97757',
      'Pro/Max速度': '#6a9bcc',
      'Lite成功率': '#d97757',
      'Pro/Max成功率': '#788c5d',
    };
    const fields = ['Lite速度', 'Pro/Max速度', 'Lite成功率', 'Pro/Max成功率'];

    return {
      type: 'common',
      data: [{ id: 'perf', values: perfValues }],
      series: [
        {
          type: 'line',
          xField: 'time',
          yField: 'value',
          seriesField: 'type',
          smooth: true,
          line: {
            style: {
              lineWidth: (d) => (d.type.includes('成功率') ? 2 : 2),
              lineDash: (d) => (d.type.includes('成功率') ? [4, 4] : [0]),
            },
          },
          point: { visible: false },
        },
      ],
      axes: [
        {
          orient: 'bottom',
          type: 'band',
          bandField: 'time',
          label: {
            style: { fontSize: 11 },
            autoRotate: false,
            formatter: (v) => {
              if (sampledLabels.includes(v)) return formatTimeLabel(v);
              return '';
            },
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          type: 'linear',
          field: 'value',
          label: {
            style: { fontSize: 10 },
            formatter: (v) => formatCompactNumber(v),
          },
          grid: { visible: true, style: { lineDash: [3, 3], stroke: 'var(--semi-color-border)' } },
        },
      ],
      color: { type: 'ordinal', range: fields.map((f) => colorMap[f]), domain: fields },
      legends: { visible: true, position: 'top', item: { label: { style: { fontSize: 11 } } }, autoPage: true, maxRow: 1 },
      tooltip: {
        visible: true,
        mark: {
          content: [{
            key: (d) => d.type,
            value: (d) => d.type.includes('成功率') ? `${d.value?.toFixed(1)}%` : `${d.value?.toFixed(1)} tokens/s`,
          }],
        },
      },
      height: 240,
      padding: { top: 10, bottom: 5, left: 10, right: 10 },
    };
  }, [perfValues, perfTimes]);

  const ranges = [
    { key: 7, label: '7天' },
    { key: 15, label: '15天' },
    { key: 30, label: '30天' },
  ];

  return (
    <div className="plan-quota-section">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 8 }}>
        <Text strong style={{ fontSize: 14 }}>系统健康度</Text>
        <div style={{ display: 'flex', gap: 4 }}>
          {ranges.map((r) => (
            <Button key={r.key} size='small' type={range === r.key ? 'primary' : 'tertiary'} onClick={() => setRange(r.key)}>
              {r.label}
            </Button>
          ))}
        </div>
      </div>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 24, marginBottom: 8 }}>
        <div>
          <Text type='tertiary' size='small'>平均速度</Text>
          <span style={{ fontSize: 20, fontWeight: 600, marginLeft: 8 }}>{avgSpeed}</span>
          <Text type='tertiary' size='small'> tokens/s</Text>
        </div>
        <div>
          <Text type='tertiary' size='small'>成功率</Text>
          <span style={{ fontSize: 20, fontWeight: 600, marginLeft: 8 }}>{avgRate}%</span>
        </div>
      </div>
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}><Spin /></div>
      ) : vchartSpec ? (
        <VChart spec={vchartSpec} />
      ) : (
        <div style={{ textAlign: 'center', padding: '20px 0', color: 'var(--semi-color-text-2)' }}>暂无数据</div>
      )}
    </div>
  );
};

const isGlmPlan = (planName) => planName === 'glm-coding-plan' || planName === 'glm-coding-plan-international';
const isKimiPlan = (planName) => planName === 'kimi-coding-plan';
const isMiniMaxPlan = (planName) => planName === 'minimax-coding-plan' || planName === 'minimax-coding-plan-international';
const isTierBasedPlan = (planName) => isKimiPlan(planName) || isMiniMaxPlan(planName);

const PlanQuotaModal = ({ visible, onCancel, channel, onRefresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [quotaData, setQuotaData] = useState(null);
  const bodyRef = useRef(null);

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
  const isGlmData = quotaData && isGlmPlan(quotaData.plan_name) && quotaData.product_name;
  const isTierData = quotaData && isTierBasedPlan(quotaData.plan_name);
  const hasRealData = isGlmData || isTierData;

  // 根据实际可见的卡片数量动态决定列数
  const visibleCardCount = [
    quotaData?.token_limit,
    quotaData?.weekly_limit,
    quotaData?.mcp_tool_limit,
  ].filter(Boolean).length || 2;

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
      width='50vw'
      centered
      style={{ minWidth: 360, maxWidth: 900 }}
      bodyStyle={{ maxHeight: 'calc(85vh - 120px)', overflowY: 'auto', padding: '16px 20px' }}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}><Spin size='large' /></div>
      ) : isGlmData ? (
        <div className="plan-quota-body" ref={bodyRef}>
          {/* 套餐基本信息 */}
          <div className="plan-quota-info-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 8 }}>
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

          {/* 限额卡片网格 */}
          <div className={`plan-quota-grid plan-quota-grid-${visibleCardCount}`}>
            {quotaData.token_limit && (
              <LimitCard title='每5小时限额' data={quotaData.token_limit}
                resetLabel={formatHourReset(quotaData.token_limit.next_reset_time)} />
            )}
            {quotaData.weekly_limit && (
              <LimitCard title='每周限额' data={quotaData.weekly_limit}
                resetLabel={formatResetTime(quotaData.weekly_limit.next_reset_time) ? `下次重置: ${formatResetTime(quotaData.weekly_limit.next_reset_time)}` : ''} />
            )}
            {quotaData.mcp_tool_limit && (
              <McpLimitCard data={quotaData.mcp_tool_limit} />
            )}
          </div>

          {/* 用量图表 */}
          <UsageChart channelId={channel?.id} />

          {/* 系统健康度图表 */}
          <PerformanceChart channelId={channel?.id} />
        </div>
      ) : isTierData ? (
        <div className="plan-quota-body" ref={bodyRef}>
          {/* 凭证状态提示 */}
          {quotaData.credential === 'expired' && (
            <div style={{ padding: '12px 16px', marginBottom: 12, borderRadius: 8, background: 'var(--semi-color-warning-light-default)', border: '1px solid var(--semi-color-warning-light-hover)' }}>
              <Text type='warning'>⚠️ API Key 无效或已过期，请检查渠道配置</Text>
            </div>
          )}
          {quotaData.credential === 'error' && (
            <div style={{ padding: '12px 16px', marginBottom: 12, borderRadius: 8, background: 'var(--semi-color-danger-light-default)', border: '1px solid var(--semi-color-danger-light-hover)' }}>
              <Text type='danger'>❌ 响应解析失败，API 格式可能已变更</Text>
            </div>
          )}

          {/* 套餐标题 */}
          <div className="plan-quota-info-card">
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Title heading={5} style={{ margin: 0 }}>{planDisplayName}</Title>
              {quotaData.credential === 'valid' && (
                <Tag color='green' size='small'>有效</Tag>
              )}
            </div>
          </div>

          {/* 限额卡片 */}
          {quotaData.tiers && quotaData.tiers.length > 0 && (
            <div className={`plan-quota-grid plan-quota-grid-${quotaData.tiers.length}`}>
              {quotaData.tiers.map((tier, idx) => {
                const title = tier.name === 'five_hour' ? '每5小时限额' : '每周限额';
                const resetLabel = tier.resets_at
                  ? (tier.name === 'five_hour' ? formatHourReset(tier.resets_at) : (formatResetTime(tier.resets_at) ? `下次重置: ${formatResetTime(tier.resets_at)}` : ''))
                  : '';
                const limitData = {
                  percentage: tier.percentage,
                  status: tier.status,
                  next_reset_time: tier.resets_at,
                };
                return (
                  <div key={idx}>
                    <LimitCard title={title} data={limitData} resetLabel={resetLabel} />
                    <div style={{ marginTop: 6, display: 'flex', justifyContent: 'space-between', fontSize: 12, color: 'var(--semi-color-text-2)' }}>
                      <span>已用 {formatCompactNumber(tier.used)}</span>
                      <span>剩余 {formatCompactNumber(tier.remaining)} / {formatCompactNumber(tier.limit)}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {/* 无限额数据 */}
          {(!quotaData.tiers || quotaData.tiers.length === 0) && quotaData.credential === 'valid' && (
            <div style={{ textAlign: 'center', padding: '20px 0', color: 'var(--semi-color-text-2)' }}>
              暂无限额数据
            </div>
          )}
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
