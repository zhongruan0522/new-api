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
  ApiResponse,
  DynamicRatioRule,
  DynamicRatioRulePayload,
  DynamicRatioStatus,
} from './types'

function unwrap<T>(res: ApiResponse<T>, fallbackMessage: string): T {
  if (!res.success) {
    throw new Error(res.message || fallbackMessage)
  }
  if (res.data === undefined) {
    throw new Error(fallbackMessage)
  }
  return res.data
}

export async function getDynamicRatioRules(): Promise<DynamicRatioRule[]> {
  const res = await api.get('/api/dynamic_ratio/rules')
  return unwrap<DynamicRatioRule[]>(
    res.data,
    'Failed to load dynamic ratio rules'
  )
}

export async function getDynamicRatioStatus(): Promise<DynamicRatioStatus> {
  const res = await api.get('/api/dynamic_ratio/status')
  return unwrap<DynamicRatioStatus>(
    res.data,
    'Failed to load dynamic ratio status'
  )
}

export async function setDynamicRatioEnabled(
  enabled: boolean
): Promise<void> {
  const res = await api.put('/api/dynamic_ratio/enabled', { enabled })
  unwrap(res.data, 'Failed to update dynamic ratio status')
}

export async function createDynamicRatioRule(
  payload: DynamicRatioRulePayload
): Promise<DynamicRatioRule> {
  const res = await api.post('/api/dynamic_ratio/rules', payload)
  return unwrap<DynamicRatioRule>(
    res.data,
    'Failed to create dynamic ratio rule'
  )
}

export async function updateDynamicRatioRule(
  payload: DynamicRatioRulePayload & { id: number }
): Promise<DynamicRatioRule> {
  const res = await api.put('/api/dynamic_ratio/rules', payload)
  return unwrap<DynamicRatioRule>(
    res.data,
    'Failed to update dynamic ratio rule'
  )
}

export async function deleteDynamicRatioRule(id: number): Promise<void> {
  const res = await api.delete(`/api/dynamic_ratio/rules/${id}`)
  unwrap(res.data, 'Failed to delete dynamic ratio rule')
}

export async function reorderDynamicRatioRules(ids: number[]): Promise<void> {
  const res = await api.put('/api/dynamic_ratio/rules/reorder', { ids })
  unwrap(res.data, 'Failed to reorder dynamic ratio rules')
}
