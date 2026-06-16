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
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import type { UsageLogFieldKey } from '../lib/field-visibility'

interface UsageLogFieldsVisibleResponse {
  success: boolean
  message: string
  data: {
    enabled: boolean
    fields: string[]
  }
}

async function fetchUsageLogFieldsVisible() {
  const res = await api.get<UsageLogFieldsVisibleResponse>(
    '/api/user/self/usage_log_fields'
  )
  // Fail closed: 后端返回 success=false 时视为不可访问
  if (!res.data.success || !res.data.data) {
    throw new Error(res.data.message || 'Failed to load field visibility')
  }
  return res.data.data
}

// 返回当前用户角色下的使用日志详情弹窗字段可见性状态。
// - detailsEnabled: 总开关，false 时前端隐藏详情按钮
// - isVisible(field): 查询指定字段是否可见
// - isLoading: 加载中时 fail closed（detailsEnabled=false, isVisible=false）
export function useUsageLogFieldVisibility() {
  const userId = useAuthStore((s) => s.auth.user?.id)

  const { data, isLoading, isError } = useQuery({
    // query key 包含 userId，登出登入后缓存自动失效
    queryKey: ['usage-log-fields-visible', userId],
    queryFn: fetchUsageLogFieldsVisible,
    enabled: !!userId,
    staleTime: 60_000,
  })

  // 加载中或出错时 fail closed：不显示详情，不显示任何字段
  const failClosed = isLoading || isError || !data

  const visibleSet = new Set(data?.fields ?? [])

  return {
    detailsEnabled: failClosed ? false : data.enabled,
    isVisible: (field: UsageLogFieldKey) =>
      failClosed ? false : visibleSet.has(field),
    isLoading,
  }
}
