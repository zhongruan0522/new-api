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
import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  formatQuota,
  formatTimestampToDate,
  quotaUnitsToDollars,
} from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { KeyUsageData } from './types'

// quota_type: 0=无限, 1=永久限额, 2=时段限额(按小时), 3=时段+周期(按小时/天混合)
const QUOTA_TYPE_UNLIMITED = 0
const QUOTA_TYPE_PERMANENT = 1
const QUOTA_TYPE_HOURLY = 2
const QUOTA_TYPE_HYBRID = 3

interface KeyInfoCardProps {
  usage: KeyUsageData
  onCopySummary: () => void
}

function StatBlock(props: { label: string; value: string; muted?: boolean }) {
  return (
    <div className='min-w-0 rounded-lg border p-3'>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className={props.muted ? 'text-muted-foreground mt-1' : 'mt-1'}>
        {props.value}
      </div>
    </div>
  )
}

/**
 * 将 quota 原始单位转换为展示货币字符串。
 * 例如 window_quota=500000（500000 tokens = $1）-> "$1" / "¥7"。
 */
function formatQuotaAsCurrency(quota: number): string {
  if (!Number.isFinite(quota) || quota <= 0) return '$0'
  return `$${quotaUnitsToDollars(quota).toFixed(2)}`
}

function resolveEffectiveQuotaType(usage: KeyUsageData): number {
  if (usage.quota_type === 0 && !usage.unlimited_quota) {
    return QUOTA_TYPE_PERMANENT
  }
  return usage.quota_type
}

export function KeyInfoCard({ usage, onCopySummary }: KeyInfoCardProps) {
  const { t } = useTranslation()
  const quotaType = resolveEffectiveQuotaType(usage)
  const isUnlimited = quotaType === QUOTA_TYPE_UNLIMITED
  const expiredAt = usage.expires_at
  const createdTimeText = formatTimestampToDate(usage.created_time)
  const accessedTimeText = formatTimestampToDate(usage.accessed_time)
  const expiresText =
    expiredAt === 0
      ? t('keyQuery.fields.never')
      : formatTimestampToDate(expiredAt)

  // 第一行：总额度 + 已用 + 剩余 + 过期时间
  let totalQuotaText: string
  if (isUnlimited) {
    totalQuotaText = t('keyQuery.fields.unlimited')
  } else if (quotaType === QUOTA_TYPE_HOURLY) {
    totalQuotaText = `${formatQuotaAsCurrency(usage.window_quota)} / ${usage.window_hours}${t('keyQuery.fields.hourUnit')}`
  } else if (quotaType === QUOTA_TYPE_HYBRID) {
    const cycleAmount = formatQuotaAsCurrency(usage.cycle_quota)
    const windowAmount = formatQuotaAsCurrency(usage.window_quota)
    totalQuotaText = `${cycleAmount} (${windowAmount} / ${usage.window_hours}${t('keyQuery.fields.hourUnit')})`
  } else {
    totalQuotaText = formatQuota(usage.total_granted)
  }

  const usedText = isUnlimited
    ? t('keyQuery.fields.unlimited')
    : formatQuota(usage.total_used)
  const remainingText = isUnlimited
    ? t('keyQuery.fields.unlimited')
    : formatQuota(usage.total_available)

  const firstRow = (
    <>
      <StatBlock
        label={t('dashboard.fields.totalQuota')}
        value={totalQuotaText}
      />
      <StatBlock label={t('keyQuery.status.usedQuota')} value={usedText} />
      <StatBlock
        label={t('keyQuery.fields.remainingQuota')}
        value={remainingText}
      />
      <StatBlock
        label={t('keyQuery.fields.expirationTime')}
        value={expiresText}
      />
    </>
  )

  // 第二/第三行按 quota_type 差异化渲染
  let secondRow: React.ReactNode = null
  let thirdRow: React.ReactNode = null

  if (quotaType === QUOTA_TYPE_PERMANENT || isUnlimited) {
    // 永久限额 / 无限额度：第二行 = 创建时间 + 最后使用时间
    secondRow = (
      <>
        <StatBlock label={t('keys.status.lastUsed')} value={accessedTimeText} />
        <StatBlock
          label={t('common.fields.createdAt')}
          value={createdTimeText}
        />
      </>
    )
  } else if (quotaType === QUOTA_TYPE_HOURLY) {
    // 按小时重置：第二行 = 重置周期 + 开始小时 + 下次重置时间
    const nextResetText = usage.window_next_reset_time
      ? formatTimestampToDate(usage.window_next_reset_time)
      : '-'
    secondRow = (
      <>
        <StatBlock
          label={t('keyQuery.fields.resetCycle')}
          value={t('keyQuery.fields.everyCountHours', {
            count: usage.window_hours,
          })}
        />
        <StatBlock
          label={t('keys.actions.startHour')}
          value={`${usage.window_start_hour}:00`}
        />
        <StatBlock
          label={t('keyQuery.fields.nextResetTime')}
          value={nextResetText}
        />
      </>
    )
    thirdRow = (
      <>
        <StatBlock label={t('keys.status.lastUsed')} value={accessedTimeText} />
        <StatBlock
          label={t('common.fields.createdAt')}
          value={createdTimeText}
        />
      </>
    )
  } else if (quotaType === QUOTA_TYPE_HYBRID) {
    // 按小时/天混合重置：第二行 = 重置周期(天) + 重置周期(小时) + 下次重置时间
    const nextResetText = usage.cycle_next_reset_time
      ? formatTimestampToDate(usage.cycle_next_reset_time)
      : '-'
    secondRow = (
      <>
        <StatBlock
          label={t('keyQuery.fields.resetCycleDays')}
          value={t('keyQuery.fields.everyCountDays', {
            count: usage.cycle_days,
          })}
        />
        <StatBlock
          label={t('keyQuery.fields.resetCycleHours')}
          value={t('keyQuery.fields.everyCountHours', {
            count: usage.window_hours,
          })}
        />
        <StatBlock
          label={t('keyQuery.fields.nextResetTime')}
          value={nextResetText}
        />
      </>
    )
    thirdRow = (
      <>
        <StatBlock label={t('keys.status.lastUsed')} value={accessedTimeText} />
        <StatBlock
          label={t('common.fields.createdAt')}
          value={createdTimeText}
        />
      </>
    )
  }

  const copySummary = async () => {
    onCopySummary()
  }

  return (
    <Card size='sm'>
      <CardHeader className='border-b'>
        <div className='flex items-center justify-between gap-3'>
          <CardTitle>{t('keyQuery.titles.information')}</CardTitle>
          <Button variant='outline' onClick={copySummary}>
            <Copy />
            {t('channels.actions.copy')}
          </Button>
        </div>
      </CardHeader>
      <CardContent className='space-y-3'>
        <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
          {firstRow}
        </div>
        {secondRow && (
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            {secondRow}
          </div>
        )}
        {thirdRow && (
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            {thirdRow}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
