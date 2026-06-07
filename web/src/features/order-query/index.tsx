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

import { useState } from 'react'
import {
  Check,
  ChevronLeft,
  ChevronRight,
  Copy,
  Loader2,
  RefreshCw,
  Search,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { formatNumber } from '@/lib/format'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { useBillingHistory } from '@/features/wallet/hooks/use-billing-history'
import {
  formatTimestamp,
  getPaymentMethodName,
} from '@/features/wallet/lib/billing'
import type { TopupRecord } from '@/features/wallet/types'

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

function getStatusMeta(status: string) {
  if (status === 'success') {
    return { label: 'Success', variant: 'success' as const }
  }
  if (status === 'expired') {
    return { label: 'Expired', variant: 'danger' as const }
  }
  if (status === 'failed') {
    return { label: 'Failed', variant: 'danger' as const }
  }
  return { label: 'Pending', variant: 'warning' as const }
}

function OrderNumberCell({ record }: { record: TopupRecord }) {
  const { t } = useTranslation()
  const { copyToClipboard, copiedText } = useCopyToClipboard({ notify: false })

  return (
    <div className='flex min-w-0 items-center gap-2'>
      <code className='truncate font-mono text-sm'>{record.trade_no}</code>
      <Button
        variant='ghost'
        size='icon-xs'
        aria-label={t('Copy order number')}
        onClick={() => copyToClipboard(record.trade_no)}
      >
        {copiedText === record.trade_no ? (
          <Check className='size-3' />
        ) : (
          <Copy className='size-3' />
        )}
      </Button>
    </div>
  )
}

export function OrderQuery() {
  const { t } = useTranslation()
  const {
    records,
    total,
    page,
    pageSize,
    keyword,
    loading,
    completing,
    isAdmin,
    handlePageChange,
    handlePageSizeChange,
    handleSearch,
    handleCompleteOrder,
    refresh,
  } = useBillingHistory()
  const [confirmTradeNo, setConfirmTradeNo] = useState<string | null>(null)
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const confirmComplete = async () => {
    if (!confirmTradeNo) return
    const ok = await handleCompleteOrder(confirmTradeNo)
    if (ok) setConfirmTradeNo(null)
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Order Query')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button variant='outline' onClick={() => refresh()}>
            <RefreshCw className={loading ? 'size-4 animate-spin' : 'size-4'} />
            {t('Refresh')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <Card size='sm'>
            <CardContent className='space-y-4'>
              <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                <div className='relative sm:w-96'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
                  <Input
                    value={keyword}
                    className='pl-8'
                    placeholder={t('Search by order number...')}
                    onChange={(event) => handleSearch(event.target.value)}
                  />
                </div>
                <Select
                  value={String(pageSize)}
                  onValueChange={(value) => {
                    const next = Number(value)
                    if (Number.isFinite(next)) handlePageSizeChange(next)
                  }}
                >
                  <SelectTrigger className='w-28'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <SelectItem key={size} value={String(size)}>
                          {size}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <div className='overflow-x-auto rounded-lg border'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className='min-w-64'>
                        {t('Order number')}
                      </TableHead>
                      {isAdmin ? (
                        <TableHead className='min-w-28'>
                          {t('User ID')}
                        </TableHead>
                      ) : null}
                      <TableHead className='min-w-36'>
                        {t('Payment Method')}
                      </TableHead>
                      <TableHead className='min-w-32'>{t('Amount')}</TableHead>
                      <TableHead className='min-w-32'>{t('Payment')}</TableHead>
                      <TableHead className='min-w-32'>{t('Status')}</TableHead>
                      <TableHead className='min-w-40'>
                        {t('Created at')}
                      </TableHead>
                      {isAdmin ? (
                        <TableHead className='w-32 text-right'>
                          {t('Actions')}
                        </TableHead>
                      ) : null}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {loading ? (
                      <TableRow>
                        <TableCell
                          colSpan={isAdmin ? 8 : 6}
                          className='text-muted-foreground h-36 text-center'
                        >
                          <div className='inline-flex items-center gap-2'>
                            <Loader2 className='size-4 animate-spin' />
                            {t('Loading...')}
                          </div>
                        </TableCell>
                      </TableRow>
                    ) : records.length === 0 ? (
                      <TableRow>
                        <TableCell
                          colSpan={isAdmin ? 8 : 6}
                          className='text-muted-foreground h-36 text-center'
                        >
                          {t('No orders found')}
                        </TableCell>
                      </TableRow>
                    ) : (
                      records.map((record) => {
                        const status = getStatusMeta(record.status)
                        return (
                          <TableRow key={record.id}>
                            <TableCell>
                              <OrderNumberCell record={record} />
                            </TableCell>
                            {isAdmin ? (
                              <TableCell>{record.user_id}</TableCell>
                            ) : null}
                            <TableCell>
                              {getPaymentMethodName(record.payment_method, t)}
                            </TableCell>
                            <TableCell>
                              {formatCurrencyFromUSD(record.amount, {
                                digitsLarge: 2,
                                digitsSmall: 2,
                                abbreviate: false,
                              })}
                            </TableCell>
                            <TableCell>{formatNumber(record.money)}</TableCell>
                            <TableCell>
                              <StatusBadge
                                label={t(status.label)}
                                variant={status.variant}
                                showDot
                                copyable={false}
                              />
                            </TableCell>
                            <TableCell>
                              {formatTimestamp(record.create_time)}
                            </TableCell>
                            {isAdmin ? (
                              <TableCell className='text-right'>
                                {record.status === 'pending' ? (
                                  <Button
                                    variant='outline'
                                    size='sm'
                                    disabled={completing}
                                    onClick={() =>
                                      setConfirmTradeNo(record.trade_no)
                                    }
                                  >
                                    {t('Complete Order')}
                                  </Button>
                                ) : null}
                              </TableCell>
                            ) : null}
                          </TableRow>
                        )
                      })
                    )}
                  </TableBody>
                </Table>
              </div>

              <div className='flex flex-col gap-2 text-sm sm:flex-row sm:items-center sm:justify-between'>
                <div className='text-muted-foreground'>
                  {t('Total')}: {total}
                </div>
                <div className='flex items-center gap-2'>
                  <Button
                    variant='outline'
                    size='icon-sm'
                    disabled={page <= 1}
                    onClick={() => handlePageChange(page - 1)}
                  >
                    <ChevronLeft className='size-4' />
                  </Button>
                  <span className='text-muted-foreground min-w-16 text-center'>
                    {page} / {totalPages}
                  </span>
                  <Button
                    variant='outline'
                    size='icon-sm'
                    disabled={page >= totalPages}
                    onClick={() => handlePageChange(page + 1)}
                  >
                    <ChevronRight className='size-4' />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <ConfirmDialog
        open={Boolean(confirmTradeNo)}
        onOpenChange={(open) => {
          if (!open && !completing) setConfirmTradeNo(null)
        }}
        title={t('Complete Order')}
        desc={t(
          'Are you sure you want to manually complete this order? The user will be credited with the corresponding quota.'
        )}
        confirmText={t('Complete Order')}
        isLoading={completing}
        handleConfirm={confirmComplete}
      />
    </>
  )
}
