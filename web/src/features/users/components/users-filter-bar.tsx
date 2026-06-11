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
import { useState, useEffect, useCallback } from 'react'
import { useQueryClient, useIsFetching } from '@tanstack/react-query'
import { useNavigate, getRouteApi } from '@tanstack/react-router'
import { type Table } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getUserStatusOptions, getUserRoleOptions } from '../constants'
import {
  LogsFilterField,
  LogsFilterInput,
  LogsFilterToolbar,
} from '@/features/usage-logs/components/logs-filter-toolbar'

const route = getRouteApi('/_authenticated/users/')

interface UsersFilterBarProps<TData> {
  table: Table<TData>
}

interface UserFilters {
  username?: string
  displayName?: string
  email?: string
  linuxDoId?: string
  githubId?: string
  status?: string
  role?: string
}

export function UsersFilterBar<TData>(props: UsersFilterBarProps<TData>) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const searchParams = route.useSearch()
  const fetchingUsers = useIsFetching({ queryKey: ['users'] })

  const [filters, setFilters] = useState<UserFilters>({})
  const [status, setStatus] = useState<string>('')
  const [role, setRole] = useState<string>('')

  useEffect(() => {
    setFilters({
      username: searchParams.username || undefined,
      displayName: searchParams.display_name || undefined,
      email: searchParams.email || undefined,
      linuxDoId: searchParams.linux_do_id || undefined,
      githubId: searchParams.github_id || undefined,
    })
    setStatus(searchParams.status || '')
    setRole(searchParams.role || '')
  }, [
    searchParams.username,
    searchParams.display_name,
    searchParams.email,
    searchParams.linux_do_id,
    searchParams.github_id,
    searchParams.status,
    searchParams.role,
  ])

  const handleChange = useCallback(
    (field: keyof UserFilters, value: string | undefined) => {
      setFilters((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  const handleApply = useCallback(() => {
    navigate({
      to: '/users',
      search: {
        username: filters.username,
        display_name: filters.displayName,
        email: filters.email,
        linux_do_id: filters.linuxDoId,
        github_id: filters.githubId,
        status,
        role,
        page: 1,
      },
    })
    queryClient.invalidateQueries({ queryKey: ['users'] })
  }, [filters, status, role, navigate, queryClient])

  const handleReset = useCallback(() => {
    setFilters({})
    setStatus('')
    setRole('')
    navigate({
      to: '/users',
      search: { page: 1 },
    })
    queryClient.invalidateQueries({ queryKey: ['users'] })
  }, [navigate, queryClient])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') handleApply()
    },
    [handleApply]
  )

  const hasActiveFilters =
    !!filters.username ||
    !!filters.displayName ||
    !!filters.email ||
    !!filters.linuxDoId ||
    !!filters.githubId ||
    !!status ||
    !!role

  const activeFilterCount = [
    filters.username,
    filters.displayName,
    filters.email,
    filters.linuxDoId,
    filters.githubId,
    status,
    role,
  ].filter(Boolean).length

  const statusOptions = getUserStatusOptions(t)
  const roleOptions = getUserRoleOptions(t)
  const statusLabel =
    statusOptions.find((opt) => opt.value === status)?.label ?? t('Status')
  const roleLabel =
    roleOptions.find((opt) => opt.value === role)?.label ?? t('Role')

  const usernameFilter = (
    <LogsFilterField>
      <LogsFilterInput
        placeholder={t('Username')}
        autoComplete="off"
        value={filters.username || ''}
        onChange={(e) => handleChange('username', e.target.value)}
        onKeyDown={handleKeyDown}
      />
    </LogsFilterField>
  )

  const displayNameFilter = (
    <LogsFilterField>
      <LogsFilterInput
        placeholder={t('Display Name')}
        autoComplete="off"
        value={filters.displayName || ''}
        onChange={(e) => handleChange('displayName', e.target.value)}
        onKeyDown={handleKeyDown}
      />
    </LogsFilterField>
  )

  const emailFilter = (
    <LogsFilterField>
      <LogsFilterInput
        placeholder={t('Email')}
        autoComplete="off"
        value={filters.email || ''}
        onChange={(e) => handleChange('email', e.target.value)}
        onKeyDown={handleKeyDown}
      />
    </LogsFilterField>
  )

  const statusFilter = (
    <LogsFilterField>
      <Select
        items={statusOptions}
        value={status}
        onValueChange={(value) => setStatus(value || '')}
      >
        <SelectTrigger>
          <SelectValue>{statusLabel}</SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {statusOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </LogsFilterField>
  )

  const roleFilter = (
    <LogsFilterField>
      <Select
        items={roleOptions}
        value={role}
        onValueChange={(value) => setRole(value || '')}
      >
        <SelectTrigger>
          <SelectValue>{roleLabel}</SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {roleOptions.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </LogsFilterField>
  )

  const advancedFilters = (
    <>
      <LogsFilterField>
        <LogsFilterInput
          placeholder={t('LinuxDo ID')}
          autoComplete="off"
          value={filters.linuxDoId || ''}
          onChange={(e) => handleChange('linuxDoId', e.target.value)}
          onKeyDown={handleKeyDown}
        />
      </LogsFilterField>
      <LogsFilterField>
        <LogsFilterInput
          placeholder={t('GitHub ID')}
          autoComplete="off"
          value={filters.githubId || ''}
          onChange={(e) => handleChange('githubId', e.target.value)}
          onKeyDown={handleKeyDown}
        />
      </LogsFilterField>
    </>
  )

  return (
    <LogsFilterToolbar
      table={props.table}
      primaryFilters={
        <>
          {usernameFilter}
          {displayNameFilter}
          {emailFilter}
          {statusFilter}
          {roleFilter}
        </>
      }
      advancedFilters={advancedFilters}
      mobilePinnedFilters={usernameFilter}
      mobileFilters={
        <>
          {displayNameFilter}
          {emailFilter}
          {statusFilter}
          {roleFilter}
          {advancedFilters}
        </>
      }
      mobileFilterCount={activeFilterCount}
      hasAdvancedActiveFilters={!!filters.linuxDoId || !!filters.githubId}
      advancedFilterCount={
        [filters.linuxDoId, filters.githubId].filter(Boolean).length
      }
      hasActiveFilters={hasActiveFilters}
      onSearch={handleApply}
      searchLoading={fetchingUsers > 0}
      onReset={handleReset}
    />
  )
}
