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
import { Layout, ImagePreview } from '@douyinfe/semi-ui';
import PricingSidebar from './PricingSidebar.jsx';
import PricingContent from './PricingContent.jsx';
import { useModelPricingData } from '../../../hooks/model-pricing/useModelPricingData.js';

const PricingPage = () => {
  const pricingData = useModelPricingData();
  const { Sider, Content } = Layout;

  // 显示倍率状态
  const [showRatio, setShowRatio] = React.useState(false);

  return (
    <div className="bg-white">
      <Layout style={{ height: 'calc(100vh - 60px)', overflow: 'hidden', marginTop: '60px' }}>
        {/* 左侧边栏 */}
        <Sider
          style={{
            width: 460,
            height: 'calc(100vh - 60px)',
            backgroundColor: 'var(--semi-color-bg-0)',
            borderRight: '1px solid var(--semi-color-border)',
            overflow: 'auto'
          }}
        >
          <PricingSidebar {...pricingData} showRatio={showRatio} setShowRatio={setShowRatio} />
        </Sider>

        {/* 右侧内容区 */}
        <Content
          style={{
            height: 'calc(100vh - 60px)',
            backgroundColor: 'var(--semi-color-bg-0)',
            display: 'flex',
            flexDirection: 'column'
          }}
        >
          <PricingContent {...pricingData} showRatio={showRatio} />
        </Content>
      </Layout>

      {/* 倍率说明图预览 */}
      <ImagePreview
        src={pricingData.modalImageUrl}
        visible={pricingData.isModalOpenurl}
        onVisibleChange={(visible) => pricingData.setIsModalOpenurl(visible)}
      />
    </div>
  );
};

export default PricingPage; 