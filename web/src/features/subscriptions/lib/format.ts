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
import type { TFunction } from 'i18next'
import dayjs from '@/lib/dayjs'
import type { SubscriptionPlan } from '../types'

export function formatDuration(
  plan: Partial<SubscriptionPlan>,
  t: TFunction
): string {
  const unit = plan?.duration_unit || 'month'
  const value = plan?.duration_value || 1
  const unitLabels: Record<string, string> = {
    year: t('subscriptions.fields.years'),
    month: t('subscriptions.fields.months'),
    day: t('dashboard.fields.days'),
    hour: t('channels.fields.hours'),
    custom: t('subscriptions.fields.customSeconds'),
  }
  if (unit === 'custom') {
    const seconds = plan?.custom_seconds || 0
    if (seconds >= 86400)
      return `${Math.floor(seconds / 86400)} ${t('dashboard.fields.days')}`
    if (seconds >= 3600)
      return `${Math.floor(seconds / 3600)} ${t('channels.fields.hours')}`
    return `${seconds} ${t('subscriptions.fields.seconds')}`
  }
  return `${value} ${unitLabels[unit] || unit}`
}

export function formatResetPeriod(
  plan: Partial<SubscriptionPlan>,
  t: TFunction
): string {
  const period = plan?.quota_reset_period || 'never'
  if (period === 'daily') return t('dynamicRatio.fields.daily')
  if (period === 'weekly') return t('subscriptions.fields.weekly')
  if (period === 'monthly') return t('subscriptions.fields.monthly')
  if (period === 'custom') {
    const seconds = Number(plan?.quota_reset_custom_seconds || 0)
    if (seconds >= 86400)
      return `${Math.floor(seconds / 86400)} ${t('dashboard.fields.days')}`
    if (seconds >= 3600)
      return `${Math.floor(seconds / 3600)} ${t('channels.fields.hours')}`
    if (seconds >= 60)
      return `${Math.floor(seconds / 60)} ${t('subscriptions.fields.minutes')}`
    return `${seconds} ${t('subscriptions.fields.seconds')}`
  }
  return t('keys.fields.noReset')
}

export function formatTimestamp(ts: number): string {
  if (!ts) return '-'
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}
