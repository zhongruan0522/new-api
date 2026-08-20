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
import { createSectionRegistry } from '../utils/section-registry'
import { DashboardLimitsSection } from './components/dashboard-limits-section'
import { DashboardMetricsSection } from './components/dashboard-metrics-section'
import { DashboardPanelsSection } from './components/dashboard-panels-section'
import { DashboardRefreshSection } from './components/dashboard-refresh-section'
import type { DashboardConfig } from './types'

export type DashboardSectionId = 'metrics' | 'panels' | 'refresh' | 'limits'

const DASHBOARD_SECTIONS = [
  {
    id: 'metrics' as const,
    titleKey: 'systemSettings.fields.dataMetrics',
    build: () => <DashboardMetricsSection />,
  },
  {
    id: 'panels' as const,
    titleKey: 'systemSettings.titles.dashboardPanels',
    build: () => <DashboardPanelsSection />,
  },
  {
    id: 'refresh' as const,
    titleKey: 'systemSettings.actions.refreshIntervals',
    build: () => <DashboardRefreshSection />,
  },
  {
    id: 'limits' as const,
    titleKey: 'systemSettings.fields.dataLimits',
    build: () => <DashboardLimitsSection />,
  },
]

const dashboardRegistry = createSectionRegistry<
  DashboardSectionId,
  DashboardConfig
>({
  sections: DASHBOARD_SECTIONS,
  defaultSection: 'metrics',
  basePath: '/system-settings/content/dashboard-config',
  urlStyle: 'path',
})

export const getDashboardSectionContent = dashboardRegistry.getSectionContent
export const getDashboardSectionMeta = dashboardRegistry.getSectionMeta
export const getDashboardSectionNavItems = dashboardRegistry.getSectionNavItems
