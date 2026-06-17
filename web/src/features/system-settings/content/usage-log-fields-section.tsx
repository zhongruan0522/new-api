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
import { Save, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  USAGE_LOG_FIELDS,
  USAGE_LOG_FIELD_GROUPS,
  buildDefaultUsageLogFieldsJSON,
  resolveUsageLogFieldsConfig,
  type UsageLogFieldGroup,
  type UsageLogFieldKey,
} from '@/features/usage-logs/lib/field-visibility'

type UsageLogFieldsSectionProps = {
  fieldsData: string
  adminEnabled: boolean
  userEnabled: boolean
}

export function UsageLogFieldsSection({
  fieldsData,
  adminEnabled,
  userEnabled,
}: UsageLogFieldsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const [config, setConfig] = useState<
    Record<string, { admin: boolean; user: boolean }>
  >(() => resolveUsageLogFieldsConfig(fieldsData))
  const [isAdminEnabled, setIsAdminEnabled] = useState(adminEnabled)
  const [isUserEnabled, setIsUserEnabled] = useState(userEnabled)
  const [hasChanges, setHasChanges] = useState(false)

  useEffect(() => {
    setConfig(resolveUsageLogFieldsConfig(fieldsData))
    setHasChanges(false)
  }, [fieldsData])

  useEffect(() => {
    setIsAdminEnabled(adminEnabled)
  }, [adminEnabled])

  useEffect(() => {
    setIsUserEnabled(userEnabled)
  }, [userEnabled])

  const groupedFields = useMemo(() => {
    const groups: Partial<Record<UsageLogFieldGroup, typeof USAGE_LOG_FIELDS>> =
      {}
    for (const g of USAGE_LOG_FIELD_GROUPS) {
      groups[g.value] = USAGE_LOG_FIELDS.filter((f) => f.group === g.value)
    }
    return groups
  }, [])

  const toggleField = (
    key: UsageLogFieldKey,
    role: 'admin' | 'user',
    value: boolean
  ) => {
    setConfig((prev) => ({
      ...prev,
      [key]: { ...prev[key], [role]: value },
    }))
    setHasChanges(true)
  }

  const handleToggleAdminEnabled = async (checked: boolean) => {
    try {
      await updateOption.mutateAsync({
        key: 'console_setting.usage_log_fields_admin_enabled',
        value: checked,
      })
      setIsAdminEnabled(checked)
    } catch {
      // useUpdateOption 已处理错误提示
    }
  }

  const handleToggleUserEnabled = async (checked: boolean) => {
    try {
      await updateOption.mutateAsync({
        key: 'console_setting.usage_log_fields_user_enabled',
        value: checked,
      })
      setIsUserEnabled(checked)
    } catch {
      // useUpdateOption 已处理错误提示
    }
  }

  const handleSaveAll = async () => {
    try {
      await updateOption.mutateAsync({
        key: 'console_setting.usage_log_fields',
        value: JSON.stringify(config),
      })
      setHasChanges(false)
    } catch {
      // useUpdateOption 已处理错误提示
    }
  }

  const handleReset = () => {
    setConfig(resolveUsageLogFieldsConfig(buildDefaultUsageLogFieldsJSON()))
    setHasChanges(true)
    toast.success(t('Reset to defaults. Click "Save Settings" to apply.'))
  }

  return (
    <SettingsSection title={t('Usage Log Field Visibility')}>
      <div className='space-y-4'>
        {/* 总开关 */}
        <div className='space-y-1 rounded-md border p-3'>
          <SettingsSwitchField
            checked={isAdminEnabled}
            onCheckedChange={handleToggleAdminEnabled}
            label={t('Admin Details Access')}
            description={t(
              'When enabled, admins can access usage log details dialog'
            )}
            className='border-b-0 py-0'
          />
          <SettingsSwitchField
            checked={isUserEnabled}
            onCheckedChange={handleToggleUserEnabled}
            label={t('User Details Access')}
            description={t(
              'When enabled, regular users can access usage log details dialog'
            )}
            className='border-b-0 py-0'
          />
        </div>

        {/* 操作按钮 */}
        <div className='flex flex-wrap items-center gap-2'>
          <Button
            onClick={handleSaveAll}
            size='sm'
            variant='secondary'
            disabled={!hasChanges || updateOption.isPending}
          >
            <Save className='mr-2 h-4 w-4' />
            {updateOption.isPending ? t('Saving...') : t('Save Settings')}
          </Button>
          <Button
            onClick={handleReset}
            size='sm'
            variant='outline'
            disabled={updateOption.isPending}
          >
            <RotateCcw className='mr-2 h-4 w-4' />
            {t('Reset')}
          </Button>
        </div>

        {/* 字段可见性表格 */}
        <div className='overflow-x-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className='min-w-[200px]'>
                  {t('Field / Description')}
                </TableHead>
                <TableHead className='w-32 text-center'>
                  {t('Admin')}
                </TableHead>
                <TableHead className='w-32 text-center'>
                  {t('User')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {USAGE_LOG_FIELD_GROUPS.map((group) => {
                const fields = groupedFields[group.value] ?? []
                if (fields.length === 0) return null
                return (
                  <FieldGroupRows
                    key={group.value}
                    groupLabel={t(group.labelKey)}
                    fields={fields}
                    config={config}
                    onToggle={toggleField}
                    t={t}
                  />
                )
              })}
            </TableBody>
          </Table>
        </div>
      </div>
    </SettingsSection>
  )
}

function FieldGroupRows({
  groupLabel,
  fields,
  config,
  onToggle,
  t,
}: {
  groupLabel: string
  fields: typeof USAGE_LOG_FIELDS
  config: Record<string, { admin: boolean; user: boolean }>
  onToggle: (
    key: UsageLogFieldKey,
    role: 'admin' | 'user',
    value: boolean
  ) => void
  t: (key: string) => string
}) {
  return (
    <>
      <TableRow className='bg-muted/40 hover:bg-muted/40'>
        <TableCell colSpan={3} className='py-1.5 text-xs font-semibold'>
          {groupLabel}
        </TableCell>
      </TableRow>
      {fields.map((field) => {
        const cfg = config[field.key] ?? { admin: field.admin, user: field.user }
        return (
          <TableRow key={field.key}>
            <TableCell>
              <div className='flex flex-col gap-0.5'>
                <span className='text-sm font-medium'>{t(field.nameKey)}</span>
                <span className='text-muted-foreground text-xs'>
                  {t(field.descriptionKey)}
                </span>
              </div>
            </TableCell>
            <TableCell className='text-center'>
              <Switch
                checked={cfg.admin}
                onCheckedChange={(v) => onToggle(field.key, 'admin', v)}
              />
            </TableCell>
            <TableCell className='text-center'>
              <Switch
                checked={cfg.user}
                onCheckedChange={(v) => onToggle(field.key, 'user', v)}
              />
            </TableCell>
          </TableRow>
        )
      })}
    </>
  )
}
