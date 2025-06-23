import React, { useState, useCallback, useMemo } from 'react';
import {
  Button,
  Table,
  Tag,
  Empty,
  Checkbox,
  Form,
  Input,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import {
  RefreshCcw,
  CheckSquare,
} from 'lucide-react';
import { API, showError, showSuccess, showWarning, stringToColor } from '../../../helpers';
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
  const [resolutions, setResolutions] = useState({});

  // 是否已经执行过同步
  const [hasSynced, setHasSynced] = useState(false);

  // 分页相关状态
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 搜索相关状态
  const [searchKeyword, setSearchKeyword] = useState('');

  const fetchAllChannels = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/ratio_sync/channels');

      if (res.data.success) {
        const channels = res.data.data || [];

        const transferData = channels.map(channel => ({
          key: channel.id,
          label: channel.name,
          value: channel.id,
          disabled: false,
          _originalData: channel,
        }));

        setAllChannels(transferData);

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

      const errorResults = test_results.filter(r => r.status === 'error');
      if (errorResults.length > 0) {
        showWarning(t('部分渠道测试失败：') + errorResults.map(r => `${r.name}: ${r.error}`).join(', '));
      }

      setDifferences(differences);
      setResolutions({});
      setHasSynced(true);

      if (Object.keys(differences).length === 0) {
        showSuccess(t('未找到差异化倍率，无需同步'));
      }
    } catch (e) {
      showError(t('请求后端接口失败：') + e.message);
    } finally {
      setSyncLoading(false);
    }
  };

  const selectValue = (model, ratioType, value) => {
    setResolutions(prev => ({
      ...prev,
      [model]: {
        ...prev[model],
        [ratioType]: value,
      },
    }));
  };

  const applySync = async () => {
    const currentRatios = {
      ModelRatio: JSON.parse(props.options.ModelRatio || '{}'),
      CompletionRatio: JSON.parse(props.options.CompletionRatio || '{}'),
      CacheRatio: JSON.parse(props.options.CacheRatio || '{}'),
      ModelPrice: JSON.parse(props.options.ModelPrice || '{}'),
    };

    Object.entries(resolutions).forEach(([model, ratios]) => {
      Object.entries(ratios).forEach(([ratioType, value]) => {
        const optionKey = ratioType
          .split('_')
          .map(word => word.charAt(0).toUpperCase() + word.slice(1))
          .join('');
        currentRatios[optionKey][model] = parseFloat(value);
      });
    });

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

        setDifferences(prevDifferences => {
          const newDifferences = { ...prevDifferences };

          Object.entries(resolutions).forEach(([model, ratios]) => {
            Object.keys(ratios).forEach(ratioType => {
              if (newDifferences[model] && newDifferences[model][ratioType]) {
                delete newDifferences[model][ratioType];

                if (Object.keys(newDifferences[model]).length === 0) {
                  delete newDifferences[model];
                }
              }
            });
          });

          return newDifferences;
        });

        setResolutions({});
      } else {
        showError(t('部分保存失败'));
      }
    } catch (error) {
      showError(t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  const getCurrentPageData = (dataSource) => {
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return dataSource.slice(startIndex, endIndex);
  };

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

          <Input
            prefix={<IconSearch size={14} />}
            placeholder={t('搜索模型名称')}
            value={searchKeyword}
            onChange={setSearchKeyword}
            className="!rounded-full w-full md:w-64 mt-2"
            showClear
          />
        </div>
      </div>
    </div>
  );

  const renderDifferenceTable = () => {
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

    const filteredDataSource = useMemo(() => {
      if (!searchKeyword.trim()) {
        return dataSource;
      }

      const keyword = searchKeyword.toLowerCase().trim();
      return dataSource.filter(item =>
        item.model.toLowerCase().includes(keyword)
      );
    }, [dataSource, searchKeyword]);

    const upstreamNames = useMemo(() => {
      const set = new Set();
      filteredDataSource.forEach((row) => {
        Object.keys(row.upstreams || {}).forEach((name) => set.add(name));
      });
      return Array.from(set);
    }, [filteredDataSource]);

    if (filteredDataSource.length === 0) {
      return (
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={<IllustrationNoResultDark style={{ width: 150, height: 150 }} />}
          description={
            searchKeyword.trim()
              ? t('未找到匹配的模型')
              : (Object.keys(differences).length === 0 ?
                (hasSynced ? t('暂无差异化倍率显示') : t('请先选择同步渠道'))
                : t('请先选择同步渠道'))
          }
          style={{ padding: 30 }}
        />
      );
    }

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
          return <Tag color={stringToColor(text)} shape="circle">{typeMap[text] || text}</Tag>;
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
      ...upstreamNames.map((upName) => {
        const channelStats = (() => {
          let selectableCount = 0;
          let selectedCount = 0;

          filteredDataSource.forEach((row) => {
            const upstreamVal = row.upstreams?.[upName];
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

        const handleBulkSelect = (checked) => {
          setResolutions((prev) => {
            const newRes = { ...prev };

            filteredDataSource.forEach((row) => {
              const upstreamVal = row.upstreams?.[upName];
              if (upstreamVal !== null && upstreamVal !== undefined && upstreamVal !== 'same') {
                if (checked) {
                  if (!newRes[row.model]) newRes[row.model] = {};
                  newRes[row.model][row.ratioType] = upstreamVal;
                } else {
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
        dataSource={getCurrentPageData(filteredDataSource)}
        pagination={{
          currentPage: currentPage,
          pageSize: pageSize,
          total: filteredDataSource.length,
          showSizeChanger: true,
          showQuickJumper: true,
          formatPageText: (page) => t('第 {{start}} - {{end}} 条，共 {{total}} 条', {
            start: page.currentStart,
            end: page.currentEnd,
            total: filteredDataSource.length,
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