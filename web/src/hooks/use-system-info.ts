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
import { getSystemInfo } from '@/lib/api'
import { useIsAdmin } from './use-admin'

/**
 * Fetch the admin-only build version (`/api/status/system_info`).
 *
 * The query is enabled only for admins: the endpoint is guarded by
 * AdminAuth, so anonymous/common users must not fire a doomed 403 request.
 * Non-admins see `undefined` and should fall back to the unknown-version
 * placeholder.
 */
export function useSystemInfo() {
  const isAdmin = useIsAdmin()
  const { data, isLoading } = useQuery({
    queryKey: ['status', 'system_info'],
    queryFn: getSystemInfo,
    enabled: isAdmin,
    // Build version only changes on redeploy; keep it cached for 30 minutes
    staleTime: 30 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  })

  return {
    version: isAdmin ? data?.data?.version : undefined,
    loading: isAdmin && isLoading,
  }
}
