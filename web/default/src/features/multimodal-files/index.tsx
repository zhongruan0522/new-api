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
import { Copy, Eye, RefreshCw, Trash2 } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import dayjs from '@/lib/dayjs'
import { formatTimestampToDate } from '@/lib/format'
import { SectionPageLayout } from '@/components/layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import {
  batchDeleteStoredMedia,
  deleteStoredMedia,
  getStoredMedia,
  getStoredMediaDetail,
} from './api'
import type { StoredMediaBatchItem, StoredMediaItem } from './types'

const DEFAULT_PAGE_SIZE = 20
const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

type DeleteTarget =
  | { mode: 'single'; item: StoredMediaItem }
  | { mode: 'batch'; items: StoredMediaBatchItem[] }

function toInputDateTime(date: Date) {
  return dayjs(date).format('YYYY-MM-DDTHH:mm')
}

function toUnixSeconds(value: string) {
  const ms = Date.parse(value)
  if (!Number.isFinite(ms)) return 0
  return Math.floor(ms / 1000)
}

function getDefaultDateRange() {
  const now = new Date()
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  const end = new Date(now.getTime() + 60 * 60 * 1000)
  return {
    start: toInputDateTime(start),
    end: toInputDateTime(end),
  }
}

function getMediaKey(item: StoredMediaItem | StoredMediaBatchItem) {
  return `${item.media_type}:${item.id}`
}

function formatSize(size: number) {
  if (!Number.isFinite(size) || size <= 0) return '-'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(2)} MB`
}

function copyToClipboard(text: string) {
  return navigator.clipboard.writeText(text)
}

export function MultimodalFiles() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const authUser = useAuthStore((state) => state.auth.user)
  const isAdmin = (authUser?.role ?? 0) >= ROLE.ADMIN
  const defaultRange = useMemo(() => getDefaultDateRange(), [])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [startTime, setStartTime] = useState(defaultRange.start)
  const [endTime, setEndTime] = useState(defaultRange.end)
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailItem, setDetailItem] = useState<StoredMediaItem | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null)

  const startTimestamp = toUnixSeconds(startTime)
  const endTimestamp = toUnixSeconds(endTime)

  const queryKey = [
    'stored-media',
    isAdmin,
    page,
    pageSize,
    startTimestamp,
    endTimestamp,
  ] as const

  const mediaQuery = useQuery({
    queryKey,
    queryFn: () =>
      getStoredMedia({
        page,
        pageSize,
        startTimestamp,
        endTimestamp,
        isAdmin,
      }),
  })

  const items = mediaQuery.data?.items ?? []
  const total = mediaQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const selectedItems = useMemo(
    () =>
      items.filter((item) => selectedKeys.includes(getMediaKey(item))).map(
        (item) => ({
          id: item.id,
          media_type: item.media_type,
        })
      ),
    [items, selectedKeys]
  )

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ['stored-media'] })
  }

  const detailMutation = useMutation({
    mutationFn: (item: StoredMediaItem) =>
      getStoredMediaDetail(item.media_type, item.id),
    onSuccess: (item) => {
      setDetailItem(item)
      setDetailOpen(true)
    },
    onError: (error) => toast.error(error.message),
  })

  const deleteMutation = useMutation({
    mutationFn: async (target: DeleteTarget) => {
      if (target.mode === 'single') {
        return deleteStoredMedia(target.item.media_type, target.item.id)
      }
      return batchDeleteStoredMedia(target.items)
    },
    onSuccess: async (deleted) => {
      toast.success(t('Deleted {{count}} file(s)', { count: deleted }))
      setDeleteTarget(null)
      setSelectedKeys([])
      await refresh()
    },
    onError: (error) => toast.error(error.message),
  })

  const toggleSelection = (item: StoredMediaItem, checked: boolean) => {
    const key = getMediaKey(item)
    setSelectedKeys((current) => {
      if (!checked) return current.filter((itemKey) => itemKey !== key)
      const next = new Set(current)
      next.add(key)
      return Array.from(next)
    })
  }

  const allVisibleSelected =
    items.length > 0 &&
    items.every((item) => selectedKeys.includes(getMediaKey(item)))

  const toggleVisible = (checked: boolean) => {
    if (!checked) {
      const visible = new Set(items.map(getMediaKey))
      setSelectedKeys((current) => current.filter((key) => !visible.has(key)))
      return
    }
    setSelectedKeys((current) => {
      const next = new Set(current)
      items.forEach((item) => next.add(getMediaKey(item)))
      return Array.from(next)
    })
  }

  const copyUrl = async (url: string) => {
    if (!url) return
    try {
      await copyToClipboard(url)
      toast.success(t('Copied'))
    } catch {
      toast.error(t('Copy failed'))
    }
  }

  const resetFilters = () => {
    setStartTime(defaultRange.start)
    setEndTime(defaultRange.end)
    setPage(1)
    setSelectedKeys([])
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Multimodal Files')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button
            variant='outline'
            disabled={selectedItems.length === 0}
            onClick={() => {
              if (selectedItems.length > 0) {
                setDeleteTarget({ mode: 'batch', items: selectedItems })
              }
            }}
          >
            <Trash2 />
            {t('Delete selected')}
          </Button>
          <Button variant='outline' onClick={() => void refresh()}>
            <RefreshCw />
            {t('Refresh')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            {mediaQuery.error instanceof Error && (
              <div className='border-destructive/40 text-destructive rounded-lg border px-3 py-2 text-sm'>
                {mediaQuery.error.message}
              </div>
            )}

            <Card size='sm'>
              <CardContent className='grid gap-3 lg:grid-cols-[1fr_1fr_auto]'>
                <div className='grid gap-1.5'>
                  <Label htmlFor='stored-media-start'>{t('Start time')}</Label>
                  <Input
                    id='stored-media-start'
                    type='datetime-local'
                    value={startTime}
                    onChange={(event) => {
                      setStartTime(event.target.value)
                      setPage(1)
                      setSelectedKeys([])
                    }}
                  />
                </div>
                <div className='grid gap-1.5'>
                  <Label htmlFor='stored-media-end'>{t('End time')}</Label>
                  <Input
                    id='stored-media-end'
                    type='datetime-local'
                    value={endTime}
                    onChange={(event) => {
                      setEndTime(event.target.value)
                      setPage(1)
                      setSelectedKeys([])
                    }}
                  />
                </div>
                <div className='flex items-end gap-2'>
                  <Button onClick={() => void refresh()} className='flex-1'>
                    {t('Query')}
                  </Button>
                  <Button variant='outline' onClick={resetFilters}>
                    {t('Reset')}
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card size='sm'>
              <CardContent className='p-0'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className='w-10'>
                        <Checkbox
                          aria-label={t('Select visible files')}
                          checked={allVisibleSelected}
                          onCheckedChange={(checked) =>
                            toggleVisible(checked === true)
                          }
                        />
                      </TableHead>
                      <TableHead>{t('Type')}</TableHead>
                      <TableHead>{t('ID')}</TableHead>
                      <TableHead>{t('Created at')}</TableHead>
                      <TableHead>{t('MIME')}</TableHead>
                      <TableHead>{t('Size')}</TableHead>
                      <TableHead>{t('Converted URL')}</TableHead>
                      <TableHead className='w-36 text-right'>
                        {t('Actions')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {mediaQuery.isLoading ? (
                      <TableRow>
                        <TableCell colSpan={8} className='h-24 text-center'>
                          {t('Loading...')}
                        </TableCell>
                      </TableRow>
                    ) : items.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={8} className='h-24 text-center'>
                          {t('No multimodal files')}
                        </TableCell>
                      </TableRow>
                    ) : (
                      items.map((item) => {
                        const key = getMediaKey(item)
                        return (
                          <TableRow key={key}>
                            <TableCell>
                              <Checkbox
                                aria-label={t('Select file')}
                                checked={selectedKeys.includes(key)}
                                onCheckedChange={(checked) =>
                                  toggleSelection(item, checked === true)
                                }
                              />
                            </TableCell>
                            <TableCell>
                              <Badge variant='outline'>
                                {t(item.media_type)}
                              </Badge>
                            </TableCell>
                            <TableCell className='max-w-48 truncate font-mono text-xs'>
                              {item.id}
                            </TableCell>
                            <TableCell>
                              {formatTimestampToDate(item.created_at)}
                            </TableCell>
                            <TableCell>{item.mime_type || '-'}</TableCell>
                            <TableCell>{formatSize(item.size_bytes)}</TableCell>
                            <TableCell className='max-w-72 truncate'>
                              {item.url || '-'}
                            </TableCell>
                            <TableCell>
                              <div className='flex justify-end gap-1'>
                                <Button
                                  size='icon-sm'
                                  variant='ghost'
                                  disabled={detailMutation.isPending}
                                  onClick={() => detailMutation.mutate(item)}
                                >
                                  <Eye />
                                  <span className='sr-only'>{t('View')}</span>
                                </Button>
                                <Button
                                  size='icon-sm'
                                  variant='ghost'
                                  disabled={!item.url}
                                  onClick={() => void copyUrl(item.url)}
                                >
                                  <Copy />
                                  <span className='sr-only'>{t('Copy')}</span>
                                </Button>
                                <Button
                                  size='icon-sm'
                                  variant='destructive'
                                  onClick={() =>
                                    setDeleteTarget({ mode: 'single', item })
                                  }
                                >
                                  <Trash2 />
                                  <span className='sr-only'>{t('Delete')}</span>
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        )
                      })
                    )}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>

            <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
              <div className='text-muted-foreground text-sm'>
                {t('Selected {{selected}} of {{total}}', {
                  selected: selectedItems.length,
                  total,
                })}
              </div>
              <div className='flex items-center gap-2'>
                <Select
                  value={String(pageSize)}
                  onValueChange={(value) => {
                    const next = Number(value)
                    if (Number.isFinite(next)) {
                      setPageSize(next)
                      setPage(1)
                      setSelectedKeys([])
                    }
                  }}
                >
                  <SelectTrigger className='w-24'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <SelectItem key={size} value={String(size)}>
                          {size}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <Button
                  variant='outline'
                  disabled={page <= 1}
                  onClick={() => {
                    setSelectedKeys([])
                    setPage((current) => Math.max(1, current - 1))
                  }}
                >
                  {t('Previous')}
                </Button>
                <span className='text-sm tabular-nums'>
                  {page} / {totalPages}
                </span>
                <Button
                  variant='outline'
                  disabled={page >= totalPages}
                  onClick={() => {
                    setSelectedKeys([])
                    setPage((current) => Math.min(totalPages, current + 1))
                  }}
                >
                  {t('Next')}
                </Button>
              </div>
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
        <DialogContent className='sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>{t('Multimodal File')}</DialogTitle>
          </DialogHeader>
          {detailItem && (
            <div className='space-y-3'>
              <div className='grid gap-2 text-sm sm:grid-cols-2'>
                <div>
                  <span className='text-muted-foreground'>{t('ID')}: </span>
                  <span className='font-mono text-xs'>{detailItem.id}</span>
                </div>
                <div>
                  <span className='text-muted-foreground'>
                    {t('Created at')}:{' '}
                  </span>
                  {formatTimestampToDate(detailItem.created_at)}
                </div>
                <div>
                  <span className='text-muted-foreground'>{t('Type')}: </span>
                  {detailItem.media_type}
                </div>
                <div>
                  <span className='text-muted-foreground'>{t('Size')}: </span>
                  {formatSize(detailItem.size_bytes)}
                </div>
              </div>
              <div className='grid gap-1.5'>
                <Label htmlFor='stored-media-url'>{t('Converted URL')}</Label>
                <Input id='stored-media-url' value={detailItem.url} readOnly />
              </div>
              {detailItem.url && detailItem.media_type === 'image' && (
                <div className='flex justify-center'>
                  <img
                    src={detailItem.url}
                    alt={detailItem.id}
                    className='max-h-80 max-w-full rounded-lg border object-contain'
                  />
                </div>
              )}
              {detailItem.url && detailItem.media_type === 'video' && (
                <div className='flex justify-center'>
                  <video
                    src={detailItem.url}
                    controls
                    className='max-h-80 max-w-full rounded-lg border'
                  />
                </div>
              )}
            </div>
          )}
          <DialogFooter>
            <Button variant='outline' onClick={() => setDetailOpen(false)}>
              {t('Close')}
            </Button>
            <Button
              disabled={!detailItem?.url}
              onClick={() => detailItem && void copyUrl(detailItem.url)}
            >
              <Copy />
              {t('Copy URL')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={deleteTarget != null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Delete file')}</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteTarget?.mode === 'batch'
                ? t('Delete {{count}} selected file(s)?', {
                    count: deleteTarget.items.length,
                  })
                : t('Delete this multimodal file?')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (deleteTarget) deleteMutation.mutate(deleteTarget)
              }}
            >
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
