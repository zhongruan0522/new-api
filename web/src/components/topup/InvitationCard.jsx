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
import {
  Avatar,
  Typography,
  Card,
  Button,
  Input,
  Badge,
} from '@douyinfe/semi-ui';
import { Copy, Users, BarChart2, TrendingUp, Gift, Zap } from 'lucide-react';

const { Text } = Typography;

const InvitationCard = ({
  t,
  userState,
  renderQuota,
  setOpenTransfer,
  affLink,
  handleAffLinkClick,
}) => {
  return (
    <Card className="!rounded-2xl shadow-sm border-0">
      {/* 卡片头部 */}
      <div className="flex items-center mb-4">
        <Avatar size="small" color="green" className="mr-3 shadow-md">
          <Gift size={16} />
        </Avatar>
        <div>
          <Typography.Text className="text-lg font-medium">{t('邀请奖励')}</Typography.Text>
          <div className="text-xs text-gray-600 dark:text-gray-400">{t('邀请好友获得额外奖励')}</div>
        </div>
      </div>

      {/* 收益展示区域 */}
      <div className='space-y-4'>
        {/* 主要收益卡片 - 待使用收益 */}
        <Card className='!rounded-xl with-pastel-balls'>
          <div className='flex justify-between items-center mb-3'>
            <div className="flex items-center">
              <TrendingUp size={16} className="mr-2 text-slate-600 dark:text-slate-300" />
              <Text strong className='text-slate-700 dark:text-slate-200'>{t('待使用收益')}</Text>
            </div>
            <Button
              type='primary'
              theme='solid'
              size='small'
              disabled={!userState?.user?.aff_quota || userState?.user?.aff_quota <= 0}
              onClick={() => setOpenTransfer(true)}
              className='!rounded-lg !bg-slate-600 hover:!bg-slate-700'
            >
              <Zap size={12} className="mr-1" />
              {t('划转到余额')}
            </Button>
          </div>
          <div className='text-2xl sm:text-3xl font-bold text-slate-900 dark:text-slate-100 mb-1'>
            {renderQuota(userState?.user?.aff_quota || 0)}
          </div>
        </Card>

        {/* 统计数据网格 */}
        <div className='grid grid-cols-2 gap-4'>
          <Card className='!rounded-xl bg-slate-50 dark:bg-slate-800'>
            <div className='flex items-center mb-2'>
              <BarChart2 size={16} className='mr-2 text-slate-600 dark:text-slate-300' />
              <Text type='tertiary' className='text-slate-600 dark:text-slate-300'>{t('总收益')}</Text>
            </div>
            <div className='text-xl font-semibold text-slate-900 dark:text-slate-100'>
              {renderQuota(userState?.user?.aff_history_quota || 0)}
            </div>
          </Card>

          <Card className='!rounded-xl bg-slate-50 dark:bg-slate-800'>
            <div className='flex items-center mb-2'>
              <Users size={16} className='mr-2 text-slate-600 dark:text-slate-300' />
              <Text type='tertiary' className='text-slate-600 dark:text-slate-300'>{t('邀请人数')}</Text>
            </div>
            <div className='text-xl font-semibold text-slate-900 dark:text-slate-100 flex items-center'>
              {userState?.user?.aff_count || 0} {t('人')}
            </div>
          </Card>
        </div>

        <div className='!mb-4'>
          {/* 邀请链接部分 */}
          <Input
            value={affLink}
            readonly
            className='!rounded-lg'
            prefix={t('邀请链接')}
            suffix={
              <Button
                type='primary'
                theme='solid'
                onClick={handleAffLinkClick}
                icon={<Copy size={14} />}
                className='!rounded-lg'
              >
                {t('复制')}
              </Button>
            }
          />
        </div>

        {/* 奖励说明 */}
        <Card
          className='!rounded-xl with-pastel-balls-warm'
          title={
            <Text strong className='text-slate-700'>
              {t('奖励说明')}
            </Text>
          }
        >
          <div className='space-y-3'>
            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm text-slate-600'>
                {t('邀请好友注册，好友充值后您可获得相应奖励')}
              </Text>
            </div>

            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm text-slate-600'>
                {t('通过划转功能将奖励额度转入到您的账户余额中')}
              </Text>
            </div>

            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm text-slate-600'>
                {t('邀请的好友越多，获得的奖励越多')}
              </Text>
            </div>
          </div>
        </Card>
      </div>
    </Card>
  );
};

export default InvitationCard;
