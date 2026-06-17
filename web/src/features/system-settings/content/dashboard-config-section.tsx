/*
Copyright (C) 2023-2026 QuantumNous

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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SettingsPageProvider } from '../components/settings-page-context'
import { DashboardMetricsSection } from '../dashboard/components/dashboard-metrics-section'
import { DashboardPanelsSection } from '../dashboard/components/dashboard-panels-section'
import { DashboardRefreshSection } from '../dashboard/components/dashboard-refresh-section'
import { DashboardLimitsSection } from '../dashboard/components/dashboard-limits-section'

type DashboardTab = 'metrics' | 'panels' | 'refresh' | 'limits'

const TABS: { id: DashboardTab; titleKey: string }[] = [
  { id: 'metrics', titleKey: 'Data Metrics' },
  { id: 'panels', titleKey: 'Dashboard Panels' },
  { id: 'refresh', titleKey: 'Refresh Intervals' },
  { id: 'limits', titleKey: 'Data Limits' },
]

// DashboardConfigSection 在 Console Content 下作为单个 section 渲染。
// 内部子分类（metrics/panels/refresh/limits）通过本地 Tabs 切换，
// 不依赖嵌套路由的 useParams，避免 TanStack Router invariant 失败。
//
// 保留 SettingsPageProvider 以提供 actionsContainer portal 容器：
// DashboardRefreshSection / DashboardLimitsSection 通过 SettingsPageFormActions
// 将保存按钮 portal 到 SectionPageLayout.Actions 区域。
export function DashboardConfigSection() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<DashboardTab>('metrics')
  const [actionsContainer, setActionsContainer] =
    useState<HTMLDivElement | null>(null)

  return (
    <SettingsPageProvider actionsContainer={actionsContainer}>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          <span className='truncate'>{t('Dashboard Configuration')}</span>
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <div
            ref={setActionsContainer}
            className='flex flex-wrap items-center justify-end gap-2'
          />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex w-full flex-col gap-4'>
            <Tabs
              value={activeTab}
              onValueChange={(v) => setActiveTab(v as DashboardTab)}
            >
              <TabsList className='w-fit'>
                {TABS.map((tab) => (
                  <TabsTrigger key={tab.id} value={tab.id}>
                    {t(tab.titleKey)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
            {activeTab === 'metrics' && <DashboardMetricsSection />}
            {activeTab === 'panels' && <DashboardPanelsSection />}
            {activeTab === 'refresh' && <DashboardRefreshSection />}
            {activeTab === 'limits' && <DashboardLimitsSection />}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </SettingsPageProvider>
  )
}
