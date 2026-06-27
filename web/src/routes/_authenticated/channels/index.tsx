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
import z from 'zod'
import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { Channels } from '@/features/channels'

// URL search params 经 preload/navigation 传入时可能是字符串、单值或数组，
// 使用 coerce/preprocess 做宽容归一化，避免校验抛错引发 TanStack Router 内部异常。
const stringArraySearchParam = z.preprocess((value) => {
  if (Array.isArray(value)) return value
  if (typeof value === 'string' && value !== '') return [value]
  return []
}, z.array(z.string()))

const channelsSearchSchema = z.object({
  page: z.coerce.number().int().positive().optional().catch(1),
  pageSize: z.coerce.number().int().positive().optional().catch(undefined),
  filter: z.string().optional().catch(''),
  status: stringArraySearchParam.optional().catch([]),
  type: stringArraySearchParam.optional().catch([]),
  group: stringArraySearchParam.optional().catch([]),
  model: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/channels/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()

    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({
        to: '/403',
      })
    }
  },
  validateSearch: channelsSearchSchema,
  component: Channels,
})
