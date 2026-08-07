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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'
import { AUDIT_ACTION_TYPES, AUDIT_MODULES } from '../constants'

export interface AuditLogFilters {
  username: string
  module: string
  actionType: string
  startTime: Date | undefined
  endTime: Date | undefined
}

interface AuditLogsFilterBarProps {
  filters: AuditLogFilters
  onApply: (filters: AuditLogFilters) => void
  onReset: () => void
  loading?: boolean
}

export function AuditLogsFilterBar({
  filters,
  onApply,
  onReset,
  loading,
}: AuditLogsFilterBarProps) {
  const { t } = useTranslation()

  const [localFilters, setLocalFilters] = useState<AuditLogFilters>(filters)

  useEffect(() => {
    setLocalFilters(filters)
  }, [filters])

  const handleChange = useCallback(
    (field: keyof AuditLogFilters, value: string | Date | undefined) => {
      setLocalFilters((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  const handleApply = useCallback(() => {
    onApply(localFilters)
  }, [localFilters, onApply])

  const handleReset = useCallback(() => {
    const resetFilters: AuditLogFilters = {
      username: '',
      module: '',
      actionType: '',
      startTime: undefined,
      endTime: undefined,
    }
    setLocalFilters(resetFilters)
    onReset()
  }, [onReset])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') handleApply()
    },
    [handleApply]
  )

  const moduleItems = useMemo(
    () => AUDIT_MODULES.map((m) => ({ value: m.value, label: t(m.label) })),
    [t]
  )
  const actionTypeItems = useMemo(
    () =>
      AUDIT_ACTION_TYPES.map((a) => ({ value: a.value, label: t(a.label) })),
    [t]
  )

  const moduleLabel =
    moduleItems.find((m) => m.value === localFilters.module)?.label ??
    t('auditLogs.fields.allModules')
  const actionTypeLabel =
    actionTypeItems.find((a) => a.value === localFilters.actionType)?.label ??
    t('auditLogs.fields.allActionTypes')

  return (
    <div className='flex flex-wrap items-center gap-2 sm:gap-3'>
      <div className='w-full sm:w-[280px]'>
        <CompactDateTimeRangePicker
          start={localFilters.startTime}
          end={localFilters.endTime}
          onChange={({ start, end }) => {
            handleChange('startTime', start)
            handleChange('endTime', end)
          }}
        />
      </div>

      <Input
        placeholder={t('auditLogs.fields.operator')}
        autoComplete='off'
        value={localFilters.username}
        onChange={(e) => handleChange('username', e.target.value)}
        onKeyDown={handleKeyDown}
        className='w-full sm:w-[160px]'
      />

      <Select
        items={moduleItems}
        value={localFilters.module}
        onValueChange={(value) => {
          handleChange('module', value ?? '')
        }}
      >
        <SelectTrigger className='w-full sm:w-[160px]'>
          <SelectValue>{moduleLabel}</SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            <SelectItem value=''>{t('auditLogs.fields.allModules')}</SelectItem>
            {AUDIT_MODULES.map((m) => (
              <SelectItem key={m.value} value={m.value}>
                {t(m.label)}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>

      <Select
        items={actionTypeItems}
        value={localFilters.actionType}
        onValueChange={(value) => {
          handleChange('actionType', value ?? '')
        }}
      >
        <SelectTrigger className='w-full sm:w-[160px]'>
          <SelectValue>{actionTypeLabel}</SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            <SelectItem value=''>{t('auditLogs.fields.allActionTypes')}</SelectItem>
            {AUDIT_ACTION_TYPES.map((a) => (
              <SelectItem key={a.value} value={a.value}>
                {t(a.label)}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>

      <div className='ms-auto flex shrink-0 items-center gap-1.5 sm:gap-2'>
        <Button variant='outline' onClick={handleReset}>
          {t('common.actions.reset')}
        </Button>
        <Button onClick={handleApply} disabled={loading}>
          {t('common.actions.search')}
        </Button>
      </div>
    </div>
  )
}
