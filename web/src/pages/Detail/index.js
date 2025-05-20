import React, { useContext, useEffect, useRef, useState } from 'react';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';

import {
  Card,
  Form,
  Spin,
  Typography,
  IconButton,
  Modal,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconSearch } from '@douyinfe/semi-icons';
import { VChart } from '@visactor/react-vchart';
import {
  API,
  isAdmin,
  showError,
  timestamp2string,
  timestamp2string1,
} from '../../helpers';
import {
  getQuotaWithUnit,
  modelColorMap,
  renderNumber,
  renderQuota,
  modelToColor,
} from '../../helpers/render';
import { UserContext } from '../../context/User/index.js';
import { StyleContext } from '../../context/Style/index.js';
import { useTranslation } from 'react-i18next';

const Detail = (props) => {
  const { t } = useTranslation();
  const { Text } = Typography;
  const formRef = useRef();
  let now = new Date();
  const [userState, userDispatch] = useContext(UserContext);
  const [styleState, styleDispatch] = useContext(StyleContext);
  const [inputs, setInputs] = useState({
    username: '',
    token_name: '',
    model_name: '',
    start_timestamp:
      localStorage.getItem('data_export_default_time') === 'hour'
        ? timestamp2string(now.getTime() / 1000 - 86400)
        : localStorage.getItem('data_export_default_time') === 'week'
          ? timestamp2string(now.getTime() / 1000 - 86400 * 30)
          : timestamp2string(now.getTime() / 1000 - 86400 * 7),
    end_timestamp: timestamp2string(now.getTime() / 1000 + 3600),
    channel: '',
    data_export_default_time: '',
  });
  const { username, model_name, start_timestamp, end_timestamp, channel } =
    inputs;
  const isAdminUser = isAdmin();
  const initialized = useRef(false);
  const [loading, setLoading] = useState(false);
  const [quotaData, setQuotaData] = useState([]);
  const [consumeQuota, setConsumeQuota] = useState(0);
  const [consumeTokens, setConsumeTokens] = useState(0);
  const [times, setTimes] = useState(0);
  const [dataExportDefaultTime, setDataExportDefaultTime] = useState(
    localStorage.getItem('data_export_default_time') || 'hour',
  );
  const [pieData, setPieData] = useState([{ type: 'null', value: '0' }]);
  const [lineData, setLineData] = useState([]);
  const [searchModalVisible, setSearchModalVisible] = useState(false);

  const [spec_pie, setSpecPie] = useState({
    type: 'pie',
    data: [
      {
        id: 'id0',
        values: pieData,
      },
    ],
    outerRadius: 0.8,
    innerRadius: 0.5,
    padAngle: 0.6,
    valueField: 'value',
    categoryField: 'type',
    pie: {
      style: {
        cornerRadius: 10,
      },
      state: {
        hover: {
          outerRadius: 0.85,
          stroke: '#000',
          lineWidth: 1,
        },
        selected: {
          outerRadius: 0.85,
          stroke: '#000',
          lineWidth: 1,
        },
      },
    },
    title: {
      visible: true,
      text: t('模型调用次数占比'),
      subtext: `${t('总计')}：${renderNumber(times)}`,
    },
    legends: {
      visible: true,
      orient: 'left',
    },
    label: {
      visible: true,
    },
    tooltip: {
      mark: {
        content: [
          {
            key: (datum) => datum['type'],
            value: (datum) => renderNumber(datum['value']),
          },
        ],
      },
    },
    color: {
      specified: modelColorMap,
    },
  });
  const [spec_line, setSpecLine] = useState({
    type: 'bar',
    data: [
      {
        id: 'barData',
        values: lineData,
      },
    ],
    xField: 'Time',
    yField: 'Usage',
    seriesField: 'Model',
    stack: true,
    legends: {
      visible: true,
      selectMode: 'single',
    },
    title: {
      visible: true,
      text: t('模型消耗分布'),
      subtext: `${t('总计')}：${renderQuota(consumeQuota, 2)}`,
    },
    bar: {
      state: {
        hover: {
          stroke: '#000',
          lineWidth: 1,
        },
      },
    },
    tooltip: {
      mark: {
        content: [
          {
            key: (datum) => datum['Model'],
            value: (datum) => renderQuota(datum['rawQuota'] || 0, 4),
          },
        ],
      },
      dimension: {
        content: [
          {
            key: (datum) => datum['Model'],
            value: (datum) => datum['rawQuota'] || 0,
          },
        ],
        updateContent: (array) => {
          array.sort((a, b) => b.value - a.value);
          let sum = 0;
          for (let i = 0; i < array.length; i++) {
            if (array[i].key == '其他') {
              continue;
            }
            let value = parseFloat(array[i].value);
            if (isNaN(value)) {
              value = 0;
            }
            if (array[i].datum && array[i].datum.TimeSum) {
              sum = array[i].datum.TimeSum;
            }
            array[i].value = renderQuota(value, 4);
          }
          array.unshift({
            key: t('总计'),
            value: renderQuota(sum, 4),
          });
          return array;
        },
      },
    },
    color: {
      specified: modelColorMap,
    },
  });

  // 添加一个新的状态来存储模型-颜色映射
  const [modelColors, setModelColors] = useState({});

  // 显示搜索Modal
  const showSearchModal = () => {
    setSearchModalVisible(true);
  };

  // 关闭搜索Modal
  const handleCloseModal = () => {
    setSearchModalVisible(false);
  };

  // 搜索Modal确认按钮
  const handleSearchConfirm = () => {
    refresh();
    setSearchModalVisible(false);
  };

  const handleInputChange = (value, name) => {
    if (name === 'data_export_default_time') {
      setDataExportDefaultTime(value);
      return;
    }
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const loadQuotaData = async () => {
    setLoading(true);
    try {
      let url = '';
      let localStartTimestamp = Date.parse(start_timestamp) / 1000;
      let localEndTimestamp = Date.parse(end_timestamp) / 1000;
      if (isAdminUser) {
        url = `/api/data/?username=${username}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&default_time=${dataExportDefaultTime}`;
      } else {
        url = `/api/data/self/?start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&default_time=${dataExportDefaultTime}`;
      }
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        setQuotaData(data);
        if (data.length === 0) {
          data.push({
            count: 0,
            model_name: '无数据',
            quota: 0,
            created_at: now.getTime() / 1000,
          });
        }
        // sort created_at
        data.sort((a, b) => a.created_at - b.created_at);
        updateChartData(data);
      } else {
        showError(message);
      }
    } finally {
      setLoading(false);
    }
  };

  const refresh = async () => {
    await loadQuotaData();
  };

  const initChart = async () => {
    await loadQuotaData();
  };

  const updateChartData = (data) => {
    let newPieData = [];
    let newLineData = [];
    let totalQuota = 0;
    let totalTimes = 0;
    let uniqueModels = new Set();
    let totalTokens = 0;

    // 收集所有唯一的模型名称
    data.forEach((item) => {
      uniqueModels.add(item.model_name);
      totalTokens += item.token_used;
      totalQuota += item.quota;
      totalTimes += item.count;
    });

    // 处理颜色映射
    const newModelColors = {};
    Array.from(uniqueModels).forEach((modelName) => {
      newModelColors[modelName] =
        modelColorMap[modelName] ||
        modelColors[modelName] ||
        modelToColor(modelName);
    });
    setModelColors(newModelColors);

    // 按时间和模型聚合数据
    let aggregatedData = new Map();
    data.forEach((item) => {
      const timeKey = timestamp2string1(item.created_at, dataExportDefaultTime);
      const modelKey = item.model_name;
      const key = `${timeKey}-${modelKey}`;

      if (!aggregatedData.has(key)) {
        aggregatedData.set(key, {
          time: timeKey,
          model: modelKey,
          quota: 0,
          count: 0,
        });
      }

      const existing = aggregatedData.get(key);
      existing.quota += item.quota;
      existing.count += item.count;
    });

    // 处理饼图数据
    let modelTotals = new Map();
    for (let [_, value] of aggregatedData) {
      if (!modelTotals.has(value.model)) {
        modelTotals.set(value.model, 0);
      }
      modelTotals.set(value.model, modelTotals.get(value.model) + value.count);
    }

    newPieData = Array.from(modelTotals).map(([model, count]) => ({
      type: model,
      value: count,
    }));

    // 生成时间点序列
    let timePoints = Array.from(
      new Set([...aggregatedData.values()].map((d) => d.time)),
    );
    if (timePoints.length < 7) {
      const lastTime = Math.max(...data.map((item) => item.created_at));
      const interval =
        dataExportDefaultTime === 'hour'
          ? 3600
          : dataExportDefaultTime === 'day'
            ? 86400
            : 604800;

      timePoints = Array.from({ length: 7 }, (_, i) =>
        timestamp2string1(lastTime - (6 - i) * interval, dataExportDefaultTime),
      );
    }

    // 生成柱状图数据
    timePoints.forEach((time) => {
      // 为每个时间点收集所有模型的数据
      let timeData = Array.from(uniqueModels).map((model) => {
        const key = `${time}-${model}`;
        const aggregated = aggregatedData.get(key);
        return {
          Time: time,
          Model: model,
          rawQuota: aggregated?.quota || 0,
          Usage: aggregated?.quota ? getQuotaWithUnit(aggregated.quota, 4) : 0,
        };
      });

      // 计算该时间点的总计
      const timeSum = timeData.reduce((sum, item) => sum + item.rawQuota, 0);

      // 按照 rawQuota 从大到小排序
      timeData.sort((a, b) => b.rawQuota - a.rawQuota);

      // 为每个数据点添加该时间的总计
      timeData = timeData.map((item) => ({
        ...item,
        TimeSum: timeSum,
      }));

      // 将排序后的数据添加到 newLineData
      newLineData.push(...timeData);
    });

    // 排序
    newPieData.sort((a, b) => b.value - a.value);
    newLineData.sort((a, b) => a.Time.localeCompare(b.Time));

    // 更新图表配置和数据
    setSpecPie((prev) => ({
      ...prev,
      data: [{ id: 'id0', values: newPieData }],
      title: {
        ...prev.title,
        subtext: `${t('总计')}：${renderNumber(totalTimes)}`,
      },
      color: {
        specified: newModelColors,
      },
    }));

    setSpecLine((prev) => ({
      ...prev,
      data: [{ id: 'barData', values: newLineData }],
      title: {
        ...prev.title,
        subtext: `${t('总计')}：${renderQuota(totalQuota, 2)}`,
      },
      color: {
        specified: newModelColors,
      },
    }));

    setPieData(newPieData);
    setLineData(newLineData);
    setConsumeQuota(totalQuota);
    setTimes(totalTimes);
    setConsumeTokens(totalTokens);
  };

  const getUserData = async () => {
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getUserData();
    if (!initialized.current) {
      initVChartSemiTheme({
        isWatchingThemeSwitch: true,
      });
      initialized.current = true;
      initChart();
    }
  }, []);

  // 数据卡片信息
  const statsData = [
    {
      title: t('当前余额'),
      value: renderQuota(userState?.user?.quota),
      icon: '💰',
      color: 'bg-blue-50',
    },
    {
      title: t('历史消耗'),
      value: renderQuota(userState?.user?.used_quota),
      icon: '📊',
      color: 'bg-purple-50',
    },
    {
      title: t('请求次数'),
      value: userState.user?.request_count,
      icon: '🔄',
      color: 'bg-green-50',
    },
    {
      title: t('统计额度'),
      value: renderQuota(consumeQuota),
      icon: '💲',
      color: 'bg-yellow-50',
    },
    {
      title: t('统计Tokens'),
      value: isNaN(consumeTokens) ? 0 : consumeTokens,
      icon: '🔤',
      color: 'bg-pink-50',
    },
    {
      title: t('统计次数'),
      value: times,
      icon: '📈',
      color: 'bg-teal-50',
    },
    {
      title: t('平均RPM'),
      value: (
        times /
        ((Date.parse(end_timestamp) - Date.parse(start_timestamp)) / 60000)
      ).toFixed(3),
      icon: '⏱️',
      color: 'bg-indigo-50',
    },
    {
      title: t('平均TPM'),
      value: (() => {
        const tpm = consumeTokens /
          ((Date.parse(end_timestamp) - Date.parse(start_timestamp)) / 60000);
        return isNaN(tpm) ? '0' : tpm.toFixed(3);
      })(),
      icon: '📝',
      color: 'bg-orange-50',
    },
  ];

  // 获取问候语
  const getGreeting = () => {
    const hours = new Date().getHours();
    let greeting = '';
    
    if (hours >= 5 && hours < 12) {
      greeting = t('早上好');
    } else if (hours >= 12 && hours < 14) {
      greeting = t('中午好');
    } else if (hours >= 14 && hours < 18) {
      greeting = t('下午好');
    } else {
      greeting = t('晚上好');
    }
    
    const username = userState?.user?.username || '';
    return `👋${greeting}，${username}`;
  };

  return (
    <div className="bg-gray-50 min-h-screen">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-semibold text-gray-800">{getGreeting()}</h2>
        <div className="flex gap-3">
          <IconButton
            icon={<IconSearch />}
            onClick={showSearchModal}
            className="bg-green-500 text-white hover:bg-green-600 !rounded-full"
            size="large"
          />
          <IconButton
            icon={<IconRefresh />}
            onClick={refresh}
            loading={loading}
            className="bg-blue-500 text-white hover:bg-blue-600 !rounded-full"
            size="large"
          />
        </div>
      </div>

      {/* 搜索条件Modal */}
      <Modal
        title={t('搜索条件')}
        visible={searchModalVisible}
        onOk={handleSearchConfirm}
        onCancel={handleCloseModal}
        closeOnEsc={true}
        width={700}
        centered
      >
        <Form ref={formRef} layout='vertical' className="w-full">
          <Form.DatePicker
            field='start_timestamp'
            label={t('起始时间')}
            className="w-full mb-4"
            initValue={start_timestamp}
            value={start_timestamp}
            type='dateTime'
            name='start_timestamp'
            onChange={(value) => handleInputChange(value, 'start_timestamp')}
          />
          <Form.DatePicker
            field='end_timestamp'
            label={t('结束时间')}
            className="w-full mb-4"
            initValue={end_timestamp}
            value={end_timestamp}
            type='dateTime'
            name='end_timestamp'
            onChange={(value) => handleInputChange(value, 'end_timestamp')}
          />
          <Form.Select
            field='data_export_default_time'
            label={t('时间粒度')}
            className="w-full mb-4"
            initValue={dataExportDefaultTime}
            placeholder={t('时间粒度')}
            name='data_export_default_time'
            optionList={[
              { label: t('小时'), value: 'hour' },
              { label: t('天'), value: 'day' },
              { label: t('周'), value: 'week' },
            ]}
            onChange={(value) => handleInputChange(value, 'data_export_default_time')}
          />
          {isAdminUser && (
            <Form.Input
              field='username'
              label={t('用户名称')}
              className="w-full mb-4"
              value={username}
              placeholder={t('可选值')}
              name='username'
              onChange={(value) => handleInputChange(value, 'username')}
            />
          )}
        </Form>
      </Modal>

      <Spin spinning={loading}>
        <div className="mb-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {statsData.map((stat, idx) => (
              <Card
                key={idx}
                shadows='hover'
                className={`${stat.color} border-0 !rounded-2xl w-full`}
                headerLine={false}
              >
                <div className="flex items-center">
                  <div className="text-2xl mr-3">{stat.icon}</div>
                  <div>
                    <div className="text-sm text-gray-500">{stat.title}</div>
                    <div className="text-xl font-semibold">{stat.value}</div>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <Card shadows='hover' className="shadow-sm !rounded-2xl" headerLine={true} title={t('模型消耗分布')}>
            <div style={{ height: 400 }}>
              <VChart
                spec={spec_line}
                option={{ mode: 'desktop-browser' }}
              />
            </div>
          </Card>

          <Card shadows='hover' className="shadow-sm !rounded-2xl" headerLine={true} title={t('模型调用次数占比')}>
            <div style={{ height: 400 }}>
              <VChart
                spec={spec_pie}
                option={{ mode: 'desktop-browser' }}
              />
            </div>
          </Card>
        </div>
      </Spin>
    </div>
  );
};

export default Detail;
