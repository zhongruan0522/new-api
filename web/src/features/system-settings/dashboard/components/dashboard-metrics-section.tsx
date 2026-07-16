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
      <SettingsSection title={t('systemSettings.fields.dataMetrics')}>
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
    <SettingsSection title={t('systemSettings.fields.dataMetrics')}>
      <p className="text-sm text-muted-foreground mb-4">
        {t(
          'systemSettings.tips.controlWhichDataMetricsAreCalculatedAndDisplayedInThe'
        )}
      </p>
      <div className="space-y-4">
        <SettingsSwitchField
          checked={config.quota_data_enabled}
          onCheckedChange={(checked) =>
            handleToggle('quota_data_enabled', checked)
          }
          label={t('systemSettings.fields.quotaData')}
          description={t(
            'systemSettings.actions.enableTimeSeriesQuotaDataDisplayDisablingWillHideModel'
          )}
        />

        <SettingsSwitchField
          checked={config.user_analytics_enabled}
          onCheckedChange={(checked) =>
            handleToggle('user_analytics_enabled', checked)
          }
          label={t('systemSettings.titles.userAnalytics')}
          description={t(
            'systemSettings.actions.enableUserConsumptionRankingsAndTrendsDisablingWillReduceDatabase'
          )}
        />

        <SettingsSwitchField
          checked={config.rankings_enabled}
          onCheckedChange={(checked) =>
            handleToggle('rankings_enabled', checked)
          }
          label={t('rankings.titles.value')}
          description={t(
            'systemSettings.actions.enableModelAndVendorRankingsThisIsAHighCost'
          )}
        />

        <SettingsSwitchField
          checked={config.media_convert_stats_enabled}
          onCheckedChange={(checked) =>
            handleToggle('media_convert_stats_enabled', checked)
          }
          label={t('systemSettings.fields.mediaConversionStats')}
          description={t(
            'systemSettings.actions.enableImageVideoToUrlConversionStatisticsDisplay'
          )}
        />

        <SettingsSwitchField
          checked={config.quota_data_track_tokens}
          onCheckedChange={(checked) =>
            handleToggle('quota_data_track_tokens', checked)
          }
          label={t('systemSettings.fields.trackTokenUsage')}
          description={t(
            'systemSettings.tips.recordTokenUsedInQuotaDataDisablingReducesWriteCost'
          )}
        />

        <SettingsSwitchField
          checked={config.quota_data_track_by_model}
          onCheckedChange={(checked) =>
            handleToggle('quota_data_track_by_model', checked)
          }
          label={t('systemSettings.tips.aggregateByModel')}
          description={t(
            'systemSettings.tips.aggregateQuotaDataPerModelDisablingCollapsesAllModelsInto'
          )}
        />

        <SettingsSwitchField
          checked={config.quota_data_track_by_user}
          onCheckedChange={(checked) =>
            handleToggle('quota_data_track_by_user', checked)
          }
          label={t('systemSettings.tips.aggregateByUser')}
          description={t(
            'systemSettings.tips.aggregateQuotaDataPerUserDisablingCollapsesAllUsersInto'
          )}
        />
      </div>
    </SettingsSection>
  )
}
