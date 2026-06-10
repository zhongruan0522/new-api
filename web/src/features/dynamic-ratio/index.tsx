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

import { useState } from 'react'
import {
  ArrowDown,
  ArrowUp,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { getGroups } from '@/features/users/api'
import { SectionPageLayout } from '@/components/layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  createDynamicRatioRule,
  deleteDynamicRatioRule,
  getDynamicRatioRules,
  getDynamicRatioStatus,
  reorderDynamicRatioRules,
  setDynamicRatioEnabled,
  updateDynamicRatioRule,
} from './api'
import type { DynamicRatioRule, DynamicRatioRulePayload } from './types'

type RuleFormState = {
  group: string
  models: string
  concurrency: string
  weekdays: number[]
  start_time: string
  end_time: string
  ratio: string
  priority: string
  enable: boolean
}

const DEFAULT_FORM: RuleFormState = {
  group: '',
  models: '',
  concurrency: '',
  weekdays: [],
  start_time: '',
  end_time: '',
  ratio: '1.5',
  priority: '0',
  enable: true,
}

const WEEKDAYS = [
  { value: 0, label: 'Sun' },
  { value: 1, label: 'Mon' },
  { value: 2, label: 'Tue' },
  { value: 3, label: 'Wed' },
  { value: 4, label: 'Thu' },
  { value: 5, label: 'Fri' },
  { value: 6, label: 'Sat' },
]

const queryKeys = {
  rules: ['dynamic-ratio', 'rules'] as const,
  status: ['dynamic-ratio', 'status'] as const,
  groups: ['dynamic-ratio', 'groups'] as const,
}

function ruleToForm(rule: DynamicRatioRule | null): RuleFormState {
  if (!rule) return DEFAULT_FORM

  let weekdays: number[] = []
  if (rule.weekdays) {
    const parsed = JSON.parse(rule.weekdays) as unknown
    if (!Array.isArray(parsed)) {
      throw new Error('Invalid weekday data')
    }
    weekdays = parsed.map((day: unknown) => Number(day))
  }

  return {
    group: rule.group,
    models: rule.models || '',
    concurrency: rule.concurrency == null ? '' : String(rule.concurrency),
    weekdays,
    start_time: rule.start_time || '',
    end_time: rule.end_time || '',
    ratio: String(rule.ratio),
    priority: String(rule.priority ?? 0),
    enable: rule.enable !== false,
  }
}

function buildPayload(form: RuleFormState): DynamicRatioRulePayload {
  const group = form.group.trim()
  if (!group) throw new Error('Please select a group')

  const ratio = Number(form.ratio)
  if (!Number.isFinite(ratio) || ratio <= 0) {
    throw new Error('Ratio must be greater than 0')
  }

  const modelsText = form.models.trim()
  let models = ''
  if (modelsText) {
    const modelList = modelsText
      .split(',')
      .map((m) => m.trim())
      .filter((m) => m.length > 0)
    if (modelList.length > 0) {
      models = JSON.stringify(modelList)
    }
  }

  const concurrencyText = form.concurrency.trim()
  const concurrency =
    concurrencyText === '' ? null : Number.parseInt(concurrencyText, 10)
  if (
    concurrency !== null &&
    (!Number.isFinite(concurrency) || concurrency <= 0)
  ) {
    throw new Error('Concurrency threshold must be greater than 0')
  }

  const startTime = form.start_time.trim()
  const endTime = form.end_time.trim()
  if ((startTime && !endTime) || (!startTime && endTime)) {
    throw new Error('Start time and end time must be set together')
  }

  const priorityText = form.priority.trim()
  const priority =
    priorityText === '' ? 0 : Number.parseInt(priorityText, 10)
  if (!Number.isFinite(priority)) {
    throw new Error('Priority must be a number')
  }

  return {
    group,
    models,
    concurrency,
    weekdays: form.weekdays.length > 0 ? JSON.stringify(form.weekdays) : '',
    start_time: startTime,
    end_time: endTime,
    ratio,
    priority,
    enable: form.enable,
  }
}

function formatWeekdays(value: string, everyDayLabel: string): string {
  if (!value) return everyDayLabel
  const parsed = JSON.parse(value) as unknown
  if (!Array.isArray(parsed) || parsed.length === 0) return everyDayLabel
  return parsed
    .map((day) => {
      const weekday = WEEKDAYS.find((item) => item.value === Number(day))
      return weekday?.label ?? String(day)
    })
    .join(', ')
}

function getRatioVariant(ratio: number) {
  if (ratio > 3) return 'destructive'
  if (ratio > 1.5) return 'secondary'
  return 'outline'
}

function formatModels(value: string, allModelsLabel: string): string {
  if (!value) return allModelsLabel
  try {
    const parsed = JSON.parse(value) as unknown
    if (!Array.isArray(parsed) || parsed.length === 0) return allModelsLabel
    return parsed.join(', ')
  } catch {
    return value || allModelsLabel
  }
}

export function DynamicRatio() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const authUser = useAuthStore((state) => state.auth.user)
  const canEdit = (authUser?.role ?? 0) >= ROLE.SUPER_ADMIN
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<DynamicRatioRule | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<DynamicRatioRule | null>(
    null
  )
  const [form, setForm] = useState<RuleFormState>(DEFAULT_FORM)

  const rulesQuery = useQuery({
    queryKey: queryKeys.rules,
    queryFn: getDynamicRatioRules,
  })

  const statusQuery = useQuery({
    queryKey: queryKeys.status,
    queryFn: getDynamicRatioStatus,
  })

  const groupsQuery = useQuery({
    queryKey: queryKeys.groups,
    queryFn: async () => {
      const res = await getGroups()
      if (!res.success) throw new Error(res.message || 'Failed to load groups')
      if (!res.data) throw new Error('Failed to load groups')
      return res.data
    },
  })

  const rules = rulesQuery.data ?? []
  const groups = groupsQuery.data ?? []
  const globalEnabled = Boolean(statusQuery.data?.enabled)

  const refreshAll = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.rules }),
      queryClient.invalidateQueries({ queryKey: queryKeys.status }),
    ])
  }

  const setEnabledMutation = useMutation({
    mutationFn: setDynamicRatioEnabled,
    onSuccess: async (_, enabled) => {
      toast.success(
        enabled
          ? t('Dynamic ratio has been enabled')
          : t('Dynamic ratio has been disabled')
      )
      await queryClient.invalidateQueries({ queryKey: queryKeys.status })
    },
    onError: (error) => toast.error(error.message),
  })

  const saveDialogRuleMutation = useMutation({
    mutationFn: (payload: DynamicRatioRulePayload) =>
      editingRule
        ? updateDynamicRatioRule({ ...payload, id: editingRule.id })
        : createDynamicRatioRule(payload),
    onSuccess: async () => {
      toast.success(editingRule ? t('Updated successfully') : t('Created'))
      setDialogOpen(false)
      setEditingRule(null)
      setForm(DEFAULT_FORM)
      await refreshAll()
    },
    onError: (error) => toast.error(error.message),
  })

  const updateRuleMutation = useMutation({
    mutationFn: updateDynamicRatioRule,
    onSuccess: async () => {
      toast.success(t('Updated successfully'))
      await refreshAll()
    },
    onError: (error) => toast.error(error.message),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteDynamicRatioRule,
    onSuccess: async () => {
      toast.success(t('Deleted successfully'))
      setDeleteTarget(null)
      await refreshAll()
    },
    onError: (error) => toast.error(error.message),
  })

  const reorderMutation = useMutation({
    mutationFn: reorderDynamicRatioRules,
    onSuccess: async () => {
      toast.success(t('Order updated'))
      await refreshAll()
    },
    onError: (error) => toast.error(error.message),
  })

  const openCreateDialog = () => {
    setEditingRule(null)
    setForm(DEFAULT_FORM)
    setDialogOpen(true)
  }

  const openEditDialog = (rule: DynamicRatioRule) => {
    try {
      setEditingRule(rule)
      setForm(ruleToForm(rule))
      setDialogOpen(true)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Invalid rule'))
    }
  }

  const handleSubmit = () => {
    try {
      saveDialogRuleMutation.mutate(buildPayload(form))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('Invalid form'))
    }
  }

  const handleMove = (index: number, direction: -1 | 1) => {
    const nextIndex = index + direction
    if (nextIndex < 0 || nextIndex >= rules.length) return
    const reordered = [...rules]
    const current = reordered[index]
    reordered[index] = reordered[nextIndex]
    reordered[nextIndex] = current
    reorderMutation.mutate(reordered.map((rule) => rule.id))
  }

  const toggleWeekday = (day: number, checked: boolean) => {
    setForm((current) => ({
      ...current,
      weekdays: checked
        ? [...current.weekdays, day].sort((a, b) => a - b)
        : current.weekdays.filter((item) => item !== day),
    }))
  }

  const error = rulesQuery.error || statusQuery.error || groupsQuery.error
  const errorMessage = error instanceof Error ? error.message : null

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Dynamic Ratio')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <div className='flex items-center gap-2 rounded-lg border px-2.5 py-1.5'>
            <span className='text-muted-foreground text-sm'>
              {t('Global')}
            </span>
            <Switch
              checked={globalEnabled}
              disabled={!canEdit || statusQuery.isLoading}
              onCheckedChange={(checked) => setEnabledMutation.mutate(checked)}
            />
          </div>
          <Button variant='outline' onClick={() => void refreshAll()}>
            <RefreshCw />
            {t('Refresh')}
          </Button>
          <Button onClick={openCreateDialog} disabled={!canEdit}>
            <Plus />
            {t('New rule')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            {errorMessage && (
              <div className='border-destructive/40 text-destructive rounded-lg border px-3 py-2 text-sm'>
                {errorMessage}
              </div>
            )}

            <Card size='sm'>
              <CardContent className='grid gap-3 sm:grid-cols-3'>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Status')}
                  </div>
                  <div className='mt-1 text-sm font-medium'>
                    {globalEnabled ? t('Enabled') : t('Disabled')}
                  </div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Active ratio')}
                  </div>
                  <div className='mt-1 text-sm font-medium'>
                    {statusQuery.data?.active_ratio ?? '-'}x
                  </div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Timezone')}
                  </div>
                  <div className='mt-1 text-sm font-medium'>
                    {statusQuery.data?.timezone || '-'}
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card size='sm'>
              <CardContent className='p-0'>
                <Table>
                  <TableHeader>
                    <TableRow>
                       <TableHead className='w-16'>{t('Enabled')}</TableHead>
                       <TableHead>{t('Group')}</TableHead>
                       <TableHead>{t('Models')}</TableHead>
                       <TableHead>{t('Concurrency')}</TableHead>
                       <TableHead>{t('Weekdays')}</TableHead>
                       <TableHead>{t('Time Range')}</TableHead>
                       <TableHead>{t('Ratio')}</TableHead>
                       <TableHead>{t('Priority')}</TableHead>
                       <TableHead className='w-40 text-right'>
                         {t('Actions')}
                       </TableHead>
                     </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rulesQuery.isLoading ? (
                       <TableRow>
                         <TableCell colSpan={9} className='h-24 text-center'>
                           {t('Loading...')}
                         </TableCell>
                       </TableRow>
                     ) : rules.length === 0 ? (
                       <TableRow>
                         <TableCell colSpan={9} className='h-24 text-center'>
                           {t('No dynamic ratio rules')}
                         </TableCell>
                       </TableRow>
                    ) : (
                      rules.map((rule, index) => (
                        <TableRow key={rule.id}>
                          <TableCell>
                            <Switch
                              size='sm'
                              checked={rule.enable !== false}
                              disabled={!canEdit}
                              onCheckedChange={(checked) =>
                                updateRuleMutation.mutate({
                                  ...rule,
                                  enable: checked,
                                })
                              }
                            />
                          </TableCell>
                          <TableCell>
                             <Badge variant='outline'>{rule.group}</Badge>
                           </TableCell>
                           <TableCell className='max-w-48 truncate'>
                             {formatModels(rule.models, t('All Models'))}
                           </TableCell>
                          <TableCell>
                            {rule.concurrency ? (
                              <Badge variant='secondary'>
                                {rule.concurrency}
                              </Badge>
                            ) : (
                              <span className='text-muted-foreground'>
                                {t('Any')}
                              </span>
                            )}
                          </TableCell>
                          <TableCell>
                            {formatWeekdays(rule.weekdays, t('Daily'))}
                          </TableCell>
                          <TableCell>
                            {rule.start_time && rule.end_time ? (
                              `${rule.start_time} - ${rule.end_time}`
                            ) : (
                              <span className='text-muted-foreground'>
                                {t('Any')}
                              </span>
                            )}
                          </TableCell>
                          <TableCell>
                            <Badge variant={getRatioVariant(rule.ratio)}>
                              {rule.ratio}x
                            </Badge>
                          </TableCell>
                          <TableCell>{rule.priority ?? 0}</TableCell>
                          <TableCell>
                            <div className='flex justify-end gap-1'>
                              <Button
                                size='icon-sm'
                                variant='ghost'
                                disabled={!canEdit || index === 0}
                                onClick={() => handleMove(index, -1)}
                              >
                                <ArrowUp />
                                <span className='sr-only'>{t('Move up')}</span>
                              </Button>
                              <Button
                                size='icon-sm'
                                variant='ghost'
                                disabled={!canEdit || index === rules.length - 1}
                                onClick={() => handleMove(index, 1)}
                              >
                                <ArrowDown />
                                <span className='sr-only'>
                                  {t('Move down')}
                                </span>
                              </Button>
                              <Button
                                size='icon-sm'
                                variant='ghost'
                                disabled={!canEdit}
                                onClick={() => openEditDialog(rule)}
                              >
                                <Pencil />
                                <span className='sr-only'>{t('Edit')}</span>
                              </Button>
                              <Button
                                size='icon-sm'
                                variant='destructive'
                                disabled={!canEdit}
                                onClick={() => setDeleteTarget(rule)}
                              >
                                <Trash2 />
                                <span className='sr-only'>{t('Delete')}</span>
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className='sm:max-w-xl'>
          <DialogHeader>
            <DialogTitle>
              {editingRule ? t('Edit Rule') : t('New rule')}
            </DialogTitle>
          </DialogHeader>
          <div className='grid gap-4'>
            <div className='grid gap-1.5'>
               <Label htmlFor='dynamic-ratio-group'>{t('Group')}</Label>
               <Select
                 value={form.group || null}
                 onValueChange={(value) =>
                   setForm((current) => ({
                     ...current,
                     group: value ?? '',
                   }))
                 }
               >
                 <SelectTrigger id='dynamic-ratio-group' className='w-full'>
                   <SelectValue placeholder={t('Select a group')} />
                 </SelectTrigger>
                 <SelectContent alignItemWithTrigger={false}>
                   <SelectGroup>
                     {groups.map((group) => (
                       <SelectItem key={group} value={group}>
                         {group}
                       </SelectItem>
                     ))}
                   </SelectGroup>
                 </SelectContent>
               </Select>
             </div>

             <div className='grid gap-1.5'>
               <Label htmlFor='dynamic-ratio-models'>{t('Models')}</Label>
               <Input
                 id='dynamic-ratio-models'
                 value={form.models}
                 placeholder={t('e.g. gpt-4*, claude-3-opus (empty = all)')}
                 onChange={(event) =>
                   setForm((current) => ({
                     ...current,
                     models: event.target.value,
                   }))
                 }
               />
               <p className='text-muted-foreground text-xs'>
                 {t('Comma-separated, supports * wildcard. Leave empty for all models.')}
               </p>
             </div>

            <div className='grid gap-3 sm:grid-cols-2'>
              <div className='grid gap-1.5'>
                <Label htmlFor='dynamic-ratio-concurrency'>
                  {t('Concurrency threshold')}
                </Label>
                <Input
                  id='dynamic-ratio-concurrency'
                  type='number'
                  min='1'
                  value={form.concurrency}
                  placeholder={t('Any')}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      concurrency: event.target.value,
                    }))
                  }
                />
              </div>
              <div className='grid gap-1.5'>
                <Label htmlFor='dynamic-ratio-ratio'>{t('Ratio')}</Label>
                <Input
                  id='dynamic-ratio-ratio'
                  type='number'
                  min='0.01'
                  step='0.1'
                  value={form.ratio}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      ratio: event.target.value,
                    }))
                  }
                />
              </div>
            </div>

            <div className='grid gap-3 sm:grid-cols-2'>
              <div className='grid gap-1.5'>
                <Label htmlFor='dynamic-ratio-start-time'>
                  {t('Start Time')}
                </Label>
                <Input
                  id='dynamic-ratio-start-time'
                  value={form.start_time}
                  placeholder='HH:MM'
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      start_time: event.target.value,
                    }))
                  }
                />
              </div>
              <div className='grid gap-1.5'>
                <Label htmlFor='dynamic-ratio-end-time'>
                  {t('End Time')}
                </Label>
                <Input
                  id='dynamic-ratio-end-time'
                  value={form.end_time}
                  placeholder='HH:MM'
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      end_time: event.target.value,
                    }))
                  }
                />
              </div>
            </div>

            <div className='grid gap-2'>
              <Label>{t('Weekdays')}</Label>
              <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
                {WEEKDAYS.map((weekday) => {
                  const checked = form.weekdays.includes(weekday.value)
                  return (
                    <label
                      key={weekday.value}
                      className={cn(
                        'flex items-center gap-2 rounded-lg border px-2.5 py-2 text-sm',
                        checked && 'border-primary/60 bg-primary/5'
                      )}
                    >
                      <Checkbox
                        checked={checked}
                        onCheckedChange={(value) =>
                          toggleWeekday(weekday.value, value === true)
                        }
                      />
                      {t(weekday.label)}
                    </label>
                  )
                })}
              </div>
            </div>

            <div className='grid gap-3 sm:grid-cols-2'>
              <div className='grid gap-1.5'>
                <Label htmlFor='dynamic-ratio-priority'>
                  {t('Priority')}
                </Label>
                <Input
                  id='dynamic-ratio-priority'
                  type='number'
                  value={form.priority}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      priority: event.target.value,
                    }))
                  }
                />
              </div>
              <div className='flex items-center justify-between rounded-lg border px-3 py-2'>
                <div>
                  <Label>{t('Enabled')}</Label>
                  <p className='text-muted-foreground text-xs'>
                    {t('Enable this rule after saving')}
                  </p>
                </div>
                <Switch
                  checked={form.enable}
                  onCheckedChange={(checked) =>
                    setForm((current) => ({ ...current, enable: checked }))
                  }
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setDialogOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={saveDialogRuleMutation.isPending}
            >
              {editingRule ? t('Save Changes') : t('Create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={deleteTarget != null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Delete rule')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('This dynamic ratio rule will be deleted.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (deleteTarget) deleteMutation.mutate(deleteTarget.id)
              }}
            >
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
