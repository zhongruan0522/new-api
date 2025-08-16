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

import React from 'react';
import { Avatar, Card, Tag, Divider, Typography } from '@douyinfe/semi-ui';
import { isRoot, isAdmin, renderQuota, stringToColor } from '../../../../helpers';
import { Coins, BarChart2, Users } from 'lucide-react';

const UserInfoHeader = ({ t, userState }) => {

  const getUsername = () => {
    if (userState.user) {
      return userState.user.username;
    } else {
      return 'null';
    }
  };

  const getAvatarText = () => {
    const username = getUsername();
    if (username && username.length > 0) {
      // 获取前两个字符，支持中文和英文
      return username.slice(0, 2).toUpperCase();
    }
    return 'NA';
  };

  return (
    <Card className="!rounded-2xl">
      {/* 装饰性背景元素 */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute -top-10 -right-10 w-40 h-40 bg-slate-400 dark:bg-slate-500 opacity-5 rounded-full"></div>
        <div className="absolute -bottom-16 -left-16 w-48 h-48 bg-slate-300 dark:bg-slate-400 opacity-8 rounded-full"></div>
        <div className="absolute top-1/2 right-1/4 w-24 h-24 bg-slate-400 dark:bg-slate-500 opacity-6 rounded-full"></div>
      </div>

      <div className="relative text-gray-600 dark:text-gray-300">
        <div className="flex justify-between items-start mb-4 sm:mb-6">
          <div className="flex items-center flex-1 min-w-0">
            <Avatar
              size='large'
              className="mr-3 sm:mr-4 shadow-md flex-shrink-0"
              color={stringToColor(getUsername())}
            >
              {getAvatarText()}
            </Avatar>
            <div className="flex-1 min-w-0">
              <div className="text-base sm:text-lg font-semibold truncate text-gray-800 dark:text-gray-100">
                {getUsername()}
              </div>
              <div className="mt-1 flex flex-wrap gap-1 sm:gap-2">
                {isRoot() ? (
                  <Tag
                    size='small'
                    className="!rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300"
                    style={{ fontWeight: '500' }}
                  >
                    {t('超级管理员')}
                  </Tag>
                ) : isAdmin() ? (
                  <Tag
                    size='small'
                    className="!rounded-full bg-gray-50 dark:bg-gray-700 text-gray-600 dark:text-gray-300"
                    style={{ fontWeight: '500' }}
                  >
                    {t('管理员')}
                  </Tag>
                ) : (
                  <Tag
                    size='small'
                    className="!rounded-full bg-slate-50 dark:bg-slate-700 text-slate-600 dark:text-slate-300"
                    style={{ fontWeight: '500' }}
                  >
                    {t('普通用户')}
                  </Tag>
                )}
                <Tag
                  size='small'
                  className="!rounded-full bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300"
                  style={{ fontWeight: '500' }}
                >
                  ID: {userState?.user?.id}
                </Tag>
              </div>
            </div>
          </div>

          {/* 右上角统计信息（Semi UI 卡片） */}
          <div className="hidden sm:block flex-shrink-0 ml-2">
            <Card size="small" className="!rounded-xl shadow-sm" bodyStyle={{ padding: '8px 12px' }}>
              <div className="flex items-center gap-3 lg:gap-4">
                <div className="flex items-center justify-end gap-2">
                  <Coins size={16} className="text-slate-600 dark:text-slate-300" />
                  <div className="text-right">
                    <Typography.Text size="small" type="tertiary">{t('历史消耗')}</Typography.Text>
                    <div className="text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-100">{renderQuota(userState?.user?.used_quota)}</div>
                  </div>
                </div>
                <Divider layout="vertical" />
                <div className="flex items-center justify-end gap-2">
                  <BarChart2 size={16} className="text-slate-600 dark:text-slate-300" />
                  <div className="text-right">
                    <Typography.Text size="small" type="tertiary">{t('请求次数')}</Typography.Text>
                    <div className="text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-100">{userState.user?.request_count || 0}</div>
                  </div>
                </div>
                <Divider layout="vertical" />
                <div className="flex items-center justify-end gap-2">
                  <Users size={16} className="text-slate-600 dark:text-slate-300" />
                  <div className="text-right">
                    <Typography.Text size="small" type="tertiary">{t('用户分组')}</Typography.Text>
                    <div className="text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-100">{userState?.user?.group || t('默认')}</div>
                  </div>
                </div>
              </div>
            </Card>
          </div>
        </div>

        <div className="mb-4 sm:mb-6">
          <div className="text-xs sm:text-sm mb-1 sm:mb-2 text-gray-500 dark:text-gray-400">
            {t('当前余额')}
          </div>
          <div className="text-2xl sm:text-3xl md:text-4xl font-bold tracking-wide text-gray-900 dark:text-gray-100">
            {renderQuota(userState?.user?.quota)}
          </div>
        </div>

        {/* 移动端统计信息卡片（仅 xs 可见） */}
        <div className="sm:hidden">
          <Card size="small" className="!rounded-xl shadow-sm" bodyStyle={{ padding: '10px 12px' }}>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Coins size={16} className="text-slate-600" />
                  <Typography.Text size="small" type="tertiary">{t('历史消耗')}</Typography.Text>
                </div>
                <div className="text-sm font-semibold text-gray-800">{renderQuota(userState?.user?.used_quota)}</div>
              </div>
              <Divider margin='8px' />
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <BarChart2 size={16} className="text-slate-600" />
                  <Typography.Text size="small" type="tertiary">{t('请求次数')}</Typography.Text>
                </div>
                <div className="text-sm font-semibold text-gray-800">{userState.user?.request_count || 0}</div>
              </div>
              <Divider margin='8px' />
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Users size={16} className="text-slate-600" />
                  <Typography.Text size="small" type="tertiary">{t('用户分组')}</Typography.Text>
                </div>
                <div className="text-sm font-semibold text-gray-800">{userState?.user?.group || t('默认')}</div>
              </div>
            </div>
          </Card>
        </div>

        <div className="absolute top-0 left-0 w-full h-2 bg-gradient-to-r from-slate-300 via-slate-400 to-slate-500 dark:from-slate-600 dark:via-slate-500 dark:to-slate-400 opacity-40"></div>
      </div>
    </Card>
  );
};

export default UserInfoHeader;
