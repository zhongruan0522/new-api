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
import { useTranslation } from 'react-i18next'
import { SettingsSection } from '../../components/settings-section'
import { SettingsSwitchField } from '../../components/settings-form-layout'
import {
  useDashboardConfig,
  useUpdateDashboardConfig,
} from '../hooks/use-dashboard-config'
import { Skeleton } from '@/components/ui/skeleton'

export function DashboardMetricsSection() {
  const { t } = useTranslation()
  const { data: config, isLoading } = useDashboardConfig()
  const updateConfig = useUpdateDashboardConfig()

  if (isLoading) {
    return (
      <SettingsSection title={t('Data Metrics')}>
        <div className="space-y-4">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      </SettingsSection>
    )
  }

  if (!config) return null

  const handleToggle = (field: string, value: boolean) => {
    updateConfig.mutate({ [field]: value })
  }

  return (
    <SettingsSection title={t('Data Metrics')}>
      <p className="text-sm text-muted-foreground mb-4">
        {t(
          'Control which data metrics are calculated and displayed in the dashboard'
        )}
      </p>
      <div className="space-y-4">
        <SettingsSwitchField
          checked={config.quota_data_enabled}
          onCheckedChange={(checked) =>
            handleToggle('quota_data_enabled', checked)
          }
          label={t('Quota Data')}
          description={t(
            'Enable time-series quota data display. Disabling will hide model analytics charts.'
          )}
        />

        <SettingsSwitchField
          checked={config.user_analytics_enabled}
          onCheckedChange={(checked) =>
            handleToggle('user_analytics_enabled', checked)
          }
          label={t('User Analytics')}
          description={t(
            'Enable user consumption rankings and trends. Disabling will reduce database query load.'
          )}
        />

        <SettingsSwitchField
          checked={config.rankings_enabled}
          onCheckedChange={(checked) =>
            handleToggle('rankings_enabled', checked)
          }
          label={t('Rankings')}
          description={t(
            'Enable model and vendor rankings. This is a high-cost query that runs every 5 minutes.'
          )}
        />

        <SettingsSwitchField
          checked={config.media_convert_stats_enabled}
          onCheckedChange={(checked) =>
            handleToggle('media_convert_stats_enabled', checked)
          }
          label={t('Media Conversion Stats')}
          description={t(
            'Enable image/video to URL conversion statistics display.'
          )}
        />
      </div>
    </SettingsSection>
  )
}
