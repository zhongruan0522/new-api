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
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface PasskeyCredential {
  id: number
  user_id: number
  credential_id: string
  device_name: string
  attestation_type: string
  aaguid: string
  sign_count: number
  clone_warning: boolean
  user_present: boolean
  user_verified: boolean
  backup_eligible: boolean
  backup_state: boolean
  transports: string
  attachment: string
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface PasskeyStatus {
  enabled: boolean
  passkeys: PasskeyCredential[]
  count: number
  max_passkeys: number
  last_used_at?: string | null
  backup_eligible?: boolean
  backup_state?: boolean
  [key: string]: unknown
}

export interface PasskeyOptionsPayload {
  options?: unknown
  publicKey?: unknown
  response?: unknown
  Response?: unknown
}
