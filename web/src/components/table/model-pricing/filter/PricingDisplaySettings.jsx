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
import { Divider, Switch, Select, Tooltip } from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';

const PricingDisplaySettings = ({
  showWithRecharge,
  setShowWithRecharge,
  currency,
  setCurrency,
  showRatio,
  setShowRatio,
  t
}) => {
  return (
    <div className="mb-6">
      <Divider margin='12px' align='left'>
        {t('显示设置')}
      </Divider>
      <div className="px-2">
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm text-gray-700">{t('以充值价格显示')}</span>
          <Switch
            checked={showWithRecharge}
            onChange={setShowWithRecharge}
            size="small"
          />
        </div>
        {showWithRecharge && (
          <div className="mt-2 mb-3">
            <div className="text-xs text-gray-500 mb-1">{t('货币单位')}</div>
            <Select
              value={currency}
              onChange={setCurrency}
              size="small"
              className="w-full"
            >
              <Select.Option value="USD">USD ($)</Select.Option>
              <Select.Option value="CNY">CNY (¥)</Select.Option>
            </Select>
          </div>
        )}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <span className="text-sm text-gray-700">{t('显示倍率')}</span>
            <Tooltip content={t('倍率是用于系统计算不同模型的最终价格用的，如果您不理解倍率，请忽略')}>
              <IconHelpCircle
                size="small"
                style={{ color: 'var(--semi-color-text-2)', cursor: 'help' }}
              />
            </Tooltip>
          </div>
          <Switch
            checked={showRatio}
            onChange={setShowRatio}
            size="small"
          />
        </div>
      </div>
    </div>
  );
};

export default PricingDisplaySettings; 