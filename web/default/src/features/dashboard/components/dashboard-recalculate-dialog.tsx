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
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Calendar, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { getRollingDateRange } from '@/lib/time'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { DateTimePicker } from '@/components/datetime-picker'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { TIME_RANGE_PRESETS } from '@/features/dashboard/constants'
import {
  buildDefaultDashboardFilters,
  getSavedChartPreferences,
} from '@/features/dashboard/lib'
import { recalculateQuotaData } from '@/features/dashboard/api'
import type { DashboardFilters } from '@/features/dashboard/types'

type DashboardRecalculateDialogProps = {
  initialTimeRange?: Pick<DashboardFilters, 'start_timestamp' | 'end_timestamp'>
  triggerClassName?: string
}

function buildInitialTimeRange(
  initialTimeRange?: DashboardRecalculateDialogProps['initialTimeRange']
): DashboardFilters {
  const filters = buildDefaultDashboardFilters(getSavedChartPreferences())
  if (initialTimeRange?.start_timestamp && initialTimeRange?.end_timestamp) {
    return {
      ...filters,
      start_timestamp: initialTimeRange.start_timestamp,
      end_timestamp: initialTimeRange.end_timestamp,
    }
  }
  return filters
}

export function DashboardRecalculateDialog(
  props: DashboardRecalculateDialogProps
) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [recalculating, setRecalculating] = useState(false)
  const [selectedRange, setSelectedRange] = useState<number | null>(() =>
    props.initialTimeRange?.start_timestamp &&
    props.initialTimeRange?.end_timestamp
      ? null
      : getSavedChartPreferences().defaultTimeRangeDays
  )
  const [timeRange, setTimeRange] = useState(() =>
    buildInitialTimeRange(props.initialTimeRange)
  )

  const resetTimeRange = () => {
    if (
      props.initialTimeRange?.start_timestamp &&
      props.initialTimeRange?.end_timestamp
    ) {
      setSelectedRange(null)
      setTimeRange(buildInitialTimeRange(props.initialTimeRange))
      return
    }

    const preferences = getSavedChartPreferences()
    setSelectedRange(preferences.defaultTimeRangeDays)
    setTimeRange(buildDefaultDashboardFilters(preferences))
  }

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) resetTimeRange()
    setOpen(nextOpen)
  }

  const handleQuickRange = (days: number) => {
    const { start, end } = getRollingDateRange(days)
    setTimeRange((prev) => ({
      ...prev,
      start_timestamp: start,
      end_timestamp: end,
    }))
    setSelectedRange(days)
  }

  const handleRecalculate = async () => {
    const start = timeRange.start_timestamp
    const end = timeRange.end_timestamp
    if (!start || !end) {
      toast.error(t('Please select a valid time range first'))
      return
    }

    const startTs = Math.floor(new Date(start).getTime() / 1000)
    const endTs = Math.floor(new Date(end).getTime() / 1000)
    if (endTs <= startTs) {
      toast.error(t('Please select a valid time range first'))
      return
    }

    setRecalculating(true)
    try {
      const { success, message } = await recalculateQuotaData({
        start_timestamp: startTs,
        end_timestamp: endTs,
      })
      if (!success) {
        toast.error(message || t('Recalculation failed'))
        return
      }

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['dashboard', 'models'],
        }),
        queryClient.invalidateQueries({
          queryKey: ['dashboard', 'user-quota'],
        }),
        queryClient.invalidateQueries({
          queryKey: ['rankings'],
        }),
      ])

      toast.success(t('Recalculation completed'))
      setOpen(false)
    } catch {
      toast.error(t('Recalculation failed'))
    } finally {
      setRecalculating(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button variant='outline' size='sm' className={props.triggerClassName} />
        }
      >
        <RefreshCw className='mr-2 h-4 w-4' />
        {t('Recalculate Dashboard')}
      </DialogTrigger>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Recalculate Dashboard')}</DialogTitle>
          <DialogDescription>
            {t(
              'Re-aggregate dashboard data from logs for the selected time range. Existing data will be overwritten.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-4 py-2'>
          <div className='grid gap-2'>
            <Label className='flex items-center gap-2'>
              <Calendar className='h-4 w-4' />
              {t('Quick Range')}
            </Label>
            <div className='grid grid-cols-2 gap-2 sm:flex'>
              {TIME_RANGE_PRESETS.map((range) => (
                <Button
                  key={range.days}
                  type='button'
                  size='sm'
                  variant={selectedRange === range.days ? 'default' : 'outline'}
                  onClick={() => handleQuickRange(range.days)}
                  className={cn(
                    'flex-1',
                    selectedRange === range.days &&
                      'ring-ring ring-2 ring-offset-2'
                  )}
                >
                  {t(range.label)}
                </Button>
              ))}
            </div>
          </div>

          <div className='grid gap-2'>
            <Label>{t('Start Time')}</Label>
            <DateTimePicker
              value={timeRange.start_timestamp}
              onChange={(date) => {
                setTimeRange((prev) => ({
                  ...prev,
                  start_timestamp: date || undefined,
                }))
                setSelectedRange(null)
              }}
              placeholder={t('Select start time')}
            />
          </div>

          <div className='grid gap-2'>
            <Label>{t('End Time')}</Label>
            <DateTimePicker
              value={timeRange.end_timestamp}
              onChange={(date) => {
                setTimeRange((prev) => ({
                  ...prev,
                  end_timestamp: date || undefined,
                }))
                setSelectedRange(null)
              }}
              placeholder={t('Select end time')}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => {
              resetTimeRange()
            }}
          >
            {t('Reset')}
          </Button>
          <Button type='button' onClick={handleRecalculate} disabled={recalculating}>
            <RefreshCw
              className={cn('mr-2 h-4 w-4', recalculating && 'animate-spin')}
            />
            {recalculating ? t('Recalculating...') : t('Recalculate Dashboard')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
