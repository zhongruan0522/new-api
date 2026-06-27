/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, version 3 of the License.
*/
import { useEffect, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
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
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Checkbox } from '@/components/ui/checkbox'
import { formatTimestampToDate } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'
import { USER_ROLE } from '@/features/users/constants'
import {
  createVoice,
  deleteVoice,
  extractApiErrorMessage,
  listVoices,
  updateVoice,
  type VoiceRecord,
  type VoiceUpsertParams,
} from './api'

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

const TYPE_OPTIONS = [
  { value: '', label: 'all' },
  { value: 'created', label: 'created' },
  { value: 'preview', label: 'preview' },
]

export function VoiceManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const role = useAuthStore((s) => s.auth.user?.role)
  const isRoot = role === USER_ROLE.ROOT

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [filterType, setFilterType] = useState('')
  const [filterOperatorId, setFilterOperatorId] = useState('')
  const [filterVoiceId, setFilterVoiceId] = useState('')

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<VoiceRecord | null>(null)
  const [form, setForm] = useState<VoiceUpsertParams>({
    voice_id: '',
    type: 'created',
    redirect_id: '',
    allowed: false,
    remark: '',
  })

  const queryKey = ['minimax-voices', page, pageSize, filterType, filterOperatorId, filterVoiceId]

  const { data, isLoading } = useQuery({
    queryKey,
    queryFn: () =>
      listVoices({
        page,
        page_size: pageSize,
        type: filterType || undefined,
        operator_id: filterOperatorId ? Number(filterOperatorId) : undefined,
        voice_id: filterVoiceId || undefined,
      }),
  })

  useEffect(() => {
    setPage(1)
  }, [filterType, filterOperatorId, filterVoiceId, pageSize])

  const items = data?.data?.items ?? []
  const total = data?.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ['minimax-voices'] })

  const openCreate = () => {
    setEditing(null)
    setForm({
      voice_id: '',
      type: 'created',
      redirect_id: '',
      allowed: false,
      remark: '',
    })
    setDialogOpen(true)
  }

  const openEdit = (rec: VoiceRecord) => {
    setEditing(rec)
    setForm({
      voice_id: rec.voice_id,
      type: rec.type,
      redirect_id: rec.redirect_id,
      allowed: rec.allowed,
      remark: rec.remark,
    })
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    try {
      if (editing) {
        const res = await updateVoice(editing.id, form)
        if (!res.success) {
          toast.error(res.message || t('Update failed'))
          return
        }
        toast.success(t('Updated'))
      } else {
        const res = await createVoice(form)
        if (!res.success) {
          toast.error(res.message || t('Create failed'))
          return
        }
        toast.success(t('Created'))
      }
      setDialogOpen(false)
      invalidate()
    } catch (e) {
      const msg = extractApiErrorMessage(e)
      toast.error(msg || t('Operation failed'))
    }
  }

  const handleDelete = async (rec: VoiceRecord) => {
    if (!confirm(t('Confirm delete?'))) return
    try {
      const res = await deleteVoice(rec.id)
      if (!res.success) {
        toast.error(res.message || t('Delete failed'))
        return
      }
      toast.success(t('Deleted'))
      invalidate()
    } catch (e) {
      const msg = extractApiErrorMessage(e)
      toast.error(msg || t('Delete failed'))
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Voice Management')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button onClick={openCreate}>{t('Add Voice')}</Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Filters')}</CardTitle>
            </CardHeader>
            <CardContent className='grid gap-4 md:grid-cols-4'>
              <div className='space-y-2'>
                <Label>{t('Type')}</Label>
                <Select value={filterType} onValueChange={(v) => setFilterType(v ?? '')}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TYPE_OPTIONS.map((o) => (
                      <SelectItem key={o.value || 'all'} value={o.value}>
                        {o.value === '' ? t('All') : t(o.label)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-2'>
                <Label>{t('Operator ID')}</Label>
                <Input
                  value={filterOperatorId}
                  onChange={(e) => setFilterOperatorId(e.target.value)}
                  placeholder={t('Operator ID')}
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('Voice ID')}</Label>
                <Input
                  value={filterVoiceId}
                  onChange={(e) => setFilterVoiceId(e.target.value)}
                  placeholder={t('Voice ID')}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className='p-0'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Time')}</TableHead>
                    <TableHead>{t('Type')}</TableHead>
                    <TableHead>{t('Operator ID')}</TableHead>
                    <TableHead>{t('Voice ID')}</TableHead>
                    <TableHead>{t('Cost')}</TableHead>
                    <TableHead>{t('Redirect ID')}</TableHead>
                    <TableHead>{t('Allowed')}</TableHead>
                    {isRoot && <TableHead>{t('Actions')}</TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading ? (
                    <TableRow>
                      <TableCell colSpan={isRoot ? 8 : 7} className='text-center text-muted-foreground'>
                        {t('Loading...')}
                      </TableCell>
                    </TableRow>
                  ) : items.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={isRoot ? 8 : 7} className='text-center text-muted-foreground'>
                        {t('No voices found')}
                      </TableCell>
                    </TableRow>
                  ) : (
                    items.map((v) => (
                      <TableRow key={v.id}>
                        <TableCell>
                          {formatTimestampToDate(v.created_at)}
                        </TableCell>
                        <TableCell>
                          {v.type === 'created' ? t('Created') : t('Preview')}
                        </TableCell>
                        <TableCell>
                          {v.operator_id}
                          {v.operator_kind ? ` (${v.operator_kind})` : ''}
                        </TableCell>
                        <TableCell>{v.voice_id}</TableCell>
                        <TableCell>{v.quota_cost}</TableCell>
                        <TableCell>{v.redirect_id || '-'}</TableCell>
                        <TableCell>{v.allowed ? t('Yes') : t('No')}</TableCell>
                        {isRoot && (
                          <TableCell className='space-x-2'>
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={() => openEdit(v)}
                            >
                              {t('Edit')}
                            </Button>
                            <Button
                              variant='destructive'
                              size='sm'
                              onClick={() => handleDelete(v)}
                            >
                              {t('Delete')}
                            </Button>
                          </TableCell>
                        )}
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-2'>
              <span className='text-sm text-muted-foreground'>
                {t('Page Size')}
              </span>
              <Select
                value={String(pageSize)}
                onValueChange={(v) => setPageSize(Number(v))}
              >
                <SelectTrigger className='w-20'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PAGE_SIZE_OPTIONS.map((s) => (
                    <SelectItem key={s} value={String(s)}>
                      {s}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                size='sm'
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
              >
                {t('Prev')}
              </Button>
              <span className='text-sm text-muted-foreground'>
                {page} / {totalPages}
              </span>
              <Button
                variant='outline'
                size='sm'
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        </div>
      </SectionPageLayout.Content>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editing ? t('Edit Voice') : t('Add Voice')}
            </DialogTitle>
          </DialogHeader>
          <div className='space-y-4'>
            <div className='space-y-2'>
              <Label>{t('Voice ID')}</Label>
              <Input
                value={form.voice_id}
                onChange={(e) =>
                  setForm((f) => ({ ...f, voice_id: e.target.value }))
                }
              />
            </div>
            <div className='space-y-2'>
              <Label>{t('Type')}</Label>
              <Select
                value={form.type || 'created'}
                onValueChange={(v) => setForm((f) => ({ ...f, type: v ?? 'created' }))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='created'>{t('Created')}</SelectItem>
                  <SelectItem value='preview'>{t('Preview')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className='space-y-2'>
              <Label>{t('Redirect ID')}</Label>
              <Input
                value={form.redirect_id || ''}
                onChange={(e) =>
                  setForm((f) => ({ ...f, redirect_id: e.target.value }))
                }
              />
            </div>
            <label className='flex items-center gap-2'>
              <Checkbox
                checked={!!form.allowed}
                onCheckedChange={(v) =>
                  setForm((f) => ({ ...f, allowed: v === true }))
                }
              />
              {t('Allowed for TTS')}
            </label>
            <div className='space-y-2'>
              <Label>{t('Remark')}</Label>
              <Input
                value={form.remark || ''}
                onChange={(e) =>
                  setForm((f) => ({ ...f, remark: e.target.value }))
                }
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setDialogOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button onClick={handleSubmit}>{t('Save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SectionPageLayout>
  )
}
