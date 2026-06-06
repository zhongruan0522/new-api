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
import { useStatus } from '@/hooks/use-status'
import { SettingsSection } from '../components/settings-section'

export function SystemMaintenanceSection() {
  const { t } = useTranslation()
  const { status, loading } = useStatus()
  const buildHash = status?.version || ''

  return (
    <SettingsSection title={t('System maintenance')}>
      <div className='rounded-lg border p-4'>
        <div className='text-muted-foreground text-sm'>
          {t('Build ID hash')}
        </div>
        <div className='mt-1 font-mono text-lg font-semibold break-all'>
          {loading && !buildHash
            ? t('Loading...')
            : buildHash || t('Unknown version')}
        </div>
      </div>
    </SettingsSection>
  )
}
