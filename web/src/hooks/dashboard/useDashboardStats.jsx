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

import { useMemo } from 'react';
import { Wallet, Activity, Zap, Gauge, ImageOff, Globe } from 'lucide-react';
import {
  IconMoneyExchangeStroked,
  IconHistogram,
  IconCoinMoneyStroked,
  IconTextStroked,
  IconPulse,
  IconStopwatchStroked,
  IconTypograph,
  IconSend,
} from '@douyinfe/semi-icons';
import { renderQuota } from '../../helpers';
import { createSectionTitle } from '../../helpers/dashboard';

const getRateColor = (stat) => {
  if (!stat || stat.success_rate < 0) return '#6b7280';
  const rate = stat.success_rate;
  if (rate >= 95) return '#10b981';
  if (rate >= 50) return '#f59e0b';
  return '#ef4444';
};

const formatRegionRate = (stat) => {
  if (!stat || stat.success_rate < 0) return '--';
  return `${stat.success_rate.toFixed(1)}%`;
};

const formatRegionCount = (stat) => {
  if (!stat) return '';
  return `${stat.success_count} / ${stat.fail_count}`;
};

const formatCacheRate = (stat) => {
  if (!stat || stat.cache_rate < 0) return '--';
  return `${stat.cache_rate.toFixed(2)}%`;
};

const formatCacheDetail = (stat) => {
  if (!stat) return '';
  return `命中 ${stat.cache_hit_tokens || 0} / 创建 ${stat.cache_creation_tokens || 0} / 输入 ${stat.input_tokens || 0}`;
};

export const useDashboardStats = (
  userState,
  consumeQuota,
  consumeTokens,
  times,
  failCount,
  trendData,
  performanceMetrics,
  navigate,
  t,
  regionStats,
  mediaConvertStats,
) => {
  const totalRate = useMemo(() => {
    if (!times && !failCount) return '--';
    const total = times + failCount;
    if (total === 0) return '0.0%';
    return ((times / total) * 100).toFixed(1) + '%';
  }, [times, failCount]);

  const groupedStatsData = useMemo(
    () => [
      {
        title: createSectionTitle(Wallet, t('账户数据')),
        color: 'bg-blue-50',
        items: [
          {
            title: t('当前余额'),
            value: renderQuota(userState?.user?.quota),
            icon: <IconMoneyExchangeStroked />,
            avatarColor: 'blue',
            trendData: [],
            trendColor: '#3b82f6',
          },
          {
            title: t('历史消耗'),
            value: renderQuota(userState?.user?.used_quota),
            icon: <IconHistogram />,
            avatarColor: 'purple',
            trendData: [],
            trendColor: '#8b5cf6',
          },
        ],
      },
      {
        title: createSectionTitle(Activity, t('使用统计')),
        color: 'bg-green-50',
        items: [
          {
            title: t('请求次数'),
            value: userState.user?.request_count,
            icon: <IconSend />,
            avatarColor: 'green',
            trendData: [],
            trendColor: '#10b981',
          },
          {
            title: t('统计次数'),
            value: `${times} / ${failCount} - ${totalRate}`,
            icon: <IconPulse />,
            avatarColor: 'cyan',
            trendData: trendData.times,
            trendColor: '#06b6d4',
          },
        ],
      },
      {
        title: createSectionTitle(Globe, t('海内外模型成功率')),
        color: 'bg-emerald-50',
        colSpan: 2,
        regionItems: [
          {
            label: t('国内模型'),
            rateValue: formatRegionRate(regionStats?.domestic),
            countValue: formatRegionCount(regionStats?.domestic),
            color: getRateColor(regionStats?.domestic),
            rate: regionStats?.domestic?.success_rate != null && regionStats.domestic.success_rate >= 0
              ? regionStats.domestic.success_rate
              : null,
          },
          {
            label: t('海外模型'),
            rateValue: formatRegionRate(regionStats?.overseas),
            countValue: formatRegionCount(regionStats?.overseas),
            color: getRateColor(regionStats?.overseas),
            rate: regionStats?.overseas?.success_rate != null && regionStats.overseas.success_rate >= 0
              ? regionStats.overseas.success_rate
              : null,
          },
        ],
      },
      {
        title: createSectionTitle(Globe, t('海内外模型缓存率')),
        color: 'bg-sky-50',
        colSpan: 2,
        regionItems: [
          {
            label: t('国内模型'),
            rateValue: formatCacheRate(regionStats?.domestic),
            countValue: formatCacheDetail(regionStats?.domestic),
            color: '#0ea5e9',
            rate: regionStats?.domestic?.cache_rate != null && regionStats.domestic.cache_rate >= 0
              ? regionStats.domestic.cache_rate
              : null,
          },
          {
            label: t('海外模型'),
            rateValue: formatCacheRate(regionStats?.overseas),
            countValue: formatCacheDetail(regionStats?.overseas),
            color: '#6366f1',
            rate: regionStats?.overseas?.cache_rate != null && regionStats.overseas.cache_rate >= 0
              ? regionStats.overseas.cache_rate
              : null,
          },
        ],
      },
      {
        title: createSectionTitle(Zap, t('资源消耗')),
        color: 'bg-yellow-50',
        items: [
          {
            title: t('统计额度'),
            value: renderQuota(consumeQuota),
            icon: <IconCoinMoneyStroked />,
            avatarColor: 'yellow',
            trendData: trendData.consumeQuota,
            trendColor: '#f59e0b',
          },
          {
            title: t('统计Tokens'),
            value: isNaN(consumeTokens) ? 0 : consumeTokens.toLocaleString(),
            icon: <IconTextStroked />,
            avatarColor: 'pink',
            trendData: trendData.tokens,
            trendColor: '#ec4899',
          },
        ],
      },
      {
        title: createSectionTitle(Gauge, t('性能指标')),
        color: 'bg-indigo-50',
        items: [
          {
            title: t('平均RPM'),
            value: performanceMetrics.avgRPM,
            icon: <IconStopwatchStroked />,
            avatarColor: 'indigo',
            trendData: trendData.rpm,
            trendColor: '#6366f1',
          },
          {
            title: t('平均TPM'),
            value: performanceMetrics.avgTPM,
            icon: <IconTypograph />,
            avatarColor: 'orange',
            trendData: trendData.tpm,
            trendColor: '#f97316',
          },
        ],
      },
      {
        title: createSectionTitle(ImageOff, t('适配非多模态')),
        color: 'bg-rose-50',
        items: [
          {
            title: t('图片转URL'),
            value: mediaConvertStats?.image_count || 0,
            icon: <IconHistogram />,
            avatarColor: 'rose',
            trendData: [],
            trendColor: '#f43f5e',
          },
          {
            title: t('视频转URL'),
            value: mediaConvertStats?.video_count || 0,
            icon: <IconPulse />,
            avatarColor: 'amber',
            trendData: [],
            trendColor: '#f59e0b',
          },
        ],
      },
    ],
    [
      userState?.user?.quota,
      userState?.user?.used_quota,
      userState?.user?.request_count,
      mediaConvertStats,
      times,
      failCount,
      totalRate,
      consumeQuota,
      consumeTokens,
      trendData,
      performanceMetrics,
      navigate,
      t,
      regionStats,
    ],
  );

  return {
    groupedStatsData,
  };
};
