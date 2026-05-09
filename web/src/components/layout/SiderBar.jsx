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

import React, { useEffect, useMemo, useState, useContext, useCallback } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getLucideIcon } from '../../helpers/render';
import { ChevronLeft } from 'lucide-react';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import { useSidebar } from '../../hooks/common/useSidebar';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { isAdmin, isRoot, API, showSuccess } from '../../helpers';
import SkeletonWrapper from './components/SkeletonWrapper';
import NotificationButton from './headerbar/NotificationButton';
import ThemeToggle from './headerbar/ThemeToggle';
import UserArea from './headerbar/UserArea';
import NoticeModal from './NoticeModal';

import { Nav, Divider, Button } from '@douyinfe/semi-ui';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useTheme, useSetTheme, useActualTheme } from '../../context/Theme';
import { useNotifications } from '../../hooks/common/useNotifications';

const routerMap = {
  home: '/',
  channel: '/console/channel',
  token: '/console/token',
  redemption: '/console/redemption',
  topup: '/console/topup',
  user: '/console/user',
  log: '/console/log',
  multimodal_files: '/console/multimodal-files',
  setting: '/console/setting',
  about: '/about',
  detail: '/console',
  pricing: '/pricing',
  models: '/console/models',
  deployment: '/console/deployment',
  personal: '/console/personal',
  ticket: '/console/ticket',
};

const SiderBar = ({ onNavigate = () => {} }) => {
  const { t } = useTranslation();
  const [collapsed, toggleCollapsed] = useSidebarCollapsed();
  const {
    isModuleVisible,
    hasSectionVisibleModules,
    loading: sidebarLoading,
  } = useSidebar();
  const isMobile = useIsMobile();
  const navigate = useNavigate();

  // Mobile action buttons: use contexts directly
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const theme = useTheme();
  const setTheme = useSetTheme();
  const actualTheme = useActualTheme();

  const {
    noticeVisible,
    unreadCount,
    handleNoticeOpen,
    handleNoticeClose,
    getUnreadKeys,
  } = useNotifications(statusState);

  const handleThemeToggle = useCallback(
    (newTheme) => {
      if (
        !newTheme ||
        (newTheme !== 'light' && newTheme !== 'dark' && newTheme !== 'auto')
      ) {
        return;
      }
      setTheme(newTheme);
    },
    [setTheme],
  );

  const logout = useCallback(async () => {
    await API.get('/api/user/logout');
    showSuccess(t('注销成功!'));
    userDispatch({ type: 'logout' });
    localStorage.removeItem('user');
    navigate('/login');
  }, [navigate, t, userDispatch]);

  const isLoading = statusState?.status === undefined;

  const showSkeleton = useMinimumLoadingTime(sidebarLoading, 200);

  const [selectedKeys, setSelectedKeys] = useState(['home']);
  const location = useLocation();

  const workspaceItems = useMemo(() => {
    const items = [
      {
        text: t('数据看板'),
        itemKey: 'detail',
        to: '/detail',
        className:
          localStorage.getItem('enable_data_export') === 'true'
            ? ''
            : 'tableHiddle',
      },
      {
        text: t('令牌管理'),
        itemKey: 'token',
        to: '/token',
      },
      {
        text: t('使用日志'),
        itemKey: 'log',
        to: '/log',
      },
      {
        text: t('多模态文件'),
        itemKey: 'multimodal_files',
        to: '/multimodal-files',
      },
    ];

    // 根据配置过滤项目
    const filteredItems = items.filter((item) => {
      const configVisible = isModuleVisible('console', item.itemKey);
      return configVisible;
    });

    return filteredItems;
  }, [
    localStorage.getItem('enable_data_export'),
    t,
    isModuleVisible,
  ]);

  const financeItems = useMemo(() => {
    const items = [
      {
        text: t('钱包管理'),
        itemKey: 'topup',
        to: '/topup',
      },
      {
        text: t('个人设置'),
        itemKey: 'personal',
        to: '/personal',
      },
    ];

    // 根据配置过滤项目
    const filteredItems = items.filter((item) => {
      const configVisible = isModuleVisible('personal', item.itemKey);
      return configVisible;
    });

    return filteredItems;
  }, [t, isModuleVisible]);

  const supportItems = useMemo(() => {
    const items = [
      {
        text: t('工单列表'),
        itemKey: 'ticket',
        to: '/ticket',
      },
    ];

    const filteredItems = items.filter((item) => {
      const configVisible = isModuleVisible('support', item.itemKey);
      return configVisible;
    });

    return filteredItems;
  }, [t, isModuleVisible]);

  const userMaintenanceItems = useMemo(() => {
    const items = [
      {
        text: t('用户管理'),
        itemKey: 'user',
        to: '/user',
        className: isAdmin() ? '' : 'tableHiddle',
      },
      {
        text: t('兑换码管理'),
        itemKey: 'redemption',
        to: '/redemption',
        className: isAdmin() ? '' : 'tableHiddle',
      },
    ];

    const filteredItems = items.filter((item) => {
      const configVisible = isModuleVisible('user_maintenance', item.itemKey);
      return configVisible;
    });

    return filteredItems;
  }, [isAdmin(), t, isModuleVisible]);

  const adminItems = useMemo(() => {
    const items = [
      {
        text: t('渠道管理'),
        itemKey: 'channel',
        to: '/channel',
        className: isAdmin() ? '' : 'tableHiddle',
      },
      {
        text: t('模型管理'),
        itemKey: 'models',
        to: '/console/models',
        className: isAdmin() ? '' : 'tableHiddle',
      },
      {
        text: t('系统设置'),
        itemKey: 'setting',
        to: '/setting',
        className: isRoot() ? '' : 'tableHiddle',
      },
    ];

    // 根据配置过滤项目
    const filteredItems = items.filter((item) => {
      const configVisible = isModuleVisible('admin', item.itemKey);
      return configVisible;
    });

    return filteredItems;
  }, [isAdmin(), isRoot(), t, isModuleVisible]);

  // 根据当前路径设置选中的菜单项
  useEffect(() => {
    const currentPath = location.pathname;
    const matchingKey = Object.keys(routerMap).find(
      (key) => routerMap[key] === currentPath,
    );
    if (matchingKey) {
      setSelectedKeys([matchingKey]);
    }
  }, [location.pathname]);

  // 监控折叠状态变化以更新 body class
  useEffect(() => {
    if (collapsed) {
      document.body.classList.add('sidebar-collapsed');
    } else {
      document.body.classList.remove('sidebar-collapsed');
    }
  }, [collapsed]);

  // 选中高亮颜色（统一）
  const SELECTED_COLOR = 'var(--semi-color-primary)';

  // 渲染自定义菜单项
  const renderNavItem = (item) => {
    // 跳过隐藏的项目
    if (item.className === 'tableHiddle') return null;

    const isSelected = selectedKeys.includes(item.itemKey);
    const textColor = isSelected ? SELECTED_COLOR : 'inherit';

    return (
      <Nav.Item
        key={item.itemKey}
        itemKey={item.itemKey}
        text={
          <span
            className='truncate font-medium text-sm'
            style={{ color: textColor }}
          >
            {item.text}
          </span>
        }
        icon={
          <div className='sidebar-icon-container flex-shrink-0'>
            {getLucideIcon(item.itemKey, isSelected)}
          </div>
        }
        className={item.className}
      />
    );
  };

  return (
    <div
      className='sidebar-container'
      style={{
        width: 'var(--sidebar-current-width)',
      }}
    >
      <SkeletonWrapper
        loading={showSkeleton}
        type='sidebar'
        className=''
        collapsed={collapsed}
        showAdmin={isAdmin()}
      >
        <Nav
          className='sidebar-nav'
          defaultIsCollapsed={collapsed}
          isCollapsed={collapsed}
          onCollapseChange={toggleCollapsed}
          selectedKeys={selectedKeys}
          itemStyle='sidebar-nav-item'
          hoverStyle='sidebar-nav-item:hover'
          selectedStyle='sidebar-nav-item-selected'
          renderWrapper={({ itemElement, props }) => {
            const to = routerMap[props.itemKey];

            // 如果没有路由，直接返回元素
            if (!to) return itemElement;

            return (
              <Link
                style={{ textDecoration: 'none' }}
                to={to}
                onClick={onNavigate}
              >
                {itemElement}
              </Link>
            );
          }}
          onSelect={(key) => setSelectedKeys([key.itemKey])}
        >
          {/* 控制台区域 */}
          {hasSectionVisibleModules('console') && (
            <>
              <Divider className='sidebar-divider' />
              <div>
                {!collapsed && (
                  <div className='sidebar-group-label'>{t('控制台')}</div>
                )}
                {workspaceItems.map((item) => renderNavItem(item))}
              </div>
            </>
          )}

          {/* 客户支持区域 */}
          {hasSectionVisibleModules('support') && (
            <>
              <Divider className='sidebar-divider' />
              <div>
                {!collapsed && (
                  <div className='sidebar-group-label'>{t('客户支持')}</div>
                )}
                {supportItems.map((item) => renderNavItem(item))}
              </div>
            </>
          )}

          {/* 个人中心区域 */}
          {hasSectionVisibleModules('personal') && (
            <>
              <Divider className='sidebar-divider' />
              <div>
                {!collapsed && (
                  <div className='sidebar-group-label'>{t('个人中心')}</div>
                )}
                {financeItems.map((item) => renderNavItem(item))}
              </div>
            </>
          )}

          {/* 用户维护区域 - 只在管理员时显示 */}
          {isAdmin() && hasSectionVisibleModules('user_maintenance') && (
            <>
              <Divider className='sidebar-divider' />
              <div>
                {!collapsed && (
                  <div className='sidebar-group-label'>{t('用户维护')}</div>
                )}
                {userMaintenanceItems.map((item) => renderNavItem(item))}
              </div>
            </>
          )}

          {/* 管理员区域 - 只在管理员时显示且配置允许时显示 */}
          {isAdmin() && hasSectionVisibleModules('admin') && (
            <>
              <Divider className='sidebar-divider' />
              <div>
                {!collapsed && (
                  <div className='sidebar-group-label'>{t('管理员')}</div>
                )}
                {adminItems.map((item) => renderNavItem(item))}
              </div>
            </>
          )}
        </Nav>
      </SkeletonWrapper>

      {/* 移动端通知弹窗 */}
      {isMobile && (
        <NoticeModal
          visible={noticeVisible}
          onClose={handleNoticeClose}
          isMobile={isMobile}
          defaultTab={unreadCount > 0 ? 'system' : 'inApp'}
          unreadKeys={getUnreadKeys()}
        />
      )}

      {/* 移动端操作按钮区域 */}
      {isMobile && (
        <div className='flex items-center justify-center gap-2 px-3 py-2 border-t border-semi-color-border'>
          <NotificationButton
            unreadCount={unreadCount}
            onNoticeOpen={handleNoticeOpen}
            t={t}
          />
          <ThemeToggle theme={theme} onThemeToggle={handleThemeToggle} t={t} />
          <UserArea
            userState={userState}
            isLoading={isLoading}
            isMobile={isMobile}
            logout={logout}
            navigate={navigate}
            t={t}
          />
        </div>
      )}

      {/* 底部折叠按钮 */}
      <div className='sidebar-collapse-button'>
        <SkeletonWrapper
          loading={showSkeleton}
          type='button'
          width={collapsed ? 36 : 156}
          height={24}
          className='w-full'
        >
          <Button
            theme='outline'
            type='tertiary'
            size='small'
            icon={
              <ChevronLeft
                size={16}
                strokeWidth={2.5}
                color='var(--semi-color-text-2)'
                style={{
                  transform: collapsed ? 'rotate(180deg)' : 'rotate(0deg)',
                }}
              />
            }
            onClick={toggleCollapsed}
            icononly={collapsed}
            style={
              collapsed
                ? { width: 36, height: 24, padding: 0 }
                : { padding: '4px 12px', width: '100%' }
            }
          >
            {!collapsed ? t('收起侧边栏') : null}
          </Button>
        </SkeletonWrapper>
      </div>
    </div>
  );
};

export default SiderBar;
