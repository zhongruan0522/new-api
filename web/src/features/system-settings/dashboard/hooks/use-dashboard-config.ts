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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import {
  getDashboardConfig,
  updateDashboardConfig,
  resetDashboardConfig,
} from '../api'
import type { DashboardConfigUpdate } from '../types'

/**
 * 获取仪表板配置
 */
export function useDashboardConfig() {
  return useQuery({
    queryKey: ['dashboard-config'],
    queryFn: getDashboardConfig,
    staleTime: 5 * 60 * 1000, // 5分钟
  })
}

/**
 * 更新仪表板配置
 */
export function useUpdateDashboardConfig() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (updates: DashboardConfigUpdate) =>
      updateDashboardConfig(updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard-config'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      queryClient.invalidateQueries({ queryKey: ['system-options'] })
      toast.success(t('Settings updated successfully'))
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })
}

/**
 * 重置仪表板配置
 */
export function useResetDashboardConfig() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: resetDashboardConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard-config'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      queryClient.invalidateQueries({ queryKey: ['system-options'] })
      toast.success(t('Configuration reset to defaults'))
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })
}
