import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  copy,
  isAdmin,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';

import {
  Button,
  Card,
  Checkbox,
  DatePicker,
  Divider,
  Input,
  Layout,
  Modal,
  Progress,
  Skeleton,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { ITEMS_PER_PAGE } from '../../constants';
import {
  IconEyeOpened,
  IconSearch,
  IconSetting,
} from '@douyinfe/semi-icons';

const { Text } = Typography;

const colors = [
  'amber',
  'blue',
  'cyan',
  'green',
  'grey',
  'indigo',
  'light-blue',
  'lime',
  'orange',
  'pink',
  'purple',
  'red',
  'teal',
  'violet',
  'yellow',
];

// 定义列键值常量
const COLUMN_KEYS = {
  SUBMIT_TIME: 'submit_time',
  FINISH_TIME: 'finish_time',
  DURATION: 'duration',
  CHANNEL: 'channel',
  PLATFORM: 'platform',
  TYPE: 'type',
  TASK_ID: 'task_id',
  TASK_STATUS: 'task_status',
  PROGRESS: 'progress',
  FAIL_REASON: 'fail_reason',
};

const renderTimestamp = (timestampInSeconds) => {
  const date = new Date(timestampInSeconds * 1000); // 从秒转换为毫秒

  const year = date.getFullYear(); // 获取年份
  const month = ('0' + (date.getMonth() + 1)).slice(-2); // 获取月份，从0开始需要+1，并保证两位数
  const day = ('0' + date.getDate()).slice(-2); // 获取日期，并保证两位数
  const hours = ('0' + date.getHours()).slice(-2); // 获取小时，并保证两位数
  const minutes = ('0' + date.getMinutes()).slice(-2); // 获取分钟，并保证两位数
  const seconds = ('0' + date.getSeconds()).slice(-2); // 获取秒钟，并保证两位数

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`; // 格式化输出
};

function renderDuration(submit_time, finishTime) {
  // 确保startTime和finishTime都是有效的时间戳
  if (!submit_time || !finishTime) return 'N/A';

  // 将时间戳转换为Date对象
  const start = new Date(submit_time);
  const finish = new Date(finishTime);

  // 计算时间差（毫秒）
  const durationMs = finish - start;

  // 将时间差转换为秒，并保留一位小数
  const durationSec = (durationMs / 1000).toFixed(1);

  // 设置颜色：大于60秒则为红色，小于等于60秒则为绿色
  const color = durationSec > 60 ? 'red' : 'green';

  // 返回带有样式的颜色标签
  return (
    <Tag color={color} size='large'>
      {durationSec} 秒
    </Tag>
  );
}

const LogsTable = () => {
  const { t } = useTranslation();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalContent, setModalContent] = useState('');

  // 列可见性状态
  const [visibleColumns, setVisibleColumns] = useState({});
  const [showColumnSelector, setShowColumnSelector] = useState(false);
  const isAdminUser = isAdmin();
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);

  // 加载保存的列偏好设置
  useEffect(() => {
    const savedColumns = localStorage.getItem('task-logs-table-columns');
    if (savedColumns) {
      try {
        const parsed = JSON.parse(savedColumns);
        const defaults = getDefaultColumnVisibility();
        const merged = { ...defaults, ...parsed };
        setVisibleColumns(merged);
      } catch (e) {
        console.error('Failed to parse saved column preferences', e);
        initDefaultColumns();
      }
    } else {
      initDefaultColumns();
    }
  }, []);

  // 获取默认列可见性
  const getDefaultColumnVisibility = () => {
    return {
      [COLUMN_KEYS.SUBMIT_TIME]: true,
      [COLUMN_KEYS.FINISH_TIME]: true,
      [COLUMN_KEYS.DURATION]: true,
      [COLUMN_KEYS.CHANNEL]: isAdminUser,
      [COLUMN_KEYS.PLATFORM]: true,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.TASK_ID]: true,
      [COLUMN_KEYS.TASK_STATUS]: true,
      [COLUMN_KEYS.PROGRESS]: true,
      [COLUMN_KEYS.FAIL_REASON]: true,
    };
  };

  // 初始化默认列可见性
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem('task-logs-table-columns', JSON.stringify(defaults));
  };

  // 处理列可见性变化
  const handleColumnVisibilityChange = (columnKey, checked) => {
    const updatedColumns = { ...visibleColumns, [columnKey]: checked };
    setVisibleColumns(updatedColumns);
  };

  // 处理全选
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};

    allKeys.forEach((key) => {
      if (key === COLUMN_KEYS.CHANNEL && !isAdminUser) {
        updatedColumns[key] = false;
      } else {
        updatedColumns[key] = checked;
      }
    });

    setVisibleColumns(updatedColumns);
  };

  // 更新表格时保存列可见性
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem('task-logs-table-columns', JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  const renderType = (type) => {
    switch (type) {
      case 'MUSIC':
        return (
          <Tag color='grey' size='large' shape='circle'>
            {t('生成音乐')}
          </Tag>
        );
      case 'LYRICS':
        return (
          <Tag color='pink' size='large' shape='circle'>
            {t('生成歌词')}
          </Tag>
        );
      default:
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
    }
  };

  const renderPlatform = (type) => {
    switch (type) {
      case 'suno':
        return (
          <Tag color='green' size='large' shape='circle'>
            Suno
          </Tag>
        );
      default:
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
    }
  };

  const renderStatus = (type) => {
    switch (type) {
      case 'SUCCESS':
        return (
          <Tag color='green' size='large' shape='circle'>
            {t('成功')}
          </Tag>
        );
      case 'NOT_START':
        return (
          <Tag color='grey' size='large' shape='circle'>
            {t('未启动')}
          </Tag>
        );
      case 'SUBMITTED':
        return (
          <Tag color='yellow' size='large' shape='circle'>
            {t('队列中')}
          </Tag>
        );
      case 'IN_PROGRESS':
        return (
          <Tag color='blue' size='large' shape='circle'>
            {t('执行中')}
          </Tag>
        );
      case 'FAILURE':
        return (
          <Tag color='red' size='large' shape='circle'>
            {t('失败')}
          </Tag>
        );
      case 'QUEUED':
        return (
          <Tag color='orange' size='large' shape='circle'>
            {t('排队中')}
          </Tag>
        );
      case 'UNKNOWN':
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
      case '':
        return (
          <Tag color='grey' size='large' shape='circle'>
            {t('正在提交')}
          </Tag>
        );
      default:
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
    }
  };

  // 定义所有列
  const allColumns = [
    {
      key: COLUMN_KEYS.SUBMIT_TIME,
      title: t('提交时间'),
      dataIndex: 'submit_time',
      render: (text, record, index) => {
        return <div>{text ? renderTimestamp(text) : '-'}</div>;
      },
    },
    {
      key: COLUMN_KEYS.FINISH_TIME,
      title: t('结束时间'),
      dataIndex: 'finish_time',
      render: (text, record, index) => {
        return <div>{text ? renderTimestamp(text) : '-'}</div>;
      },
    },
    {
      key: COLUMN_KEYS.DURATION,
      title: t('花费时间'),
      dataIndex: 'finish_time',
      render: (finish, record) => {
        return <>{finish ? renderDuration(record.submit_time, finish) : '-'}</>;
      },
    },
    {
      key: COLUMN_KEYS.CHANNEL,
      title: t('渠道'),
      dataIndex: 'channel_id',
      className: isAdminUser ? 'tableShow' : 'tableHiddle',
      render: (text, record, index) => {
        return isAdminUser ? (
          <div>
            <Tag
              color={colors[parseInt(text) % colors.length]}
              size='large'
              shape='circle'
              onClick={() => {
                copyText(text);
              }}
            >
              {text}
            </Tag>
          </div>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.PLATFORM,
      title: t('平台'),
      dataIndex: 'platform',
      render: (text, record, index) => {
        return <div>{renderPlatform(text)}</div>;
      },
    },
    {
      key: COLUMN_KEYS.TYPE,
      title: t('类型'),
      dataIndex: 'action',
      render: (text, record, index) => {
        return <div>{renderType(text)}</div>;
      },
    },
    {
      key: COLUMN_KEYS.TASK_ID,
      title: t('任务ID'),
      dataIndex: 'task_id',
      render: (text, record, index) => {
        return (
          <Typography.Text
            ellipsis={{ showTooltip: true }}
            onClick={() => {
              setModalContent(JSON.stringify(record, null, 2));
              setIsModalOpen(true);
            }}
          >
            <div>{text}</div>
          </Typography.Text>
        );
      },
    },
    {
      key: COLUMN_KEYS.TASK_STATUS,
      title: t('任务状态'),
      dataIndex: 'status',
      render: (text, record, index) => {
        return <div>{renderStatus(text)}</div>;
      },
    },
    {
      key: COLUMN_KEYS.PROGRESS,
      title: t('进度'),
      dataIndex: 'progress',
      render: (text, record, index) => {
        return (
          <div>
            {
              isNaN(text?.replace('%', '')) ? (
                text || '-'
              ) : (
                <Progress
                  stroke={
                    record.status === 'FAILURE'
                      ? 'var(--semi-color-warning)'
                      : null
                  }
                  percent={text ? parseInt(text.replace('%', '')) : 0}
                  showInfo={true}
                  aria-label='task progress'
                  style={{ minWidth: '160px' }}
                />
              )
            }
          </div>
        );
      },
    },
    {
      key: COLUMN_KEYS.FAIL_REASON,
      title: t('失败原因'),
      dataIndex: 'fail_reason',
      fixed: 'right',
      render: (text, record, index) => {
        if (!text) {
          return t('无');
        }
        return (
          <Typography.Text
            ellipsis={{ showTooltip: true }}
            style={{ width: 100 }}
            onClick={() => {
              setModalContent(text);
              setIsModalOpen(true);
            }}
          >
            {text}
          </Typography.Text>
        );
      },
    },
  ];

  // 根据可见性设置过滤列
  const getVisibleColumns = () => {
    return allColumns.filter((column) => visibleColumns[column.key]);
  };

  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(ITEMS_PER_PAGE);
  const [logType] = useState(0);

  let now = new Date();
  // 初始化start_timestamp为前一天
  let zeroNow = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const [inputs, setInputs] = useState({
    channel_id: '',
    task_id: '',
    start_timestamp: timestamp2string(zeroNow.getTime() / 1000),
    end_timestamp: '',
  });
  const { channel_id, task_id, start_timestamp, end_timestamp } = inputs;

  const handleInputChange = (value, name) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const setLogsFormat = (logs) => {
    for (let i = 0; i < logs.length; i++) {
      logs[i].timestamp2string = timestamp2string(logs[i].created_at);
      logs[i].key = '' + logs[i].id;
    }
    // data.key = '' + data.id
    setLogs(logs);
    setLogCount(logs.length + ITEMS_PER_PAGE);
    // console.log(logCount);
  };

  const loadLogs = async (startIdx, pageSize = ITEMS_PER_PAGE) => {
    setLoading(true);

    let url = '';
    let localStartTimestamp = parseInt(Date.parse(start_timestamp) / 1000);
    let localEndTimestamp = parseInt(Date.parse(end_timestamp) / 1000);
    if (isAdminUser) {
      url = `/api/task/?p=${startIdx}&page_size=${pageSize}&channel_id=${channel_id}&task_id=${task_id}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    } else {
      url = `/api/task/self?p=${startIdx}&page_size=${pageSize}&task_id=${task_id}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    }
    const res = await API.get(url);
    let { success, message, data } = res.data;
    if (success) {
      if (startIdx === 0) {
        setLogsFormat(data);
      } else {
        let newLogs = [...logs];
        newLogs.splice(startIdx * pageSize, data.length, ...data);
        setLogsFormat(newLogs);
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const pageData = logs.slice(
    (activePage - 1) * pageSize,
    activePage * pageSize,
  );

  const handlePageChange = (page) => {
    setActivePage(page);
    if (page === Math.ceil(logs.length / pageSize) + 1) {
      loadLogs(page - 1, pageSize).then((r) => { });
    }
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('task-page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    await loadLogs(0, size);
  };

  const refresh = async () => {
    setActivePage(1);
    await loadLogs(0, pageSize);
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制：') + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  useEffect(() => {
    const localPageSize = parseInt(localStorage.getItem('task-page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadLogs(0, localPageSize).then();
  }, [logType]);

  // 列选择器模态框
  const renderColumnSelector = () => {
    return (
      <Modal
        title={t('列设置')}
        visible={showColumnSelector}
        onCancel={() => setShowColumnSelector(false)}
        footer={
          <div className="flex justify-end">
            <Button
              theme="light"
              onClick={() => initDefaultColumns()}
              className="!rounded-full"
            >
              {t('重置')}
            </Button>
            <Button
              theme="light"
              onClick={() => setShowColumnSelector(false)}
              className="!rounded-full"
            >
              {t('取消')}
            </Button>
            <Button
              type='primary'
              onClick={() => setShowColumnSelector(false)}
              className="!rounded-full"
            >
              {t('确定')}
            </Button>
          </div>
        }
      >
        <div style={{ marginBottom: 20 }}>
          <Checkbox
            checked={Object.values(visibleColumns).every((v) => v === true)}
            indeterminate={
              Object.values(visibleColumns).some((v) => v === true) &&
              !Object.values(visibleColumns).every((v) => v === true)
            }
            onChange={(e) => handleSelectAll(e.target.checked)}
          >
            {t('全选')}
          </Checkbox>
        </div>
        <div className="flex flex-wrap max-h-96 overflow-y-auto rounded-lg p-4" style={{ border: '1px solid var(--semi-color-border)' }}>
          {allColumns.map((column) => {
            // 为非管理员用户跳过管理员专用列
            if (!isAdminUser && column.key === COLUMN_KEYS.CHANNEL) {
              return null;
            }

            return (
              <div key={column.key} className="w-1/2 mb-4 pr-2">
                <Checkbox
                  checked={!!visibleColumns[column.key]}
                  onChange={(e) =>
                    handleColumnVisibilityChange(column.key, e.target.checked)
                  }
                >
                  {column.title}
                </Checkbox>
              </div>
            );
          })}
        </div>
      </Modal>
    );
  };

  return (
    <>
      {renderColumnSelector()}
      <Layout>
        <Card
          className="!rounded-2xl mb-4"
          title={
            <div className="flex flex-col w-full">
              <div className="flex flex-col md:flex-row justify-between items-center">
                <div className="flex items-center text-orange-500 mb-2 md:mb-0">
                  <IconEyeOpened className="mr-2" />
                  {loading ? (
                    <Skeleton.Title
                      style={{
                        width: 300,
                        marginBottom: 0,
                        marginTop: 0
                      }}
                    />
                  ) : (
                    <Text>{t('任务记录')}</Text>
                  )}
                </div>
              </div>

              <Divider margin="12px" />

              {/* 搜索表单区域 */}
              <div className="flex flex-col gap-4">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                  {/* 时间选择器 */}
                  <div className="col-span-1 lg:col-span-2">
                    <DatePicker
                      className="w-full"
                      value={[start_timestamp, end_timestamp]}
                      type='dateTimeRange'
                      onChange={(value) => {
                        if (Array.isArray(value) && value.length === 2) {
                          handleInputChange(value[0], 'start_timestamp');
                          handleInputChange(value[1], 'end_timestamp');
                        }
                      }}
                    />
                  </div>

                  {/* 任务 ID */}
                  <Input
                    prefix={<IconSearch />}
                    placeholder={t('任务 ID')}
                    value={task_id}
                    onChange={(value) => handleInputChange(value, 'task_id')}
                    className="!rounded-full"
                    showClear
                  />

                  {/* 渠道 ID - 仅管理员可见 */}
                  {isAdminUser && (
                    <Input
                      prefix={<IconSearch />}
                      placeholder={t('渠道 ID')}
                      value={channel_id}
                      onChange={(value) => handleInputChange(value, 'channel_id')}
                      className="!rounded-full"
                      showClear
                    />
                  )}
                </div>

                {/* 操作按钮区域 */}
                <div className="flex justify-between items-center pt-2">
                  <div></div>
                  <div className="flex gap-2">
                    <Button
                      type='primary'
                      onClick={refresh}
                      loading={loading}
                      className="!rounded-full"
                    >
                      {t('查询')}
                    </Button>
                    <Button
                      theme='light'
                      type='tertiary'
                      icon={<IconSetting />}
                      onClick={() => setShowColumnSelector(true)}
                      className="!rounded-full"
                    >
                      {t('列设置')}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          }
          shadows='always'
          bordered={false}
        >
          <Table
            columns={getVisibleColumns()}
            dataSource={pageData}
            rowKey='key'
            loading={loading}
            scroll={{ x: 'max-content' }}
            className="rounded-xl overflow-hidden"
            size="middle"
            pagination={{
              formatPageText: (page) =>
                t('第 {{start}} - {{end}} 条，共 {{total}} 条', {
                  start: page.currentStart,
                  end: page.currentEnd,
                  total: logCount,
                }),
              currentPage: activePage,
              pageSize: pageSize,
              total: logCount,
              pageSizeOptions: [10, 20, 50, 100],
              showSizeChanger: true,
              onPageSizeChange: (size) => {
                handlePageSizeChange(size);
              },
              onPageChange: handlePageChange,
            }}
          />
        </Card>

        <Modal
          visible={isModalOpen}
          onOk={() => setIsModalOpen(false)}
          onCancel={() => setIsModalOpen(false)}
          closable={null}
          bodyStyle={{ height: '400px', overflow: 'auto' }} // 设置模态框内容区域样式
          width={800} // 设置模态框宽度
        >
          <p style={{ whiteSpace: 'pre-line' }}>{modalContent}</p>
        </Modal>
      </Layout>
    </>
  );
};

export default LogsTable;
