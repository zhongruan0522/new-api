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

// 字段默认可见性元数据：key → { nameZH, description, group, admin, user }
// 默认值严格映射 details-dialog.tsx 中现有的 isAdmin 条件渲染逻辑。
export interface UsageLogFieldMeta {
  key: UsageLogFieldKey
  nameZH: string
  description: string
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
  { key: 'request_id', nameZH: '请求ID', description: '本次请求的唯一标识', group: 'basic', admin: true, user: true },
  { key: 'upstream_request_id', nameZH: '上游请求ID', description: '上游供应商返回的请求ID', group: 'basic', admin: true, user: true },
  { key: 'retry_chain', nameZH: '重试链路', description: '请求在多渠道间的重试路径', group: 'basic', admin: true, user: false },
  { key: 'ip_address', nameZH: 'IP地址', description: '请求来源的客户端IP', group: 'basic', admin: true, user: true },
  // 请求信息
  { key: 'client_headers', nameZH: '客户端请求头', description: 'HTTP-Referer、X-Title、UA', group: 'request', admin: true, user: true },
  { key: 'request_conversion', nameZH: '请求转换', description: '协议转换路径与实际请求路径', group: 'request', admin: true, user: false },
  { key: 'reasoning_effort', nameZH: '推理强度', description: '模型的推理强度设置', group: 'request', admin: true, user: true },
  { key: 'system_prompt_override', nameZH: '系统提示覆盖', description: '是否覆盖了系统提示词', group: 'request', admin: true, user: true },
  { key: 'model_mapping', nameZH: '模型映射', description: '请求模型与实际上游模型的映射', group: 'request', admin: true, user: true },
  { key: 'parameter_override', nameZH: '参数覆盖', description: '请求中被覆盖的参数列表', group: 'request', admin: true, user: true },
  // 计费
  { key: 'billing_source', nameZH: '计费来源', description: '本地计费或上游响应计费', group: 'billing', admin: true, user: false },
  { key: 'billing_details', nameZH: '计费详情', description: '计费模式、倍率与总费用', group: 'billing', admin: true, user: true },
  { key: 'price_table', nameZH: '当前价格表格', description: '各计费项的数量、单价、小计', group: 'billing', admin: true, user: true },
  { key: 'tiered_pricing', nameZH: '阶梯定价详情', description: '动态阶梯计费的匹配详情', group: 'billing', admin: true, user: true },
  { key: 'violation_fee', nameZH: '违规费用', description: '违规扣费的代码、标记与金额', group: 'billing', admin: true, user: true },
  { key: 'refund_details', nameZH: '退款详情', description: '退款的任务ID与原因', group: 'billing', admin: true, user: true },
  { key: 'subscription_billing', nameZH: '订阅计费', description: '订阅实例的计费详情', group: 'billing', admin: true, user: true },
  // Token
  { key: 'token_breakdown', nameZH: 'Token明细', description: '标准/缓存/多模态Token细分', group: 'token', admin: true, user: true },
  { key: 'audio_tokens', nameZH: '音频Token', description: '音频/文本的输入输出统计', group: 'token', admin: true, user: true },
  // 系统/管理
  { key: 'topup_audit', nameZH: '充值审计', description: '充值订单的支付方式、回调IP等', group: 'system', admin: true, user: false },
  { key: 'operator_admin', nameZH: '操作管理员', description: '执行管理操作的管理员信息', group: 'system', admin: true, user: false },
  { key: 'stream_status', nameZH: '流式状态', description: '流式响应的状态与错误信息', group: 'system', admin: true, user: false },
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
