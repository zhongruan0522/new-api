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
import { useSystemInfo } from '@/hooks/use-system-info'
import { SettingsSection } from '../components/settings-section'

export function SystemMaintenanceSection() {
  const { t } = useTranslation()
  const { version: buildHash, loading } = useSystemInfo()

  return (
    <SettingsSection title={t('systemSettings.titles.maintenance')}>
      <div className='rounded-lg border p-4'>
        <div className='text-muted-foreground text-sm'>
          {t('systemSettings.fields.buildIdHash')}
        </div>
        <div className='mt-1 font-mono text-lg font-semibold break-all'>
          {loading && !buildHash
            ? t('common.tips.loading')
            : buildHash || t('layout.fields.unknownVersion')}
        </div>
      </div>
    </SettingsSection>
  )
}
