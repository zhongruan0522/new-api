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
import { useMemo, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Copy, Download, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  formatCurrencyUSD,
  formatLogQuota,
  formatTimestampToDate,
  formatUseTime,
} from '@/lib/format'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { PublicLayout } from '@/components/layout'
import { fetchKeyQueryReport } from './api'
import type { KeyQueryReport, TokenLog } from './types'

const LOG_TYPE_CONSUME = 2
const LOG_TYPE_ERROR = 5
const LOG_TYPE_REFUND = 6
const UNLIMITED_USD = 100000000
const EMPTY_LOGS: TokenLog[] = []

function isApiCallLog(record: TokenLog) {
  return (
    record.type === LOG_TYPE_CONSUME ||
    record.type === LOG_TYPE_ERROR ||
    record.type === LOG_TYPE_REFUND
  )
}

function getLogTypeLabel(type: number) {
  if (type === LOG_TYPE_CONSUME) return 'Consume'
  if (type === LOG_TYPE_ERROR) return 'Error'
  if (type === LOG_TYPE_REFUND) return 'Refund'
  return 'Other'
}

function getLogTypeVariant(type: number) {
  if (type === LOG_TYPE_ERROR) return 'destructive'
  if (type === LOG_TYPE_REFUND) return 'secondary'
  if (type === LOG_TYPE_CONSUME) return 'outline'
  return 'ghost'
}

function downloadCsv(filename: string, rows: Array<Record<string, unknown>>) {
  const headers = Object.keys(rows[0] ?? {})
  const body = [
    headers.join(','),
    ...rows.map((row) =>
      headers.map((header) => escapeCsvValue(row[header])).join(',')
    ),
  ].join('\n')
  const blob = new Blob([`\uFEFF${body}`], {
    type: 'text/csv;charset=utf-8;',
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function escapeCsvValue(value: unknown) {
  const text = String(value ?? '')
  const formulaSafe = /^[=+\-@\t\r]/.test(text) ? `'${text}` : text
  const escaped = formulaSafe.replace(/"/g, '""')
  return /[",\n\r]/.test(escaped) ? `"${escaped}"` : escaped
}

function copyToClipboard(text: string) {
  return navigator.clipboard.writeText(text)
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

export function KeyQuery() {
  const { t } = useTranslation()
  const [key, setKey] = useState('')
  const [report, setReport] = useState<KeyQueryReport | null>(null)

  const queryMutation = useMutation({
    mutationFn: fetchKeyQueryReport,
    onSuccess: (data) => {
      setReport(data)
    },
    onError: (error) => {
      setReport(null)
      toast.error(error.message)
    },
  })

  const logs = report?.logs ?? EMPTY_LOGS
  const balance = report?.subscription.hard_limit_usd ?? null
  const usage = report ? report.usage.total_usage / 100 : null
  const expiredTime = report?.subscription.access_until ?? null
  const isUnlimited = balance === UNLIMITED_USD
  const totalLogQuota = useMemo(
    () =>
      logs.reduce((sum, log) => {
        if (log.type === LOG_TYPE_CONSUME || log.type === LOG_TYPE_REFUND) {
          return sum + (log.quota || 0)
        }
        return sum
      }, 0),
    [logs]
  )

  const runQuery = () => {
    queryMutation.mutate(key)
  }

  const copySummary = async () => {
    if (!report) return
    const remaining =
      balance != null && usage != null
        ? formatCurrencyUSD(balance - usage)
        : '-'
    const summary = [
      `${t('Total quota')}: ${
        isUnlimited ? t('Unlimited') : formatCurrencyUSD(balance)
      }`,
      `${t('Used quota')}: ${
        isUnlimited || usage == null ? '-' : formatCurrencyUSD(usage)
      }`,
      `${t('Remaining quota')}: ${isUnlimited ? t('Unlimited') : remaining}`,
      `${t('Expires at')}: ${
        expiredTime === 0 ? t('Never') : formatTimestampToDate(expiredTime ?? 0)
      }`,
    ].join('\n')

    try {
      await copyToClipboard(summary)
      toast.success(t('Copied'))
    } catch {
      toast.error(t('Copy failed'))
    }
  }

  const exportLogs = () => {
    if (logs.length === 0) return
    downloadCsv(
      `key-usage-${Date.now()}.csv`,
      logs.map((log) => ({
        Time: formatTimestampToDate(log.created_at),
        Type: getLogTypeLabel(log.type),
        Model: log.model_name || '-',
        Duration: log.use_time ? formatUseTime(log.use_time / 1000) : '-',
        Stream: log.is_stream ? 'yes' : 'no',
        PromptTokens: log.prompt_tokens || 0,
        CompletionTokens: log.completion_tokens || 0,
        Cost: formatLogQuota(log.quota),
      }))
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <div className='relative'>
        <div
          aria-hidden
          className='pointer-events-none absolute inset-x-0 top-0 h-[600px] opacity-20 dark:opacity-[0.10]'
          style={{
            background: [
              'radial-gradient(ellipse 60% 50% at 20% 20%, oklch(0.72 0.18 250 / 80%) 0%, transparent 70%)',
              'radial-gradient(ellipse 50% 40% at 80% 15%, oklch(0.65 0.15 200 / 60%) 0%, transparent 70%)',
              'radial-gradient(ellipse 40% 35% at 50% 70%, oklch(0.70 0.12 280 / 40%) 0%, transparent 70%)',
            ].join(', '),
            maskImage:
              'linear-gradient(to bottom, black 40%, transparent 100%)',
            WebkitMaskImage:
              'linear-gradient(to bottom, black 40%, transparent 100%)',
          }}
        />
        <main className='relative mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-6xl flex-col gap-4 px-3 pt-20 pb-8 sm:px-6 sm:pt-24 lg:px-8'>
          <header className='space-y-2'>
            <h1 className='text-2xl font-bold tracking-tight'>
              {t('Key Usage Query')}
            </h1>
            <p className='text-muted-foreground max-w-2xl text-sm'>
              {t('Query balance, recent usage, and token logs by API key.')}
            </p>
          </header>

          <Card size='sm'>
            <CardContent className='flex flex-col gap-2 sm:flex-row'>
              <div className='relative flex-1'>
                <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
                <Input
                  value={key}
                  className='pl-8'
                  placeholder='sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
                  onChange={(event) => setKey(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') runQuery()
                  }}
                />
              </div>
              <Button
                onClick={runQuery}
                disabled={queryMutation.isPending}
                className='sm:w-28'
              >
                {queryMutation.isPending ? t('Querying...') : t('Query')}
              </Button>
            </CardContent>
          </Card>

          {report && (
            <>
              <Card size='sm'>
                <CardHeader className='border-b'>
                  <div className='flex items-center justify-between gap-3'>
                    <CardTitle>{t('Key Information')}</CardTitle>
                    <Button variant='outline' onClick={copySummary}>
                      <Copy />
                      {t('Copy')}
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
                  <StatBlock
                    label={t('Total quota')}
                    value={
                      isUnlimited ? t('Unlimited') : formatCurrencyUSD(balance)
                    }
                  />
                  <StatBlock
                    label={t('Used quota')}
                    value={
                      isUnlimited || usage == null
                        ? '-'
                        : formatCurrencyUSD(usage)
                    }
                  />
                  <StatBlock
                    label={t('Remaining quota')}
                    value={
                      isUnlimited
                        ? t('Unlimited')
                        : balance == null || usage == null
                          ? '-'
                          : formatCurrencyUSD(balance - usage)
                    }
                  />
                  <StatBlock
                    label={t('Expires at')}
                    value={
                      expiredTime === 0
                        ? t('Never')
                        : formatTimestampToDate(expiredTime ?? 0)
                    }
                  />
                  <StatBlock
                    label={t('Recent log cost')}
                    value={formatLogQuota(totalLogQuota)}
                  />
                  <StatBlock
                    label={t('Recent calls')}
                    value={`${logs.length}`}
                  />
                </CardContent>
              </Card>

              <Card size='sm'>
                <CardHeader className='border-b'>
                  <div className='flex items-center justify-between gap-3'>
                    <CardTitle>{t('Call Details')}</CardTitle>
                    <Button
                      variant='outline'
                      onClick={exportLogs}
                      disabled={logs.length === 0}
                    >
                      <Download />
                      {t('Export CSV')}
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className='p-0'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Time')}</TableHead>
                        <TableHead>{t('Type')}</TableHead>
                        <TableHead>{t('Model')}</TableHead>
                        <TableHead>{t('Duration')}</TableHead>
                        <TableHead>{t('Prompt')}</TableHead>
                        <TableHead>{t('Completion')}</TableHead>
                        <TableHead>{t('Cost')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {logs.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={7} className='h-24 text-center'>
                            {t('No token logs')}
                          </TableCell>
                        </TableRow>
                      ) : (
                        logs.map((log) => (
                          <TableRow key={log.id}>
                            <TableCell>
                              {formatTimestampToDate(log.created_at)}
                            </TableCell>
                            <TableCell>
                              <Badge variant={getLogTypeVariant(log.type)}>
                                {t(getLogTypeLabel(log.type))}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              {log.model_name ? (
                                <Badge variant='outline'>
                                  {log.model_name}
                                </Badge>
                              ) : (
                                <span className='text-muted-foreground'>-</span>
                              )}
                            </TableCell>
                            <TableCell>
                              {isApiCallLog(log) && log.use_time
                                ? formatUseTime(log.use_time / 1000)
                                : '-'}
                              {isApiCallLog(log) && (
                                <Badge variant='secondary' className='ml-2'>
                                  {log.is_stream
                                    ? t('Stream')
                                    : t('Non-stream')}
                                </Badge>
                              )}
                            </TableCell>
                            <TableCell>
                              {isApiCallLog(log) && log.prompt_tokens
                                ? log.prompt_tokens
                                : '-'}
                            </TableCell>
                            <TableCell>
                              {isApiCallLog(log) && log.completion_tokens
                                ? log.completion_tokens
                                : '-'}
                            </TableCell>
                            <TableCell>
                              {isApiCallLog(log)
                                ? formatLogQuota(log.quota)
                                : '-'}
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </>
          )}
        </main>
      </div>
    </PublicLayout>
  )
}
