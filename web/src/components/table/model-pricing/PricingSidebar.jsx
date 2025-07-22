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
import { Divider, Button, Switch, Select, Tooltip } from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import PricingCategories from './sidebar/PricingCategories.jsx';
import PricingGroups from './sidebar/PricingGroups.jsx';
import PricingQuotaTypes from './sidebar/PricingQuotaTypes.jsx';

const PricingSidebar = ({
  showWithRecharge,
  setShowWithRecharge,
  currency,
  setCurrency,
  handleChange,
  setActiveKey,
  showRatio,
  setShowRatio,
  filterGroup,
  setFilterGroup,
  filterQuotaType,
  setFilterQuotaType,
  t,
  ...categoryProps
}) => {

  // 重置所有筛选条件
  const handleResetFilters = () => {
    // 重置搜索
    if (handleChange) {
      handleChange('');
    }

    // 重置模型分类到默认
    if (setActiveKey && categoryProps.availableCategories?.length > 0) {
      setActiveKey(categoryProps.availableCategories[0]);
    }

    // 重置充值价格显示
    if (setShowWithRecharge) {
      setShowWithRecharge(false);
    }

    // 重置货币
    if (setCurrency) {
      setCurrency('USD');
    }

    // 重置显示倍率
    setShowRatio(false);

    // 重置分组筛选
    if (setFilterGroup) {
      setFilterGroup('all');
    }

    // 重置计费类型筛选
    if (setFilterQuotaType) {
      setFilterQuotaType('all');
    }
  };

  return (
    <div className="p-4">
      {/* 筛选标题和重置按钮 */}
      <div className="flex items-center justify-between mb-6">
        <div className="text-lg font-semibold text-gray-800">
          {t('筛选')}
        </div>
        <Button
          theme="outline"
          onClick={handleResetFilters}
          className="text-gray-500 hover:text-gray-700"
        >
          {t('重置')}
        </Button>
      </div>

      {/* 显示设置 */}
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

      {/* 模型分类 */}
      <PricingCategories {...categoryProps} setActiveKey={setActiveKey} t={t} />

      <PricingGroups filterGroup={filterGroup} setFilterGroup={setFilterGroup} usableGroup={categoryProps.usableGroup} models={categoryProps.models} t={t} />

      <PricingQuotaTypes filterQuotaType={filterQuotaType} setFilterQuotaType={setFilterQuotaType} models={categoryProps.models} t={t} />
    </div>
  );
};

export default PricingSidebar; 