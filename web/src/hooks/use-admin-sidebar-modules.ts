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

You should have received a copy of the GNU Affero General License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { fetchAdminSidebarModules } from '@/lib/nav-modules'

/**
 * Fetch the admin sidebar modules config from the admin-only endpoint.
 *
 * The config is intentionally NOT part of the public `/api/status` payload:
 * it describes the admin console structure and must only be fetched after
 * the current user is confirmed to be an admin. Non-admin callers get a
 * 401 from the backend. On success the raw config is also cached under a
 * dedicated storage key so non-React route guards (`isSidebarModuleEnabled`)
 * can read it without hitting the public status cache.
 */
export function useAdminSidebarModules(enabled: boolean) {
  return useQuery({
    queryKey: ['admin-modules'],
    queryFn: async () => {
      const raw = await fetchAdminSidebarModules()
      return raw
    },
    enabled,
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    retry: false,
  })
}
