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
import { Card, Typography, Divider } from '@douyinfe/semi-ui';
import { Wallet, Coins, BarChart2 } from 'lucide-react';

const { Text } = Typography;

const RightStatsCard = ({ t, userState, renderQuota }) => {
  return (
    <Card size="small" className="!rounded-xl shadow-sm" bodyStyle={{ padding: '8px 12px' }}>
      <div className="flex items-center gap-3 lg:gap-4 w-full">
        <div className="flex items-center justify-end gap-2 flex-1">
          <Wallet size={16} className="text-slate-600 dark:text-slate-300" />
          <div className="text-right">
            <Text size="small" type="tertiary">{t('当前余额')}</Text>
            <div className="text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-100">{renderQuota(userState?.user?.quota)}</div>
          </div>
        </div>
        <Divider layout="vertical" className="hidden md:block" />
        <div className="flex items-center justify-end gap-2 flex-1">
          <Coins size={16} className="text-slate-600 dark:text-slate-300" />
          <div className="text-right">
            <Text size="small" type="tertiary">{t('历史消耗')}</Text>
            <div className="text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-100">{renderQuota(userState?.user?.used_quota)}</div>
          </div>
        </div>
        <Divider layout="vertical" className="hidden md:block" />
        <div className="flex items-center justify-end gap-2 flex-1">
          <BarChart2 size={16} className="text-slate-600 dark:text-slate-300" />
          <div className="text-right">
            <Text size="small" type="tertiary">{t('请求次数')}</Text>
            <div className="text-xs sm:text-sm font-semibold text-gray-800 dark:text-gray-100">{userState?.user?.request_count || 0}</div>
          </div>
        </div>
      </div>
    </Card>
  );
};

export default RightStatsCard;


