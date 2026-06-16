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
import { api } from '@/lib/api'
import type {
  DashboardConfig,
  DashboardConfigUpdate,
  DashboardConfigResponse,
} from './types'

/**
 * 获取仪表板配置
 */
export async function getDashboardConfig(): Promise<DashboardConfig> {
  const response = await api.get<DashboardConfigResponse>(
    '/api/dashboard/config'
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || '获取配置失败')
  }
  return response.data.data
}

/**
 * 更新仪表板配置
 */
export async function updateDashboardConfig(
  updates: DashboardConfigUpdate
): Promise<DashboardConfig> {
  const response = await api.put<DashboardConfigResponse>(
    '/api/dashboard/config',
    updates
  )
  if (!response.data.success) {
    throw new Error(response.data.message || '更新配置失败')
  }
  return response.data.data!
}

/**
 * 重置仪表板配置为默认值
 */
export async function resetDashboardConfig(): Promise<DashboardConfig> {
  const response = await api.post<DashboardConfigResponse>(
    '/api/dashboard/config/reset'
  )
  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.message || '重置配置失败')
  }
  return response.data.data
}
