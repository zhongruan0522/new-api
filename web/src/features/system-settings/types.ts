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
export type SystemOption = {
  key: string
  value: string
}

export type SystemOptionKey = string

export type SystemOptionsResponse = {
  success: boolean
  message: string
  data: SystemOption[]
}

export type UpdateOptionRequest = {
  key: string
  value: string | boolean | number
}

export type UpdateOptionResponse = {
  success: boolean
  message: string
}

export type DeleteLogsResponse = {
  success: boolean
  message: string
  data?: number
}

export type SiteSettings = {
  Notice: string
  SystemName: string
  Logo: string
  Footer: string
  About: string
  HomePageContent: string
  ServerAddress: string
  'legal.user_agreement': string
  'legal.privacy_policy': string
  HeaderNavModules: string
  SidebarModulesAdmin: string
}

export type AuthSettings = {
  PasswordLoginEnabled: boolean
  PasswordRegisterEnabled: boolean
  EmailVerificationEnabled: boolean
  RegisterEnabled: boolean
  EmailDomainRestrictionEnabled: boolean
  EmailAliasRestrictionEnabled: boolean
  EmailDomainWhitelist: string
  GitHubOAuthEnabled: boolean
  GitHubClientId: string
  GitHubClientSecret: string
  LinuxDOOAuthEnabled: boolean
  LinuxDOClientId: string
  LinuxDOClientSecret: string
  LinuxDOMinimumTrustLevel: string
  TurnstileCheckEnabled: boolean
  TurnstileSiteKey: string
  TurnstileSecretKey: string
  'passkey.enabled': boolean
  'passkey.rp_display_name': string
  'passkey.rp_id': string
  'passkey.origins': string
  'passkey.allow_insecure_origin': boolean
  'passkey.user_verification': 'required' | 'preferred' | 'discouraged'
  'passkey.attachment_preference': '' | 'platform' | 'cross-platform'
  'passkey.max_passkeys_per_user': number
}

export type ContentSettings = {
  'console_setting.api_info': string
  'console_setting.announcements': string
  'console_setting.faq': string
  'console_setting.uptime_kuma_groups': string
  'console_setting.api_info_enabled': boolean
  'console_setting.announcements_enabled': boolean
  'console_setting.faq_enabled': boolean
  'console_setting.uptime_kuma_enabled': boolean
  DataExportEnabled: boolean
  DataExportDefaultTime: string
  DataExportInterval: number
}

export type ModelSettings = {
  'general_setting.ping_interval_enabled': boolean
  'general_setting.ping_interval_seconds': number
  'gemini.safety_settings': string
  'gemini.version_settings': string
  'gemini.supported_imagine_models': string
  'gemini.function_call_thought_signature_enabled': boolean
  'gemini.remove_function_response_id_enabled': boolean
  'claude.model_headers_settings': string
  'claude.default_max_tokens': string
  'grok.violation_deduction_enabled': boolean
  'grok.violation_deduction_amount': number
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ContextPricing: string
  'tool_billing_setting.rules': string
  TopupGroupRatio: string
  GroupRatio: string
  UserUsableGroups: string
  GroupGroupRatio: string
  AutoGroups: string
  DefaultUseAutoGroup: boolean
  'group_ratio_setting.group_special_usable_group': string
  'channel_affinity_setting.enabled': boolean
  'channel_affinity_setting.switch_on_success': boolean
  'channel_affinity_setting.max_entries': number
  'channel_affinity_setting.default_ttl_seconds': number
  'channel_affinity_setting.rules': string
}

export type BillingSettings = {
  QuotaForNewUser: number
  PreConsumedQuota: number
  QuotaForInviter: number
  QuotaForInvitee: number
  TopUpLink: string
  'general_setting.docs_link': string
  'quota_setting.free_model_pre_consumed_quota': number
  ModelPrice: string
  ModelRatio: string
  CacheRatio: string
  CreateCacheRatio: string
  CompletionRatio: string
  AudioRatio: string
  AudioCompletionRatio: string
  ContextPricing: string
  'tool_billing_setting.rules': string
  TopupGroupRatio: string
  GroupRatio: string
  UserUsableGroups: string
  GroupGroupRatio: string
  AutoGroups: string
  DefaultUseAutoGroup: boolean
  'group_ratio_setting.group_special_usable_group': string
  ServerAddress: string
  PayAddress: string
  EpayId: string
  EpayKey: string
  Price: number
  MinTopUp: number
  CustomCallbackAddress: string
  PayMethods: string
  'payment_setting.amount_options': string
  'payment_setting.amount_discount': string
  StripeApiSecret: string
  StripeWebhookSecret: string
  StripePriceId: string
  StripeUnitPrice: number
  StripeMinTopUp: number
  StripePromotionCodesEnabled: boolean
  'checkin_setting.enabled': boolean
  'checkin_setting.min_quota': number
  'checkin_setting.max_quota': number
}

export type OperationsSettings = {
  RetryTimes: number
  DefaultCollapseSidebar: boolean
  ChannelDisableThreshold: string
  QuotaRemindThreshold: string
  AutomaticDisableChannelEnabled: boolean
  AutomaticEnableChannelEnabled: boolean
  AutomaticDisableKeywords: string
  AutomaticDisableStatusCodes: string
  AutomaticRetryStatusCodes: string
  'monitor_setting.auto_test_channel_enabled': boolean
  'monitor_setting.auto_test_channel_minutes': number
  SMTPServer: string
  SMTPPort: string
  SMTPAccount: string
  SMTPFrom: string
  SMTPToken: string
  SMTPSSLEnabled: boolean
  WorkerUrl: string
  WorkerValidKey: string
  WorkerAllowHttpImageRequestEnabled: boolean
  LogConsumeEnabled: boolean
  'performance_setting.disk_cache_enabled': boolean
  'performance_setting.disk_cache_threshold_mb': number
  'performance_setting.disk_cache_max_size_mb': number
  'performance_setting.disk_cache_path': string
  'performance_setting.monitor_enabled': boolean
  'performance_setting.monitor_cpu_threshold': number
  'performance_setting.monitor_memory_threshold': number
  'performance_setting.monitor_disk_threshold': number
}

export type SecuritySettings = {
  ModelRequestRateLimitEnabled: boolean
  ModelRequestRateLimitCount: number
  ModelRequestRateLimitSuccessCount: number
  ModelRequestRateLimitDurationMinutes: number
  ModelRequestRateLimitGroup: string
  CheckSensitiveEnabled: boolean
  CheckSensitiveOnPromptEnabled: boolean
  SensitiveWords: string
  'fetch_setting.enable_ssrf_protection': boolean
  'fetch_setting.allow_private_ip': boolean
  'fetch_setting.domain_filter_mode': boolean
  'fetch_setting.ip_filter_mode': boolean
  'fetch_setting.domain_list': string[]
  'fetch_setting.ip_list': string[]
  'fetch_setting.allowed_ports': number[]
  'fetch_setting.apply_ip_filter_for_domain': boolean
}
