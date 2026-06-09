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
import { useState, useCallback } from 'react'
import type { TFunction } from 'i18next'
import { Check, Copy, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { copyToClipboard } from '@/lib/copy-to-clipboard'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Progress } from '@/components/ui/progress'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { type ApiKey } from '../types'
import { useApiKeys } from './api-keys-provider'

const MASKED_API_KEY = `sk-${'*'.repeat(12)}`

type QuotaLimit = {
  expired: boolean
  remaining: number
  total: number
  used: number
}

type QuotaDetailLine = {
  labelKey: string
  value: string
}

type QuotaUsage = {
  detailLines: QuotaDetailLine[]
  primaryLabelKey: string
  quotaType: number
  quotaTypeLabelKey: string
  quotaTypeVariant: StatusVariant
  remaining: number
  total: number
  used: number
}

function getQuotaProgressColor(percentage: number): string {
  if (percentage <= 10) return '[&_[data-slot=progress-indicator]]:bg-rose-500'
  if (percentage <= 30) return '[&_[data-slot=progress-indicator]]:bg-amber-500'
  return '[&_[data-slot=progress-indicator]]:bg-emerald-500'
}

function getEffectiveQuotaLimit(
  total: number,
  used: number,
  startTime: number,
  durationSeconds: number,
  nowSeconds: number
): QuotaLimit {
  const expired = startTime > 0 && nowSeconds >= startTime + durationSeconds
  const effectiveUsed = expired ? 0 : used

  return {
    expired,
    remaining: Math.max(total - effectiveUsed, 0),
    total,
    used: effectiveUsed,
  }
}

function resolveQuotaType(apiKey: ApiKey): number {
  if (apiKey.quota_type === 0 && !apiKey.unlimited_quota) return 1
  return apiKey.quota_type ?? (apiKey.unlimited_quota ? 0 : 1)
}

function getQuotaTypeMeta(quotaType: number): {
  labelKey: string
  variant: StatusVariant
} {
  if (quotaType === 0) {
    return { labelKey: 'Unlimited Quota', variant: 'neutral' }
  }
  if (quotaType === 2) {
    return { labelKey: 'Hourly Reset Quota', variant: 'info' }
  }
  if (quotaType === 3) {
    return { labelKey: 'Hourly + Days Reset Quota', variant: 'purple' }
  }
  return { labelKey: 'Permanent Quota', variant: 'blue' }
}

function buildQuotaUsage(apiKey: ApiKey): QuotaUsage {
  const quotaType = resolveQuotaType(apiKey)
  const quotaTypeMeta = getQuotaTypeMeta(quotaType)

  if (quotaType === 2 || quotaType === 3) {
    const nowSeconds = Math.floor(Date.now() / 1000)
    const windowHours = apiKey.window_hours || 1
    const windowLimit = getEffectiveQuotaLimit(
      apiKey.window_quota || 0,
      apiKey.window_used_quota || 0,
      apiKey.window_start_time || 0,
      windowHours * 3600,
      nowSeconds
    )

    if (quotaType === 2) {
      return {
        detailLines: [
          {
            labelKey: 'Window Quota',
            value: `${formatQuota(windowLimit.remaining)} / ${formatQuota(windowLimit.total)}`,
          },
          {
            labelKey: 'Reset window',
            value: `Every ${windowHours}h`,
          },
        ],
        primaryLabelKey: 'Window Quota',
        quotaType,
        quotaTypeLabelKey: quotaTypeMeta.labelKey,
        quotaTypeVariant: quotaTypeMeta.variant,
        remaining: windowLimit.remaining,
        total: windowLimit.total,
        used: windowLimit.used,
      }
    }

    const cycleDays = apiKey.cycle_days || 1
    const cycleLimit = getEffectiveQuotaLimit(
      apiKey.cycle_quota || 0,
      apiKey.cycle_used_quota || 0,
      apiKey.cycle_start_time || 0,
      cycleDays * 86400,
      nowSeconds
    )
    const primaryLimit =
      windowLimit.remaining <= cycleLimit.remaining ? windowLimit : cycleLimit
    const primaryLabelKey =
      windowLimit.remaining <= cycleLimit.remaining
        ? 'Window Quota'
        : 'Cycle Quota'

    return {
      detailLines: [
        {
          labelKey: 'Window Quota',
          value: `${formatQuota(windowLimit.remaining)} / ${formatQuota(windowLimit.total)}`,
        },
        {
          labelKey: 'Reset window',
          value: `Every ${windowHours}h`,
        },
        {
          labelKey: 'Cycle Quota',
          value: `${formatQuota(cycleLimit.remaining)} / ${formatQuota(cycleLimit.total)}`,
        },
        {
          labelKey: 'Reset Period',
          value: `Every ${cycleDays}d`,
        },
      ],
      primaryLabelKey,
      quotaType,
      quotaTypeLabelKey: quotaTypeMeta.labelKey,
      quotaTypeVariant: quotaTypeMeta.variant,
      remaining: primaryLimit.remaining,
      total: primaryLimit.total,
      used: primaryLimit.used,
    }
  }

  const used = apiKey.used_quota || 0
  const remaining = apiKey.remain_quota || 0

  return {
    detailLines: [],
    primaryLabelKey: 'Permanent quota',
    quotaType,
    quotaTypeLabelKey: quotaTypeMeta.labelKey,
    quotaTypeVariant: quotaTypeMeta.variant,
    remaining,
    total: used + remaining,
    used,
  }
}

function formatQuotaScheduleValue(value: string, t: TFunction) {
  const hourMatch = value.match(/^Every (\d+)h$/)
  if (hourMatch) return t('Every {{count}}h', { count: Number(hourMatch[1]) })

  const dayMatch = value.match(/^Every (\d+)d$/)
  if (dayMatch) return t('Every {{count}}d', { count: Number(dayMatch[1]) })

  return value
}

export function ApiKeyCell({ apiKey }: { apiKey: ApiKey }) {
  const { t } = useTranslation()
  const {
    resolveRealKey,
    resolvedKeys,
    loadingKeys,
    copiedKeyId,
    markKeyCopied,
  } = useApiKeys()
  const [popoverOpen, setPopoverOpen] = useState(false)

  const isLoading = !!loadingKeys[apiKey.id]
  const resolvedFullKey = resolvedKeys[apiKey.id]
  const isCopied = copiedKeyId === apiKey.id

  const handlePopoverOpen = useCallback(
    (open: boolean) => {
      setPopoverOpen(open)
      if (open && !resolvedFullKey) {
        resolveRealKey(apiKey.id)
      }
    },
    [resolvedFullKey, resolveRealKey, apiKey.id]
  )

  const handleCopy = useCallback(async () => {
    const realKey = resolvedFullKey
    if (!realKey) {
      void resolveRealKey(apiKey.id)
      toast.info(t('API key is loading, please try again in a moment'))
      return
    }
    if (realKey) {
      const ok = await copyToClipboard(realKey)
      if (ok) markKeyCopied(apiKey.id)
    }
  }, [resolvedFullKey, resolveRealKey, apiKey.id, markKeyCopied, t])

  return (
    <div className='flex items-center'>
      <Popover open={popoverOpen} onOpenChange={handlePopoverOpen}>
        <PopoverTrigger
          render={
            <Button
              variant='ghost'
              size='sm'
              className='text-muted-foreground h-7 font-mono text-xs'
            />
          }
        >
          {MASKED_API_KEY}
        </PopoverTrigger>
        <PopoverContent
          className='w-auto max-w-[min(90vw,28rem)]'
          align='start'
        >
          <div className='space-y-2'>
            <p className='text-muted-foreground text-xs'>{t('Full API Key')}</p>
            {isLoading ? (
              <div className='flex items-center gap-2 py-2'>
                <Loader2 className='size-3.5 animate-spin' />
                <span className='text-muted-foreground text-xs'>
                  {t('Loading...')}
                </span>
              </div>
            ) : (
              <input
                readOnly
                value={resolvedFullKey || MASKED_API_KEY}
                autoFocus
                onFocus={(e) => e.target.select()}
                className='bg-muted/50 w-full min-w-[280px] rounded-md border px-3 py-2 font-mono text-xs outline-none'
              />
            )}
          </div>
        </PopoverContent>
      </Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon'
              className='size-7 shrink-0'
              onClick={handleCopy}
              onFocus={() => {
                if (!resolvedFullKey) void resolveRealKey(apiKey.id)
              }}
              onPointerEnter={() => {
                if (!resolvedFullKey) void resolveRealKey(apiKey.id)
              }}
              disabled={isLoading}
            />
          }
        >
          {isLoading ? (
            <Loader2 className='size-3.5 animate-spin' />
          ) : isCopied ? (
            <Check className='size-3.5 text-green-600' />
          ) : (
            <Copy className='size-3.5' />
          )}
        </TooltipTrigger>
        <TooltipContent>
          {isLoading
            ? t('Loading...')
            : isCopied
              ? t('Copied!')
              : t('Copy API key')}
        </TooltipContent>
      </Tooltip>
    </div>
  )
}

export function ApiKeyQuotaCell({
  apiKey,
  className,
}: {
  apiKey: ApiKey
  className?: string
}) {
  const { t } = useTranslation()
  const usage = buildQuotaUsage(apiKey)
  const percentage = usage.total > 0 ? (usage.remaining / usage.total) * 100 : 0

  const isUnlimited = usage.quotaType === 0
  const displayValue = isUnlimited ? usage.used : usage.total
  const progressValue = isUnlimited ? 100 : percentage

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div
            className={cn(
              'w-[170px] cursor-default space-y-1 rounded-md p-1',
              className
            )}
          />
        }
      >
        <div className='flex items-center justify-between gap-2'>
          <span className='truncate text-xs font-medium tabular-nums'>
            {formatQuota(displayValue)}
          </span>
          <StatusBadge
            label={t(usage.quotaTypeLabelKey)}
            variant={usage.quotaTypeVariant}
            copyable={false}
            className='max-w-24'
          />
        </div>
        <Progress
          value={progressValue}
          className={cn('h-1.5', getQuotaProgressColor(progressValue))}
        />
      </TooltipTrigger>
      <TooltipContent side='top' className='max-w-xs'>
        <div className='space-y-1 text-xs'>
          <StatusBadge
            label={t(usage.quotaTypeLabelKey)}
            variant={usage.quotaTypeVariant}
            copyable={false}
          />
          {isUnlimited ? (
            <>
              <div className='text-muted-foreground'>
                {t('No quota cap; usage still depends on account balance.')}
              </div>
              <div>
                {t('labelWithColon', { label: t('Used') })} {formatQuota(usage.used)}
              </div>
            </>
          ) : (
            <>
              <div>
                {t('labelWithColon', { label: t('Used') })} {formatQuota(usage.used)}
              </div>
              <div>
                {t('labelWithColon', { label: t('Remaining') })} {formatQuota(usage.remaining)} (
                {percentage.toFixed(1)}%)
              </div>
              <div>
                {t('labelWithColon', { label: t('Total') })} {formatQuota(usage.total)}
              </div>
              <div>
                {t('labelWithColon', { label: t('Reset') })}{' '}
                {usage.detailLines.length > 0
                  ? usage.detailLines
                      .map((line) => formatQuotaScheduleValue(line.value, t))
                      .join(', ')
                  : t('No Reset')}
              </div>
            </>
          )}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

export function ModelLimitsCell({ apiKey }: { apiKey: ApiKey }) {
  const { t } = useTranslation()

  if (!apiKey.model_limits_enabled || !apiKey.model_limits) {
    return (
      <StatusBadge label={t('Unlimited')} variant='neutral' copyable={false} />
    )
  }

  const models = apiKey.model_limits.split(',').filter(Boolean)

  return (
    <Tooltip>
      <TooltipTrigger render={<span />}>
        <StatusBadge
          label={t('{{count}} models', { count: models.length })}
          variant='neutral'
          copyable={false}
        />
      </TooltipTrigger>
      <TooltipContent side='top' className='max-w-xs'>
        <div className='max-h-[200px] space-y-0.5 overflow-y-auto text-xs'>
          {models.map((m) => (
            <div key={m} className='font-mono'>
              {m}
            </div>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

export function IpRestrictionsCell({ apiKey }: { apiKey: ApiKey }) {
  const { t } = useTranslation()
  const allowIps = apiKey.allow_ips?.trim()

  if (!allowIps) {
    return (
      <StatusBadge
        label={t('No restriction')}
        variant='neutral'
        copyable={false}
      />
    )
  }

  const ips = allowIps
    .split('\n')
    .map((ip) => ip.trim())
    .filter(Boolean)

  return (
    <Tooltip>
      <TooltipTrigger render={<span />}>
        <StatusBadge
          label={t('{{count}} IP(s)', { count: ips.length })}
          variant='neutral'
          copyable={false}
        />
      </TooltipTrigger>
      <TooltipContent side='top' className='max-w-xs'>
        <div className='max-h-[200px] space-y-0.5 overflow-y-auto text-xs'>
          {ips.map((ip) => (
            <div key={ip} className='font-mono'>
              {ip}
            </div>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}
