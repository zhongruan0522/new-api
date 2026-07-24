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
import { useCallback, useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatQuota, formatCompactNumber } from '@/lib/format'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { getUserInfo } from '../../api'
import type { UserInfo } from '../../types'

interface UserInfoDialogProps {
  userId: number | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function UserInfoDialog({
  userId,
  open,
  onOpenChange,
}: UserInfoDialogProps) {
  const { t } = useTranslation()
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const fetchUserInfo = useCallback(
    async (id: number) => {
      setIsLoading(true)
      try {
        const result = await getUserInfo(id)
        if (result.success) {
          setUserInfo(result.data || null)
        } else {
          toast.error(result.message || t('usageLogs.errors.failedToFetchUserInformation'))
        }
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Failed to fetch user info:', error)
        toast.error(t('usageLogs.errors.failedToFetchUserInformation'))
      } finally {
        setIsLoading(false)
      }
    },
    [t]
  )

  useEffect(() => {
    if (open && userId) {
      fetchUserInfo(userId)
    }
  }, [open, userId, fetchUserInfo])

  const InfoItem = ({
    label,
    value,
  }: {
    label: string
    value: string | number
  }) => (
    <div className='space-y-1.5'>
      <Label className='text-muted-foreground text-xs'>{label}</Label>
      <div className='text-sm font-semibold'>{value}</div>
    </div>
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('usageLogs.titles.userInformation')}</DialogTitle>
          <DialogDescription>
            {t(
              'usageLogs.actions.viewDetailedInformationAboutThisUserIncludingBalanceUsage'
            )}
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className='flex items-center justify-center py-8'>
            <Loader2 className='text-muted-foreground size-6 animate-spin' />
          </div>
        ) : userInfo ? (
          <div className='space-y-4 py-4'>
            {/* Basic Info */}
            <div className='grid grid-cols-2 gap-4'>
              <InfoItem label={t('auth.fields.username')} value={userInfo.username} />
              {userInfo.display_name && (
                <InfoItem
                  label={t('usageLogs.fields.displayName')}
                  value={userInfo.display_name}
                />
              )}
            </div>

            {/* Balance Info */}
            <div className='grid grid-cols-2 gap-4'>
              <InfoItem
                label={t('usageLogs.fields.balance')}
                value={formatQuota(userInfo.quota)}
              />
              <InfoItem
                label={t('usageLogs.status.usedQuota')}
                value={formatQuota(userInfo.used_quota)}
              />
            </div>

            {/* Statistics */}
            <div className='grid grid-cols-2 gap-4'>
              <InfoItem
                label={t('dashboard.fields.requestCount')}
                value={formatCompactNumber(userInfo.request_count)}
              />
              {userInfo.group && (
                <InfoItem label={t('common.fields.userGroup')} value={userInfo.group} />
              )}
            </div>

            {/* Invitation Info */}
            {(userInfo.aff_code ||
              userInfo.aff_count !== undefined ||
              (userInfo.aff_quota !== undefined && userInfo.aff_quota > 0)) && (
              <>
                <div className='grid grid-cols-2 gap-4'>
                  {userInfo.aff_code && (
                    <InfoItem
                      label={t('usageLogs.fields.invitationCode')}
                      value={userInfo.aff_code}
                    />
                  )}
                  {userInfo.aff_count !== undefined && (
                    <InfoItem
                      label={t('usageLogs.titles.invitedUsers')}
                      value={formatCompactNumber(userInfo.aff_count)}
                    />
                  )}
                </div>

                {userInfo.aff_quota !== undefined && userInfo.aff_quota > 0 && (
                  <InfoItem
                    label={t('usageLogs.fields.invitationQuota')}
                    value={formatQuota(userInfo.aff_quota)}
                  />
                )}
              </>
            )}

            {/* Remark */}
            {userInfo.remark && (
              <div className='space-y-1.5'>
                <Label className='text-muted-foreground text-xs'>
                  {t('channels.fields.remark')}
                </Label>
                <div className='text-sm leading-relaxed font-semibold wrap-break-word'>
                  {userInfo.remark}
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className='text-muted-foreground py-8 text-center text-sm'>
            {t('usageLogs.titles.noUserInformationAvailable')}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
