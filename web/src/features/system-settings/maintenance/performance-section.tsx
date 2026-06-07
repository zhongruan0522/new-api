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
import { useCallback, useEffect, useState } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { StatusBadge } from '@/components/status-badge'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const perfSchema = z.object({
  'performance_setting.disk_cache_enabled': z.boolean(),
  'performance_setting.disk_cache_threshold_mb': z.coerce.number().min(1),
  'performance_setting.disk_cache_max_size_mb': z.coerce.number().min(100),
  'performance_setting.disk_cache_path': z.string().optional(),
  'performance_setting.monitor_enabled': z.boolean(),
  'performance_setting.monitor_cpu_threshold': z.coerce.number().min(0),
  'performance_setting.monitor_memory_threshold': z.coerce
    .number()
    .min(0)
    .max(100),
  'performance_setting.monitor_disk_threshold': z.coerce
    .number()
    .min(0)
    .max(100),
})

type PerfFormValues = z.infer<typeof perfSchema>

function formatBytes(bytes: number, decimals = 2): string {
  if (!bytes || isNaN(bytes)) return '0 Bytes'
  if (bytes === 0) return '0 Bytes'
  if (bytes < 0) return '-' + formatBytes(-bytes, decimals)
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k))
  if (i < 0 || i >= sizes.length) return bytes + ' Bytes'
  return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i]
}

interface Props {
  defaultValues: PerfFormValues
}

type PerformanceStats = {
  cache_stats?: {
    current_disk_usage_bytes: number
    disk_cache_max_bytes: number
    active_disk_files: number
    disk_cache_hits: number
    current_memory_usage_bytes: number
    active_memory_buffers: number
    memory_cache_hits: number
  }
  disk_space_info?: {
    total: number
    free: number
    used: number
    used_percent: number
  }
  memory_stats?: {
    alloc: number
    total_alloc: number
    sys: number
    num_gc: number
    num_goroutine: number
  }
  disk_cache_info?: {
    path: string
    file_count: number
    total_size: number
  }
  config?: {
    is_running_in_container: boolean
  }
}

export function PerformanceSection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [stats, setStats] = useState<PerformanceStats | null>(null)

  const form = useForm<PerfFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(perfSchema) as any,
    defaultValues: props.defaultValues,
  })

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  useResetForm(form as any, props.defaultValues)

  const fetchStats = useCallback(async () => {
    try {
      const res = await api.get('/api/performance/stats')
      if (res.data.success) setStats(res.data.data)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to fetch performance stats')
      )
    }
  }, [t])

  useEffect(() => {
    fetchStats()
  }, [fetchStats])

  const onSubmit = async (data: PerfFormValues) => {
    const entries = Object.entries(data) as [string, unknown][]
    const updates = entries.filter(
      ([key, value]) =>
        value !== (props.defaultValues[key as keyof PerfFormValues] as unknown)
    )
    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }
    for (const [key, value] of updates) {
      await updateOption.mutateAsync({
        key,
        value: value as string | number | boolean,
      })
    }
    toast.success(t('Saved successfully'))
    fetchStats()
  }

  const clearDiskCache = async () => {
    try {
      const res = await api.delete('/api/performance/disk_cache')
      if (res.data.success) {
        toast.success(t('Disk cache cleared'))
        fetchStats()
      }
    } catch {
      toast.error(t('Cleanup failed'))
    }
  }

  const resetStats = async () => {
    try {
      const res = await api.post('/api/performance/reset_stats')
      if (res.data.success) {
        toast.success(t('Statistics reset'))
        fetchStats()
      }
    } catch {
      toast.error(t('Reset failed'))
    }
  }

  const forceGC = async () => {
    try {
      const res = await api.post('/api/performance/gc')
      if (res.data.success) {
        toast.success(t('GC executed'))
        fetchStats()
      }
    } catch {
      toast.error(t('GC execution failed'))
    }
  }

  const diskEnabled = form.watch('performance_setting.disk_cache_enabled')
  const monitorEnabled = form.watch('performance_setting.monitor_enabled')
  const maxCacheSizeMb = form.watch(
    'performance_setting.disk_cache_max_size_mb'
  )

  const lowDiskSpace =
    diskEnabled &&
    stats?.disk_space_info &&
    stats.disk_space_info.free > 0 &&
    maxCacheSizeMb > 0 &&
    stats.disk_space_info.free < maxCacheSizeMb * 1024 * 1024

  const diskCachePercent =
    stats?.cache_stats?.disk_cache_max_bytes &&
    stats.cache_stats.disk_cache_max_bytes > 0
      ? Math.round(
          (stats.cache_stats.current_disk_usage_bytes /
            stats.cache_stats.disk_cache_max_bytes) *
            100
        )
      : 0

  return (
    <SettingsSection title={t('Performance Settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          {/* Disk Cache Settings */}
          <div>
            <h4 className='font-medium'>{t('Disk Cache Settings')}</h4>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t(
                'When enabled, large request bodies are temporarily stored on disk instead of memory, significantly reducing memory usage. SSD recommended.'
              )}
            </p>
          </div>

          <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
            <FormField
              control={form.control}
              name='performance_setting.disk_cache_enabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Enable Disk Cache')}</FormLabel>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
            <FormField
              control={form.control}
              name='performance_setting.disk_cache_threshold_mb'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Disk Cache Threshold (MB)')}</FormLabel>
                  <FormControl>
                    <Input type='number' {...field} disabled={!diskEnabled} />
                  </FormControl>
                  <FormDescription>
                    {t('Use disk cache when request body exceeds this size')}
                  </FormDescription>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='performance_setting.disk_cache_max_size_mb'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max Disk Cache Size (MB)')}</FormLabel>
                  <FormControl>
                    <Input type='number' {...field} disabled={!diskEnabled} />
                  </FormControl>
                  {stats?.disk_space_info &&
                    stats.disk_space_info.total > 0 && (
                      <FormDescription>
                        {t('Free: {{free}} / Total: {{total}}', {
                          free: formatBytes(stats.disk_space_info.free),
                          total: formatBytes(stats.disk_space_info.total),
                        })}
                      </FormDescription>
                    )}
                </FormItem>
              )}
            />
          </div>

          {lowDiskSpace && (
            <Alert variant='destructive'>
              <AlertDescription>
                {`${t('Warning')}: ${t('Available disk space')} (${formatBytes(stats?.disk_space_info?.free ?? 0)}) ${t('is less than the configured maximum cache size')} (${maxCacheSizeMb} MB). ${t('This may cause cache failures.')}`}
              </AlertDescription>
            </Alert>
          )}

          {!stats?.config?.is_running_in_container && (
            <FormField
              control={form.control}
              name='performance_setting.disk_cache_path'
              render={({ field }) => (
                <FormItem className='max-w-md'>
                  <FormLabel>{t('Cache Directory')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t(
                        'Leave empty to use system temp directory'
                      )}
                      {...field}
                      value={field.value ?? ''}
                      disabled={!diskEnabled}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
          )}

          <Separator />

          {/* System Performance Monitor */}
          <div>
            <h4 className='font-medium'>
              {t('System Performance Monitoring')}
            </h4>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t(
                'When performance monitoring is enabled and system resource usage exceeds the set threshold, new Relay requests will be rejected.'
              )}
            </p>
          </div>

          <div className='grid grid-cols-1 gap-4 md:grid-cols-4'>
            <FormField
              control={form.control}
              name='performance_setting.monitor_enabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Enable Performance Monitoring')}</FormLabel>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />
            <FormField
              control={form.control}
              name='performance_setting.monitor_cpu_threshold'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('CPU Threshold (%)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      {...field}
                      disabled={!monitorEnabled}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='performance_setting.monitor_memory_threshold'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Memory Threshold (%)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      {...field}
                      disabled={!monitorEnabled}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='performance_setting.monitor_disk_threshold'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Disk Threshold (%)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      {...field}
                      disabled={!monitorEnabled}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>

      <Separator />

      {/* Performance Stats Dashboard */}
      <div className='space-y-4'>
        <div className='flex items-center gap-2'>
          <h4 className='font-medium'>{t('Performance Monitor')}</h4>
          <Button variant='outline' size='sm' onClick={fetchStats}>
            {t('Refresh Stats')}
          </Button>
          <AlertDialog>
            <AlertDialogTrigger render={<Button variant='outline' size='sm' />}>
              {t('Clean up inactive cache')}
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  {t('Confirm cleanup of inactive disk cache?')}
                </AlertDialogTitle>
                <AlertDialogDescription>
                  {t(
                    'This will delete temporary cache files that have not been used for more than 10 minutes'
                  )}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
                <AlertDialogAction onClick={clearDiskCache}>
                  {t('Confirm')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <Button variant='outline' size='sm' onClick={resetStats}>
            {t('Reset Stats')}
          </Button>
          <Button variant='outline' size='sm' onClick={forceGC}>
            {t('Run GC')}
          </Button>
        </div>

        {stats && (
          <>
            <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
              <div className='space-y-2 rounded-lg border p-4'>
                <p className='text-sm font-medium'>
                  {t('Request Body Disk Cache')}
                </p>
                <Progress value={diskCachePercent} />
                <div className='text-muted-foreground flex justify-between text-xs'>
                  <span>
                    {formatBytes(
                      stats.cache_stats?.current_disk_usage_bytes ?? 0
                    )}{' '}
                    /{' '}
                    {formatBytes(stats.cache_stats?.disk_cache_max_bytes ?? 0)}
                  </span>
                  <span>
                    {t('Active Files')}:{' '}
                    {stats.cache_stats?.active_disk_files ?? 0}
                  </span>
                </div>
                <StatusBadge variant='neutral' copyable={false}>
                  {t('Disk Hits')}: {stats.cache_stats?.disk_cache_hits ?? 0}
                </StatusBadge>
              </div>
              <div className='space-y-2 rounded-lg border p-4'>
                <p className='text-sm font-medium'>
                  {t('Request Body Memory Cache')}
                </p>
                <div className='text-muted-foreground flex justify-between text-xs'>
                  <span>
                    {t('Current Cache Size')}:{' '}
                    {formatBytes(
                      stats.cache_stats?.current_memory_usage_bytes ?? 0
                    )}
                  </span>
                  <span>
                    {t('Active Cache Count')}:{' '}
                    {stats.cache_stats?.active_memory_buffers ?? 0}
                  </span>
                </div>
                <StatusBadge variant='neutral' copyable={false}>
                  {t('Memory Hits')}:{' '}
                  {stats.cache_stats?.memory_cache_hits ?? 0}
                </StatusBadge>
              </div>
            </div>

            {stats.disk_space_info && stats.disk_space_info.total > 0 && (
              <div className='rounded-lg border p-4'>
                <p className='mb-2 text-sm font-medium'>
                  {t('Cache Directory Disk Space')}
                </p>
                <Progress
                  value={Math.round(stats.disk_space_info.used_percent)}
                />
                <div className='text-muted-foreground mt-2 flex justify-between text-xs'>
                  <span>
                    {t('Used')}: {formatBytes(stats.disk_space_info.used)}
                  </span>
                  <span>
                    {t('Available')}: {formatBytes(stats.disk_space_info.free)}
                  </span>
                  <span>
                    {t('Total')}: {formatBytes(stats.disk_space_info.total)}
                  </span>
                </div>
              </div>
            )}

            {stats.memory_stats && (
              <div className='rounded-lg border p-4'>
                <p className='mb-2 text-sm font-medium'>
                  {t('System Memory Stats')}
                </p>
                <div className='grid grid-cols-2 gap-2 text-xs md:grid-cols-5'>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Allocated Memory')}:
                    </span>{' '}
                    {formatBytes(stats.memory_stats.alloc)}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Total Allocated')}:
                    </span>{' '}
                    {formatBytes(stats.memory_stats.total_alloc)}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('System Memory')}:
                    </span>{' '}
                    {formatBytes(stats.memory_stats.sys)}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('GC Count')}:
                    </span>{' '}
                    {stats.memory_stats.num_gc}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>Goroutines:</span>{' '}
                    {stats.memory_stats.num_goroutine}
                  </div>
                </div>
              </div>
            )}

            {stats.disk_cache_info && (
              <div className='rounded-lg border p-4'>
                <p className='mb-2 text-sm font-medium'>
                  {t('Cache Directory Info')}
                </p>
                <div className='grid grid-cols-3 gap-2 text-xs'>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Cache Directory')}:
                    </span>{' '}
                    <span className='font-mono'>
                      {stats.disk_cache_info.path}
                    </span>
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Directory File Count')}:
                    </span>{' '}
                    {stats.disk_cache_info.file_count}
                  </div>
                  <div>
                    <span className='text-muted-foreground'>
                      {t('Directory Total Size')}:
                    </span>{' '}
                    {formatBytes(stats.disk_cache_info.total_size)}
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </SettingsSection>
  )
}
