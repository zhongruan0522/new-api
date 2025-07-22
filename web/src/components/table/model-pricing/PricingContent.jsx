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
import PricingSearchBar from './PricingSearchBar.jsx';
import PricingTable from './PricingTable.jsx';

const PricingContent = (props) => {
  return (
    <div className="pricing-scroll-hide">
      {/* 固定的搜索和操作区域 */}
      <div
        style={{
          padding: '16px 24px',
          borderBottom: '1px solid var(--semi-color-border)',
          backgroundColor: 'var(--semi-color-bg-0)',
          flexShrink: 0
        }}
      >
        <PricingSearchBar {...props} />
      </div>

      {/* 可滚动的内容区域 */}
      <div
        style={{
          flex: 1,
          overflow: 'auto'
        }}
      >
        <PricingTable {...props} />
      </div>
    </div>
  );
};

export default PricingContent; 