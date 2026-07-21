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
// ============================================================================
// Profile Constants
// ============================================================================

/**
 * Default quota warning threshold (500,000 = $1)
 */
export const DEFAULT_QUOTA_WARNING_THRESHOLD = 500000

/**
 * Notification methods. Email label is an i18n key (resolved via t()).
 * Webhook / Bark / Gotify are technical proper nouns rendered verbatim.
 */
export const NOTIFICATION_METHODS = [
  { value: 'email' as const, label: 'auth.fields.email', isI18nKey: true },
  { value: 'webhook' as const, label: 'Webhook', isI18nKey: false },
  { value: 'bark' as const, label: 'Bark', isI18nKey: false },
  { value: 'gotify' as const, label: 'Gotify', isI18nKey: false },
] as const
