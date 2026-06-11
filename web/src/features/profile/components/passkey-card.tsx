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
import { useCallback, useMemo, useState } from 'react'
import { KeyRound, Loader2, Pencil, ShieldAlert, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import dayjs from '@/lib/dayjs'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusBadge } from '@/components/status-badge'
import { usePasskeyManagement } from '@/features/auth/passkey'
import { updatePasskey } from '@/features/auth/passkey/api'
import type { PasskeyCredential } from '@/features/auth/passkey/types'
import {
  SecureVerificationDialog,
  useSecureVerification,
  type VerificationMethod,
  type VerificationMethods,
} from '@/features/auth/secure-verification'

interface PasskeyCardProps {
  loading: boolean
}

export function PasskeyCard({ loading: pageLoading }: PasskeyCardProps) {
  const { t } = useTranslation()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteId, setDeleteId] = useState<number | undefined>(undefined)
  const [restrictedMethod, setRestrictedMethod] =
    useState<VerificationMethod | null>(null)
  const [renameOpen, setRenameOpen] = useState(false)
  const [renameId, setRenameId] = useState<number>(0)
  const [renameName, setRenameName] = useState('')
  const [renaming, setRenaming] = useState(false)

  const {
    status,
    loading,
    registering,
    removing,
    supported,
    fetchStatus,
    register,
    remove,
  } = usePasskeyManagement()

  const {
    open: verificationOpen,
    setOpen: setVerificationOpen,
    methods: verificationMethods,
    state: verificationState,
    startVerification,
    executeVerification,
    cancel: cancelVerification,
    setCode,
    switchMethod,
    fetchVerificationMethods,
  } = useSecureVerification({
    onSuccess: () => {
      setRestrictedMethod(null)
    },
  })

  const dialogMethods = useMemo<VerificationMethods>(() => {
    if (!restrictedMethod) return verificationMethods
    return {
      ...verificationMethods,
      has2FA: restrictedMethod === '2fa' && verificationMethods.has2FA,
      hasPasskey:
        restrictedMethod === 'passkey' && verificationMethods.hasPasskey,
    }
  }, [restrictedMethod, verificationMethods])

  const handleRegister = useCallback(async () => {
    if (!supported) {
      toast.info(t('This device does not support Passkey'))
      return
    }

    const methods = await fetchVerificationMethods()
    if (!methods.has2FA) {
      await register()
      return
    }

    setRestrictedMethod('2fa')
    await startVerification(register, {
      preferredMethod: '2fa',
      title: t('Security Verification'),
      description: t(
        'Confirm your identity with Two-factor Authentication before registering a Passkey.'
      ),
    })
  }, [fetchVerificationMethods, register, startVerification, supported, t])

  const handleRemove = useCallback(
    async (id?: number) => {
      const methods = await fetchVerificationMethods()
      const required: VerificationMethod | null = methods.has2FA
        ? '2fa'
        : methods.hasPasskey
          ? 'passkey'
          : null

      if (!required) {
        toast.error(
          t(
            'Please enable Two-factor Authentication or Passkey before proceeding'
          )
        )
        return
      }

      if (required === 'passkey' && !methods.passkeySupported) {
        toast.info(t('This device does not support Passkey'))
        return
      }

      setConfirmOpen(false)
      setRestrictedMethod(required)
      await startVerification(() => remove(id), {
        preferredMethod: required,
        title: t('Security Verification'),
        description: t(
          'Confirm your identity before removing this Passkey from your account.'
        ),
      })
    },
    [fetchVerificationMethods, remove, startVerification, t]
  )

  const handleRename = useCallback((credential: PasskeyCredential) => {
    setRenameId(credential.id)
    setRenameName(credential.device_name || '')
    setRenameOpen(true)
  }, [])

  const handleRenameSubmit = useCallback(async () => {
    if (!renameId) return
    setRenaming(true)
    try {
      const res = await updatePasskey(renameId, renameName)
      if (!res.success) {
        toast.error(res.message || t('Failed to update device name'))
        return
      }
      toast.success(t('Device name updated'))
      setRenameOpen(false)
      await fetchStatus()
    } catch (error) {
      toast.error(t('Failed to update device name'))
    } finally {
      setRenaming(false)
    }
  }, [renameId, renameName, fetchStatus, t])

  const handleVerificationCancel = useCallback(() => {
    setRestrictedMethod(null)
    cancelVerification()
  }, [cancelVerification])

  const handleVerificationOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setRestrictedMethod(null)
      }
      setVerificationOpen(next)
    },
    [setVerificationOpen]
  )

  const handleDialogVerify = useCallback(
    async (method: VerificationMethod, code?: string) => {
      try {
        await executeVerification(method, code)
      } catch {
        // Errors are already surfaced by useSecureVerification via toast.
      }
    },
    [executeVerification]
  )

  if (pageLoading || loading) {
    return (
      <Card className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='mt-2 h-4 w-64' />
        </CardHeader>
        <CardContent className='p-3 sm:p-5'>
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  const passkeys = status?.passkeys || []
  const count = status?.count || 0
  const maxPasskeys = status?.max_passkeys || 1
  const canAddMore = count < maxPasskeys
  const showUnsupportedNotice = !supported && count === 0

  return (
    <>
      <Card className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <CardTitle className='text-lg tracking-tight sm:text-xl'>
            {t('Passkey Login')}
          </CardTitle>
          <CardDescription className='text-xs sm:text-sm'>
            {t('Use Passkey to sign in without entering your password.')}
          </CardDescription>
        </CardHeader>

        <CardContent className='p-3 sm:p-5'>
          <div className='space-y-6'>
            {passkeys.length > 0 && (
              <div className='space-y-3'>
                {passkeys.map((credential) => (
                  <div
                    key={credential.id}
                    className='flex items-start justify-between gap-4 rounded-lg border p-3'
                  >
                    <div className='flex min-w-0 flex-1 items-start gap-3'>
                      <div className='bg-muted rounded-md p-2'>
                        <KeyRound className='h-4 w-4' />
                      </div>
                      <div className='min-w-0 flex-1 space-y-1'>
                        <p className='truncate font-medium'>
                          {credential.device_name || t('Unnamed Device')}
                        </p>
                        <div className='flex flex-wrap items-center gap-2 text-sm'>
                          <StatusBadge
                            label={
                              credential.attachment === 'platform'
                                ? t('Built-in')
                                : credential.attachment === 'cross-platform'
                                  ? t('External')
                                  : t('Unknown')
                            }
                            variant='neutral'
                            copyable={false}
                          />
                          {credential.backup_eligible && (
                            <StatusBadge
                              label={
                                credential.backup_state
                                  ? t('Backed up')
                                  : t('Not backed up')
                              }
                              variant={
                                credential.backup_state ? 'success' : 'warning'
                              }
                              copyable={false}
                            />
                          )}
                        </div>
                        <p className='text-muted-foreground text-xs'>
                          {credential.last_used_at
                            ? t('labelWithColon', { label: t('Last Used') }) +
                              ' ' +
                              dayjs(credential.last_used_at).fromNow()
                            : t('Not used yet')}
                        </p>
                      </div>
                    </div>
                    <div className='flex gap-2'>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => handleRename(credential)}
                        disabled={removing}
                      >
                        <Pencil className='h-4 w-4' />
                      </Button>
                      <AlertDialog
                        open={confirmOpen && deleteId === credential.id}
                        onOpenChange={(open) => {
                          setConfirmOpen(open)
                          if (open) setDeleteId(credential.id)
                        }}
                      >
                        <AlertDialogTrigger
                          render={
                            <Button
                              variant='ghost'
                              size='icon'
                              disabled={removing}
                            />
                          }
                        >
                          <Trash2 className='h-4 w-4' />
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              {t('Remove Passkey?')}
                            </AlertDialogTitle>
                            <AlertDialogDescription>
                              {t(
                                'This Passkey will be removed from your account. You can register it again anytime.'
                              )}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel disabled={removing}>
                              {t('Cancel')}
                            </AlertDialogCancel>
                            <AlertDialogAction
                              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
                              disabled={removing}
                              onClick={(event) => {
                                event.preventDefault()
                                handleRemove(credential.id)
                              }}
                            >
                              {t('Remove')}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {canAddMore && (
              <Button
                className='w-full'
                onClick={handleRegister}
                disabled={!supported || registering}
              >
                {registering && (
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                )}
                {count === 0 ? t('Enable Passkey') : t('Add Another Passkey')}
              </Button>
            )}

            {!canAddMore && count > 0 && (
              <p className='text-muted-foreground text-center text-sm'>
                {t('You have reached the maximum number of Passkeys')} ({count}/
                {maxPasskeys})
              </p>
            )}

            {showUnsupportedNotice && (
              <div className='bg-muted/60 text-muted-foreground flex items-start gap-3 rounded-md p-4 text-sm'>
                <ShieldAlert className='mt-0.5 h-4 w-4 flex-shrink-0 text-amber-500' />
                <div>
                  <p className='text-foreground font-medium'>
                    {t('Passkey not supported on this device')}
                  </p>
                  <p>
                    {t(
                      'Use a compatible browser or device with biometric authentication or a security key to register a Passkey.'
                    )}
                  </p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Rename Device')}</DialogTitle>
            <DialogDescription>
              {t('Give this Passkey a memorable name')}
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-2'>
            <Label htmlFor='device-name'>{t('Device Name')}</Label>
            <Input
              id='device-name'
              value={renameName}
              onChange={(e) => setRenameName(e.target.value)}
              placeholder={t('e.g. My iPhone, Work Laptop')}
              disabled={renaming}
            />
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setRenameOpen(false)}
              disabled={renaming}
            >
              {t('Cancel')}
            </Button>
            <Button onClick={handleRenameSubmit} disabled={renaming}>
              {renaming && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('Save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SecureVerificationDialog
        open={verificationOpen}
        onOpenChange={handleVerificationOpenChange}
        methods={dialogMethods}
        state={verificationState}
        onVerify={handleDialogVerify}
        onCancel={handleVerificationCancel}
        onCodeChange={setCode}
        onMethodChange={switchMethod}
      />
    </>
  )
}
