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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Edit, Loader2, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getLobeIcon } from '@/lib/lobe-icon'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { StatusBadge } from '@/components/status-badge'
import { getVendors } from '../../api'
import { handleDeleteVendor, vendorsQueryKeys } from '../../lib'
import type { Vendor } from '../../types'
import { VendorMutateDialog } from './vendor-mutate-dialog'

type VendorManagementDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function VendorManagementDialog({
  open,
  onOpenChange,
}: VendorManagementDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [mutateOpen, setMutateOpen] = useState(false)
  const [currentVendor, setCurrentVendor] = useState<Vendor | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Vendor | null>(null)
  const [deleting, setDeleting] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: vendorsQueryKeys.list({ page_size: 1000 }),
    queryFn: () => getVendors({ page_size: 1000 }),
    enabled: open,
  })

  const vendors = data?.data?.items || []

  const openCreate = () => {
    setCurrentVendor(null)
    setMutateOpen(true)
  }

  const openEdit = (vendor: Vendor) => {
    setCurrentVendor(vendor)
    setMutateOpen(true)
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await handleDeleteVendor(deleteTarget.id, queryClient, () => {
        queryClient.invalidateQueries({ queryKey: ['pricing'] })
        setDeleteTarget(null)
      })
    } finally {
      setDeleting(false)
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className='max-h-[85vh] overflow-hidden p-0 sm:max-w-4xl'>
          <DialogHeader className='border-b px-6 py-4'>
            <div className='flex items-start justify-between gap-4'>
              <div>
                <DialogTitle>{t('Manage Vendors')}</DialogTitle>
                <DialogDescription>
                  {t(
                    'Configure model vendors, icons, descriptions, and data privacy metadata.'
                  )}
                </DialogDescription>
              </div>
              <Button size='sm' onClick={openCreate}>
                <Plus className='h-4 w-4' />
                {t('Add Vendor')}
              </Button>
            </div>
          </DialogHeader>

          <div className='max-h-[65vh] overflow-auto px-6 py-4'>
            {isLoading ? (
              <div className='flex h-40 items-center justify-center'>
                <Loader2 className='text-muted-foreground h-5 w-5 animate-spin' />
              </div>
            ) : vendors.length === 0 ? (
              <div className='border-border/70 bg-muted/20 flex h-40 flex-col items-center justify-center gap-3 rounded-md border'>
                <p className='text-muted-foreground text-sm'>
                  {t('No vendors configured yet.')}
                </p>
                <Button size='sm' onClick={openCreate}>
                  <Plus className='h-4 w-4' />
                  {t('Add Vendor')}
                </Button>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Vendor')}</TableHead>
                    <TableHead>{t('Description')}</TableHead>
                    <TableHead>{t('Data retention')}</TableHead>
                    <TableHead>{t('Training opt-out')}</TableHead>
                    <TableHead className='w-[120px] text-right'>
                      {t('Actions')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {vendors.map((vendor) => (
                    <TableRow key={vendor.id}>
                      <TableCell>
                        <div className='flex min-w-0 items-center gap-2'>
                          <span className='flex h-8 w-8 shrink-0 items-center justify-center rounded-md border bg-background'>
                            {getLobeIcon(vendor.icon || vendor.name, 18)}
                          </span>
                          <div className='min-w-0'>
                            <div className='truncate text-sm font-medium'>
                              {vendor.name}
                            </div>
                            <div className='text-muted-foreground truncate text-xs'>
                              {vendor.icon || '-'}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className='max-w-[280px]'>
                        <span className='text-muted-foreground line-clamp-2 whitespace-normal'>
                          {vendor.description || t('No description provided')}
                        </span>
                      </TableCell>
                      <TableCell>
                        {typeof vendor.data_retention_days === 'number' ? (
                          <StatusBadge
                            label={
                              vendor.data_retention_days === 0
                                ? t('Zero retention')
                                : `${vendor.data_retention_days} ${t('days')}`
                            }
                            variant={
                              vendor.data_retention_days === 0
                                ? 'success'
                                : 'neutral'
                            }
                            copyable={false}
                          />
                        ) : (
                          <span className='text-muted-foreground text-sm'>
                            {t('Provider-specific')}
                          </span>
                        )}
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          label={
                            vendor.training_opt_out === true
                              ? t('Enabled')
                              : t('Unknown')
                          }
                          variant={
                            vendor.training_opt_out === true
                              ? 'success'
                              : 'neutral'
                          }
                          copyable={false}
                        />
                      </TableCell>
                      <TableCell>
                        <div className='flex justify-end gap-1'>
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon'
                            onClick={() => openEdit(vendor)}
                            aria-label={t('Edit')}
                          >
                            <Edit className='h-4 w-4' />
                          </Button>
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon'
                            className='text-destructive hover:text-destructive'
                            onClick={() => setDeleteTarget(vendor)}
                            aria-label={t('Delete')}
                          >
                            <Trash2 className='h-4 w-4' />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        </DialogContent>
      </Dialog>

      <VendorMutateDialog
        open={mutateOpen}
        onOpenChange={setMutateOpen}
        currentVendor={currentVendor}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(next) => !next && setDeleteTarget(null)}
        title={t('Delete vendor')}
        desc={t('This vendor will be removed from the list.')}
        destructive
        isLoading={deleting}
        confirmText={t('Delete')}
        handleConfirm={confirmDelete}
      />
    </>
  )
}
