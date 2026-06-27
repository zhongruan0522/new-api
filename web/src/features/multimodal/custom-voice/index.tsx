/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, version 3 of the License.
*/
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import {
  confirmCustomVoice,
  extractApiErrorMessage,
  getCustomVoiceTags,
  getTtsModels,
  previewCustomVoice,
  type CustomVoicePreviewResult,
} from './api'

const FALLBACK_MODELS = [
  { value: 'tts-2-hd', label: 'tts-2-hd' },
  { value: 'tts-2-turbo', label: 'tts-2-turbo' },
  { value: 'tts-1-hd', label: 'tts-1-hd' },
  { value: 'tts-1-turbo', label: 'tts-1-turbo' },
]

export function CustomVoice() {
  const { t } = useTranslation()

  const [file, setFile] = useState<File | null>(null)
  const [model, setModel] = useState('tts-2-hd')
  const [voiceId, setVoiceId] = useState('')
  const [previewText, setPreviewText] = useState('')
  const [noiseReduction, setNoiseReduction] = useState(false)
  const [volumeNormalization, setVolumeNormalization] = useState(false)

  const [previewing, setPreviewing] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [preview, setPreview] = useState<CustomVoicePreviewResult | null>(null)

  // 从后端拉取可用模型并筛选 tts- 前缀；拉取失败时回退到内置列表，保证页面可用。
  const { data: ttsModels } = useQuery({
    queryKey: ['custom-voice-tts-models'],
    queryFn: async () => {
      try {
        return await getTtsModels()
      } catch {
        return null
      }
    },
  })

  // 拉取管理员配置的情绪/语气词标签源值（后端只返回 redirect map 的 key）。
  // 仅展示用户应输入的源标签，不展示会被现有逻辑删除的上游重定向目标值。
  const { data: voiceTags } = useQuery({
    queryKey: ['custom-voice-tags'],
    queryFn: async () => {
      try {
        return await getCustomVoiceTags()
      } catch {
        return null
      }
    },
  })

  const emotionTags = voiceTags?.enabled ? voiceTags.emotion_tags ?? [] : []
  const toneWordTags = voiceTags?.enabled ? voiceTags.tone_word_tags ?? [] : []
  const hasTags = emotionTags.length > 0 || toneWordTags.length > 0

  const modelOptions =
    ttsModels && ttsModels.length > 0 ? ttsModels : FALLBACK_MODELS

  // 当前选中的模型不在可用列表中时，回落到第一个可用模型。
  const effectiveModel =
    modelOptions.some((m) => m.value === model) && modelOptions.length > 0
      ? model
      : (modelOptions[0]?.value ?? model)

  const validateVoiceId = (id: string) => {
    const regex = /^[a-zA-Z][a-zA-Z0-9_-]*[^-_]$/
    return id.length >= 8 && id.length <= 256 && regex.test(id)
  }

  const handlePreview = async () => {
    if (!file) {
      toast.error(t('Please select an audio file'))
      return
    }
    if (!validateVoiceId(voiceId)) {
      toast.error(t('Voice ID is invalid'))
      return
    }
    setPreviewing(true)
    setPreview(null)
    try {
      const res = await previewCustomVoice({
        file,
        model: effectiveModel,
        voice_id: voiceId,
        text: previewText,
        need_noise_reduction: noiseReduction,
        need_volume_normalization: volumeNormalization,
      })
      if (res.success && res.data) {
        setPreview(res.data)
        toast.success(t('Preview audio generated'))
      } else {
        toast.error(res.message || t('Preview failed'))
      }
    } catch (e) {
      toast.error(extractApiErrorMessage(e) || t('Preview failed'))
    } finally {
      setPreviewing(false)
    }
  }

  const handleConfirm = async () => {
    if (!preview) return
    setConfirming(true)
    try {
      const res = await confirmCustomVoice(preview.voice_id)
      if (res.success) {
        toast.success(t('Voice customization confirmed'))
        setPreview(null)
        setVoiceId('')
        setFile(null)
      } else {
        toast.error(res.message || t('Confirmation failed'))
      }
    } catch (e) {
      toast.error(extractApiErrorMessage(e) || t('Confirmation failed'))
    } finally {
      setConfirming(false)
    }
  }

  return (
    <div className='space-y-6'>
      <div>
        <h1 className='mb-2 text-2xl font-semibold'>{t('Custom Voice')}</h1>
        <p className='text-muted-foreground'>
          {t('Customize voice configurations.')}
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('Custom Voice')}</CardTitle>
          <CardDescription>
            {t(
              'Upload an audio sample, choose a TTS model, enter a voice ID, then generate a preview audio and confirm whether to customize.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='space-y-2'>
            <Label>{t('Audio File')}</Label>
            <Input
              type='file'
              accept='.mp3,.m4a,.wav'
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Supports mp3, m4a, wav. Duration 10s-5min, max 20MB.')}
            </p>
          </div>

          <div className='grid gap-4 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label>{t('Voice ID')}</Label>
              <Input
                value={voiceId}
                onChange={(e) => setVoiceId(e.target.value)}
                maxLength={256}
                placeholder='myVoice123'
              />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Must start with a letter, only letters, numbers, - and _, length 8-256.'
                )}
              </p>
            </div>

            <div className='space-y-2'>
              <Label>{t('Preview Model')}</Label>
              <Select
                value={effectiveModel}
                onValueChange={(v) => v && setModel(v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {modelOptions.map((m) => (
                    <SelectItem key={m.value} value={m.value}>
                      {m.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className='space-y-2'>
            <Label>{t('Preview Text')}</Label>
            <Textarea
              value={previewText}
              onChange={(e) => setPreviewText(e.target.value)}
              rows={3}
              maxLength={2000}
              placeholder={t('Optional preview text')}
            />
          </div>

          {voiceTags && !voiceTags.enabled && (
            <p className='text-muted-foreground text-xs'>
              {t('TTS enhancement is disabled.')}
            </p>
          )}
          {voiceTags?.enabled && !hasTags && (
            <p className='text-muted-foreground text-xs'>
              {t('No emotion or tone-word tags configured.')}
            </p>
          )}
          {voiceTags?.enabled && hasTags && (
            <div className='space-y-2'>
              {emotionTags.length > 0 && (
                <div className='space-y-1'>
                  <p className='text-muted-foreground text-xs'>
                    {t('Emotion tags')}
                  </p>
                  <div className='flex flex-wrap gap-1.5'>
                    {emotionTags.map((tag) => (
                      <Badge key={tag} variant='secondary'>
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
              {toneWordTags.length > 0 && (
                <div className='space-y-1'>
                  <p className='text-muted-foreground text-xs'>
                    {t('Tone-word tags (wrap with parentheses)')}
                  </p>
                  <div className='flex flex-wrap gap-1.5'>
                    {toneWordTags.map((tag) => (
                      <Badge key={tag} variant='secondary'>
                        ({tag})
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          <div className='flex gap-6'>
            <label className='flex items-center gap-2'>
              <Checkbox
                checked={noiseReduction}
                onCheckedChange={(v) => setNoiseReduction(v === true)}
              />
              {t('Noise Reduction')}
            </label>
            <label className='flex items-center gap-2'>
              <Checkbox
                checked={volumeNormalization}
                onCheckedChange={(v) => setVolumeNormalization(v === true)}
              />
              {t('Volume Normalization')}
            </label>
          </div>

          <Button onClick={handlePreview} disabled={previewing}>
            {previewing ? t('Generating...') : t('Generate Preview')}
          </Button>
        </CardContent>
      </Card>

      {preview && (
        <Card>
          <CardHeader>
            <CardTitle>{t('Preview Audio')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-4'>
            {preview.demo_audio ? (
              <audio controls className='w-full'>
                <source src={preview.demo_audio} type='audio/mpeg' />
              </audio>
            ) : (
              <p className='text-muted-foreground text-sm'>
                {t('No preview audio returned.')}
              </p>
            )}
            <Button onClick={handleConfirm} disabled={confirming}>
              {confirming ? t('Confirming...') : t('Confirm Customization')}
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
