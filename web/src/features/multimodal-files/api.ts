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

import { api } from '@/lib/api'
import type {
  ApiResponse,
  StoredMediaBatchItem,
  StoredMediaItem,
  StoredMediaListParams,
  StoredMediaPage,
  StoredMediaType,
} from './types'

function unwrap<T>(res: ApiResponse<T>, fallbackMessage: string): T {
  if (!res.success) {
    throw new Error(res.message || fallbackMessage)
  }
  if (res.data === undefined || res.data === null) {
    throw new Error(fallbackMessage)
  }
  return res.data
}

export async function getStoredMedia(
  params: StoredMediaListParams
): Promise<StoredMediaPage> {
  const endpoint = params.isAdmin ? '/api/stored_media/' : '/api/stored_media/self'
  const res = await api.get(endpoint, {
    params: {
      p: params.page,
      page_size: params.pageSize,
      start_timestamp: params.startTimestamp,
      end_timestamp: params.endTimestamp,
    },
  })
  return unwrap<StoredMediaPage>(res.data, 'Failed to load stored media')
}

export async function getStoredMediaDetail(
  mediaType: StoredMediaType,
  id: string
): Promise<StoredMediaItem> {
  const res = await api.get(`/api/stored_media/${mediaType}/${id}`)
  return unwrap<StoredMediaItem>(res.data, 'Failed to load stored media detail')
}

export async function deleteStoredMedia(
  mediaType: StoredMediaType,
  id: string
): Promise<number> {
  const res = await api.delete(`/api/stored_media/${mediaType}/${id}`)
  return unwrap<number>(res.data, 'Failed to delete stored media')
}

export async function batchDeleteStoredMedia(
  items: StoredMediaBatchItem[]
): Promise<number> {
  const res = await api.post('/api/stored_media/batch', { items })
  return unwrap<number>(res.data, 'Failed to delete selected stored media')
}
