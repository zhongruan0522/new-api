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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { AuditLog } from '../api'

type AuditDiffDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  auditLog: AuditLog | null
}

function prettyJson(raw: string | undefined): string | null {
  if (!raw) return null
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    // Not valid JSON — return the original text for display.
    return raw
  }
}

function DiffPanel({ title, raw }: { title: string; raw: string | undefined }) {
  const { t } = useTranslation()
  const pretty = prettyJson(raw)
  const hasContent = pretty !== null && pretty !== ''

  return (
    <div className='flex min-w-0 flex-col gap-2'>
      <div className='text-muted-foreground text-xs font-medium'>{title}</div>
      <div className='bg-muted/40 min-w-0 overflow-auto rounded-lg border p-3'>
        {hasContent ? (
          <pre className='text-foreground/90 text-xs leading-relaxed break-all whitespace-pre-wrap'>
            {pretty}
          </pre>
        ) : (
          <div className='text-muted-foreground text-xs italic'>
            {t('No Data')}
          </div>
        )}
      </div>
    </div>
  )
}

export function AuditDiffDialog({
  open,
  onOpenChange,
  auditLog,
}: AuditDiffDialogProps) {
  const { t } = useTranslation()
  // Reset internal state when dialog closes — avoids flashing stale content
  // during the close animation.
  const [visibleLog, setVisibleLog] = useState<AuditLog | null>(auditLog)

  useEffect(() => {
    if (open) {
      setVisibleLog(auditLog)
    }
  }, [open, auditLog])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{t('Audit Log Details')}</DialogTitle>
          {visibleLog && (
            <DialogDescription>{visibleLog.description}</DialogDescription>
          )}
        </DialogHeader>

        <div className='grid gap-3 sm:grid-cols-2'>
          <DiffPanel title={t('Before')} raw={visibleLog?.before_data} />
          <DiffPanel title={t('After')} raw={visibleLog?.after_data} />
        </div>
      </DialogContent>
    </Dialog>
  )
}
