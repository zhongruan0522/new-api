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
import {
  Copy,
  Check,
  Route,
  Settings2,
  AlertTriangle,
  Headphones,
  Monitor,
  Cloud,
  Globe,
  ShieldCheck,
  UserCog,
  Info,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'
import { formatLogQuota, formatTokens, formatUseTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { DynamicPricingBreakdown } from '@/features/pricing/components/dynamic-pricing-breakdown'
import type { UsageLog } from '../../data/schema'
import {
  parseLogOther,
  getParamOverrideActionLabel,
  parseAuditLine,
  decodeBillingExprB64,
  getTieredBillingSummary,
  hasAnyCacheTokens,
  isViolationFeeLog,
  getFirstResponseTimeColor,
  getResponseTimeColor,
} from '../../lib/format'
import {
  getLogTypeConfig,
  isPerCallBilling,
  isTimingLogType,
} from '../../lib/utils'
import type { LogOtherData } from '../../types'

function timingTextColorClass(
  variant: 'success' | 'warning' | 'danger'
): string {
  if (variant === 'success') return 'text-emerald-600'
  if (variant === 'warning') return 'text-amber-600'
  return 'text-rose-600'
}

function DetailRow(props: {
  label: React.ReactNode
  value: React.ReactNode
  mono?: boolean
  muted?: boolean
}) {
  return (
    <div className='grid min-w-0 grid-cols-[5.25rem_minmax(0,1fr)] gap-2 text-sm sm:grid-cols-[7rem_minmax(0,1fr)] sm:gap-3'>
      <span className='text-muted-foreground min-w-0 text-xs'>
        {props.label}
      </span>
      <span
        className={cn(
          'max-w-full min-w-0 text-xs break-all sm:break-words',
          props.mono && 'font-mono',
          props.muted && 'text-muted-foreground'
        )}
      >
        {props.value}
      </span>
    </div>
  )
}

function DetailSection(props: {
  icon?: React.ReactNode
  label: string
  variant?: 'default' | 'danger'
  children: React.ReactNode
}) {
  const isDanger = props.variant === 'danger'
  return (
    <div className='min-w-0 space-y-1.5'>
      <Label
        className={cn(
          'flex items-center gap-1.5 text-xs font-semibold',
          isDanger && 'text-red-500'
        )}
      >
        {props.icon}
        {props.label}
      </Label>
      <div
        className={cn(
          'min-w-0 space-y-1 overflow-hidden rounded-md border p-2.5 max-sm:p-2',
          isDanger
            ? 'border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/20'
            : 'bg-muted/30'
        )}
      >
        {props.children}
      </div>
    </div>
  )
}

function compactRatio(ratio: number | undefined): string {
  if (ratio == null || !Number.isFinite(ratio)) return '-'
  return ratio % 1 === 0
    ? String(ratio)
    : ratio.toFixed(4).replace(/\.?0+$/, '')
}

function isValidRatio(ratio: number | undefined): boolean {
  return ratio != null && Number.isFinite(ratio) && ratio !== -1
}

function getEffectiveGroupRatio(other: LogOtherData): {
  labelKey: string
  value: number
} | null {
  if (isValidRatio(other.user_group_ratio)) {
    return { labelKey: 'User Exclusive Ratio', value: other.user_group_ratio! }
  }
  if (isValidRatio(other.group_ratio)) {
    return { labelKey: 'Group Ratio', value: other.group_ratio! }
  }
  return null
}

function getDynamicRatio(other: LogOtherData): number {
  if (
    other.dynamic_ratio != null &&
    Number.isFinite(other.dynamic_ratio) &&
    other.dynamic_ratio > 0
  ) {
    return other.dynamic_ratio
  }
  return 1
}

function getCacheCreationTotal(other: LogOtherData): number {
  const splitTotal =
    (other.cache_creation_tokens_5m || 0) +
    (other.cache_creation_tokens_1h || 0)
  if (splitTotal > 0) return splitTotal
  return other.cache_creation_tokens || 0
}

function getAudioInputTokens(other: LogOtherData): number {
  if (other.audio_input != null) return other.audio_input
  if (other.audio_input_token_count != null)
    return other.audio_input_token_count
  return 0
}

function getOrdinaryInputTokens(log: UsageLog, other: LogOtherData): number {
  if ((other.audio || other.ws) && other.text_input != null) {
    return Math.max(other.text_input, 0)
  }

  const cacheRead = other.cache_tokens || 0
  const cacheCreation = getCacheCreationTotal(other)
  const audioInput = other.audio_input_seperate_price
    ? other.audio_input_token_count || 0
    : 0
  const input =
    (log.prompt_tokens || 0) - cacheRead - cacheCreation - audioInput
  return Math.max(input, 0)
}

function formatExactTokens(tokens: number): string {
  return Number.isFinite(tokens) ? Math.max(tokens, 0).toLocaleString() : '-'
}

function getPriceFormatter() {
  const priceOpts = { digitsLarge: 4, digitsSmall: 6, abbreviate: false }
  return (usd: number | null | undefined) =>
    formatBillingCurrencyFromUSD(usd, priceOpts)
}

function formatTokenRange(min?: number, max?: number): string {
  const formatBound = (value?: number) => {
    if (value == null || !Number.isFinite(value)) return '-'
    if (value >= 1000000 && value % 1000000 === 0) {
      return `${value / 1000000}M`
    }
    if (value >= 1000 && value % 1000 === 0) {
      return `${value / 1000}K`
    }
    return value.toLocaleString()
  }

  const minValue = min || 0
  if (max == null) return `>=${formatBound(minValue)}`
  if (minValue <= 0) return `<${formatBound(max)}`
  return `${formatBound(minValue)}-${formatBound(max)}`
}

function getContextPricingRange(other: LogOtherData): string {
  const contextPricing = other.context_pricing
  return formatTokenRange(
    other.context_pricing_tier_min_tokens ?? contextPricing?.min_tokens,
    other.context_pricing_tier_max_tokens ?? contextPricing?.max_tokens
  )
}

function getContextPricingPrices(other: LogOtherData) {
  return other.context_pricing_prices || other.context_pricing?.prices || null
}

type BillingRow = {
  labelKey: string
  quantity: string
  unitPrice: string
  ratios: string
  subtotal: string
}

function pushTokenBillingRow(args: {
  rows: BillingRow[]
  labelKey: string
  tokens: number
  unitPriceUSD: number | undefined
  groupRatio: number
  dynamicRatio: number
  ratioText: string
  formatPrice: (usd: number | null | undefined) => string
}) {
  if (args.tokens <= 0) return
  if (args.unitPriceUSD == null || !Number.isFinite(args.unitPriceUSD)) return

  const subtotalUSD =
    (args.tokens / 1000000) *
    args.unitPriceUSD *
    args.groupRatio *
    args.dynamicRatio
  args.rows.push({
    labelKey: args.labelKey,
    quantity: formatExactTokens(args.tokens),
    unitPrice: `${args.formatPrice(args.unitPriceUSD)}/M`,
    ratios: args.ratioText,
    subtotal: args.formatPrice(subtotalUSD),
  })
}

function pushMeteredBillingRow(args: {
  rows: BillingRow[]
  labelKey: string
  quantity: number
  unitPriceUSD: number | undefined
  unitLabel: string
  divisor: number
  groupRatio: number
  dynamicRatio: number
  ratioText: string
  formatPrice: (usd: number | null | undefined) => string
}) {
  if (args.quantity <= 0) return
  if (args.unitPriceUSD == null || !Number.isFinite(args.unitPriceUSD)) return

  const subtotalUSD =
    (args.quantity / args.divisor) *
    args.unitPriceUSD *
    args.groupRatio *
    args.dynamicRatio
  args.rows.push({
    labelKey: args.labelKey,
    quantity: args.quantity.toLocaleString(),
    unitPrice: `${args.formatPrice(args.unitPriceUSD)}/${args.unitLabel}`,
    ratios: args.ratioText,
    subtotal: args.formatPrice(subtotalUSD),
  })
}

function buildBillingRows(
  log: UsageLog,
  other: LogOtherData,
  t: (key: string) => string
): BillingRow[] {
  const rows: BillingRow[] = []
  const formatPrice = getPriceFormatter()
  const contextPrices = getContextPricingPrices(other)
  const modelRatio = contextPrices?.model_ratio ?? other.model_ratio
  const completionRatio =
    contextPrices?.completion_ratio ?? other.completion_ratio
  const baseInputUSD =
    modelRatio != null && Number.isFinite(modelRatio) ? modelRatio * 2 : 0
  const effectiveGroupRatio = getEffectiveGroupRatio(other)
  const groupRatio = effectiveGroupRatio?.value ?? 1
  const dynamicRatio = getDynamicRatio(other)
  const ratioLabel = effectiveGroupRatio?.labelKey || 'Group Ratio'
  const ratioParts = [`${t(ratioLabel)} ${compactRatio(groupRatio)}x`]
  if (dynamicRatio !== 1) {
    ratioParts.push(`${t('Dynamic Ratio')} ${compactRatio(dynamicRatio)}x`)
  }
  const ratioText = ratioParts.join(' * ')

  if (isPerCallBilling(other.model_price)) {
    const subtotalUSD = other.model_price! * groupRatio * dynamicRatio
    rows.push({
      labelKey: 'Model Price',
      quantity: '1',
      unitPrice: formatPrice(other.model_price),
      ratios: ratioText,
      subtotal: formatPrice(subtotalUSD),
    })
    return rows
  }

  pushTokenBillingRow({
    rows,
    labelKey: 'Input Tokens',
    tokens: getOrdinaryInputTokens(log, other),
    unitPriceUSD: baseInputUSD,
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushTokenBillingRow({
    rows,
    labelKey: 'Output Tokens',
    tokens: log.completion_tokens || 0,
    unitPriceUSD: baseInputUSD * (completionRatio ?? 0),
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushTokenBillingRow({
    rows,
    labelKey: 'Cache Read',
    tokens: other.cache_tokens || 0,
    unitPriceUSD:
      baseInputUSD * (contextPrices?.cache_ratio ?? other.cache_ratio ?? 0),
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })

  const cacheWrite5m = other.cache_creation_tokens_5m || 0
  const cacheWrite1h = other.cache_creation_tokens_1h || 0
  const hasSplitCacheWrite = cacheWrite5m > 0 || cacheWrite1h > 0
  if (hasSplitCacheWrite) {
    pushTokenBillingRow({
      rows,
      labelKey: 'Cache Creation (5m)',
      tokens: cacheWrite5m,
      unitPriceUSD:
        baseInputUSD *
        (contextPrices?.cache_creation_ratio_5m ??
          other.cache_creation_ratio_5m ??
          other.cache_creation_ratio ??
          0),
      groupRatio,
      dynamicRatio,
      ratioText,
      formatPrice,
    })
    pushTokenBillingRow({
      rows,
      labelKey: 'Cache Creation (1h)',
      tokens: cacheWrite1h,
      unitPriceUSD:
        baseInputUSD *
        (contextPrices?.cache_creation_ratio_1h ??
          other.cache_creation_ratio_1h ??
          other.cache_creation_ratio ??
          0),
      groupRatio,
      dynamicRatio,
      ratioText,
      formatPrice,
    })
  } else {
    pushTokenBillingRow({
      rows,
      labelKey: 'Cache Creation',
      tokens: other.cache_creation_tokens || 0,
      unitPriceUSD:
        baseInputUSD *
        (contextPrices?.cache_creation_ratio ??
          other.cache_creation_ratio ??
          0),
      groupRatio,
      dynamicRatio,
      ratioText,
      formatPrice,
    })
  }

  const audioInputUnitPrice = other.audio_input_seperate_price
    ? other.audio_input_price
    : baseInputUSD * (contextPrices?.audio_ratio ?? other.audio_ratio ?? 0)
  const audioOutputUnitPrice =
    baseInputUSD *
    (contextPrices?.audio_ratio ?? other.audio_ratio ?? 0) *
    (contextPrices?.audio_completion_ratio ?? other.audio_completion_ratio ?? 0)
  pushTokenBillingRow({
    rows,
    labelKey: 'Audio Input',
    tokens: getAudioInputTokens(other),
    unitPriceUSD: audioInputUnitPrice,
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushTokenBillingRow({
    rows,
    labelKey: 'Audio Output',
    tokens: other.audio_output || 0,
    unitPriceUSD: audioOutputUnitPrice,
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushTokenBillingRow({
    rows,
    labelKey: 'Image Output',
    tokens: other.image_output || 0,
    unitPriceUSD: baseInputUSD * (other.image_ratio ?? 0),
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushMeteredBillingRow({
    rows,
    labelKey: 'Web Search',
    quantity: other.web_search_call_count || 0,
    unitPriceUSD: other.web_search_price,
    unitLabel: t('1K calls'),
    divisor: 1000,
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushMeteredBillingRow({
    rows,
    labelKey: 'File Search',
    quantity: other.file_search_call_count || 0,
    unitPriceUSD: other.file_search_price,
    unitLabel: t('1K calls'),
    divisor: 1000,
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })
  pushMeteredBillingRow({
    rows,
    labelKey: 'Image Generation',
    quantity: other.image_generation_call ? 1 : 0,
    unitPriceUSD: other.image_generation_call_price,
    unitLabel: t('call'),
    divisor: 1,
    groupRatio,
    dynamicRatio,
    ratioText,
    formatPrice,
  })

  return rows
}

function BillingBreakdown(props: {
  log: UsageLog
  other: LogOtherData
  isAdmin: boolean
}) {
  const { t } = useTranslation()
  const { log, other, isAdmin } = props
  const isPerCall = isPerCallBilling(other.model_price)
  const isTieredExpr = other.billing_mode === 'tiered_expr'
  const tieredSummary = getTieredBillingSummary(other)
  const isContextPricing = other.context_pricing_enabled === true
  const billingRows = buildBillingRows(log, other, t)
  const summaryRows: Array<{ label: string; value: string }> = []
  const multiplierRows: Array<{ label: string; value: string }> = []
  const contextPrices = getContextPricingPrices(other)
  const modelRatio = contextPrices?.model_ratio ?? other.model_ratio
  const completionRatio =
    contextPrices?.completion_ratio ?? other.completion_ratio

  if (isTieredExpr) {
    summaryRows.push({
      label: t('Billing Mode'),
      value: t('Dynamic Pricing'),
    })
    if (tieredSummary) {
      if (tieredSummary.tier.label) {
        summaryRows.push({
          label: t('Matched Tier'),
          value: tieredSummary.tier.label,
        })
      }
    } else {
      summaryRows.push({
        label: t('Matched Tier'),
        value: t('No matching results'),
      })
    }
  } else if (isPerCall) {
    summaryRows.push({ label: t('Billing Mode'), value: t('Per-call billing') })
  } else {
    const modeKey = isContextPricing
      ? 'Per-token segmented billing'
      : 'Per-token non-segmented billing'
    summaryRows.push({ label: t('Billing Mode'), value: t(modeKey) })
  }

  if (isContextPricing) {
    summaryRows.push({
      label: t('Matched Segment'),
      value: [
        getContextPricingRange(other),
        other.context_pricing_tier_name || other.context_pricing?.tier_name,
      ]
        .filter(Boolean)
        .join(' '),
    })
    summaryRows.push({
      label: t('Segment Context Tokens'),
      value: formatExactTokens(other.context_tokens_for_tier || 0),
    })
  }

  if (modelRatio != null && Number.isFinite(modelRatio)) {
    multiplierRows.push({
      label: t('Model Ratio'),
      value: `${compactRatio(modelRatio)}x`,
    })
  }
  if (completionRatio != null && Number.isFinite(completionRatio)) {
    multiplierRows.push({
      label: t('Completion Ratio'),
      value: `${compactRatio(completionRatio)}x`,
    })
  }

  const effectiveGroupRatio = getEffectiveGroupRatio(other)
  if (effectiveGroupRatio) {
    multiplierRows.push({
      label: t(effectiveGroupRatio.labelKey),
      value: `${compactRatio(effectiveGroupRatio.value)}x`,
    })
  }

  const dynamicRatio = getDynamicRatio(other)
  if (dynamicRatio !== 1) {
    multiplierRows.push({
      label: t('Dynamic Ratio'),
      value: `${compactRatio(dynamicRatio)}x`,
    })
  }

  const ratioEntries = [
    ['Cache Read Ratio', contextPrices?.cache_ratio ?? other.cache_ratio],
    [
      'Cache Creation Ratio',
      contextPrices?.cache_creation_ratio ?? other.cache_creation_ratio,
    ],
    [
      'Cache Creation 5m Ratio',
      contextPrices?.cache_creation_ratio_5m ?? other.cache_creation_ratio_5m,
    ],
    [
      'Cache Creation 1h Ratio',
      contextPrices?.cache_creation_ratio_1h ?? other.cache_creation_ratio_1h,
    ],
    ['Audio Input Ratio', contextPrices?.audio_ratio ?? other.audio_ratio],
    [
      'Audio Output Ratio',
      contextPrices?.audio_completion_ratio ?? other.audio_completion_ratio,
    ],
  ] as const

  for (const [labelKey, value] of ratioEntries) {
    if (value != null && Number.isFinite(value)) {
      multiplierRows.push({
        label: t(labelKey),
        value: `${compactRatio(value)}x`,
      })
    }
  }

  if (isAdmin && other.admin_info) {
    summaryRows.push({
      label: t('Billing Source'),
      value: other.admin_info.local_count_tokens
        ? t('Local Billing')
        : t('Upstream Response'),
    })
  }

  summaryRows.push({
    label: t('Total Cost'),
    value: formatLogQuota(log.quota),
  })

  if (summaryRows.length === 0 && billingRows.length === 0) return null

  return (
    <DetailSection label={t('Billing Details')}>
      <div className='grid gap-1.5 md:grid-cols-2'>
        {summaryRows.map((row, idx) => (
          <DetailRow key={idx} label={row.label} value={row.value} mono />
        ))}
      </div>
      {multiplierRows.length > 0 && (
        <div className='border-border/70 mt-2 border-t pt-2'>
          <Label className='mb-1.5 block text-xs font-semibold'>
            {t('Multipliers')}
          </Label>
          <div className='grid gap-1.5 md:grid-cols-2'>
            {multiplierRows.map((row, idx) => (
              <DetailRow key={idx} label={row.label} value={row.value} mono />
            ))}
          </div>
        </div>
      )}
      {billingRows.length > 0 && (
        <div className='border-border/70 mt-2 min-w-0 border-t pt-2'>
          <Label className='mb-1.5 block text-xs font-semibold'>
            {t('Current Price Table')}
          </Label>
          <div className='overflow-x-auto rounded-md border'>
            <table className='w-full min-w-[680px] text-left text-xs'>
              <thead className='bg-muted/60 text-muted-foreground'>
                <tr>
                  <th className='px-2 py-1.5 font-medium'>
                    {t('Billing Item')}
                  </th>
                  <th className='px-2 py-1.5 font-medium'>{t('Quantity')}</th>
                  <th className='px-2 py-1.5 font-medium'>{t('Unit Price')}</th>
                  <th className='px-2 py-1.5 font-medium'>{t('Ratios')}</th>
                  <th className='px-2 py-1.5 font-medium'>{t('Subtotal')}</th>
                </tr>
              </thead>
              <tbody>
                {billingRows.map((row, idx) => (
                  <tr key={idx} className='border-t'>
                    <td className='px-2 py-1.5 font-medium'>
                      {t(row.labelKey)}
                    </td>
                    <td className='px-2 py-1.5 font-mono'>{row.quantity}</td>
                    <td className='px-2 py-1.5 font-mono'>{row.unitPrice}</td>
                    <td className='px-2 py-1.5 font-mono'>{row.ratios}</td>
                    <td className='px-2 py-1.5 font-mono'>{row.subtotal}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </DetailSection>
  )
}

function TokenBreakdown(props: { log: UsageLog; other: LogOtherData }) {
  const { t } = useTranslation()
  const { log, other } = props

  const promptTokens = log.prompt_tokens || 0
  const completionTokens = log.completion_tokens || 0
  const cacheRead = other.cache_tokens || 0
  const cacheWrite = other.cache_creation_tokens || 0
  const cacheWrite5m = other.cache_creation_tokens_5m || 0
  const cacheWrite1h = other.cache_creation_tokens_1h || 0
  const ordinaryInput = getOrdinaryInputTokens(log, other)
  const audioInput = getAudioInputTokens(other)
  const audioOutput = other.audio_output || 0
  const textOutput = other.text_output || 0
  const imageOutput = other.image_output || 0
  const hasTokens =
    promptTokens > 0 ||
    completionTokens > 0 ||
    cacheRead > 0 ||
    getCacheCreationTotal(other) > 0 ||
    audioInput > 0 ||
    audioOutput > 0 ||
    imageOutput > 0

  if (!hasTokens) return null

  const standardRows = [
    { label: t('Input Tokens'), value: formatExactTokens(ordinaryInput) },
    { label: t('Output Tokens'), value: formatExactTokens(completionTokens) },
  ]
  if (cacheRead > 0 || getCacheCreationTotal(other) > 0) {
    standardRows.push({
      label: t('Total Request Input'),
      value: formatExactTokens(promptTokens),
    })
  }

  const cacheRows: Array<{ label: string; value: string }> = []
  if (cacheRead > 0) {
    cacheRows.push({
      label: t('Cache Read'),
      value: formatExactTokens(cacheRead),
    })
  }
  if (cacheWrite > 0 && cacheWrite5m === 0 && cacheWrite1h === 0) {
    cacheRows.push({
      label: t('Cache Creation'),
      value: formatExactTokens(cacheWrite),
    })
  }
  if (cacheWrite5m > 0) {
    cacheRows.push({
      label: t('Cache Creation (5m)'),
      value: formatExactTokens(cacheWrite5m),
    })
  }
  if (cacheWrite1h > 0) {
    cacheRows.push({
      label: t('Cache Creation (1h)'),
      value: formatExactTokens(cacheWrite1h),
    })
  }

  const multimodalRows: Array<{ label: string; value: string }> = []
  if ((other.audio || other.ws) && other.text_input != null) {
    multimodalRows.push({
      label: t('Text Input'),
      value: formatExactTokens(other.text_input),
    })
  }
  if ((other.audio || other.ws) && textOutput > 0) {
    multimodalRows.push({
      label: t('Text Output'),
      value: formatExactTokens(textOutput),
    })
  }
  if (audioInput > 0) {
    multimodalRows.push({
      label: t('Audio Input'),
      value: formatExactTokens(audioInput),
    })
  }
  if (audioOutput > 0) {
    multimodalRows.push({
      label: t('Audio Output'),
      value: formatExactTokens(audioOutput),
    })
  }
  if (imageOutput > 0) {
    multimodalRows.push({
      label: t('Image Output'),
      value: formatExactTokens(imageOutput),
    })
  }

  const groups = [
    { title: t('Standard Tokens'), rows: standardRows },
    { title: t('Cache Tokens'), rows: cacheRows },
    { title: t('Multimodal Tokens'), rows: multimodalRows },
  ]

  return (
    <DetailSection label={t('Token Breakdown')}>
      <div className='grid gap-2 md:grid-cols-3'>
        {groups.map((group) => (
          <div
            key={group.title}
            className='bg-background/50 min-w-0 rounded-md border p-2'
          >
            <div className='text-muted-foreground mb-1.5 text-xs font-medium'>
              {group.title}
            </div>
            {group.rows.length > 0 ? (
              <div className='space-y-1'>
                {group.rows.map((row, idx) => (
                  <DetailRow
                    key={idx}
                    label={row.label}
                    value={row.value}
                    mono
                  />
                ))}
              </div>
            ) : (
              <span className='text-muted-foreground text-xs'>-</span>
            )}
          </div>
        ))}
      </div>
    </DetailSection>
  )
}

interface DetailsDialogProps {
  log: UsageLog
  isAdmin: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DetailsDialog(props: DetailsDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const details = props.log.content ?? ''
  const other = parseLogOther(props.log.other)
  const typeConfig = getLogTypeConfig(props.log.type)

  const isViolation = isViolationFeeLog(other)
  const isRefund = props.log.type === 6
  const isConsume = props.log.type === 2
  const isTopup = props.log.type === 1
  const isManage = props.log.type === 3
  const isSubscription = other?.billing_source === 'subscription'
  const isTieredBilling =
    isConsume &&
    !isViolation &&
    other?.billing_mode === 'tiered_expr' &&
    !!other?.expr_b64
  const hasAudioTokens = other?.ws || other?.audio
  const showTiming = isTimingLogType(props.log.type)
  const showAdminIp =
    !!props.log.ip && (showTiming || (props.isAdmin && isTopup))
  const adminInfo = other?.admin_info
  const topupAuditFields =
    isTopup && props.isAdmin && adminInfo
      ? ([
          adminInfo.payment_method && {
            label: t('Order Payment Method'),
            value: adminInfo.payment_method,
          },
          adminInfo.callback_payment_method && {
            label: t('Callback Payment Method'),
            value: adminInfo.callback_payment_method,
          },
          adminInfo.caller_ip && {
            label: t('Callback Caller IP'),
            value: adminInfo.caller_ip,
          },
          adminInfo.server_ip && {
            label: t('Server IP'),
            value: adminInfo.server_ip,
          },
          adminInfo.node_name && {
            label: t('Node Name'),
            value: adminInfo.node_name,
          },
          adminInfo.version && {
            label: t('System Version'),
            value: adminInfo.version,
          },
        ].filter(Boolean) as Array<{ label: string; value: string }>)
      : []
  const showLegacyTopupWarning = isTopup && props.isAdmin && !adminInfo
  const showTopupAuditSection =
    isTopup &&
    props.isAdmin &&
    (topupAuditFields.length > 0 || showLegacyTopupWarning)
  const manageOperator = (() => {
    if (!isManage || !props.isAdmin || !adminInfo) return null
    const username = adminInfo.admin_username
    const id = adminInfo.admin_id
    const hasUsername = username != null && String(username).trim() !== ''
    const hasId = id != null && String(id).trim() !== ''
    if (!hasUsername && !hasId) return null
    if (hasUsername && hasId) return `${username} (ID: ${id})`
    if (hasUsername) return String(username)
    return `ID: ${id}`
  })()

  const conversionChain =
    other && Array.isArray(other.request_conversion)
      ? other.request_conversion.filter(Boolean)
      : []
  const conversionLabel =
    conversionChain.length <= 1
      ? t('Native format')
      : conversionChain.join(' -> ')
  const showConversion =
    props.isAdmin &&
    props.log.type !== 6 &&
    (other?.request_path || conversionChain.length > 0)

  const useChannel = other?.admin_info?.use_channel
  const channelChain =
    useChannel && useChannel.length > 0 ? useChannel.join(' → ') : undefined

  let dialogWidthClass = 'sm:max-w-lg'
  if (isConsume && !isViolation) {
    dialogWidthClass = 'sm:max-w-[64rem] lg:max-w-[72rem]'
  } else if (isTieredBilling) {
    dialogWidthClass = 'sm:max-w-4xl lg:max-w-5xl'
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent
        className={cn(
          'min-w-0 overflow-hidden',
          'max-sm:max-h-[calc(100dvh-1.5rem)] max-sm:w-[calc(100vw-1.5rem)] max-sm:max-w-[calc(100vw-1.5rem)] max-sm:p-4',
          dialogWidthClass
        )}
      >
        <DialogHeader className='max-sm:gap-1'>
          <DialogTitle className='flex items-center gap-2 text-base'>
            {t('Log Details')}
            <StatusBadge
              label={t(typeConfig.label)}
              variant={typeConfig.color as StatusBadgeProps['variant']}
              size='sm'
              copyable={false}
            />
          </DialogTitle>
          <DialogDescription className='sr-only'>
            {t('View the complete details for this log entry')}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[70vh] min-w-0 overflow-hidden pr-2 max-sm:max-h-[calc(100dvh-7rem)] sm:pr-4'>
          <div className='w-full max-w-full min-w-0 space-y-2.5 overflow-hidden py-1 sm:space-y-3'>
            {/* Overview section - key identifiers */}
            <div className='min-w-0 space-y-1'>
              {props.log.request_id && (
                <DetailRow
                  label={t('Request ID')}
                  value={props.log.request_id}
                  mono
                />
              )}
              {props.log.upstream_request_id && (
                <DetailRow
                  label={t('Upstream Request ID')}
                  value={props.log.upstream_request_id}
                  mono
                />
              )}

              {props.isAdmin && props.log.channel > 0 && (
                <DetailRow
                  label={t('Channel')}
                  value={
                    <span>
                      {props.log.channel}
                      {props.log.channel_name && (
                        <span className='text-muted-foreground'>
                          {' '}
                          ({props.log.channel_name})
                        </span>
                      )}
                    </span>
                  }
                  mono
                />
              )}

              {channelChain && props.isAdmin && (
                <DetailRow label={t('Retry Chain')} value={channelChain} mono />
              )}

              {props.log.token_name && (
                <DetailRow
                  label={t('Token')}
                  value={props.log.token_name}
                  mono
                />
              )}

              {(props.log.group || other?.group) && (
                <DetailRow
                  label={t('Group')}
                  value={props.log.group || other?.group || ''}
                  mono
                />
              )}

              {showAdminIp && (
                <DetailRow
                  label={t('IP Address')}
                  value={
                    <span className='flex items-center gap-1'>
                      <Globe
                        className='size-3 text-amber-500'
                        aria-hidden='true'
                      />
                      {props.log.ip}
                    </span>
                  }
                  mono
                />
              )}

              {showTiming && props.log.use_time > 0 && (
                <DetailRow
                  label={t('Response Time')}
                  value={
                    <span
                      className={cn(
                        'font-medium',
                        timingTextColorClass(
                          getResponseTimeColor(
                            props.log.use_time / 1000,
                            props.log.completion_tokens
                          )
                        )
                      )}
                    >
                      {formatUseTime(props.log.use_time / 1000)}
                      {props.log.is_stream &&
                        other?.frt != null &&
                        other.frt > 0 && (
                          <span
                            className={cn(
                              'font-normal',
                              timingTextColorClass(
                                getFirstResponseTimeColor(other.frt / 1000)
                              )
                            )}
                          >
                            {' '}
                            (FRT: {formatUseTime(other.frt / 1000)})
                          </span>
                        )}
                    </span>
                  }
                />
              )}
            </div>

            {/* Request conversion (admin only, not for refund) */}
            {showConversion && (
              <DetailSection label={t('Request Conversion')}>
                <div className='relative min-w-0'>
                  <Button
                    variant='ghost'
                    size='sm'
                    className='absolute top-0 right-0 h-5 w-5 p-0'
                    onClick={() => copyToClipboard(conversionLabel)}
                    title={t('Copy to clipboard')}
                    aria-label={t('Copy to clipboard')}
                  >
                    {copiedText === conversionLabel ? (
                      <Check className='size-3 text-green-600' />
                    ) : (
                      <Copy className='size-3' />
                    )}
                  </Button>
                  <div className='min-w-0 space-y-1 pr-6'>
                    {other?.request_path && (
                      <DetailRow
                        label={t('Path')}
                        value={other.request_path}
                        mono
                      />
                    )}
                    <div className='flex min-w-0 items-center gap-1.5 text-xs'>
                      <Route
                        className='text-muted-foreground size-3'
                        aria-hidden='true'
                      />
                      <span className='min-w-0 break-all sm:break-words'>
                        {conversionLabel}
                      </span>
                    </div>
                  </div>
                </div>
              </DetailSection>
            )}

            {/* Reject reason (admin only) */}
            {props.isAdmin && other?.reject_reason && (
              <DetailSection
                icon={<AlertTriangle className='size-3.5' aria-hidden='true' />}
                label={t('Reject Reason')}
                variant='danger'
              >
                <p className='text-xs break-words'>{other.reject_reason}</p>
              </DetailSection>
            )}

            {/* Violation fee info */}
            {isViolation && other && (
              <DetailSection
                icon={<AlertTriangle className='size-3.5' aria-hidden='true' />}
                label={t('Violation Fee')}
                variant='danger'
              >
                {other.violation_fee_code && (
                  <DetailRow
                    label={t('Violation Code')}
                    value={other.violation_fee_code}
                    mono
                  />
                )}
                {other.violation_fee_marker && (
                  <DetailRow
                    label={t('Violation Marker')}
                    value={other.violation_fee_marker}
                  />
                )}
                <DetailRow
                  label={t('Fee Amount')}
                  value={formatLogQuota(other.fee_quota ?? props.log.quota)}
                  mono
                />
              </DetailSection>
            )}

            {/* Refund details (type=6) */}
            {isRefund && other && (other.task_id || other.reason) && (
              <DetailSection label={t('Refund Details')}>
                {other.task_id && (
                  <DetailRow label={t('Task ID')} value={other.task_id} mono />
                )}
                {other.reason && (
                  <DetailRow label={t('Reason')} value={other.reason} />
                )}
              </DetailSection>
            )}

            {/* Top-up audit info (type=1, admin only) */}
            {showTopupAuditSection && (
              <DetailSection
                icon={<ShieldCheck className='size-3.5' aria-hidden='true' />}
                label={t('Top-up Audit Info')}
              >
                {topupAuditFields.map((field, idx) => (
                  <DetailRow
                    key={idx}
                    label={field.label}
                    value={field.value}
                    mono
                  />
                ))}
                {showLegacyTopupWarning && (
                  <div className='flex items-start gap-1.5 text-xs text-amber-600 dark:text-amber-400'>
                    <Info
                      className='mt-0.5 size-3.5 shrink-0'
                      aria-hidden='true'
                    />
                    <span>
                      {t(
                        'This record was written by a pre-upgrade instance and lacks audit info. Upgrade the instance to record server IP, callback IP, payment method and system version.'
                      )}
                    </span>
                  </div>
                )}
              </DetailSection>
            )}

            {/* Manage operator (type=3, admin only) */}
            {manageOperator && (
              <DetailRow
                label={
                  <span className='flex items-center gap-1.5'>
                    <UserCog
                      className='text-muted-foreground size-3.5'
                      aria-hidden='true'
                    />
                    {t('Operator Admin')}
                  </span>
                }
                value={manageOperator}
                mono
              />
            )}

            {/* Audio/WebSocket token breakdown */}
            {hasAudioTokens && other && (
              <DetailSection
                icon={<Headphones className='size-3.5' aria-hidden='true' />}
                label={t('Audio Tokens')}
              >
                {other.audio_input != null && other.audio_input > 0 && (
                  <DetailRow
                    label={t('Audio Input')}
                    value={formatTokens(other.audio_input)}
                    mono
                  />
                )}
                {other.audio_output != null && other.audio_output > 0 && (
                  <DetailRow
                    label={t('Audio Output')}
                    value={formatTokens(other.audio_output)}
                    mono
                  />
                )}
                {other.text_input != null && other.text_input > 0 && (
                  <DetailRow
                    label={t('Text Input')}
                    value={formatTokens(other.text_input)}
                    mono
                  />
                )}
                {other.text_output != null && other.text_output > 0 && (
                  <DetailRow
                    label={t('Text Output')}
                    value={formatTokens(other.text_output)}
                    mono
                  />
                )}
              </DetailSection>
            )}

            {/* Reasoning effort */}
            {other?.reasoning_effort && (
              <DetailRow
                label={t('Reasoning Effort')}
                value={
                  <StatusBadge
                    label={other.reasoning_effort}
                    variant={
                      other.reasoning_effort === 'high'
                        ? 'orange'
                        : other.reasoning_effort === 'medium'
                          ? 'yellow'
                          : 'green'
                    }
                    size='sm'
                    copyable={false}
                  />
                }
              />
            )}

            {/* System prompt override */}
            {other?.is_system_prompt_overwritten && (
              <DetailRow
                label={t('System Prompt')}
                value={
                  <StatusBadge
                    label={t('Overwritten')}
                    variant='orange'
                    size='sm'
                    copyable={false}
                  />
                }
              />
            )}

            {/* Model mapping */}
            {other?.is_model_mapped && other?.upstream_model_name && (
              <DetailSection label={t('Model Mapping')}>
                <DetailRow
                  label={t('Request Model')}
                  value={props.log.model_name}
                  mono
                />
                <DetailRow
                  label={t('Actual Model')}
                  value={other.upstream_model_name}
                  mono
                />
              </DetailSection>
            )}

            {/* Token breakdown (for consume/error types with token data) */}
            {isDisplayableType(props.log.type) && other && (
              <TokenBreakdown log={props.log} other={other} />
            )}

            {/* Billing breakdown (consume type) */}
            {isConsume && other && !isViolation && (
              <BillingBreakdown
                log={props.log}
                other={other}
                isAdmin={props.isAdmin}
              />
            )}

            {/* Tiered pricing breakdown (when billing_mode is tiered_expr) */}
            {isTieredBilling && other?.expr_b64 && (
              <div className='bg-muted/30 min-w-0 overflow-hidden rounded-md border px-3 max-sm:px-2'>
                <DynamicPricingBreakdown
                  billingExpr={decodeBillingExprB64(other.expr_b64)}
                  matchedTierLabel={other.matched_tier}
                  hideCacheColumns={!hasAnyCacheTokens(other)}
                />
              </div>
            )}

            {/* Admin billing mode indicator for non-consume */}
            {props.isAdmin &&
              !isConsume &&
              props.log.type !== 6 &&
              other?.admin_info && (
                <DetailRow
                  label={t('Billing Source')}
                  value={
                    <span className='flex items-center gap-1'>
                      {other.admin_info.local_count_tokens ? (
                        <Monitor className='size-3 text-blue-500' />
                      ) : (
                        <Cloud className='size-3 text-emerald-500' />
                      )}
                      <span className='text-xs'>
                        {other.admin_info.local_count_tokens
                          ? t('Local Billing')
                          : t('Upstream Response')}
                      </span>
                    </span>
                  }
                />
              )}

            {/* Stream status details (admin only) */}
            {props.isAdmin &&
              other?.stream_status &&
              other.stream_status.status !== 'ok' && (
                <DetailSection label={t('Stream Status')}>
                  <DetailRow
                    label={t('Status')}
                    value={
                      <StatusBadge
                        label={other.stream_status.status || t('Error')}
                        variant='red'
                        size='sm'
                        copyable={false}
                      />
                    }
                  />
                  {other.stream_status.end_reason && (
                    <DetailRow
                      label={t('End Reason')}
                      value={other.stream_status.end_reason}
                    />
                  )}
                  {(other.stream_status.error_count ?? 0) > 0 && (
                    <DetailRow
                      label={t('Soft Errors')}
                      value={String(other.stream_status.error_count)}
                    />
                  )}
                  {other.stream_status.end_error && (
                    <DetailRow
                      label={t('End Error')}
                      value={other.stream_status.end_error}
                    />
                  )}
                  {Array.isArray(other.stream_status.errors) &&
                    other.stream_status.errors.length > 0 && (
                      <pre className='bg-background/60 mt-1 max-h-32 overflow-y-auto rounded border p-2 font-mono text-[11px] leading-relaxed break-words whitespace-pre-wrap'>
                        {other.stream_status.errors.join('\n')}
                      </pre>
                    )}
                </DetailSection>
              )}

            {/* Subscription billing details */}
            {isSubscription && other && (
              <DetailSection label={t('Subscription Billing')}>
                {other.subscription_plan_id && (
                  <DetailRow
                    label={t('Plan')}
                    value={`#${other.subscription_plan_id} ${other.subscription_plan_title || ''}`.trim()}
                  />
                )}
                {other.subscription_id && (
                  <DetailRow
                    label={t('Instance')}
                    value={`#${other.subscription_id}`}
                    mono
                  />
                )}
                {other.subscription_pre_consumed != null && (
                  <DetailRow
                    label={t('Pre-consumed')}
                    value={formatLogQuota(other.subscription_pre_consumed)}
                    mono
                  />
                )}
                {other.subscription_post_delta != null &&
                  other.subscription_post_delta !== 0 && (
                    <DetailRow
                      label={t('Post Delta')}
                      value={formatLogQuota(other.subscription_post_delta)}
                      mono
                    />
                  )}
                {other.subscription_consumed != null && (
                  <DetailRow
                    label={t('Final Consumed')}
                    value={formatLogQuota(other.subscription_consumed)}
                    mono
                  />
                )}
                {other.subscription_remain != null && (
                  <DetailRow
                    label={t('Remaining')}
                    value={`${formatLogQuota(other.subscription_remain)}${other.subscription_total != null ? ` / ${formatLogQuota(other.subscription_total)}` : ''}`}
                    mono
                  />
                )}
              </DetailSection>
            )}

            {/* Param override */}
            {other?.po && Array.isArray(other.po) && other.po.length > 0 && (
              <DetailSection
                icon={<Settings2 className='size-3.5' aria-hidden='true' />}
                label={`${t('Param Override')} (${other.po.length})`}
              >
                {other.po.filter(Boolean).map((line, idx) => {
                  const parsed = parseAuditLine(line)
                  if (!parsed) return null
                  return (
                    <div
                      key={idx}
                      className='bg-background/60 flex min-w-0 flex-col gap-1.5 rounded border p-2 sm:flex-row sm:items-start sm:gap-2'
                    >
                      <StatusBadge
                        variant='neutral'
                        label={getParamOverrideActionLabel(parsed.action, t)}
                        className='shrink-0 font-medium'
                        copyable={false}
                      />
                      <span className='min-w-0 font-mono text-[11px] leading-relaxed break-all sm:break-words'>
                        {parsed.content}
                      </span>
                    </div>
                  )
                })}
              </DetailSection>
            )}

            {/* Content */}
            {details && (
              <div className='space-y-1.5'>
                <Label className='text-xs font-semibold'>{t('Content')}</Label>
                <div className='bg-muted/30 relative min-w-0 overflow-hidden rounded-md border p-2.5'>
                  <Button
                    variant='ghost'
                    size='sm'
                    className='absolute top-1.5 right-1.5 h-5 w-5 p-0'
                    onClick={() => copyToClipboard(details)}
                    title={t('Copy to clipboard')}
                    aria-label={t('Copy to clipboard')}
                  >
                    {copiedText === details ? (
                      <Check className='size-3 text-green-600' />
                    ) : (
                      <Copy className='size-3' />
                    )}
                  </Button>
                  <p className='min-w-0 pr-6 text-xs leading-relaxed break-all whitespace-pre-wrap sm:break-words'>
                    {details}
                  </p>
                </div>
              </div>
            )}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

function isDisplayableType(type: number): boolean {
  return [0, 2, 5, 6].includes(type)
}
