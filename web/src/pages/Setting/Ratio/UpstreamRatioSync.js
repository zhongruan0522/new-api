import React, { useState, useCallback, useMemo } from 'react';
import {
  Button,
  Table,
  Tag,
  Empty,
  Checkbox,
  Form,
} from '@douyinfe/semi-ui';
import {
  RefreshCcw,
  CheckSquare,
} from 'lucide-react';
import { API, showError, showSuccess, showWarning } from '../../../helpers';
import { DEFAULT_ENDPOINT } from '../../../constants';
import { useTranslation } from 'react-i18next';
import {
  IllustrationNoResult,
  IllustrationNoResultDark
} from '@douyinfe/semi-illustrations';
import ChannelSelectorModal from '../../../components/settings/ChannelSelectorModal';

export default function UpstreamRatioSync(props) {
  const { t } = useTranslation();
  const [modalVisible, setModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);

  // 渠道选择相关
  const [allChannels, setAllChannels] = useState([]);
  const [selectedChannelIds, setSelectedChannelIds] = useState([]);

  // 渠道端点配置
  const [channelEndpoints, setChannelEndpoints] = useState({}); // { channelId: endpoint }

  // 差异数据和测试结果
  const [differences, setDifferences] = useState({});
  const [testResults, setTestResults] = useState([]);
  const [resolutions, setResolutions] = useState({});

  // 分页相关状态
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 获取所有渠道
  const fetchAllChannels = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/ratio_sync/channels');

      if (res.data.success) {
        const channels = res.data.data || [];

        // 转换为Transfer组件所需格式
        const transferData = channels.map(channel => ({
          key: channel.id,
          label: channel.name,
          value: channel.id,
          disabled: false, // 所有渠道都可以选择
          _originalData: channel,
        }));

        setAllChannels(transferData);

        // 初始化端点配置
        const initialEndpoints = {};
        transferData.forEach(channel => {
          initialEndpoints[channel.key] = DEFAULT_ENDPOINT;
        });
        setChannelEndpoints(initialEndpoints);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('获取渠道失败：') + error.message);
    } finally {
      setLoading(false);
    }
  };

  // 确认选择渠道
  const confirmChannelSelection = () => {
    const selected = allChannels
      .filter(ch => selectedChannelIds.includes(ch.value))
      .map(ch => ch._originalData);

    if (selected.length === 0) {
      showWarning(t('请至少选择一个渠道'));
      return;
    }

    setModalVisible(false);
    fetchRatiosFromChannels(selected);
  };

  // 从选定渠道获取倍率
  const fetchRatiosFromChannels = async (channelList) => {
    setSyncLoading(true);

    const payload = {
      channel_ids: channelList.map(ch => parseInt(ch.id)),
      timeout: 10,
    };

    try {
      const res = await API.post('/api/ratio_sync/fetch', payload);

      if (!res.data.success) {
        showError(res.data.message || t('后端请求失败'));
        setSyncLoading(false);
        return;
      }

      const { differences = {}, test_results = [] } = res.data.data;

      // 显示测试结果
      const errorResults = test_results.filter(r => r.status === 'error');
      if (errorResults.length > 0) {
        showWarning(t('部分渠道测试失败：') + errorResults.map(r => `${r.name}: ${r.error}`).join(', '));
      }

      setDifferences(differences);
      setTestResults(test_results);
      setResolutions({});

      // 判断是否有差异
      if (Object.keys(differences).length === 0) {
        showSuccess(t('已与上游倍率完全一致，无需同步'));
      }
    } catch (e) {
      showError(t('请求后端接口失败：') + e.message);
    } finally {
      setSyncLoading(false);
    }
  };

  // 解决冲突/选择值
  const selectValue = (model, ratioType, value) => {
    setResolutions(prev => ({
      ...prev,
      [model]: {
        ...prev[model],
        [ratioType]: value,
      },
    }));
  };

  // 应用同步
  const applySync = async () => {
    const currentRatios = {
      ModelRatio: JSON.parse(props.options.ModelRatio || '{}'),
      CompletionRatio: JSON.parse(props.options.CompletionRatio || '{}'),
      CacheRatio: JSON.parse(props.options.CacheRatio || '{}'),
      ModelPrice: JSON.parse(props.options.ModelPrice || '{}'),
    };

    // 应用已选择的值
    Object.entries(resolutions).forEach(([model, ratios]) => {
      Object.entries(ratios).forEach(([ratioType, value]) => {
        const optionKey = ratioType
          .split('_')
          .map(word => word.charAt(0).toUpperCase() + word.slice(1))
          .join('');
        currentRatios[optionKey][model] = parseFloat(value);
      });
    });

    // 保存到后端
    setLoading(true);
    try {
      const updates = Object.entries(currentRatios).map(([key, value]) =>
        API.put('/api/option/', {
          key,
          value: JSON.stringify(value, null, 2),
        })
      );

      const results = await Promise.all(updates);

      if (results.every(res => res.data.success)) {
        showSuccess(t('同步成功'));
        props.refresh();
        // 清空状态
        setDifferences({});
        setTestResults([]);
        setResolutions({});
        setSelectedChannelIds([]);
      } else {
        showError(t('部分保存失败'));
      }
    } catch (error) {
      showError(t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  // 计算当前页显示的数据
  const getCurrentPageData = (dataSource) => {
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return dataSource.slice(startIndex, endIndex);
  };

  // 渲染表格头部
  const renderHeader = () => (
    <div className="flex flex-col w-full">
      <div className="flex flex-col md:flex-row justify-between items-center gap-4 w-full">
        <div className="flex gap-2 w-full md:w-auto order-2 md:order-1">
          <Button
            icon={<RefreshCcw size={14} />}
            className="!rounded-full w-full md:w-auto mt-2"
            onClick={() => {
              setModalVisible(true);
              fetchAllChannels();
            }}
          >
            {t('选择同步渠道')}
          </Button>

          {(() => {
            // 检查是否有选择可应用的值
            const hasSelections = Object.keys(resolutions).length > 0;

            return (
              <Button
                icon={<CheckSquare size={14} />}
                type='secondary'
                onClick={applySync}
                disabled={!hasSelections}
                className="!rounded-full w-full md:w-auto mt-2"
              >
                {t('应用同步')}
              </Button>
            );
          })()}
        </div>
      </div>
    </div>
  );

  // 渲染差异表格
  const renderDifferenceTable = () => {
    // 构建数据源
    const dataSource = useMemo(() => {
      const tmp = [];

      Object.entries(differences).forEach(([model, ratioTypes]) => {
        Object.entries(ratioTypes).forEach(([ratioType, diff]) => {
          tmp.push({
            key: `${model}_${ratioType}`,
            model,
            ratioType,
            current: diff.current,
            upstreams: diff.upstreams,
          });
        });
      });

      return tmp;
    }, [differences]);

    // 收集所有上游渠道名称
    const upstreamNames = useMemo(() => {
      const set = new Set();
      dataSource.forEach((row) => {
        Object.keys(row.upstreams || {}).forEach((name) => set.add(name));
      });
      return Array.from(set);
    }, [dataSource]);

    if (dataSource.length === 0) {
      return (
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={<IllustrationNoResultDark style={{ width: 150, height: 150 }} />}
          description={Object.keys(differences).length === 0 ? t('已与上游倍率完全一致') : t('请先选择同步渠道')}
          style={{ padding: 30 }}
        />
      );
    }

    // 列定义
    const columns = [
      {
        title: t('模型'),
        dataIndex: 'model',
        fixed: 'left',
      },
      {
        title: t('倍率类型'),
        dataIndex: 'ratioType',
        render: (text) => {
          const typeMap = {
            model_ratio: t('模型倍率'),
            completion_ratio: t('补全倍率'),
            cache_ratio: t('缓存倍率'),
            model_price: t('固定价格'),
          };
          return <Tag shape="circle">{typeMap[text] || text}</Tag>;
        },
      },
      {
        title: t('当前值'),
        dataIndex: 'current',
        render: (text) => (
          <Tag color={text !== null && text !== undefined ? 'blue' : 'default'} shape="circle">
            {text !== null && text !== undefined ? text : t('未设置')}
          </Tag>
        ),
      },
      // 动态上游列
      ...upstreamNames.map((upName) => {
        // 计算该渠道的全选状态
        const channelStats = (() => {
          let selectableCount = 0;  // 可选择的项目数量
          let selectedCount = 0;    // 已选择的项目数量

          dataSource.forEach((row) => {
            const upstreamVal = row.upstreams?.[upName];
            // 只有具体数值的才是可选择的（不是null、undefined或"same"）
            if (upstreamVal !== null && upstreamVal !== undefined && upstreamVal !== 'same') {
              selectableCount++;
              const isSelected = resolutions[row.model]?.[row.ratioType] === upstreamVal;
              if (isSelected) {
                selectedCount++;
              }
            }
          });

          return {
            selectableCount,
            selectedCount,
            allSelected: selectableCount > 0 && selectedCount === selectableCount,
            partiallySelected: selectedCount > 0 && selectedCount < selectableCount,
            hasSelectableItems: selectableCount > 0
          };
        })();

        // 处理全选/取消全选
        const handleBulkSelect = (checked) => {
          setResolutions((prev) => {
            const newRes = { ...prev };

            dataSource.forEach((row) => {
              const upstreamVal = row.upstreams?.[upName];
              if (upstreamVal !== null && upstreamVal !== undefined && upstreamVal !== 'same') {
                if (checked) {
                  // 选择该值
                  if (!newRes[row.model]) newRes[row.model] = {};
                  newRes[row.model][row.ratioType] = upstreamVal;
                } else {
                  // 取消选择该值
                  if (newRes[row.model]) {
                    delete newRes[row.model][row.ratioType];
                    if (Object.keys(newRes[row.model]).length === 0) {
                      delete newRes[row.model];
                    }
                  }
                }
              }
            });

            return newRes;
          });
        };

        return {
          title: channelStats.hasSelectableItems ? (
            <Checkbox
              checked={channelStats.allSelected}
              indeterminate={channelStats.partiallySelected}
              onChange={(e) => handleBulkSelect(e.target.checked)}
            >
              {upName}
            </Checkbox>
          ) : (
            <span>{upName}</span>
          ),
          dataIndex: upName,
          render: (_, record) => {
            const upstreamVal = record.upstreams?.[upName];

            if (upstreamVal === null || upstreamVal === undefined) {
              return <Tag color="default" shape="circle">{t('未设置')}</Tag>;
            }

            if (upstreamVal === 'same') {
              return <Tag color="blue" shape="circle">{t('与本地相同')}</Tag>;
            }

            // 有具体值，可以选择
            const isSelected = resolutions[record.model]?.[record.ratioType] === upstreamVal;

            return (
              <Checkbox
                checked={isSelected}
                onChange={(e) => {
                  const isChecked = e.target.checked;
                  if (isChecked) {
                    selectValue(record.model, record.ratioType, upstreamVal);
                  } else {
                    setResolutions((prev) => {
                      const newRes = { ...prev };
                      if (newRes[record.model]) {
                        delete newRes[record.model][record.ratioType];
                        if (Object.keys(newRes[record.model]).length === 0) {
                          delete newRes[record.model];
                        }
                      }
                      return newRes;
                    });
                  }
                }}
              >
                {upstreamVal}
              </Checkbox>
            );
          },
        };
      }),
    ];

    return (
      <Table
        columns={columns}
        dataSource={getCurrentPageData(dataSource)}
        pagination={{
          currentPage: currentPage,
          pageSize: pageSize,
          total: dataSource.length,
          showSizeChanger: true,
          showQuickJumper: true,
          formatPageText: (page) => t('第 {{start}} - {{end}} 条，共 {{total}} 条', {
            start: page.currentStart,
            end: page.currentEnd,
            total: dataSource.length,
          }),
          pageSizeOptions: ['5', '10', '20', '50'],
          onChange: (page, size) => {
            setCurrentPage(page);
            setPageSize(size);
          },
          onShowSizeChange: (current, size) => {
            setCurrentPage(1);
            setPageSize(size);
          }
        }}
        scroll={{ x: 'max-content' }}
        size='middle'
        loading={loading || syncLoading}
        className="rounded-xl overflow-hidden"
      />
    );
  };

  // 更新渠道端点
  const updateChannelEndpoint = useCallback((channelId, endpoint) => {
    setChannelEndpoints(prev => ({ ...prev, [channelId]: endpoint }));
  }, []);

  return (
    <>
      <Form.Section text={renderHeader()}>
        {renderDifferenceTable()}
      </Form.Section>

      <ChannelSelectorModal
        t={t}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={confirmChannelSelection}
        allChannels={allChannels}
        selectedChannelIds={selectedChannelIds}
        setSelectedChannelIds={setSelectedChannelIds}
        channelEndpoints={channelEndpoints}
        updateChannelEndpoint={updateChannelEndpoint}
      />
    </>
  );
} 