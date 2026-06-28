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
import { useEffect, useMemo, useState } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatTimestampToDate } from '@/lib/format'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'
import { DateTimePicker } from '@/components/datetime-picker'
import { cleanLogs } from '../api'
import type { CleanLogsResult } from '../types'
import {
  SettingsControlGroup,
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const logSettingsSchema = z.object({
  LogConsumeEnabled: z.boolean(),
})

type LogSettingsFormValues = z.infer<typeof logSettingsSchema>

type LogSettingsSectionProps = {
  defaultEnabled: boolean
}

const HOURS_IN_DAY = 24

const getDateHoursAgo = (hours: number) => {
  const date = new Date()
  date.setHours(date.getHours() - hours)
  return date
}

const getDateDaysAgo = (days: number) => getDateHoursAgo(days * HOURS_IN_DAY)

type CleanTargets = {
  cleanLogs: boolean
  cleanStoredImages: boolean
  cleanStoredVideos: boolean
  cleanAuditLogs: boolean
}

export function LogSettingsSection({
  defaultEnabled,
}: LogSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<LogSettingsFormValues>({
    resolver: zodResolver(logSettingsSchema),
    defaultValues: {
      LogConsumeEnabled: defaultEnabled,
    },
  })

  // 默认仅勾选使用日志
  const [targets, setTargets] = useState<CleanTargets>({
    cleanLogs: true,
    cleanStoredImages: false,
    cleanStoredVideos: false,
    cleanAuditLogs: false,
  })
  const [startDate, setStartDate] = useState<Date | undefined>(() =>
    getDateDaysAgo(30)
  )
  const [endDate, setEndDate] = useState<Date | undefined>(() => getDateDaysAgo(0))
  const [isCleaning, setIsCleaning] = useState(false)
  const [showConfirmDialog, setShowConfirmDialog] = useState(false)

  useEffect(() => {
    form.reset({ LogConsumeEnabled: defaultEnabled })
  }, [defaultEnabled, form])

  const startTimestamp = useMemo(() => {
    if (!startDate) return null
    return Math.floor(startDate.getTime() / 1000)
  }, [startDate])

  const endTimestamp = useMemo(() => {
    if (!endDate) return null
    return Math.floor(endDate.getTime() / 1000)
  }, [endDate])

  const formattedRange = useMemo(() => {
    const parts: string[] = []
    if (startTimestamp) {
      parts.push(formatTimestampToDate(startTimestamp * 1000, 'milliseconds'))
    }
    if (endTimestamp) {
      parts.push(formatTimestampToDate(endTimestamp * 1000, 'milliseconds'))
    }
    return parts.join(' ~ ')
  }, [startTimestamp, endTimestamp])

  const noTargetSelected =
    !targets.cleanLogs &&
    !targets.cleanStoredImages &&
    !targets.cleanStoredVideos &&
    !targets.cleanAuditLogs

  const onSubmit = async (values: LogSettingsFormValues) => {
    if (values.LogConsumeEnabled === defaultEnabled) return
    await updateOption.mutateAsync({
      key: 'LogConsumeEnabled',
      value: values.LogConsumeEnabled,
    })
  }

  const handleRequestCleanLogs = () => {
    if (!endTimestamp) {
      toast.error(t('Select an end time before clearing logs.'))
      return
    }
    if (startTimestamp && endTimestamp && startTimestamp > endTimestamp) {
      toast.error(t('Start time must not be later than end time.'))
      return
    }
    if (noTargetSelected) {
      toast.error(t('Select at least one log type to clean.'))
      return
    }

    setShowConfirmDialog(true)
  }

  const handleCleanLogs = async () => {
    if (!endTimestamp) {
      toast.error(t('Select an end time before clearing logs.'))
      return
    }

    setIsCleaning(true)
    try {
      const res = await cleanLogs({
        start_timestamp: startTimestamp ?? undefined,
        end_timestamp: endTimestamp,
        clean_logs: targets.cleanLogs,
        clean_stored_images: targets.cleanStoredImages,
        clean_stored_videos: targets.cleanStoredVideos,
        clean_audit_logs: targets.cleanAuditLogs,
      })
      if (!res.success) {
        throw new Error(res.message || t('Failed to clean logs'))
      }
      const summary = summarizeResult(res.data)
      if (summary.total > 0) {
        toast.success(
          t('{{count}} entries removed.', { count: summary.total }) +
            ' ' +
            summary.detail
        )
      } else {
        toast.success(t('No log entries matched the selected range.'))
      }
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('Failed to clean logs')
      toast.error(message)
    } finally {
      setIsCleaning(false)
    }
  }

  const summarizeResult = (result?: CleanLogsResult) => {
    const parts: string[] = []
    let total = 0
    if (result?.logs !== undefined) {
      parts.push(`${t('Usage logs')}: ${result.logs}`)
      total += result.logs
    }
    if (result?.stored_images !== undefined) {
      parts.push(`${t('Stored images')}: ${result.stored_images}`)
      total += result.stored_images
    }
    if (result?.stored_videos !== undefined) {
      parts.push(`${t('Stored videos')}: ${result.stored_videos}`)
      total += result.stored_videos
    }
    if (result?.audit_logs !== undefined) {
      parts.push(`${t('Audit logs')}: ${result.audit_logs}`)
      total += result.audit_logs
    }
    return { total, detail: parts.join(', ') }
  }

  const targetOptions: Array<{
    key: keyof CleanTargets
    label: string
    description: string
  }> = [
    {
      key: 'cleanLogs',
      label: t('Usage logs'),
      description: t('Consume, error, topup, manage and system logs.'),
    },
    {
      key: 'cleanStoredImages',
      label: t('Stored images'),
      description: t(
        'Image bytes cached for the image-to-URL conversion feature.'
      ),
    },
    {
      key: 'cleanStoredVideos',
      label: t('Stored videos'),
      description: t(
        'Video bytes cached for the video-to-URL conversion feature.'
      ),
    },
    {
      key: 'cleanAuditLogs',
      label: t('Audit logs'),
      description: t(
        'Administrative operation records. Deletion cannot be undone.'
      ),
    },
  ]

  return (
    <SettingsSection title={t('Log Maintenance')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save log settings'
          />
          <FormField
            control={form.control}
            name='LogConsumeEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Record quota usage')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Track per-request consumption to power usage analytics. Keeping this on increases database writes.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
                <FormMessage />
              </SettingsSwitchItem>
            )}
          />

          <SettingsControlGroup className='space-y-3'>
            <div>
              <h4 className='text-sm font-medium'>{t('Clean history logs')}</h4>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Select log types and time range to remove matching entries.'
                )}
              </p>
            </div>

            <div className='space-y-2'>
              {targetOptions.map((option) => (
                <label
                  key={option.key}
                  className='flex items-start gap-3 rounded-md border p-3 cursor-pointer hover:bg-accent/50'
                >
                  <Checkbox
                    checked={targets[option.key]}
                    onCheckedChange={(value) =>
                      setTargets((prev) => ({
                        ...prev,
                        [option.key]: value === true,
                      }))
                    }
                    className='mt-0.5'
                  />
                  <div className='space-y-0.5'>
                    <div className='text-sm font-medium leading-none'>
                      {option.label}
                    </div>
                    <p className='text-muted-foreground text-xs'>
                      {option.description}
                    </p>
                  </div>
                </label>
              ))}
            </div>

            <div className='space-y-2'>
              <div className='text-sm font-medium'>{t('Time range')}</div>
              <div className='grid gap-2 sm:grid-cols-2'>
                <div className='space-y-1'>
                  <div className='text-muted-foreground text-xs'>
                    {t('Start')}
                  </div>
                  <DateTimePicker
                    value={startDate}
                    onChange={setStartDate}
                    placeholder={t('No lower bound')}
                  />
                </div>
                <div className='space-y-1'>
                  <div className='text-muted-foreground text-xs'>
                    {t('End')}
                  </div>
                  <DateTimePicker
                    value={endDate}
                    onChange={setEndDate}
                    placeholder={t('Required')}
                  />
                </div>
              </div>
            </div>

            <div className='flex flex-wrap gap-3'>
              <Button
                type='button'
                variant='outline'
                onClick={() => {
                  setStartDate(getDateDaysAgo(7))
                  setEndDate(getDateDaysAgo(0))
                }}
              >
                {t('Last 7 days')}
              </Button>
              <Button
                type='button'
                variant='outline'
                onClick={() => {
                  setStartDate(getDateDaysAgo(30))
                  setEndDate(getDateDaysAgo(0))
                }}
              >
                {t('Last 30 days')}
              </Button>
              <Button
                type='button'
                variant='destructive'
                onClick={handleRequestCleanLogs}
                disabled={isCleaning || noTargetSelected}
              >
                {isCleaning ? t('Cleaning...') : t('Clean logs')}
              </Button>
            </div>
          </SettingsControlGroup>
        </SettingsForm>
      </Form>
      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Confirm log cleanup')}</AlertDialogTitle>
            <AlertDialogDescription>
              {formattedRange
                ? t(
                    'This will permanently remove log entries created in the range {{range}}.',
                    { range: formattedRange }
                  )
                : t(
                    'This will permanently remove log entries matching the selected range.'
                  )}{' '}
              {t('This action cannot be undone.')}
              {targets.cleanAuditLogs && (
                <span className='mt-2 block font-medium text-destructive'>
                  {t(
                    'Audit logs selected: administrative operation records will be permanently deleted.'
                  )}
                </span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isCleaning}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction onClick={handleCleanLogs} disabled={isCleaning}>
              {isCleaning ? t('Cleaning...') : t('Delete logs')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
