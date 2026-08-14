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
import { type Table } from '@tanstack/react-table'
import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { CopyButton } from '@/components/copy-button'
import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { deleteInvalidRedemptions, getRedemptionKey } from '../api'
import { type Redemption } from '../types'
import { useRedemptions } from './redemptions-provider'

type DataTableBulkActionsProps<TData> = {
  table: Table<TData>
}

export function DataTableBulkActions<TData>({
  table,
}: DataTableBulkActionsProps<TData>) {
  const { t } = useTranslation()
  const { triggerRefresh } = useRedemptions()
  const { copyToClipboard } = useCopyToClipboard({ notify: false })
  const [showDeleteInvalidConfirm, setShowDeleteInvalidConfirm] =
    useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isCopying, setIsCopying] = useState(false)
  const selectedRows = table.getFilteredSelectedRowModel().rows

  const handleCopySelected = async (): Promise<string> => {
    if (isCopying) return ''
    setIsCopying(true)
    try {
      const lines: string[] = []
      for (const row of selectedRows) {
        const redemption = row.original as Redemption
        try {
          const res = await getRedemptionKey(redemption.id)
          if (!res.success || !res.data?.key) {
            throw new Error(res.message)
          }
          lines.push(`${redemption.name}\t${res.data.key}`)
        } catch {
          toast.error(
            t('redemptionCodes.status.failedToFetchKeyFor', {
              name: redemption.name,
            })
          )
        }
      }
      const content = lines.join('\n')
      if (content) {
        await copyToClipboard(content)
        toast.success(t('redemptionCodes.status.codesCopied'))
      }
      return content
    } finally {
      setIsCopying(false)
    }
  }

  const handleDeleteInvalid = async () => {
    setIsDeleting(true)
    try {
      const result = await deleteInvalidRedemptions()

      if (result.success) {
        const count = result.data || 0
        toast.success(
          t('redemptionCodes.status.successfullyDeletedCountInvalidRedemptionCodes', {
            count,
          })
        )
        table.resetRowSelection()
        triggerRefresh()
        setShowDeleteInvalidConfirm(false)
      }
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <>
      <BulkActionsToolbar table={table} entityName={t('redemptionCodes.fields.codes')}>
        {isCopying ? (
          <span className='text-muted-foreground inline-flex h-8 w-8 items-center justify-center text-xs'>
            …
          </span>
        ) : (
          <CopyButton
            value=''
            onBeforeCopy={handleCopySelected}
            variant='outline'
            size='icon'
            className='size-8'
            tooltip={t('redemptionCodes.actions.copySelectedCodes')}
            successTooltip={t('redemptionCodes.status.codesCopied')}
            aria-label={t('redemptionCodes.actions.copySelectedCodes')}
          />
        )}

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant='destructive'
                size='icon'
                onClick={() => setShowDeleteInvalidConfirm(true)}
                className='size-8'
                aria-label={t('redemptionCodes.actions.deleteInvalidRedemptionCodes')}
                title={t('redemptionCodes.actions.deleteInvalidRedemptionCodes')}
              />
            }
          >
            <Trash2 />
            <span className='sr-only'>{t('redemptionCodes.actions.deleteInvalidCodes')}</span>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t('redemptionCodes.actions.deleteInvalidCodesUsedDisabledExpired')}</p>
          </TooltipContent>
        </Tooltip>
      </BulkActionsToolbar>

      <ConfirmDialog
        destructive
        open={showDeleteInvalidConfirm}
        onOpenChange={setShowDeleteInvalidConfirm}
        handleConfirm={handleDeleteInvalid}
        isLoading={isDeleting}
        className='max-w-md'
        title={t('redemptionCodes.actions.deleteInvalidRedemptionCodesfa2061')}
        desc={
          <>
            {t('redemptionCodes.tips.deleteAll')} <strong>{t('common.status.used')}</strong>,{' '}
            <strong>{t('channels.status.disabled')}</strong>
            {t('redemptionCodes.fields.value')} <strong>{t('redemptionCodes.status.expired')}</strong>{' '}
            {t('redemptionCodes.tips.codes')}
            <br />
            {t('keys.errors.actionCannotBeUndone951f49')}
          </>
        }
        confirmText={t('redemptionCodes.actions.deleteInvalid')}
      />
    </>
  )
}
