/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, version 3 of the License.
*/
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
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
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { VoiceRecord, VoiceUpsertParams } from '../api'

type VoiceDialogProps = {
  open: boolean
  editing: VoiceRecord | null
  form: VoiceUpsertParams
  isSubmitting?: boolean
  onOpenChange: (open: boolean) => void
  onFormChange: (form: VoiceUpsertParams) => void
  onSubmit: () => void
}

export function VoiceDialog(props: VoiceDialogProps) {
  const { t } = useTranslation()

  const updateForm = (patch: Partial<VoiceUpsertParams>) => {
    props.onFormChange({ ...props.form, ...patch })
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {props.editing ? t('minimax.actions.editVoice') : t('minimax.actions.addVoice')}
          </DialogTitle>
        </DialogHeader>

        <div className='space-y-4'>
          <div className='space-y-2'>
            <Label>{t('minimax.fields.voiceId')}</Label>
            <Input
              className='font-mono'
              value={props.form.voice_id}
              onChange={(event) => updateForm({ voice_id: event.target.value })}
            />
          </div>

          <div className='space-y-2'>
            <Label>{t('channels.fields.type')}</Label>
            <Select
              value={props.form.type || 'created'}
              onValueChange={(value) =>
                updateForm({ type: value ?? 'created' })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='created'>
                  {t('minimax.fields.voiceStatusPaid')}
                </SelectItem>
                <SelectItem value='preview'>
                  {t('minimax.fields.voiceStatusPreview')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className='space-y-2'>
            <Label>{t('minimax.fields.redirectId')}</Label>
            <Input
              className='font-mono'
              placeholder={t('minimax.tips.leaveEmptyToSendVoiceIdUpstream')}
              value={props.form.redirect_id || ''}
              onChange={(event) =>
                updateForm({ redirect_id: event.target.value })
              }
            />
          </div>

          <label className='flex items-center gap-2 rounded-md border p-3 text-sm'>
            <Checkbox
              checked={!!props.form.allowed}
              onCheckedChange={(value) =>
                updateForm({ allowed: value === true })
              }
            />
            <span>
              <span className='block font-medium'>{t('minimax.fields.allowedForTts')}</span>
              <span className='text-muted-foreground'>
                {t('minimax.status.onlyPaidVoicesWithTtsEnabledCanPassThe')}
              </span>
            </span>
          </label>

          <div className='space-y-2'>
            <Label>{t('channels.fields.remark')}</Label>
            <Input
              value={props.form.remark || ''}
              onChange={(event) => updateForm({ remark: event.target.value })}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSubmitting}
          >
            {t('common.actions.cancel')}
          </Button>
          <Button onClick={props.onSubmit} disabled={props.isSubmitting}>
            {props.isSubmitting ? t('channels.tips.saving') : t('channels.actions.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
