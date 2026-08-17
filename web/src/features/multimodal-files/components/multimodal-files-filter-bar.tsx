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
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export interface MultimodalFilesFilterBarProps {
  startTime: string
  endTime: string
  onStartTimeChange: (value: string) => void
  onEndTimeChange: (value: string) => void
  onQuery: () => void
  onReset: () => void
  onRefresh: () => void
}

export function MultimodalFilesFilterBar({
  startTime,
  endTime,
  onStartTimeChange,
  onEndTimeChange,
  onQuery,
  onReset,
  onRefresh,
}: MultimodalFilesFilterBarProps) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-col gap-2 sm:flex-row sm:items-end sm:gap-3'>
      <div className='grid gap-1.5'>
        <Label htmlFor='stored-media-start' className='text-xs'>
          {t('dashboard.actions.startTime')}
        </Label>
        <Input
          id='stored-media-start'
          type='datetime-local'
          value={startTime}
          onChange={(event) => onStartTimeChange(event.target.value)}
        />
      </div>
      <div className='grid gap-1.5'>
        <Label htmlFor='stored-media-end' className='text-xs'>
          {t('dashboard.fields.endTime')}
        </Label>
        <Input
          id='stored-media-end'
          type='datetime-local'
          value={endTime}
          onChange={(event) => onEndTimeChange(event.target.value)}
        />
      </div>
      <div className='flex items-end gap-2'>
        <Button onClick={onQuery} className='flex-1'>
          {t('keyQuery.titles.query')}
        </Button>
        <Button variant='outline' onClick={onReset}>
          {t('common.actions.reset')}
        </Button>
        <Button variant='outline' size='icon' onClick={onRefresh}>
          <RefreshCw />
          <span className='sr-only'>{t('channels.actions.refresh')}</span>
        </Button>
      </div>
    </div>
  )
}
