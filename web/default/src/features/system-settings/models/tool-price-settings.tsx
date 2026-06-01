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
import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import { Code2, Copy, Eye, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { useUpdateOption } from '../hooks/use-update-option'

const OPTION_KEY = 'tool_billing_setting.rules'

type ToolBillingRule = {
  id: string
  name: string
  tool_type: 'web_search' | 'image_generation' | string
  billing_mode: 'per_call' | string
  price: number
  model_filter?: string
  quality?: string
  size?: string
  provider?: string
  enabled: boolean
}

type ToolBillingRow = ToolBillingRule & {
  rowId: number
}

function normalizeRule(rule: ToolBillingRule): ToolBillingRule {
  return {
    id: rule.id?.trim() ?? '',
    name: rule.name?.trim() ?? '',
    tool_type: rule.tool_type || 'web_search',
    billing_mode: rule.billing_mode || 'per_call',
    price: Number(rule.price) || 0,
    model_filter: rule.model_filter ?? '',
    quality: rule.quality ?? '',
    size: rule.size ?? '',
    provider: rule.provider ?? '',
    enabled: rule.enabled !== false,
  }
}

function rowsToRules(rows: ToolBillingRow[]): ToolBillingRule[] {
  return rows.map(({ rowId: _rowId, ...rule }) => normalizeRule(rule))
}

function validateRules(rules: ToolBillingRule[]): string | null {
  const seen = new Set<string>()
  for (const [index, rule] of rules.entries()) {
    if (!rule.id) return `rule ${index}: id is required`
    if (seen.has(rule.id)) return `rule ${index}: duplicate id ${rule.id}`
    seen.add(rule.id)
    if (!rule.name) return `rule ${index} (${rule.id}): name is required`
    if (!['web_search', 'image_generation'].includes(rule.tool_type)) {
      return `rule ${index} (${rule.id}): unsupported tool_type`
    }
    if (rule.billing_mode !== 'per_call') {
      return `rule ${index} (${rule.id}): only per_call billing is supported`
    }
    if (!Number.isFinite(rule.price) || rule.price < 0) {
      return `rule ${index} (${rule.id}): price must be a non-negative number`
    }
  }
  return null
}

function parseInitialRules(rawValue: string | undefined): ToolBillingRule[] {
  if (!rawValue) return []
  const parsed = JSON.parse(rawValue) as unknown
  if (!Array.isArray(parsed)) {
    throw new Error('Tool billing rules must be a JSON array')
  }
  return parsed.map((item) => normalizeRule(item as ToolBillingRule))
}

type ToolPriceSettingsProps = {
  defaultValue: string
}

export const ToolPriceSettings = memo(function ToolPriceSettings({
  defaultValue,
}: ToolPriceSettingsProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')
  const [rows, setRows] = useState<ToolBillingRow[]>([])
  const [jsonText, setJsonText] = useState('[]')
  const [jsonError, setJsonError] = useState('')
  const [nextRowId, setNextRowId] = useState(1)

  useEffect(() => {
    try {
      const rules = parseInitialRules(defaultValue)
      const initialRows = rules.map((rule, index) => ({
        ...rule,
        rowId: index + 1,
      }))
      setRows(initialRows)
      setJsonText(JSON.stringify(rules, null, 2))
      setJsonError('')
      setNextRowId(initialRows.length + 1)
    } catch (error) {
      setRows([])
      setJsonText(defaultValue || '[]')
      setJsonError(error instanceof Error ? error.message : t('Invalid JSON'))
    }
  }, [defaultValue, t])

  const currentRules = useMemo(() => rowsToRules(rows), [rows])

  const syncFromRows = useCallback((nextRows: ToolBillingRow[]) => {
    const rules = rowsToRules(nextRows)
    setRows(nextRows)
    setJsonText(JSON.stringify(rules, null, 2))
    setJsonError(validateRules(rules) ?? '')
  }, [])

  const handleJsonChange = useCallback(
    (text: string) => {
      setJsonText(text)
      try {
        const parsed = JSON.parse(text) as unknown
        if (!Array.isArray(parsed)) {
          setJsonError(t('JSON must be an array'))
          return
        }
        const rules = parsed.map((item) => normalizeRule(item as ToolBillingRule))
        const validationError = validateRules(rules)
        if (validationError) {
          setJsonError(validationError)
          return
        }
        const nextRows = rules.map((rule, index) => ({
          ...rule,
          rowId: index + 1,
        }))
        setRows(nextRows)
        setNextRowId(nextRows.length + 1)
        setJsonError('')
      } catch (error) {
        setJsonError(error instanceof Error ? error.message : t('Invalid JSON'))
      }
    },
    [t]
  )

  const updateRow = useCallback(
    (
      rowId: number,
      field: keyof ToolBillingRule,
      value: string | number | boolean
    ) => {
      syncFromRows(
        rows.map((row) =>
          row.rowId === rowId ? { ...row, [field]: value } : row
        )
      )
    },
    [rows, syncFromRows]
  )

  const addRow = useCallback(() => {
    const newRow: ToolBillingRow = {
      rowId: nextRowId,
      id: '',
      name: '',
      tool_type: 'web_search',
      billing_mode: 'per_call',
      price: 0,
      model_filter: '',
      quality: '',
      size: '',
      provider: '',
      enabled: true,
    }
    setNextRowId((prev) => prev + 1)
    syncFromRows([...rows, newRow])
  }, [nextRowId, rows, syncFromRows])

  const removeRow = useCallback(
    (rowId: number) => {
      syncFromRows(rows.filter((row) => row.rowId !== rowId))
    },
    [rows, syncFromRows]
  )

  const handleCopyJson = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(jsonText)
      toast.success(t('Copied to clipboard'))
    } catch {
      toast.error(t('Failed to copy'))
    }
  }, [jsonText, t])

  const handleSave = useCallback(async () => {
    const validationError = validateRules(currentRules)
    if (validationError || jsonError) {
      toast.error(validationError || jsonError)
      return
    }
    await updateOption.mutateAsync({
      key: OPTION_KEY,
      value: JSON.stringify(currentRules),
    })
  }, [currentRules, jsonError, updateOption])

  const toggleEditMode = useCallback(() => {
    setEditMode((prev) => (prev === 'visual' ? 'json' : 'visual'))
  }, [])

  return (
    <div className='space-y-4'>
      <Alert>
        <AlertDescription className='space-y-1 text-sm'>
          <div>
            {t('Configure tool billing rules. Prices are USD per call.')}
          </div>
          <div>
            <code className='bg-muted rounded px-1 py-0.5 text-xs'>
              tool_billing_setting.rules
            </code>
          </div>
        </AlertDescription>
      </Alert>

      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='flex flex-wrap items-center gap-2'>
          {editMode === 'visual' ? (
            <Button variant='outline' size='sm' onClick={addRow}>
              <Plus className='mr-2 h-4 w-4' />
              {t('Add')}
            </Button>
          ) : (
            <Button variant='ghost' size='sm' onClick={handleCopyJson}>
              <Copy className='mr-2 h-4 w-4' />
              {t('Copy')}
            </Button>
          )}
        </div>
        <Button variant='outline' size='sm' onClick={toggleEditMode}>
          {editMode === 'visual' ? (
            <>
              <Code2 className='mr-2 h-4 w-4' />
              {t('Switch to JSON')}
            </>
          ) : (
            <>
              <Eye className='mr-2 h-4 w-4' />
              {t('Switch to Visual')}
            </>
          )}
        </Button>
      </div>

      {editMode === 'visual' ? (
        <div className='overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className='min-w-[180px]'>ID</TableHead>
                <TableHead className='min-w-[180px]'>{t('Name')}</TableHead>
                <TableHead className='min-w-[150px]'>
                  {t('Tool type')}
                </TableHead>
                <TableHead className='min-w-[120px]'>
                  {t('Price')}
                </TableHead>
                <TableHead className='min-w-[130px]'>
                  {t('Provider')}
                </TableHead>
                <TableHead className='min-w-[180px]'>
                  {t('Model filter')}
                </TableHead>
                <TableHead className='min-w-[120px]'>
                  {t('Quality')}
                </TableHead>
                <TableHead className='min-w-[140px]'>{t('Size')}</TableHead>
                <TableHead className='w-[90px]'>{t('Enabled')}</TableHead>
                <TableHead className='w-[80px] text-right'>
                  {t('Actions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={10}
                    className='text-muted-foreground py-8 text-center'
                  >
                    {t('No rules configured')}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((row) => (
                  <TableRow key={row.rowId}>
                    <TableCell>
                      <Input
                        value={row.id}
                        placeholder='web_search_openai'
                        onChange={(event) =>
                          updateRow(row.rowId, 'id', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.name}
                        placeholder='OpenAI Web Search'
                        onChange={(event) =>
                          updateRow(row.rowId, 'name', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.tool_type}
                        placeholder='web_search'
                        onChange={(event) =>
                          updateRow(row.rowId, 'tool_type', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        type='number'
                        min={0}
                        step={0.001}
                        value={row.price}
                        onChange={(event) =>
                          updateRow(
                            row.rowId,
                            'price',
                            Number(event.target.value) || 0
                          )
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.provider ?? ''}
                        placeholder='openai'
                        onChange={(event) =>
                          updateRow(row.rowId, 'provider', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.model_filter ?? ''}
                        placeholder='gpt-4o*,gpt-4.1*'
                        onChange={(event) =>
                          updateRow(
                            row.rowId,
                            'model_filter',
                            event.target.value
                          )
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.quality ?? ''}
                        placeholder='high'
                        onChange={(event) =>
                          updateRow(row.rowId, 'quality', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={row.size ?? ''}
                        placeholder='1024x1024'
                        onChange={(event) =>
                          updateRow(row.rowId, 'size', event.target.value)
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={row.enabled}
                        onCheckedChange={(checked) =>
                          updateRow(row.rowId, 'enabled', checked)
                        }
                      />
                    </TableCell>
                    <TableCell className='text-right'>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => removeRow(row.rowId)}
                        aria-label={t('Delete')}
                      >
                        <Trash2 className='text-destructive h-4 w-4' />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className='space-y-2'>
          <Textarea
            value={jsonText}
            onChange={(event) => handleJsonChange(event.target.value)}
            className='font-mono text-sm'
            rows={16}
            spellCheck={false}
          />
          {jsonError && <p className='text-destructive text-sm'>{jsonError}</p>}
        </div>
      )}

      <div className='flex justify-end'>
        <Button
          onClick={handleSave}
          disabled={
            updateOption.isPending || (editMode === 'json' && !!jsonError)
          }
        >
          {t('Save tool billing rules')}
        </Button>
      </div>
    </div>
  )
})
