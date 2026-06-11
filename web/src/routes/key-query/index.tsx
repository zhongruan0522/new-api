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

import { createFileRoute, redirect } from '@tanstack/react-router'
import { getStatus } from '@/lib/api'
import { parseHeaderNavModules } from '@/lib/nav-modules'
import { KeyQuery } from '@/features/key-query'

export const Route = createFileRoute('/key-query/')({
  beforeLoad: async () => {
    const status = (await getStatus()) as Record<string, unknown> | null
    const modules = parseHeaderNavModules(status?.HeaderNavModules)
    if (modules.key_query === false) {
      throw redirect({ to: '/' })
    }
  },
  component: KeyQuery,
})
