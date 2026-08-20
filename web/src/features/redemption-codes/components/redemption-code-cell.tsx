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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { MaskedValueDisplay } from '@/components/masked-value-display'
import { getRedemptionKey } from '../api'
import { type Redemption } from '../types'

/**
 * 兑换码完整 key 不随列表下发（列表默认脱敏），点击展示或复制时
 * 按需请求完整值，服务端会记录查看日志。
 */
export function RedemptionCodeCell({ redemption }: { redemption: Redemption }) {
  const { t } = useTranslation()
  const [fullKey, setFullKey] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const loadKey = async (): Promise<string> => {
    if (fullKey !== null) return fullKey
    setIsLoading(true)
    try {
      const res = await getRedemptionKey(redemption.id)
      if (!res.success || !res.data?.key) {
        throw new Error(res.message)
      }
      setFullKey(res.data.key)
      return res.data.key
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('redemptionCodes.errors.failedToFetchKey')
      )
      return ''
    } finally {
      setIsLoading(false)
    }
  }

  const maskedDisplay = fullKey
    ? `${fullKey.slice(0, 8)}${'*'.repeat(16)}${fullKey.slice(-8)}`
    : '********'

  return (
    <MaskedValueDisplay
      label={t('redemptionCodes.fields.fullCode')}
      fullValue={fullKey ?? '********'}
      maskedValue={maskedDisplay}
      copyTooltip={t('redemptionCodes.actions.copyCode')}
      copyAriaLabel={t('redemptionCodes.actions.copyRedemptionCode')}
      isLoading={isLoading}
      onReveal={loadKey}
    />
  )
}
