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
import { useEffect } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const dataDashboardSchema = z.object({
  DataExportEnabled: z.boolean(),
  DataExportInterval: z.number().int().min(1).max(1440),
  DataExportDefaultTime: z.enum(['hour', 'day', 'week']),
})

type DataDashboardFormValues = z.infer<typeof dataDashboardSchema>

type DashboardSectionProps = {
  defaultValues: DataDashboardFormValues
}

const granularityOptions = [
  { label: 'channels.fields.hours', value: 'hour' },
  { label: 'common.fields.day', value: 'day' },
  { label: 'common.fields.week', value: 'week' },
]

export function DashboardSection({ defaultValues }: DashboardSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<DataDashboardFormValues>({
    resolver: zodResolver(dataDashboardSchema),
    defaultValues,
  })

  useEffect(() => {
    form.reset(defaultValues)
  }, [defaultValues, form])

  const onSubmit = async (values: DataDashboardFormValues) => {
    const updates = Object.entries(values).filter(
      ([key, value]) =>
        value !== defaultValues[key as keyof DataDashboardFormValues]
    )

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  const isEnabled = form.watch('DataExportEnabled')

  return (
    <SettingsSection title={t('systemSettings.titles.dataDashboard')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='DataExportEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('systemSettings.actions.enableDataDashboard')}</FormLabel>
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

          <div className='grid gap-6 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='DataExportInterval'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('systemSettings.actions.refreshIntervalMinutes')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={1}
                      max={1440}
                      step={1}
                      disabled={!isEnabled}
                      value={field.value}
                      onChange={(e) => field.onChange(e.target.valueAsNumber)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('systemSettings.tips.keepThisAbove1MinuteToAvoidHeavyDatabase')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='DataExportDefaultTime'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('dashboard.fields.defaultTimeGranularity')}</FormLabel>
                  <Select
                    items={[
                      ...granularityOptions.map((option) => ({
                        value: option.value,
                        label: t(option.label),
                      })),
                    ]}
                    onValueChange={field.onChange}
                    value={field.value}
                    disabled={!isEnabled}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('systemSettings.placeholders.selectGranularity')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {granularityOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {t(option.label)}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t(
                      'systemSettings.tips.uiGranularityOnlyDataIsStillAggregatedHourly'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
