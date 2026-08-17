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
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { CopyButton } from '@/components/copy-button'

interface MaskedValueDisplayProps {
  /** 弹层内标题，如 "Full API Key" / "Full Code" */
  label: string
  /** 完整值，在 Popover 内完整显示 */
  fullValue: string
  /** 表格内显示的脱敏值 */
  maskedValue: string
  /** 复制按钮的 tooltip */
  copyTooltip: string
  /** 复制按钮的 aria-label */
  copyAriaLabel: string
  /** 完整值需按需加载时：复制/展示动作的加载态 */
  isLoading?: boolean
  /**
   * 按需加载完整值：Popover 打开或复制前触发，返回加载后的完整值。
   * 不传表示 fullValue 已就绪。
   */
  onReveal?: () => Promise<string>
}

/**
 * 用于在表格中展示脱敏密钥/兑换码：点击显示完整内容（文本块完整显示，非 Input），支持一键复制。
 * 当完整值需要按需拉取（如兑换码 key 不随列表下发）时，传 onReveal +
 * isLoading，首次展示/复制时加载，复制的是加载后的值。
 */
export function MaskedValueDisplay(props: MaskedValueDisplayProps) {
  return (
    <div className='flex items-center'>
      <Popover>
        <PopoverTrigger
          render={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 font-mono'
              onClick={
                props.onReveal ? () => void props.onReveal?.() : undefined
              }
            />
          }
        >
          {props.maskedValue}
        </PopoverTrigger>
        <PopoverContent
          className='w-auto max-w-[min(90vw,28rem)]'
          align='start'
        >
          <div className='space-y-2'>
            <p className='text-muted-foreground text-xs'>{props.label}</p>
            <pre
              className='bg-muted/50 max-h-[50vh] overflow-auto rounded-md border px-3 py-2 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'
              style={{ wordBreak: 'break-all' }}
            >
              {props.fullValue}
            </pre>
          </div>
        </PopoverContent>
      </Popover>
      {props.isLoading ? (
        <span className='text-muted-foreground inline-flex h-7 w-7 items-center justify-center text-xs'>
          …
        </span>
      ) : (
        <CopyButton
          value={props.fullValue}
          className='size-7'
          iconClassName='size-3.5'
          tooltip={props.copyTooltip}
          aria-label={props.copyAriaLabel}
          onBeforeCopy={props.onReveal}
        />
      )}
    </div>
  )
}
