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

export type BillingError = {
  error?: {
    message?: string
    type?: string
  }
}

export type TokenSubscription = {
  object: 'billing_subscription'
  has_payment_method: boolean
  soft_limit_usd: number
  hard_limit_usd: number
  system_hard_limit_usd: number
  access_until: number
}

export type TokenUsage = {
  object: 'list'
  total_usage: number
}

export type TokenLog = {
  id: number
  created_at: number
  type: number
  content: string
  username: string
  token_name: string
  model_name: string
  quota: number
  prompt_tokens: number
  completion_tokens: number
  use_time: number
  is_stream: boolean
  channel: number
  channel_name: string
  token_id: number
  group: string
  ip: string
  request_id?: string
  other: string
}

export type TokenLogResponse = {
  success: boolean
  message?: string
  data?: TokenLog[]
}

export type KeyQueryReport = {
  subscription: TokenSubscription
  usage: TokenUsage
  logs: TokenLog[]
}
