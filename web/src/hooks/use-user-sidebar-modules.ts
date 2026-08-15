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
import { fetchUserSidebarModules } from '@/lib/nav-modules'

/**
 * Fetch the user-visible sidebar modules config from the UserAuth endpoint.
 *
 * 管理员在「侧栏模块」中关闭面向用户的模块（chat.playground、使用日志、
 * 充值、工单等）时，普通用户的侧栏与路由守卫必须同样受控。该配置由
 * `/api/status/user_modules`（UserAuth）下发：服务端剥离 admin 段，仅含
 * 面向用户的开关，登录即可拉取。原始配置缓存在独立 storage key 下，
 * 供非 React 路由守卫（isSidebarModuleEnabled）读取。
 */
export function useUserSidebarModules(enabled: boolean) {
  return useQuery({
    queryKey: ['user-modules'],
    queryFn: async () => {
      const raw = await fetchUserSidebarModules()
      return raw
    },
    enabled,
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    retry: false,
  })
}
