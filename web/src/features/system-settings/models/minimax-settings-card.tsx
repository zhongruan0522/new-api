/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, version 3 of the License.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY. See the GNU Affero General Public License
for more details.
*/
import { useEffect, useRef } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { DisabledSettingsNotice } from '../components/disabled-settings-notice'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  formatJsonForTextarea,
  normalizeJsonString,
  validateJsonString,
} from './utils'

function jsonMapField(value: string, ctx: z.RefinementCtx) {
  const result = validateJsonString(value, { allowEmpty: false })
  if (!result.valid) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: result.message || 'Invalid JSON',
    })
  }
}

// 音色白名单/重定向已迁移到数据库音色管理表，这里只保留 TTS 增强（模型/情绪/语气词）。
const schema = z.object({
  minimax: z.object({
    enabled: z.boolean(),
    model_redirect: z.string().superRefine(jsonMapField),
    voice_whitelist_enabled: z.boolean(),
    custom_voice_enabled: z.boolean(),
    custom_voice_group: z.string(),
    custom_voice_billing_model_id: z.string(),
    emotion_pattern: z.string(),
    emotion_redirect: z.string().superRefine(jsonMapField),
    tone_word_pattern: z.string(),
    tone_word_redirect: z.string().superRefine(jsonMapField),
  }),
})

type MiniMaxFormValues = z.output<typeof schema>
type MiniMaxFormInput = z.input<typeof schema>

type FlatMiniMaxSettings = {
  'minimax.enabled': boolean
  'minimax.model_redirect': string
  'minimax.voice_whitelist_enabled': boolean
  'minimax.custom_voice_enabled': boolean
  'minimax.custom_voice_group': string
  'minimax.custom_voice_billing_model_id': string
  'minimax.emotion_pattern': string
  'minimax.emotion_redirect': string
  'minimax.tone_word_pattern': string
  'minimax.tone_word_redirect': string
}

type MiniMaxJsonMapKey =
  | 'minimax.model_redirect'
  | 'minimax.emotion_redirect'
  | 'minimax.tone_word_redirect'

interface Props {
  defaultValues: MiniMaxFormInput
}

export function MiniMaxSettingsCard({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const buildFormDefaults = (values: MiniMaxFormInput): MiniMaxFormInput => ({
    minimax: {
      enabled: values.minimax.enabled ?? false,
      model_redirect: formatJsonForTextarea(values.minimax.model_redirect),
      voice_whitelist_enabled: values.minimax.voice_whitelist_enabled ?? false,
      custom_voice_enabled: values.minimax.custom_voice_enabled ?? false,
      custom_voice_group: values.minimax.custom_voice_group ?? '',
      custom_voice_billing_model_id:
        values.minimax.custom_voice_billing_model_id ?? '',
      emotion_pattern: values.minimax.emotion_pattern ?? '',
      emotion_redirect: formatJsonForTextarea(values.minimax.emotion_redirect),
      tone_word_pattern: values.minimax.tone_word_pattern ?? '',
      tone_word_redirect: formatJsonForTextarea(
        values.minimax.tone_word_redirect
      ),
    },
  })

  const buildNormalizedDefaults = (
    values: MiniMaxFormInput
  ): FlatMiniMaxSettings => ({
    'minimax.enabled': values.minimax.enabled ?? false,
    'minimax.model_redirect': normalizeJsonString(values.minimax.model_redirect),
    'minimax.voice_whitelist_enabled':
      values.minimax.voice_whitelist_enabled ?? false,
    'minimax.custom_voice_enabled': values.minimax.custom_voice_enabled ?? false,
    'minimax.custom_voice_group': values.minimax.custom_voice_group ?? '',
    'minimax.custom_voice_billing_model_id':
      values.minimax.custom_voice_billing_model_id ?? '',
    'minimax.emotion_pattern': values.minimax.emotion_pattern ?? '',
    'minimax.emotion_redirect': normalizeJsonString(
      values.minimax.emotion_redirect
    ),
    'minimax.tone_word_pattern': values.minimax.tone_word_pattern ?? '',
    'minimax.tone_word_redirect': normalizeJsonString(
      values.minimax.tone_word_redirect
    ),
  })

  const normalizedDefaultsRef = useRef<FlatMiniMaxSettings>(
    buildNormalizedDefaults(defaultValues)
  )

  const form = useForm<MiniMaxFormInput, unknown, MiniMaxFormValues>({
    resolver: zodResolver(schema),
    defaultValues: buildFormDefaults(defaultValues),
  })
  const enabled = form.watch('minimax.enabled')

  const syncJsonMapBaseline =
    (key: MiniMaxJsonMapKey, fieldName: MiniMaxJsonMapKey) =>
    (loadedValue: string) => {
      const safeValue = loadedValue.trim() ? loadedValue : '{}'
      normalizedDefaultsRef.current = {
        ...normalizedDefaultsRef.current,
        [key]: normalizeJsonString(safeValue),
      }
      form.setValue(fieldName, formatJsonForTextarea(safeValue), {
        shouldDirty: false,
        shouldTouch: false,
        shouldValidate: false,
      })
    }

  useEffect(() => {
    normalizedDefaultsRef.current = buildNormalizedDefaults(defaultValues)
    form.reset(buildFormDefaults(defaultValues))
  }, [defaultValues, form])

  const onSubmit = async (values: MiniMaxFormValues) => {
    const normalized: FlatMiniMaxSettings = {
      'minimax.enabled': values.minimax.enabled,
      'minimax.model_redirect': normalizeJsonString(
        values.minimax.model_redirect
      ),
      'minimax.voice_whitelist_enabled': values.minimax.voice_whitelist_enabled,
      'minimax.custom_voice_enabled': values.minimax.custom_voice_enabled,
      'minimax.custom_voice_group': values.minimax.custom_voice_group,
      'minimax.custom_voice_billing_model_id':
        values.minimax.custom_voice_billing_model_id,
      'minimax.emotion_pattern': values.minimax.emotion_pattern,
      'minimax.emotion_redirect': normalizeJsonString(
        values.minimax.emotion_redirect
      ),
      'minimax.tone_word_pattern': values.minimax.tone_word_pattern,
      'minimax.tone_word_redirect': normalizeJsonString(
        values.minimax.tone_word_redirect
      ),
    }

    const updates = (
      Object.keys(normalized) as Array<keyof FlatMiniMaxSettings>
    ).filter((key) => normalized[key] !== normalizedDefaultsRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      const value = normalized[key]
      await updateOption.mutateAsync({
        key,
        value,
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
          <DisabledSettingsNotice enabled={enabled} />

          <FormField
            control={form.control}
            name='minimax.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable TTS Enhancement')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When enabled, model redirect, emotion tag and tone word tag will be applied to MiniMax TTS requests.'
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
                    optionKey='minimax.model_redirect'
                    fromLabel='Client Model'
                    toLabel='MiniMax Model'
                    fromPlaceholder='tts-1'
                    toPlaceholder='speech-01-turbo'
                    jsonPlaceholder={t('{"client-model": "minimax-model"}')}
                    template={JSON.stringify(
                      {
                        'tts-1': 'speech-01-turbo',
                        'tts-1-hd': 'speech-01-240228',
                      },
                      null,
                      2
                    )}
                    emptyText='No redirects configured. Click "Add Mapping" to get started.'
                    onFullValueLoaded={syncJsonMapBaseline(
                      'minimax.model_redirect',
                      'minimax.model_redirect'
                    )}
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
            name='minimax.emotion_pattern'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Emotion Pattern')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    placeholder={`<tts\\s+emotion="([^"]+)">([\\s\\S]*?)</tts>`}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Regex to extract MiniMax voice_setting.emotion from text. Only English punctuation (< > " and parentheses) is matched. The first match wins; when a 2nd capture group exists its text is kept. Default'
                  )}{' '}
                  {`<tts\\s+emotion="happy">text</tts> -> emotion="happy", text kept`}
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
                    optionKey='minimax.emotion_redirect'
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
                    onFullValueLoaded={syncJsonMapBaseline(
                      'minimax.emotion_redirect',
                      'minimax.emotion_redirect'
                    )}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'JSON map of emotion tag value to MiniMax voice_setting.emotion. Empty map uses the captured tag value directly; non-empty map uses the mapped value and ignores unmapped tags. Example'
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
                  <Input {...field} placeholder={`\\(([^()]+)\\)`} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Regex to identify tone word tags in text. Only English half-width parentheses are matched. All matches are replaced in place. Default'
                  )}{' '}
                  {`\\(([^()]+)\\) -> (laugh) (crying)`}
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
                    optionKey='minimax.tone_word_redirect'
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
                    onFullValueLoaded={syncJsonMapBaseline(
                      'minimax.tone_word_redirect',
                      'minimax.tone_word_redirect'
                    )}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'JSON map of tone word tag value to replacement. Parentheses are preserved, only content is replaced. If the user passes a tag whose value equals a configured replacement, the whole tag is deleted (not sent upstream). Example'
                  )}{' '}
                  {`{ "laughs": "笑", "crying": "哭" }`}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>

      <div className='mt-8'>
        <SettingsSection title={t('Custom Voice')}>
          <Form {...form}>
            <SettingsForm
              className='lg:grid-cols-1'
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <SettingsPageFormActions
                onSave={form.handleSubmit(onSubmit)}
                isSaving={updateOption.isPending}
              />
              <DisabledSettingsNotice
                enabled={form.watch('minimax.custom_voice_enabled')}
              />

              <FormField
                control={form.control}
                name='minimax.custom_voice_enabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>
                        {t('Enable Custom Voice')}
                      </FormLabel>
                      <FormDescription>
                        {t(
                          'When enabled, users can use the Custom Voice page to upload samples, generate preview audio and confirm voice customization.'
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
                name='minimax.custom_voice_group'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Custom Voice Group')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder='default'
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'User group used by the Custom Voice page. It decides which channels are used for upload/preview and the billing group ratio.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='minimax.custom_voice_billing_model_id'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Custom Voice Billing Model ID')}
                    </FormLabel>
                    <FormControl>
                      <Input {...field} placeholder='voice-clone' />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Model ID charged once when a user confirms voice customization. It must have a configured Model Price or Model Ratio.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='minimax.voice_whitelist_enabled'
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('Enable Voice Whitelist')}</FormLabel>
                      <FormDescription>
                        {t(
                          'When enabled, only voices marked as allowed in Voice Management (status: created) can be used for TTS. Voice IDs and redirects are managed in Voice Management.'
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
            </SettingsForm>
          </Form>
        </SettingsSection>
      </div>
    </SettingsSection>
  )
}
