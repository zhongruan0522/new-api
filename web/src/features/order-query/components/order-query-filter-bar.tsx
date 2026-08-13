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
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'

export interface OrderQueryFilterBarProps {
  keyword: string
  onKeywordChange: (value: string) => void
}

export function OrderQueryFilterBar({
  keyword,
  onKeywordChange,
}: OrderQueryFilterBarProps) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3'>
      <div className='relative sm:w-96'>
        <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
        <Input
          value={keyword}
          className='pl-8'
          placeholder={t('orderQuery.actions.searchByOrderNumber')}
          onChange={(event) => onKeywordChange(event.target.value)}
        />
      </div>
    </div>
  )
}
