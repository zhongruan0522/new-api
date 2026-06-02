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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { VChart } from '@visactor/react-vchart'
import { AlertTriangle, CheckCircle2, Loader2, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { getGlmPlanUsage, getPlanQuota } from '../../api'
import type {
  Channel,
  GlmUsageType,
  PlanLimitInfo,
  PlanMcpLimitInfo,
  PlanQuotaData,
  PlanTierInfo,
} from '../../types'
import { useChannels } from '../channels-provider'

const PLAN_DISPLAY_NAMES: Record<string, string> = {
  'glm-coding-plan': '智谱 GLM 官方套餐',
  'glm-coding-plan-international': '智谱 GLM 国际套餐',
  'kimi-coding-plan': 'Kimi 官方套餐',
  'minimax-coding-plan': 'MiniMax 官方套餐',
  'minimax-coding-plan-international': 'MiniMax 国际套餐',
  'ollama-coding-plan': 'Ollama 官方套餐',
}

const FALLBACK_MODEL_COLORS = [
  '#d97757',
  '#6a9bcc',
  '#788c5d',
  '#9b6db7',
  '#c4a44e',
  '#5bb8a9',
]

const CHART_COLOR_VARIABLES = [
  '--chart-1',
  '--chart-2',
  '--chart-3',
  '--chart-4',
  '--chart-5',
] as const

type PlanQuotaDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type UsagePoint = {
  time: string
  value: number
  type: string
}

type ModelSummary = {
  modelName?: string
  totalTokens?: number
}

type FlattenedUsageData = {
  values: UsagePoint[]
  total: number
  summary: ModelSummary[]
  fields: string[]
  times: string[]
}

type PerformanceData = {
  values: UsagePoint[]
  avgSpeed: string
  avgRate: string
  times: string[]
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function getThemeChartColors(themeKey?: string): string[] {
  if (typeof document === 'undefined') return FALLBACK_MODEL_COLORS
  void themeKey

  const bodyStyle = window.getComputedStyle(document.body)
  const rootStyle = window.getComputedStyle(document.documentElement)
  const colors = CHART_COLOR_VARIABLES.map((name) => {
    return (
      bodyStyle.getPropertyValue(name) || rootStyle.getPropertyValue(name)
    ).trim()
  }).filter(Boolean)

  return colors.length > 0 ? colors : FALLBACK_MODEL_COLORS
}

function numberArray(value: unknown): number[] {
  return Array.isArray(value)
    ? value.map((item) => {
        const n = Number(item)
        return Number.isFinite(n) ? n : 0
      })
    : []
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.map((item) => String(item)) : []
}

function clampPercent(value: unknown): number {
  const n = Number(value)
  if (!Number.isFinite(n)) return 0
  return Math.max(0, Math.min(100, Math.round(n)))
}

function formatCompactNumber(value: unknown): string {
  const num = Number(value)
  if (!Number.isFinite(num)) return '0'
  const abs = Math.abs(num)
  const sign = num < 0 ? '-' : ''
  if (abs >= 1_000_000_000) {
    return sign + (abs / 1_000_000_000).toFixed(1).replace(/\.0$/, '') + 'B'
  }
  if (abs >= 1_000_000) {
    return sign + (abs / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  }
  if (abs >= 1_000) {
    return sign + (abs / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  }
  return sign + abs.toLocaleString()
}

function getStatusVariant(status?: string, percentage?: number): StatusVariant {
  if (status === '紧张' || (percentage ?? 0) >= 80) return 'danger'
  if (status === '适中' || (percentage ?? 0) >= 50) return 'warning'
  if (status === '充裕') return 'success'
  return 'neutral'
}

function getPlanDisplayName(planName?: string): string {
  if (!planName) return ''
  return PLAN_DISPLAY_NAMES[planName] || planName
}

function isGlmPlan(planName?: string) {
  return (
    planName === 'glm-coding-plan' ||
    planName === 'glm-coding-plan-international'
  )
}

function isTierBasedPlan(planName?: string) {
  return (
    planName === 'kimi-coding-plan' ||
    planName === 'minimax-coding-plan' ||
    planName === 'minimax-coding-plan-international'
  )
}

function formatTimeLabel(timeStr: string): string {
  if (!timeStr) return ''
  const match = timeStr.match(/^(\d{4})-(\d{2})-(\d{2})\s*(.*)?$/)
  if (!match) return timeStr
  const date = `${match[2]}-${match[3]}`
  const time = match[4] || ''
  const timeMatch = time.match(/^(\d{2}):(\d{2})/)
  return timeMatch ? `${date} ${timeMatch[1]}:${timeMatch[2]}` : date
}

function sampleTimeLabels(times: string[], maxLabels = 4): string[] {
  if (times.length <= maxLabels) return times
  const step = (times.length - 1) / (maxLabels - 1)
  const result: string[] = []
  for (let i = 0; i < maxLabels; i += 1) {
    result.push(times[Math.round(i * step)])
  }
  return result
}

function toBjDate(date: Date): Date {
  const utc = date.getTime() + date.getTimezoneOffset() * 60000
  return new Date(utc + 8 * 3600000)
}

function formatBjParamDate(date: Date): string {
  const bj = toBjDate(date)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${bj.getFullYear()}-${pad(bj.getMonth() + 1)}-${pad(bj.getDate())} ${pad(bj.getHours())}:${pad(bj.getMinutes())}:${pad(bj.getSeconds())}`
}

function getUsageTimeParams(days: number) {
  const now = new Date()
  const end = new Date(now.getTime() - 600000)
  const start = new Date()
  if (days === 0) {
    start.setHours(0, 0, 0, 0)
  } else {
    start.setTime(now.getTime() - days * 86400000)
  }
  return {
    startTime: formatBjParamDate(start),
    endTime: formatBjParamDate(end),
  }
}

function getPerfTimeParams(days: number) {
  const now = new Date()
  const yesterday = new Date(now.getTime() - 86400000)
  const end = new Date(
    yesterday.getFullYear(),
    yesterday.getMonth(),
    yesterday.getDate(),
    23,
    59,
    59
  )
  const start = new Date(end.getTime() - (days - 1) * 86400000)
  return {
    startTime: formatBjParamDate(start),
    endTime: formatBjParamDate(end),
  }
}

function formatResetTime(timeStr?: string): string {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  if (Number.isNaN(date.getTime())) return timeStr
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function formatHourReset(timeStr?: string): string {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  if (Number.isNaN(date.getTime())) return timeStr
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')} 重置`
}

function flattenUsageData(
  rawData: Record<string, unknown> | null,
  usageType: GlmUsageType
): FlattenedUsageData {
  const data = isRecord(rawData?.data) ? rawData.data : {}
  const times = stringArray(data.x_time)
  if (times.length === 0) {
    return { values: [], total: 0, summary: [], fields: [], times: [] }
  }

  if (usageType === 'model') {
    const totalUsage = isRecord(data.totalUsage) ? data.totalUsage : {}
    const totalTokens = Number(totalUsage.totalTokensUsage) || 0
    const totalArr = numberArray(data.tokensUsage)
    const modelList = Array.isArray(data.modelDataList)
      ? data.modelDataList.filter(isRecord)
      : []
    const summarySource = Array.isArray(data.modelSummaryList)
      ? data.modelSummaryList
      : Array.isArray(totalUsage.modelSummaryList)
        ? totalUsage.modelSummaryList
        : []
    const summary = summarySource.filter(isRecord).map((item) => ({
      modelName: String(item.modelName ?? ''),
      totalTokens: Number(item.totalTokens) || 0,
    }))

    const fields = [
      '总用量',
      ...modelList.map((model) => String(model.modelName ?? '')),
    ].filter(Boolean)
    const values: UsagePoint[] = []
    times.forEach((time, index) => {
      values.push({ time, value: totalArr[index] || 0, type: '总用量' })
      modelList.forEach((model) => {
        const usage = numberArray(model.tokensUsage)
        values.push({
          time,
          value: usage[index] || 0,
          type: String(model.modelName ?? ''),
        })
      })
    })

    return { values, total: totalTokens, summary, fields, times }
  }

  const networkSearch = numberArray(data.networkSearchCount)
  const webRead = numberArray(data.webReadMcpCount)
  const zread = numberArray(data.zreadMcpCount)
  const total =
    networkSearch.reduce((sum, value) => sum + value, 0) +
    webRead.reduce((sum, value) => sum + value, 0) +
    zread.reduce((sum, value) => sum + value, 0)
  const values: UsagePoint[] = []
  times.forEach((time, index) => {
    values.push({ time, value: networkSearch[index] || 0, type: '联网搜索' })
    values.push({ time, value: webRead[index] || 0, type: '网页读取' })
    values.push({ time, value: zread[index] || 0, type: '开源仓库' })
  })
  return {
    values,
    total,
    summary: [],
    fields: ['联网搜索', '网页读取', '开源仓库'],
    times,
  }
}

function flattenPerformanceData(
  rawData: Record<string, unknown> | null,
  productLevel?: string
): PerformanceData {
  const data = isRecord(rawData?.data) ? rawData.data : {}
  const times = stringArray(data.x_time)
  if (times.length === 0) {
    return { values: [], avgSpeed: '--', avgRate: '--', times: [] }
  }

  const isLite = productLevel === 'Lite'
  const speedLabel = isLite ? 'Lite速度' : 'Pro/Max速度'
  const rateLabel = isLite ? 'Lite成功率' : 'Pro/Max成功率'
  const liteSpeed = numberArray(data.liteDecodeSpeed).map((v) =>
    Number(v.toFixed(2))
  )
  const proMaxSpeed = numberArray(data.proMaxDecodeSpeed).map((v) =>
    Number(v.toFixed(2))
  )
  const liteRate = numberArray(data.liteSuccessRate).map((v) =>
    Number((v * 100).toFixed(2))
  )
  const proMaxRate = numberArray(data.proMaxSuccessRate).map((v) =>
    Number((v * 100).toFixed(2))
  )
  const speedArr = isLite ? liteSpeed : proMaxSpeed
  const rateArr = isLite ? liteRate : proMaxRate
  const values: UsagePoint[] = []
  times.forEach((time, index) => {
    values.push({ time, value: speedArr[index] || 0, type: speedLabel })
    values.push({ time, value: rateArr[index] || 0, type: rateLabel })
  })
  const avg = (arr: number[]) =>
    arr.length
      ? (arr.reduce((sum, value) => sum + value, 0) / arr.length).toFixed(1)
      : '0'

  return { values, avgSpeed: avg(speedArr), avgRate: avg(rateArr), times }
}

function LimitCard({
  title,
  data,
  resetLabel,
}: {
  title: string
  data?: PlanLimitInfo | null
  resetLabel?: string
}) {
  if (!data) return null
  const percent = clampPercent(data.percentage)
  const variant = getStatusVariant(data.status, percent)

  return (
    <div className='rounded-lg border p-4'>
      <div className='flex items-center justify-between gap-2'>
        <div className='text-sm font-medium'>{title}</div>
        <StatusBadge
          label={data.status || `${percent}%`}
          variant={variant}
          copyable={false}
        />
      </div>
      <div className='mt-3 flex items-baseline gap-2'>
        <span className='text-2xl font-semibold tabular-nums'>{percent}%</span>
      </div>
      <Progress value={percent} aria-label={`${title}: ${percent}%`} />
      {resetLabel && (
        <div className='text-muted-foreground mt-2 text-xs'>{resetLabel}</div>
      )}
    </div>
  )
}

function McpLimitCard({ data }: { data?: PlanMcpLimitInfo | null }) {
  if (!data) return null
  const percent = clampPercent(data.percentage)
  const variant = getStatusVariant(data.status, percent)

  return (
    <div className='rounded-lg border p-4'>
      <div className='flex items-center justify-between gap-2'>
        <div className='text-sm font-medium'>MCP 工具限额</div>
        <StatusBadge
          label={data.status || `${percent}%`}
          variant={variant}
          copyable={false}
        />
      </div>
      <div className='mt-3 flex items-baseline gap-2'>
        <span className='text-2xl font-semibold tabular-nums'>{percent}%</span>
        {data.current_usage && (
          <span className='text-muted-foreground text-xs'>
            {data.current_usage}
          </span>
        )}
      </div>
      <Progress value={percent} aria-label={`MCP: ${percent}%`} />
      <div className='text-muted-foreground mt-2 text-xs'>每月1号重置</div>
      {data.tools && data.tools.length > 0 && (
        <div className='mt-3 border-t pt-3'>
          {data.tools.map((tool, index) => (
            <div
              key={`${tool.name ?? 'tool'}-${index}`}
              className='flex items-center justify-between gap-3 py-1 text-xs'
            >
              <span className='text-muted-foreground truncate'>
                {tool.name || '-'}
              </span>
              <span className='font-medium tabular-nums'>
                {formatCompactNumber(tool.usage)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function TierLimitCard({ tier }: { tier: PlanTierInfo }) {
  const title = tier.name === 'five_hour' ? '每5小时限额' : '每周限额'
  const percent = clampPercent(tier.percentage)
  const reset = tier.resets_at
    ? tier.name === 'five_hour'
      ? formatHourReset(tier.resets_at)
      : formatResetTime(tier.resets_at)
    : ''

  return (
    <div className='space-y-2'>
      <LimitCard
        title={title}
        data={{ percentage: percent, status: tier.status }}
        resetLabel={
          reset
            ? tier.name === 'five_hour'
              ? reset
              : `下次重置: ${reset}`
            : ''
        }
      />
      <div className='text-muted-foreground flex justify-between gap-3 px-1 text-xs'>
        <span>已用 {formatCompactNumber(tier.used)}</span>
        <span>
          剩余 {formatCompactNumber(tier.remaining)} /{' '}
          {formatCompactNumber(tier.limit)}
        </span>
      </div>
    </div>
  )
}

function UsageChart({ channelId }: { channelId: number }) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const [usageType, setUsageType] = useState<GlmUsageType>('model')
  const [range, setRange] = useState(7)
  const [loading, setLoading] = useState(false)
  const [rawData, setRawData] = useState<Record<string, unknown> | null>(null)

  const fetchUsage = useCallback(async () => {
    setLoading(true)
    try {
      const response = await getGlmPlanUsage(channelId, {
        type: usageType,
        ...getUsageTimeParams(range),
      })
      setRawData(response)
    } catch {
      setRawData(null)
    } finally {
      setLoading(false)
    }
  }, [channelId, range, usageType])

  useEffect(() => {
    fetchUsage()
  }, [fetchUsage])

  const { values, total, summary, fields, times } = useMemo(
    () => flattenUsageData(rawData, usageType),
    [rawData, usageType]
  )
  const chartColors = useMemo(
    () => getThemeChartColors(resolvedTheme),
    [resolvedTheme]
  )

  const spec = useMemo(() => {
    if (values.length === 0) return null
    const sampledLabels = sampleTimeLabels(times, 4)
    const colorRange = fields.map((field, index) =>
      field === '总用量'
        ? 'rgba(148, 163, 184, 0.85)'
        : chartColors[index % chartColors.length]
    )

    return {
      type: 'common',
      data: [{ id: 'usage', values }],
      series: [
        {
          type: 'line',
          xField: 'time',
          yField: 'value',
          seriesField: 'type',
          smooth: true,
          line: { style: { lineWidth: 2 } },
          point: { visible: false },
        },
      ],
      axes: [
        {
          orient: 'bottom',
          type: 'band',
          bandField: 'time',
          label: {
            style: { fontSize: 11 },
            autoRotate: false,
            formatMethod: (value: number | string) => {
              const label = String(value)
              return sampledLabels.includes(label) ? formatTimeLabel(label) : ''
            },
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          type: 'linear',
          field: 'value',
          label: {
            style: { fontSize: 10 },
            formatMethod: (value: number | string) =>
              formatCompactNumber(value),
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: 'rgba(148, 163, 184, 0.35)' },
          },
        },
      ],
      color: { type: 'ordinal', range: colorRange, domain: fields },
      legends: {
        visible: true,
        position: 'top',
        item: { label: { style: { fontSize: 11 } } },
        autoPage: true,
        maxRow: 1,
      },
      tooltip: {
        visible: true,
        mark: {
          content: [
            {
              key: (datum: UsagePoint) => datum.type,
              value: (datum: UsagePoint) =>
                formatCompactNumber(datum.value ?? 0),
            },
          ],
        },
      },
      height: 240,
      padding: { top: 10, bottom: 5, left: 10, right: 10 },
      background: 'transparent',
    }
  }, [chartColors, fields, times, values])

  const ranges = [
    { key: 0, label: '当日' },
    { key: 7, label: '7天' },
    { key: 15, label: '15天' },
    { key: 30, label: '30天' },
  ]

  return (
    <section className='space-y-3 rounded-lg border p-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='bg-muted inline-flex rounded-lg p-0.5'>
          {[
            { key: 'model' as const, label: '模型' },
            { key: 'tool' as const, label: '工具' },
          ].map((item) => (
            <Button
              key={item.key}
              type='button'
              variant={usageType === item.key ? 'default' : 'ghost'}
              size='sm'
              onClick={() => setUsageType(item.key)}
            >
              {item.label}
            </Button>
          ))}
        </div>
        <div className='flex flex-wrap gap-1'>
          {ranges.map((item) => (
            <Button
              key={item.key}
              type='button'
              variant={range === item.key ? 'default' : 'outline'}
              size='sm'
              onClick={() => setRange(item.key)}
            >
              {item.label}
            </Button>
          ))}
        </div>
      </div>

      <div className='flex flex-wrap items-baseline gap-x-4 gap-y-2'>
        <div>
          <span className='text-muted-foreground text-xs'>
            {usageType === 'model' ? 'Tokens总量' : '工具调用次'}
          </span>
          <span className='ml-2 text-xl font-semibold'>
            {formatCompactNumber(total)}
          </span>
        </div>
        {usageType === 'model' && summary.length > 0 && (
          <div className='flex flex-wrap gap-2'>
            {summary.map((item, index) => (
              <span
                key={`${item.modelName ?? 'model'}-${index}`}
                className='inline-flex items-center gap-1 text-xs'
              >
                <span
                  className='h-2 w-2 rounded-full'
                  style={{
                    backgroundColor:
                      chartColors[(index + 1) % chartColors.length],
                  }}
                />
                <span>{item.modelName || '-'}</span>
                <span className='text-muted-foreground'>
                  {formatCompactNumber(item.totalTokens)}
                </span>
              </span>
            ))}
          </div>
        )}
      </div>

      <div className='min-h-[240px]'>
        {loading ? (
          <div className='flex h-[240px] items-center justify-center'>
            <Loader2 className='text-muted-foreground h-5 w-5 animate-spin' />
          </div>
        ) : themeReady && spec ? (
          <VChart
            key={`plan-usage-${channelId}-${usageType}-${range}-${resolvedTheme}`}
            spec={{
              ...spec,
              theme: resolvedTheme === 'dark' ? 'dark' : 'light',
              background: 'transparent',
            }}
            option={VCHART_OPTION}
          />
        ) : (
          <div className='text-muted-foreground flex h-[120px] items-center justify-center text-sm'>
            {t('No data')}
          </div>
        )}
      </div>
      <div className='text-muted-foreground text-right text-xs'>
        数据延迟约10分钟
      </div>
    </section>
  )
}

function PerformanceChart({
  channelId,
  productLevel,
}: {
  channelId: number
  productLevel?: string
}) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const [range, setRange] = useState(7)
  const [loading, setLoading] = useState(false)
  const [rawData, setRawData] = useState<Record<string, unknown> | null>(null)

  const fetchPerformance = useCallback(async () => {
    setLoading(true)
    try {
      const response = await getGlmPlanUsage(channelId, {
        type: 'performance',
        ...getPerfTimeParams(range),
      })
      setRawData(response)
    } catch {
      setRawData(null)
    } finally {
      setLoading(false)
    }
  }, [channelId, range])

  useEffect(() => {
    fetchPerformance()
  }, [fetchPerformance])

  const { values, avgSpeed, avgRate, times } = useMemo(
    () => flattenPerformanceData(rawData, productLevel),
    [productLevel, rawData]
  )
  const chartColors = useMemo(
    () => getThemeChartColors(resolvedTheme),
    [resolvedTheme]
  )

  const spec = useMemo(() => {
    if (values.length === 0) return null
    const sampledLabels = sampleTimeLabels(times, 4)
    const fields = Array.from(new Set(values.map((item) => item.type)))

    return {
      type: 'common',
      data: [{ id: 'performance', values }],
      series: [
        {
          type: 'line',
          xField: 'time',
          yField: 'value',
          seriesField: 'type',
          smooth: true,
          line: {
            style: {
              lineWidth: 2,
              lineDash: (datum: UsagePoint) =>
                String(datum?.type ?? '').includes('成功率') ? [4, 4] : [0],
            },
          },
          point: { visible: false },
        },
      ],
      axes: [
        {
          orient: 'bottom',
          type: 'band',
          bandField: 'time',
          label: {
            style: { fontSize: 11 },
            autoRotate: false,
            formatMethod: (value: number | string) => {
              const label = String(value)
              return sampledLabels.includes(label) ? formatTimeLabel(label) : ''
            },
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          type: 'linear',
          field: 'value',
          label: {
            style: { fontSize: 10 },
            formatMethod: (value: number | string) =>
              formatCompactNumber(value),
          },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: 'rgba(148, 163, 184, 0.35)' },
          },
        },
      ],
      color: {
        type: 'ordinal',
        range: [chartColors[0], chartColors[2] ?? chartColors[1]],
        domain: fields,
      },
      legends: {
        visible: true,
        position: 'top',
        item: { label: { style: { fontSize: 11 } } },
      },
      tooltip: {
        visible: true,
        mark: {
          content: [
            {
              key: (datum: UsagePoint) => datum.type,
              value: (datum: UsagePoint) =>
                datum.type.includes('成功率')
                  ? `${Number(datum.value || 0).toFixed(1)}%`
                  : `${Number(datum.value || 0).toFixed(1)} tokens/s`,
            },
          ],
        },
      },
      height: 240,
      padding: { top: 10, bottom: 5, left: 10, right: 10 },
      background: 'transparent',
    }
  }, [chartColors, times, values])

  const ranges = [
    { key: 7, label: '7天' },
    { key: 15, label: '15天' },
    { key: 30, label: '30天' },
  ]

  return (
    <section className='space-y-3 rounded-lg border p-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='text-sm font-semibold'>系统健康度</div>
        <div className='flex flex-wrap gap-1'>
          {ranges.map((item) => (
            <Button
              key={item.key}
              type='button'
              variant={range === item.key ? 'default' : 'outline'}
              size='sm'
              onClick={() => setRange(item.key)}
            >
              {item.label}
            </Button>
          ))}
        </div>
      </div>
      <div className='flex flex-wrap gap-6'>
        <div>
          <span className='text-muted-foreground text-xs'>平均速度</span>
          <span className='ml-2 text-xl font-semibold'>{avgSpeed}</span>
          <span className='text-muted-foreground ml-1 text-xs'>tokens/s</span>
        </div>
        <div>
          <span className='text-muted-foreground text-xs'>成功率</span>
          <span className='ml-2 text-xl font-semibold'>{avgRate}%</span>
        </div>
      </div>
      <div className='min-h-[240px]'>
        {loading ? (
          <div className='flex h-[240px] items-center justify-center'>
            <Loader2 className='text-muted-foreground h-5 w-5 animate-spin' />
          </div>
        ) : themeReady && spec ? (
          <VChart
            key={`plan-performance-${channelId}-${range}-${resolvedTheme}`}
            spec={{
              ...spec,
              theme: resolvedTheme === 'dark' ? 'dark' : 'light',
              background: 'transparent',
            }}
            option={VCHART_OPTION}
          />
        ) : (
          <div className='text-muted-foreground flex h-[120px] items-center justify-center text-sm'>
            {t('No data')}
          </div>
        )}
      </div>
    </section>
  )
}

function GlmPlanContent({
  channel,
  quotaData,
}: {
  channel: Channel
  quotaData: PlanQuotaData
}) {
  const visibleLimits = [
    quotaData.token_limit,
    quotaData.weekly_limit,
    quotaData.mcp_tool_limit,
  ].filter(Boolean).length

  return (
    <div className='space-y-4'>
      <div className='rounded-lg border p-4'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div className='text-base font-semibold'>
            {quotaData.product_name || getPlanDisplayName(quotaData.plan_name)}
          </div>
          <div className='flex flex-wrap gap-2'>
            {quotaData.plan_version && (
              <StatusBadge
                label={`${quotaData.plan_version}套餐`}
                variant={
                  quotaData.plan_version === '新' ? 'success' : 'warning'
                }
                copyable={false}
              />
            )}
            {quotaData.product_level && (
              <StatusBadge
                label={quotaData.product_level}
                variant='info'
                copyable={false}
              />
            )}
          </div>
        </div>
        <div className='text-muted-foreground mt-3 grid gap-2 text-xs sm:grid-cols-3'>
          <div>生效: {quotaData.effective_date || '-'}</div>
          <div>到期: {quotaData.expiry_date || '-'}</div>
          <div>
            {quotaData.auto_renew ? (
              <span className='text-success inline-flex items-center gap-1'>
                <CheckCircle2 className='h-3.5 w-3.5' />
                自动续费
              </span>
            ) : (
              <span className='text-warning inline-flex items-center gap-1'>
                <AlertTriangle className='h-3.5 w-3.5' />
                未开续费
              </span>
            )}
          </div>
        </div>
      </div>

      <div
        className={`grid gap-3 ${visibleLimits >= 3 ? 'lg:grid-cols-3' : 'sm:grid-cols-2'}`}
      >
        <LimitCard
          title='每5小时限额'
          data={quotaData.token_limit}
          resetLabel={formatHourReset(quotaData.token_limit?.next_reset_time)}
        />
        <LimitCard
          title='每周限额'
          data={quotaData.weekly_limit}
          resetLabel={
            formatResetTime(quotaData.weekly_limit?.next_reset_time)
              ? `下次重置: ${formatResetTime(quotaData.weekly_limit?.next_reset_time)}`
              : ''
          }
        />
        <McpLimitCard data={quotaData.mcp_tool_limit} />
      </div>

      <UsageChart channelId={channel.id} />
      <PerformanceChart
        channelId={channel.id}
        productLevel={quotaData.product_level}
      />
    </div>
  )
}

function TierPlanContent({ quotaData }: { quotaData: PlanQuotaData }) {
  return (
    <div className='space-y-4'>
      {quotaData.credential === 'expired' && (
        <div className='border-warning/30 bg-warning/10 text-warning rounded-lg border px-4 py-3 text-sm'>
          API Key 无效或已过期，请检查渠道配置
        </div>
      )}
      {quotaData.credential === 'error' && (
        <div className='border-destructive/30 bg-destructive/10 text-destructive rounded-lg border px-4 py-3 text-sm'>
          响应解析失败，API 格式可能已变更
        </div>
      )}

      <div className='rounded-lg border p-4'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div className='text-base font-semibold'>
            {getPlanDisplayName(quotaData.plan_name)}
          </div>
          {quotaData.credential === 'valid' && (
            <StatusBadge label='有效' variant='success' copyable={false} />
          )}
        </div>
      </div>

      {quotaData.tiers && quotaData.tiers.length > 0 ? (
        <div
          className={`grid gap-3 ${quotaData.tiers.length >= 3 ? 'lg:grid-cols-3' : 'sm:grid-cols-2'}`}
        >
          {quotaData.tiers.map((tier, index) => (
            <TierLimitCard
              key={`${tier.name ?? 'tier'}-${index}`}
              tier={tier}
            />
          ))}
        </div>
      ) : (
        <div className='text-muted-foreground rounded-lg border px-4 py-10 text-center text-sm'>
          暂无限额数据
        </div>
      )}
    </div>
  )
}

export function PlanQuotaDialog({ open, onOpenChange }: PlanQuotaDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const [loading, setLoading] = useState(false)
  const [quotaData, setQuotaData] = useState<PlanQuotaData | null>(null)

  const fetchQuotaData = useCallback(async () => {
    if (!currentRow?.id) return
    setLoading(true)
    try {
      const response = await getPlanQuota(currentRow.id)
      if (!response.success) {
        throw new Error(response.message || t('Failed to query plan usage'))
      }
      setQuotaData(response.data ?? null)
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to query plan usage')
      )
      setQuotaData(null)
    } finally {
      setLoading(false)
    }
  }, [currentRow?.id, t])

  useEffect(() => {
    if (open && currentRow?.id) {
      fetchQuotaData()
    }
    if (!open) {
      setQuotaData(null)
    }
  }, [currentRow?.id, fetchQuotaData, open])

  if (!currentRow) return null

  const planName = quotaData?.plan_name || currentRow.channel_info?.plan_name
  const planDisplayName = getPlanDisplayName(planName)
  const isGlmData = isGlmPlan(planName) && !!quotaData?.product_name
  const isTierData = isTierBasedPlan(planName)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[90vh] sm:max-w-5xl'>
        <DialogHeader>
          <DialogTitle>{t('Plan Usage')}</DialogTitle>
          <DialogDescription>
            {currentRow.name} {currentRow.id ? `#${currentRow.id}` : ''}
            {planDisplayName ? ` · ${planDisplayName}` : ''}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[calc(90vh-11rem)] pr-3'>
          {loading ? (
            <div className='flex h-64 items-center justify-center'>
              <Loader2 className='text-muted-foreground h-6 w-6 animate-spin' />
            </div>
          ) : quotaData && isGlmData ? (
            <GlmPlanContent channel={currentRow} quotaData={quotaData} />
          ) : quotaData && isTierData ? (
            <TierPlanContent quotaData={quotaData} />
          ) : quotaData?.quota_supported === false ? (
            <div className='text-muted-foreground rounded-lg border px-4 py-12 text-center text-sm'>
              {planDisplayName
                ? `${planDisplayName} 的额度查询功能即将上线，敬请期待。`
                : '该套餐的额度查询功能即将上线，敬请期待。'}
            </div>
          ) : (
            <div className='text-muted-foreground rounded-lg border px-4 py-12 text-center text-sm'>
              {t('No data')}
            </div>
          )}
        </ScrollArea>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={fetchQuotaData}
            disabled={loading}
          >
            {loading ? (
              <Loader2 className='mr-1.5 h-4 w-4 animate-spin' />
            ) : (
              <RefreshCw className='mr-1.5 h-4 w-4' />
            )}
            {t('Refresh')}
          </Button>
          <Button type='button' onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
