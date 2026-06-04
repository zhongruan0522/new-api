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
import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Plus, Search, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { getEnabledModels } from '@/features/channels/api'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { safeJsonParse } from '../utils/json-parser'
import { formatPricingNumber } from './pricing-format'

type ModelRatioVisualEditorProps = {
  modelPrice: string
  modelRatio: string
  cacheRatio: string
  createCacheRatio: string
  completionRatio: string
  audioRatio: string
  audioCompletionRatio: string
  contextPricing: string
  onChange: (field: ModelRatioField, value: string) => void
  onValidityChange?: (isValid: boolean) => void
}

export type ModelRatioField =
  | 'ModelPrice'
  | 'ModelRatio'
  | 'CacheRatio'
  | 'CreateCacheRatio'
  | 'CompletionRatio'
  | 'AudioRatio'
  | 'AudioCompletionRatio'
  | 'ContextPricing'

type NumericMap = Record<string, number>
type UnknownMap = Record<string, unknown>
type PricingMode = 'unset' | 'per-request' | 'per-token'

type ModelRow = {
  name: string
  mode: PricingMode
  fixedPrice?: number
  inputPrice?: number
  completionPrice?: number
  cachePrice?: number
  createCachePrice?: number
  audioInputPrice?: number
  audioOutputPrice?: number
  contextPricing?: unknown
}

type ContextPricingEditorProps = {
  modelName: string
  contextPricing: unknown
  onCommit: (name: string, value: string) => void
  onValidityChange?: (isValid: boolean) => void
}

const PAGE_SIZE_OPTIONS = [20, 50, 100]
const numberInputPattern = /^(\d+(\.\d*)?|\.\d*)?$/

/** Values that look like an in-progress decimal input (e.g. "2.", ".") */
const isDeferredDecimal = (value: string) =>
  value === '.' || (typeof value === 'string' && /\.$/.test(value))

function hasValue(value: unknown) {
  return value !== '' && value !== undefined && value !== null
}

function parseNumber(value: string) {
  if (!hasValue(value)) return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

function toInputValue(value: number | undefined) {
  if (value === undefined || value === null || !Number.isFinite(value)) {
    return ''
  }
  return formatPricingNumber(value)
}

function normalizeNumber(value: number) {
  return Number(formatPricingNumber(value))
}

function parseNumericMap(value: string, context: string): NumericMap {
  return safeJsonParse<NumericMap>(value, {
    fallback: {},
    context,
  })
}

function sortedJson<T extends Record<string, unknown>>(map: T) {
  const cleanedEntries = Object.entries(map).filter(([, value]) => {
    if (value === undefined || value === null || value === '') return false
    if (typeof value === 'number' && !Number.isFinite(value)) return false
    return true
  })

  const sorted = Object.fromEntries(
    cleanedEntries.sort(([left], [right]) =>
      left.localeCompare(right, 'en', {
        numeric: true,
        sensitivity: 'base',
      })
    )
  )
  return JSON.stringify(sorted, null, 2)
}

function buildRow(
  name: string,
  maps: {
    price: NumericMap
    ratio: NumericMap
    cache: NumericMap
    createCache: NumericMap
    completion: NumericMap
    audio: NumericMap
    audioCompletion: NumericMap
    context: UnknownMap
  }
): ModelRow {
  const fixedPrice = maps.price[name]
  const inputRatio = maps.ratio[name]
  const inputPrice =
    typeof inputRatio === 'number' ? normalizeNumber(inputRatio * 2) : undefined
  const audioInputPrice =
    typeof inputPrice === 'number' && typeof maps.audio[name] === 'number'
      ? normalizeNumber(inputPrice * maps.audio[name])
      : undefined

  let mode: PricingMode = 'unset'
  if (fixedPrice !== undefined) {
    mode = 'per-request'
  } else if (
    inputRatio !== undefined ||
    maps.cache[name] !== undefined ||
    maps.createCache[name] !== undefined ||
    maps.completion[name] !== undefined ||
    maps.audio[name] !== undefined ||
    maps.audioCompletion[name] !== undefined ||
    maps.context[name] !== undefined
  ) {
    mode = 'per-token'
  }

  return {
    name,
    mode,
    fixedPrice,
    inputPrice,
    completionPrice:
      typeof inputPrice === 'number' && typeof maps.completion[name] === 'number'
        ? normalizeNumber(inputPrice * maps.completion[name])
        : undefined,
    cachePrice:
      typeof inputPrice === 'number' && typeof maps.cache[name] === 'number'
        ? normalizeNumber(inputPrice * maps.cache[name])
        : undefined,
    createCachePrice:
      typeof inputPrice === 'number' &&
      typeof maps.createCache[name] === 'number'
        ? normalizeNumber(inputPrice * maps.createCache[name])
        : undefined,
    audioInputPrice,
    audioOutputPrice:
      typeof audioInputPrice === 'number' &&
      typeof maps.audioCompletion[name] === 'number'
        ? normalizeNumber(audioInputPrice * maps.audioCompletion[name])
        : undefined,
    contextPricing: maps.context[name],
  }
}

function getSortRank(mode: PricingMode) {
  if (mode === 'unset') return 0
  if (mode === 'per-request') return 1
  return 2
}

function getRowSummary(row: ModelRow, t: (key: string) => string) {
  if (row.mode === 'unset') return t('Unset price')
  if (row.mode === 'per-request') {
    return `$${toInputValue(row.fixedPrice)} / ${t('request')}`
  }
  if (row.contextPricing) return t('Context pricing')
  return row.inputPrice !== undefined
    ? `$${toInputValue(row.inputPrice)} / 1M ${t('tokens')}`
    : t('Per-token')
}

function PriceInput({
  label,
  value,
  disabled,
  placeholder,
  onChange,
}: {
  label: string
  value: string
  disabled?: boolean
  placeholder?: string
  onChange: (value: string) => void
}) {
  const [draft, setDraft] = useState(value)
  const [focused, setFocused] = useState(false)

  // Sync draft with external value when not focused
  useEffect(() => {
    if (!focused) setDraft(value)
  }, [focused, value])

  const handleChange = (next: string) => {
    if (!numberInputPattern.test(next)) return
    setDraft(next)
    // Defer normalization for in-progress decimal input like "2."
    if (!isDeferredDecimal(next)) {
      onChange(next)
    }
  }

  const handleBlur = () => {
    setFocused(false)
    // Flush the final value on blur
    onChange(draft)
  }

  return (
    <div className='space-y-1.5'>
      <Label className='text-xs'>{label}</Label>
      <div className='flex'>
        <span className='border-input bg-muted text-muted-foreground inline-flex h-9 items-center rounded-l-md border border-r-0 px-3 text-sm'>
          $
        </span>
        <Input
          className='rounded-l-none'
          inputMode='decimal'
          value={focused ? draft : value}
          disabled={disabled}
          placeholder={placeholder}
          onFocus={() => setFocused(true)}
          onChange={(event) => handleChange(event.target.value)}
          onBlur={handleBlur}
        />
      </div>
    </div>
  )
}

function getContextPricingText(value: unknown) {
  if (!value) return ''
  return JSON.stringify(value, null, 2)
}

function ContextPricingEditor({
  modelName,
  contextPricing,
  onCommit,
  onValidityChange,
}: ContextPricingEditorProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState(() => getContextPricingText(contextPricing))
  const [error, setError] = useState<string | null>(null)

  const validateDraft = (next: string) => {
    if (!next.trim()) return null
    try {
      JSON.parse(next)
      return null
    } catch (parseError) {
      return parseError instanceof Error ? parseError.message : t('Invalid JSON')
    }
  }

  return (
    <div className='space-y-2'>
      <Label>{t('Context pricing JSON')}</Label>
      <Textarea
        className='min-h-32 font-mono text-xs'
        value={draft}
        placeholder={t('Leave empty to disable context pricing for this model')}
        onChange={(event) => {
          const next = event.target.value
          const nextError = validateDraft(next)
          setDraft(next)
          setError(nextError)
          onValidityChange?.(!nextError)
        }}
        onBlur={(event) => {
          const nextError = validateDraft(event.target.value)
          setError(nextError)
          onValidityChange?.(!nextError)
          if (nextError) {
            toast.error(t('Invalid context pricing JSON'))
            return
          }
          onCommit(modelName, event.target.value)
        }}
      />
      {error && <p className='text-destructive text-xs'>{error}</p>}
      <p className='text-muted-foreground text-xs'>
        {t('Stores this model entry inside ContextPricing. Use enabled:false or empty content to disable.')}
      </p>
    </div>
  )
}

export function ModelRatioVisualEditor({
  modelPrice,
  modelRatio,
  cacheRatio,
  createCacheRatio,
  completionRatio,
  audioRatio,
  audioCompletionRatio,
  contextPricing,
  onChange,
  onValidityChange,
}: ModelRatioVisualEditorProps) {
  const { t } = useTranslation()
  const [searchText, setSearchText] = useState('')
  const [pageIndex, setPageIndex] = useState(0)
  const [pageSize, setPageSize] = useState(50)
  const [selectedName, setSelectedName] = useState<string>('')
  const [customModelName, setCustomModelName] = useState('')

  const { data: enabledModelsData, isLoading: isLoadingEnabledModels } =
    useQuery({
      queryKey: ['channel', 'models-enabled'],
      queryFn: getEnabledModels,
      staleTime: 5 * 60 * 1000,
    })

  const maps = useMemo(
    () => ({
      price: parseNumericMap(modelPrice, 'model prices'),
      ratio: parseNumericMap(modelRatio, 'model ratios'),
      cache: parseNumericMap(cacheRatio, 'cache ratios'),
      createCache: parseNumericMap(createCacheRatio, 'create cache ratios'),
      completion: parseNumericMap(completionRatio, 'completion ratios'),
      audio: parseNumericMap(audioRatio, 'audio ratios'),
      audioCompletion: parseNumericMap(
        audioCompletionRatio,
        'audio completion ratios'
      ),
      context: safeJsonParse<UnknownMap>(contextPricing, {
        fallback: {},
        context: 'context pricing',
      }),
    }),
    [
      audioCompletionRatio,
      audioRatio,
      cacheRatio,
      completionRatio,
      contextPricing,
      createCacheRatio,
      modelPrice,
      modelRatio,
    ]
  )

  const rows = useMemo(() => {
    const names = new Set<string>([
      ...Object.keys(maps.price),
      ...Object.keys(maps.ratio),
      ...Object.keys(maps.cache),
      ...Object.keys(maps.createCache),
      ...Object.keys(maps.completion),
      ...Object.keys(maps.audio),
      ...Object.keys(maps.audioCompletion),
      ...Object.keys(maps.context),
      ...(enabledModelsData?.data || []),
    ])

    return Array.from(names)
      .filter((name) => name.trim())
      .map((name) => buildRow(name, maps))
      .sort((left, right) => {
        const rankCompare = getSortRank(left.mode) - getSortRank(right.mode)
        if (rankCompare !== 0) return rankCompare
        return left.name.localeCompare(right.name, 'en', {
          numeric: true,
          sensitivity: 'base',
        })
      })
  }, [enabledModelsData?.data, maps])

  const filteredRows = useMemo(() => {
    const keyword = searchText.trim().toLowerCase()
    if (!keyword) return rows
    return rows.filter((row) => row.name.toLowerCase().includes(keyword))
  }, [rows, searchText])

  const pageCount = Math.max(1, Math.ceil(filteredRows.length / pageSize))
  const safePageIndex = Math.min(pageIndex, pageCount - 1)
  const pageRows = filteredRows.slice(
    safePageIndex * pageSize,
    safePageIndex * pageSize + pageSize
  )
  const selectedRow =
    filteredRows.find((row) => row.name === selectedName) || pageRows[0] || null

  const selectModel = (name: string) => {
    setSelectedName(name)
    onValidityChange?.(true)
  }

  const writeMap = (field: ModelRatioField, map: Record<string, unknown>) => {
    onChange(field, sortedJson(map))
  }

  const clearModel = (name: string) => {
    const nextPrice = { ...maps.price }
    const nextRatio = { ...maps.ratio }
    const nextCache = { ...maps.cache }
    const nextCreateCache = { ...maps.createCache }
    const nextCompletion = { ...maps.completion }
    const nextAudio = { ...maps.audio }
    const nextAudioCompletion = { ...maps.audioCompletion }
    const nextContext = { ...maps.context }

    delete nextPrice[name]
    delete nextRatio[name]
    delete nextCache[name]
    delete nextCreateCache[name]
    delete nextCompletion[name]
    delete nextAudio[name]
    delete nextAudioCompletion[name]
    delete nextContext[name]

    writeMap('ModelPrice', nextPrice)
    writeMap('ModelRatio', nextRatio)
    writeMap('CacheRatio', nextCache)
    writeMap('CreateCacheRatio', nextCreateCache)
    writeMap('CompletionRatio', nextCompletion)
    writeMap('AudioRatio', nextAudio)
    writeMap('AudioCompletionRatio', nextAudioCompletion)
    writeMap('ContextPricing', nextContext)
  }

  const setMode = (name: string, mode: PricingMode) => {
    clearModel(name)
    if (mode === 'per-request') {
      writeMap('ModelPrice', { ...maps.price, [name]: 0 })
    }
    if (mode === 'per-token') {
      writeMap('ModelRatio', { ...maps.ratio, [name]: 0 })
    }
  }

  const setFixedPrice = (name: string, value: string) => {
    const parsed = parseNumber(value)
    const next = { ...maps.price }
    if (parsed === null) delete next[name]
    else next[name] = parsed
    writeMap('ModelPrice', next)
  }

  const setInputPrice = (name: string, value: string) => {
    const parsed = parseNumber(value)
    const next = { ...maps.ratio }
    if (parsed === null) delete next[name]
    else next[name] = normalizeNumber(parsed / 2)
    writeMap('ModelRatio', next)
  }

  const setRelativePrice = (
    field: 'CompletionRatio' | 'CacheRatio' | 'CreateCacheRatio' | 'AudioRatio',
    sourceMap: NumericMap,
    name: string,
    value: string,
    basePrice: number | undefined
  ) => {
    const parsed = parseNumber(value)
    const next = { ...sourceMap }
    if (parsed === null) {
      delete next[name]
    } else if (basePrice && basePrice > 0) {
      next[name] = normalizeNumber(parsed / basePrice)
    }
    writeMap(field, next)
  }

  const setAudioOutputPrice = (
    name: string,
    value: string,
    audioInputPrice: number | undefined
  ) => {
    const parsed = parseNumber(value)
    const next = { ...maps.audioCompletion }
    if (parsed === null) {
      delete next[name]
    } else if (audioInputPrice && audioInputPrice > 0) {
      next[name] = normalizeNumber(parsed / audioInputPrice)
    }
    writeMap('AudioCompletionRatio', next)
  }

  const setContextPricing = (name: string, value: string) => {
    const trimmed = value.trim()
    const next = { ...maps.context }
    if (!trimmed) {
      delete next[name]
      writeMap('ContextPricing', next)
      return
    }

    try {
      const parsed = JSON.parse(trimmed)
      next[name] = parsed
      writeMap('ContextPricing', next)
    } catch (error) {
      const message =
        error instanceof Error ? error.message : t('Invalid JSON')
      toast.error(t('Invalid context pricing JSON'), { description: message })
    }
  }

  const addCustomModel = () => {
    const name = customModelName.trim()
    if (!name) return
    setCustomModelName('')
    selectModel(name)
    setMode(name, 'per-token')
  }

  return (
    <div className='grid min-h-[560px] gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(360px,0.75fr)]'>
      <Card className='min-w-0'>
        <CardHeader className='border-b'>
          <CardTitle>{t('Models')}</CardTitle>
          <div className='flex flex-col gap-2 sm:flex-row'>
            <div className='relative min-w-0 flex-1'>
              <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
              <Input
                className='pl-9'
                value={searchText}
                placeholder={t('Search model name')}
                onChange={(event) => {
                  setSearchText(event.target.value)
                  setPageIndex(0)
                }}
              />
            </div>
            <div className='flex gap-2'>
              <Input
                value={customModelName}
                placeholder={t('Custom model name')}
                onChange={(event) => setCustomModelName(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    event.preventDefault()
                    addCustomModel()
                  }
                }}
              />
              <Button type='button' variant='outline' onClick={addCustomModel}>
                <Plus className='h-4 w-4' />
                {t('Add')}
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className='min-h-0 px-0'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className='pl-4'>{t('Model name')}</TableHead>
                <TableHead>{t('Billing type')}</TableHead>
                <TableHead>{t('Price summary')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pageRows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={3}
                    className='text-muted-foreground h-32 text-center'
                  >
                    {isLoadingEnabledModels
                      ? t('Loading...')
                      : t('No models found')}
                  </TableCell>
                </TableRow>
              ) : (
                pageRows.map((row) => (
                  <TableRow
                    key={row.name}
                    className={cn(
                      'cursor-pointer',
                      row.name === selectedRow?.name && 'bg-muted/70'
                    )}
                    onClick={() => selectModel(row.name)}
                  >
                    <TableCell className='max-w-[260px] truncate pl-4 font-medium'>
                      {row.name}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          row.mode === 'unset'
                            ? 'outline'
                            : row.mode === 'per-request'
                              ? 'secondary'
                              : 'default'
                        }
                      >
                        {row.mode === 'unset'
                          ? t('Unset price')
                          : row.mode === 'per-request'
                            ? t('Per-request')
                            : t('Per-token')}
                      </Badge>
                    </TableCell>
                    <TableCell className='max-w-[240px] truncate text-xs'>
                      {getRowSummary(row, t)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <div className='flex flex-col gap-3 border-t px-4 py-3 sm:flex-row sm:items-center sm:justify-between'>
            <div className='text-muted-foreground text-xs'>
              {t('{{count}} models', { count: filteredRows.length })}
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              <select
                className='border-input bg-background h-8 rounded-md border px-2 text-sm'
                value={pageSize}
                onChange={(event) => {
                  setPageSize(Number(event.target.value))
                  setPageIndex(0)
                }}
              >
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={safePageIndex === 0}
                onClick={() =>
                  setPageIndex(() => Math.max(0, safePageIndex - 1))
                }
              >
                {t('Previous')}
              </Button>
              <span className='text-muted-foreground text-xs'>
                {safePageIndex + 1} / {pageCount}
              </span>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={safePageIndex >= pageCount - 1}
                onClick={() =>
                  setPageIndex(() => Math.min(pageCount - 1, safePageIndex + 1))
                }
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className='min-w-0'>
        <CardHeader className='border-b'>
          <CardTitle className='truncate'>
            {selectedRow ? selectedRow.name : t('Price settings')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {!selectedRow ? (
            <div className='text-muted-foreground py-10 text-center text-sm'>
              {t('Select a model to edit pricing')}
            </div>
          ) : (
            <div className='space-y-5'>
              <div className='space-y-2'>
                <Label>{t('Billing type')}</Label>
                <RadioGroup
                  value={selectedRow.mode}
                  onValueChange={(value) =>
                    setMode(selectedRow.name, value as PricingMode)
                  }
                  className='grid gap-2 sm:grid-cols-3'
                >
                  {[
                    ['unset', t('Unset price')],
                    ['per-request', t('Per-request')],
                    ['per-token', t('Per-token')],
                  ].map(([value, label]) => (
                    <Label
                      key={value}
                      className='border-input bg-background has-data-checked:border-primary flex cursor-pointer items-center gap-2 rounded-md border p-3 text-sm font-normal'
                    >
                      <RadioGroupItem value={value} />
                      {label}
                    </Label>
                  ))}
                </RadioGroup>
              </div>

              {selectedRow.mode === 'unset' && (
                <div className='text-muted-foreground rounded-md border border-dashed p-4 text-sm'>
                  {t('This model has no custom pricing. Choose a billing type to configure it.')}
                </div>
              )}

              {selectedRow.mode === 'per-request' && (
                <PriceInput
                  label={t('Fixed price per request')}
                  value={toInputValue(selectedRow.fixedPrice)}
                  placeholder='0.01'
                  onChange={(value) => setFixedPrice(selectedRow.name, value)}
                />
              )}

              {selectedRow.mode === 'per-token' && (
                <div className='space-y-4'>
                  <PriceInput
                    label={t('Input price per 1M tokens')}
                    value={toInputValue(selectedRow.inputPrice)}
                    placeholder='2'
                    onChange={(value) => setInputPrice(selectedRow.name, value)}
                  />
                  <div className='grid gap-3 sm:grid-cols-2'>
                    <PriceInput
                      label={t('Completion price per 1M tokens')}
                      value={toInputValue(selectedRow.completionPrice)}
                      placeholder='4'
                      disabled={!selectedRow.inputPrice}
                      onChange={(value) =>
                        setRelativePrice(
                          'CompletionRatio',
                          maps.completion,
                          selectedRow.name,
                          value,
                          selectedRow.inputPrice
                        )
                      }
                    />
                    <PriceInput
                      label={t('Cache read price per 1M tokens')}
                      value={toInputValue(selectedRow.cachePrice)}
                      placeholder='0.2'
                      disabled={!selectedRow.inputPrice}
                      onChange={(value) =>
                        setRelativePrice(
                          'CacheRatio',
                          maps.cache,
                          selectedRow.name,
                          value,
                          selectedRow.inputPrice
                        )
                      }
                    />
                    <PriceInput
                      label={t('Cache write price per 1M tokens')}
                      value={toInputValue(selectedRow.createCachePrice)}
                      placeholder='1'
                      disabled={!selectedRow.inputPrice}
                      onChange={(value) =>
                        setRelativePrice(
                          'CreateCacheRatio',
                          maps.createCache,
                          selectedRow.name,
                          value,
                          selectedRow.inputPrice
                        )
                      }
                    />
                    <PriceInput
                      label={t('Audio input price per 1M tokens')}
                      value={toInputValue(selectedRow.audioInputPrice)}
                      placeholder='8'
                      disabled={!selectedRow.inputPrice}
                      onChange={(value) =>
                        setRelativePrice(
                          'AudioRatio',
                          maps.audio,
                          selectedRow.name,
                          value,
                          selectedRow.inputPrice
                        )
                      }
                    />
                    <PriceInput
                      label={t('Audio output price per 1M tokens')}
                      value={toInputValue(selectedRow.audioOutputPrice)}
                      placeholder='16'
                      disabled={!selectedRow.audioInputPrice}
                      onChange={(value) =>
                        setAudioOutputPrice(
                          selectedRow.name,
                          value,
                          selectedRow.audioInputPrice
                        )
                      }
                    />
                  </div>

                  <ContextPricingEditor
                    key={`${selectedRow.name}:${getContextPricingText(selectedRow.contextPricing)}`}
                    modelName={selectedRow.name}
                    contextPricing={selectedRow.contextPricing}
                    onCommit={setContextPricing}
                    onValidityChange={onValidityChange}
                  />
                </div>
              )}

              <Button
                type='button'
                variant='outline'
                className='w-full'
                onClick={() => clearModel(selectedRow.name)}
              >
                <Trash2 className='h-4 w-4' />
                {t('Clear this model pricing')}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
