import React, { useState, useCallback } from 'react';
import {
  Input,
  Button,
  Table,
  Tag,
  Spin,
  Card,
  Space,
  Typography,
  Tooltip,
  Descriptions,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconCopy,
  IconDownload,
} from '@douyinfe/semi-icons';
import { Toast, Modal } from '@douyinfe/semi-ui';
import { timestamp2string, copy, renderQuota, stringToColor } from '../../helpers';
import Papa from 'papaparse';

const { Text } = Typography;

const ITEMS_PER_PAGE = 20;

// 日志类型常量
const LOG_TYPE_CONSUME = 2;
const LOG_TYPE_ERROR = 5;
const LOG_TYPE_REFUND = 6;

// 判断是否为 API 调用相关日志（消费、错误、退款）
function isApiCallLog(record) {
  return record.type === LOG_TYPE_CONSUME || record.type === LOG_TYPE_ERROR || record.type === LOG_TYPE_REFUND;
}

function renderLogType(type) {
  switch (type) {
    case LOG_TYPE_CONSUME:
      return <Tag color='green' size='small'>消费</Tag>;
    case LOG_TYPE_ERROR:
      return <Tag color='red' size='small'>错误</Tag>;
    case LOG_TYPE_REFUND:
      return <Tag color='blue' size='small'>退款</Tag>;
    default:
      return null;
  }
}

function renderIsStream(bool) {
  if (bool === undefined || bool === null) return null;
  return bool ? (
    <Tag color='blue' size='small'>
      流
    </Tag>
  ) : (
    <Tag color='purple' size='small'>
      非流
    </Tag>
  );
}

function renderUseTime(timeMs) {
  if (!timeMs) return '-';
  const seconds = (parseInt(timeMs) / 1000).toFixed(1);
  if (seconds < 101) {
    return (
      <Tag color='green' size='small'>
        {seconds}s
      </Tag>
    );
  } else if (seconds < 300) {
    return (
      <Tag color='orange' size='small'>
        {seconds}s
      </Tag>
    );
  } else {
    return (
      <Tag color='red' size='small'>
        {seconds}s
      </Tag>
    );
  }
}

const KeyQuery = () => {
  const [key, setKey] = useState('');
  const [balance, setBalance] = useState(null);
  const [usage, setUsage] = useState(null);
  const [expiredTime, setExpiredTime] = useState(null);
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [tokenValid, setTokenValid] = useState(false);
  const [totalQuota, setTotalQuota] = useState(0);

  const resetData = useCallback(() => {
    setBalance(null);
    setUsage(null);
    setExpiredTime(null);
    setLogs([]);
    setTokenValid(false);
    setTotalQuota(0);
  }, []);

  const fetchData = useCallback(async () => {
    if (!key.trim()) {
      Toast.warning('请先输入令牌，再进行查询');
      return;
    }
    if (!/^sk-[a-zA-Z0-9]{48}$/.test(key.trim())) {
      Toast.error('令牌格式非法！正确格式为 sk- 后跟48位字母数字');
      return;
    }

    setLoading(true);
    resetData();

    const headers = { Authorization: `Bearer ${key.trim()}` };

    try {
      // 1. Fetch subscription (balance info)
      const subRes = await fetch('/v1/dashboard/billing/subscription', {
        headers,
      });
      if (!subRes.ok) {
        throw new Error('令牌无效或已过期');
      }
      const subData = await subRes.json();

      if (subData.error) {
        throw new Error(subData.error.message || '查询失败');
      }

      setBalance(subData.hard_limit_usd);
      setExpiredTime(subData.access_until);
      setTokenValid(true);

      // 2. Fetch usage
      const now = new Date();
      const start = new Date(now.getTime() - 100 * 24 * 3600 * 1000);
      const startDate = `${start.getFullYear()}-${start.getMonth() + 1}-${start.getDate()}`;
      const endDate = `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}`;

      const usageRes = await fetch(
        `/v1/dashboard/billing/usage?start_date=${startDate}&end_date=${endDate}`,
        { headers },
      );
      if (usageRes.ok) {
        const usageData = await usageRes.json();
        if (!usageData.error) {
          setUsage(usageData.total_usage / 100);
        }
      }

      // 3. Fetch logs
      const logRes = await fetch(`/api/log/token`, { headers });
      if (logRes.ok) {
        const logData = await logRes.json();
        if (logData.success) {
          const logItems = logData.data || [];
          setLogs(logItems);
          let quotaSum = 0;
          for (const item of logItems) {
            if (item.type === LOG_TYPE_CONSUME || item.type === LOG_TYPE_REFUND) {
              quotaSum += item.quota || 0;
            }
          }
          setTotalQuota(quotaSum);
        }
      }
    } catch (e) {
      Toast.error(e.message || '查询失败，请输入正确的令牌');
      resetData();
    } finally {
      setLoading(false);
    }
  }, [key, resetData]);

  const copyText = useCallback(async (text) => {
    if (await copy(text)) {
      Toast.success('已复制：' + text);
    } else {
      Modal.error({
        title: '无法复制到剪贴板，请手动复制',
        content: text,
      });
    }
  }, []);

  const copyTokenInfo = useCallback(() => {
    const isUnlimited = balance === 100000000;
    const info = [
      `令牌总额: ${isUnlimited ? '无限' : `$${balance?.toFixed(3)}`}`,
      `已用额度: ${isUnlimited ? '不进行计算' : usage !== null ? `$${usage.toFixed(3)}` : '未知'}`,
      `剩余额度: ${isUnlimited ? '无限制' : balance !== null && usage !== null ? `$${(balance - usage).toFixed(3)}` : '未知'}`,
      `有效期至: ${expiredTime === 0 ? '永不过期' : expiredTime ? timestamp2string(expiredTime) : '未知'}`,
    ].join('\n');
    copyText(info);
  }, [balance, usage, expiredTime, copyText]);

  const exportCSV = useCallback(() => {
    const csvData = logs.map((log) => ({
      时间: timestamp2string(log.created_at),
      类型: log.type === LOG_TYPE_CONSUME ? '消费' : log.type === LOG_TYPE_ERROR ? '错误' : log.type === LOG_TYPE_REFUND ? '退款' : '其他',
      模型: log.model_name || '-',
      用时: log.use_time ? `${(log.use_time / 1000).toFixed(1)}s` : '-',
      流式: log.is_stream ? '是' : '否',
      提示tokens: log.prompt_tokens || 0,
      补全tokens: log.completion_tokens || 0,
      花费: renderQuota(log.quota),
    }));
    const csvString = '\ufeff' + Papa.unparse(csvData, { escapeFormulae: true });
    const blob = new Blob([csvString], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', `key-usage-${Date.now()}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [logs]);

  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      render: (text) => timestamp2string(text),
      sorter: (a, b) => a.created_at - b.created_at,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 70,
      render: (text) => renderLogType(text),
      filters: [
        { text: '消费', value: LOG_TYPE_CONSUME },
        { text: '错误', value: LOG_TYPE_ERROR },
        { text: '退款', value: LOG_TYPE_REFUND },
      ],
      onFilter: (value, record) => record.type === value,
    },
    {
      title: '模型',
      dataIndex: 'model_name',
      render: (text) => {
        if (!text) return <Text type='weak'>-</Text>;
        return (
          <Tag
            color={stringToColor(text)}
            size='small'
            onClick={() => copyText(text)}
            style={{ cursor: 'pointer' }}
          >
            {text}
          </Tag>
        );
      },
      sorter: (a, b) => ('' + a.model_name).localeCompare(b.model_name),
    },
    {
      title: '用时',
      dataIndex: 'use_time',
      width: 130,
      render: (text, record) => {
        if (!isApiCallLog(record) || !text) return '-';
        return (
          <Space>
            {renderUseTime(text)}
            {renderIsStream(record.is_stream)}
          </Space>
        );
      },
      sorter: (a, b) => a.use_time - b.use_time,
    },
    {
      title: '提示',
      dataIndex: 'prompt_tokens',
      width: 70,
      render: (text, record) =>
        isApiCallLog(record) && text ? text : '-',
      sorter: (a, b) => a.prompt_tokens - b.prompt_tokens,
    },
    {
      title: '补全',
      dataIndex: 'completion_tokens',
      width: 70,
      render: (text, record) =>
        isApiCallLog(record) && parseInt(text) > 0 ? text : '-',
      sorter: (a, b) => a.completion_tokens - b.completion_tokens,
    },
    {
      title: '花费',
      dataIndex: 'quota',
      width: 100,
      render: (text, record) => {
        if (!isApiCallLog(record)) return '-';
        return renderQuota(text);
      },
      sorter: (a, b) => a.quota - b.quota,
    },
  ];

  const isUnlimited = balance === 100000000;

  return (
    <div className='flex flex-col gap-4'>
      {/* Search Input */}
      <Card bodyStyle={{ padding: '16px' }}>
        <Input
          showClear
          value={key}
          onChange={setKey}
          placeholder='请输入要查询的令牌（sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx）'
          prefix={<IconSearch />}
          suffix={
            <Button onClick={fetchData} loading={loading}>
              查询
            </Button>
          }
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              fetchData();
            }
          }}
        />
      </Card>

      {/* Token Info */}
      {tokenValid && (
        <Card
          title='令牌信息'
          headerExtraContent={
            <Space>
              <Tooltip content='复制令牌信息'>
                <Button
                  icon={<IconCopy />}
                  theme='borderless'
                  type='primary'
                  onClick={copyTokenInfo}
                />
              </Tooltip>
            </Space>
          }
        >
          <Spin spinning={loading}>
            <Descriptions
              align='left'
              size='small'
              row
            >
              <Descriptions.Item itemKey='令牌总额'>
                <Text>
                  {isUnlimited ? '无限' : balance !== null ? `$${balance.toFixed(3)}` : '未知'}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item itemKey='已用额度'>
                <Text>
                  {isUnlimited
                    ? '不进行计算'
                    : usage !== null
                      ? `$${usage.toFixed(3)}`
                      : '未知'}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item itemKey='剩余额度'>
                <Text>
                  {isUnlimited
                    ? '无限制'
                    : balance !== null && usage !== null
                      ? `$${(balance - usage).toFixed(3)}`
                      : '未知'}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item itemKey='有效期至'>
                <Text>
                  {expiredTime === 0
                    ? '永不过期'
                    : expiredTime
                      ? timestamp2string(expiredTime)
                      : '未知'}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item itemKey='近期调用花费'>
                <Text>{renderQuota(totalQuota)}</Text>
              </Descriptions.Item>
              <Descriptions.Item itemKey='近期调用次数'>
                <Text>{logs.length} 次</Text>
              </Descriptions.Item>
            </Descriptions>
          </Spin>
        </Card>
      )}

      {/* Logs Table */}
      {tokenValid && (
        <Card
          title='调用详情'
          headerExtraContent={
            <Space>
              <Button
                icon={<IconDownload />}
                theme='borderless'
                type='primary'
                onClick={exportCSV}
                disabled={logs.length === 0}
              >
                导出CSV
              </Button>
            </Space>
          }
        >
          <Spin spinning={loading}>
            <Table
              columns={columns}
              dataSource={logs}
              pagination={{
                pageSize: ITEMS_PER_PAGE,
                hideOnSinglePage: true,
                showSizeChanger: false,
              }}
              size='small'
              rowKey='id'
              scroll={{ x: 800 }}
            />
          </Spin>
        </Card>
      )}
    </div>
  );
};

export default KeyQuery;
