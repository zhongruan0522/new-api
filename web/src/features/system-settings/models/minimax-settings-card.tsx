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
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ModelMappingEditor } from '@/features/channels/components/model-mapping-editor'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  formatJsonForTextarea,
  normalizeJsonString,
  validateJsonString,
} from './utils'

// JSON map fields share the same validation: must parse as non-empty JSON object.
// 后端拒绝空串（清空请用 "{}"），因此前端也必须 allowEmpty=false。
function jsonMapField(value: string, ctx: z.RefinementCtx) {
  const result = validateJsonString(value, { allowEmpty: false })
  if (!result.valid) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: result.message || 'Invalid JSON',
    })
  }
}

const minimaxSchema = z.object({
  'minimax.enabled': z.boolean(),
  'minimax.model_redirect': z.string().superRefine(jsonMapField),
  'minimax.voice_redirect': z.string().superRefine(jsonMapField),
  'minimax.emotion_pattern': z.string(),
  'minimax.emotion_redirect': z.string().superRefine(jsonMapField),
  'minimax.tone_word_pattern': z.string(),
  'minimax.tone_word_redirect': z.string().superRefine(jsonMapField),
})

type MiniMaxFormValues = z.infer<typeof minimaxSchema>

interface Props {
  defaultValues: MiniMaxFormValues
}

const MINIMAX_JSON_KEYS = [
  'minimax.model_redirect',
  'minimax.voice_redirect',
  'minimax.emotion_redirect',
  'minimax.tone_word_redirect',
] as const

export function MiniMaxSettingsCard(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  // Format JSON defaults for display once, so the textarea shows pretty JSON
  // while the user edits raw text. Saving normalizes via normalizeJsonString.
  const formattedDefaults: MiniMaxFormValues = {
    ...props.defaultValues,
    'minimax.model_redirect': formatJsonForTextarea(
      props.defaultValues['minimax.model_redirect']
    ),
    'minimax.voice_redirect': formatJsonForTextarea(
      props.defaultValues['minimax.voice_redirect']
    ),
    'minimax.emotion_redirect': formatJsonForTextarea(
      props.defaultValues['minimax.emotion_redirect']
    ),
    'minimax.tone_word_redirect': formatJsonForTextarea(
      props.defaultValues['minimax.tone_word_redirect']
    ),
  }

  // normalizedDefaults 用于保存时的差异比较基线。
  // 必须与 onSubmit 中对用户输入的 normalize 方式一致，否则会误判变化。
  const normalizedDefaults: MiniMaxFormValues = {
    ...props.defaultValues,
    'minimax.model_redirect': normalizeJsonString(
      props.defaultValues['minimax.model_redirect']
    ),
    'minimax.voice_redirect': normalizeJsonString(
      props.defaultValues['minimax.voice_redirect']
    ),
    'minimax.emotion_redirect': normalizeJsonString(
      props.defaultValues['minimax.emotion_redirect']
    ),
    'minimax.tone_word_redirect': normalizeJsonString(
      props.defaultValues['minimax.tone_word_redirect']
    ),
  }

  const form = useForm<MiniMaxFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(minimaxSchema) as any,
    defaultValues: formattedDefaults,
  })

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  useResetForm(form as any, formattedDefaults)

  const onSubmit = async (data: MiniMaxFormValues) => {
    // Normalize JSON map fields before comparing/saving
    const normalized = { ...data }
    for (const key of MINIMAX_JSON_KEYS) {
      normalized[key] = normalizeJsonString(data[key]) as never
    }

    const entries = Object.entries(normalized) as [string, unknown][]
    const updates = entries.filter(
      ([key, value]) =>
        value !== (normalizedDefaults[key as keyof MiniMaxFormValues] as unknown)
    )
    for (const [key, value] of updates) {
      await updateOption.mutateAsync({
        key,
        value: value as string | number | boolean,
      })
    }
  }

  return (
    <SettingsSection title={t('MiniMax TTS Enhancement')}>
      <Form {...form}>
        <SettingsForm
          className='lg:grid-cols-1'
          onSubmit={form.handleSubmit(onSubmit)}
        >
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='minimax.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable TTS Enhancement')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When enabled, model redirect, emotion tag, tone word tag and voice redirect will be applied to MiniMax TTS requests.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='minimax.model_redirect'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Model Redirect')}</FormLabel>
                <FormControl>
                  <ModelMappingEditor
                    value={field.value}
                    onChange={field.onChange}
                    fromLabel='Client Model'
                    toLabel='MiniMax Model'
                    fromPlaceholder='tts-1'
                    toPlaceholder='speech-01-turbo'
                    jsonPlaceholder={t(
                      '{"client-model": "minimax-model"}'
                    )}
                    template={JSON.stringify(
                      { 'tts-1': 'speech-01-turbo', 'tts-1-hd': 'speech-01-240228' },
                      null,
                      2
                    )}
                    emptyText='No redirects configured. Click "Add Mapping" to get started.'
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'JSON map of client model name to MiniMax real model name. Example'
                  )}{' '}
                  {`{ "tts-1": "speech-01-turbo", "tts-1-hd": "speech-01-240228" }`}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='minimax.voice_redirect'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Voice Redirect')}</FormLabel>
                <FormControl>
                  <ModelMappingEditor
                    value={field.value}
                    onChange={field.onChange}
                    fromLabel='OpenAI Voice'
                    toLabel='MiniMax Voice ID'
                    fromPlaceholder='alloy'
                    toPlaceholder='male-qn-qingse'
                    jsonPlaceholder={t('{"openai-voice": "minimax-voice_id"}')}
                    template={JSON.stringify(
                      { alloy: 'male-qn-qingse', nova: 'female-shaonv' },
                      null,
                      2
                    )}
                    emptyText='No redirects configured. Click "Add Mapping" to get started.'
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'JSON map of OpenAI voice name to MiniMax voice_id. Example'
                  )}{' '}
                  {`{ "alloy": "male-qn-qingse", "nova": "female-shaonv" }`}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='minimax.emotion_pattern'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Emotion Pattern')}</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Regex to identify emotion tags in text, e.g. \\((happy\\|sad)\\)'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='minimax.emotion_redirect'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Emotion Redirect')}</FormLabel>
                <FormControl>
                  <ModelMappingEditor
                    value={field.value}
                    onChange={field.onChange}
                    fromLabel='Emotion Tag'
                    toLabel='MiniMax Emotion'
                    fromPlaceholder='happy'
                    toPlaceholder='happy'
                    jsonPlaceholder={t('{"tag-value": "minimax-emotion"}')}
                    template={JSON.stringify(
                      { happy: 'happy', sad: 'sad' },
                      null,
                      2
                    )}
                    emptyText='No redirects configured. Click "Add Mapping" to get started.'
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'JSON map of emotion tag value (inside parentheses) to MiniMax voice_setting.emotion. Tags are stripped from text. Example'
                  )}{' '}
                  {`{ "happy": "happy", "sad": "sad" }`}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='minimax.tone_word_pattern'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Tone Word Pattern')}</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Regex to identify tone word tags in text, e.g. \\((laughs\\|crying)\\)'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='minimax.tone_word_redirect'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Tone Word Redirect')}</FormLabel>
                <FormControl>
                  <ModelMappingEditor
                    value={field.value}
                    onChange={field.onChange}
                    fromLabel='Tone Word Tag'
                    toLabel='Replacement'
                    fromPlaceholder='laughs'
                    toPlaceholder='笑'
                    jsonPlaceholder={t('{"tag-value": "replacement"}')}
                    template={JSON.stringify(
                      { laughs: '笑', crying: '哭' },
                      null,
                      2
                    )}
                    emptyText='No redirects configured. Click "Add Mapping" to get started.'
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'JSON map of tone word tag value to replacement. Parentheses are preserved, only content is replaced. Example'
                  )}{' '}
                  {`{ "laughs": "笑", "crying": "哭" }`}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
