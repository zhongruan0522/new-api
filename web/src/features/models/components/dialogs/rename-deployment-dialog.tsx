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
import { useEffect, useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { checkClusterNameAvailability, updateDeploymentName } from '../../api'
import { deploymentsQueryKeys } from '../../lib'

export function RenameDeploymentDialog({
  open,
  onOpenChange,
  deploymentId,
  currentName,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  deploymentId: string | number | null
  currentName?: string
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [name, setName] = useState(currentName || '')
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (open) setName(currentName || '')
  }, [open, currentName])

  const trimmed = name.trim()

  const { data: checkRes, isFetching: isChecking } = useQuery({
    queryKey: ['deployment-rename-check', trimmed],
    queryFn: () => (trimmed ? checkClusterNameAvailability(trimmed) : null),
    enabled: open && Boolean(trimmed),
    staleTime: 10_000,
  })

  const available =
    checkRes?.success === true ? checkRes?.data?.available : undefined

  const helper = useMemo(() => {
    if (!trimmed) return t('models.placeholders.enterANewName')
    if (isChecking) return t('models.tips.checkingName')
    if (available === true) return t('models.fields.nameIsAvailable')
    if (available === false) return t('models.fields.nameIsNotAvailable')
    return ''
  }, [available, isChecking, t, trimmed])

  const canSubmit =
    Boolean(deploymentId) &&
    Boolean(trimmed) &&
    available !== false &&
    !isSubmitting

  const onSubmit = async () => {
    if (!deploymentId) return
    if (!trimmed) {
      toast.error(t('keys.errors.pleaseEnterAName'))
      return
    }
    if (available === false) {
      toast.error(t('models.fields.nameIsNotAvailable'))
      return
    }

    setIsSubmitting(true)
    try {
      const res = await updateDeploymentName(deploymentId, trimmed)
      if (res.success) {
        toast.success(t('models.fields.renamedSuccessfully'))
        queryClient.invalidateQueries({
          queryKey: deploymentsQueryKeys.lists(),
        })
        queryClient.invalidateQueries({ queryKey: ['deployment-details'] })
        onOpenChange(false)
        return
      }
      toast.error(res.message || t('models.status.renameFailed'))
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('models.status.renameFailed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('models.fields.renameDeployment')}</DialogTitle>
        </DialogHeader>

        <div className='space-y-2'>
          <div className='text-muted-foreground text-sm'>
            {t('channels.fields.deploymentId')}:{' '}
            <span className='font-mono'>{deploymentId}</span>
          </div>
          <Input
            placeholder={t('models.placeholders.enterANewName')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoComplete='off'
          />
          <div className='text-muted-foreground text-xs'>{helper}</div>
        </div>

        <DialogFooter className='mt-4'>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('common.actions.cancel')}
          </Button>
          <Button onClick={() => void onSubmit()} disabled={!canSubmit}>
            {isSubmitting ? (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            ) : null}
            {t('models.fields.rename')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
