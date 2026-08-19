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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Loader2, RefreshCw, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { applyGlmResetCard, getGlmResetCards } from '../../api'
import type { GlmResetCard, GlmResetCardType } from '../../types'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatusBadge } from '@/components/status-badge'
import { useChannels } from '../channels-provider'

// 上游在指定重置次数不可用时返回的固定文案，触发自动重新拉列表。
const RESET_CARD_REFRESH_HINT_ZH = '指定的重置次数不可用，请刷新后重试'

interface ResetCardsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function isExpiredCard(card: GlmResetCard): boolean {
  return !card.available
}

function ResetCardRow({
  card,
  resetType,
  pending,
  onUse,
}: {
  card: GlmResetCard
  resetType: GlmResetCardType
  pending: boolean
  onUse: (resetType: GlmResetCardType, recordId: string) => void
}) {
  const { t } = useTranslation()
  const expired = isExpiredCard(card)
  const recordId = card.recordId || ''
  const disabled = expired || pending || !recordId

  return (
    <div className='flex items-center justify-between gap-4 py-2.5'>
      <div className='min-w-0 flex-1 space-y-0.5'>
        <div className='flex items-center gap-2'>
          <span className='text-sm font-medium'>
            {t('channels.fields.resetCardOnce')}
          </span>
          {card.priority && !expired && (
            <StatusBadge
              label={t('channels.fields.resetCardRecommended')}
              variant='info'
              copyable={false}
            />
          )}
        </div>
        <div className='text-muted-foreground text-xs'>
          {t('channels.fields.resetCardExpireTime')}：{card.expireTime || '-'}
        </div>
      </div>
      <Button
        type='button'
        size='sm'
        variant={expired ? 'outline' : 'default'}
        disabled={disabled}
        onClick={() => onUse(resetType, recordId)}
        className='px-4'
      >
        {pending ? (
          <Loader2 className='mr-1.5 h-3.5 w-3.5 animate-spin' />
        ) : null}
        {expired
          ? t('channels.fields.resetCardExpired')
          : t('channels.actions.resetCardUse')}
      </Button>
    </div>
  )
}

function ResetCardSection({
  title,
  resetType,
  cards,
  pendingRecordId,
  onUse,
}: {
  title: string
  resetType: GlmResetCardType
  cards: GlmResetCard[]
  pendingRecordId?: string
  onUse: (resetType: GlmResetCardType, recordId: string) => void
}) {
  const { t } = useTranslation()
  const sortedCards = useMemo(() => {
    // 可用卡按 expireTime 升序（后端已排序，这里兜底）；过期卡排在末尾
    return [...cards].sort((a, b) => {
      const ea = a.available ? 0 : 1
      const eb = b.available ? 0 : 1
      if (ea !== eb) return ea - eb
      return (a.expireTime || '').localeCompare(b.expireTime || '')
    })
  }, [cards])

  return (
    <section className='space-y-2'>
      <h3 className='text-sm font-semibold'>{title}</h3>
      {sortedCards.length === 0 ? (
        <div className='text-muted-foreground rounded-lg border px-4 py-6 text-center text-xs'>
          {t('channels.tips.resetCardNoAvailable')}
        </div>
      ) : (
        <div className='divide-y rounded-lg border'>
          {sortedCards.map((card, index) => (
            <div
              key={`${card.recordId ?? 'card'}-${index}`}
              className='px-4'
            >
              <ResetCardRow
                card={card}
                resetType={resetType}
                pending={
                  pendingRecordId !== undefined &&
                  pendingRecordId === card.recordId
                }
                onUse={onUse}
              />
            </div>
          ))}
        </div>
      )}
    </section>
  )
}

export function ResetCardsDialog({ open, onOpenChange }: ResetCardsDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const [loading, setLoading] = useState(false)
  const [usingRecordId, setUsingRecordId] = useState<string | undefined>()
  const [cards, setCards] = useState<{
    fiveHourResets: GlmResetCard[]
    weekResets: GlmResetCard[]
  }>({ fiveHourResets: [], weekResets: [] })

  const fetchCards = useCallback(async () => {
    if (!currentRow?.id) return
    setLoading(true)
    try {
      const response = await getGlmResetCards(currentRow.id)
      if (!response.success) {
        throw new Error(
          response.message || t('channels.errors.failedToQueryResetCards')
        )
      }
      setCards({
        fiveHourResets: response.data?.fiveHourResets ?? [],
        weekResets: response.data?.weekResets ?? [],
      })
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('channels.errors.failedToQueryResetCards')
      )
      setCards({ fiveHourResets: [], weekResets: [] })
    } finally {
      setLoading(false)
    }
  }, [currentRow?.id, t])

  useEffect(() => {
    if (open && currentRow?.id) {
      fetchCards()
    }
    if (!open) {
      setCards({ fiveHourResets: [], weekResets: [] })
      setUsingRecordId(undefined)
    }
  }, [currentRow?.id, fetchCards, open])

  if (!currentRow) return null

  const handleUse = async (resetType: GlmResetCardType, recordId: string) => {
    if (!currentRow?.id || !recordId) return
    setUsingRecordId(recordId)
    try {
      const response = await applyGlmResetCard(currentRow.id, {
        resetType,
        recordId,
      })
      if (!response.success) {
        const msg = response.msg || response.message || ''
        toast.error(msg || t('channels.errors.failedToUseResetCard'))
        // 上游在指定重置次数不可用时会要求刷新后重试，此时列表已过期，自动重新拉取
        if (msg.includes(RESET_CARD_REFRESH_HINT_ZH)) {
          await fetchCards()
        }
      } else {
        toast.success(t('channels.tips.resetCardUsed'))
        await fetchCards()
      }
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('channels.errors.failedToUseResetCard')
      )
    } finally {
      setUsingRecordId(undefined)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[90vh] sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <RefreshCw className='size-4' />
            {t('channels.titles.resetCards')}
          </DialogTitle>
          <DialogDescription>
            {currentRow.name} {currentRow.id ? `#${currentRow.id}` : ''}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className='max-h-[calc(90vh-12rem)] pr-3'>
          {loading ? (
            <div className='flex h-48 items-center justify-center'>
              <Loader2 className='text-muted-foreground h-6 w-6 animate-spin' />
            </div>
          ) : (
            <div className='space-y-5'>
              <p className='text-muted-foreground text-xs leading-relaxed'>
                <Sparkles className='mr-1 inline size-3.5 align-text-bottom' />
                {t('channels.tips.resetCardDescription')}
              </p>

              <ResetCardSection
                title={t('channels.titles.resetCardWeekSection')}
                resetType='WEEK'
                cards={cards.weekResets}
                pendingRecordId={usingRecordId}
                onUse={handleUse}
              />

              <ResetCardSection
                title={t('channels.titles.resetCardFiveHourSection')}
                resetType='FIVE_HOUR'
                cards={cards.fiveHourResets}
                pendingRecordId={usingRecordId}
                onUse={handleUse}
              />
            </div>
          )}
        </ScrollArea>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={fetchCards}
            disabled={loading}
          >
            {loading ? (
              <Loader2 className='mr-1.5 h-4 w-4 animate-spin' />
            ) : (
              <RefreshCw className='mr-1.5 h-4 w-4' />
            )}
            {t('channels.actions.refresh')}
          </Button>
          <Button type='button' onClick={() => onOpenChange(false)}>
            {t('common.actions.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
