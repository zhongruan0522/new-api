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
import PricingDisplaySettings from '../../filter/PricingDisplaySettings';
import PricingCategories from '../../filter/PricingCategories';
import PricingGroups from '../../filter/PricingGroups';
import PricingQuotaTypes from '../../filter/PricingQuotaTypes';
import PricingEndpointTypes from '../../filter/PricingEndpointTypes';

const FilterModalContent = ({ sidebarProps, t }) => {
  const {
    showWithRecharge,
    setShowWithRecharge,
    currency,
    setCurrency,
    handleChange,
    setActiveKey,
    showRatio,
    setShowRatio,
    viewMode,
    setViewMode,
    filterGroup,
    setFilterGroup,
    filterQuotaType,
    setFilterQuotaType,
    filterEndpointType,
    setFilterEndpointType,
    tokenUnit,
    setTokenUnit,
    loading,
    ...categoryProps
  } = sidebarProps;

  return (
    <div className="p-2">
      <PricingDisplaySettings
        showWithRecharge={showWithRecharge}
        setShowWithRecharge={setShowWithRecharge}
        currency={currency}
        setCurrency={setCurrency}
        showRatio={showRatio}
        setShowRatio={setShowRatio}
        viewMode={viewMode}
        setViewMode={setViewMode}
        tokenUnit={tokenUnit}
        setTokenUnit={setTokenUnit}
        loading={loading}
        t={t}
      />

      <PricingCategories {...categoryProps} setActiveKey={setActiveKey} loading={loading} t={t} />

      <PricingGroups
        filterGroup={filterGroup}
        setFilterGroup={setFilterGroup}
        usableGroup={categoryProps.usableGroup}
        groupRatio={categoryProps.groupRatio}
        models={categoryProps.models}
        loading={loading}
        t={t}
      />

      <PricingQuotaTypes
        filterQuotaType={filterQuotaType}
        setFilterQuotaType={setFilterQuotaType}
        models={categoryProps.models}
        loading={loading}
        t={t}
      />

      <PricingEndpointTypes
        filterEndpointType={filterEndpointType}
        setFilterEndpointType={setFilterEndpointType}
        models={categoryProps.models}
        loading={loading}
        t={t}
      />
    </div>
  );
};

export default FilterModalContent; 