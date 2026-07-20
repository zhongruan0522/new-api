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
      toast.error(t('systemSettings.placeholders.selectAnEndTimeBeforeClearingLogs'))
      return
    }
    if (startTimestamp && endTimestamp && startTimestamp > endTimestamp) {
      toast.error(t('systemSettings.errors.startTimeMustNotBeLaterThanEndTime'))
      return
    }
    if (noTargetSelected) {
      toast.error(t('systemSettings.placeholders.selectAtLeastOneLogTypeToClean'))
      return
    }

    setShowConfirmDialog(true)
  }

  const handleCleanLogs = async () => {
    if (!endTimestamp) {
      toast.error(t('systemSettings.placeholders.selectAnEndTimeBeforeClearingLogs'))
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
        throw new Error(res.message || t('systemSettings.errors.failedToCleanLogs'))
      }
      const summary = summarizeResult(res.data)
      if (summary.total > 0) {
        toast.success(
          t('systemSettings.status.countEntriesRemoved', { count: summary.total }) +
            ' ' +
            summary.detail
        )
      } else {
        toast.success(t('systemSettings.tips.noLogEntriesMatchedTheSelectedRange'))
      }
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('systemSettings.errors.failedToCleanLogs')
      toast.error(message)
    } finally {
      setIsCleaning(false)
    }
  }

  const summarizeResult = (result?: CleanLogsResult) => {
    const parts: string[] = []
    let total = 0
    if (result?.logs !== undefined) {
      parts.push(`${t('systemSettings.titles.usageLogs')}: ${result.logs}`)
      total += result.logs
    }
    if (result?.stored_images !== undefined) {
      parts.push(`${t('systemSettings.fields.storedImages')}: ${result.stored_images}`)
      total += result.stored_images
    }
    if (result?.stored_videos !== undefined) {
      parts.push(`${t('systemSettings.fields.storedVideos')}: ${result.stored_videos}`)
      total += result.stored_videos
    }
    if (result?.audit_logs !== undefined) {
      parts.push(`${t('systemSettings.titles.auditLogs')}: ${result.audit_logs}`)
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
      label: t('systemSettings.titles.usageLogs'),
      description: t('systemSettings.tips.consumeErrorTopupManageAndSystemLogs'),
    },
    {
      key: 'cleanStoredImages',
      label: t('systemSettings.fields.storedImages'),
      description: t(
        'systemSettings.tips.imageBytesCachedForTheImageToUrlConversion'
      ),
    },
    {
      key: 'cleanStoredVideos',
      label: t('systemSettings.fields.storedVideos'),
      description: t(
        'systemSettings.tips.videoBytesCachedForTheVideoToUrlConversion'
      ),
    },
    {
      key: 'cleanAuditLogs',
      label: t('systemSettings.titles.auditLogs'),
      description: t(
        'systemSettings.errors.administrativeOperationRecordsDeletionCannotBeUndone'
      ),
    },
  ]

  return (
    <SettingsSection title={t('systemSettings.fields.logMaintenance')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='common.actions.saveLogSettings'
          />
          <FormField
            control={form.control}
            name='LogConsumeEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('systemSettings.fields.recordQuotaUsage')}</FormLabel>
                  <FormDescription>
                    {t(
                      'systemSettings.tips.trackPerRequestConsumptionToPowerUsageAnalyticsKeeping'
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
              <h4 className='text-sm font-medium'>{t('systemSettings.actions.cleanHistoryLogs')}</h4>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'systemSettings.placeholders.selectLogTypesAndTimeRangeToRemoveMatching'
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
              <div className='text-sm font-medium'>{t('systemSettings.fields.timeRange')}</div>
              <div className='grid gap-2 sm:grid-cols-2'>
                <div className='space-y-1'>
                  <div className='text-muted-foreground text-xs'>
                    {t('subscriptions.actions.start')}
                  </div>
                  <DateTimePicker
                    value={startDate}
                    onChange={setStartDate}
                    placeholder={t('systemSettings.fields.noLowerBound')}
                  />
                </div>
                <div className='space-y-1'>
                  <div className='text-muted-foreground text-xs'>
                    {t('subscriptions.fields.end')}
                  </div>
                  <DateTimePicker
                    value={endDate}
                    onChange={setEndDate}
                    placeholder={t('systemSettings.errors.required')}
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
                {t('systemSettings.fields.last7Days')}
              </Button>
              <Button
                type='button'
                variant='outline'
                onClick={() => {
                  setStartDate(getDateDaysAgo(30))
                  setEndDate(getDateDaysAgo(0))
                }}
              >
                {t('systemSettings.fields.last30Days')}
              </Button>
              <Button
                type='button'
                variant='destructive'
                onClick={handleRequestCleanLogs}
                disabled={isCleaning || noTargetSelected}
              >
                {isCleaning ? t('systemSettings.tips.cleaning') : t('systemSettings.actions.cleanLogs')}
              </Button>
            </div>
          </SettingsControlGroup>
        </SettingsForm>
      </Form>
      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('systemSettings.actions.confirmLogCleanup')}</AlertDialogTitle>
            <AlertDialogDescription>
              {formattedRange
                ? t(
                    'systemSettings.status.permanentlyRemoveLogEntriesCreatedInTheRangeRange',
                    { range: formattedRange }
                  )
                : t(
                    'systemSettings.tips.permanentlyRemoveLogEntriesMatchingTheSelectedRange'
                  )}{' '}
              {t('keys.errors.actionCannotBeUndone951f49')}
              {targets.cleanAuditLogs && (
                <span className='mt-2 block font-medium text-destructive'>
                  {t(
                    'systemSettings.status.auditLogsSelectedAdministrativeOperationRecordsWillBePermanently'
                  )}
                </span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isCleaning}>
              {t('common.actions.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction onClick={handleCleanLogs} disabled={isCleaning}>
              {isCleaning ? t('systemSettings.tips.cleaning') : t('systemSettings.actions.deleteLogs')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
