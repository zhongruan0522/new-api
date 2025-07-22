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
import { Modal } from '@douyinfe/semi-ui';
import PricingSidebar from '../PricingSidebar';

const PricingFilterModal = ({
  visible,
  onClose,
  sidebarProps,
  t
}) => {
  return (
    <Modal
      title={t('筛选')}
      visible={visible}
      onCancel={onClose}
      footer={null}
      style={{ width: '100%', height: '100%', margin: 0 }}
      bodyStyle={{ 
        padding: 0, 
        height: 'calc(100vh - 110px)', 
        overflow: 'auto' 
      }}
    >
      <PricingSidebar {...sidebarProps} />
    </Modal>
  );
};

export default PricingFilterModal; 