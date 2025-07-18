import React from 'react';
import {
  Button,
  Dropdown,
  Space,
  Tag,
  Tooltip,
  Typography
} from '@douyinfe/semi-ui';
import {
  User,
  Shield,
  Crown,
  HelpCircle,
  CheckCircle,
  XCircle,
  Minus,
  Coins,
  Activity,
  Users,
  DollarSign,
  UserPlus,
} from 'lucide-react';
import { IconMore } from '@douyinfe/semi-icons';
import { renderGroup, renderNumber, renderQuota } from '../../../helpers';

const { Text } = Typography;

/**
 * Render user role
 */
const renderRole = (role, t) => {
  switch (role) {
    case 1:
      return (
        <Tag color='blue' shape='circle' prefixIcon={<User size={14} />}>
          {t('普通用户')}
        </Tag>
      );
    case 10:
      return (
        <Tag color='yellow' shape='circle' prefixIcon={<Shield size={14} />}>
          {t('管理员')}
        </Tag>
      );
    case 100:
      return (
        <Tag color='orange' shape='circle' prefixIcon={<Crown size={14} />}>
          {t('超级管理员')}
        </Tag>
      );
    default:
      return (
        <Tag color='red' shape='circle' prefixIcon={<HelpCircle size={14} />}>
          {t('未知身份')}
        </Tag>
      );
  }
};

/**
 * Render user status
 */
const renderStatus = (status, t) => {
  switch (status) {
    case 1:
      return <Tag color='green' shape='circle' prefixIcon={<CheckCircle size={14} />}>{t('已激活')}</Tag>;
    case 2:
      return (
        <Tag color='red' shape='circle' prefixIcon={<XCircle size={14} />}>
          {t('已封禁')}
        </Tag>
      );
    default:
      return (
        <Tag color='grey' shape='circle' prefixIcon={<HelpCircle size={14} />}>
          {t('未知状态')}
        </Tag>
      );
  }
};

/**
 * Render username with remark
 */
const renderUsername = (text, record) => {
  const remark = record.remark;
  if (!remark) {
    return <span>{text}</span>;
  }
  const maxLen = 10;
  const displayRemark = remark.length > maxLen ? remark.slice(0, maxLen) + '…' : remark;
  return (
    <Space spacing={2}>
      <span>{text}</span>
      <Tooltip content={remark} position="top" showArrow>
        <Tag color='white' shape='circle' className="!text-xs">
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 flex-shrink-0 rounded-full" style={{ backgroundColor: '#10b981' }} />
            {displayRemark}
          </div>
        </Tag>
      </Tooltip>
    </Space>
  );
};

/**
 * Render user statistics
 */
const renderStatistics = (text, record, t) => {
  return (
    <div>
      <Space spacing={1}>
        <Tag color='white' shape='circle' className="!text-xs" prefixIcon={<Coins size={14} />}>
          {t('剩余')}: {renderQuota(record.quota)}
        </Tag>
        <Tag color='white' shape='circle' className="!text-xs" prefixIcon={<Coins size={14} />}>
          {t('已用')}: {renderQuota(record.used_quota)}
        </Tag>
        <Tag color='white' shape='circle' className="!text-xs" prefixIcon={<Activity size={14} />}>
          {t('调用')}: {renderNumber(record.request_count)}
        </Tag>
      </Space>
    </div>
  );
};

/**
 * Render invite information
 */
const renderInviteInfo = (text, record, t) => {
  return (
    <div>
      <Space spacing={1}>
        <Tag color='white' shape='circle' className="!text-xs" prefixIcon={<Users size={14} />}>
          {t('邀请')}: {renderNumber(record.aff_count)}
        </Tag>
        <Tag color='white' shape='circle' className="!text-xs" prefixIcon={<DollarSign size={14} />}>
          {t('收益')}: {renderQuota(record.aff_history_quota)}
        </Tag>
        <Tag color='white' shape='circle' className="!text-xs" prefixIcon={<UserPlus size={14} />}>
          {record.inviter_id === 0 ? t('无邀请人') : `邀请人: ${record.inviter_id}`}
        </Tag>
      </Space>
    </div>
  );
};

/**
 * Render overall status including deleted status
 */
const renderOverallStatus = (status, record, t) => {
  if (record.DeletedAt !== null) {
    return <Tag color='red' shape='circle' prefixIcon={<Minus size={14} />}>{t('已注销')}</Tag>;
  } else {
    return renderStatus(status, t);
  }
};

/**
 * Render operations column
 */
const renderOperations = (text, record, {
  setEditingUser,
  setShowEditUser,
  showPromoteModal,
  showDemoteModal,
  showEnableDisableModal,
  showDeleteModal,
  t
}) => {
  if (record.DeletedAt !== null) {
    return <></>;
  }

  // Create more operations dropdown menu items
  const moreMenuItems = [
    {
      node: 'item',
      name: t('提升'),
      type: 'warning',
      onClick: () => showPromoteModal(record),
    },
    {
      node: 'item',
      name: t('降级'),
      type: 'secondary',
      onClick: () => showDemoteModal(record),
    },
    {
      node: 'item',
      name: t('注销'),
      type: 'danger',
      onClick: () => showDeleteModal(record),
    }
  ];

  // Add enable/disable button dynamically
  if (record.status === 1) {
    moreMenuItems.splice(-1, 0, {
      node: 'item',
      name: t('禁用'),
      type: 'warning',
      onClick: () => showEnableDisableModal(record, 'disable'),
    });
  } else {
    moreMenuItems.splice(-1, 0, {
      node: 'item',
      name: t('启用'),
      type: 'secondary',
      onClick: () => showEnableDisableModal(record, 'enable'),
      disabled: record.status === 3,
    });
  }

  return (
    <Space>
      <Button
        type='tertiary'
        size="small"
        onClick={() => {
          setEditingUser(record);
          setShowEditUser(true);
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
          type='tertiary'
          size="small"
          icon={<IconMore />}
        />
      </Dropdown>
    </Space>
  );
};

/**
 * Get users table column definitions
 */
export const getUsersColumns = ({
  t,
  setEditingUser,
  setShowEditUser,
  showPromoteModal,
  showDemoteModal,
  showEnableDisableModal,
  showDeleteModal
}) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      render: (text, record) => renderUsername(text, record),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text, record, index) => {
        return <div>{renderGroup(text)}</div>;
      },
    },
    {
      title: t('统计信息'),
      dataIndex: 'info',
      render: (text, record, index) => renderStatistics(text, record, t),
    },
    {
      title: t('邀请信息'),
      dataIndex: 'invite',
      render: (text, record, index) => renderInviteInfo(text, record, t),
    },
    {
      title: t('角色'),
      dataIndex: 'role',
      render: (text, record, index) => {
        return <div>{renderRole(text, t)}</div>;
      },
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (text, record, index) => renderOverallStatus(text, record, t),
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      render: (text, record, index) => renderOperations(text, record, {
        setEditingUser,
        setShowEditUser,
        showPromoteModal,
        showDemoteModal,
        showEnableDisableModal,
        showDeleteModal,
        t
      }),
    },
  ];
}; 