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
import { type ReactNode, useRef } from 'react'
import { Check, Copy, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface CopyButtonProps {
  value: string
  children?: ReactNode
  className?: string
  iconClassName?: string
  variant?: 'ghost' | 'outline' | 'default' | 'secondary' | 'destructive'
  size?: 'default' | 'sm' | 'lg' | 'icon'
  tooltip?: string
  successTooltip?: string
  'aria-label'?: string
  /**
   * 复制前回调：返回实际要复制的值（如按需加载后的完整 key）。
   * 返回空字符串表示加载失败，跳过复制。
   */
  onBeforeCopy?: () => Promise<string>
  /** 复制结果是否弹全局 toast（默认 false，仅切换图标/tooltip 反馈） */
  notify?: boolean
  /** 禁用按钮（如按需内容仍在加载中） */
  disabled?: boolean
  /** 加载态：显示 spinner，替代复制图标 */
  loading?: boolean
}

export function CopyButton({
  value,
  children,
  className,
  iconClassName,
  variant = 'ghost',
  size = 'icon',
  tooltip,
  successTooltip,
  'aria-label': ariaLabel,
  onBeforeCopy,
  notify = false,
  disabled = false,
  loading = false,
}: CopyButtonProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify })
  // onBeforeCopy 场景下 value 可能为空占位（如批量复制），实际复制的是
  // resolved 值；两种情况都要能切换到"已复制"状态。
  const resolvedValueRef = useRef<string | null>(null)
  // value 变化时旧 resolved 值随之失效，避免复制目标切换后 copiedText
  // 仍匹配旧值而误显示"已复制"。
  const prevValueRef = useRef(value)
  if (prevValueRef.current !== value) {
    prevValueRef.current = value
    resolvedValueRef.current = null
  }
  const isCopied =
    copiedText !== null &&
    (copiedText === value || copiedText === resolvedValueRef.current)
  const resolvedTooltip = tooltip ?? t('common.actions.copyToClipboard')
  const resolvedSuccessTooltip = successTooltip ?? t('common.status.copiedb7c3ca')
  const resolvedAriaLabel = ariaLabel ?? resolvedTooltip
  const copiedAriaLabel = t('common.status.copied')

  const handleCopy = async () => {
    if (onBeforeCopy) {
      const resolved = await onBeforeCopy()
      if (!resolved) return
      resolvedValueRef.current = resolved
      await copyToClipboard(resolved)
      return
    }
    resolvedValueRef.current = null
    await copyToClipboard(value)
  }

  const button = (
    <Button
      variant={variant}
      size={size}
      className={cn('shrink-0', className)}
      onClick={() => void handleCopy()}
      disabled={disabled || loading}
      aria-label={isCopied ? copiedAriaLabel : resolvedAriaLabel}
    >
      {loading ? (
        <Loader2 className={cn('animate-spin', iconClassName)} />
      ) : isCopied ? (
        <Check className={cn('text-success', iconClassName)} />
      ) : (
        <Copy className={cn(iconClassName)} />
      )}
      {children}
    </Button>
  )

  if (tooltip || successTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger render={button}></TooltipTrigger>
        <TooltipContent>
          <p>{isCopied ? resolvedSuccessTooltip : resolvedTooltip}</p>
        </TooltipContent>
      </Tooltip>
    )
  }

  return button
}
