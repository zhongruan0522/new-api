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

export function DashboardPanelsSection() {
  const { t } = useTranslation()
  const { data: config, isLoading } = useDashboardConfig()
  const updateConfig = useUpdateDashboardConfig()

  if (isLoading) {
    return (
      <SettingsSection title={t('Dashboard Panels')}>
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
    <SettingsSection title={t('Dashboard Panels')}>
      <p className="text-sm text-muted-foreground mb-4">
        {t(
          'Control which panels are displayed in the Overview section of the dashboard'
        )}
      </p>
      <div className="space-y-4">
        <SettingsSwitchField
          checked={config.api_info_enabled}
          onCheckedChange={(checked) =>
            handleToggle('api_info_enabled', checked)
          }
          label={t('API Information Panel')}
          description={t(
            'Display API endpoint information and documentation links'
          )}
        />

        <SettingsSwitchField
          checked={config.uptime_kuma_enabled}
          onCheckedChange={(checked) =>
            handleToggle('uptime_kuma_enabled', checked)
          }
          label={t('Uptime Monitoring Panel')}
          description={t(
            'Display Uptime Kuma service monitoring status. Requires external API calls.'
          )}
        />

        <SettingsSwitchField
          checked={config.announcements_enabled}
          onCheckedChange={(checked) =>
            handleToggle('announcements_enabled', checked)
          }
          label={t('Announcements Panel')}
          description={t('Display system announcements to users')}
        />

        <SettingsSwitchField
          checked={config.faq_enabled}
          onCheckedChange={(checked) => handleToggle('faq_enabled', checked)}
          label={t('FAQ Panel')}
          description={t('Display frequently asked questions')}
        />
      </div>
    </SettingsSection>
  )
}
