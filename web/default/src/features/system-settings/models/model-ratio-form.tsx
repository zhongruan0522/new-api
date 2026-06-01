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
import { memo } from 'react'
import { type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Textarea } from '@/components/ui/textarea'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageActionsPortal } from '../components/settings-page-context'

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

type RatioTextareaConfig = {
  name: keyof ModelFormValues
  label: string
  description: string
  rows?: number
}

const ratioFields: RatioTextareaConfig[] = [
  {
    name: 'ModelPrice',
    label: 'Model fixed pricing',
    description:
      'JSON map of model to USD cost per request. Takes precedence over ratio based billing.',
  },
  {
    name: 'ModelRatio',
    label: 'Model ratio',
    description: 'JSON map of model to quota billing multiplier.',
  },
  {
    name: 'CacheRatio',
    label: 'Prompt cache ratio',
    description: 'JSON map used when upstream cache hits occur.',
  },
  {
    name: 'CreateCacheRatio',
    label: 'Create cache ratio',
    description:
      'JSON map applied when creating cache entries for supported models.',
  },
  {
    name: 'CompletionRatio',
    label: 'Completion ratio',
    description:
      'JSON map for custom completion endpoint output multipliers.',
  },
  {
    name: 'AudioRatio',
    label: 'Audio ratio',
    description:
      'JSON map for audio input multipliers where supported by upstream models.',
    rows: 6,
  },
  {
    name: 'AudioCompletionRatio',
    label: 'Audio completion ratio',
    description: 'JSON map for audio output completion multipliers.',
    rows: 6,
  },
  {
    name: 'ContextPricing',
    label: 'Context pricing',
    description:
      'JSON map for context-token segment pricing overrides.',
    rows: 8,
  },
]

export const ModelRatioForm = memo(function ModelRatioForm({
  form,
  onSave,
  onReset,
  isSaving,
  isResetting,
}: ModelRatioFormProps) {
  const { t } = useTranslation()

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
          disabled={isSaving}
        >
          {isSaving ? t('Saving...') : t('Save model prices')}
        </Button>
      </SettingsPageActionsPortal>
      <SettingsForm onSubmit={form.handleSubmit(onSave)}>
        {ratioFields.map((config) => (
          <FormField
            key={config.name}
            control={form.control}
            name={config.name}
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t(config.label)}</FormLabel>
                <FormControl>
                  <Textarea rows={config.rows ?? 8} {...field} />
                </FormControl>
                <FormDescription>{t(config.description)}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        ))}
      </SettingsForm>
    </Form>
  )
})
