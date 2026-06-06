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

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export type TicketType = 'bug' | 'feature' | 'question' | 'other'

export type TicketStatus = 'pending' | 'processing' | 'completed'

export type TicketStatusFilter = 'all' | TicketStatus

export type TicketBase = {
  id: number
  title: string
  type: TicketType
  status: TicketStatus
  created_at: number
  updated_at: number
}

export type TicketSummary = TicketBase

export type TicketMessage = {
  id: number
  type: 'message' | 'status'
  role: 'admin' | 'user'
  username: string
  content?: string
  value?: TicketStatus
  time: number
}

export type TicketDetail = TicketBase & {
  closed_at: number
  messages: TicketMessage[]
}

export type TicketPage = {
  items: TicketSummary[]
  total: number
}

export type TicketListParams = {
  page: number
  pageSize: number
  status: TicketStatusFilter
  keyword?: string
  isAdmin: boolean
}

export type CreateTicketPayload = {
  title: string
  type: TicketType
  content: string
}
