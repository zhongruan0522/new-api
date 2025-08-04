/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Button,
  Table,
  Tag,
  Typography,
  Space,
  Tooltip,
  Popconfirm,
  Empty,
  Spin,
  Banner
} from '@douyinfe/semi-ui';
import { 
  IconRefresh,
  IconDelete,
  IconClose,
  IconSave,
  IconSetting
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess, timestamp2string } from '../../../../helpers/index.js';

const { Text, Title } = Typography;

const MultiKeyManageModal = ({
  visible,
  onCancel,
  channel,
  onRefresh
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [keyStatusList, setKeyStatusList] = useState([]);
  const [operationLoading, setOperationLoading] = useState({});

  // Load key status data
  const loadKeyStatus = async () => {
    if (!channel?.id) return;
    
    setLoading(true);
    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'get_key_status'
      });
      
      if (res.data.success) {
        setKeyStatusList(res.data.data.keys || []);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('获取密钥状态失败'));
    } finally {
      setLoading(false);
    }
  };

  // Disable a specific key
  const handleDisableKey = async (keyIndex) => {
    const operationId = `disable_${keyIndex}`;
    setOperationLoading(prev => ({ ...prev, [operationId]: true }));
    
    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'disable_key',
        key_index: keyIndex
      });
      
      if (res.data.success) {
        showSuccess(t('密钥已禁用'));
        await loadKeyStatus(); // Reload data
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('禁用密钥失败'));
    } finally {
      setOperationLoading(prev => ({ ...prev, [operationId]: false }));
    }
  };

  // Enable a specific key
  const handleEnableKey = async (keyIndex) => {
    const operationId = `enable_${keyIndex}`;
    setOperationLoading(prev => ({ ...prev, [operationId]: true }));
    
    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'enable_key',
        key_index: keyIndex
      });
      
      if (res.data.success) {
        showSuccess(t('密钥已启用'));
        await loadKeyStatus(); // Reload data
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('启用密钥失败'));
    } finally {
      setOperationLoading(prev => ({ ...prev, [operationId]: false }));
    }
  };

  // Delete all disabled keys
  const handleDeleteDisabledKeys = async () => {
    setOperationLoading(prev => ({ ...prev, delete_disabled: true }));
    
    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'delete_disabled_keys'
      });
      
      if (res.data.success) {
        showSuccess(res.data.message);
        await loadKeyStatus(); // Reload data
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('删除禁用密钥失败'));
    } finally {
      setOperationLoading(prev => ({ ...prev, delete_disabled: false }));
    }
  };

  // Effect to load data when modal opens
  useEffect(() => {
    if (visible && channel?.id) {
      loadKeyStatus();
    }
  }, [visible, channel?.id]);

  // Get status tag component
  const renderStatusTag = (status) => {
    switch (status) {
      case 1:
        return <Tag color='green' shape='circle'>{t('已启用')}</Tag>;
      case 2:
        return <Tag color='red' shape='circle'>{t('已禁用')}</Tag>;
      case 3:
        return <Tag color='orange' shape='circle'>{t('自动禁用')}</Tag>;
      default:
        return <Tag color='grey' shape='circle'>{t('未知状态')}</Tag>;
    }
  };

  // Table columns definition
  const columns = [
    {
      title: t('索引'),
      dataIndex: 'index',
      render: (text) => `#${text}`,
    },
    {
      title: t('密钥预览'),
      dataIndex: 'key_preview',
      render: (text) => (
        <Text code style={{ fontSize: '12px' }}>
          {text}
        </Text>
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (status) => renderStatusTag(status),
    },
    {
      title: t('禁用原因'),
      dataIndex: 'reason',
      width: 220,
      render: (reason, record) => {
        if (record.status === 1 || !reason) {
          return <Text type='quaternary'>-</Text>;
        }
        return (
          <Tooltip content={reason}>
            <Text style={{ maxWidth: '200px', display: 'block' }} ellipsis>
              {reason}
            </Text>
          </Tooltip>
        );
      },
    },
    {
      title: t('禁用时间'),
      dataIndex: 'disabled_time',
      width: 150,
      render: (time, record) => {
        if (record.status === 1 || !time) {
          return <Text type='quaternary'>-</Text>;
        }
        return (
          <Tooltip content={timestamp2string(time)}>
            <Text style={{ fontSize: '12px' }}>
              {timestamp2string(time)}
            </Text>
          </Tooltip>
        );
      },
    },
    {
      title: t('操作'),
      key: 'action',
      width: 120,
      render: (_, record) => (
        <Space>
          {record.status === 1 ? (
            <Popconfirm
              title={t('确定要禁用此密钥吗？')}
              content={t('禁用后该密钥将不再被使用')}
              onConfirm={() => handleDisableKey(record.index)}
            >
              <Button
                type='danger'
                size='small'
                loading={operationLoading[`disable_${record.index}`]}
              >
                {t('禁用')}
              </Button>
            </Popconfirm>
          ) : (
            <Popconfirm
              title={t('确定要启用此密钥吗？')}
              content={t('启用后该密钥将重新被使用')}
              onConfirm={() => handleEnableKey(record.index)}
            >
              <Button
                type='primary'
                size='small'
                loading={operationLoading[`enable_${record.index}`]}
              >
                {t('启用')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // Calculate statistics
  const enabledCount = keyStatusList.filter(key => key.status === 1).length;
  const manualDisabledCount = keyStatusList.filter(key => key.status === 2).length;
  const autoDisabledCount = keyStatusList.filter(key => key.status === 3).length;
  const totalCount = keyStatusList.length;

  return (
    <Modal
      title={
        <Space>
          <IconSetting />
          <span>{t('多密钥管理')} - {channel?.name}</span>
        </Space>
      }
      visible={visible}
      onCancel={onCancel}
      width={800}
      height={600}
      footer={
        <Space>
          <Button onClick={onCancel}>{t('关闭')}</Button>
          <Button
            icon={<IconRefresh />}
            onClick={loadKeyStatus}
            loading={loading}
          >
            {t('刷新')}
          </Button>
          {autoDisabledCount > 0 && (
            <Popconfirm
              title={t('确定要删除所有已自动禁用的密钥吗？')}
              content={t('此操作不可撤销，将永久删除已自动禁用的密钥')}
              onConfirm={handleDeleteDisabledKeys}
            >
              <Button
                type='danger'
                icon={<IconDelete />}
                loading={operationLoading.delete_disabled}
              >
                {t('删除自动禁用密钥')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      }
    >
      <div style={{ padding: '16px 0' }}>
        {/* Statistics Banner */}
        <Banner
          type='info'
          style={{ marginBottom: '16px' }}
          description={
            <div>
              <Text>
                {t('总共 {{total}} 个密钥，{{enabled}} 个已启用，{{manual}} 个手动禁用，{{auto}} 个自动禁用', {
                  total: totalCount,
                  enabled: enabledCount,
                  manual: manualDisabledCount,
                  auto: autoDisabledCount
                })}
              </Text>
              {channel?.channel_info?.multi_key_mode && (
                <div style={{ marginTop: '4px' }}>
                  <Text type='quaternary' style={{ fontSize: '12px' }}>
                    {t('多密钥模式')}: {channel.channel_info.multi_key_mode === 'random' ? t('随机') : t('轮询')}
                  </Text>
                </div>
              )}
            </div>
          }
        />

        {/* Key Status Table */}
        <Spin spinning={loading}>
          {keyStatusList.length > 0 ? (
            <Table
              columns={columns}
              dataSource={keyStatusList}
              pagination={false}
              size='small'
              bordered
              rowKey='index'
              style={{ maxHeight: '400px', overflow: 'auto' }}
            />
          ) : (
            !loading && (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                title={t('暂无密钥数据')}
                description={t('请检查渠道配置或刷新重试')}
              />
            )
          )}
        </Spin>
      </div>
    </Modal>
  );
};

export default MultiKeyManageModal; 