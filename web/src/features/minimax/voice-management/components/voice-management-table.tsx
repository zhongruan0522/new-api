/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, version 3 of the License.
*/
import { useMemo, type ReactNode } from 'react'
import {
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type PaginationState,
} from '@tanstack/react-table'
import { Edit, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { TableCell, TableRow } from '@/components/ui/table'
import { DataTablePage } from '@/components/data-table'
import type { VoiceRecord } from '../api'

type VoiceManagementTableProps = {
  items: VoiceRecord[]
  total: number
  isLoading: boolean
  isFetching: boolean
  isRoot: boolean
  pagination: PaginationState
  onPaginationChange: OnChangeFn<PaginationState>
  onEdit: (voice: VoiceRecord) => void
  onRequestDelete: (voice: VoiceRecord) => void
  toolbar: ReactNode
}

function getVoiceTypeLabelKey(type: string): string {
  if (type === 'created') return 'Voice Status - Paid'
  if (type === 'preview') return 'Voice Status - Preview'
  return type || '-'
}

function getOperatorKindLabelKey(kind: string): string {
  if (kind === 'admin') return 'Admin'
  if (kind === 'user') return 'User'
  return kind
}

function MonoValue({ value }: { value?: string }) {
  if (!value) return <span className='text-muted-foreground'>-</span>

  return (
    <span
      className='block max-w-[260px] truncate font-mono text-xs'
      title={value}
    >
      {value}
    </span>
  )
}

export function VoiceManagementTable(props: VoiceManagementTableProps) {
  const { t } = useTranslation()

  const columns = useMemo<ColumnDef<VoiceRecord>[]>(() => {
    const baseColumns: ColumnDef<VoiceRecord>[] = [
      {
        accessorKey: 'created_at',
        header: t('Time'),
        size: 170,
        cell: ({ row }) => (
          <span className='whitespace-nowrap'>
            {formatTimestampToDate(row.original.created_at)}
          </span>
        ),
      },
      {
        accessorKey: 'type',
        header: t('Type'),
        size: 110,
        cell: ({ row }) => (
          <Badge variant='secondary'>
            {t(getVoiceTypeLabelKey(row.original.type))}
          </Badge>
        ),
      },
      {
        accessorKey: 'operator_id',
        header: t('Operator ID'),
        size: 150,
        cell: ({ row }) => (
          <div className='space-y-0.5'>
            <div className='font-medium'>{row.original.operator_id}</div>
            {row.original.operator_kind && (
              <div className='text-muted-foreground text-xs'>
                {t(getOperatorKindLabelKey(row.original.operator_kind))}
              </div>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'voice_id',
        header: t('Voice ID'),
        size: 230,
        cell: ({ row }) => <MonoValue value={row.original.voice_id} />,
      },
      {
        accessorKey: 'quota_cost',
        header: t('Cost'),
        size: 130,
        cell: ({ row }) => (
          <span className='whitespace-nowrap'>
            {formatQuota(row.original.quota_cost)}
          </span>
        ),
      },
      {
        accessorKey: 'redirect_id',
        header: t('Redirect ID'),
        size: 230,
        cell: ({ row }) => <MonoValue value={row.original.redirect_id} />,
      },
      {
        accessorKey: 'allowed',
        header: t('Whitelist'),
        size: 120,
        cell: ({ row }) => (
          <Badge variant={row.original.allowed ? 'default' : 'outline'}>
            {row.original.allowed ? t('Allowed') : t('Not Allowed')}
          </Badge>
        ),
      },
      {
        accessorKey: 'remark',
        header: t('Remark'),
        size: 220,
        cell: ({ row }) => (
          <span
            className='block max-w-[260px] truncate text-sm'
            title={row.original.remark || undefined}
          >
            {row.original.remark || '-'}
          </span>
        ),
      },
    ]

    if (!props.isRoot) return baseColumns

    return [
      ...baseColumns,
      {
        id: 'actions',
        header: t('Actions'),
        size: 150,
        cell: ({ row }) => (
          <div className='flex items-center justify-end gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => props.onEdit(row.original)}
            >
              <Edit />
              {t('Edit')}
            </Button>
            <Button
              variant='destructive'
              size='sm'
              onClick={() => props.onRequestDelete(row.original)}
            >
              <Trash2 />
              {t('Delete')}
            </Button>
          </div>
        ),
      },
    ]
  }, [props, t])

  const table = useReactTable({
    data: props.items,
    columns,
    state: {
      pagination: props.pagination,
    },
    manualPagination: true,
    pageCount: Math.max(1, Math.ceil(props.total / props.pagination.pageSize)),
    onPaginationChange: props.onPaginationChange,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={props.isLoading}
      isFetching={props.isFetching}
      emptyTitle={t('No voices found')}
      emptyDescription={t(
        'Voice records will appear here after users create voices or admins add them.'
      )}
      skeletonKeyPrefix='minimax-voice-skeleton'
      tableClassName='overflow-x-auto'
      tableHeaderClassName='bg-muted/30 sticky top-0 z-10'
      toolbar={props.toolbar}
      applyHeaderSize
      renderRow={(row) => (
        <TableRow key={row.id} className='transition-colors'>
          {row.getVisibleCells().map((cell) => (
            <TableCell
              key={cell.id}
              className={cn(cell.column.id === 'actions' && 'text-right')}
            >
              {flexRender(cell.column.columnDef.cell, cell.getContext())}
            </TableCell>
          ))}
        </TableRow>
      )}
    />
  )
}
