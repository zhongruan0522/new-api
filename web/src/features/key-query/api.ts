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

import { formatDateStr } from '@/lib/format'
import type {
  BillingError,
  KeyQueryReport,
  TokenLogResponse,
  TokenSubscription,
  TokenUsage,
} from './types'

function getTokenHeaders(key: string) {
  return {
    Authorization: `Bearer ${key}`,
  }
}

async function fetchJson<T>(url: string, key: string): Promise<T> {
  const res = await fetch(url, {
    headers: getTokenHeaders(key),
  })
  if (!res.ok) {
    throw new Error('The key is invalid or expired')
  }

  const data = (await res.json()) as T & BillingError
  if (data.error) {
    throw new Error(data.error.message || 'Key query failed')
  }
  return data
}

export async function fetchKeyQueryReport(
  rawKey: string
): Promise<KeyQueryReport> {
  const key = rawKey.trim()
  if (!/^sk-[a-zA-Z0-9]{48}$/.test(key)) {
    throw new Error('Invalid key format')
  }

  const now = new Date()
  const start = new Date(now.getTime() - 100 * 24 * 60 * 60 * 1000)
  const startDate = formatDateStr(start)
  const endDate = formatDateStr(now)

  const subscription = await fetchJson<TokenSubscription>(
    '/v1/dashboard/billing/subscription',
    key
  )
  const usage = await fetchJson<TokenUsage>(
    `/v1/dashboard/billing/usage?start_date=${startDate}&end_date=${endDate}`,
    key
  )
  const logRes = await fetchJson<TokenLogResponse>('/api/log/token', key)

  if (!logRes.success) {
    throw new Error(logRes.message || 'Failed to load token logs')
  }

  return {
    subscription,
    usage,
    logs: logRes.data ?? [],
  }
}
