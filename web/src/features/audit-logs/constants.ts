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
import type { AuditLog } from './api'

export const AUDIT_ACTION_TYPES = [
  { value: 'create', label: 'Audit: Create' },
  { value: 'update', label: 'Audit: Update' },
  { value: 'delete', label: 'Audit: Delete' },
] as const

export const AUDIT_MODULES = [
  { value: 'option', label: 'Audit: Module - System Settings' },
  { value: 'channel', label: 'Audit: Module - Channels' },
  { value: 'user', label: 'Audit: Module - Users' },
  { value: 'token', label: 'Audit: Module - Tokens' },
  { value: 'redemption', label: 'Audit: Module - Redemption Codes' },
  { value: 'model', label: 'Audit: Module - Models' },
  { value: 'vendor', label: 'Audit: Module - Vendors' },
  { value: 'dynamic_ratio', label: 'Audit: Module - Dynamic Ratio' },
  { value: 'prefill_group', label: 'Audit: Module - Prefill Groups' },
  { value: 'db', label: 'Audit: Module - Database Migration' },
  { value: 'performance', label: 'Audit: Module - Performance' },
  { value: 'log', label: 'Audit: Module - Log Cleanup' },
  { value: 'setup', label: 'Audit: Module - System Setup' },
  { value: 'dashboard_config', label: 'Audit: Module - Dashboard Configuration' },
] as const

// Action type -> badge color class mapping for table display
export const ACTION_TYPE_BADGE_CLASS: Record<string, string> = {
  create: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
  update: 'bg-blue-500/15 text-blue-700 dark:text-blue-300',
  delete: 'bg-rose-500/15 text-rose-700 dark:text-rose-300',
}

export const DEFAULT_AUDIT_LOGS_DATA: {
  items: AuditLog[]
  total: number
  page: number
  page_size: number
} = {
  items: [],
  total: 0,
  page: 1,
  page_size: 20,
}
