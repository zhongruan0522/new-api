import { api } from '@/lib/api'

export type CustomVoicePreviewParams = {
  file: File
  model: string
  voice_id: string
  text?: string
  need_noise_reduction?: boolean
  need_volume_normalization?: boolean
}

export type CustomVoicePreviewResult = {
  voice_id: string
  demo_audio: string
  file_id: string
  record_id: number
}

export type CustomVoicePreviewResponse = {
  success: boolean
  message: string
  data: CustomVoicePreviewResult
}

export async function previewCustomVoice(
  params: CustomVoicePreviewParams
): Promise<CustomVoicePreviewResponse> {
  const form = new FormData()
  form.append('file', params.file)
  form.append('model', params.model)
  form.append('voice_id', params.voice_id)
  if (params.text) form.append('text', params.text)
  if (params.need_noise_reduction) form.append('need_noise_reduction', 'true')
  if (params.need_volume_normalization)
    form.append('need_volume_normalization', 'true')

  const res = await api.post<CustomVoicePreviewResponse>(
    '/api/custom_voice/preview',
    form,
    {
      headers: { 'Content-Type': 'multipart/form-data' },
    }
  )
  return res.data
}

export type CustomVoiceConfirmResponse = {
  success: boolean
  message: string
  data: {
    voice_id: string
    status: string
  }
}

export type TtsModelOption = {
  value: string
  label: string
}

// 仅展示以 tts- 开头的可用模型，与游乐场保持一致地复用 /api/user/models。
export async function getTtsModels(): Promise<TtsModelOption[]> {
  const res = await api.get<{ success: boolean; data?: unknown }>(
    '/api/user/models'
  )
  const { data } = res
  if (!data?.success || !Array.isArray(data.data)) return []

  return (data.data as string[])
    .filter((m) => typeof m === 'string' && m.startsWith('tts-'))
    .map((m) => ({ label: m, value: m }))
}

export async function confirmCustomVoice(
  voiceId: string
): Promise<CustomVoiceConfirmResponse> {
  const res = await api.post<CustomVoiceConfirmResponse>(
    '/api/custom_voice/confirm',
    { voice_id: voiceId }
  )
  return res.data
}

// 从 axios 错误对象中安全提取后端业务错误信息，避免在组件中使用 any。
export function extractApiErrorMessage(e: unknown): string | undefined {
  if (typeof e !== 'object' || e === null) return undefined
  const err = e as {
    response?: { data?: { message?: string } }
    message?: string
  }
  return err.response?.data?.message ?? err.message
}
