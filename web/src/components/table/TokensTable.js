import React, { useEffect, useState } from 'react';
import {
  API,
  copy,
  showError,
  showSuccess,
  timestamp2string,
  renderGroup,
  renderQuota
} from '../../helpers';

import { ITEMS_PER_PAGE } from '../../constants';
import {
  Button,
  Card,
  Dropdown,
  Modal,
  Space,
  SplitButtonGroup,
  Table,
  Tag,
  Input,
} from '@douyinfe/semi-ui';

import {
  IconPlus,
  IconCopy,
  IconSearch,
  IconTreeTriangleDown,
  IconEyeOpened,
  IconEdit,
  IconDelete,
  IconStop,
  IconPlay,
  IconMore,
} from '@douyinfe/semi-icons';
import EditToken from '../../pages/Token/EditToken';
import { useTranslation } from 'react-i18next';

function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

const TokensTable = () => {
  const { t } = useTranslation();

  const renderStatus = (status, model_limits_enabled = false) => {
    switch (status) {
      case 1:
        if (model_limits_enabled) {
          return (
            <Tag color='green' size='large' shape='circle'>
              {t('已启用：限制模型')}
            </Tag>
          );
        } else {
          return (
            <Tag color='green' size='large' shape='circle'>
              {t('已启用')}
            </Tag>
          );
        }
      case 2:
        return (
          <Tag color='red' size='large' shape='circle'>
            {t('已禁用')}
          </Tag>
        );
      case 3:
        return (
          <Tag color='yellow' size='large' shape='circle'>
            {t('已过期')}
          </Tag>
        );
      case 4:
        return (
          <Tag color='grey' size='large' shape='circle'>
            {t('已耗尽')}
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
      title: t('名称'),
      dataIndex: 'name',
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      render: (text, record, index) => {
        return (
          <div>
            <Space>
              {renderStatus(text, record.model_limits_enabled)}
              {renderGroup(record.group)}
            </Space>
          </div>
        );
      },
    },
    {
      title: t('已用额度'),
      dataIndex: 'used_quota',
      render: (text, record, index) => {
        return <div>{renderQuota(parseInt(text))}</div>;
      },
    },
    {
      title: t('剩余额度'),
      dataIndex: 'remain_quota',
      render: (text, record, index) => {
        return (
          <div>
            {record.unlimited_quota ? (
              <Tag size={'large'} color={'white'} shape='circle'>
                {t('无限制')}
              </Tag>
            ) : (
              <Tag size={'large'} color={'light-blue'} shape='circle'>
                {renderQuota(parseInt(text))}
              </Tag>
            )}
          </div>
        );
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
      title: t('过期时间'),
      dataIndex: 'expired_time',
      render: (text, record, index) => {
        return (
          <div>
            {record.expired_time === -1 ? t('永不过期') : renderTimestamp(text)}
          </div>
        );
      },
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      render: (text, record, index) => {
        let chats = localStorage.getItem('chats');
        let chatsArray = [];
        let shouldUseCustom = true;

        if (shouldUseCustom) {
          try {
            chats = JSON.parse(chats);
            if (Array.isArray(chats)) {
              for (let i = 0; i < chats.length; i++) {
                let chat = {};
                chat.node = 'item';
                for (let key in chats[i]) {
                  if (chats[i].hasOwnProperty(key)) {
                    chat.key = i;
                    chat.name = key;
                    chat.onClick = () => {
                      onOpenLink(key, chats[i][key], record);
                    };
                  }
                }
                chatsArray.push(chat);
              }
            }
          } catch (e) {
            console.log(e);
            showError(t('聊天链接配置错误，请联系管理员'));
          }
        }

        // 创建更多操作的下拉菜单项
        const moreMenuItems = [
          {
            node: 'item',
            name: t('查看'),
            icon: <IconEyeOpened />,
            onClick: () => {
              Modal.info({
                title: t('令牌详情'),
                content: 'sk-' + record.key,
                size: 'large',
              });
            },
          },
          {
            node: 'item',
            name: t('删除'),
            icon: <IconDelete />,
            type: 'danger',
            onClick: () => {
              Modal.confirm({
                title: t('确定是否要删除此令牌？'),
                content: t('此修改将不可逆'),
                onOk: () => {
                  manageToken(record.id, 'delete', record).then(() => {
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
              manageToken(record.id, 'disable', record);
            },
          });
        } else {
          moreMenuItems.push({
            node: 'item',
            name: t('启用'),
            icon: <IconPlay />,
            type: 'secondary',
            onClick: () => {
              manageToken(record.id, 'enable', record);
            },
          });
        }

        return (
          <Space wrap>
            <SplitButtonGroup
              className="!rounded-full overflow-hidden"
              aria-label={t('项目操作按钮组')}
            >
              <Button
                theme='light'
                size="small"
                style={{ color: 'rgba(var(--semi-teal-7), 1)' }}
                onClick={() => {
                  if (chatsArray.length === 0) {
                    showError(t('请联系管理员配置聊天链接'));
                  } else {
                    onOpenLink(
                      'default',
                      chats[0][Object.keys(chats[0])[0]],
                      record,
                    );
                  }
                }}
              >
                {t('聊天')}
              </Button>
              <Dropdown
                trigger='click'
                position='bottomRight'
                menu={chatsArray}
              >
                <Button
                  style={{
                    padding: '4px 4px',
                    color: 'rgba(var(--semi-teal-7), 1)',
                  }}
                  type='primary'
                  icon={<IconTreeTriangleDown />}
                  size="small"
                ></Button>
              </Dropdown>
            </SplitButtonGroup>

            <Button
              icon={<IconCopy />}
              theme='light'
              type='secondary'
              size="small"
              className="!rounded-full"
              onClick={async (text) => {
                await copyText('sk-' + record.key);
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
                setEditingToken(record);
                setShowEdit(true);
              }}
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

  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [showEdit, setShowEdit] = useState(false);
  const [tokens, setTokens] = useState([]);
  const [selectedKeys, setSelectedKeys] = useState([]);
  const [tokenCount, setTokenCount] = useState(pageSize);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchToken, setSearchToken] = useState('');
  const [searching, setSearching] = useState(false);
  const [chats, setChats] = useState([]);
  const [editingToken, setEditingToken] = useState({
    id: undefined,
  });

  const closeEdit = () => {
    setShowEdit(false);
    setTimeout(() => {
      setEditingToken({
        id: undefined,
      });
    }, 500);
  };

  const setTokensFormat = (tokens) => {
    setTokens(tokens);
    if (tokens.length >= pageSize) {
      setTokenCount(tokens.length + pageSize);
    } else {
      setTokenCount(tokens.length);
    }
  };

  let pageData = tokens.slice(
    (activePage - 1) * pageSize,
    activePage * pageSize,
  );
  const loadTokens = async (startIdx) => {
    setLoading(true);
    const res = await API.get(`/api/token/?p=${startIdx}&size=${pageSize}`);
    const { success, message, data } = res.data;
    if (success) {
      if (startIdx === 0) {
        setTokensFormat(data);
      } else {
        let newTokens = [...tokens];
        newTokens.splice(startIdx * pageSize, data.length, ...data);
        setTokensFormat(newTokens);
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const refresh = async () => {
    await loadTokens(activePage - 1);
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess(t('已复制到剪贴板！'));
    } else {
      Modal.error({
        title: t('无法复制到剪贴板，请手动复制'),
        content: text,
        size: 'large',
      });
    }
  };

  const onOpenLink = async (type, url, record) => {
    let status = localStorage.getItem('status');
    let serverAddress = '';
    if (status) {
      status = JSON.parse(status);
      serverAddress = status.server_address;
    }
    if (serverAddress === '') {
      serverAddress = window.location.origin;
    }
    let encodedServerAddress = encodeURIComponent(serverAddress);
    url = url.replaceAll('{address}', encodedServerAddress);
    url = url.replaceAll('{key}', 'sk-' + record.key);

    window.open(url, '_blank');
  };



  useEffect(() => {
    loadTokens(0)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, [pageSize]);

  const removeRecord = (key) => {
    let newDataSource = [...tokens];
    if (key != null) {
      let idx = newDataSource.findIndex((data) => data.key === key);

      if (idx > -1) {
        newDataSource.splice(idx, 1);
        setTokensFormat(newDataSource);
      }
    }
  };

  const manageToken = async (id, action, record) => {
    setLoading(true);
    let data = { id };
    let res;
    switch (action) {
      case 'delete':
        res = await API.delete(`/api/token/${id}/`);
        break;
      case 'enable':
        data.status = 1;
        res = await API.put('/api/token/?status_only=true', data);
        break;
      case 'disable':
        data.status = 2;
        res = await API.put('/api/token/?status_only=true', data);
        break;
    }
    const { success, message } = res.data;
    if (success) {
      showSuccess('操作成功完成！');
      let token = res.data.data;
      let newTokens = [...tokens];
      if (action === 'delete') {
      } else {
        record.status = token.status;
      }
      setTokensFormat(newTokens);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const searchTokens = async () => {
    if (searchKeyword === '' && searchToken === '') {
      await loadTokens(0);
      setActivePage(1);
      return;
    }
    setSearching(true);
    const res = await API.get(
      `/api/token/search?keyword=${searchKeyword}&token=${searchToken}`,
    );
    const { success, message, data } = res.data;
    if (success) {
      setTokensFormat(data);
      setActivePage(1);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  const handleKeywordChange = async (value) => {
    setSearchKeyword(value.trim());
  };

  const handleSearchTokenChange = async (value) => {
    setSearchToken(value.trim());
  };

  const sortToken = (key) => {
    if (tokens.length === 0) return;
    setLoading(true);
    let sortedTokens = [...tokens];
    sortedTokens.sort((a, b) => {
      return ('' + a[key]).localeCompare(b[key]);
    });
    if (sortedTokens[0].id === tokens[0].id) {
      sortedTokens.reverse();
    }
    setTokens(sortedTokens);
    setLoading(false);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    if (page === Math.ceil(tokens.length / pageSize) + 1) {
      loadTokens(page - 1).then((r) => { });
    }
  };

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
      <div className="flex flex-col md:flex-row justify-between items-center gap-4 w-full">
        <div className="flex gap-2 w-full md:w-auto order-2 md:order-1">
          <Button
            theme="light"
            type="primary"
            icon={<IconPlus />}
            className="!rounded-full w-full md:w-auto"
            onClick={() => {
              setEditingToken({
                id: undefined,
              });
              setShowEdit(true);
            }}
          >
            {t('添加令牌')}
          </Button>
          <Button
            theme="light"
            type="warning"
            icon={<IconCopy />}
            className="!rounded-full w-full md:w-auto"
            onClick={async () => {
              if (selectedKeys.length === 0) {
                showError(t('请至少选择一个令牌！'));
                return;
              }
              let keys = '';
              for (let i = 0; i < selectedKeys.length; i++) {
                keys +=
                  selectedKeys[i].name + '    sk-' + selectedKeys[i].key + '\n';
              }
              await copyText(keys);
            }}
          >
            {t('复制所选令牌到剪贴板')}
          </Button>
        </div>

        <div className="flex flex-col md:flex-row items-center gap-4 w-full md:w-auto order-1 md:order-2">
          <div className="relative w-full md:w-56">
            <Input
              prefix={<IconSearch />}
              placeholder={t('搜索关键字')}
              value={searchKeyword}
              onChange={handleKeywordChange}
              className="!rounded-full"
              showClear
            />
          </div>
          <div className="relative w-full md:w-56">
            <Input
              prefix={<IconSearch />}
              placeholder={t('密钥')}
              value={searchToken}
              onChange={handleSearchTokenChange}
              className="!rounded-full"
              showClear
            />
          </div>
          <Button
            type="primary"
            onClick={searchTokens}
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
      <EditToken
        refresh={refresh}
        editingToken={editingToken}
        visiable={showEdit}
        handleClose={closeEdit}
      ></EditToken>

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
                total: tokens.length,
              }),
            onPageSizeChange: (size) => {
              setPageSize(size);
              setActivePage(1);
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

export default TokensTable;
