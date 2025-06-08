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
  ImagePreview,
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
  DURATION: 'duration',
  CHANNEL: 'channel',
  TYPE: 'type',
  TASK_ID: 'task_id',
  SUBMIT_RESULT: 'submit_result',
  TASK_STATUS: 'task_status',
  PROGRESS: 'progress',
  IMAGE: 'image',
  PROMPT: 'prompt',
  PROMPT_EN: 'prompt_en',
  FAIL_REASON: 'fail_reason',
};

const LogsTable = () => {
  const { t } = useTranslation();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalContent, setModalContent] = useState('');

  // 列可见性状态
  const [visibleColumns, setVisibleColumns] = useState({});
  const [showColumnSelector, setShowColumnSelector] = useState(false);
  const isAdminUser = isAdmin();

  // 加载保存的列偏好设置
  useEffect(() => {
    const savedColumns = localStorage.getItem('mj-logs-table-columns');
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
      [COLUMN_KEYS.DURATION]: true,
      [COLUMN_KEYS.CHANNEL]: isAdminUser,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.TASK_ID]: true,
      [COLUMN_KEYS.SUBMIT_RESULT]: isAdminUser,
      [COLUMN_KEYS.TASK_STATUS]: true,
      [COLUMN_KEYS.PROGRESS]: true,
      [COLUMN_KEYS.IMAGE]: true,
      [COLUMN_KEYS.PROMPT]: true,
      [COLUMN_KEYS.PROMPT_EN]: true,
      [COLUMN_KEYS.FAIL_REASON]: true,
    };
  };

  // 初始化默认列可见性
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem('mj-logs-table-columns', JSON.stringify(defaults));
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
      if ((key === COLUMN_KEYS.CHANNEL || key === COLUMN_KEYS.SUBMIT_RESULT) && !isAdminUser) {
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
      localStorage.setItem('mj-logs-table-columns', JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  function renderType(type) {
    switch (type) {
      case 'IMAGINE':
        return (
          <Tag color='blue' size='large' shape='circle'>
            {t('绘图')}
          </Tag>
        );
      case 'UPSCALE':
        return (
          <Tag color='orange' size='large' shape='circle'>
            {t('放大')}
          </Tag>
        );
      case 'VARIATION':
        return (
          <Tag color='purple' size='large' shape='circle'>
            {t('变换')}
          </Tag>
        );
      case 'HIGH_VARIATION':
        return (
          <Tag color='purple' size='large' shape='circle'>
            {t('强变换')}
          </Tag>
        );
      case 'LOW_VARIATION':
        return (
          <Tag color='purple' size='large' shape='circle'>
            {t('弱变换')}
          </Tag>
        );
      case 'PAN':
        return (
          <Tag color='cyan' size='large' shape='circle'>
            {t('平移')}
          </Tag>
        );
      case 'DESCRIBE':
        return (
          <Tag color='yellow' size='large' shape='circle'>
            {t('图生文')}
          </Tag>
        );
      case 'BLEND':
        return (
          <Tag color='lime' size='large' shape='circle'>
            {t('图混合')}
          </Tag>
        );
      case 'UPLOAD':
        return (
          <Tag color='blue' size='large' shape='circle'>
            上传文件
          </Tag>
        );
      case 'SHORTEN':
        return (
          <Tag color='pink' size='large' shape='circle'>
            {t('缩词')}
          </Tag>
        );
      case 'REROLL':
        return (
          <Tag color='indigo' size='large' shape='circle'>
            {t('重绘')}
          </Tag>
        );
      case 'INPAINT':
        return (
          <Tag color='violet' size='large' shape='circle'>
            {t('局部重绘-提交')}
          </Tag>
        );
      case 'ZOOM':
        return (
          <Tag color='teal' size='large' shape='circle'>
            {t('变焦')}
          </Tag>
        );
      case 'CUSTOM_ZOOM':
        return (
          <Tag color='teal' size='large' shape='circle'>
            {t('自定义变焦-提交')}
          </Tag>
        );
      case 'MODAL':
        return (
          <Tag color='green' size='large' shape='circle'>
            {t('窗口处理')}
          </Tag>
        );
      case 'SWAP_FACE':
        return (
          <Tag color='light-green' size='large' shape='circle'>
            {t('换脸')}
          </Tag>
        );
      default:
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
    }
  }

  function renderCode(code) {
    switch (code) {
      case 1:
        return (
          <Tag color='green' size='large' shape='circle'>
            {t('已提交')}
          </Tag>
        );
      case 21:
        return (
          <Tag color='lime' size='large' shape='circle'>
            {t('等待中')}
          </Tag>
        );
      case 22:
        return (
          <Tag color='orange' size='large' shape='circle'>
            {t('重复提交')}
          </Tag>
        );
      case 0:
        return (
          <Tag color='yellow' size='large' shape='circle'>
            {t('未提交')}
          </Tag>
        );
      default:
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
    }
  }

  function renderStatus(type) {
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
      case 'MODAL':
        return (
          <Tag color='yellow' size='large' shape='circle'>
            {t('窗口等待')}
          </Tag>
        );
      default:
        return (
          <Tag color='white' size='large' shape='circle'>
            {t('未知')}
          </Tag>
        );
    }
  }

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
  // 修改renderDuration函数以包含颜色逻辑
  function renderDuration(submit_time, finishTime) {
    if (!submit_time || !finishTime) return 'N/A';

    const start = new Date(submit_time);
    const finish = new Date(finishTime);
    const durationMs = finish - start;
    const durationSec = (durationMs / 1000).toFixed(1);
    const color = durationSec > 60 ? 'red' : 'green';

    return (
      <Tag color={color} size='large' shape='circle'>
        {durationSec} {t('秒')}
      </Tag>
    );
  }

  // 定义所有列
  const allColumns = [
    {
      key: COLUMN_KEYS.SUBMIT_TIME,
      title: t('提交时间'),
      dataIndex: 'submit_time',
      render: (text, record, index) => {
        return <div>{renderTimestamp(text / 1000)}</div>;
      },
    },
    {
      key: COLUMN_KEYS.DURATION,
      title: t('花费时间'),
      dataIndex: 'finish_time',
      render: (finish, record) => {
        return renderDuration(record.submit_time, finish);
      },
    },
    {
      key: COLUMN_KEYS.CHANNEL,
      title: t('渠道'),
      dataIndex: 'channel_id',
      className: isAdmin() ? 'tableShow' : 'tableHiddle',
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
              {' '}
              {text}{' '}
            </Tag>
          </div>
        ) : (
          <></>
        );
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
      dataIndex: 'mj_id',
      render: (text, record, index) => {
        return <div>{text}</div>;
      },
    },
    {
      key: COLUMN_KEYS.SUBMIT_RESULT,
      title: t('提交结果'),
      dataIndex: 'code',
      className: isAdmin() ? 'tableShow' : 'tableHiddle',
      render: (text, record, index) => {
        return isAdminUser ? <div>{renderCode(text)}</div> : <></>;
      },
    },
    {
      key: COLUMN_KEYS.TASK_STATUS,
      title: t('任务状态'),
      dataIndex: 'status',
      className: isAdmin() ? 'tableShow' : 'tableHiddle',
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
              <Progress
                stroke={
                  record.status === 'FAILURE'
                    ? 'var(--semi-color-warning)'
                    : null
                }
                percent={text ? parseInt(text.replace('%', '')) : 0}
                showInfo={true}
                aria-label='drawing progress'
                style={{ minWidth: '160px' }}
              />
            }
          </div>
        );
      },
    },
    {
      key: COLUMN_KEYS.IMAGE,
      title: t('结果图片'),
      dataIndex: 'image_url',
      render: (text, record, index) => {
        if (!text) {
          return t('无');
        }
        return (
          <Button
            onClick={() => {
              setModalImageUrl(text);
              setIsModalOpenurl(true);
            }}
            className="!rounded-full"
          >
            {t('查看图片')}
          </Button>
        );
      },
    },
    {
      key: COLUMN_KEYS.PROMPT,
      title: 'Prompt',
      dataIndex: 'prompt',
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
    {
      key: COLUMN_KEYS.PROMPT_EN,
      title: 'PromptEn',
      dataIndex: 'prompt_en',
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
  const [logType, setLogType] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [isModalOpenurl, setIsModalOpenurl] = useState(false);
  const [showBanner, setShowBanner] = useState(false);

  // 定义模态框图片URL的状态和更新函数
  const [modalImageUrl, setModalImageUrl] = useState('');
  let now = new Date();
  // 初始化start_timestamp为前一天
  const [inputs, setInputs] = useState({
    channel_id: '',
    mj_id: '',
    start_timestamp: timestamp2string(now.getTime() / 1000 - 2592000),
    end_timestamp: timestamp2string(now.getTime() / 1000 + 3600),
  });
  const { channel_id, mj_id, start_timestamp, end_timestamp } = inputs;

  const [stat, setStat] = useState({
    quota: 0,
    token: 0,
  });

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
    setLogCount(logs.length + pageSize);
    // console.log(logCount);
  };

  const loadLogs = async (startIdx, pageSize = ITEMS_PER_PAGE) => {
    setLoading(true);

    let url = '';
    let localStartTimestamp = Date.parse(start_timestamp);
    let localEndTimestamp = Date.parse(end_timestamp);
    if (isAdminUser) {
      url = `/api/mj/?p=${startIdx}&page_size=${pageSize}&channel_id=${channel_id}&mj_id=${mj_id}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    } else {
      url = `/api/mj/self/?p=${startIdx}&page_size=${pageSize}&mj_id=${mj_id}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
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
      // In this case we have to load more data and then append them.
      loadLogs(page - 1, pageSize).then((r) => { });
    }
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('mj-page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    await loadLogs(0, size);
  };

  const refresh = async () => {
    // setLoading(true);
    setActivePage(1);
    await loadLogs(0, pageSize);
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制：') + text);
    } else {
      // setSearchKeyword(text);
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  useEffect(() => {
    const localPageSize = parseInt(localStorage.getItem('mj-page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadLogs(0, localPageSize).then();
  }, [logType]);

  useEffect(() => {
    const mjNotifyEnabled = localStorage.getItem('mj_notify_enabled');
    if (mjNotifyEnabled !== 'true') {
      setShowBanner(true);
    }
  }, []);

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
            if (
              !isAdminUser &&
              (column.key === COLUMN_KEYS.CHANNEL ||
                column.key === COLUMN_KEYS.SUBMIT_RESULT)
            ) {
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
                    <Text>
                      {isAdminUser && showBanner
                        ? t('当前未开启Midjourney回调，部分项目可能无法获得绘图结果，可在运营设置中开启。')
                        : t('Midjourney 任务记录')}
                    </Text>
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
                    value={mj_id}
                    onChange={(value) => handleInputChange(value, 'mj_id')}
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
        <ImagePreview
          src={modalImageUrl}
          visible={isModalOpenurl}
          onVisibleChange={(visible) => setIsModalOpenurl(visible)}
        />
      </Layout>
    </>
  );
};

export default LogsTable;
