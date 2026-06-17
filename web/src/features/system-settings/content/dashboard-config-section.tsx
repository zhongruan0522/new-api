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
import { SettingsPage } from '../components/settings-page'
import {
  getDashboardSectionContent,
  getDashboardSectionMeta,
} from '../dashboard/section-registry'

const DEFAULT_DASHBOARD_CONFIG = {
  quota_data_enabled: true,
  user_analytics_enabled: true,
  rankings_enabled: true,
  media_convert_stats_enabled: true,
  quota_data_track_tokens: true,
  quota_data_track_by_model: true,
  quota_data_track_by_user: true,
  api_info_enabled: true,
  uptime_kuma_enabled: true,
  announcements_enabled: true,
  faq_enabled: true,
  quota_data_refresh_interval: 3600,
  user_analytics_refresh_interval: 3600,
  rankings_refresh_interval: 300,
  uptime_kuma_refresh_interval: 60,
  default_time_range_days: 7,
  max_time_range_days: 31,
  rankings_model_limit: 20,
  rankings_vendor_limit: 5,
  user_analytics_top_n: 20,
}

export function DashboardConfigSection() {
  return (
    <SettingsPage
      routePath="/system-settings/content/dashboard-config"
      defaultSettings={DEFAULT_DASHBOARD_CONFIG}
      defaultSection="metrics"
      getSectionContent={getDashboardSectionContent}
      getSectionMeta={getDashboardSectionMeta}
    />
  )
}
