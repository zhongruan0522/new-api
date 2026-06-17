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

// 使用日志详情弹窗字段 key 常量。
// 必须与后端 setting/console_setting/config.go 中的 UsageLogField* 常量保持一致。
// 仅包含详情弹窗独有字段；同时出现在列表表格列和详情弹窗中的字段
// （channel/token/group/response_time/content）不在配置范围内。
export const USAGE_LOG_FIELD_KEYS = {
  request_id: 'request_id',
  upstream_request_id: 'upstream_request_id',
  retry_chain: 'retry_chain',
  ip_address: 'ip_address',
  client_headers: 'client_headers',
  request_conversion: 'request_conversion',
  reasoning_effort: 'reasoning_effort',
  system_prompt_override: 'system_prompt_override',
  model_mapping: 'model_mapping',
  parameter_override: 'parameter_override',
  billing_source: 'billing_source',
  billing_details: 'billing_details',
  price_table: 'price_table',
  tiered_pricing: 'tiered_pricing',
  violation_fee: 'violation_fee',
  refund_details: 'refund_details',
  subscription_billing: 'subscription_billing',
  token_breakdown: 'token_breakdown',
  audio_tokens: 'audio_tokens',
  topup_audit: 'topup_audit',
  operator_admin: 'operator_admin',
  stream_status: 'stream_status',
} as const

export type UsageLogFieldKey =
  (typeof USAGE_LOG_FIELD_KEYS)[keyof typeof USAGE_LOG_FIELD_KEYS]

// 字段默认可见性元数据：key → { nameKey, descriptionKey, group, admin, user }
// nameKey / descriptionKey 为 i18n 翻译 key，在渲染时通过 t() 解析。
// 默认值严格映射 details-dialog.tsx 中现有的 isAdmin 条件渲染逻辑。
export interface UsageLogFieldMeta {
  key: UsageLogFieldKey
  nameKey: string
  descriptionKey: string
  group: UsageLogFieldGroup
  admin: boolean
  user: boolean
}

export type UsageLogFieldGroup =
  | 'basic'
  | 'request'
  | 'billing'
  | 'token'
  | 'system'
  | 'other'

export const USAGE_LOG_FIELD_GROUPS: {
  value: UsageLogFieldGroup
  labelKey: string
}[] = [
  { value: 'basic', labelKey: 'Usage Log Field Group: Basic' },
  { value: 'request', labelKey: 'Usage Log Field Group: Request' },
  { value: 'billing', labelKey: 'Usage Log Field Group: Billing' },
  { value: 'token', labelKey: 'Usage Log Field Group: Token' },
  { value: 'system', labelKey: 'Usage Log Field Group: System' },
  { value: 'other', labelKey: 'Usage Log Field Group: Other' },
]

export const USAGE_LOG_FIELDS: UsageLogFieldMeta[] = [
  // 基本信息
  { key: 'request_id', nameKey: 'UsageLogField.name.request_id', descriptionKey: 'UsageLogField.desc.request_id', group: 'basic', admin: true, user: true },
  { key: 'upstream_request_id', nameKey: 'UsageLogField.name.upstream_request_id', descriptionKey: 'UsageLogField.desc.upstream_request_id', group: 'basic', admin: true, user: true },
  { key: 'retry_chain', nameKey: 'UsageLogField.name.retry_chain', descriptionKey: 'UsageLogField.desc.retry_chain', group: 'basic', admin: true, user: false },
  { key: 'ip_address', nameKey: 'UsageLogField.name.ip_address', descriptionKey: 'UsageLogField.desc.ip_address', group: 'basic', admin: true, user: true },
  // 请求信息
  { key: 'client_headers', nameKey: 'UsageLogField.name.client_headers', descriptionKey: 'UsageLogField.desc.client_headers', group: 'request', admin: true, user: true },
  { key: 'request_conversion', nameKey: 'UsageLogField.name.request_conversion', descriptionKey: 'UsageLogField.desc.request_conversion', group: 'request', admin: true, user: false },
  { key: 'reasoning_effort', nameKey: 'UsageLogField.name.reasoning_effort', descriptionKey: 'UsageLogField.desc.reasoning_effort', group: 'request', admin: true, user: true },
  { key: 'system_prompt_override', nameKey: 'UsageLogField.name.system_prompt_override', descriptionKey: 'UsageLogField.desc.system_prompt_override', group: 'request', admin: true, user: true },
  { key: 'model_mapping', nameKey: 'UsageLogField.name.model_mapping', descriptionKey: 'UsageLogField.desc.model_mapping', group: 'request', admin: true, user: true },
  { key: 'parameter_override', nameKey: 'UsageLogField.name.parameter_override', descriptionKey: 'UsageLogField.desc.parameter_override', group: 'request', admin: true, user: true },
  // 计费
  { key: 'billing_source', nameKey: 'UsageLogField.name.billing_source', descriptionKey: 'UsageLogField.desc.billing_source', group: 'billing', admin: true, user: false },
  { key: 'billing_details', nameKey: 'UsageLogField.name.billing_details', descriptionKey: 'UsageLogField.desc.billing_details', group: 'billing', admin: true, user: true },
  { key: 'price_table', nameKey: 'UsageLogField.name.price_table', descriptionKey: 'UsageLogField.desc.price_table', group: 'billing', admin: true, user: true },
  { key: 'tiered_pricing', nameKey: 'UsageLogField.name.tiered_pricing', descriptionKey: 'UsageLogField.desc.tiered_pricing', group: 'billing', admin: true, user: true },
  { key: 'violation_fee', nameKey: 'UsageLogField.name.violation_fee', descriptionKey: 'UsageLogField.desc.violation_fee', group: 'billing', admin: true, user: true },
  { key: 'refund_details', nameKey: 'UsageLogField.name.refund_details', descriptionKey: 'UsageLogField.desc.refund_details', group: 'billing', admin: true, user: true },
  { key: 'subscription_billing', nameKey: 'UsageLogField.name.subscription_billing', descriptionKey: 'UsageLogField.desc.subscription_billing', group: 'billing', admin: true, user: true },
  // Token
  { key: 'token_breakdown', nameKey: 'UsageLogField.name.token_breakdown', descriptionKey: 'UsageLogField.desc.token_breakdown', group: 'token', admin: true, user: true },
  { key: 'audio_tokens', nameKey: 'UsageLogField.name.audio_tokens', descriptionKey: 'UsageLogField.desc.audio_tokens', group: 'token', admin: true, user: true },
  // 系统/管理
  { key: 'topup_audit', nameKey: 'UsageLogField.name.topup_audit', descriptionKey: 'UsageLogField.desc.topup_audit', group: 'system', admin: true, user: false },
  { key: 'operator_admin', nameKey: 'UsageLogField.name.operator_admin', descriptionKey: 'UsageLogField.desc.operator_admin', group: 'system', admin: true, user: false },
  { key: 'stream_status', nameKey: 'UsageLogField.name.stream_status', descriptionKey: 'UsageLogField.desc.stream_status', group: 'system', admin: true, user: false },
]

// 构建 UsageLogFields 配置的默认 JSON 字符串。
// 格式：{ "<fieldKey>": { "admin": true, "user": false }, ... }
export function buildDefaultUsageLogFieldsJSON(): string {
  const m: Record<string, { admin: boolean; user: boolean }> = {}
  for (const f of USAGE_LOG_FIELDS) {
    m[f.key] = { admin: f.admin, user: f.user }
  }
  return JSON.stringify(m)
}

// 解析 UsageLogFields 配置 JSON 字符串为 map。
// 如果配置为空或解析失败，返回 null，由调用方决定是否使用默认值。
export function parseUsageLogFieldsConfig(
  raw: string
): Record<string, { admin: boolean; user: boolean }> | null {
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, { admin: boolean; user: boolean }>
    }
  } catch {
    // fallthrough
  }
  return null
}

// 根据配置 map 和字段元数据构建完整的字段可见性状态。
// 如果配置中缺少某字段，使用 USAGE_LOG_FIELDS 中的默认值。
export function resolveUsageLogFieldsConfig(
  raw: string
): Record<string, { admin: boolean; user: boolean }> {
  const parsed = parseUsageLogFieldsConfig(raw)
  if (!parsed) {
    // 使用默认值
    const defaults: Record<string, { admin: boolean; user: boolean }> = {}
    for (const f of USAGE_LOG_FIELDS) {
      defaults[f.key] = { admin: f.admin, user: f.user }
    }
    return defaults
  }
  // 合并默认值，确保新字段有默认值
  const merged: Record<string, { admin: boolean; user: boolean }> = {}
  for (const f of USAGE_LOG_FIELDS) {
    const cfg = parsed[f.key]
    merged[f.key] = {
      admin: cfg?.admin ?? f.admin,
      user: cfg?.user ?? f.user,
    }
  }
  return merged
}
