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
import { memo, useCallback, useState } from 'react'
import { type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Form } from '@/components/ui/form'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageActionsPortal } from '../components/settings-page-context'
import {
  ModelRatioVisualEditor,
  type ModelRatioField,
} from './model-ratio-visual-editor'

type ModelFormValues = {
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ContextPricing: string
}

type ModelRatioFormProps = {
  form: UseFormReturn<ModelFormValues>
  onSave: (values: ModelFormValues) => Promise<void>
  onReset: () => void
  isSaving: boolean
  isResetting: boolean
}

export const ModelRatioForm = memo(function ModelRatioForm({
  form,
  onSave,
  onReset,
  isSaving,
  isResetting,
}: ModelRatioFormProps) {
  const { t } = useTranslation()
  const [isEditorValid, setIsEditorValid] = useState(true)
  const values = form.watch()

  const handleFieldChange = useCallback(
    (field: ModelRatioField, value: string) => {
      form.setValue(field, value, {
        shouldDirty: true,
        shouldValidate: true,
      })
    },
    [form]
  )

  const handleValidityChange = useCallback((isValid: boolean) => {
    setIsEditorValid(isValid)
  }, [])

  return (
    <Form {...form}>
      <SettingsPageActionsPortal>
        <Button
          type='button'
          variant='destructive'
          size='sm'
          onClick={onReset}
          disabled={isResetting}
        >
          {t('Reset model ratios')}
        </Button>
        <Button
          type='button'
          size='sm'
          onClick={form.handleSubmit(onSave)}
          disabled={isSaving || !isEditorValid}
        >
          {isSaving ? t('Saving...') : t('Save model prices')}
        </Button>
      </SettingsPageActionsPortal>
      <SettingsForm onSubmit={form.handleSubmit(onSave)}>
        <ModelRatioVisualEditor
          modelPrice={values.ModelPrice}
          modelRatio={values.ModelRatio}
          cacheRatio={values.CacheRatio}
          createCacheRatio={values.CreateCacheRatio}
          completionRatio={values.CompletionRatio}
          audioRatio={values.AudioRatio}
          audioCompletionRatio={values.AudioCompletionRatio}
          contextPricing={values.ContextPricing}
          onChange={handleFieldChange}
          onValidityChange={handleValidityChange}
        />
      </SettingsForm>
    </Form>
  )
})
