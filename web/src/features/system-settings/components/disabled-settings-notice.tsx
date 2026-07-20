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
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

type DisabledSettingsNoticeProps = {
  enabled: boolean
  title?: string
  description?: string
}

export function DisabledSettingsNotice({
  enabled,
  title = 'common.status.settingIsCurrentlyDisabled',
  description = 'common.tips.ifYouAreReportingARelatedIssueEnableThis',
}: DisabledSettingsNoticeProps) {
  const { t } = useTranslation()

  if (enabled) {
    return null
  }

  return (
    <Alert className='border-destructive/30 bg-destructive/10 text-destructive'>
      <AlertTitle>{t(title)}</AlertTitle>
      <AlertDescription className='text-destructive/90'>
        {t(description)}
      </AlertDescription>
    </Alert>
  )
}
