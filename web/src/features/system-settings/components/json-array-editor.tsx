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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Code, Plus, Save, Table, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  deleteOptionJsonArrayEntry,
  getOptionJsonArray,
  getSystemOptionValue,
  upsertOptionJsonArrayEntry,
} from '../api'

type JsonArrayEditorProps = {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  itemLabel?: string
  itemPlaceholder?: string
  jsonPlaceholder?: string
  template?: string
  emptyText?: string
  addButtonText?: string
  optionKey?: string
  onFullValueLoaded?: (value: string) => void
  onPendingChange?: (hasPending: boolean) => void
}

type ArrayRow = {
  id: string
  value: string
  originalValue?: string
  isNew?: boolean
}

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]
const DEFAULT_JSON_ARRAY_PAGE_SIZE = 20

function formatJsonForEditor(value: string) {
  const trimmed = value.trim()
  if (!trimmed || trimmed === '{}') {
    return '[]'
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return value
  }
}

function parseArrayValue(json: string): string[] | undefined {
  const trimmed = json.trim()
  if (!trimmed || trimmed === '{}') {
    return []
  }
  const parsed = JSON.parse(trimmed)
  if (!Array.isArray(parsed)) {
    return undefined
  }
  return parsed.map((item) => String(item))
}

export function JsonArrayEditor({
  value,
  onChange,
  disabled = false,
  itemLabel,
  itemPlaceholder,
  jsonPlaceholder,
  template,
  emptyText,
  addButtonText,
  optionKey,
  onFullValueLoaded,
  onPendingChange,
}: JsonArrayEditorProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [mode, setMode] = useState<'visual' | 'json'>('visual')
  const [rows, setRows] = useState<ArrayRow[]>([])
  const [jsonValue, setJsonValue] = useState(value)
  const [pageIndex, setPageIndex] = useState(0)
  const [pageSize, setPageSize] = useState(DEFAULT_JSON_ARRAY_PAGE_SIZE)
  const nextRowIdRef = useRef(0)
  const emittedValueRef = useRef<string | null>(null)

  const createRowId = useCallback(() => {
    nextRowIdRef.current += 1
    return `array-${nextRowIdRef.current}`
  }, [])

  const isServerPaginated = Boolean(optionKey)
  const failedToLoadSettingsMessage = t('channels.errors.failedToLoadSettings')

  const jsonArrayQuery = useQuery({
    queryKey: [
      'system-option-json-array',
      optionKey,
      pageIndex + 1,
      pageSize,
      failedToLoadSettingsMessage,
    ],
    queryFn: async () => {
      const data = await getOptionJsonArray({
        key: optionKey ?? '',
        page: pageIndex + 1,
        pageSize,
      })
      if (!data.success) {
        throw new Error(data.message || failedToLoadSettingsMessage)
      }
      return data.data
    },
    enabled: isServerPaginated && mode === 'visual',
  })

  const fullJsonQuery = useQuery({
    queryKey: ['system-option-value', optionKey, failedToLoadSettingsMessage],
    queryFn: async () => {
      const data = await getSystemOptionValue(optionKey ?? '')
      if (!data.success) {
        throw new Error(data.message || failedToLoadSettingsMessage)
      }
      return data.data.value
    },
    enabled: false,
  })

  const deleteEntryMutation = useMutation({
    mutationFn: async (entryValue: string) => {
      if (!optionKey) return
      const data = await deleteOptionJsonArrayEntry({
        key: optionKey,
        value: entryValue,
      })
      if (!data.success) {
        throw new Error(data.message || t('channels.errors.failedToUpdateSetting'))
      }
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['system-option-json-array', optionKey],
        }),
        queryClient.invalidateQueries({
          queryKey: ['system-option-value', optionKey],
        }),
        queryClient.invalidateQueries({ queryKey: ['system-options'] }),
      ])
      toast.success(t('channels.status.settingsUpdatedSuccessfully'))
    },
    onError: (error: Error) => {
      toast.error(error.message || t('channels.errors.failedToUpdateSetting'))
    },
  })

  const upsertEntryMutation = useMutation({
    mutationFn: async (row: ArrayRow) => {
      if (!optionKey) return
      const entryValue = row.value.trim()
      if (!entryValue) {
        throw new Error(t('systemSettings.errors.arrayItemCannotBeEmpty'))
      }
      const data = await upsertOptionJsonArrayEntry({
        key: optionKey,
        value: entryValue,
        old_value: row.isNew ? undefined : row.originalValue,
      })
      if (!data.success) {
        throw new Error(data.message || t('channels.errors.failedToUpdateSetting'))
      }
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['system-option-json-array', optionKey],
        }),
        queryClient.invalidateQueries({
          queryKey: ['system-option-value', optionKey],
        }),
        queryClient.invalidateQueries({ queryKey: ['system-options'] }),
      ])
      toast.success(t('channels.status.settingsUpdatedSuccessfully'))
    },
    onError: (error: Error) => {
      toast.error(error.message || t('channels.errors.failedToUpdateSetting'))
    },
  })

  const parseJsonToRows = useCallback(
    (json: string) => {
      try {
        const parsed = parseArrayValue(json)
        if (!parsed) {
          return
        }
        setRows((previousRows) =>
          parsed.map((entry, index) => ({
            id: previousRows[index]?.id ?? createRowId(),
            value: entry,
          }))
        )
      } catch (_error) {
        // Invalid JSON, keep current rows.
      }
    },
    [createRowId]
  )

  useEffect(() => {
    if (isServerPaginated) {
      setJsonValue(value)
      return
    }
    setJsonValue(value)
    if (value === emittedValueRef.current) {
      return
    }
    emittedValueRef.current = null
    parseJsonToRows(value)
  }, [isServerPaginated, parseJsonToRows, value])

  useEffect(() => {
    if (!isServerPaginated || !jsonArrayQuery.data) {
      return
    }
    setRows(
      (jsonArrayQuery.data.items ?? []).map((item) => ({
        id: `server-${item.value}`,
        value: item.value,
        originalValue: item.value,
      }))
    )
  }, [isServerPaginated, jsonArrayQuery.data])

  const convertRowsToJson = (updatedRows: ArrayRow[]): string => {
    const values = updatedRows
      .map((row) => row.value.trim())
      .filter((entry) => entry)
    return JSON.stringify(values, null, 2)
  }

  const isRowDirty = (row: ArrayRow) =>
    Boolean(row.isNew || row.value !== row.originalValue)

  const hasPendingServerRows = useMemo(
    () => isServerPaginated && rows.some(isRowDirty),
    [isServerPaginated, rows]
  )

  useEffect(() => {
    onPendingChange?.(hasPendingServerRows)
  }, [hasPendingServerRows, onPendingChange])

  const unsavedServerRowCount = useMemo(
    () => (isServerPaginated ? rows.filter((row) => row.isNew).length : 0),
    [isServerPaginated, rows]
  )

  const totalRows = isServerPaginated
    ? (jsonArrayQuery.data?.total ?? 0) + unsavedServerRowCount
    : rows.length

  const pageCount = useMemo(
    () => Math.max(1, Math.ceil(totalRows / pageSize)),
    [pageSize, totalRows]
  )

  const safePageIndex = Math.min(pageIndex, pageCount - 1)
  const pageRows = useMemo(
    () =>
      rows.slice(
        safePageIndex * pageSize,
        safePageIndex * pageSize + pageSize
      ),
    [pageSize, rows, safePageIndex]
  )

  const visibleRows = isServerPaginated ? rows : pageRows
  const isLoadingRows = isServerPaginated && jsonArrayQuery.isLoading

  useEffect(() => {
    if (pageIndex !== safePageIndex) {
      setPageIndex(safePageIndex)
    }
  }, [pageIndex, safePageIndex])

  const emitChange = (json: string) => {
    emittedValueRef.current = json
    setJsonValue(json)
    onChange(json)
  }

  const handleAddRow = () => {
    const newRow: ArrayRow = {
      id: createRowId(),
      value: '',
      isNew: true,
    }
    if (isServerPaginated) {
      setRows((currentRows) => [...currentRows, newRow])
      return
    }
    const updatedRows = [...rows, newRow]
    setRows(updatedRows)
    setPageIndex(Math.max(0, Math.ceil(updatedRows.length / pageSize) - 1))
  }

  const handleDeleteRow = (id: string) => {
    if (isServerPaginated) {
      const row = rows.find((item) => item.id === id)
      if (row?.isNew) {
        setRows((currentRows) => currentRows.filter((item) => item.id !== id))
        return
      }
      const entryValue = row?.originalValue ?? row?.value ?? id
      deleteEntryMutation.mutate(entryValue)
      return
    }
    const updatedRows = rows.filter((row) => row.id !== id)
    setRows(updatedRows)
    emitChange(convertRowsToJson(updatedRows))
  }

  const handleRowChange = (id: string, newValue: string) => {
    const updatedRows = rows.map((row) =>
      row.id === id ? { ...row, value: newValue } : row
    )
    setRows(updatedRows)
    if (isServerPaginated) {
      return
    }
    emitChange(convertRowsToJson(updatedRows))
  }

  const handleSaveRow = (row: ArrayRow) => {
    upsertEntryMutation.mutate(row)
  }

  const handleJsonChange = (newJson: string) => {
    emitChange(newJson)
    if (!isServerPaginated) {
      parseJsonToRows(newJson)
    }
  }

  const handleFillTemplate = () => {
    const templateJson = template ?? JSON.stringify(['voice-id'], null, 2)
    emitChange(templateJson)
    if (!isServerPaginated) {
      parseJsonToRows(templateJson)
    }
  }

  const toggleMode = async () => {
    if (mode === 'visual') {
      if (isServerPaginated) {
        const result = await fullJsonQuery.refetch()
        if (result.isError) {
          toast.error(result.error.message || t('channels.errors.failedToLoadSettings'))
          return
        }
        const fullValue = result.data ?? '[]'
        onFullValueLoaded?.(fullValue)
        setJsonValue(formatJsonForEditor(fullValue))
      } else {
        emitChange(convertRowsToJson(rows))
      }
      setMode('json')
    } else {
      if (!isServerPaginated) {
        parseJsonToRows(jsonValue)
      }
      setMode('visual')
    }
  }

  return (
    <div className='space-y-2'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='flex gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={toggleMode}
            disabled={disabled}
          >
            {mode === 'visual' ? (
              <>
                <Code className='mr-2 h-4 w-4' />
                {t('common.fields.jsonMode')}
              </>
            ) : (
              <>
                <Table className='mr-2 h-4 w-4' />
                {t('common.fields.visualMode')}
              </>
            )}
          </Button>
          {!isServerPaginated || mode === 'json' ? (
            <Button
              type='button'
              variant='link'
              size='sm'
              className='h-auto p-0'
              onClick={handleFillTemplate}
              disabled={disabled}
            >
              {t('common.actions.fillTemplate')}
            </Button>
          ) : null}
        </div>
        {mode === 'visual' ? (
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={handleAddRow}
            disabled={disabled}
          >
            <Plus className='mr-2 h-4 w-4' />
            {t(addButtonText ?? 'common.actions.addItem')}
          </Button>
        ) : null}
      </div>

      {mode === 'visual' ? (
        <div className='space-y-2'>
          {isLoadingRows || totalRows > 0 ? (
            <div className='space-y-2'>
              {hasPendingServerRows ? (
                <div className='text-muted-foreground rounded-md border border-dashed px-3 py-2 text-sm'>
                  {t(
                    'systemSettings.actions.saveOrDeletePendingRowsInThisListBefore'
                  )}
                </div>
              ) : null}
              <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2 text-sm font-medium'>
                <div>{itemLabel ? t(itemLabel) : t('systemSettings.fields.item')}</div>
                <div className={isServerPaginated ? 'w-20' : 'w-10'}></div>
              </div>
              {isLoadingRows ? (
                <div className='text-muted-foreground flex h-24 items-center justify-center rounded-md border border-dashed text-sm'>
                  {t('common.tips.loading')}
                </div>
              ) : null}
              {visibleRows.map((row) => (
                <div
                  key={row.id}
                  className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'
                >
                  <Input
                    value={row.value}
                    onChange={(e) => handleRowChange(row.id, e.target.value)}
                    placeholder={itemPlaceholder}
                    disabled={disabled}
                  />
                  <div className='flex gap-1'>
                    {isServerPaginated ? (
                      <Button
                        type='button'
                        variant='outline'
                        size='icon'
                        onClick={() => handleSaveRow(row)}
                        disabled={
                          disabled ||
                          !isRowDirty(row) ||
                          upsertEntryMutation.isPending
                        }
                      >
                        <Save className='h-4 w-4' />
                      </Button>
                    ) : null}
                    <Button
                      type='button'
                      variant='outline'
                      size='icon'
                      onClick={() => handleDeleteRow(row.id)}
                      disabled={disabled || deleteEntryMutation.isPending}
                    >
                      <Trash2 className='h-4 w-4' />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className='text-muted-foreground rounded-md border border-dashed p-4 text-center text-sm'>
              {t(emptyText ?? 'common.tips.noItemsConfiguredClickAddItemToGetStarted')}
            </div>
          )}
          {pageCount > 1 || totalRows > 0 ? (
            <div className='flex flex-wrap items-center justify-between gap-2 text-sm'>
              <div className='text-muted-foreground'>
                {t('dashboard.fields.total')}: {totalRows}
              </div>
             <div className='flex items-center gap-2'>
               <span className='text-muted-foreground whitespace-nowrap'>
                 {t('common.fields.rowsPerPage')}
               </span>
              <Select
                value={String(pageSize)}
                onValueChange={(value) => {
                  setPageSize(Number(value))
                  setPageIndex(0)
                }}
              >
                 <SelectTrigger className='h-8 w-[70px]' disabled={disabled}>
                  <SelectValue />
                </SelectTrigger>
                 <SelectContent alignItemWithTrigger={false}>
                   <SelectGroup>
                     {PAGE_SIZE_OPTIONS.map((size) => (
                       <SelectItem key={size} value={String(size)}>
                         {size}
                       </SelectItem>
                     ))}
                   </SelectGroup>
                 </SelectContent>
               </Select>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => setPageIndex((current) => Math.max(0, current - 1))}
                  disabled={disabled || safePageIndex === 0}
                >
                  {t('common.fields.previous')}
                </Button>
                <span>
                  {safePageIndex + 1} / {pageCount}
                </span>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() =>
                    setPageIndex((current) => Math.min(pageCount - 1, current + 1))
                  }
                  disabled={disabled || safePageIndex >= pageCount - 1}
                >
                  {t('common.fields.next')}
                </Button>
              </div>
            </div>
          ) : null}
        </div>
      ) : (
        <Textarea
          value={jsonValue}
          onChange={(event) => handleJsonChange(event.target.value)}
          placeholder={jsonPlaceholder}
          className='min-h-[180px] font-mono text-sm'
          disabled={disabled}
        />
      )}
    </div>
  )
}
