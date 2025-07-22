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
import { Modal, Button } from '@douyinfe/semi-ui';
import PricingCategories from '../filter/PricingCategories';
import PricingGroups from '../filter/PricingGroups';
import PricingQuotaTypes from '../filter/PricingQuotaTypes';
import PricingDisplaySettings from '../filter/PricingDisplaySettings';
import { resetPricingFilters } from '../../../../helpers/utils';

const PricingFilterModal = ({
  visible,
  onClose,
  sidebarProps,
  t
}) => {
  const {
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
    ...categoryProps
  } = sidebarProps;

  const handleResetFilters = () =>
    resetPricingFilters({
      handleChange,
      setActiveKey,
      availableCategories: categoryProps.availableCategories,
      setShowWithRecharge,
      setCurrency,
      setShowRatio,
      setFilterGroup,
      setFilterQuotaType,
    });

  const handleConfirm = () => {
    onClose();
  };

  const footer = (
    <div className="flex justify-end">
      <Button
        theme="outline"
        type='tertiary'
        onClick={handleResetFilters}
      >
        {t('重置')}
      </Button>
      <Button
        theme="solid"
        type="primary"
        onClick={handleConfirm}
      >
        {t('确定')}
      </Button>
    </div>
  );

  return (
    <Modal
      title={t('筛选')}
      visible={visible}
      onCancel={onClose}
      footer={footer}
      style={{ width: '100%', height: '100%', margin: 0 }}
      bodyStyle={{
        padding: 0,
        height: 'calc(100vh - 160px)',
        overflowY: 'auto',
        scrollbarWidth: 'none',
        msOverflowStyle: 'none'
      }}
    >
      <div className="p-2">
        <PricingDisplaySettings
          showWithRecharge={showWithRecharge}
          setShowWithRecharge={setShowWithRecharge}
          currency={currency}
          setCurrency={setCurrency}
          showRatio={showRatio}
          setShowRatio={setShowRatio}
          t={t}
        />

        <PricingCategories {...categoryProps} setActiveKey={setActiveKey} t={t} />

        <PricingGroups
          filterGroup={filterGroup}
          setFilterGroup={setFilterGroup}
          usableGroup={categoryProps.usableGroup}
          models={categoryProps.models}
          t={t}
        />

        <PricingQuotaTypes
          filterQuotaType={filterQuotaType}
          setFilterQuotaType={setFilterQuotaType}
          models={categoryProps.models}
          t={t}
        />
      </div>
    </Modal>
  );
};

export default PricingFilterModal; 