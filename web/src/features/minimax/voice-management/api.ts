import { api } from '@/lib/api'

export type VoiceRecord = {
  id: number
  created_at: number
  updated_at: number
  type: string
  operator_id: number
  operator_kind: string
  voice_id: string
  quota_cost: number
  redirect_id: string
  allowed: boolean
  remark: string
}

export type VoiceListParams = {
  page?: number
  page_size?: number
  type?: string
  operator_id?: number
  voice_id?: string
  start_timestamp?: number
  end_timestamp?: number
}

export type VoiceListResponse = {
  success: boolean
  message: string
  data: {
    items: VoiceRecord[]
    total: number
    page: number
    page_size: number
  }
}

export async function listVoices(
  params: VoiceListParams
): Promise<VoiceListResponse> {
  const res = await api.get<VoiceListResponse>('/api/minimax/voices/', {
    params,
  })
  return res.data
}

export type VoiceUpsertParams = {
  voice_id: string
  type?: string
  redirect_id?: string
  allowed?: boolean
  remark?: string
}

export async function createVoice(
  params: VoiceUpsertParams
): Promise<{ success: boolean; message: string; data: VoiceRecord }> {
  const res = await api.post('/api/minimax/voices/', params)
  return res.data
}

export async function updateVoice(
  id: number,
  params: VoiceUpsertParams
): Promise<{ success: boolean; message: string; data: VoiceRecord }> {
  const res = await api.put(`/api/minimax/voices/${id}`, params)
  return res.data
}

export async function deleteVoice(
  id: number
): Promise<{ success: boolean; message: string }> {
  const res = await api.delete(`/api/minimax/voices/${id}`)
  return res.data
}

// 从 axios 错误对象中安全提取后端业务错误信息，避免在组件中使用 any。
export function extractApiErrorMessage(e: unknown): string | undefined {
  if (typeof e !== 'object' || e === null) return undefined
  const err = e as { response?: { data?: { message?: string } }; message?: string }
  return err.response?.data?.message ?? err.message
}
