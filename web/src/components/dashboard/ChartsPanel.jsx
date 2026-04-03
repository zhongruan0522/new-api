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
import { Card, Tabs, TabPane, Skeleton } from '@douyinfe/semi-ui';
import { PieChart } from 'lucide-react';
import { VChart } from '@visactor/react-vchart';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const RankColumn = ({ title, items, loading, isMobile }) => (
  <div className='flex-1 min-w-0'>
    <div className='text-xs font-semibold text-gray-500 mb-2 text-center truncate'>
      {title}
    </div>
    <ol className='space-y-1'>
      {items.map((item, idx) => {
        const rateStr =
          item.success_rate !== undefined
            ? `${item.success_rate.toFixed(1)}%`
            : '--';
        return (
          <li
            key={idx}
            className='flex items-center justify-between text-sm px-1 py-0.5 rounded hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors'
          >
            <span className='truncate mr-2 flex-shrink min-w-0'>
              <span className='text-gray-400 mr-1'>{idx + 1}.</span>
              <span className='font-medium'>{item.model_name}</span>
            </span>
            <Skeleton
              loading={loading}
              active
              placeholder={
                <Skeleton.Paragraph
                  active
                  rows={1}
                  style={{ width: '80px', height: '16px' }}
                />
              }
            >
              <span className='flex-shrink-0 text-gray-500 whitespace-nowrap'>
                {isMobile ? (
                  item.success_count
                ) : (
                  `${item.success_count}/${item.fail_count} - ${rateStr}`
                )}
              </span>
            </Skeleton>
          </li>
        );
      })}
    </ol>
    {items.length === 0 && !loading && (
      <div className='text-xs text-gray-400 text-center py-4'>--</div>
    )}
  </div>
);

const ModelRankPanel = ({ modelRank, loading, t }) => {
  const isMobile = useIsMobile();

  const sections = [
    { key: 'all', title: t('全部模型'), items: modelRank?.all || [] },
    { key: 'domestic', title: t('国内模型'), items: modelRank?.domestic || [] },
    { key: 'overseas', title: t('海外模型'), items: modelRank?.overseas || [] },
  ];

  return (
    <div className='flex gap-4 h-full overflow-auto px-2 py-1'>
      {sections.map((sec) => (
        <RankColumn
          key={sec.key}
          title={sec.title}
          items={sec.items}
          loading={loading}
          isMobile={isMobile}
        />
      ))}
    </div>
  );
};

const ChartsPanel = ({
  activeChartTab,
  setActiveChartTab,
  spec_line,
  spec_model_line,
  spec_pie,
  spec_rank_bar,
  modelRank,
  loading,
  CARD_PROPS,
  CHART_CONFIG,
  FLEX_CENTER_GAP2,
  hasApiInfoPanel,
  t,
}) => {
  return (
    <Card
      {...CARD_PROPS}
      className={`!rounded-2xl ${hasApiInfoPanel ? 'lg:col-span-3' : ''}`}
      title={
        <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between w-full gap-3'>
          <div className={FLEX_CENTER_GAP2}>
            <PieChart size={16} />
            {t('模型数据分析')}
          </div>
          <Tabs
            type='slash'
            activeKey={activeChartTab}
            onChange={setActiveChartTab}
          >
            <TabPane tab={<span>{t('消耗分布')}</span>} itemKey='1' />
            <TabPane tab={<span>{t('消耗趋势')}</span>} itemKey='2' />
            <TabPane tab={<span>{t('调用次数分布')}</span>} itemKey='3' />
            <TabPane tab={<span>{t('调用次数排行')}</span>} itemKey='4' />
          </Tabs>
        </div>
      }
      bodyStyle={{ padding: 0 }}
    >
      <div className='h-96 p-2'>
        {activeChartTab === '1' && (
          <VChart spec={spec_line} option={CHART_CONFIG} />
        )}
        {activeChartTab === '2' && (
          <VChart spec={spec_model_line} option={CHART_CONFIG} />
        )}
        {activeChartTab === '3' && (
          <VChart spec={spec_pie} option={CHART_CONFIG} />
        )}
        {activeChartTab === '4' && (
          <ModelRankPanel modelRank={modelRank} loading={loading} t={t} />
        )}
      </div>
    </Card>
  );
};

export default ChartsPanel;
