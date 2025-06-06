import React, { useEffect, useState } from 'react';
import {
  API,
  copy,
  showError,
  showSuccess,
  timestamp2string,
  renderQuota
} from '../../helpers';

import { ITEMS_PER_PAGE } from '../../constants';
import {
  Button,
  Card,
  Divider,
  Dropdown,
  Input,
  Modal,
  Popover,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconCopy,
  IconSearch,
  IconEyeOpened,
  IconEdit,
  IconDelete,
  IconStop,
  IconPlay,
  IconMore,
} from '@douyinfe/semi-icons';
import EditRedemption from '../../pages/Redemption/EditRedemption';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

const RedemptionsTable = () => {
  const { t } = useTranslation();

  const renderStatus = (status) => {
    switch (status) {
      case 1:
        return (
          <Tag color='green' size='large' shape='circle'>
            {t('未使用')}
          </Tag>
        );
      case 2:
        return (
          <Tag color='red' size='large' shape='circle'>
            {t('已禁用')}
          </Tag>
        );
      case 3:
        return (
          <Tag color='grey' size='large' shape='circle'>
            {t('已使用')}
          </Tag>
        );
      default:
        return (
          <Tag color='black' size='large' shape='circle'>
            {t('未知状态')}
          </Tag>
        );
    }
  };

  const columns = [
    {
      title: t('ID'),
      dataIndex: 'id',
    },
    {
      title: t('名称'),
      dataIndex: 'name',
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      render: (text, record, index) => {
        return <div>{renderStatus(text)}</div>;
      },
    },
    {
      title: t('额度'),
      dataIndex: 'quota',
      render: (text, record, index) => {
        return <div>{renderQuota(parseInt(text))}</div>;
      },
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_time',
      render: (text, record, index) => {
        return <div>{renderTimestamp(text)}</div>;
      },
    },
    {
      title: t('兑换人ID'),
      dataIndex: 'used_user_id',
      render: (text, record, index) => {
        return <div>{text === 0 ? t('无') : text}</div>;
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      render: (text, record, index) => {
        // 创建更多操作的下拉菜单项
        const moreMenuItems = [
          {
            node: 'item',
            name: t('删除'),
            icon: <IconDelete />,
            type: 'danger',
            onClick: () => {
              Modal.confirm({
                title: t('确定是否要删除此兑换码？'),
                content: t('此修改将不可逆'),
                onOk: () => {
                  manageRedemption(record.id, 'delete', record).then(() => {
                    removeRecord(record.key);
                  });
                },
              });
            },
          }
        ];

        // 动态添加启用/禁用按钮
        if (record.status === 1) {
          moreMenuItems.push({
            node: 'item',
            name: t('禁用'),
            icon: <IconStop />,
            type: 'warning',
            onClick: () => {
              manageRedemption(record.id, 'disable', record);
            },
          });
        } else {
          moreMenuItems.push({
            node: 'item',
            name: t('启用'),
            icon: <IconPlay />,
            type: 'secondary',
            onClick: () => {
              manageRedemption(record.id, 'enable', record);
            },
            disabled: record.status === 3,
          });
        }

        return (
          <Space>
            <Popover content={record.key} style={{ padding: 20 }} position='top'>
              <Button
                icon={<IconEyeOpened />}
                theme='light'
                type='tertiary'
                size="small"
                className="!rounded-full"
              >
                {t('查看')}
              </Button>
            </Popover>
            <Button
              icon={<IconCopy />}
              theme='light'
              type='secondary'
              size="small"
              className="!rounded-full"
              onClick={async () => {
                await copyText(record.key);
              }}
            >
              {t('复制')}
            </Button>
            <Button
              icon={<IconEdit />}
              theme='light'
              type='tertiary'
              size="small"
              className="!rounded-full"
              onClick={() => {
                setEditingRedemption(record);
                setShowEdit(true);
              }}
              disabled={record.status !== 1}
            >
              {t('编辑')}
            </Button>
            <Dropdown
              trigger='click'
              position='bottomRight'
              menu={moreMenuItems}
            >
              <Button
                icon={<IconMore />}
                theme='light'
                type='tertiary'
                size="small"
                className="!rounded-full"
              />
            </Dropdown>
          </Space>
        );
      },
    },
  ];

  const [redemptions, setRedemptions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [tokenCount, setTokenCount] = useState(ITEMS_PER_PAGE);
  const [selectedKeys, setSelectedKeys] = useState([]);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [editingRedemption, setEditingRedemption] = useState({
    id: undefined,
  });
  const [showEdit, setShowEdit] = useState(false);

  const closeEdit = () => {
    setShowEdit(false);
    setTimeout(() => {
      setEditingRedemption({
        id: undefined,
      });
    }, 500);
  };

  const setRedemptionFormat = (redeptions) => {
    setRedemptions(redeptions);
  };

  const loadRedemptions = async (startIdx, pageSize) => {
    const res = await API.get(
      `/api/redemption/?p=${startIdx}&page_size=${pageSize}`,
    );
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setTokenCount(data.total);
      setRedemptionFormat(newPageData);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const removeRecord = (key) => {
    let newDataSource = [...redemptions];
    if (key != null) {
      let idx = newDataSource.findIndex((data) => data.key === key);

      if (idx > -1) {
        newDataSource.splice(idx, 1);
        setRedemptions(newDataSource);
      }
    }
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制到剪贴板！'));
    } else {
      Modal.error({
        title: t('无法复制到剪贴板，请手动复制'),
        content: text,
        size: 'large'
      });
    }
  };

  const onPaginationChange = (e, { activePage }) => {
    (async () => {
      if (activePage === Math.ceil(redemptions.length / pageSize) + 1) {
        await loadRedemptions(activePage - 1, pageSize);
      }
      setActivePage(activePage);
    })();
  };

  useEffect(() => {
    loadRedemptions(0, pageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, [pageSize]);

  const refresh = async () => {
    await loadRedemptions(activePage - 1, pageSize);
  };

  const manageRedemption = async (id, action, record) => {
    setLoading(true);
    let data = { id };
    let res;
    switch (action) {
      case 'delete':
        res = await API.delete(`/api/redemption/${id}/`);
        break;
      case 'enable':
        data.status = 1;
        res = await API.put('/api/redemption/?status_only=true', data);
        break;
      case 'disable':
        data.status = 2;
        res = await API.put('/api/redemption/?status_only=true', data);
        break;
    }
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作成功完成！'));
      let redemption = res.data.data;
      let newRedemptions = [...redemptions];
      if (action === 'delete') {
      } else {
        record.status = redemption.status;
      }
      setRedemptions(newRedemptions);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const searchRedemptions = async (keyword, page, pageSize) => {
    if (searchKeyword === '') {
      await loadRedemptions(page, pageSize);
      return;
    }
    setSearching(true);
    const res = await API.get(
      `/api/redemption/search?keyword=${keyword}&p=${page}&page_size=${pageSize}`,
    );
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setTokenCount(data.total);
      setRedemptionFormat(newPageData);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  const handleKeywordChange = async (value) => {
    setSearchKeyword(value.trim());
  };

  const sortRedemption = (key) => {
    if (redemptions.length === 0) return;
    setLoading(true);
    let sortedRedemptions = [...redemptions];
    sortedRedemptions.sort((a, b) => {
      return ('' + a[key]).localeCompare(b[key]);
    });
    if (sortedRedemptions[0].id === redemptions[0].id) {
      sortedRedemptions.reverse();
    }
    setRedemptions(sortedRedemptions);
    setLoading(false);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    if (searchKeyword === '') {
      loadRedemptions(page, pageSize).then();
    } else {
      searchRedemptions(searchKeyword, page, pageSize).then();
    }
  };

  let pageData = redemptions;
  const rowSelection = {
    onSelect: (record, selected) => { },
    onSelectAll: (selected, selectedRows) => { },
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedKeys(selectedRows);
    },
  };

  const handleRow = (record, index) => {
    if (record.status !== 1) {
      return {
        style: {
          background: 'var(--semi-color-disabled-border)',
        },
      };
    } else {
      return {};
    }
  };

  const renderHeader = () => (
    <div className="flex flex-col w-full">
      <div className="mb-2">
        <div className="flex items-center text-orange-500">
          <IconEyeOpened className="mr-2" />
          <Text>{t('兑换码可以批量生成和分发，适合用于推广活动或批量充值。')}</Text>
        </div>
      </div>

      <Divider margin="12px" />

      <div className="flex flex-col md:flex-row justify-between items-center gap-4 w-full">
        <div className="flex gap-2 w-full md:w-auto order-2 md:order-1">
          <Button
            theme='light'
            type='primary'
            icon={<IconPlus />}
            className="!rounded-full w-full md:w-auto"
            onClick={() => {
              setEditingRedemption({
                id: undefined,
              });
              setShowEdit(true);
            }}
          >
            {t('添加兑换码')}
          </Button>
          <Button
            type='warning'
            icon={<IconCopy />}
            className="!rounded-full w-full md:w-auto"
            onClick={async () => {
              if (selectedKeys.length === 0) {
                showError(t('请至少选择一个兑换码！'));
                return;
              }
              let keys = '';
              for (let i = 0; i < selectedKeys.length; i++) {
                keys +=
                  selectedKeys[i].name + '    ' + selectedKeys[i].key + '\n';
              }
              await copyText(keys);
            }}
          >
            {t('复制所选兑换码到剪贴板')}
          </Button>
        </div>

        <div className="flex flex-col md:flex-row items-center gap-4 w-full md:w-auto order-1 md:order-2">
          <div className="relative w-full md:w-64">
            <Input
              prefix={<IconSearch />}
              placeholder={t('关键字(id或者名称)')}
              value={searchKeyword}
              onChange={handleKeywordChange}
              className="!rounded-full"
              showClear
            />
          </div>
          <Button
            type="primary"
            onClick={() => {
              searchRedemptions(searchKeyword, 1, pageSize).then();
            }}
            loading={searching}
            className="!rounded-full w-full md:w-auto"
          >
            {t('查询')}
          </Button>
        </div>
      </div>
    </div>
  );

  return (
    <>
      <EditRedemption
        refresh={refresh}
        editingRedemption={editingRedemption}
        visiable={showEdit}
        handleClose={closeEdit}
      ></EditRedemption>

      <Card
        className="!rounded-2xl"
        title={renderHeader()}
        shadows='always'
        bordered={false}
      >
        <Table
          columns={columns}
          dataSource={pageData}
          scroll={{ x: 'max-content' }}
          pagination={{
            currentPage: activePage,
            pageSize: pageSize,
            total: tokenCount,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            formatPageText: (page) =>
              t('第 {{start}} - {{end}} 条，共 {{total}} 条', {
                start: page.currentStart,
                end: page.currentEnd,
                total: tokenCount,
              }),
            onPageSizeChange: (size) => {
              setPageSize(size);
              setActivePage(1);
              if (searchKeyword === '') {
                loadRedemptions(1, size).then();
              } else {
                searchRedemptions(searchKeyword, 1, size).then();
              }
            },
            onPageChange: handlePageChange,
          }}
          loading={loading}
          rowSelection={rowSelection}
          onRow={handleRow}
          className="rounded-xl overflow-hidden"
          size="middle"
        ></Table>
      </Card>
    </>
  );
};

export default RedemptionsTable;
