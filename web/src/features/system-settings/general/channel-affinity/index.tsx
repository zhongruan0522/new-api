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
import { Edit, FileText, Plus, RefreshCw, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { StatusBadge, StatusBadgeList } from '@/components/status-badge'
import { DisabledSettingsNotice } from '../../components/disabled-settings-notice'
import { SettingsSwitchField } from '../../components/settings-form-layout'
import { SettingsPageActionsPortal } from '../../components/settings-page-context'
import { SettingsSection } from '../../components/settings-section'
import { useUpdateOption } from '../../hooks/use-update-option'
import { getCacheStats, clearAllCache, clearRuleCache } from './api'
import { RULE_TEMPLATES, cloneTemplate, makeUniqueName } from './constants'
import { RuleEditorDialog } from './rule-editor-dialog'
import type { AffinityRule, CacheStats, ChannelAffinitySettings } from './types'

function parseRules(jsonStr: string): AffinityRule[] {
  try {
    const arr = JSON.parse(jsonStr || '[]')
    if (!Array.isArray(arr)) return []
    return arr.map(
      (r: Record<string, unknown>, i: number) =>
        ({ id: i, ...r }) as AffinityRule
    )
  } catch {
    return []
  }
}

function RuleBadgeList(props: { items: string[] }) {
  return (
    <StatusBadgeList
      items={props.items}
      max={2}
      getKey={(item) => item}
      renderItem={(item) => (
        <StatusBadge
          label={item}
          variant='neutral'
          size='sm'
          copyable={false}
        />
      )}
    />
  )
}

function serializeRules(rules: AffinityRule[]): string {
  return JSON.stringify(
    rules.map(({ id: _, ...rest }) => ({
      ...rest,
      skip_retry_on_failure: !!rest.skip_retry_on_failure,
    }))
  )
}

interface Props {
  defaultValues: ChannelAffinitySettings
}

export function ChannelAffinitySection(props: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const [enabled, setEnabled] = useState(
    props.defaultValues['channel_affinity_setting.enabled']
  )
  const [switchOnSuccess, setSwitchOnSuccess] = useState(
    props.defaultValues['channel_affinity_setting.switch_on_success']
  )
  const [keepOnChannelDisabled, setKeepOnChannelDisabled] = useState(
    props.defaultValues['channel_affinity_setting.keep_on_channel_disabled']
  )
  const [maxEntries, setMaxEntries] = useState(
    props.defaultValues['channel_affinity_setting.max_entries']
  )
  const [defaultTtl, setDefaultTtl] = useState(
    props.defaultValues['channel_affinity_setting.default_ttl_seconds']
  )
  const [rules, setRules] = useState<AffinityRule[]>(() =>
    parseRules(props.defaultValues['channel_affinity_setting.rules'])
  )

  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')
  const [jsonText, setJsonText] = useState(() =>
    JSON.stringify(
      parseRules(props.defaultValues['channel_affinity_setting.rules']).map(
        ({ id: _, ...r }) => r
      ),
      null,
      2
    )
  )
  const [cacheStats, setCacheStats] = useState<CacheStats | null>(null)
  const [cacheLoading, setCacheLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  const [ruleEditorOpen, setRuleEditorOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AffinityRule | null>(null)
  const [ruleTemplateKey, setRuleTemplateKey] = useState<string | null>(null)
  const [clearAllDialogOpen, setClearAllDialogOpen] = useState(false)
  const [clearRuleName, setClearRuleName] = useState<string | null>(null)
  const [fillTemplateDialogOpen, setFillTemplateDialogOpen] = useState(false)

  useEffect(() => {
    setEnabled(props.defaultValues['channel_affinity_setting.enabled'])
    setSwitchOnSuccess(
      props.defaultValues['channel_affinity_setting.switch_on_success']
    )
    setKeepOnChannelDisabled(
      props.defaultValues['channel_affinity_setting.keep_on_channel_disabled']
    )
    setMaxEntries(props.defaultValues['channel_affinity_setting.max_entries'])
    setDefaultTtl(
      props.defaultValues['channel_affinity_setting.default_ttl_seconds']
    )
    const parsed = parseRules(
      props.defaultValues['channel_affinity_setting.rules']
    )
    setRules(parsed)
    setJsonText(
      JSON.stringify(
        parsed.map(({ id: _, ...r }) => r),
        null,
        2
      )
    )
  }, [props.defaultValues])

  const refreshCache = useCallback(async () => {
    setCacheLoading(true)
    try {
      const res = await getCacheStats()
      if (res.success) setCacheStats(res.data || null)
    } catch {
      toast.error(t('systemSettings.errors.failedToRefreshCacheStats'))
    } finally {
      setCacheLoading(false)
    }
  }, [t])

  useEffect(() => {
    refreshCache()
  }, [refreshCache])

  const appendCliTemplates = () => {
    const existingNames = new Set(
      rules.map((r) => (r.name || '').trim()).filter((x) => x.length > 0)
    )

    const templates = Object.values(RULE_TEMPLATES).map((tpl) => {
      const base = cloneTemplate(tpl)
      const name = makeUniqueName(existingNames, tpl.name)
      existingNames.add(name)
      return { ...base, name }
    })

    setRules((prev) =>
      [...prev, ...templates].map((r, idx) => ({ ...r, id: idx }))
    )
    toast.success(t('systemSettings.fields.templatesAppended'))
    setFillTemplateDialogOpen(false)
  }

  const handleFillTemplates = () => {
    if (rules.length === 0) {
      appendCliTemplates()
    } else {
      setFillTemplateDialogOpen(true)
    }
  }

  const handleSave = async () => {
    let rulesJson: string
    if (editMode === 'json') {
      try {
        const parsed = JSON.parse(jsonText)
        if (!Array.isArray(parsed)) {
          toast.error(t('systemSettings.errors.rulesJsonMustBeAnArray'))
          return
        }
        rulesJson = JSON.stringify(parsed)
      } catch {
        toast.error(t('systemSettings.errors.invalidRulesJsonFormat'))
        return
      }
    } else {
      rulesJson = serializeRules(rules)
    }

    setSaving(true)
    try {
      const updates: { key: string; value: string }[] = []

      if (enabled !== props.defaultValues['channel_affinity_setting.enabled'])
        updates.push({
          key: 'channel_affinity_setting.enabled',
          value: String(enabled),
        })
      if (
        switchOnSuccess !==
        props.defaultValues['channel_affinity_setting.switch_on_success']
      )
        updates.push({
          key: 'channel_affinity_setting.switch_on_success',
          value: String(switchOnSuccess),
        })
      if (
        keepOnChannelDisabled !==
        props.defaultValues['channel_affinity_setting.keep_on_channel_disabled']
      )
        updates.push({
          key: 'channel_affinity_setting.keep_on_channel_disabled',
          value: String(keepOnChannelDisabled),
        })
      if (
        maxEntries !==
        props.defaultValues['channel_affinity_setting.max_entries']
      )
        updates.push({
          key: 'channel_affinity_setting.max_entries',
          value: String(maxEntries),
        })
      if (
        defaultTtl !==
        props.defaultValues['channel_affinity_setting.default_ttl_seconds']
      )
        updates.push({
          key: 'channel_affinity_setting.default_ttl_seconds',
          value: String(defaultTtl),
        })

      const origRules = props.defaultValues['channel_affinity_setting.rules']
      const origSerialized = (() => {
        try {
          return JSON.stringify(JSON.parse(origRules || '[]'))
        } catch {
          return '[]'
        }
      })()
      if (rulesJson !== origSerialized) {
        updates.push({
          key: 'channel_affinity_setting.rules',
          value: rulesJson,
        })
      }

      if (updates.length === 0) {
        toast.info(t('systemSettings.fields.noChanges'))
        return
      }

      for (const u of updates) {
        await updateOption.mutateAsync(u)
      }
      toast.success(t('systemSettings.status.savedSuccessfully'))
    } catch {
      toast.error(t('systemSettings.errors.failedToSave'))
    } finally {
      setSaving(false)
    }
  }

  const handleRuleSave = (rule: AffinityRule) => {
    setRules((prev) => {
      const existIdx = prev.findIndex(
        (r) => r.id === rule.id || (rule.name && r.name === editingRule?.name)
      )
      if (existIdx >= 0) {
        const next = [...prev]
        next[existIdx] = { ...rule, id: existIdx }
        return next
      }
      return [...prev, { ...rule, id: prev.length }]
    })
    setEditingRule(null)
  }

  const handleDeleteRule = (idx: number) => {
    setRules((prev) =>
      prev.filter((_, i) => i !== idx).map((r, i) => ({ ...r, id: i }))
    )
    toast.success(t('dynamicRatio.status.deletedSuccessfully'))
  }

  const handleClearAll = async () => {
    const res = await clearAllCache()
    if (res.success) {
      toast.success(t('systemSettings.fields.cleared'))
      refreshCache()
    }
    setClearAllDialogOpen(false)
  }

  const handleClearRule = async () => {
    if (!clearRuleName) return
    const res = await clearRuleCache(clearRuleName)
    if (res.success) {
      toast.success(t('systemSettings.fields.cleared'))
      refreshCache()
    }
    setClearRuleName(null)
  }

  const switchToJsonMode = () => {
    setJsonText(
      JSON.stringify(
        rules.map(({ id: _, ...r }) => r),
        null,
        2
      )
    )
    setEditMode('json')
  }

  const switchToVisualMode = () => {
    try {
      const parsed = JSON.parse(jsonText)
      if (!Array.isArray(parsed)) {
        toast.error(t('systemSettings.errors.rulesJsonMustBeAnArray'))
        return
      }
      setRules(
        parsed.map(
          (r: Record<string, unknown>, i: number) =>
            ({ id: i, ...r }) as AffinityRule
        )
      )
      setEditMode('visual')
    } catch {
      toast.error(t('systemSettings.errors.invalidRulesJsonFormat'))
    }
  }

  return (
    <>
      <SettingsSection title={t('systemSettings.fields.channelAffinity')}>
        <Alert>
          <AlertDescription className='text-xs'>
            {t(
              'systemSettings.status.channelAffinityReusesTheLastSuccessfulChannelBasedOn'
            )}
          </AlertDescription>
        </Alert>

        <DisabledSettingsNotice enabled={enabled} />

        {/* Basic Settings */}
        <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
          <SettingsSwitchField
            checked={enabled}
            onCheckedChange={setEnabled}
            label={t('channels.actions.enable')}
            className='border-b-0 py-0'
          />
          <div className='grid gap-1.5'>
            <Label>{t('systemSettings.fields.maxEntries')}</Label>
            <Input
              type='number'
              min={0}
              value={maxEntries}
              onChange={(e) => setMaxEntries(Number(e.target.value))}
            />
          </div>
          <div className='grid gap-1.5'>
            <Label>{t('systemSettings.fields.defaultTtlSeconds')}</Label>
            <Input
              type='number'
              min={0}
              value={defaultTtl}
              onChange={(e) => setDefaultTtl(Number(e.target.value))}
            />
          </div>
        </div>

        <SettingsSwitchField
          checked={switchOnSuccess}
          onCheckedChange={setSwitchOnSuccess}
          label={t('systemSettings.actions.switchAffinityOnSuccess')}
          description={t(
            'systemSettings.status.ifTheAffinityChannelFailsAndRetrySucceedsOn'
          )}
        />

        <SettingsSwitchField
          checked={keepOnChannelDisabled}
          onCheckedChange={setKeepOnChannelDisabled}
          label={t('systemSettings.actions.keepAffinityOnChannelDisabled')}
          description={t(
            'systemSettings.status.whenEnabledKeepTheAffinityEntryEvenIfThe'
          )}
        />

        <Separator />

        <SettingsPageActionsPortal>
          <Button
            variant={editMode === 'visual' ? 'default' : 'outline'}
            size='sm'
            onClick={editMode === 'json' ? switchToVisualMode : undefined}
          >
            {t('channels.fields.visual')}
          </Button>
          <Button
            variant={editMode === 'json' ? 'default' : 'outline'}
            size='sm'
            onClick={editMode === 'visual' ? switchToJsonMode : undefined}
          >
            JSON
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button variant='outline' size='sm' />}
            >
              <Plus className='mr-1 h-3 w-3' />
              {t('systemSettings.actions.addRule')}
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem
                onClick={() => {
                  setEditingRule(null)
                  setRuleTemplateKey(null)
                  setRuleEditorOpen(true)
                }}
              >
                {t('systemSettings.fields.blankRule')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => {
                  setEditingRule(null)
                  setRuleTemplateKey('codexCli')
                  setRuleEditorOpen(true)
                }}
              >
                Codex CLI
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => {
                  setEditingRule(null)
                  setRuleTemplateKey('claudeCli')
                  setRuleEditorOpen(true)
                }}
              >
                Claude CLI
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button variant='outline' size='sm' onClick={handleFillTemplates}>
            <FileText className='mr-1 h-3 w-3' />
            {t('systemSettings.actions.fillTemplates')}
          </Button>
          <Button size='sm' onClick={handleSave} disabled={saving}>
            {saving ? t('channels.tips.saving') : t('channels.actions.save')}
          </Button>
          <Button
            variant='outline'
            size='sm'
            onClick={refreshCache}
            disabled={cacheLoading}
          >
            <RefreshCw
              className={`mr-1 h-3 w-3 ${cacheLoading ? 'animate-spin' : ''}`}
            />
            {t('systemSettings.actions.refreshCache')}
          </Button>
          <Button
            variant='destructive'
            size='sm'
            onClick={() => setClearAllDialogOpen(true)}
          >
            {t('systemSettings.actions.clearAllCache')}
          </Button>
          {cacheStats && (
            <span className='text-muted-foreground text-xs'>
              {t('systemSettings.fields.cacheEntries')}: {cacheStats.total} /{' '}
              {cacheStats.cache_capacity}
            </span>
          )}
        </SettingsPageActionsPortal>

        {/* Rules Table or JSON Editor */}
        {editMode === 'visual' ? (
          <div className='overflow-x-auto rounded-md border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('channels.fields.name')}</TableHead>
                  <TableHead>{t('systemSettings.fields.modelRegex')}</TableHead>
                  <TableHead>{t('systemSettings.fields.keySources')}</TableHead>
                  <TableHead>{t('systemSettings.fields.ttl')}</TableHead>
                  <TableHead>{t('common.actions.retry')}</TableHead>
                  <TableHead>{t('systemSettings.fields.scope')}</TableHead>
                  <TableHead>{t('pricing.fields.cache')}</TableHead>
                  <TableHead className='text-right'>
                    {t('channels.fields.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={8}
                      className='text-muted-foreground py-8 text-center'
                    >
                      {t('systemSettings.fields.noRulesYet')}
                    </TableCell>
                  </TableRow>
                ) : (
                  rules.map((rule, idx) => (
                    <TableRow key={idx}>
                      <TableCell className='font-medium'>
                        {rule.name || '-'}
                      </TableCell>
                      <TableCell>
                        <RuleBadgeList items={rule.model_regex || []} />
                      </TableCell>
                      <TableCell>
                        <RuleBadgeList
                          items={(rule.key_sources || []).map(
                            (src) =>
                              `${src.type}:${src.type === 'gjson' ? src.path : src.key}`
                          )}
                        />
                      </TableCell>
                      <TableCell>{rule.ttl_seconds || '-'}</TableCell>
                      <TableCell>
                        <StatusBadge
                          label={
                            rule.skip_retry_on_failure
                              ? t('systemSettings.fields.noRetry')
                              : t('common.actions.retry')
                          }
                          variant={
                            rule.skip_retry_on_failure ? 'danger' : 'neutral'
                          }
                          copyable={false}
                        />
                      </TableCell>
                      <TableCell>
                        {(() => {
                          const scopeItems = [
                            rule.include_using_group &&
                              t('common.fields.group'),
                            rule.include_model_name && t('common.fields.model'),
                            rule.include_rule_name &&
                              t('systemSettings.fields.rule'),
                          ].filter(Boolean) as string[]
                          if (scopeItems.length === 0) return '-'
                          return <RuleBadgeList items={scopeItems} />
                        })()}
                      </TableCell>
                      <TableCell>
                        {rule.include_rule_name && cacheStats?.by_rule_name
                          ? cacheStats.by_rule_name[rule.name] || 0
                          : 'N/A'}
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='flex justify-end gap-1'>
                          {rule.include_rule_name && (
                            <Button
                              variant='ghost'
                              size='icon'
                              className='h-7 w-7'
                              onClick={() => setClearRuleName(rule.name)}
                              title={t(
                                'systemSettings.actions.clearCacheForThisRule'
                              )}
                            >
                              <X className='h-3 w-3' />
                            </Button>
                          )}
                          <Button
                            variant='ghost'
                            size='icon'
                            className='h-7 w-7'
                            onClick={() => {
                              setEditingRule(rule)
                              setRuleTemplateKey(null)
                              setRuleEditorOpen(true)
                            }}
                          >
                            <Edit className='h-3 w-3' />
                          </Button>
                          <Button
                            variant='ghost'
                            size='icon'
                            className='h-7 w-7'
                            onClick={() => handleDeleteRule(idx)}
                          >
                            <Trash2 className='h-3 w-3' />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className='grid gap-1.5'>
            <Label>{t('systemSettings.fields.rulesJson')}</Label>
            <Textarea
              className='min-h-[300px] font-mono text-xs'
              value={jsonText}
              onChange={(e) => setJsonText(e.target.value)}
            />
          </div>
        )}
      </SettingsSection>

      <RuleEditorDialog
        open={ruleEditorOpen}
        onOpenChange={setRuleEditorOpen}
        rule={editingRule}
        onSave={handleRuleSave}
        templateKey={ruleTemplateKey}
      />

      <ConfirmDialog
        open={clearAllDialogOpen}
        onOpenChange={setClearAllDialogOpen}
        title={t(
          'systemSettings.actions.confirmClearingAllChannelAffinityCache'
        )}
        desc={t(
          'systemSettings.tips.deleteAllChannelAffinityCacheEntriesStillInMemory'
        )}
        handleConfirm={handleClearAll}
        destructive
      />

      {clearRuleName !== null && (
        <ConfirmDialog
          open
          onOpenChange={(v) => !v && setClearRuleName(null)}
          title={t('systemSettings.actions.confirmClearingCacheForThisRule')}
          desc={`${t('systemSettings.fields.rule')}: ${clearRuleName}`}
          handleConfirm={handleClearRule}
          destructive
        />
      )}

      <ConfirmDialog
        open={fillTemplateDialogOpen}
        onOpenChange={setFillTemplateDialogOpen}
        title={t('systemSettings.actions.fillCodexCliClaudeCliTemplates')}
        desc={t('systemSettings.tips.append2TemplateRulesCodexCliAndClaudeCli')}
        handleConfirm={appendCliTemplates}
      />
    </>
  )
}
