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
import { Skeleton } from '@/components/ui/skeleton'
import { SettingsSwitchField } from '../../components/settings-form-layout'
import { SettingsSection } from '../../components/settings-section'
import {
  useDashboardConfig,
  useUpdateDashboardConfig,
} from '../hooks/use-dashboard-config'

export function DashboardPanelsSection() {
  const { t } = useTranslation()
  const { data: config, isLoading } = useDashboardConfig()
  const updateConfig = useUpdateDashboardConfig()

  if (isLoading) {
    return (
      <SettingsSection title={t('systemSettings.titles.dashboardPanels')}>
        <div className='space-y-4'>
          <Skeleton className='h-16 w-full' />
          <Skeleton className='h-16 w-full' />
          <Skeleton className='h-16 w-full' />
          <Skeleton className='h-16 w-full' />
        </div>
      </SettingsSection>
    )
  }

  if (!config) return null

  const handleToggle = (field: string, value: boolean) => {
    updateConfig.mutate({ [field]: value })
  }

  return (
    <SettingsSection title={t('systemSettings.titles.dashboardPanels')}>
      <p className='text-muted-foreground mb-4 text-sm'>
        {t(
          'systemSettings.tips.controlWhichPanelsAreDisplayedInTheOverviewSectionOf'
        )}
      </p>
      <div className='space-y-4'>
        <SettingsSwitchField
          checked={config.api_info_enabled}
          onCheckedChange={(checked) =>
            handleToggle('api_info_enabled', checked)
          }
          label={t('systemSettings.fields.apiInformationPanel')}
          description={t(
            'systemSettings.tips.displayApiEndpointInformationAndDocumentationLinks'
          )}
        />

        <SettingsSwitchField
          checked={config.uptime_kuma_enabled}
          onCheckedChange={(checked) =>
            handleToggle('uptime_kuma_enabled', checked)
          }
          label={t('systemSettings.tips.uptimeMonitoringPanel')}
          description={t(
            'systemSettings.tips.displayUptimeKumaServiceMonitoringStatusRequiresExternalApiCalls'
          )}
        />

        <SettingsSwitchField
          checked={config.announcements_enabled}
          onCheckedChange={(checked) =>
            handleToggle('announcements_enabled', checked)
          }
          label={t('systemSettings.fields.announcementsPanel')}
          description={t(
            'systemSettings.tips.displaySystemAnnouncementsToUsers'
          )}
        />

        <SettingsSwitchField
          checked={config.faq_enabled}
          onCheckedChange={(checked) => handleToggle('faq_enabled', checked)}
          label={t('systemSettings.fields.faqPanel')}
          description={t('systemSettings.tips.displayFrequentlyAskedQuestions')}
        />
      </div>
    </SettingsSection>
  )
}
