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
import { type StatusBadgeProps } from '@/components/status-badge'

// ============================================================================
// API Key Status Configuration
// label values are i18n keys; use t(config.label) in components (e.g. StatusBadge)
// ============================================================================

export const API_KEY_STATUS = {
  ENABLED: 1,
  DISABLED: 2,
  EXPIRED: 3,
  EXHAUSTED: 4,
} as const

export const API_KEY_STATUSES: Record<
  number,
  Pick<StatusBadgeProps, 'variant'> & {
    label: string
    value: number
  }
> = {
  [API_KEY_STATUS.ENABLED]: {
    label: 'channels.status.enabled',
    variant: 'success',
    value: API_KEY_STATUS.ENABLED,
  },
  [API_KEY_STATUS.DISABLED]: {
    label: 'channels.status.disabled',
    variant: 'neutral',
    value: API_KEY_STATUS.DISABLED,
  },
  [API_KEY_STATUS.EXPIRED]: {
    label: 'redemptionCodes.status.expired',
    variant: 'warning',
    value: API_KEY_STATUS.EXPIRED,
  },
  [API_KEY_STATUS.EXHAUSTED]: {
    label: 'common.status.exhausted',
    variant: 'danger',
    value: API_KEY_STATUS.EXHAUSTED,
  },
} as const

export const API_KEY_STATUS_OPTIONS = Object.values(API_KEY_STATUSES).map(
  (config) => ({
    label: config.label,
    value: String(config.value),
  })
)

// ============================================================================
// Default Values
// ============================================================================

export const DEFAULT_GROUP = '' as const

// ============================================================================
// Error Messages (i18n keys: use t(ERROR_MESSAGES.xxx) when displaying)
// ============================================================================

export const ERROR_MESSAGES = {
  UNEXPECTED: 'common.fields.unexpectedErrorOccurred',
  LOAD_FAILED: 'common.errors.failedToLoadApiKeys',
  SEARCH_FAILED: 'common.errors.failedToSearchApiKeys',
  CREATE_FAILED: 'common.errors.failedToCreateApiKey',
  UPDATE_FAILED: 'common.errors.failedToUpdateApiKey',
  DELETE_FAILED: 'common.errors.failedToDeleteApiKey',
  BATCH_DELETE_FAILED: 'common.errors.failedToDeleteApiKeys',
  STATUS_UPDATE_FAILED: 'common.errors.failedToUpdateApiKeyStatus',
  RESET_FAILED: 'common.errors.failedToResetApiKey',
} as const

// ============================================================================
// Success Messages (i18n keys: use t(SUCCESS_MESSAGES.xxx) when displaying)
// ============================================================================

export const SUCCESS_MESSAGES = {
  API_KEY_CREATED: 'common.status.apiKeyCreatedSuccessfully',
  API_KEY_UPDATED: 'common.status.apiKeyUpdatedSuccessfully',
  API_KEY_DELETED: 'common.status.apiKeyDeletedSuccessfully',
  API_KEY_ENABLED: 'common.status.apiKeyEnabledSuccessfully',
  API_KEY_DISABLED: 'common.status.apiKeyDisabledSuccessfully',
  API_KEY_RESET: 'common.fields.apiKeyResetSuccessfully',
} as const
